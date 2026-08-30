package request

import "backend/model"

// UserSessionQueryReq 查询登录设备。Status 支持 online、active、closed 和 all。
type UserSessionQueryReq struct {
	Keyword        string            `json:"keyword"`
	Status         string            `json:"status"`
	LoginStartedAt *model.CustomTime `json:"login_started_at"`
	LoginEndedAt   *model.CustomTime `json:"login_ended_at"`
	Page           int               `json:"page"`
	Num            int               `json:"num"`
}

type UserSessionRevokeReq struct {
	Reason string `json:"reason" validate:"omitempty,max=160"`
}
