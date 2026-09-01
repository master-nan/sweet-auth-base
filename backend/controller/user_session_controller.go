package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	appErrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

// UserSessionController 提供当前设备心跳、强制下线和登录设备查询。
type UserSessionController struct {
	sessions    *service.UserSessionService
	translators map[string]ut.Translator
}

func NewUserSessionController(sessions *service.UserSessionService, translators map[string]ut.Translator) *UserSessionController {
	return &UserSessionController{sessions: sessions, translators: translators}
}

func (u *UserSessionController) Heartbeat(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	userID, sessionID, ok := authenticatedSession(ctx)
	if !ok {
		_ = ctx.Error(appErrors.ErrUserNotLogin)
		return
	}
	if err := u.sessions.Heartbeat(ctx.Request.Context(), userID, sessionID); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(map[string]string{"server_time": model.CustomTime(time.Now().UTC()).String()})
}

func (u *UserSessionController) Query(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.UserSessionQueryReq
	if err := utils.ValidatorBody[request.UserSessionQueryReq](ctx, &data, u.translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return
	}
	_, sessionID, _ := authenticatedSession(ctx)
	result, err := u.sessions.Query(ctx.Request.Context(), data, sessionID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (u *UserSessionController) Revoke(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(appErrors.ErrParamInvalid)
		return
	}
	data, ok := u.revokeRequest(ctx)
	if !ok {
		return
	}
	closure := sessionClosure(ctx, data.Reason, "管理员手动下线")
	if err := u.sessions.RevokeSession(ctx.Request.Context(), id, closure); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(true)
}

func (u *UserSessionController) RevokeUser(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	userID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || userID <= 0 {
		_ = ctx.Error(appErrors.ErrParamInvalid)
		return
	}
	data, ok := u.revokeRequest(ctx)
	if !ok {
		return
	}
	closure := sessionClosure(ctx, data.Reason, "管理员手动下线全部会话")
	if err := u.sessions.RevokeUser(ctx.Request.Context(), userID, model.UserSessionStatusForcedOffline, closure); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(true)
}

func (u *UserSessionController) Export(ctx *gin.Context) {
	var data request.UserSessionQueryReq
	if err := utils.ValidatorBody[request.UserSessionQueryReq](ctx, &data, u.translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return
	}
	items, err := u.sessions.Export(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var content bytes.Buffer
	content.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&content)
	_ = writer.Write([]string{"会话编号", "用户", "账号已删除", "状态", "登录时间", "最后活动", "可刷新至", "退出时间", "退出原因", "结束操作人", "登录渠道", "IP 地址", "设备", "浏览器", "操作系统", "User-Agent"})
	for _, item := range items {
		logoutAt := ""
		if item.LogoutAt != nil {
			logoutAt = item.LogoutAt.String()
		}
		_ = writer.Write([]string{
			strconv.Itoa(item.ID), item.UserName, strconv.FormatBool(item.UserDeleted), item.Status,
			item.LoginAt.String(), item.LastSeenAt.String(), item.ExpiresAt.String(), logoutAt,
			item.LogoutReason, item.ClosedByUserName, item.LoginChannel, item.IPAddress,
			item.DeviceType, item.Browser, item.OperatingSystem, item.UserAgent,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = ctx.Error(appErrors.WrapSystemError(err))
		return
	}
	fileName := "login-sessions-" + time.Now().In(model.AppLocation()).Format("20060102150405") + ".csv"
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	ctx.Data(http.StatusOK, "text/csv; charset=utf-8", content.Bytes())
}

func (u *UserSessionController) revokeRequest(ctx *gin.Context) (request.UserSessionRevokeReq, bool) {
	var data request.UserSessionRevokeReq
	if ctx.Request.ContentLength <= 0 {
		return data, true
	}
	if err := utils.ValidatorBody[request.UserSessionRevokeReq](ctx, &data, u.translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return request.UserSessionRevokeReq{}, false
	}
	return data, true
}

func sessionClosure(ctx *gin.Context, reason, fallback string) service.UserSessionClosure {
	operatorID, _ := ctx.Get("id")
	userValue, _ := ctx.Get("user")
	user, _ := userValue.(model.SysUser)
	operator, _ := operatorID.(int)
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = fallback
	}
	return service.UserSessionClosure{Reason: trimmedReason, OperatorID: operator, OperatorName: user.UserName}
}

// Events 保持一个轻量 SSE 连接。每五秒检查共享 Redis，因此多实例部署也能收到下线结果。
func (u *UserSessionController) Events(ctx *gin.Context) {
	userID, sessionID, ok := authenticatedSession(ctx)
	if !ok {
		_ = ctx.Error(appErrors.ErrUserNotLogin)
		return
	}
	expiresAt, _ := ctx.Get("token_expires_at")
	accessExpiresAt, _ := expiresAt.(time.Time)
	if err := http.NewResponseController(ctx.Writer).SetWriteDeadline(time.Time{}); err != nil {
		_ = ctx.Error(appErrors.WrapSystemError(err))
		return
	}
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")
	ctx.Status(http.StatusOK)
	writeSessionEvent(ctx, "connected", map[string]any{"connected": true})

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	keepaliveTicks := 0
	for {
		select {
		case <-ctx.Request.Context().Done():
			return
		case now := <-ticker.C:
			if !accessExpiresAt.IsZero() && !now.Before(accessExpiresAt.Add(-30*time.Second)) {
				writeSessionEvent(ctx, "access_expiring", map[string]string{"message": "Access Token 即将到期"})
				return
			}
			active, err := u.sessions.IsActive(userID, sessionID)
			if err != nil {
				writeSessionEvent(ctx, "check_failed", map[string]string{"message": "会话状态检查失败"})
				return
			}
			if !active {
				writeSessionEvent(ctx, "session_revoked", map[string]string{"message": u.sessions.ClosureReason(ctx.Request.Context(), userID, sessionID)})
				return
			}
			keepaliveTicks++
			if keepaliveTicks%12 == 0 {
				// SSE 连接仍在时每分钟更新活动时间，后台标签页也会保持在线状态。
				if err := u.sessions.Heartbeat(ctx.Request.Context(), userID, sessionID); err != nil {
					writeSessionEvent(ctx, "check_failed", map[string]string{"message": "会话心跳更新失败"})
					return
				}
			}
			if keepaliveTicks%3 == 0 {
				_, _ = fmt.Fprint(ctx.Writer, ": keepalive\n\n")
				ctx.Writer.Flush()
			}
		}
	}
}

func authenticatedSession(ctx *gin.Context) (int, string, bool) {
	userIDValue, userOK := ctx.Get("id")
	sessionIDValue, sessionOK := ctx.Get("auth_session_id")
	userID, idOK := userIDValue.(int)
	sessionID, sessionIDOK := sessionIDValue.(string)
	return userID, sessionID, userOK && sessionOK && idOK && sessionIDOK && userID > 0 && sessionID != ""
}

func writeSessionEvent(ctx *gin.Context, event string, data any) {
	payload, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(ctx.Writer, "event: %s\ndata: %s\n\n", event, payload)
	ctx.Writer.Flush()
}
