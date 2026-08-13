/**
 * @Author: Nan
 * @Date: 2025/2/8 13:59
 */

package sms

import (
	error2 "backend/internal/errors"
	"backend/internal/utils"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"
)

// GetSmsClient 获取短信客户端
func GetSmsClient(AccessKeyId, AccessKeySecret string) (client *dysmsapi20170525.Client, err error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(AccessKeyId),
		AccessKeySecret: tea.String(AccessKeySecret),
	}
	config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
	client = &dysmsapi20170525.Client{}
	client, err = dysmsapi20170525.NewClient(config)
	return client, err
}

// SendSms 发送短信
func SendSms(client *dysmsapi20170525.Client, signName, templateCode, phoneNumbers, templateParam string) (*dysmsapi20170525.SendSmsResponseBody, error) {
	// 检查手机号码是否合法
	if b := utils.IsMobile(phoneNumbers); !b {
		return nil, error2.ErrMobileInvalid
	}
	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(phoneNumbers),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(templateParam),
	}
	// 复制代码运行请自行打印 API 的返回值
	result, err := client.SendSms(sendSmsRequest)
	if err != nil {
		zap.L().Error("发送短信失败", zap.String("error_type", fmt.Sprintf("%T", err)))
		return nil, error2.WrapSmsSendFailed(err)
	}
	zap.L().Debug("发送短信调用完成")
	return result.Body, nil
}

// CheckSmsStatus 检查短信发送状态
func CheckSmsStatus(client *dysmsapi20170525.Client, bizId, phoneNumber string, SendData string) (*dysmsapi20170525.QuerySendDetailsResponseBody, error) {
	querySendDetailsRequest := &dysmsapi20170525.QuerySendDetailsRequest{
		PhoneNumber: tea.String(phoneNumber),
		BizId:       tea.String(bizId),
		SendDate:    tea.String(SendData),
		PageSize:    tea.Int64(10),
		CurrentPage: tea.Int64(1),
	}
	// 复制代码运行请自行打印 API 的返回值
	result, err := client.QuerySendDetails(querySendDetailsRequest)
	if err != nil {
		zap.L().Error("查询短信发送状态失败", zap.String("error_type", fmt.Sprintf("%T", err)))
		return nil, error2.WrapSmsStatusQueryFailed(err)
	}
	zap.L().Debug("查询短信发送状态调用完成")
	return result.Body, nil
}
