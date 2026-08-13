/**
 * @Author: Nan
 * @Date: 2024/11/12 16:13
 */

package service

import (
	"backend/dto/request"
	"backend/internal/cache"
	"backend/internal/dingtalk"
	myerrors "backend/internal/errors"
	"backend/model"
	"errors"
	"strings"

	"go.uber.org/zap"
)

type DingTalkService struct {
	dingTalkCache       *cache.DingTalkCache
	dingTalkUserIDCache *cache.DingTalkUserIDCache
}

func NewDingTalkService(dingTalkCache *cache.DingTalkCache, dingTalkUserIDCache *cache.DingTalkUserIDCache) *DingTalkService {
	return &DingTalkService{
		dingTalkCache,
		dingTalkUserIDCache,
	}
}

func (d *DingTalkService) GetAccessToken(application model.Application) (string, error) {
	dingTalkAccessToken, err := d.dingTalkCache.Get(application.AppKey)
	if err == nil {
		return dingTalkAccessToken.AccessToken, nil
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		zap.L().Error("DingTalkService GetAccessToken 1=======>", zap.Error(err))
		return "", err
	}
	// 从钉钉处获取access_token
	dingTalkClient := dingtalk.NewClient()
	dingTalkAccessToken, err = dingTalkClient.GetAccessToken(application.DingKey, application.DingSecret)
	if err != nil {
		zap.L().Error("DingTalkService GetAccessToken 2=======>", zap.Error(err))
		return "", err
	}
	// 缓存
	_ = d.dingTalkCache.SetExpiration(application.AppKey, dingTalkAccessToken, dingTalkAccessToken.ExpiresIn)
	return dingTalkAccessToken.AccessToken, nil
}

func (d *DingTalkService) GetIdentityPrincipal(accessToken, code string) (string, error) {
	// 从钉钉处获取用户信息
	dingTalkClient := dingtalk.NewClient()
	dingTailUser, err := dingTalkClient.GetUser(accessToken, code)
	if err != nil {
		zap.L().Error("DingTalkService GetUser 1=======>", zap.Error(err))
		return "", err
	}
	return strings.TrimSpace(dingTailUser.Email), nil
}

// SendMessage 发送消息
func (d *DingTalkService) SendMessage(accessToken string, data request.DingTalkMessageReq) error {
	dingTalkClient := dingtalk.NewClient()
	// data.UseridList是以逗号分隔的字符串，转换为数组
	users := strings.Split(data.UseridList, ",")
	userIds := make([]string, 0)
	if len(users) == 0 {
		data.ToAllUser = true
	} else {
		for _, u := range users {
			id, err := d.dingTalkUserIDCache.Get(u)
			if err != nil {
				if !errors.Is(err, cache.ErrCacheMiss) {
					zap.L().Error("DingTalkService SendMessage 1=======>", zap.Error(err))
					continue
				}
				id, err = dingTalkClient.GetUserByMobile(accessToken, u)
				if err != nil {
					zap.L().Error("DingTalkService SendMessage 1=======>", zap.Error(err))
					continue
				}
				go func() {
					_ = d.dingTalkUserIDCache.Set(u, id)
				}()
			}
			userIds = append(userIds, id)

		}
	}
	msgReq := dingtalk.SendMessageRequest{
		AgentId:    data.AgentId,
		UseridList: strings.Join(userIds, ","),
		DeptIdList: data.DeptIdList,
		ToAllUser:  data.ToAllUser,
	}
	switch data.MsgType {
	case request.TextType:
		msgReq.Msg = struct {
			MsgType string      `json:"msgtype"`
			Text    interface{} `json:"text"`
		}{
			MsgType: string(data.MsgType),
			Text:    data.Msg,
		}
	case request.ImageType:
		msgReq.Msg = struct {
			MsgType string      `json:"msgtype"`
			Image   interface{} `json:"image"`
		}{
			MsgType: string(data.MsgType),
			Image:   data.Msg,
		}
	case request.LinkType:
		msgReq.Msg = struct {
			MsgType string      `json:"msgtype"`
			Link    interface{} `json:"link"`
		}{
			MsgType: string(data.MsgType),
			Link:    data.Msg,
		}
	case request.FileType:
		msgReq.Msg = struct {
			MsgType string      `json:"msgtype"`
			File    interface{} `json:"file"`
		}{
			MsgType: string(data.MsgType),
			File:    data.Msg,
		}
	case request.VoiceType:
		msgReq.Msg = struct {
			MsgType string      `json:"msgtype"`
			Voice   interface{} `json:"voice"`
		}{
			MsgType: string(data.MsgType),
			Voice:   data.Msg,
		}
	case request.OAType:
		msgReq.Msg = struct {
			MsgType string      `json:"msgtype"`
			OA      interface{} `json:"oa"`
		}{
			MsgType: string(data.MsgType),
			OA:      data.Msg,
		}
	case request.MarkdownType:
		msgReq.Msg = struct {
			MsgType  string      `json:"msgtype"`
			Markdown interface{} `json:"markdown"`
		}{
			MsgType:  string(data.MsgType),
			Markdown: data.Msg,
		}
	case request.ActionCardType:
		msgReq.Msg = struct {
			MsgType    string      `json:"msgtype"`
			ActionCard interface{} `json:"action_card"`
		}{
			MsgType:    string(data.MsgType),
			ActionCard: data.Msg,
		}
	default:
		return myerrors.ErrDingTalkMsgTypeInvalid
	}
	err := dingTalkClient.AsyncSendMessage(accessToken, msgReq)
	return err
}
