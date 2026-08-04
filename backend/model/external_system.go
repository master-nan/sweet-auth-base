package model

const (
	ExternalSystemStatusDraft    = "draft"
	ExternalSystemStatusEnabled  = "enabled"
	ExternalSystemStatusDisabled = "disabled"

	ExternalSystemTypeHR    = "hr"
	ExternalSystemTypeERP   = "erp"
	ExternalSystemTypeTMS   = "tms"
	ExternalSystemTypeWMS   = "wms"
	ExternalSystemTypeOther = "other"
)

// ExternalSystem 描述一个受平台管理的外围系统边界，不保存凭证或执行规则。
type ExternalSystem struct {
	Basic

	SystemCode      string `gorm:"size:64;not null;uniqueIndex:uni_integration_external_system_code" json:"system_code"`
	Name            string `gorm:"size:128;not null;index:idx_integration_external_system_name" json:"name"`
	SystemType      string `gorm:"size:32;not null;index:idx_integration_external_system_type" json:"system_type"`
	BaseURL         string `gorm:"size:512;not null" json:"base_url"`
	OwnerIdentifier string `gorm:"size:128;not null;index:idx_integration_external_system_owner" json:"owner_identifier"`
	OwnerName       string `gorm:"size:128;not null" json:"owner_name"`
	Status          string `gorm:"size:16;not null;default:draft;index:idx_integration_external_system_status" json:"status"`
	Description     string `gorm:"size:512" json:"description"`
	Revision        int    `gorm:"not null;default:1" json:"revision"`
}

func (ExternalSystem) TableName() string {
	return "integration_external_system"
}
