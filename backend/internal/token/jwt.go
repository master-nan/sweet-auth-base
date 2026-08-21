/**
 * @Author: Nan
 * @Date: 2024/1/17 上午10:26
 */

package token

import (
	"backend/enum"
	error2 "backend/internal/errors"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTGenerator JWT令牌生成器
type JWTGenerator struct{}

// NewJWTGenerator 创建JWT令牌生成器
func NewJWTGenerator() *JWTGenerator {
	return &JWTGenerator{}
}

// GenerateToken 生成JWT令牌
func (g *JWTGenerator) GenerateToken(claims Claims, config Config) (string, error) {
	values := jwt.MapClaims{
		"iss":  config.Issuer,
		"iat":  jwt.NewNumericDate(claims.IssuedAt),
		"exp":  jwt.NewNumericDate(claims.ExpiresAt),
		"nbf":  jwt.NewNumericDate(claims.NotBefore),
		"sub":  claims.ID,
		"type": claims.Type,
	}
	if claims.TokenID != "" {
		values["jti"] = claims.TokenID
	}
	if claims.SessionID != "" {
		values["sid"] = claims.SessionID
	}
	tkn := jwt.NewWithClaims(jwt.SigningMethodHS512, values)
	return tkn.SignedString([]byte(config.SecretKey))
}

// ParseToken 解析JWT令牌
func (g *JWTGenerator) ParseToken(token string, config Config) (*Claims, error) {
	res, err := jwt.Parse(token, func(parsed *jwt.Token) (interface{}, error) {
		if parsed.Method != jwt.SigningMethodHS512 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(config.SecretKey), nil
	})
	if err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, error2.ErrTokenInvalid
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, error2.ErrTokenExpired
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, error2.ErrTokenInvalid
			} else {
				return nil, error2.ErrTokenInvalid
			}
		}
	}
	if res == nil || !res.Valid {
		return nil, error2.ErrTokenInvalid
	}
	if claims, ok := res.Claims.(jwt.MapClaims); ok {
		subject, subjectOK := claims["sub"].(string)
		typeValue, typeOK := claims["type"].(string)
		issuer, issuerOK := claims["iss"].(string)
		tokenID := stringClaim(claims, "jti")
		sessionID := stringClaim(claims, "sid")
		if !subjectOK || subject == "" || !typeOK || !issuerOK || issuer != config.Issuer || tokenID == "" || sessionID == "" {
			return nil, error2.ErrTokenInvalid
		}
		tokenClaims := &Claims{
			ID:        subject,
			TokenID:   tokenID,
			SessionID: sessionID,
			Type:      enum.TokenTypeEnum(typeValue),
		}
		if iat, ok := claims["iat"].(float64); ok {
			tokenClaims.IssuedAt = time.Unix(int64(iat), 0)
		}
		if exp, ok := claims["exp"].(float64); ok {
			tokenClaims.ExpiresAt = time.Unix(int64(exp), 0)
		}
		if nbf, ok := claims["nbf"].(float64); ok {
			tokenClaims.NotBefore = time.Unix(int64(nbf), 0)
		}
		if tokenClaims.IssuedAt.IsZero() || tokenClaims.ExpiresAt.IsZero() || tokenClaims.NotBefore.IsZero() {
			return nil, error2.ErrTokenInvalid
		}
		return tokenClaims, nil
	} else {
		return nil, error2.ErrTokenInvalid
	}
}

func stringClaim(claims jwt.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}
