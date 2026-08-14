package response

import "backend/enum"

// SmsStatusRes exposes only the platform delivery state.
type SmsStatusRes struct {
	Status enum.SmsStatus `json:"status"`
}
