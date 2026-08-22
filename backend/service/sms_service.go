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
	"backend/internal/asynctask"
	"backend/internal/cache"
	error2 "backend/internal/errors"
	"backend/internal/sms"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"

	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SmsService 编排短信发送、供应商状态查询和模板管理，
// 对外只返回平台稳定状态与安全错误。
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
func (s *SmsService) SendSms(
	taskContext asynctask.Context,
	application model.Application,
	templateCode string,
	mobile string,
	params map[string]interface{},
) (map[string]interface{}, error) {
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
		return nil, error2.WrapSystemError(err)
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
		zap.L().Error("短信发送失败", zap.String("provider_code", *result.Code))
		return nil, error2.ErrSmsSendFailed
	}
	// 记录短信发送日志
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return nil, err
	}
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
	s.createSmsLogAsync(taskContext, smsLog)
	return tempParam, nil
}

func (s *SmsService) createSmsLogAsync(taskContext asynctask.Context, smsLog model.SmsLog) {
	asynctask.Run(taskContext, func(ctx context.Context) {
		if err := s.smsLogRepo.CreateSmsLogContext(ctx, &smsLog); err != nil {
			logAsyncSmsError(ctx, err)
		}
	})
}

func logAsyncSmsError(ctx context.Context, err error) {
	metadata := asynctask.MetadataFrom(ctx)
	zap.L().Error("记录短信发送日志失败",
		zap.Error(err),
		zap.String("request_id", metadata.RequestID),
		zap.String("trace_id", metadata.TraceID),
		zap.Int("user_id", metadata.UserID),
		zap.String("client_ip", metadata.ClientIP),
	)
}

func redactSmsTemplateParamsForLog(params map[string]interface{}) map[string]interface{} {
	redacted := make(map[string]interface{}, len(params))
	for key := range params {
		redacted[key] = "***"
	}
	return redacted
}

// CheckSmsStatus 检查短信发送状态
func (s *SmsService) CheckSmsStatus(ctx context.Context, applicationID int, bizID, mobile string) (response.SmsStatusRes, error) {
	if applicationID <= 0 {
		return response.SmsStatusRes{}, error2.ErrAppUnauthorized
	}
	log, err := s.smsLogRepo.FindByFieldWithDB(s.smsLogRepo.DBWithContext(ctx), "biz_id", bizID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SmsStatusRes{}, error2.ErrSmsStatusNotFound
		}
		return response.SmsStatusRes{}, err
	}
	if !smsStatusLogOwnedBy(log, applicationID, mobile) {
		return response.SmsStatusRes{}, error2.ErrSmsStatusNotFound
	}

	client, err := sms.GetSmsClient(s.serverConfig.ALiYun.SMS.AccessKeyId, s.serverConfig.ALiYun.SMS.AccessKeySecret)
	if err != nil {
		zap.L().Error("获取短信客户端失败", zap.Error(err))
		return response.SmsStatusRes{}, error2.ErrClientNotFound
	}
	sendData := log.GmtCreate.Format("20060102")
	result, err := sms.CheckSmsStatus(client, bizID, mobile, sendData)
	if err != nil {
		return response.SmsStatusRes{}, err
	}
	status, err := smsStatusFromProvider(result)
	if err != nil {
		return response.SmsStatusRes{}, err
	}
	if err := s.smsLogRepo.Update(s.smsLogRepo.DBWithContext(ctx), map[string]interface{}{"status": status}, log.Id); err != nil {
		return response.SmsStatusRes{}, err
	}
	return response.SmsStatusRes{Status: status}, nil
}

func smsStatusFromProvider(result *dysmsapi20170525.QuerySendDetailsResponseBody) (enum.SmsStatus, error) {
	if result == nil || result.Code == nil || *result.Code != "OK" || result.SmsSendDetailDTOs == nil {
		return 0, error2.ErrSmsStatusQueryFailed
	}
	details := result.SmsSendDetailDTOs.SmsSendDetailDTO
	if len(details) == 0 || details[0] == nil || details[0].SendStatus == nil {
		return 0, error2.ErrSmsStatusQueryFailed
	}
	switch *details[0].SendStatus {
	case 1:
		return enum.SmsStatusSending, nil
	case 2:
		return enum.SmsStatusFailed, nil
	case 3:
		return enum.SmsStatusSuccess, nil
	default:
		return 0, error2.ErrSmsStatusQueryFailed
	}
}

func smsStatusLogOwnedBy(log model.SmsLog, applicationID int, mobile string) bool {
	return applicationID > 0 && log.ApplicationId == applicationID && log.Mobile == mobile
}

// CreateSmsTemplate 创建短信模板
func (s *SmsService) CreateSmsTemplate(ctx context.Context, data request.SmsTemplateCreateReq) (int, error) {
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
func (s *SmsService) UpdateSmsTemplate(ctx context.Context, req request.SmsTemplateUpdateReq) error {
	tx := s.smsTemplateRepo.DBWithContext(ctx)
	err := s.smsTemplateRepo.Update(tx, &req, req.Id)
	if err != nil {
		return err
	}
	return nil
}

func smsTemplateResponse(data model.SmsTemplate) response.SmsTemplateRes {
	return response.SmsTemplateRes{
		BasicRes:       response.NewBasicRes(data.Basic),
		SignName:       data.SignName,
		TemplateCode:   data.TemplateCode,
		TemplateName:   data.TemplateName,
		TemplateParams: json.RawMessage(data.TemplateParams),
	}
}

func (s *SmsService) GetSmsTemplateListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.SmsTemplateRes], error) {
	data, err := s.smsTemplateRepo.GetSmsTemplateList(basic, table)
	if err != nil {
		return response.ListResult[response.SmsTemplateRes]{}, err
	}
	items := make([]response.SmsTemplateRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, smsTemplateResponse(item))
	}
	return response.ListResult[response.SmsTemplateRes]{Data: items, Total: data.Total}, nil
}

func (s *SmsService) GetSmsTemplateByIdResponse(id int) (response.SmsTemplateRes, error) {
	data, err := s.getSmsTemplateByID(id)
	return smsTemplateResponse(data), err
}

// getSmsTemplateByID 根据ID获取短信模板
func (s *SmsService) getSmsTemplateByID(id int) (model.SmsTemplate, error) {
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
func (s *SmsService) DeleteSmsTemplateById(ctx context.Context, id int) error {
	tx := s.smsTemplateRepo.DBWithContext(ctx)
	err := s.smsTemplateRepo.DeleteById(tx, id)
	if err != nil {
		return err
	}
	return nil
}
