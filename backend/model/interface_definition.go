package model

import "gorm.io/datatypes"

const (
	InterfaceDefinitionStatusDraft    = "draft"
	InterfaceDefinitionStatusEnabled  = "enabled"
	InterfaceDefinitionStatusDisabled = "disabled"

	InterfaceProtocolHTTP  = "http"
	InterfaceProtocolHTTPS = "https"

	InterfaceMethodGET    = "GET"
	InterfaceMethodPOST   = "POST"
	InterfaceMethodPUT    = "PUT"
	InterfaceMethodPATCH  = "PATCH"
	InterfaceMethodDELETE = "DELETE"

	InterfaceIdempotencyModeNone             = "none"
	InterfaceIdempotencyModeSafeMethod       = "safe_method"
	InterfaceIdempotencyModeIdempotentMethod = "idempotent_method"
	InterfaceIdempotencyModeRemoteKeyHeader  = "remote_key_header"
)

// InterfaceDefinition 描述外部系统接口的版本化技术契约，不包含执行和转换逻辑。
type InterfaceDefinition struct {
	Basic

	ExternalSystemID        int            `gorm:"type:bigint;not null;index:idx_integration_interface_system;uniqueIndex:uni_integration_interface_identity,priority:1" json:"external_system_id"`
	InterfaceCode           string         `gorm:"size:64;not null;uniqueIndex:uni_integration_interface_identity,priority:2" json:"interface_code"`
	Name                    string         `gorm:"size:128;not null;index:idx_integration_interface_name" json:"name"`
	Version                 int            `gorm:"not null;uniqueIndex:uni_integration_interface_identity,priority:3" json:"version"`
	Protocol                string         `gorm:"size:16;not null" json:"protocol"`
	HTTPMethod              string         `gorm:"size:16;not null;index:idx_integration_interface_method" json:"http_method"`
	RelativePath            string         `gorm:"size:512;not null" json:"relative_path"`
	InputContract           datatypes.JSON `gorm:"type:jsonb;not null;default:'{\"version\":1,\"parameters\":[]}'" json:"input_contract"`
	CredentialID            *int           `gorm:"type:bigint;index:idx_integration_interface_credential" json:"credential_id"`
	TimeoutSeconds          int            `gorm:"not null;default:30" json:"timeout_seconds"`
	ResponseLimit           int64          `gorm:"not null;default:10485760" json:"response_limit"`
	RetryPolicyID           *int           `gorm:"type:bigint;index:idx_integration_interface_retry_policy" json:"retry_policy_id"`
	IdempotencyMode         string         `gorm:"size:32;not null;default:none" json:"idempotency_mode"`
	RemoteIdempotencyHeader string         `gorm:"size:64" json:"remote_idempotency_header"`
	Status                  string         `gorm:"size:16;not null;default:draft;index:idx_integration_interface_status" json:"status"`
	Description             string         `gorm:"size:512" json:"description"`
	Revision                int            `gorm:"not null;default:1" json:"revision"`

	ExternalSystem ExternalSystem `gorm:"foreignKey:ExternalSystemID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

func (InterfaceDefinition) TableName() string {
	return "integration_interface_definition"
}
