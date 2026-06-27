/**
 * @Author: Nan
 * @Date: 2024/11/12 17:37
 */

package api

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type DingTalkApi struct {
	applicationService *service.ApplicationService
	dingTalkService    *service.DingTalkService
	translators        map[string]ut.Translator
}

func NewDingTalkApi(applicationService *service.ApplicationService, dingTalkService *service.DingTalkService, translators map[string]ut.Translator) *DingTalkApi {
	return &DingTalkApi{
		applicationService,
		dingTalkService,
		translators,
	}
}

// SendMessage 发送消息
// @Summary 发送消息
// @Tags 钉钉
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param X-APP-TOKEN header string true "自定义应用令牌"
// @Param data body request.DingTalkMessageReq true "消息内容"
// @Success 200 {object} response.Response
// @Router /api/dingtalk/send_message [post]
func (d *DingTalkApi) SendMessage(ctx *gin.Context) {
	var data request.DingTalkMessageReq
	resp := response.NewResponse()
	ctx.Set("response", resp)
	translator := d.translators["zh"]
	err := utils.ValidatorBody[request.DingTalkMessageReq](ctx, &data, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if appData, exists := ctx.Get("application"); exists {
		application := appData.(model.Application)
		// 获取钉钉访问令牌
		accessToken, err := d.dingTalkService.GetAccessToken(application)
		if err != nil {
			_ = ctx.Error(err)
			return
		}
		// 判断data.AgentId是否为空字符串
		if data.AgentId == "" {
			data.AgentId = application.DingAppID
		}
		if data.UseridList == "" && data.DeptIdList == "" && !data.ToAllUser {
			_ = ctx.Error(errors.ErrDingTalkRecipientEmpty)
			return
		}
		data.ToAllUser = false
		// 异步发送消息
		go d.dingTalkService.SendMessage(accessToken, data)
		resp.SetData("消息发送成功")
	} else {
		_ = ctx.Error(errors.ErrAppUnauthorized)
	}
}
