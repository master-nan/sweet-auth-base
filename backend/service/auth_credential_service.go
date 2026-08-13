package service

import (
	"backend/config"
	"backend/internal/cache"
	"backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	stderrors "errors"
	"strconv"
	"strings"

	"github.com/dchest/captcha"
	"gorm.io/gorm"
)

type PasswordCredentialProvider struct {
	users        repository.AuthenticationUserRepository
	serverConfig *config.Server
}

func NewPasswordCredentialProvider(users repository.AuthenticationUserRepository, serverConfig *config.Server) *PasswordCredentialProvider {
	return &PasswordCredentialProvider{users: users, serverConfig: serverConfig}
}

func (p *PasswordCredentialProvider) Type() AuthCredentialType { return AuthCredentialPassword }

func (p *PasswordCredentialProvider) Verify(ctx context.Context, req AuthenticationRequest) (CredentialVerification, error) {
	principal := strings.TrimSpace(req.Principal)
	result := CredentialVerification{Principal: principal}
	user, err := p.users.FindAuthenticationByPrincipal(ctx, principal)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) || stderrors.Is(err, repository.ErrAmbiguousAuthenticationPrincipal) {
			// Keep password work on the unknown-user path without retaining the secret.
			_ = utils.Encryption(req.Secret, "0"+p.serverConfig.Conf.Salt)
			return result, nil
		}
		return result, errors.WrapSystemError(err)
	}
	result.Identity = confirmedIdentity(user, AuthCredentialPassword, string(req.Channel))
	result.Verified = utils.Encryption(req.Secret, strconv.Itoa(user.Id)+p.serverConfig.Conf.Salt) == user.Password
	return result, nil
}

type SMSCredentialProvider struct {
	users repository.AuthenticationUserRepository
	codes *cache.SendCodeCache
}

func NewSMSCredentialProvider(users repository.AuthenticationUserRepository, codes *cache.SendCodeCache) *SMSCredentialProvider {
	return &SMSCredentialProvider{users: users, codes: codes}
}

func (p *SMSCredentialProvider) Type() AuthCredentialType { return AuthCredentialSMS }

func (p *SMSCredentialProvider) Verify(ctx context.Context, req AuthenticationRequest) (CredentialVerification, error) {
	principal := strings.TrimSpace(req.Principal)
	result := CredentialVerification{Principal: principal}
	consumed, err := p.codes.Consume(cache.SMSVerificationKey(req.Application.Id, principal), req.Secret)
	if err != nil {
		return result, errors.WrapSystemError(err)
	}
	if !consumed {
		return result, nil
	}
	user, err := p.users.FindAuthenticationByPhone(ctx, principal)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) || stderrors.Is(err, repository.ErrAmbiguousAuthenticationPrincipal) {
			return result, nil
		}
		return result, errors.WrapSystemError(err)
	}
	result.Verified = true
	result.Identity = confirmedIdentity(user, AuthCredentialSMS, string(req.Channel))
	return result, nil
}

type DingTalkCredentialProvider struct {
	dingTalk *DingTalkService
	users    repository.AuthenticationUserRepository
}

func NewDingTalkCredentialProvider(dingTalk *DingTalkService, users repository.AuthenticationUserRepository) *DingTalkCredentialProvider {
	return &DingTalkCredentialProvider{dingTalk: dingTalk, users: users}
}

func (p *DingTalkCredentialProvider) Type() AuthCredentialType { return AuthCredentialDingTalk }

func (p *DingTalkCredentialProvider) Verify(ctx context.Context, req AuthenticationRequest) (CredentialVerification, error) {
	result := CredentialVerification{Principal: "dingtalk"}
	if req.Application.DingKey == "" || req.Application.DingSecret == "" {
		return result, errors.ErrDingTalkSecretNotFound
	}
	accessToken, err := p.dingTalk.GetAccessToken(req.Application)
	if err != nil {
		return result, err
	}
	principal, err := p.dingTalk.GetIdentityPrincipal(accessToken, req.Secret)
	if err != nil {
		return result, err
	}
	result.Principal = principal
	user, err := p.users.FindAuthenticationByEmail(ctx, principal)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) || stderrors.Is(err, repository.ErrAmbiguousAuthenticationPrincipal) {
			return result, nil
		}
		return result, errors.WrapSystemError(err)
	}
	if user.Id == 0 {
		return result, nil
	}
	result.Verified = true
	result.Identity = confirmedIdentity(user, AuthCredentialDingTalk, string(req.Channel))
	return result, nil
}

type CaptchaVerifier struct{}

func NewCaptchaVerifier() *CaptchaVerifier { return &CaptchaVerifier{} }

func (*CaptchaVerifier) Verify(id, value string) bool { return captcha.VerifyString(id, value) }

func confirmedIdentity(user model.SysUser, method AuthCredentialType, source string) ConfirmedIdentity {
	return ConfirmedIdentity{
		UserID:               user.Id,
		Username:             user.UserName,
		AuthenticationMethod: method,
		CredentialSource:     source,
	}
}
