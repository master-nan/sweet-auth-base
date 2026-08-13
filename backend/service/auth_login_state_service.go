package service

import (
	"backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type AuthLoginStateService struct {
	users repository.AuthenticationUserRepository
}

func NewAuthLoginStateService(users repository.AuthenticationUserRepository) *AuthLoginStateService {
	return &AuthLoginStateService{users: users}
}

func (s *AuthLoginStateService) RecordLogin(ctx context.Context, userID int, accessToken string, at time.Time) error {
	return RunInTransaction(ctx, s.users.DBWithContext(ctx), func(tx *gorm.DB) error {
		user, err := s.users.FindAuthenticationByIDForUpdate(tx, userID)
		if err != nil {
			return errors.WrapDatabaseError(err)
		}
		digest := sha256.Sum256([]byte(accessToken))
		return s.users.UpdateAuthenticationState(tx, userID, map[string]any{
			"access_tokens":  utils.UpdateAccessTokens(onlyAccessTokenDigests(user.AccessTokens), fmt.Sprintf("sha256:%x", digest)),
			"gmt_last_login": model.CustomTime(at),
		})
	})
}

func (s *AuthLoginStateService) Logout(ctx context.Context, userID int, accessToken string) error {
	return RunInTransaction(ctx, s.users.DBWithContext(ctx), func(tx *gorm.DB) error {
		user, err := s.users.FindAuthenticationByIDForUpdate(tx, userID)
		if err != nil {
			return errors.WrapDatabaseError(err)
		}
		return s.users.UpdateAuthenticationState(tx, userID, map[string]any{"access_tokens": removeAccessTokenDigest(user.AccessTokens, accessToken)})
	})
}

func (s *AuthLoginStateService) RollbackLogin(ctx context.Context, userID int, accessToken string) error {
	return RunInTransaction(ctx, s.users.DBWithContext(ctx), func(tx *gorm.DB) error {
		user, err := s.users.FindAuthenticationByIDForUpdate(tx, userID)
		if err != nil {
			return errors.WrapDatabaseError(err)
		}
		return s.users.UpdateAuthenticationState(tx, userID, map[string]any{"access_tokens": removeAccessTokenDigest(user.AccessTokens, accessToken)})
	})
}

func removeAccessTokenDigest(existing, accessToken string) string {
	digest := sha256.Sum256([]byte(accessToken))
	target := fmt.Sprintf("sha256:%x", digest)
	tokens := strings.Split(onlyAccessTokenDigests(existing), ",")
	kept := tokens[:0]
	for _, value := range tokens {
		if value != "" && value != target {
			kept = append(kept, value)
		}
	}
	return strings.Join(kept, ",")
}

func onlyAccessTokenDigests(existing string) string {
	values := strings.Split(existing, ",")
	digests := values[:0]
	for _, value := range values {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "sha256:") && len(value) == len("sha256:")+64 {
			digests = append(digests, value)
		}
	}
	return strings.Join(digests, ",")
}
