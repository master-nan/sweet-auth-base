/**
 * @Author: Nan
 * @Date: 2024/10/24 14:28
 */

package request

type AppTokenReq struct {
	AppKey    string `json:"app_key" binding:"required"`
	AppSecret string `json:"app_secret" binding:"required"`
}
