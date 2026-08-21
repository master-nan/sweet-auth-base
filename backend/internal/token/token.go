/**
 * @Author: Nan
 * @Date: 2024/1/17 上午10:26
 */

package token

import (
	"backend/enum"
	"time"
)

// Config 统一的token配置
type Config struct {
	// Issuer token的签发者
	Issuer string
	// SecretKey 密钥
	SecretKey string
	// AccessTokenExpiration 访问令牌过期时间(小时)
	AccessTokenExpiration int64
	// RefreshTokenExpiration 刷新令牌过期时间(小时)
	RefreshTokenExpiration int64
}

// Claims token的声明信息
type Claims struct {
	// ID 用户ID或应用ID
	ID string
	// TokenID 服务端生成的令牌唯一标识
	TokenID string
	// SessionID 服务端会话标识
	SessionID string
	// Type token类型
	Type enum.TokenTypeEnum
	// IssuedAt 签发时间
	IssuedAt time.Time
	// ExpiresAt 过期时间
	ExpiresAt time.Time
	// NotBefore 生效时间
	NotBefore time.Time
}

type JWTToken struct {
	Generator
}

type HMACToken struct {
	Generator
}

// Generator token生成器接口
type Generator interface {
	// GenerateToken 生成token
	// claims: token的声明信息
	// config: token的配置信息
	GenerateToken(claims Claims, config Config) (string, error)

	// ParseToken 解析token
	// token: 待解析的token
	// config: token的配置信息
	ParseToken(token string, config Config) (*Claims, error)
}
