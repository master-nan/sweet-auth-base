/**
 * @Author: Nan
 * @Date: 2024/10/23 21:47
 */

package model

type Application struct {
	Basic
	Name       string `json:"name" gorm:"type:varchar(255);not null;comment:应用名称"`
	AppKey     string `json:"app_key" gorm:"type:varchar(255);not null;comment:应用key"`
	AppSecret  string `json:"app_secret" gorm:"type:varchar(255);not null;comment:应用secret"`
	Expiration int64  `json:"expiration" gorm:"type:int;not null;comment:过期时间"`
	DingKey    string `json:"ding_key" gorm:"type:varchar(255);comment:钉钉key"`
	DingSecret string `json:"ding_secret" gorm:"type:varchar(255);comment:钉钉secret"`
	DingAppID  string `json:"ding_app_id" gorm:"column:ding_app_id;type:varchar(255);comment:钉钉appID"`
	Remark     string `json:"remark" gorm:"type:varchar(255);comment:备注"`
}
