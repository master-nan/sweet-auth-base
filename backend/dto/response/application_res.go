package response

import "backend/model"

type ApplicationRes struct {
	BasicRes
	Name       string `json:"name"`
	AppKey     string `json:"app_key"`
	Expiration int64  `json:"expiration"`
	DingKey    string `json:"ding_key"`
	DingAppID  string `json:"ding_app_id"`
	Remark     string `json:"remark"`
}

type ApplicationSecretRes struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	AppKey     string `json:"app_key"`
	AppSecret  string `json:"app_secret"`
	Expiration int64  `json:"expiration"`
}

func NewApplicationSecretRes(application model.Application) ApplicationSecretRes {
	return ApplicationSecretRes{
		Id:         application.Id,
		Name:       application.Name,
		AppKey:     application.AppKey,
		AppSecret:  application.AppSecret,
		Expiration: application.Expiration,
	}
}
