package service

import (
	"backend/model"
	"context"
	"time"
)

type AuthCredentialType string
type AuthChannel string

const (
	AuthCredentialPassword AuthCredentialType = "password"
	AuthCredentialSMS      AuthCredentialType = "sms"
	AuthCredentialDingTalk AuthCredentialType = "dingtalk_sso"
	AuthCredentialRefresh  AuthCredentialType = "refresh_token"
	AuthCredentialAccess   AuthCredentialType = "access_token"

	AuthChannelAdminPassword AuthChannel = "admin_password"
	AuthChannelAPIPassword   AuthChannel = "api_password"
	AuthChannelSMS           AuthChannel = "sms"
	AuthChannelDingTalk      AuthChannel = "dingtalk_sso"
	AuthChannelRefresh       AuthChannel = "refresh"
	AuthChannelLogout        AuthChannel = "logout"
)

type AuthenticationRequest struct {
	Channel        AuthChannel
	CredentialType AuthCredentialType
	Principal      string
	Secret         string
	CaptchaID      string
	Captcha        string
	Application    model.Application
}

type ConfirmedIdentity struct {
	UserID               int
	Username             string
	AuthenticationMethod AuthCredentialType
	CredentialSource     string
}

type CredentialVerification struct {
	Verified  bool
	Identity  ConfirmedIdentity
	Principal string
}

type AuthCredentialProvider interface {
	Type() AuthCredentialType
	Verify(context.Context, AuthenticationRequest) (CredentialVerification, error)
}

type AuthenticationResult struct {
	AccessToken          string
	RefreshToken         string
	MustChangePassword   bool
	PasswordChangeReason string
}

type AuthenticatedAccess struct {
	User                 model.SysUser
	Issued               time.Time
	MustChangePassword   bool
	PasswordChangeReason string
}

type AuthAuditEvent struct {
	Channel        AuthChannel
	CredentialType AuthCredentialType
	Success        bool
	ReasonCode     string
	UserID         int
	Principal      string
	HTTPStatus     int
}

type AuthAuditRecorder interface {
	RecordAuthEvent(context.Context, AuthAuditEvent) error
}
