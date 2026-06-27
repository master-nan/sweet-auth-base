package request

type AccessLogQueryReq struct {
	Basic
	UserName     string `json:"user_name"`
	Action       string `json:"action"`
	ResourceCode string `json:"resource_code"`
	Method       string `json:"method"`
	Url          string `json:"url"`
	Ip           string `json:"ip"`
	Success      *bool  `json:"success"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}
