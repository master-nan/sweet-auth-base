package service

import (
	"backend/internal/audit"
	"backend/model"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type AuthAuditService struct {
	logs *LogService
}

func NewAuthAuditService(logs *LogService) *AuthAuditService {
	return &AuthAuditService{logs: logs}
}

func (s *AuthAuditService) RecordAuthEvent(ctx context.Context, event AuthAuditEvent) error {
	metadata := audit.GetRequestMetadata(ctx)
	correlation := audit.GetCorrelationIDs(ctx)
	principal := safeAuthPrincipal(event.UserID, event.Principal)
	body, err := json.Marshal(map[string]string{
		"channel":         string(event.Channel),
		"credential_type": string(event.CredentialType),
		"user_agent":      metadata.UserAgent,
	})
	if err != nil {
		return err
	}
	status := http.StatusUnauthorized
	result := "failure"
	if event.Success {
		status = http.StatusOK
		result = "success"
	} else if event.HTTPStatus > 0 {
		status = event.HTTPStatus
	}
	if event.Channel != AuthChannelRefresh && event.Channel != AuthChannelLogout {
		if err := s.logs.CreateLoginLog(ctx, model.LoginLog{
			Ip:       metadata.ClientIP,
			Locality: "",
			UserName: principal,
		}); err != nil {
			return err
		}
	}
	if err := s.logs.CreateAccessLog(ctx, model.AccessLog{
		UserId:       event.UserID,
		UserName:     principal,
		RequestId:    correlation.RequestID,
		TraceId:      correlation.TraceID,
		Method:       metadata.Method,
		Ip:           metadata.ClientIP,
		Url:          metadata.Path,
		Action:       "authenticate",
		ResourceType: "authentication",
		ResourceCode: string(event.Channel),
		ResourceId:   principal,
		StatusCode:   status,
		Success:      event.Success,
		Result:       result,
		ErrorCode:    event.ReasonCode,
		Body:         string(body),
	}); err != nil {
		return err
	}
	audit.MarkAccessAuditPersisted(ctx)
	return nil
}

func safeAuthPrincipal(userID int, principal string) string {
	if userID > 0 {
		return fmt.Sprintf("user:%d", userID)
	}
	normalized := strings.ToLower(strings.TrimSpace(principal))
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("principal:sha256:%x", sum[:8])
}
