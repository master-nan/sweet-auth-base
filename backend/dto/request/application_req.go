/**
 * @Author: Nan
 * @Date: 2024/10/24 10:49
 */

package request

type ApplicationCreateReq struct {
	Name       string `json:"name" binding:"required"`
	DingKey    string `json:"ding_key"`
	DingSecret string `json:"ding_secret"`
	DingAppID  string `json:"ding_app_id"`
	Remark     string `json:"remark"`
	Expiration int64  `json:"expiration" binding:"required"`
}

type ApplicationUpdateReq struct {
	Id         int    `json:"id"`
	Name       string `json:"name" binding:"required"`
	Expiration int64  `json:"expiration" binding:"required"`
	DingAppID  string `json:"ding_app_id"`
	DingKey    string `json:"ding_key"`
	DingSecret string `json:"ding_secret"`
	Remark     string `json:"remark"`
}
