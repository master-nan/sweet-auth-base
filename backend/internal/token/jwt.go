/**
 * @Author: Nan
 * @Date: 2024/1/17 上午10:26
 */

package token

import (
	"backend/enum"
	error2 "backend/internal/errors"
	"errors"
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
	tkn := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"iss":  config.Issuer,
		"iat":  jwt.NewNumericDate(claims.IssuedAt),
		"exp":  jwt.NewNumericDate(claims.ExpiresAt),
		"nbf":  jwt.NewNumericDate(claims.NotBefore),
		"sub":  claims.ID,
		"type": claims.Type,
	})
	return tkn.SignedString([]byte(config.SecretKey))
}

// ParseToken 解析JWT令牌
func (g *JWTGenerator) ParseToken(token string, config Config) (*Claims, error) {
	res, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
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
	if !res.Valid {
		return nil, error2.ErrTokenInvalid
	}
	if claims, ok := res.Claims.(jwt.MapClaims); ok {
		tokenClaims := &Claims{
			ID:   claims["sub"].(string),
			Type: enum.TokenTypeEnum(claims["type"].(string)),
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
		return tokenClaims, nil
	} else {
		return nil, error2.ErrTokenInvalid
	}
}
