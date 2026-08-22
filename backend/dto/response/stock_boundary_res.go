package response

import "encoding/json"

type SysDictRes struct {
	BasicRes
	DictName  string           `json:"dict_name"`
	DictCode  string           `json:"dict_code"`
	DictItems []SysDictItemRes `json:"dict_items,omitempty"`
}

type SysDictItemRes struct {
	BasicRes
	DictId    int    `json:"dict_id"`
	ItemName  string `json:"item_name"`
	ItemCode  string `json:"item_code"`
	ItemValue string `json:"item_value"`
}

// RuntimeDictRes 是已认证页面使用的只读字典结构，不包含管理审计字段和内部标识。
type RuntimeDictRes struct {
	DictName  string               `json:"dict_name"`
	DictCode  string               `json:"dict_code"`
	DictItems []RuntimeDictItemRes `json:"dict_items"`
}

type RuntimeDictItemRes struct {
	ItemName  string `json:"item_name"`
	ItemCode  string `json:"item_code"`
	ItemValue string `json:"item_value"`
}

type SmsTemplateRes struct {
	BasicRes
	SignName       string          `json:"sign_name"`
	TemplateCode   string          `json:"template_code"`
	TemplateName   string          `json:"template_name"`
	TemplateParams json.RawMessage `json:"template_params"`
}

// AccessLogRes 排除请求/响应Payload及内部关系ID，审计页面只接收稳定的非Payload摘要。
type AccessLogRes struct {
	BasicRes
	UserName     string `json:"user_name"`
	RequestId    string `json:"request_id"`
	TraceId      string `json:"trace_id"`
	Method       string `json:"method"`
	Ip           string `json:"ip"`
	Locality     string `json:"locality"`
	Url          string `json:"url"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceCode string `json:"resource_code"`
	ResourceId   string `json:"resource_id"`
	StatusCode   int    `json:"status_code"`
	Success      bool   `json:"success"`
	Result       string `json:"result"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	DurationMs   int64  `json:"duration_ms"`
}
