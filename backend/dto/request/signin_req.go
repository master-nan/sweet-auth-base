/**
 * @Author: Nan
 * @Date: 2023/3/14 21:10
 */

package request

type SignInReq struct {
	UserName  string `json:"user_name" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Captcha   string `json:"captcha"`
	CaptchaId string `json:"captcha_id"`
}

type SmsCodeLoginReq struct {
	Mobile string `json:"mobile" binding:"required"`
	Code   string `json:"code" binding:"required"`
}
