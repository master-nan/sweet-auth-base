/**
 * @Author: Nan
 * @Date: 2024/10/21 17:28
 */

package api

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/errors"
	"backend/internal/security"
	"backend/internal/utils"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/jinzhu/copier"
)

type SysUserApi struct {
	sysUserService      *service.SysUserService
	sysConfigureService *service.SysConfigureService
	translators         map[string]ut.Translator
}

func NewSysUserApi(sysUserService *service.SysUserService, sysConfigureService *service.SysConfigureService, translators map[string]ut.Translator) *SysUserApi {
	return &SysUserApi{
		sysUserService,
		sysConfigureService,
		translators,
	}
}

// UpdatePassword 修改密码
// @Summary 修改密码
// @Description 修改密码
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Param data body request.SysUserUpdatePasswordReq true "修改密码"
// @Success 200 {object} response.Response "请求成功"
// @Router /api/user/password [post]
func (u *SysUserApi) UpdatePassword(ctx *gin.Context) {
	if data, exists := ctx.Get("id"); exists {
		resp := response.NewResponse()
		ctx.Set("response", resp)
		id := data.(int)
		var data request.SysUserUpdatePasswordReq
		data.Id = id
		translator := u.translators["zh"]
		err := utils.ValidatorBody[request.SysUserUpdatePasswordReq](ctx, &data, translator)
		if err != nil {
			_ = ctx.Error(err)
			return
		}

		cfg, err := u.sysConfigureService.Query()
		if err != nil {
			_ = ctx.Error(err)
			return
		}
		if err := security.ValidatePasswordByConfigure(data.Password, cfg); err != nil {
			_ = ctx.Error(err)
			return
		}

		err = u.sysUserService.UpdatePassword(ctx, data)
		if err != nil {
			_ = ctx.Error(err)
			return
		}
	} else {
		_ = ctx.Error(errors.ErrParamInvalid)
	}
}

// GetMe 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前用户信息
// @Tags 用户
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Success 200 {object} response.Response "请求成功"
// @Router /api/user/me [get]
func (u *SysUserApi) GetMe(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	user := ctx.MustGet("user").(model.SysUser)
	var userRes response.SysUserRes
	err := copier.Copy(&userRes, &user)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(userRes)
}
