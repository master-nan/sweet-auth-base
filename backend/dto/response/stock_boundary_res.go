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

// RuntimeDictRes is the read-only dictionary shape used by authenticated pages.
// Administration audit fields and internal identifiers are intentionally absent.
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

// AccessLogRes deliberately excludes request/response payloads and internal
// relation IDs. Audit pages only receive the stable, non-payload summary.
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
