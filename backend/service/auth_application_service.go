package service

import (
	"backend/internal/cache"
	"backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"crypto/sha256"
	stderrors "errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const passwordChangedAtFutureTolerance = 5 * time.Minute

type AuthApplicationService struct {
	users      repository.AuthenticationUserRepository
	configure  *SysConfigureService
	attempts   *cache.LoginAttemptCache
	tokens     *AuthTokenService
	loginState *AuthLoginStateService
	audit      AuthAuditRecorder
	captcha    *CaptchaVerifier
	providers  map[AuthCredentialType]AuthCredentialProvider
	now        func() time.Time
}

func NewAuthApplicationService(
	users repository.AuthenticationUserRepository,
	configure *SysConfigureService,
	attempts *cache.LoginAttemptCache,
	tokens *AuthTokenService,
	loginState *AuthLoginStateService,
	audit AuthAuditRecorder,
	captchaVerifier *CaptchaVerifier,
	password *PasswordCredentialProvider,
	sms *SMSCredentialProvider,
	dingTalk *DingTalkCredentialProvider,
) *AuthApplicationService {
	providers := map[AuthCredentialType]AuthCredentialProvider{}
	for _, provider := range []AuthCredentialProvider{password, sms, dingTalk} {
		providers[provider.Type()] = provider
	}
	return &AuthApplicationService{
		users: users, configure: configure, attempts: attempts, tokens: tokens,
		loginState: loginState, audit: audit, captcha: captchaVerifier,
		providers: providers, now: time.Now,
	}
}

func (s *AuthApplicationService) Authenticate(ctx context.Context, req AuthenticationRequest) (AuthenticationResult, error) {
	cfg, err := s.configure.Query()
	if err != nil {
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	if req.Channel == AuthChannelAdminPassword && cfg.EnableCaptcha && !s.captcha.Verify(req.CaptchaID, req.Captcha) {
		if auditErr := s.record(ctx, req, CredentialVerification{Principal: req.Principal}, false, "captcha_invalid"); auditErr != nil {
			return AuthenticationResult{}, errors.WrapSystemError(auditErr)
		}
		return AuthenticationResult{}, errors.ErrCaptchaInvalid
	}
	provider, ok := s.providers[req.CredentialType]
	if !ok {
		return AuthenticationResult{}, errors.ErrAuthenticationFailed
	}
	verification, err := provider.Verify(ctx, req)
	if err != nil {
		_ = s.recordStatus(ctx, req, verification, false, "credential_dependency_failed", authAuditHTTPStatus(err))
		return AuthenticationResult{}, err
	}
	principalKey := authenticationAttemptKey(verification)
	locked, err := s.attempts.IsLocked(principalKey)
	if err != nil {
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	if !locked && verification.Identity.UserID > 0 && strings.TrimSpace(verification.Principal) != "" {
		locked, err = s.attempts.IsLocked(verification.Principal)
		if err != nil {
			return AuthenticationResult{}, errors.WrapSystemError(err)
		}
	}
	if locked {
		if auditErr := s.record(ctx, req, verification, false, "account_locked"); auditErr != nil {
			return AuthenticationResult{}, errors.WrapSystemError(auditErr)
		}
		return AuthenticationResult{}, errors.ErrAuthenticationFailed
	}
	if !verification.Verified || verification.Identity.UserID <= 0 {
		if req.CredentialType == AuthCredentialPassword {
			locked, err = s.attempts.RecordFailure(
				principalKey,
				cfg.PasswordErrorCount,
				time.Duration(cfg.PasswordLockMinutes)*time.Minute,
			)
			if err != nil {
				return AuthenticationResult{}, errors.WrapSystemError(err)
			}
		}
		reason := "credential_invalid"
		if locked {
			reason = "account_locked"
		}
		if auditErr := s.record(ctx, req, verification, false, reason); auditErr != nil {
			return AuthenticationResult{}, errors.WrapSystemError(auditErr)
		}
		if locked {
			return AuthenticationResult{}, errors.ErrAuthenticationFailed
		}
		return AuthenticationResult{}, errors.ErrAuthenticationFailed
	}
	user, err := s.users.FindAuthenticationByID(ctx, verification.Identity.UserID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			_ = s.record(ctx, req, verification, false, "account_not_runnable")
			return AuthenticationResult{}, errors.ErrAuthenticationFailed
		}
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	if !user.State || user.Id == 0 {
		if auditErr := s.record(ctx, req, verification, false, "account_not_runnable"); auditErr != nil {
			return AuthenticationResult{}, errors.WrapSystemError(auditErr)
		}
		return AuthenticationResult{}, errors.ErrAuthenticationFailed
	}
	issuedAt := s.now().UTC()
	mustChange, reason := PasswordChangeRequirement(user, cfg, issuedAt)
	if req.CredentialType != AuthCredentialPassword && mustChange {
		if auditErr := s.record(ctx, req, verification, false, "password_change_required"); auditErr != nil {
			return AuthenticationResult{}, errors.WrapSystemError(auditErr)
		}
		return AuthenticationResult{}, errors.ErrAuthenticationFailed
	}
	if req.CredentialType == AuthCredentialPassword {
		allowed, err := s.attempts.CompleteSuccess(principalKey)
		if err != nil {
			return AuthenticationResult{}, errors.WrapSystemError(err)
		}
		if !allowed {
			if auditErr := s.record(ctx, req, verification, false, "account_locked"); auditErr != nil {
				return AuthenticationResult{}, errors.WrapSystemError(auditErr)
			}
			return AuthenticationResult{}, errors.ErrAuthenticationFailed
		}
	}
	pair, err := s.tokens.Issue(ctx, user.Id, issuedAt)
	if err != nil {
		_ = s.recordStatus(ctx, req, verification, false, "token_issue_failed", authAuditHTTPStatus(err))
		return AuthenticationResult{}, err
	}
	if err := s.loginState.RecordLogin(ctx, user.Id, pair.AccessToken, issuedAt); err != nil {
		s.tokens.RevokePair(pair)
		_ = s.recordStatus(ctx, req, verification, false, "login_state_failed", authAuditHTTPStatus(err))
		return AuthenticationResult{}, err
	}
	if _, err := s.tokens.ValidateAccess(ctx, pair.AccessToken); err != nil {
		s.tokens.RevokePair(pair)
		_ = s.loginState.RollbackLogin(ctx, user.Id, pair.AccessToken)
		_ = s.record(ctx, req, verification, false, "authentication_conflict")
		return AuthenticationResult{}, errors.ErrAuthenticationFailed
	}
	verification.Identity.Username = user.UserName
	if err := s.record(ctx, req, verification, true, "authentication_succeeded"); err != nil {
		s.tokens.RevokePair(pair)
		_ = s.loginState.RollbackLogin(ctx, user.Id, pair.AccessToken)
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	return AuthenticationResult{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken,
		MustChangePassword: mustChange, PasswordChangeReason: reason,
	}, nil
}

func (s *AuthApplicationService) Refresh(ctx context.Context, refreshToken string) (AuthenticationResult, error) {
	startedAt := s.now().UTC()
	claims, err := s.tokens.ValidateRefresh(ctx, refreshToken)
	if err != nil {
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, ReasonCode: "refresh_invalid", HTTPStatus: authAuditHTTPStatus(err)})
		return AuthenticationResult{}, err
	}
	userID, _ := strconv.Atoi(claims.ID)
	user, err := s.users.FindAuthenticationByID(ctx, userID)
	if err != nil || user.Id == 0 || !user.State {
		if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return AuthenticationResult{}, errors.WrapSystemError(err)
		}
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, UserID: userID, ReasonCode: "account_not_runnable"})
		return AuthenticationResult{}, errors.ErrInvalidRefreshToken
	}
	locked, err := s.attempts.IsLocked(cache.UserLoginPrincipal(userID))
	if err != nil {
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	if locked || TokenIssuedBeforePasswordChangeAt(claims.IssuedAt, user, startedAt) {
		reason := "account_locked"
		if !locked {
			reason = "password_changed"
		}
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, UserID: userID, ReasonCode: reason})
		return AuthenticationResult{}, errors.ErrInvalidRefreshToken
	}
	cfg, err := s.configure.Query()
	if err != nil {
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	if mustChange, _ := PasswordChangeRequirement(user, cfg, startedAt); mustChange {
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{
			Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh,
			UserID: userID, ReasonCode: "password_change_required",
		})
		return AuthenticationResult{}, errors.ErrPasswordChangeRequired
	}
	consumed, err := s.tokens.ConsumeRefresh(refreshToken, claims.ExpiresAt)
	if err != nil {
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, UserID: userID, ReasonCode: "refresh_dependency_failed", HTTPStatus: authAuditHTTPStatus(err)})
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	if !consumed {
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, UserID: userID, ReasonCode: "refresh_reused"})
		return AuthenticationResult{}, errors.ErrInvalidRefreshToken
	}
	issuedAt := startedAt
	pair, err := s.tokens.IssueRefresh(ctx, userID, issuedAt, claims.SessionID)
	if err != nil {
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, UserID: userID, ReasonCode: "token_issue_failed", HTTPStatus: authAuditHTTPStatus(err)})
		return AuthenticationResult{}, err
	}
	if err := s.loginState.RecordLogin(ctx, userID, pair.AccessToken, startedAt); err != nil {
		s.tokens.RevokePair(pair)
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, UserID: userID, ReasonCode: "login_state_failed", HTTPStatus: authAuditHTTPStatus(err)})
		return AuthenticationResult{}, err
	}
	// Close the logout/refresh race after the local login-state write. A logout
	// deactivates the session shared by the old and newly issued token pair.
	if _, err := s.tokens.ValidateAccess(ctx, pair.AccessToken); err != nil {
		s.tokens.RevokePair(pair)
		_ = s.loginState.RollbackLogin(ctx, userID, pair.AccessToken)
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, UserID: userID, ReasonCode: "refresh_conflict"})
		return AuthenticationResult{}, errors.ErrInvalidRefreshToken
	}
	if err := s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh, Success: true, UserID: userID, ReasonCode: "refresh_succeeded"}); err != nil {
		s.tokens.RevokePair(pair)
		_ = s.loginState.RollbackLogin(ctx, userID, pair.AccessToken)
		return AuthenticationResult{}, errors.WrapSystemError(err)
	}
	return AuthenticationResult{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken,
	}, nil
}

func (s *AuthApplicationService) Logout(ctx context.Context, accessToken string) error {
	userID, err := s.tokens.RevokeAccessAndSession(accessToken)
	if err != nil {
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelLogout, CredentialType: AuthCredentialAccess, ReasonCode: "logout_token_invalid", HTTPStatus: authAuditHTTPStatus(err)})
		return err
	}
	if err := s.loginState.Logout(ctx, userID, accessToken); err != nil {
		_ = s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelLogout, CredentialType: AuthCredentialAccess, UserID: userID, ReasonCode: "login_state_failed", HTTPStatus: authAuditHTTPStatus(err)})
		return err
	}
	return s.audit.RecordAuthEvent(ctx, AuthAuditEvent{Channel: AuthChannelLogout, CredentialType: AuthCredentialAccess, Success: true, UserID: userID, ReasonCode: "logout_succeeded"})
}

func (s *AuthApplicationService) AuthenticateAccessToken(ctx context.Context, accessToken string) (AuthenticatedAccess, error) {
	claims, err := s.tokens.ValidateAccess(ctx, accessToken)
	if err != nil {
		return AuthenticatedAccess{}, err
	}
	userID, _ := strconv.Atoi(claims.ID)
	user, err := s.users.FindAuthenticationByID(ctx, userID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return AuthenticatedAccess{}, errors.ErrUserNotLogin
		}
		return AuthenticatedAccess{}, errors.WrapSystemError(err)
	}
	if user.Id == 0 || !user.State {
		return AuthenticatedAccess{}, errors.ErrUserNotLogin
	}
	locked, err := s.attempts.IsLocked(cache.UserLoginPrincipal(userID))
	if err != nil {
		return AuthenticatedAccess{}, errors.WrapSystemError(err)
	}
	if locked {
		return AuthenticatedAccess{}, errors.ErrLoginLocked
	}
	if TokenIssuedBeforePasswordChangeAt(claims.IssuedAt, user, s.now().UTC()) {
		return AuthenticatedAccess{}, errors.ErrTokenExpired
	}
	cfg, err := s.configure.Query()
	if err != nil {
		return AuthenticatedAccess{}, errors.WrapSystemError(err)
	}
	mustChange, reason := PasswordChangeRequirement(user, cfg, s.now().UTC())
	return AuthenticatedAccess{
		User: user, Issued: claims.IssuedAt,
		MustChangePassword: mustChange, PasswordChangeReason: reason,
	}, nil
}

func (s *AuthApplicationService) record(ctx context.Context, req AuthenticationRequest, verification CredentialVerification, success bool, reason string) error {
	return s.recordStatus(ctx, req, verification, success, reason, 0)
}

func (s *AuthApplicationService) recordStatus(ctx context.Context, req AuthenticationRequest, verification CredentialVerification, success bool, reason string, status int) error {
	return s.audit.RecordAuthEvent(ctx, AuthAuditEvent{
		Channel: req.Channel, CredentialType: req.CredentialType, Success: success,
		ReasonCode: reason, UserID: verification.Identity.UserID, Principal: verification.Principal, HTTPStatus: status,
	})
}

func authAuditHTTPStatus(err error) int {
	switch errors.KindOf(err) {
	case errors.KindInvalidArgument:
		return http.StatusBadRequest
	case errors.KindUnauthenticated:
		return http.StatusUnauthorized
	case errors.KindForbidden:
		return http.StatusForbidden
	case errors.KindRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func authenticationAttemptKey(verification CredentialVerification) string {
	if verification.Identity.UserID > 0 {
		return cache.UserLoginPrincipal(verification.Identity.UserID)
	}
	normalized := strings.ToLower(strings.TrimSpace(verification.Principal))
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("principal:sha256:%x", sum[:16])
}

func TokenIssuedBeforePasswordChangeAt(issuedAt time.Time, user model.SysUser, now time.Time) bool {
	if user.PasswordChangedAt == nil || time.Time(*user.PasswordChangedAt).IsZero() {
		return false
	}
	changedAt := time.Time(*user.PasswordChangedAt)
	if changedAt.After(now.Add(passwordChangedAtFutureTolerance)) {
		return false
	}
	return issuedAt.UTC().Truncate(time.Second).Before(changedAt.UTC().Truncate(time.Second))
}
