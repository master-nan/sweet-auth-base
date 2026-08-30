package controller

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/service"
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type BasicController struct {
	authService         *service.AuthApplicationService
	sysConfigureService *service.SysConfigureService
	logService          *service.LogService
	translators         map[string]ut.Translator
	serverConfig        *config.Server
}

func NewBasicController(authService *service.AuthApplicationService, sysConfigureService *service.SysConfigureService, logService *service.LogService, translators map[string]ut.Translator, serverConfig *config.Server) *BasicController {
	return &BasicController{
		authService,
		sysConfigureService,
		logService,
		translators,
		serverConfig,
	}
}

// Login 登录
// @Summary 登录
// @Description 登录
// @Tags 登录
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param data body request.SignInReq true "登录"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/login [post]
func (b *BasicController) Login(ctx *gin.Context) {
	var data request.SignInReq
	resp := response.NewResponse()
	ctx.Set("response", resp)
	translator := b.translators["zh"]
	err := utils.ValidatorBody[request.SignInReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := b.authService.Authenticate(ctx.Request.Context(), service.AuthenticationRequest{
		Channel: service.AuthChannelAdminPassword, CredentialType: service.AuthCredentialPassword,
		Principal: data.UserName, Secret: data.Password, CaptchaID: data.CaptchaId, Captcha: data.Captcha,
		Client: service.UserSessionClient{IPAddress: ctx.ClientIP(), UserAgent: ctx.Request.UserAgent()},
	})
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	b.setRefreshCookie(ctx, result.RefreshToken, 30*24*time.Hour)
	signInRes := response.SignInRes{
		AccessToken:        result.AccessToken,
		MustChangePassword: result.MustChangePassword, PasswordChangeReason: result.PasswordChangeReason,
	}
	resp.SetData(signInRes)
}

// Refresh 使用 HttpOnly Cookie 中的 Refresh Token 换取新的 Access Token，并同时轮换 Cookie。
func (b *BasicController) Refresh(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	refreshToken, err := ctx.Cookie(refreshTokenCookieName)
	if err != nil || strings.TrimSpace(refreshToken) == "" {
		_ = ctx.Error(myerrors.ErrInvalidRefreshToken)
		return
	}
	result, err := b.authService.Refresh(ctx.Request.Context(), refreshToken, service.UserSessionClient{
		IPAddress: ctx.ClientIP(), UserAgent: ctx.Request.UserAgent(), Channel: string(service.AuthChannelRefresh),
	})
	if err != nil {
		b.clearRefreshCookie(ctx)
		_ = ctx.Error(err)
		return
	}
	b.setRefreshCookie(ctx, result.RefreshToken, 30*24*time.Hour)
	resp.SetData(response.SignInRes{AccessToken: result.AccessToken})
}

// Captcha 验证码
// @Summary 验证码
// @Description 验证码
// @Tags 验证码
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/captcha [get]
func (b *BasicController) Captcha(ctx *gin.Context) {
	l := captcha.DefaultLen
	w, h := 110, 50
	captchaId := captcha.NewLen(l)
	var content bytes.Buffer
	_ = captcha.WriteImage(&content, captchaId, w, h)
	imageData := content.Bytes()
	// 返回JSON数据，包含captchaId和图片的base64编码
	resp := response.NewResponse()
	ctx.Set("response", resp)
	captchaRes := response.CaptchaRes{
		CaptchaId: captchaId,
		Image:     imageData,
	}
	resp.SetData(captchaRes)
}

// Configure 配置
// @Summary 配置
// @Description 配置
// @Tags 配置
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/configure [get]
func (b *BasicController) Configure(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	configureRes, err := b.sysConfigureService.QueryPublicResponse()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(configureRes)
}

// ConfigureDetail 配置详情
// @Summary 配置详情
// @Description 获取后台配置详情
// @Tags 配置
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/configure/detail [get]
func (b *BasicController) ConfigureDetail(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	configureRes, err := b.sysConfigureService.QueryDetailResponse()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(configureRes)
}

// UpdateConfigure 更新配置
// @Summary 更新配置
// @Description 更新配置
// @Tags 配置
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param b body request.ConfigureUpdateReq  true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/configure/{id} [put]
func (b *BasicController) UpdateConfigure(ctx *gin.Context) {
	var data request.ConfigureUpdateReq
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	translator := b.translators["zh"]
	err = utils.ValidatorBody[request.ConfigureUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data.Id = id
	err = b.sysConfigureService.Update(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// TestConfigureEmail 发送配置测试邮件
// @Summary 发送配置测试邮件
// @Description 使用已保存的邮件配置发送一封测试邮件
// @Tags 配置
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param b body request.ConfigureTestEmailReq true "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/configure/test-email [post]
func (b *BasicController) TestConfigureEmail(ctx *gin.Context) {
	var data request.ConfigureTestEmailReq
	resp := response.NewResponse()
	ctx.Set("response", resp)
	translator := b.translators["zh"]
	err := utils.ValidatorBody[request.ConfigureTestEmailReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := b.sysConfigureService.SendTestEmail(data.To); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(map[string]bool{"sent": true})
}

// QueryAccessLogs 查询访问审计日志
// @Summary 访问审计日志
// @Description 查询访问审计日志
// @Tags 日志
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param q body request.AccessLogQueryReq false "请求参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/log/access/query [post]
func (b *BasicController) QueryAccessLogs(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.AccessLogQueryReq
	translator := b.translators["zh"]
	err := utils.ValidatorBody[request.AccessLogQueryReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := b.logService.QueryAccessLogsResponse(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result.Data).SetTotal(result.Total)
}

// GetAccessLogById 查询访问审计详情
// @Summary 访问审计详情
// @Description 查询访问审计详情
// @Tags 日志
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "日志ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/log/access/{id} [get]
func (b *BasicController) GetAccessLogById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	logData, err := b.logService.GetAccessLogByIdResponse(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(logData)
}

// Logout 退出
// @Summary 退出
// @Description 退出
// @Tags 退出
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/logout [post]
func (b *BasicController) Logout(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	defer b.clearRefreshCookie(ctx)
	authorization := ctx.GetHeader("Authorization")
	if len(authorization) >= len("Bearer ") {
		if err := b.authService.Logout(ctx.Request.Context(), authorization[len("Bearer "):]); err == nil {
			return
		}
	}
	if refreshToken, err := ctx.Cookie(refreshTokenCookieName); err == nil && strings.TrimSpace(refreshToken) != "" {
		// 退出接口保持幂等：Token 已被管理员撤销时，清除浏览器 Cookie 即可。
		_ = b.authService.LogoutWithRefresh(ctx.Request.Context(), refreshToken)
	}
}

const refreshTokenCookieName = "sweet_refresh_token"

func (b *BasicController) setRefreshCookie(ctx *gin.Context, value string, ttl time.Duration) {
	secure := ctx.Request.TLS != nil || strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https") || isProduction(b.serverConfig.Environment)
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name: refreshTokenCookieName, Value: value, Path: "/" + strings.Trim(b.serverConfig.Name, "/") + "/admin",
		MaxAge: int(ttl / time.Second), Expires: time.Now().Add(ttl), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func (b *BasicController) clearRefreshCookie(ctx *gin.Context) {
	secure := ctx.Request.TLS != nil || strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https") || isProduction(b.serverConfig.Environment)
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name: refreshTokenCookieName, Value: "", Path: "/" + strings.Trim(b.serverConfig.Name, "/") + "/admin",
		MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func isProduction(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "pro", "prod", "production":
		return true
	default:
		return false
	}
}
