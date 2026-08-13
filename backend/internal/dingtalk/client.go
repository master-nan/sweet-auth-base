/**
 * @Author: Nan
 * @Date: 2024/11/12 10:48
 */

package dingtalk

import (
	"backend/internal/errors"
	"backend/internal/http"
	"encoding/json"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dingtalkoauth "github.com/alibabacloud-go/dingtalk/oauth2_1_0"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"
	"time"
)

const (

	// UserInfoURL 获取钉钉用户信息的 URL
	UserInfoURL = "https://oapi.dingtalk.com/topapi/v2/user/getuserinfo"

	// UserGetURL 获取钉钉用户详情的 URL
	UserGetURL = "https://oapi.dingtalk.com/topapi/v2/user/get"

	// SendMessageURL 发送钉钉消息的 URL
	SendMessageURL = "https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2"

	// UserGetByMobileURL 根据手机号获取用户信息的 URL
	UserGetByMobileURL = "https://oapi.dingtalk.com/topapi/v2/user/getbymobile"
)

type Client struct {
	config *openapi.Config
}

type AccessToken struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresInn"`
}

type UserInfo struct {
	UserId   string `json:"userId"`
	UserName string `json:"userName"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	IsAdmin  bool   `json:"isAdmin"`
}

type DingUserInfoResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Result  struct {
		UnionId string `json:"unionid"`
		UserId  string `json:"userid"`
		Name    string `json:"name"`
	} `json:"result"`
}

type DingUserGetResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Result  struct {
		UnionId string `json:"unionid"`
		UserId  string `json:"userid"`
		Name    string `json:"name"`
		Avatar  string `json:"avatar"`
		Mobile  string `json:"mobile"`
		Email   string `json:"email"`
		Admin   bool   `json:"admin"`
	} `json:"result"`
}

func NewClient() *Client {
	config := &openapi.Config{}
	config.Protocol = tea.String("https")
	config.RegionId = tea.String("central")
	return &Client{
		config,
	}
}

func (d *Client) GetClient() (*dingtalkoauth.Client, error) {
	client, err := dingtalkoauth.NewClient(d.config)
	return client, err
}

func (d *Client) GetAccessToken(appKey, appSecret string) (AccessToken, error) {
	var res AccessToken
	client, err := d.GetClient()
	if err != nil {
		return res, err
	}
	getAccessTokenRequest := &dingtalkoauth.GetAccessTokenRequest{
		AppKey:    tea.String(appKey),
		AppSecret: tea.String(appSecret),
	}
	result, err := client.GetAccessToken(getAccessTokenRequest)
	if err != nil {
		return res, err
	}
	res.AccessToken = *result.Body.AccessToken
	res.ExpiresIn = *result.Body.ExpireIn
	return res, nil
}

func (d *Client) GetUserInfo(accessToken string, code string) (DingUserInfoResponse, error) {
	var res DingUserInfoResponse
	httpClient := http.NewClient(60 * time.Second)
	body := map[string]interface{}{}
	body["code"] = code
	result, err := httpClient.Post(UserInfoURL+"?access_token="+accessToken, body, nil)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(result, &res)
	if err != nil {
		return res, err
	}
	if res.ErrCode != 0 {
		return res, errors.WrapDingTalkRequestFailed(fmt.Errorf("provider code %d", res.ErrCode))
	}
	return res, nil
}

func (d *Client) GetUser(accessToken string, code string) (UserInfo, error) {
	var res UserInfo
	dingTalkUserInfoResp, err := d.GetUserInfo(accessToken, code)
	if err != nil {
		return res, err
	}
	httpClient := http.NewClient(60 * time.Second)
	body := map[string]interface{}{}
	body["userid"] = dingTalkUserInfoResp.Result.UserId
	result, err := httpClient.Post(UserGetURL+"?access_token="+accessToken, body, nil)
	if err != nil {
		return res, err
	}
	var dingTalkUserGetResp DingUserGetResponse
	err = json.Unmarshal(result, &dingTalkUserGetResp)
	if err != nil {
		return res, err
	}
	if dingTalkUserGetResp.ErrCode != 0 {
		return res, errors.WrapDingTalkRequestFailed(fmt.Errorf("provider code %d", dingTalkUserGetResp.ErrCode))
	}
	zap.L().Debug("DingTalk GetUser succeeded")
	res.UserName = dingTalkUserGetResp.Result.Name
	res.UserId = dingTalkUserGetResp.Result.UserId
	res.Email = dingTalkUserGetResp.Result.Email
	res.Avatar = dingTalkUserGetResp.Result.Avatar
	res.IsAdmin = dingTalkUserGetResp.Result.Admin
	return res, nil
}

// GetUserByMobile 根据手机号获取用户信息
func (d *Client) GetUserByMobile(accessToken, mobile string) (string, error) {
	httpClient := http.NewClient(60 * time.Second)
	body := map[string]interface{}{}
	body["mobile"] = mobile
	result, err := httpClient.Post(UserGetByMobileURL+"?access_token="+accessToken, body, nil)
	if err != nil {
		return "", err
	}
	var res DingUserGetResponse
	err = json.Unmarshal(result, &res)
	if err != nil {
		return "", err
	}
	if res.ErrCode != 0 {
		return "", errors.WrapDingTalkRequestFailed(fmt.Errorf("provider code %d", res.ErrCode))
	}
	zap.L().Debug("DingTalk GetUserByMobile succeeded")
	return res.Result.UserId, nil
}

type SendMessageRequest struct {
	AgentId    string      `json:"agent_id"`
	UseridList string      `json:"userid_list,omitempty"`
	DeptIdList string      `json:"dept_id_list,omitempty"`
	ToAllUser  bool        `json:"to_all_user,omitempty"`
	Msg        interface{} `json:"msg"`
}

type Msg struct {
	MsgType string      `json:"msgtype"`
	Content interface{} `json:"content"`
}

// TextContent 文本消息内容
type TextContent struct {
	Content string `json:"content"`
}

// ImageContent 图片消息内容
type ImageContent struct {
	MediaId string `json:"media_id"`
}

// LinkContent 链接消息内容
type LinkContent struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageUrl string `json:"messageUrl"`
	PicUrl     string `json:"picUrl"`
}

// FileContent 文件消息内容
type FileContent struct {
	MediaId string `json:"media_id"`
}

// VoiceContent 语音消息内容
type VoiceContent struct {
	MediaId  string `json:"media_id"`
	Duration string `json:"duration"`
}

// OAContent OA消息内容
type OAContent struct {
	PCMessageURL string      `json:"pc_message_url"`
	MessageURL   string      `json:"message_url"`
	Head         OAHead      `json:"head"`
	Body         OABody      `json:"body"`
	StatusBar    OAStatusBar `json:"status_bar"`
}

type OAHead struct {
	BgColor string `json:"bgcolor"`
	Text    string `json:"text"`
}
type OABody struct {
	Author    string     `json:"author"`
	FileCount string     `json:"file_count"`
	Image     string     `json:"image"`
	Content   string     `json:"content"`
	Rich      OABodyRich `json:"rich"`
	Form      OABodyForm `json:"form"`
	Title     string     `json:"title"`
}

type OABodyRich struct {
	Unit string `json:"unit"`
	Num  string `json:"num"`
}
type OABodyForm struct {
	Value string `json:"value"`
	Key   string `json:"key"`
}
type OAStatusBar struct {
	StatusValue string `json:"status_value"`
	StatusBg    string `json:"status_bg"`
}

// MarkdownContent Markdown消息内容
type MarkdownContent struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// ActionCardContent ActionCard消息内容
type ActionCardContent struct {
	Title          string `json:"title"`
	Markdown       string `json:"markdown"`
	SingleTitle    string `json:"single_title"`
	SingleURL      string `json:"single_url"`
	BtnOrientation string `json:"btn_orientation"`
	BtnJsonList    string `json:"btn_json_list"`
}

type SendMessageResponse struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	TaskId    int64  `json:"task_id"`
	RequestId string `json:"request_id"`
}

// AsyncSendMessage 异步发送钉钉消息
func (d *Client) AsyncSendMessage(accessToken string, msg SendMessageRequest) error {
	httpClient := http.NewClient(60 * time.Second)
	resp, err := httpClient.Post(SendMessageURL+"?access_token="+accessToken, msg, nil)
	if err != nil {
		zap.L().Error("AsyncSendDingTalkMessage Error", zap.Error(err))
		return err
	}
	var sendMessageResponse SendMessageResponse
	err = json.Unmarshal(resp, &sendMessageResponse)
	if err != nil {
		zap.L().Error("AsyncSendDingTalkMessage Error", zap.Error(err))
		return err
	}
	if sendMessageResponse.ErrCode != 0 {
		zap.L().Error("AsyncSendDingTalkMessage Error", zap.Int("provider_code", sendMessageResponse.ErrCode))
		return errors.WrapDingTalkRequestFailed(fmt.Errorf("provider code %d", sendMessageResponse.ErrCode))
	}
	zap.L().Info("AsyncSendDingTalkMessage", zap.Int64("task_id", sendMessageResponse.TaskId), zap.String("request_id", sendMessageResponse.RequestId))
	return nil
}
