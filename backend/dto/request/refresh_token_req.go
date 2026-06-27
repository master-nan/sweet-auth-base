/**
 * @Author: Nan
 * @Date: 2024/10/23 17:45
 */

package request

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
