/**
 * @Author: Nan
 * @Date: 2023/3/19 14:47
 */

package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	error2 "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

type LogService struct {
	loginLogRepository  repository.LoginLogRepository
	accessLogRepository repository.AccessLogRepository
	sf                  *utils.Snowflake
}

func NewLogServer(loginLogRepository repository.LoginLogRepository, accessLogRepository repository.AccessLogRepository, sf *utils.Snowflake) *LogService {
	return &LogService{
		loginLogRepository,
		accessLogRepository,
		sf,
	}
}

func (ls *LogService) CreateLoginLog(ctx *gin.Context, log model.LoginLog) error {
	id, err := ls.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	log.Id = int(id)
	err = ls.loginLogRepository.Create(ls.loginLogRepository.DBWithContext(ctx), &log)
	return err
}

func (ls *LogService) CreateAccessLog(ctx *gin.Context, log model.AccessLog) error {
	id, err := ls.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	log.Id = int(id)
	err = ls.accessLogRepository.Create(ls.accessLogRepository.DBWithContext(ctx), &log)
	return err
}

func (ls *LogService) QueryAccessLogs(ctx *gin.Context, req request.AccessLogQueryReq) (response.ListResult[model.AccessLog], error) {
	basic, err := buildAccessLogQueryBasic(req)
	if err != nil {
		return response.ListResult[model.AccessLog]{}, err
	}
	return ls.accessLogRepository.GetAccessLogList(ctx, basic)
}

func buildAccessLogQueryBasic(req request.AccessLogQueryReq) (*request.Basic, error) {
	basic := req.Basic
	if basic.Order.Field == "" {
		basic.Order = request.Order{
			Field: "gmt_create",
			IsAsc: false,
		}
	}
	rules := make([]request.QueryRule, 0, 9)

	if userName := strings.TrimSpace(req.UserName); userName != "" {
		rules = append(rules, accessLogRule("user_name", enum.Like, userName, enum.VarcharFieldType))
	}
	if action := strings.TrimSpace(req.Action); action != "" {
		rules = append(rules, accessLogRule("action", enum.Eq, action, enum.VarcharFieldType))
	}
	if resourceCode := strings.TrimSpace(req.ResourceCode); resourceCode != "" {
		rules = append(rules, accessLogRule("resource_code", enum.Like, resourceCode, enum.VarcharFieldType))
	}
	if method := strings.TrimSpace(req.Method); method != "" {
		rules = append(rules, accessLogRule("method", enum.Eq, strings.ToUpper(method), enum.VarcharFieldType))
	}
	if url := strings.TrimSpace(req.Url); url != "" {
		rules = append(rules, accessLogRule("url", enum.Like, url, enum.VarcharFieldType))
	}
	if ip := strings.TrimSpace(req.Ip); ip != "" {
		rules = append(rules, accessLogRule("ip", enum.Like, ip, enum.VarcharFieldType))
	}
	if req.Success != nil {
		rules = append(rules, accessLogRule("success", enum.Eq, *req.Success, enum.BooleanFieldType))
	}
	startTime, err := parseAccessLogQueryTime(req.StartTime)
	if err != nil {
		return nil, err
	}
	endTime, err := parseAccessLogQueryTime(req.EndTime)
	if err != nil {
		return nil, err
	}
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return nil, error2.NewBadRequestError("开始时间不能晚于结束时间")
	}
	if startTime != nil {
		rules = append(rules, accessLogRule("gmt_create", enum.Gte, *startTime, enum.DatetimeFieldType))
	}
	if endTime != nil {
		rules = append(rules, accessLogRule("gmt_create", enum.Lte, *endTime, enum.DatetimeFieldType))
	}
	if len(rules) > 0 {
		basic.Expressions = append(basic.Expressions, request.ExpressionGroup{
			Logic: enum.And,
			Rules: rules,
		})
	}
	return &basic, nil
}

func accessLogRule(field string, expressionType enum.ExpressionType, value interface{}, valueType enum.SysTableFieldType) request.QueryRule {
	return request.QueryRule{
		Field:          field,
		ExpressionType: expressionType,
		Value:          value,
		Type:           valueType,
	}
}

func parseAccessLogQueryTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	layouts := []string{time.DateTime, time.DateOnly, time.RFC3339}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, model.AppLocation()); err == nil {
			if layout == time.DateOnly {
				parsed = parsed.Truncate(24 * time.Hour)
			}
			return &parsed, nil
		}
	}
	return nil, error2.NewBadRequestError("时间格式不正确，请使用 YYYY-MM-DD HH:mm:ss")
}

func (ls *LogService) GetAccessLogById(ctx *gin.Context, id int) (model.AccessLog, error) {
	return ls.accessLogRepository.WithContext(ctx).FindById(id)
}
