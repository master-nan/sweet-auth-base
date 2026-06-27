package response

type ResetPasswordRes struct {
	TemporaryPassword  string `json:"temporary_password"`
	MustChangePassword bool   `json:"must_change_password"`
	EmailSent          bool   `json:"email_sent"`
	EmailMessage       string `json:"email_message,omitempty"`
}
