/**
 * @Author: Nan
 * @Date: 2023/3/19 14:47
 */

package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/audit"
	error2 "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrTransactionalAuditRepositoryRequired = errors.New("transactional audit repository is required")
	ErrTransactionalAuditGeneratorRequired  = errors.New("transactional audit id generator is required")
	ErrLoginLogContextWriterRequired        = errors.New("login log context writer is required")
)

type TransactionalAuditChange struct {
	OldValue any `json:"old_value"`
	NewValue any `json:"new_value"`
}

type TransactionalAuditRecord struct {
	Action       string
	ResourceType string
	ResourceCode string
	ResourceId   string
	Changes      map[string]TransactionalAuditChange
}

type TransactionalAuditWriter interface {
	RecordTransactionalAudit(context.Context, *gorm.DB, TransactionalAuditRecord) error
}

type StandardContextAuditWriter interface {
	RecordTransactionalAuditContext(context.Context, *gorm.DB, TransactionalAuditRecord) error
}

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

func (ls *LogService) CreateLoginLog(ctx context.Context, log model.LoginLog) error {
	if ls == nil || ls.loginLogRepository == nil {
		return ErrLoginLogContextWriterRequired
	}
	if ls.sf == nil {
		return ErrTransactionalAuditGeneratorRequired
	}
	id, err := ls.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	log.Id = int(id)
	return ls.loginLogRepository.CreateLoginLogContext(ctx, &log)
}

func (ls *LogService) CreateAccessLog(ctx context.Context, log model.AccessLog) error {
	id, err := ls.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	log.Id = int(id)
	err = ls.accessLogRepository.Create(ls.accessLogRepository.DBWithContext(ctx), &log)
	return err
}

// RecordTransactionalAudit 在领域写入事务中一并持久化成功的敏感操作。
// 事务回滚后的失败请求仍由请求级 LogHandler 记录。
func (ls *LogService) RecordTransactionalAudit(
	ctx context.Context,
	tx *gorm.DB,
	record TransactionalAuditRecord,
) error {
	return ls.recordTransactionalAudit(ctx, tx, record)
}

// RecordTransactionalAuditContext keeps the standard-context audit contract
// used by Integration and other non-HTTP services.
func (ls *LogService) RecordTransactionalAuditContext(
	ctx context.Context,
	tx *gorm.DB,
	record TransactionalAuditRecord,
) error {
	return ls.recordTransactionalAudit(ctx, tx, record)
}

func (ls *LogService) recordTransactionalAudit(
	ctx context.Context,
	tx *gorm.DB,
	record TransactionalAuditRecord,
) error {
	if tx == nil {
		return ErrTransactionDatabaseRequired
	}
	if ls == nil || ls.accessLogRepository == nil {
		return ErrTransactionalAuditRepositoryRequired
	}
	if ls.sf == nil {
		return ErrTransactionalAuditGeneratorRequired
	}

	id, err := ls.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{
		"resource_id": record.ResourceId,
		"changes":     record.Changes,
	})
	if err != nil {
		return err
	}

	subject, _ := audit.GetAuditSubject(ctx)
	correlation := audit.GetCorrelationIDs(ctx)
	metadata := audit.GetRequestMetadata(ctx)
	method := metadata.Method
	if method == "" {
		method = "AUDIT"
	}
	return ls.accessLogRepository.Create(tx.WithContext(ctx), &model.AccessLog{
		Basic:        model.Basic{Id: int(id)},
		UserId:       subject.UserID,
		UserName:     subject.UserName,
		RequestId:    correlation.RequestID,
		TraceId:      correlation.TraceID,
		Method:       method,
		Ip:           metadata.ClientIP,
		Url:          metadata.Path,
		Action:       record.Action,
		ResourceType: record.ResourceType,
		ResourceCode: record.ResourceCode,
		ResourceId:   record.ResourceId,
		StatusCode:   http.StatusOK,
		Success:      true,
		Result:       "success",
		Body:         string(body),
	})
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
		return nil, error2.NewValidationError("开始时间不能晚于结束时间")
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
	return nil, error2.NewValidationError("时间格式不正确，请使用 YYYY-MM-DD HH:mm:ss")
}

func accessLogResponse(data model.AccessLog) response.AccessLogRes {
	return response.AccessLogRes{
		BasicRes:     response.NewBasicRes(data.Basic),
		UserName:     data.UserName,
		RequestId:    data.RequestId,
		TraceId:      data.TraceId,
		Method:       data.Method,
		Ip:           data.Ip,
		Locality:     data.Locality,
		Url:          data.Url,
		Action:       data.Action,
		ResourceType: data.ResourceType,
		ResourceCode: data.ResourceCode,
		ResourceId:   data.ResourceId,
		StatusCode:   data.StatusCode,
		Success:      data.Success,
		Result:       data.Result,
		ErrorCode:    data.ErrorCode,
		ErrorMessage: data.ErrorMessage,
		DurationMs:   data.DurationMs,
	}
}

func (ls *LogService) QueryAccessLogsResponse(ctx context.Context, req request.AccessLogQueryReq) (response.ListResult[response.AccessLogRes], error) {
	basic, err := buildAccessLogQueryBasic(req)
	if err != nil {
		return response.ListResult[response.AccessLogRes]{}, err
	}
	data, err := ls.accessLogRepository.GetAccessLogList(ctx, basic)
	if err != nil {
		return response.ListResult[response.AccessLogRes]{}, err
	}
	items := make([]response.AccessLogRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, accessLogResponse(item))
	}
	return response.ListResult[response.AccessLogRes]{Data: items, Total: data.Total}, nil
}

func (ls *LogService) GetAccessLogByIdResponse(ctx context.Context, id int) (response.AccessLogRes, error) {
	data, err := ls.accessLogRepository.WithContext(ctx).FindById(id)
	return accessLogResponse(data), err
}
