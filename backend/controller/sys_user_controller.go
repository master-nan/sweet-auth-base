/**
 * @Author: Nan
 * @Date: 2024/6/28 上午11:46
 */

package controller

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/cache"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/jinzhu/copier"
)

type UserController struct {
	sysUserService      *service.SysUserService
	sysConfigureService *service.SysConfigureService
	translators         map[string]ut.Translator
	serverConfig        *config.Server
	sysTableService     *service.SysTableService
	loginAttemptCache   *cache.LoginAttemptCache
}

func NewUserController(
	sysUserService *service.SysUserService,
	sysConfigureService *service.SysConfigureService,
	translators map[string]ut.Translator,
	serverConfig *config.Server,
	sysTableService *service.SysTableService,
	loginAttemptCache *cache.LoginAttemptCache,
) *UserController {
	return &UserController{
		sysUserService,
		sysConfigureService,
		translators,
		serverConfig,
		sysTableService,
		loginAttemptCache,
	}
}

func (u *UserController) QuerySysUser(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.Basic
	translator := u.translators["zh"]
	err := utils.ValidatorBody[request.Basic](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	table, err := u.sysTableService.GetTableByTableCode(data.TableCode)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := u.sysUserService.GetUserList(&data, table)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	users := make([]response.SysUserRes, 0, len(result.Data))
	for _, user := range result.Data {
		var userRes response.SysUserRes
		copier.Copy(&userRes, &user)
		users = append(users, userRes)
	}
	resp.SetData(users).SetTotal(result.Total)
}

// GetMe 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前用户信息
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} response.Response
// @Router /admin/user/me [get]
func (u *UserController) GetMe(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	user := ctx.MustGet("user").(model.SysUser)
	var userRes response.SysUserRes
	copier.Copy(&userRes, &user)
	resp.SetData(userRes)
}

// UpdatePassword 修改密码
// @Summary 修改密码
// @Description 修改密码
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.SysUserUpdatePasswordReq true "修改密码"
// @Success 200 {object} response.Response
// @Router /admin/user/password [post]
func (u *UserController) UpdatePassword(ctx *gin.Context) {
	if data, exists := ctx.Get("id"); exists {
		resp := response.NewResponse()
		ctx.Set("response", resp)
		id := data.(int)
		var req request.SysUserUpdatePasswordReq
		req.Id = id
		translator := u.translators["zh"]
		err := utils.ValidatorBody[request.SysUserUpdatePasswordReq](ctx, &req, translator)
		if err != nil {
			_ = ctx.Error(err)
			return
		}

		cfg, err := u.sysConfigureService.Query()
		if err != nil {
			_ = ctx.Error(err)
			return
		}
		if err = security.ValidatePasswordByConfigure(req.Password, cfg); err != nil {
			_ = ctx.Error(err)
			return
		}
		err = u.sysUserService.UpdatePassword(ctx.Request.Context(), req)
		if err != nil {
			_ = ctx.Error(err)
			return
		}
		return
	}
	_ = ctx.Error(myerrors.ErrParamInvalid)
}

// GetUserByUserName 根据用户名获取用户
// @Summary 根据用户名获取用户
// @Description 根据用户名获取用户
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param username path string true "用户名"
// @Success 200 {object} response.Response
// @Router /admin/user/{username} [get]
func (u *UserController) GetUserByUserName(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	username := ctx.Param("username")
	data, err := u.sysUserService.GetByUserName(username)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var userRes response.SysUserRes
	copier.Copy(&userRes, &data)
	resp.SetData(userRes)
}

// GetUserById 根据ID获取用户
// @Summary 根据ID获取用户
// @Description 根据ID获取用户
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /admin/user/{id} [get]
func (u *UserController) GetUserById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	data, err := u.sysUserService.GetById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var userRes response.SysUserRes
	copier.Copy(&userRes, &data)
	resp.SetData(userRes)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建用户
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param data body request.SysUserCreateReq true "创建用户"
// @Success 200 {object} response.Response
// @Router /admin/user [post]
func (u *UserController) CreateUser(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	var data request.SysUserCreateReq
	translator := u.translators["zh"]
	err := utils.ValidatorBody[request.SysUserCreateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	cfg, err := u.sysConfigureService.Query()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err = security.ValidatePasswordByConfigure(data.Password, cfg); err != nil {
		_ = ctx.Error(err)
		return
	}
	err = u.sysUserService.Create(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新用户
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "用户ID"
// @Param data body request.SysUserUpdateReq true "更新用户"
// @Success 200 {object} response.Response
// @Router /admin/user/{id} [put]
func (u *UserController) UpdateUser(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.SysUserUpdateReq
	data.Id = id
	translator := u.translators["zh"]
	err = utils.ValidatorBody[request.SysUserUpdateReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = u.sysUserService.Update(ctx.Request.Context(), data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// AssignRoles 分配用户角色
func (u *UserController) AssignRoles(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	var data request.SysUserAssignRolesReq
	translator := u.translators["zh"]
	err = utils.ValidatorBody[request.SysUserAssignRolesReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = u.sysUserService.AssignRoles(ctx.Request.Context(), id, data.RoleIds)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(true)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除用户
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /admin/user/{id} [delete]
func (u *UserController) DeleteUser(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = u.sysUserService.Delete(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// ResetPassword 重置密码
// @Summary 重置密码
// @Description 重置密码
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /admin/user/reset_password/{id} [post]
func (u *UserController) ResetPassword(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user, err := u.sysUserService.GetById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if user.Id == 0 {
		_ = ctx.Error(myerrors.ErrUserNotExist)
		return
	}
	cfg, err := u.sysConfigureService.Query()
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	password, err := security.GeneratePasswordByConfigure(cfg)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	err = u.sysUserService.ResetPassword(ctx.Request.Context(), id, password)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	emailSent, emailMessage := sendResetPasswordEmail(cfg, user, password)
	resp.SetData(response.ResetPasswordRes{
		TemporaryPassword:  password,
		MustChangePassword: true,
		EmailSent:          emailSent,
		EmailMessage:       emailMessage,
	})
}

// UnlockLogin 解除登录锁定
// @Summary 解除登录锁定
// @Description 清除用户后台登录失败次数和锁定状态
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /admin/user/unlock_login/{id} [post]
func (u *UserController) UnlockLogin(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	user, err := u.sysUserService.GetById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if user.Id == 0 {
		_ = ctx.Error(myerrors.ErrUserNotExist)
		return
	}
	if u.loginAttemptCache != nil {
		if err := u.loginAttemptCache.Clear(user.UserName); err != nil {
			_ = ctx.Error(err)
			return
		}
	}
	resp.SetData(true)
}

func sendResetPasswordEmail(cfg model.SysConfigure, user model.SysUser, password string) (bool, string) {
	if !cfg.EnableEmail {
		return false, "邮件服务未启用，需手工告知临时密码"
	}
	recipient := strings.TrimSpace(user.Email)
	if recipient == "" {
		return false, "用户未配置邮箱，需手工告知临时密码"
	}
	userName := strings.TrimSpace(user.UserName)
	if userName == "" {
		userName = "用户"
	}
	subject := "Sweet Admin 密码重置通知"
	body := fmt.Sprintf(
		"您好，%s：\n\n您的 Sweet Admin 登录密码已由管理员重置。\n临时密码：%s\n\n请在下次登录后立即修改密码。如非本人或管理员操作，请及时联系系统管理员。",
		userName,
		password,
	)
	if err := service.SendEmailWithConfigure(cfg, recipient, subject, body); err != nil {
		return false, "邮件发送失败，需手工告知临时密码"
	}
	return true, "临时密码已发送至用户邮箱"
}
