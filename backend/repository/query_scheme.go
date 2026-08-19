package repository

import (
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type QuerySchemeListFilter struct {
	Name          string
	ScopeCode     string
	SchemeType    model.QuerySchemeType
	Enabled       *bool
	Page          int
	Num           int
	UserID        int
	RoleIDs       []int
	SharedManager bool
}

type QuerySchemePage struct {
	Data  []model.QueryScheme
	Total int64
}

type QuerySchemeRepository interface {
	BasicRepository[model.QueryScheme]
	FindByIDWithDB(*gorm.DB, int, bool) (model.QueryScheme, error)
	FindVisibleByScope(context.Context, int, []int, string) ([]model.QueryScheme, error)
	List(context.Context, QuerySchemeListFilter) (QuerySchemePage, error)
	RoleIDs(*gorm.DB, int) ([]int, error)
	FindRoleIDsBySchemeIDs(context.Context, []int) (map[int][]int, error)
	ReplaceRoles(*gorm.DB, int, []int) error
	ClearDefault(*gorm.DB, model.QuerySchemeType, int, string, int) error
	DeleteByRevision(*gorm.DB, int, int) (bool, error)
	FindActiveScopeMenu(context.Context, int, string) (model.SysMenu, error)
	FindActiveScopeLabels(context.Context, []string) (map[string]string, error)
	ActiveRoleIDs(context.Context, int) ([]int, error)
	EmployeeID(context.Context, int) (*int, error)
	CountActiveRoles(context.Context, []int) (int64, error)
	HasSharedManageCapability(context.Context, int, string) (bool, error)
}
