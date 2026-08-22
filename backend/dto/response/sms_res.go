package response

import "backend/enum"

// SmsStatusRes 只暴露平台可识别的短信投递状态。
type SmsStatusRes struct {
	Status enum.SmsStatus `json:"status"`
}
