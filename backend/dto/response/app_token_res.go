/**
 * @Author: Nan
 * @Date: 2024/10/24 14:39
 */

package response

type AppTokenRes struct {
	AppToken  string `json:"app_token"`
	ExpiresIn int64  `json:"expires_in"`
}
