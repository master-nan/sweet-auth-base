package service

import (
	"backend/config"
	"backend/enum"
	"backend/internal/audit"
	"backend/internal/cache"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/token"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type authMemoryCache struct {
	mu     sync.Mutex
	values map[string]any
}

func newAuthMemoryCache() *authMemoryCache { return &authMemoryCache{values: map[string]any{}} }

func (m *authMemoryCache) Get(key string, target any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[key]
	if !ok {
		return cache.ErrCacheMiss
	}
	switch typed := target.(type) {
	case *string:
		*typed = toAuthCacheString(value)
	case *int:
		value, _ := strconv.Atoi(toAuthCacheString(value))
		*typed = value
	case *model.SysConfigure:
		*typed = value.(model.SysConfigure)
	}
	return nil
}

func (m *authMemoryCache) Set(key string, value any, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ptr, ok := value.(*model.SysConfigure); ok {
		m.values[key] = *ptr
	} else {
		m.values[key] = value
	}
	return nil
}
func (m *authMemoryCache) Del(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
	return nil
}
func (m *authMemoryCache) Exists(keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, key := range keys {
		if _, ok := m.values[key]; ok {
			count++
		}
	}
	return count, nil
}
func (m *authMemoryCache) Expire(string, time.Duration) (bool, error) { return true, nil }
func (m *authMemoryCache) SetIfAbsent(key string, value any, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.values[key]; exists {
		return false, nil
	}
	m.values[key] = value
	return true, nil
}
func (m *authMemoryCache) ConsumeCode(key, attemptKey, expected string, maxAttempts int, _ time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	actual, exists := m.values[key]
	if !exists {
		return 0, nil
	}
	if toAuthCacheString(actual) == expected {
		delete(m.values, key)
		delete(m.values, attemptKey)
		return 1, nil
	}
	attempts, _ := strconv.Atoi(toAuthCacheString(m.values[attemptKey]))
	attempts++
	if attempts >= maxAttempts {
		delete(m.values, key)
		delete(m.values, attemptKey)
	} else {
		m.values[attemptKey] = attempts
	}
	return -1, nil
}
func (m *authMemoryCache) RecordLoginFailure(attemptKey, lockKey string, max int, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count, _ := strconv.Atoi(toAuthCacheString(m.values[attemptKey]))
	count++
	if count >= max {
		delete(m.values, attemptKey)
		m.values[lockKey] = 1
		return true, nil
	}
	m.values[attemptKey] = count
	return false, nil
}
func (m *authMemoryCache) CompleteLoginSuccess(attemptKey, lockKey string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, locked := m.values[lockKey]; locked {
		return false, nil
	}
	delete(m.values, attemptKey)
	return true, nil
}
func toAuthCacheString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case *string:
		return *typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	}
	return ""
}

type authAuditSpy struct {
	mu     sync.Mutex
	events []AuthAuditEvent
	ctxs   []context.Context
}

type authStaticCredentialProvider struct {
	credentialType AuthCredentialType
	verification   CredentialVerification
}

func (p authStaticCredentialProvider) Type() AuthCredentialType { return p.credentialType }

func (p authStaticCredentialProvider) Verify(context.Context, AuthenticationRequest) (CredentialVerification, error) {
	return p.verification, nil
}

func (s *authAuditSpy) RecordAuthEvent(ctx context.Context, event AuthAuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	s.ctxs = append(s.ctxs, ctx)
	return nil
}

type authTestSubject struct {
	service *AuthApplicationService
	tokens  *AuthTokenService
	cache   *authMemoryCache
	dbUsers *impl.SysUserRepositoryImpl
	audit   *authAuditSpy
}

func newAuthTestSubject(t *testing.T, user model.SysUser) authTestSubject {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.SysUser{}, &model.SysRole{}, &model.SysUserRole{}, &model.SysConfigure{})
	server := &config.Server{Name: "sweet_admin"}
	server.Conf.Salt = "test-auth-salt"
	isReset := user.IsReset
	user.Password = utils.Encryption(user.Password, strconv.Itoa(user.Id)+server.Conf.Salt)
	if user.Basic.Id == 0 {
		user.Basic.Id = 1
	}
	testutil.MustCreate(t, db, &user)
	if err := db.Model(&model.SysUser{}).Where("id = ?", user.Id).Update("is_reset", isReset).Error; err != nil {
		t.Fatal(err)
	}
	testutil.MustCreate(t, db, &model.SysConfigure{Basic: model.Basic{Id: 1, State: true}, PasswordErrorCount: 2, PasswordLockMinutes: 10, PasswordExpireTime: 90})
	primary := &database.PrimaryDB{DB: db}
	users := impl.NewSysUserRepositoryImpl(primary)
	configRepo := impl.NewSysConfigureRepositoryImpl(primary)
	store := newAuthMemoryCache()
	configure := NewSysConfigureService(configRepo, cache.NewSysConfigureCache(store))
	attempts := cache.NewLoginAttemptCache(store)
	tokenState := cache.NewTokenBlackCache(store)
	tokens := NewAuthTokenService(token.JWTToken{Generator: token.NewJWTGenerator()}, tokenState, server)
	loginState := NewAuthLoginStateService(users)
	auditSpy := &authAuditSpy{}
	password := NewPasswordCredentialProvider(users, server)
	sms := NewSMSCredentialProvider(users, cache.NewSendCodeCache(store))
	service := &AuthApplicationService{
		users: users, configure: configure, attempts: attempts, tokens: tokens,
		loginState: loginState, audit: auditSpy, captcha: NewCaptchaVerifier(),
		providers: map[AuthCredentialType]AuthCredentialProvider{AuthCredentialPassword: password, AuthCredentialSMS: sms},
		now:       time.Now,
	}
	return authTestSubject{service: service, tokens: tokens, cache: store, dbUsers: users, audit: auditSpy}
}

func TestAuthPasswordSuccessAndAPIPasswordSharePolicy(t *testing.T) {
	for _, channel := range []AuthChannel{AuthChannelAdminPassword, AuthChannelAPIPassword} {
		t.Run(string(channel), func(t *testing.T) {
			subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct", IsReset: true})
			ctx := audit.WithCorrelationIDs(context.Background(), audit.CorrelationIDs{RequestID: "req-auth", TraceID: "trace-auth"})
			result, err := subject.service.Authenticate(ctx, AuthenticationRequest{Channel: channel, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
			if err != nil || result.AccessToken == "" || result.RefreshToken == "" || !result.MustChangePassword {
				t.Fatalf("authenticate: result=%+v err=%v", result, err)
			}
			access, err := subject.service.AuthenticateAccessToken(context.Background(), result.AccessToken)
			if err != nil || access.User.Id != 1 || !access.MustChangePassword || access.PasswordChangeReason != "initial_reset" {
				t.Fatalf("validate access: %+v %v", access, err)
			}
			if got := audit.GetCorrelationIDs(subject.audit.ctxs[0]); got.RequestID != "req-auth" || got.TraceID != "trace-auth" {
				t.Fatalf("audit context lost: %+v", got)
			}
		})
	}
}

func TestAuthLoginStateRemovesLegacyRawAccessTokens(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{
		Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct",
		AccessTokens: "legacy.raw.jwt,sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if _, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAdminPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"}); err != nil {
		t.Fatal(err)
	}
	user, err := subject.dbUsers.FindAuthenticationByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(user.AccessTokens, "legacy.raw.jwt") || strings.Count(user.AccessTokens, "sha256:") != 2 {
		t.Fatalf("login state retained a raw access token: %q", user.AccessTokens)
	}
}

func TestAuthPasswordFailureLocksCanonicalAccount(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Email: "admin@example.test", Password: "correct"})
	for _, principal := range []string{"admin", "admin@example.test"} {
		_, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: principal, Secret: "wrong"})
		if err == nil {
			t.Fatal("expected authentication failure")
		}
	}
	locked, err := subject.cache.Exists(cache.LoginLockCacheKey + cache.UserLoginPrincipal(1))
	if err != nil || locked != 1 {
		t.Fatalf("expected canonical lock, count=%d err=%v", locked, err)
	}
}

func TestAuthUnknownDisabledAndLockedDoNotIssueTokens(t *testing.T) {
	for name, setup := range map[string]func(authTestSubject){
		"disabled": func(subject authTestSubject) {
			subject.dbUsers.UpdateAuthenticationState(subject.dbUsers.DBWithContext(context.Background()), 1, map[string]any{"state": false})
		},
		"locked": func(subject authTestSubject) {
			subject.cache.Set(cache.LoginLockCacheKey+cache.UserLoginPrincipal(1), 1, time.Minute)
		},
	} {
		t.Run(name, func(t *testing.T) {
			subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
			setup(subject)
			result, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
			if !errors.Is(err, myerrors.ErrAuthenticationFailed) || result.AccessToken != "" {
				t.Fatalf("expected fail closed, result=%+v err=%v", result, err)
			}
		})
	}
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	_, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "missing", Secret: "correct"})
	if !errors.Is(err, myerrors.ErrAuthenticationFailed) {
		t.Fatalf("expected stable failure, got %v", err)
	}
}

func TestAuthSMSConsumesCodeAndChecksAccountState(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", PhoneNumber: "13800138000", Password: "unused"})
	key := cache.SMSVerificationKey(7, "13800138000")
	if err := cache.NewSendCodeCache(subject.cache).Set(key, "123456"); err != nil {
		t.Fatal(err)
	}
	req := AuthenticationRequest{Channel: AuthChannelSMS, CredentialType: AuthCredentialSMS, Principal: "13800138000", Secret: "123456", Application: model.Application{Basic: model.Basic{Id: 7, State: true}}}
	if result, err := subject.service.Authenticate(context.Background(), req); err != nil || result.AccessToken == "" {
		t.Fatalf("sms login: %+v %v", result, err)
	}
	if _, err := subject.service.Authenticate(context.Background(), req); !errors.Is(err, myerrors.ErrAuthenticationFailed) {
		t.Fatalf("expected one-time code rejection, got %v", err)
	}
}

func TestAuthSMSFailedCodesExhaustChallengeWithoutPasswordLock(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", PhoneNumber: "13800138000", Password: "unused"})
	key := cache.SMSVerificationKey(7, "13800138000")
	if err := cache.NewSendCodeCache(subject.cache).Set(key, "123456"); err != nil {
		t.Fatal(err)
	}
	req := AuthenticationRequest{Channel: AuthChannelSMS, CredentialType: AuthCredentialSMS, Principal: "13800138000", Application: model.Application{Basic: model.Basic{Id: 7, State: true}}}
	for i := 0; i < 5; i++ {
		req.Secret = "000000"
		if _, err := subject.service.Authenticate(context.Background(), req); !errors.Is(err, myerrors.ErrAuthenticationFailed) {
			t.Fatalf("wrong code attempt %d: %v", i+1, err)
		}
	}
	req.Secret = "123456"
	if _, err := subject.service.Authenticate(context.Background(), req); !errors.Is(err, myerrors.ErrAuthenticationFailed) {
		t.Fatalf("exhausted challenge accepted the original code: %v", err)
	}
	locked, err := subject.cache.Exists(cache.LoginLockCacheKey + cache.UserLoginPrincipal(1))
	if err != nil || locked != 0 {
		t.Fatalf("SMS failures must not increment password lock state: locked=%d err=%v", locked, err)
	}
}

func TestAuthChannelSpecificIdentityMappingDoesNotCrossPrincipalFields(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{
		Basic: model.Basic{Id: 1, State: true}, UserName: "employee", PhoneNumber: "13800138000", Email: "employee@example.test", Password: "unused",
	})
	conflicting := model.SysUser{Basic: model.Basic{Id: 2, State: true}, UserName: "13800138000", Email: "other@example.test", Password: "unused"}
	if err := subject.dbUsers.DBWithContext(context.Background()).Create(&conflicting).Error; err != nil {
		t.Fatal(err)
	}
	key := cache.SMSVerificationKey(7, "13800138000")
	if err := cache.NewSendCodeCache(subject.cache).Set(key, "123456"); err != nil {
		t.Fatal(err)
	}
	result, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{
		Channel: AuthChannelSMS, CredentialType: AuthCredentialSMS, Principal: "13800138000", Secret: "123456",
		Application: model.Application{Basic: model.Basic{Id: 7, State: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := subject.tokens.ValidateAccess(context.Background(), result.AccessToken)
	if err != nil || claims.ID != "1" {
		t.Fatalf("SMS principal mapped through a non-phone field: claims=%+v err=%v", claims, err)
	}

	conflicting.UserName = "employee@example.test"
	if err := subject.dbUsers.DBWithContext(context.Background()).Model(&model.SysUser{}).Where("id = ?", 2).Update("user_name", conflicting.UserName).Error; err != nil {
		t.Fatal(err)
	}
	user, err := subject.dbUsers.FindAuthenticationByEmail(context.Background(), "employee@example.test")
	if err != nil || user.Id != 1 {
		t.Fatalf("SSO email mapped through a non-email field: user=%+v err=%v", user, err)
	}
}

func TestAuthSMSDoesNotBypassDisabledOrLockedAccount(t *testing.T) {
	for name, setup := range map[string]func(authTestSubject){
		"disabled": func(subject authTestSubject) {
			_ = subject.dbUsers.UpdateAuthenticationState(subject.dbUsers.DBWithContext(context.Background()), 1, map[string]any{"state": false})
		},
		"locked": func(subject authTestSubject) {
			_ = subject.cache.Set(cache.LoginLockCacheKey+cache.UserLoginPrincipal(1), 1, time.Minute)
		},
	} {
		t.Run(name, func(t *testing.T) {
			subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", PhoneNumber: "13800138000", Password: "unused"})
			setup(subject)
			key := cache.SMSVerificationKey(7, "13800138000")
			if err := cache.NewSendCodeCache(subject.cache).Set(key, "123456"); err != nil {
				t.Fatal(err)
			}
			result, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{
				Channel: AuthChannelSMS, CredentialType: AuthCredentialSMS,
				Principal: "13800138000", Secret: "123456", Application: model.Application{Basic: model.Basic{Id: 7, State: true}},
			})
			if !errors.Is(err, myerrors.ErrAuthenticationFailed) || result.AccessToken != "" {
				t.Fatalf("expected account state rejection, result=%+v err=%v", result, err)
			}
		})
	}
}

func TestAuthNonPasswordCredentialCannotBypassRequiredPasswordChange(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", PhoneNumber: "13800138000", Password: "unused", IsReset: true})
	key := cache.SMSVerificationKey(7, "13800138000")
	if err := cache.NewSendCodeCache(subject.cache).Set(key, "123456"); err != nil {
		t.Fatal(err)
	}
	result, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{
		Channel: AuthChannelSMS, CredentialType: AuthCredentialSMS, Principal: "13800138000", Secret: "123456",
		Application: model.Application{Basic: model.Basic{Id: 7, State: true}},
	})
	if !errors.Is(err, myerrors.ErrAuthenticationFailed) || result.AccessToken != "" {
		t.Fatalf("SMS bypassed required password change: result=%+v err=%v", result, err)
	}
}

func TestAuthSSOIdentityUsesSharedAccountPolicy(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "unused"})
	subject.service.providers[AuthCredentialDingTalk] = authStaticCredentialProvider{
		credentialType: AuthCredentialDingTalk,
		verification: CredentialVerification{
			Verified: true, Principal: "external-principal",
			Identity: ConfirmedIdentity{UserID: 1, AuthenticationMethod: AuthCredentialDingTalk, CredentialSource: string(AuthChannelDingTalk)},
		},
	}
	result, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelDingTalk, CredentialType: AuthCredentialDingTalk})
	if err != nil || result.AccessToken == "" {
		t.Fatalf("sso authentication through shared application service: result=%+v err=%v", result, err)
	}
	if err := subject.cache.Set(cache.LoginLockCacheKey+cache.UserLoginPrincipal(1), 1, time.Minute); err != nil {
		t.Fatal(err)
	}
	if result, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelDingTalk, CredentialType: AuthCredentialDingTalk}); !errors.Is(err, myerrors.ErrAuthenticationFailed) || result.AccessToken != "" {
		t.Fatalf("sso bypassed shared account lock: result=%+v err=%v", result, err)
	}
}

func TestAuthRefreshSingleUseAndAccountChecks(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	login, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := subject.service.Refresh(context.Background(), login.RefreshToken)
	if err != nil || refreshed.RefreshToken == "" || refreshed.MustChangePassword || refreshed.PasswordChangeReason != "" {
		t.Fatalf("refresh: %+v %v audit=%+v", refreshed, err, subject.audit.events)
	}
	originalClaims, err := subject.tokens.codec.ParseToken(login.RefreshToken, subject.tokens.conf)
	if err != nil {
		t.Fatal(err)
	}
	refreshedClaims, err := subject.tokens.codec.ParseToken(refreshed.RefreshToken, subject.tokens.conf)
	if err != nil || originalClaims.TokenID == "" || refreshedClaims.TokenID == "" || originalClaims.TokenID == refreshedClaims.TokenID || originalClaims.SessionID == "" || originalClaims.SessionID != refreshedClaims.SessionID {
		t.Fatalf("refresh token rotation must use a new token id: before=%+v after=%+v err=%v", originalClaims, refreshedClaims, err)
	}
	if _, err := subject.service.Refresh(context.Background(), login.RefreshToken); !errors.Is(err, myerrors.ErrInvalidRefreshToken) {
		t.Fatalf("expected consumed refresh rejection, got %v", err)
	}
	if err := subject.dbUsers.UpdateAuthenticationState(subject.dbUsers.DBWithContext(context.Background()), 1, map[string]any{"state": false}); err != nil {
		t.Fatal(err)
	}
	if _, err := subject.service.Refresh(context.Background(), refreshed.RefreshToken); !errors.Is(err, myerrors.ErrInvalidRefreshToken) {
		t.Fatalf("expected disabled refresh rejection, got %v", err)
	}
}

func TestAuthPasswordChangeRequiredSessionCannotRefresh(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{
		Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct", IsReset: true,
	})
	login, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{
		Channel: AuthChannelAdminPassword, CredentialType: AuthCredentialPassword,
		Principal: "admin", Secret: "correct",
	})
	if err != nil || !login.MustChangePassword {
		t.Fatalf("restricted login: %+v %v", login, err)
	}
	access, err := subject.service.AuthenticateAccessToken(context.Background(), login.AccessToken)
	if err != nil || !access.MustChangePassword || access.PasswordChangeReason != "initial_reset" {
		t.Fatalf("restricted access state: %+v %v", access, err)
	}
	if _, err := subject.service.Refresh(context.Background(), login.RefreshToken); !errors.Is(err, myerrors.ErrPasswordChangeRequired) {
		t.Fatalf("password-change-required session refreshed: %v", err)
	}
}

func TestAuthRefreshRejectsLockedAndExplicitlyRevokedTokens(t *testing.T) {
	for name, revoke := range map[string]func(*testing.T, authTestSubject, AuthenticationResult){
		"locked": func(_ *testing.T, subject authTestSubject, _ AuthenticationResult) {
			_ = subject.cache.Set(cache.LoginLockCacheKey+cache.UserLoginPrincipal(1), 1, time.Minute)
		},
		"revoked": func(t *testing.T, subject authTestSubject, login AuthenticationResult) {
			claims, err := subject.tokens.codec.ParseToken(login.RefreshToken, subject.tokens.conf)
			if err != nil {
				t.Fatalf("parse refresh: %v", err)
			}
			if err := subject.tokens.state.Revoke(enum.RefreshToken, login.RefreshToken, claims.ExpiresAt); err != nil {
				t.Fatalf("revoke refresh: %v", err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
			login, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
			if err != nil {
				t.Fatal(err)
			}
			revoke(t, subject, login)
			if _, err := subject.service.Refresh(context.Background(), login.RefreshToken); err == nil {
				t.Fatal("expected refresh rejection")
			}
		})
	}
}

func TestAuthLogoutInvalidatesAccessAndRefresh(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	login, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAdminPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatal(err)
	}
	if err := subject.service.Logout(context.Background(), login.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := subject.service.AuthenticateAccessToken(context.Background(), login.AccessToken); err == nil {
		t.Fatal("expected access token revoked")
	}
	if _, err := subject.service.Refresh(context.Background(), login.RefreshToken); err == nil {
		t.Fatal("expected refresh invalidated by logout")
	}
	if err := subject.service.Logout(context.Background(), login.AccessToken); err != nil {
		t.Fatalf("expected repeated logout to be idempotent: %v", err)
	}
}

func TestAuthNewLoginDoesNotResurrectLoggedOutRefreshToken(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	first, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAdminPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatal(err)
	}
	if err := subject.service.Logout(context.Background(), first.AccessToken); err != nil {
		t.Fatal(err)
	}
	second, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAdminPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatalf("new credential login after logout: %v", err)
	}
	if _, err := subject.service.AuthenticateAccessToken(context.Background(), second.AccessToken); err != nil {
		t.Fatalf("new session must be usable: %v", err)
	}
	if err := subject.service.Logout(context.Background(), first.AccessToken); err != nil {
		t.Fatalf("repeated old-session logout must remain idempotent: %v", err)
	}
	if _, err := subject.service.AuthenticateAccessToken(context.Background(), second.AccessToken); err != nil {
		t.Fatalf("old logout token revoked a new session: %v", err)
	}
	if _, err := subject.service.Refresh(context.Background(), first.RefreshToken); !errors.Is(err, myerrors.ErrInvalidRefreshToken) {
		t.Fatalf("logged-out refresh token was resurrected: %v", err)
	}
	if _, err := subject.service.Refresh(context.Background(), second.RefreshToken); err != nil {
		t.Fatalf("new session refresh: %v", err)
	}
}

func TestAuthLogoutOnlyClosesCurrentSession(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	first, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAdminPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatal(err)
	}
	if err := subject.service.Logout(context.Background(), first.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := subject.service.Refresh(context.Background(), first.RefreshToken); !errors.Is(err, myerrors.ErrInvalidRefreshToken) {
		t.Fatalf("logged-out session refresh survived: %v", err)
	}
	if _, err := subject.service.AuthenticateAccessToken(context.Background(), second.AccessToken); err != nil {
		t.Fatalf("logout incorrectly revoked another session: %v", err)
	}
	if _, err := subject.service.Refresh(context.Background(), second.RefreshToken); err != nil {
		t.Fatalf("logout incorrectly revoked another session refresh: %v", err)
	}
}

func TestAuthConcurrentRefreshOnlyOneSucceeds(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	login, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 12
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := subject.service.Refresh(context.Background(), login.RefreshToken); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("expected one successful refresh, got %d", successes)
	}
}

func TestAuthLogoutAndRefreshRaceLeavesNoUsableSession(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	login, err := subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "admin", Secret: "correct"})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var refreshed AuthenticationResult
	var refreshErr, logoutErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		refreshed, refreshErr = subject.service.Refresh(context.Background(), login.RefreshToken)
	}()
	go func() {
		defer wg.Done()
		<-start
		logoutErr = subject.service.Logout(context.Background(), login.AccessToken)
	}()
	close(start)
	wg.Wait()
	if logoutErr != nil {
		t.Fatalf("logout failed: %v", logoutErr)
	}
	if refreshErr == nil {
		if _, err := subject.service.AuthenticateAccessToken(context.Background(), refreshed.AccessToken); err == nil {
			t.Fatal("refresh won the race but its session survived logout")
		}
	}
	user, err := subject.dbUsers.FindAuthenticationByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if user.AccessTokens != "" {
		t.Fatalf("logout/refresh race left login state behind: %q", user.AccessTokens)
	}
}

func TestAuthAuditNeverContainsCredentialSecrets(t *testing.T) {
	subject := newAuthTestSubject(t, model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin", Password: "correct"})
	secret := "very-secret-password"
	_, _ = subject.service.Authenticate(context.Background(), AuthenticationRequest{Channel: AuthChannelAPIPassword, CredentialType: AuthCredentialPassword, Principal: "unknown-account", Secret: secret})
	if len(subject.audit.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(subject.audit.events))
	}
	event := subject.audit.events[0]
	if event.Principal == secret || event.ReasonCode == secret {
		t.Fatal("authentication audit leaked credential material")
	}
}
