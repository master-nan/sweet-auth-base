package request

// UserSessionQueryReq 查询登录设备。Status 支持 online、active、closed 和 all。
type UserSessionQueryReq struct {
	Keyword string `json:"keyword"`
	Status  string `json:"status"`
	Page    int    `json:"page"`
	Num     int    `json:"num"`
}
