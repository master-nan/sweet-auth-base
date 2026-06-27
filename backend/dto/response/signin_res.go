/**
 * @Author: Nan
 * @Date: 2023/3/14 16:55
 */

package response

type SignInRes struct {
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	MustChangePassword   bool   `json:"must_change_password"`
	PasswordChangeReason string `json:"password_change_reason,omitempty"`
}

type CaptchaRes struct {
	CaptchaId string `json:"captcha_id"`
	Image     []byte `json:"image"`
}
