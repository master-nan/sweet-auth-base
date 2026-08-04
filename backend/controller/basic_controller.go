package controller

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/cache"
	myerrors "backend/internal/errors"
	"backend/internal/token"
	"backend/internal/utils"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"bytes"
	"strconv"
	"time"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

type BasicController struct {
	tokenGenerator      token.JWTToken
	serverConfig        *config.Server
	sysConfigureService *service.SysConfigureService
	logService          *service.LogService
	sysUserService      *service.SysUserService
	tokenBlackCache     *cache.TokenBlackCache
	loginAttemptCache   *cache.LoginAttemptCache
	translators         map[string]ut.Translator
}

func NewBasicController(tokenGenerator token.JWTToken, serverConfig *config.Server, sysConfigureService *service.SysConfigureService, logService *service.LogService, sysUserService *service.SysUserService, tokenBlackCache *cache.TokenBlackCache, loginAttemptCache *cache.LoginAttemptCache, translators map[string]ut.Translator) *BasicController {
	return &BasicController{
		tokenGenerator,
		serverConfig,
		sysConfigureService,
		logService,
		sysUserService,
		tokenBlackCache,
		loginAttemptCache,
		translators,
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
	configUre, err := b.sysConfigureService.Query()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if b.loginAttemptCache != nil {
		locked, lockErr := b.loginAttemptCache.IsLocked(data.UserName)
		if lockErr != nil {
			zap.L().Warn("failed to check login lock", zap.Error(lockErr))
			_ = ctx.Error(lockErr)
			return
		}
		if locked {
			_ = ctx.Error(myerrors.ErrLoginLocked)
			return
		}
	}
	if configUre.EnableCaptcha {
		boolean := captcha.VerifyString(data.CaptchaId, data.Captcha)
		if boolean == false {
			_ = ctx.Error(myerrors.ErrCaptchaInvalid)
			return
		}
	}
	var loginLog = model.LoginLog{
		Ip:       ctx.ClientIP(),
		Locality: "",
		UserName: data.UserName,
	}
	b.logService.CreateLoginLogAsync(middleware.DetachedTaskContext(ctx), loginLog)
	user, err := b.sysUserService.GetByUserName(data.UserName)
	if err != nil || user.Id == 0 || utils.Encryption(data.Password, strconv.Itoa(user.Id)+b.serverConfig.Conf.Salt) != user.Password || !user.State {
		if b.loginAttemptCache != nil {
			locked, cacheErr := b.loginAttemptCache.RecordFailure(data.UserName, configUre.PasswordErrorCount, time.Duration(configUre.PasswordLockMinutes)*time.Minute)
			if cacheErr != nil {
				zap.L().Warn("failed to record login failure", zap.Error(cacheErr))
				_ = ctx.Error(cacheErr)
				return
			}
			if locked {
				_ = ctx.Error(myerrors.ErrLoginLocked)
				return
			}
		}
		_ = ctx.Error(myerrors.ErrUserNotFound)
		return
	}
	if b.loginAttemptCache != nil {
		if err = b.loginAttemptCache.Clear(data.UserName); err != nil {
			zap.L().Warn("failed to clear login failure", zap.Error(err))
			_ = ctx.Error(err)
			return
		}
	}
	conf := token.Config{
		Issuer:                 b.serverConfig.Name,
		SecretKey:              b.serverConfig.Conf.Salt,
		AccessTokenExpiration:  7200,
		RefreshTokenExpiration: 60 * 60 * 24 * 30,
	}
	claimsAccess := token.Claims{
		ID:        strconv.Itoa(user.Id),
		Type:      enum.AccessToken,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(7200 * time.Second),
		NotBefore: time.Now(),
	}

	accessToken, err := b.tokenGenerator.GenerateToken(claimsAccess, conf)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	claimsRefresh := token.Claims{
		ID:        strconv.Itoa(user.Id),
		Type:      enum.RefreshToken,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(60 * 60 * 24 * 30 * time.Second),
		NotBefore: time.Now(),
	}
	refreshToken, err := b.tokenGenerator.GenerateToken(claimsRefresh, conf)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	mustChangePassword, changeReason := service.PasswordChangeRequirement(user, configUre, time.Now())
	lastLogin := model.CustomTime(time.Now())
	b.sysUserService.UpdateLoginStateAsync(
		middleware.DetachedTaskContext(ctx).WithActor(user.Id, user.UserName),
		user.Id,
		utils.UpdateAccessTokens(user.AccessTokens, accessToken),
		lastLogin,
	)
	signInRes := response.SignInRes{
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		MustChangePassword:   mustChangePassword,
		PasswordChangeReason: changeReason,
	}
	resp.SetData(signInRes)
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
	configUre, err := b.sysConfigureService.Query()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var configureRes response.PublicConfigureRes
	if err = copier.Copy(&configureRes, &configUre); err != nil {
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
	configUre, err := b.sysConfigureService.Query()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var configureRes response.ConfigureRes
	if err = copier.Copy(&configureRes, &configUre); err != nil {
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
	err = b.sysConfigureService.Update(ctx, data)
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
	result, err := b.logService.QueryAccessLogs(ctx, data)
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
	logData, err := b.logService.GetAccessLogById(ctx, id)
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
	authorization := ctx.GetHeader("Authorization")
	if len(authorization) < len("Bearer ") {
		_ = ctx.Error(myerrors.ErrUserNotLogin)
		return
	}
	_, err := b.tokenBlackCache.Set(enum.AccessToken, authorization[len("Bearer "):])
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}
