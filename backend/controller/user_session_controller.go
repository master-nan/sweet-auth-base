package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	appErrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	operatorID, _ := ctx.Get("id")
	reason := fmt.Sprintf("管理员 %v 已将该设备下线", operatorID)
	if err := u.sessions.RevokeSession(ctx.Request.Context(), id, reason); err != nil {
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
	operatorID, _ := ctx.Get("id")
	reason := fmt.Sprintf("管理员 %v 已将该用户的全部设备下线", operatorID)
	if err := u.sessions.RevokeUser(ctx.Request.Context(), userID, model.UserSessionStatusForcedOffline, reason); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(true)
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
			if !accessExpiresAt.IsZero() && !now.Before(accessExpiresAt) {
				writeSessionEvent(ctx, "access_expired", map[string]string{"message": "Access Token 已到期"})
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
