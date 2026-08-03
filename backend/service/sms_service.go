/**
 * @Author: Nan
 * @Date: 2025/2/7 21:53
 */

package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/cache"
	error2 "backend/internal/errors"
	"backend/internal/sms"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SmsService struct {
	smsLogRepo      repository.SmsLogRepository
	smsTemplateRepo repository.SmsTemplateRepository
	sf              *utils.Snowflake
	smsTempCache    *cache.SmsTemplateCache
	serverConfig    *config.Server
}

func NewSmsService(smsLogRepo repository.SmsLogRepository, smsTemplateRepo repository.SmsTemplateRepository,
	sf *utils.Snowflake,
	smsTempCache *cache.SmsTemplateCache,
	serverConfig *config.Server) *SmsService {
	return &SmsService{
		smsLogRepo,
		smsTemplateRepo,
		sf,
		smsTempCache,
		serverConfig,
	}
}

// SendSms 发送短信
func (s *SmsService) SendSms(ctx *gin.Context, templateCode, mobile string, params map[string]interface{}) (map[string]interface{}, error) {
	// 通过模板编号查询缓存
	smsTemp, err := s.smsTempCache.Get(templateCode)
	if err != nil {
		// 查询模版
		smsTemp, err := s.smsTemplateRepo.FindByField("template_code", templateCode)
		if err != nil {
			if !errors.Is(err, cache.ErrCacheMiss) {
				return nil, error2.ErrSmsTemplateNotFound
			}
			return nil, err
		}
		// 缓存模板
		_ = s.smsTempCache.Set(templateCode, smsTemp)
	}
	// 解析模版参数列表（存储为 JSON 格式）
	var expectedKeys []string
	if err := json.Unmarshal(smsTemp.TemplateParams, &expectedKeys); err != nil {
		return nil, error2.NewBadRequestError("解析模板参数失败" + err.Error())
	}
	client, err := sms.GetSmsClient(s.serverConfig.ALiYun.SMS.AccessKeyId, s.serverConfig.ALiYun.SMS.AccessKeySecret)
	if err != nil {
		zap.L().Error("获取短信客户端失败", zap.Error(err))
		return nil, error2.ErrClientNotFound
	}
	tempParam := make(map[string]interface{})
	// 如果模板参数为验证码短信（仅包含 "code"），则不需要传入其他参数
	if len(expectedKeys) == 1 && expectedKeys[0] == "code" {
		// 获取验证码
		code := utils.GenerateRandomNumber(6)
		zap.L().Debug("短信验证码已生成", zap.String("template_code", templateCode))
		tempParam["code"] = code
	} else {
		// 校验传入参数是否包含所有预期的键
		for _, key := range expectedKeys {
			if _, ok := params[key]; !ok {
				zap.L().Error("字段不合法", zap.String("key", key))
				return nil, error2.ErrSmsFieldInvalid
			}
			tempParam[key] = params[key]
		}
	}
	// tempParam 转为字符串
	tempParamStr, _ := json.Marshal(tempParam)
	smsLogContent, _ := json.Marshal(redactSmsTemplateParamsForLog(tempParam))
	// 发送短信
	result, err := sms.SendSms(client, smsTemp.SignName, smsTemp.TemplateCode, mobile, string(tempParamStr))
	if err != nil {
		return nil, err
	}
	if *result.Code != "OK" {
		zap.L().Error("短信发送失败", zap.Any("result", result))
		return nil, error2.NewBadRequestError(*result.Message)
	}
	// 记录短信发送日志
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return nil, err
	}
	if d, exists := ctx.Get("application"); exists {
		application := d.(model.Application)
		smsLog := model.SmsLog{
			TemplateCode:    templateCode,
			SignName:        smsTemp.SignName,
			Mobile:          mobile,
			Content:         string(smsLogContent),
			Status:          enum.SmsStatusSuccess,
			BizId:           *result.BizId,
			Result:          result.String(),
			ApplicationId:   application.Id,
			ApplicationName: application.Name,
		}
		smsLog.Id = int(id)
		// 异步记录短信发送日志
		go func() {
			if err := s.smsLogRepo.Create(s.smsLogRepo.DBWithContext(ctx), smsLog); err != nil {
				zap.L().Error("记录短信发送日志失败", zap.Error(err))
			}
		}()
	}
	return tempParam, nil
}

func redactSmsTemplateParamsForLog(params map[string]interface{}) map[string]interface{} {
	redacted := make(map[string]interface{}, len(params))
	for key, value := range params {
		if key == "code" {
			redacted[key] = "***"
			continue
		}
		redacted[key] = value
	}
	return redacted
}

// CheckSmsStatus 检查短信发送状态
func (s *SmsService) CheckSmsStatus(ctx *gin.Context, bizId, mobile string) (interface{}, error) {
	client, err := sms.GetSmsClient(s.serverConfig.ALiYun.SMS.AccessKeyId, s.serverConfig.ALiYun.SMS.AccessKeySecret)
	if err != nil {
		zap.L().Error("获取短信客户端失败", zap.Error(err))
		return nil, error2.ErrClientNotFound
	}
	// 根据 BizId 查询短信记录
	log, err := s.smsLogRepo.FindByField("biz_id", bizId)
	if err != nil {
		return nil, err
	}
	// 获取创建日期
	sendData := log.GmtCreate.Format("20060102")
	result, err := sms.CheckSmsStatus(client, bizId, mobile, sendData)
	if err != nil {
		return nil, err
	}
	if *result.Code != "OK" {
		return nil, error2.ErrSmsSendFailed
	}
	sms := map[string]interface{}{
		"status": enum.SmsStatusFailed,
	}
	if result.SmsSendDetailDTOs != nil {
		if *result.SmsSendDetailDTOs.SmsSendDetailDTO[0].SendStatus == 3 {
			sms["status"] = enum.SmsStatusSuccess
		}
	}
	// 更新短信发送状态
	if err := s.smsLogRepo.Update(s.smsLogRepo.DBWithContext(ctx), sms, log.Id); err != nil {
		return nil, err
	}
	return result.SmsSendDetailDTOs.SmsSendDetailDTO[0], nil
}

// GetSmsTemplateList 获取短信模板列表
func (s *SmsService) GetSmsTemplateList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SmsTemplate], error) {
	result, err := s.smsTemplateRepo.GetSmsTemplateList(basic, table)
	return result, err
}

// CreateSmsTemplate 创建短信模板
func (s *SmsService) CreateSmsTemplate(ctx *gin.Context, data request.SmsTemplateCreateReq) (int, error) {
	var template model.SmsTemplate
	// 查询模板编号是否存在
	smsTemp, err := s.smsTemplateRepo.FindByField("template_code", data.TemplateCode)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if smsTemp.Id != 0 {
		return 0, error2.ErrSmsTemplateExist
	}
	err = copier.Copy(&template, &data)
	if err != nil {
		return 0, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, err
	}
	template.Id = int(id)
	tx := s.smsTemplateRepo.DBWithContext(ctx)
	return template.Id, s.smsTemplateRepo.Create(tx, &template)
}

// UpdateSmsTemplate 更新短信模板
func (s *SmsService) UpdateSmsTemplate(ctx *gin.Context, req request.SmsTemplateUpdateReq) error {
	tx := s.smsTemplateRepo.DBWithContext(ctx)
	err := s.smsTemplateRepo.Update(tx, &req, req.Id)
	if err != nil {
		return err
	}
	return nil
}

// GetSmsTemplateById 根据ID获取短信模板
func (s *SmsService) GetSmsTemplateById(id int) (model.SmsTemplate, error) {
	smsTemp, err := s.smsTemplateRepo.FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SmsTemplate{}, nil
		}
		return model.SmsTemplate{}, err
	}
	return smsTemp, nil
}

// DeleteSmsTemplateById 根据ID删除短信模板
func (s *SmsService) DeleteSmsTemplateById(ctx *gin.Context, id int) error {
	tx := s.smsTemplateRepo.DBWithContext(ctx)
	err := s.smsTemplateRepo.DeleteById(tx, id)
	if err != nil {
		return err
	}
	return nil
}
