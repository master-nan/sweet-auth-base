/**
 * @Author: Nan
 * @Date: 2025/2/7 21:01
 */

package model

import "backend/enum"
import "gorm.io/datatypes"

// SmsTemplate 短信模版
type SmsTemplate struct {
	Basic
	// 短信签名
	SignName string `json:"sign_name" gorm:"type:varchar(50);comment:短信签名"`
	// 模版编号
	TemplateCode string `json:"template_code" gorm:"type:varchar(50);comment:模版编号"`
	// 模版名称
	TemplateName string `json:"template_name" gorm:"type:varchar(255);comment:模版名称"`
	// 模版参数，以 JSON 数组形式存储，如 ["code"] 或 ["name","userno","password"]
	TemplateParams datatypes.JSON `json:"template_params" gorm:"type:json;comment:模板参数列表"`
}

// SmsLog 短信发送记录
type SmsLog struct {
	Basic
	// 模版编号
	TemplateCode string `json:"template_code" gorm:"type:varchar(50);comment:模版编号"`
	// 短信签名
	SignName string `json:"sign_name" gorm:"type:varchar(50);comment:短信签名"`
	// 手机号
	Mobile string `json:"mobile" gorm:"type:varchar(20);comment:手机号"`
	// 短信内容JSON
	Content string `json:"content" gorm:"type:json;comment:短信内容JSON"`
	// 发送状态 1:发送中, 2:发送成功, 3:发送失败
	Status enum.SmsStatus `json:"status" gorm:"type:tinyint;comment:发送状态（1:发送中, 2:发送成功, 3:发送失败）"`
	// 发送回执ID
	BizId string `json:"biz_id" gorm:"type:varchar(50);comment:发送回执ID"`
	// 发送结果
	Result string `json:"result" gorm:"type:text;comment:发送结果"`
	// 应用ID
	ApplicationId int `json:"application_id" gorm:"type:int;comment:应用ID"`
	// 应用名称
	ApplicationName string `json:"application_name" gorm:"type:varchar(255);comment:应用名称"`
}
