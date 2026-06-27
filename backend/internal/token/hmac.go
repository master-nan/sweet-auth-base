/**
 * @Author: Nan
 * @Date: 2024/1/17 上午10:26
 */

package token

import (
	"backend/internal/errors"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// HMACGenerator HMAC令牌生成器
type HMACGenerator struct{}

// NewHMACGenerator 创建HMAC令牌生成器
func NewHMACGenerator() *HMACGenerator {
	return &HMACGenerator{}
}

// GenerateToken 生成HMAC令牌
func (g *HMACGenerator) GenerateToken(claims Claims, config Config) (string, error) {
	timestamp := claims.IssuedAt.Unix()
	expiration := timestamp + config.AccessTokenExpiration
	message := claims.ID + strconv.FormatInt(timestamp, 10) + strconv.FormatInt(expiration, 10)
	mac := hmac.New(sha256.New, []byte(config.SecretKey))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil)) + "." + strconv.FormatInt(expiration, 10), nil
}

// ParseToken 解析HMAC令牌
func (g *HMACGenerator) ParseToken(token string, config Config) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.ErrTokenInvalid
	}
	tokenPart := parts[0]
	expiration, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	if time.Now().Unix() > expiration {
		return nil, errors.ErrTokenExpired
	}
	message := config.Issuer + strconv.FormatInt(expiration-config.AccessTokenExpiration, 10) + strconv.FormatInt(expiration, 10)
	mac := hmac.New(sha256.New, []byte(config.SecretKey))
	mac.Write([]byte(message))
	expectedToken := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(tokenPart), []byte(expectedToken)) {
		return nil, errors.ErrTokenInvalid
	}
	tokenClaims := &Claims{
		ExpiresAt: time.Unix(expiration, 0),
		IssuedAt:  time.Unix(expiration-config.AccessTokenExpiration*3600, 0),
		NotBefore: time.Unix(expiration-config.AccessTokenExpiration*3600, 0),
	}
	return tokenClaims, nil
}
