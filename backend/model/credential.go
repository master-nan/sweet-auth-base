package model

import "time"

const (
	CredentialTypeBasic       = "basic"
	CredentialTypeAPIKey      = "api_key"
	CredentialTypeBearerToken = "bearer_token"
	CredentialTypeOAuthClient = "oauth_client"

	CredentialStatusDraft    = "draft"
	CredentialStatusActive   = "active"
	CredentialStatusDisabled = "disabled"
	CredentialStatusRevoked  = "revoked"
)

// Credential 保存外部系统凭证的配置元数据与加密信封，密钥材料不参与 JSON 序列化。
type Credential struct {
	Basic

	ExternalSystemID  int             `gorm:"type:bigint;not null;uniqueIndex:uni_integration_credential_code,priority:1;index:idx_integration_credential_system" json:"external_system_id"`
	CredentialCode    string          `gorm:"size:64;not null;uniqueIndex:uni_integration_credential_code,priority:2" json:"credential_code"`
	Name              string          `gorm:"size:128;not null;index:idx_integration_credential_name" json:"name"`
	CredentialType    string          `gorm:"size:32;not null;index:idx_integration_credential_type" json:"credential_type"`
	Status            string          `gorm:"size:16;not null;default:draft;index:idx_integration_credential_status" json:"status"`
	SecretStorageRef  string          `gorm:"size:64;not null;uniqueIndex:uni_integration_credential_secret_ref" json:"-"`
	SecretCiphertext  string          `gorm:"type:text;not null" json:"-"`
	SecretNonce       string          `gorm:"size:64;not null" json:"-"`
	SecretFingerprint string          `gorm:"size:64;not null" json:"-"`
	ExpiresAt         *time.Time      `gorm:"type:timestamp;index:idx_integration_credential_expires_at" json:"expires_at"`
	Version           int             `gorm:"not null;default:1" json:"version"`
	RotatedAt         *time.Time      `gorm:"type:timestamp" json:"rotated_at"`
	Description       string          `gorm:"size:512" json:"description"`
	Revision          int             `gorm:"not null;default:1" json:"revision"`
	ExternalSystem    *ExternalSystem `gorm:"foreignKey:ExternalSystemID;references:Id" json:"-"`
}

func (Credential) TableName() string {
	return "integration_credential"
}
