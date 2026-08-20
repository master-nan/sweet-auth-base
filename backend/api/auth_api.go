/**
 * @Author: Nan
 * @Date: 2024/10/21 17:08
 */

package api

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/errors"
	"backend/internal/token"
	"backend/internal/utils"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"crypto/sha256"
	"crypto/subtle"
	stderrors "errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin/binding"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

const defaultSmsSendIntervalSeconds = 60

type AuthApi struct {
	authService        *service.AuthApplicationService
	serverConfig       *config.Server
	applicationService *service.ApplicationService
	smsService         *service.SmsService
	applicationCache   *cache.ApplicationCache
	sendCodeCache      *cache.SendCodeCache
	translators        map[string]ut.Translator
	hmacToken          token.HMACToken
}

func NewAuthApi(authService *service.AuthApplicationService, serverConfig *config.Server, applicationService *service.ApplicationService, smsService *service.SmsService, applicationCache *cache.ApplicationCache, sendCodeCache *cache.SendCodeCache, translators map[string]ut.Translator, hmacToken token.HMACToken) *AuthApi {
	return &AuthApi{
		authService,
		serverConfig,
		applicationService,
		smsService,
		applicationCache,
		sendCodeCache,
		translators,
		hmacToken,
	}
}

// GetAppToken 获取AppToken
// @Summary 获取AppToken
// @Description 获取AppToken
// @Tags 授权
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param X-APP-TOKEN header string false "自定义应用令牌"
// @Param data body request.AppTokenReq true "AppToken"
// @Success 200 {object} response.Response
// @Router /api/app_token [post]
func (a *AuthApi) GetAppToken(ctx *gin.Context) {
	var data request.AppTokenReq
	resp := response.NewResponse()
	ctx.Set("response", resp)
	translator := a.translators["zh"]
	err := utils.ValidatorBody[request.AppTokenReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	// 获取AppId对应的配置
	application, err := a.applicationService.GetApplicationByAppKey(data.AppKey)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if application.Id == 0 || !application.State || subtle.ConstantTimeCompare([]byte(application.AppSecret), []byte(data.AppSecret)) != 1 {
		_ = ctx.Error(errors.ErrAppNotFound)
		return
	}
	claims := token.Claims{
		ID:        strconv.Itoa(application.Id),
		Type:      enum.AppToken,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(application.Expiration) * time.Second),
		NotBefore: time.Now(),
	}
	conf := token.Config{
		Issuer:                 strconv.Itoa(application.Id),
		SecretKey:              application.AppSecret,
		AccessTokenExpiration:  application.Expiration,
		RefreshTokenExpiration: 0,
	}
	hmacToken, err := a.hmacToken.GenerateToken(claims, conf)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = a.applicationCache.SetExpiration(hmacToken, application, application.Expiration)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	appTokenRes := &response.AppTokenRes{
		AppToken:  hmacToken,
		ExpiresIn: application.Expiration,
	}
	resp.SetData(appTokenRes)
}

// SendSms 发送短信
// @Summary 发送短信
// @Description 发送短信
// @Tags 授权
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Param data body map[string]interface{} false "发送短信"
// @Param mobile path string true "手机号"
// @Param templateCode path string true "短信模板编号"
// @Success 200 {object} response.Response
// @Router /api/send_sms/{mobile}/{templateCode} [post]
func (a *AuthApi) SendSms(ctx *gin.Context) {
	var data map[string]interface{}
	resp := response.NewResponse()
	ctx.Set("response", resp)
	// 获取手机号和模板编号
	mobile := ctx.Param("mobile")
	// 判断手机号是否合法
	if !utils.IsMobile(mobile) {
		_ = ctx.Error(errors.ErrMobileInvalid)
		return
	}
	templateCode := ctx.Param("templateCode")
	// 获取请求参数
	err := ctx.ShouldBindBodyWith(&data, binding.JSON)
	if err != nil {
		// 判断错误是否是空内容导致，如果是则赋值为空 map，允许通过
		if stderrors.Is(err, io.EOF) {
			data = map[string]interface{}{}
		} else {
			_ = ctx.Error(errors.WrapParameterError(err, "请求参数格式错误"))
			return
		}
	}
	applicationValue, exists := ctx.Get("application")
	if !exists {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	application, ok := applicationValue.(model.Application)
	if !ok {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	rateLimitKey := smsSendRateLimitKey(application.Id, templateCode, mobile)
	if err := reserveSmsSendSlot(a.sendCodeCache, rateLimitKey, smsSendIntervalSeconds(a.serverConfig)); err != nil {
		_ = ctx.Error(err)
		return
	}
	// 发送短信
	tempParam, err := a.smsService.SendSms(
		middleware.DetachedTaskContext(ctx),
		application,
		templateCode,
		mobile,
		data,
	)
	if err != nil {
		_ = a.sendCodeCache.Delete(rateLimitKey)
		_ = ctx.Error(err)
		return
	}
	// 判断是否为验证码短信
	if code, ok := smsVerificationCodeFromParams(tempParam); ok {
		if err := a.sendCodeCache.Set(cache.SMSVerificationKey(application.Id, mobile), code); err != nil {
			_ = ctx.Error(err)
			return
		}
	}
	resp.SetData(map[string]interface{}{"sent": true})
}

func smsVerificationCodeFromParams(params map[string]interface{}) (string, bool) {
	if len(params) != 1 {
		return "", false
	}
	codeValue, ok := params["code"]
	if !ok {
		return "", false
	}
	code, ok := codeValue.(string)
	if !ok || code == "" {
		return "", false
	}
	return code, true
}

func smsSendIntervalSeconds(serverConfig *config.Server) int64 {
	if serverConfig == nil || serverConfig.ALiYun.SMS.SendIntervalSeconds <= 0 {
		return defaultSmsSendIntervalSeconds
	}
	return int64(serverConfig.ALiYun.SMS.SendIntervalSeconds)
}

func smsSendRateLimitKey(applicationId int, templateCode, mobile string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s", applicationId, templateCode, mobile)))
	return fmt.Sprintf("SMS_SEND_RATE_LIMIT_%x", sum)
}

func reserveSmsSendSlot(sendCodeCache *cache.SendCodeCache, key string, intervalSeconds int64) error {
	if sendCodeCache.Exists(key) {
		return errors.ErrSmsSendTooFrequent
	}
	return sendCodeCache.SetExpiration(key, "1", intervalSeconds)
}

// Login 登录
// @Summary 登录
// @Description 登录
// @Tags 授权
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Param data body request.SignInReq true "登录信息"
// @Success 200 {object} response.Response
// @Router /api/login [post]
func (a *AuthApi) Login(ctx *gin.Context) {
	var data request.SignInReq
	resp := response.NewResponse()
	ctx.Set("response", resp)
	translator := a.translators["zh"]
	err := utils.ValidatorBody[request.SignInReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := a.authService.Authenticate(ctx.Request.Context(), service.AuthenticationRequest{
		Channel: service.AuthChannelAPIPassword, CredentialType: service.AuthCredentialPassword,
		Principal: data.UserName, Secret: data.Password,
	})
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(authSignInResponse(result))
}

// RefreshToken 刷新Token
// @Summary 刷新Token
// @Description 刷新Token
// @Tags 授权
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param refreshToken query string true "刷新Token"
// @Success 200 {object} response.Response "请求成功"
// @Router /api/refresh_token [get]
func (a *AuthApi) RefreshToken(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	refreshToken := ctx.Query("refreshToken")
	if refreshToken == "" {
		refreshToken = ctx.Query("refresh_token")
	}
	if refreshToken == "" {
		_ = ctx.Error(errors.ErrInvalidRefreshToken)
		return
	}
	result, err := a.authService.Refresh(ctx.Request.Context(), refreshToken)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(authSignInResponse(result))
}

// Logout 退出登录
// @Summary 退出登录
// @Description 退出登录
// @Tags 授权
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Success 200 {object} response.Response  "请求成功"
// @Router /api/logout [post]
func (a *AuthApi) Logout(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	authorization := ctx.GetHeader("Authorization")
	if len(authorization) < len("Bearer ") {
		_ = ctx.Error(errors.ErrUserNotLogin)
		return
	}
	if err := a.authService.Logout(ctx.Request.Context(), authorization[len("Bearer "):]); err != nil {
		_ = ctx.Error(err)
		return
	}
}

// SSOLogin 单点登录
// @Summary 单点登录
// @Description 单点登录
// @Tags 授权
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Param code query string true "code"
// @Success 200 {object} response.Response
// @Router /api/sso_login [get]
func (a *AuthApi) SSOLogin(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	code := ctx.Query("code")
	if code == "" {
		_ = ctx.Error(errors.ErrParamInvalid)
		return
	}
	data, exists := ctx.Get("application")
	if !exists {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	application, ok := data.(model.Application)
	if !ok {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	result, err := a.authService.Authenticate(ctx.Request.Context(), service.AuthenticationRequest{
		Channel: service.AuthChannelDingTalk, CredentialType: service.AuthCredentialDingTalk,
		Secret: code, Application: application,
	})
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(authSignInResponse(result))
}

// SmsCodeLogin 短信验证码登录
// @Summary 短信验证码登录
// @Description 短信验证码登录
// @Tags 授权
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Param data body request.SmsCodeLoginReq true "短信验证码登录"
// @Success 200 {object} response.Response
// @Router /api/sms_code_login [post]
func (a *AuthApi) SmsCodeLogin(ctx *gin.Context) {
	var data request.SmsCodeLoginReq
	resp := response.NewResponse()
	ctx.Set("response", resp)
	translator := a.translators["zh"]
	err := utils.ValidatorBody[request.SmsCodeLoginReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	d, exists := ctx.Get("application")
	if !exists {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	application, ok := d.(model.Application)
	if !ok {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	result, err := a.authService.Authenticate(ctx.Request.Context(), service.AuthenticationRequest{
		Channel: service.AuthChannelSMS, CredentialType: service.AuthCredentialSMS,
		Principal: data.Mobile, Secret: data.Code, Application: application,
	})
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(authSignInResponse(result))
}

func authSignInResponse(result service.AuthenticationResult) response.SignInRes {
	return response.SignInRes{
		AccessToken: result.AccessToken, RefreshToken: result.RefreshToken,
		MustChangePassword: result.MustChangePassword, PasswordChangeReason: result.PasswordChangeReason,
	}
}

// CheckSmsStatus 检查短信发送状态
// @Summary 检查短信发送状态
// @Description 检查短信发送状态
// @Tags 授权
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Param bizId path string true "bizId"
// @Param mobile path string true "手机号"
// @Success 200 {object} response.Response
// @Router /api/check_sms_status/{bizId}/{mobile} [get]
func (a *AuthApi) CheckSmsStatus(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	bizId := ctx.Param("bizId")
	mobile := ctx.Param("mobile")
	if bizId == "" || mobile == "" {
		_ = ctx.Error(errors.ErrParamInvalid)
		return
	}
	applicationValue, exists := ctx.Get("application")
	if !exists {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	application, ok := applicationValue.(model.Application)
	if !ok {
		_ = ctx.Error(errors.ErrAppUnauthorized)
		return
	}
	result, err := a.smsService.CheckSmsStatus(ctx.Request.Context(), application.Id, bizId, mobile)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}
