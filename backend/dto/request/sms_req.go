/**
 * @Author: Nan
 * @Date: 2025/3/1 15:57
 */

package request

import (
	"gorm.io/datatypes"
)

// SmsTemplateCreateReq 创建短信模板请求
type SmsTemplateCreateReq struct {
	SignName       string         `json:"sign_name" binding:"required" example:"签名"`
	TemplateCode   string         `json:"template_code" binding:"required" example:"模板编号"`
	TemplateName   string         `json:"template_name" binding:"required" example:"模板名称"`
	TemplateParams datatypes.JSON `json:"template_params" binding:"required,non_empty_json" example:"模板参数"`
}

// SmsTemplateUpdateReq 更新短信模板请求
type SmsTemplateUpdateReq struct {
	Id             int            `json:"id"  example:"1"`
	SignName       string         `json:"sign_name"  binding:"required" example:"签名"`
	TemplateCode   string         `json:"template_code"  binding:"required" example:"模板编号"`
	TemplateName   string         `json:"template_name"  binding:"required" example:"模板名称"`
	TemplateParams datatypes.JSON `json:"template_params"  binding:"required,non_empty_json" example:"模板参数"`
}
