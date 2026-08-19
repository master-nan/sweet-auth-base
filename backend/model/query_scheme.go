package model

import "gorm.io/datatypes"

type QuerySchemeType string

const (
	QuerySchemeTypePersonal    QuerySchemeType = "PERSONAL"
	QuerySchemeTypePublic      QuerySchemeType = "PUBLIC"
	QuerySchemeTypeRole        QuerySchemeType = "ROLE"
	QuerySchemeTypePageDefault QuerySchemeType = "PAGE_DEFAULT"
)

func (value QuerySchemeType) Valid() bool {
	switch value {
	case QuerySchemeTypePersonal, QuerySchemeTypePublic, QuerySchemeTypeRole, QuerySchemeTypePageDefault:
		return true
	default:
		return false
	}
}

type QueryScheme struct {
	Basic
	Name               string          `gorm:"size:64;not null;index:idx_query_scheme_scope_type" json:"-"`
	ScopeCode          string          `gorm:"size:128;not null;index:idx_query_scheme_scope_type" json:"-"`
	SchemeType         QuerySchemeType `gorm:"size:24;not null;index:idx_query_scheme_scope_type" json:"-"`
	OwnerUserID        *int            `gorm:"type:bigint;index:idx_query_scheme_owner_scope" json:"-"`
	QuerySchemaVersion int16           `gorm:"not null;default:1" json:"-"`
	QueryPayload       datatypes.JSON  `gorm:"type:jsonb;not null" json:"-"`
	IsDefault          bool            `gorm:"not null;default:false" json:"-"`
	Enabled            bool            `gorm:"not null;default:true;index:idx_query_scheme_scope_type" json:"-"`
	Revision           int             `gorm:"not null;default:1" json:"-"`
}

type QuerySchemeRole struct {
	SchemeID int `gorm:"type:bigint;primaryKey;autoIncrement:false" json:"-"`
	RoleID   int `gorm:"type:bigint;primaryKey;autoIncrement:false;index:idx_query_scheme_role_role" json:"-"`
}
