package service

import (
	"backend/config"
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/errors"
	"backend/internal/token"
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

const (
	authAccessTokenTTL  = 30 * time.Minute
	authRefreshTokenTTL = 30 * 24 * time.Hour
)

type AuthTokenPair struct {
	UserID       int
	AccessToken  string
	RefreshToken string
	IssuedAt     time.Time
	SessionID    string
}

// AuthTokenService 管理Token签发、校验、刷新和撤销，
// 不负责Credential校验或业务权限判断。
type AuthTokenService struct {
	codec token.JWTToken
	state *cache.TokenBlackCache
	conf  token.Config
}

func NewAuthTokenService(codec token.JWTToken, state *cache.TokenBlackCache, serverConfig *config.Server) *AuthTokenService {
	return &AuthTokenService{
		codec: codec,
		state: state,
		conf: token.Config{
			Issuer:                 serverConfig.Name,
			SecretKey:              serverConfig.Conf.Salt,
			AccessTokenExpiration:  int64(authAccessTokenTTL / time.Second),
			RefreshTokenExpiration: int64(authRefreshTokenTTL / time.Second),
		},
	}
}

func (s *AuthTokenService) Issue(_ context.Context, userID int, issuedAt time.Time) (AuthTokenPair, error) {
	sessionID, err := newAuthTokenID()
	if err != nil {
		return AuthTokenPair{}, errors.WrapSystemError(err)
	}
	if err := s.state.ActivateSession(userID, sessionID, authRefreshTokenTTL); err != nil {
		return AuthTokenPair{}, errors.WrapSystemError(err)
	}
	pair, err := s.issue(userID, issuedAt, sessionID)
	if err != nil {
		_ = s.state.DeactivateSession(userID, sessionID)
	}
	return pair, err
}

func (s *AuthTokenService) IssueRefresh(_ context.Context, userID int, issuedAt time.Time, sessionID string) (AuthTokenPair, error) {
	created := false
	if sessionID == "" {
		var err error
		sessionID, err = newAuthTokenID()
		if err != nil {
			return AuthTokenPair{}, errors.WrapSystemError(err)
		}
		if err := s.state.ActivateSession(userID, sessionID, authRefreshTokenTTL); err != nil {
			return AuthTokenPair{}, errors.WrapSystemError(err)
		}
		created = true
	} else {
		active, err := s.state.TouchSession(userID, sessionID, authRefreshTokenTTL)
		if err != nil {
			return AuthTokenPair{}, errors.WrapSystemError(err)
		}
		if !active {
			return AuthTokenPair{}, errors.ErrInvalidRefreshToken
		}
	}
	pair, err := s.issue(userID, issuedAt, sessionID)
	if err != nil && created {
		_ = s.state.DeactivateSession(userID, sessionID)
	}
	return pair, err
}

func (s *AuthTokenService) issue(userID int, issuedAt time.Time, sessionID string) (AuthTokenPair, error) {
	issuedAt = issuedAt.UTC().Truncate(time.Second)
	tokenID, err := newAuthTokenID()
	if err != nil {
		return AuthTokenPair{}, errors.WrapSystemError(err)
	}
	notBefore := issuedAt
	current := time.Now().UTC().Truncate(time.Second)
	if notBefore.After(current) {
		notBefore = current
	}
	access, err := s.codec.GenerateToken(token.Claims{
		ID: strconv.Itoa(userID), TokenID: tokenID, SessionID: sessionID, Type: enum.AccessToken,
		IssuedAt: issuedAt, NotBefore: notBefore, ExpiresAt: issuedAt.Add(authAccessTokenTTL),
	}, s.conf)
	if err != nil {
		return AuthTokenPair{}, err
	}
	refresh, err := s.codec.GenerateToken(token.Claims{
		ID: strconv.Itoa(userID), TokenID: tokenID, SessionID: sessionID, Type: enum.RefreshToken,
		IssuedAt: issuedAt, NotBefore: notBefore, ExpiresAt: issuedAt.Add(authRefreshTokenTTL),
	}, s.conf)
	if err != nil {
		_ = s.state.Revoke(enum.AccessToken, access, issuedAt.Add(authAccessTokenTTL))
		return AuthTokenPair{}, err
	}
	return AuthTokenPair{UserID: userID, AccessToken: access, RefreshToken: refresh, IssuedAt: issuedAt, SessionID: sessionID}, nil
}

func newAuthTokenID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func (s *AuthTokenService) ValidateAccess(_ context.Context, value string) (*token.Claims, error) {
	return s.validate(value, enum.AccessToken)
}

func (s *AuthTokenService) ValidateRefresh(_ context.Context, value string) (*token.Claims, error) {
	claims, err := s.validate(value, enum.RefreshToken)
	if err != nil {
		if errors.CategoryOf(err) == errors.CategorySystem {
			return nil, err
		}
		return nil, errors.ErrInvalidRefreshToken
	}
	return claims, nil
}

func (s *AuthTokenService) ParseForLogout(value string) (*token.Claims, error) {
	claims, err := s.codec.ParseToken(value, s.conf)
	if err != nil || claims.Type != enum.AccessToken {
		return nil, errors.ErrUserNotLogin
	}
	if _, err := strconv.Atoi(claims.ID); err != nil {
		return nil, errors.ErrTokenInvalid
	}
	return claims, nil
}

func (s *AuthTokenService) ConsumeRefresh(value string, expiresAt time.Time) (bool, error) {
	return s.state.ConsumeRefresh(value, expiresAt)
}

func (s *AuthTokenService) RevokePair(pair AuthTokenPair) {
	_ = s.state.Revoke(enum.AccessToken, pair.AccessToken, pair.IssuedAt.Add(authAccessTokenTTL))
	_ = s.state.Revoke(enum.RefreshToken, pair.RefreshToken, pair.IssuedAt.Add(authRefreshTokenTTL))
	if pair.UserID > 0 && pair.SessionID != "" {
		_ = s.state.DeactivateSession(pair.UserID, pair.SessionID)
	}
}

func (s *AuthTokenService) RevokeAccessAndSession(value string) (int, error) {
	claims, err := s.ParseForLogout(value)
	if err != nil {
		return 0, err
	}
	userID, _ := strconv.Atoi(claims.ID)
	if err := s.state.Revoke(enum.AccessToken, value, claims.ExpiresAt); err != nil {
		return 0, errors.WrapSystemError(err)
	}
	if err := s.state.DeactivateSession(userID, claims.SessionID); err != nil {
		return 0, errors.WrapSystemError(err)
	}
	return userID, nil
}

func (s *AuthTokenService) RevokeRefreshAndSession(value string) (int, string, error) {
	claims, err := s.ValidateRefresh(context.Background(), value)
	if err != nil {
		return 0, "", err
	}
	userID, _ := strconv.Atoi(claims.ID)
	if err := s.state.Revoke(enum.RefreshToken, value, claims.ExpiresAt); err != nil {
		return 0, "", errors.WrapSystemError(err)
	}
	if err := s.state.DeactivateSession(userID, claims.SessionID); err != nil {
		return 0, "", errors.WrapSystemError(err)
	}
	return userID, claims.SessionID, nil
}

func (s *AuthTokenService) RevokeSessionDigest(userID int, sessionDigest string) error {
	if err := s.state.DeactivateSessionDigest(userID, sessionDigest); err != nil {
		return errors.WrapSystemError(err)
	}
	return nil
}

func (s *AuthTokenService) IsSessionActive(userID int, sessionID string) (bool, error) {
	active, err := s.state.IsSessionActive(userID, sessionID)
	if err != nil {
		return false, errors.WrapSystemError(err)
	}
	return active, nil
}

func (s *AuthTokenService) validate(value string, expected enum.TokenTypeEnum) (*token.Claims, error) {
	claims, err := s.codec.ParseToken(value, s.conf)
	if err != nil {
		return nil, err
	}
	if claims.Type != expected {
		return nil, errors.ErrTokenInvalidType
	}
	userID, err := strconv.Atoi(claims.ID)
	if err != nil || userID <= 0 {
		return nil, errors.ErrTokenInvalid
	}
	if claims.TokenID == "" || claims.SessionID == "" {
		return nil, errors.ErrTokenInvalid
	}
	revoked, err := s.state.IsRevoked(expected, value)
	if err != nil {
		return nil, errors.WrapSystemError(err)
	}
	if revoked {
		return nil, errors.ErrTokenExpired
	}
	active, err := s.state.IsSessionActive(userID, claims.SessionID)
	if err != nil {
		return nil, errors.WrapSystemError(err)
	}
	if !active {
		return nil, errors.ErrTokenExpired
	}
	return claims, nil
}
