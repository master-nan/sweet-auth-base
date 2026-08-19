package impl

import (
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QuerySchemeRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.QueryScheme]
}

func NewQuerySchemeRepositoryImpl(primaryDB *database.PrimaryDB) *QuerySchemeRepositoryImpl {
	return &QuerySchemeRepositoryImpl{
		db:                  primaryDB.DB,
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.QueryScheme{}),
	}
}

func (repositoryImpl *QuerySchemeRepositoryImpl) FindByIDWithDB(
	db *gorm.DB,
	id int,
	forUpdate bool,
) (model.QueryScheme, error) {
	var value model.QueryScheme
	query := db.Model(&model.QueryScheme{}).Where("id = ?", id)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&value).Error
	return value, err
}

func (repositoryImpl *QuerySchemeRepositoryImpl) FindVisibleByScope(
	ctx context.Context,
	userID int,
	roleIDs []int,
	scopeCode string,
) ([]model.QueryScheme, error) {
	query := repositoryImpl.db.WithContext(ctx).Model(&model.QueryScheme{}).
		Where("scope_code = ? AND state = ?", scopeCode, true).
		Where(`
			(scheme_type = ? AND owner_user_id = ?)
			OR (scheme_type IN ? AND enabled = ?)
			OR (scheme_type = ? AND enabled = ? AND EXISTS (
				SELECT 1 FROM query_scheme_role qsr
				JOIN sys_role role ON role.id = qsr.role_id AND role.state = TRUE AND role.gmt_delete IS NULL
				WHERE qsr.scheme_id = query_scheme.id AND qsr.role_id IN ?
			))`,
			model.QuerySchemeTypePersonal, userID,
			[]model.QuerySchemeType{model.QuerySchemeTypePublic, model.QuerySchemeTypePageDefault}, true,
			model.QuerySchemeTypeRole, true, nonEmptyIDs(roleIDs),
		).
		Order(`CASE
			WHEN scheme_type = 'PERSONAL' AND is_default = TRUE THEN 0
			WHEN scheme_type = 'PAGE_DEFAULT' AND is_default = TRUE THEN 1
			ELSE 2
		END ASC, gmt_modify DESC, lower(name) ASC, id ASC`).Limit(200)
	var values []model.QueryScheme
	err := query.Find(&values).Error
	return values, err
}

func (repositoryImpl *QuerySchemeRepositoryImpl) List(
	ctx context.Context,
	filter repository.QuerySchemeListFilter,
) (repository.QuerySchemePage, error) {
	query := repositoryImpl.db.WithContext(ctx).Model(&model.QueryScheme{}).
		Where("state = ?", true).
		Where(`EXISTS (
			SELECT 1 FROM sys_menu menu
			JOIN sys_role_menu rm ON rm.menu_id = menu.id
			JOIN sys_user_role ur ON ur.role_id = rm.role_id AND ur.user_id = ?
			JOIN sys_role role ON role.id = ur.role_id AND role.state = TRUE AND role.gmt_delete IS NULL
			WHERE menu.query_scope_code = query_scheme.scope_code
				AND menu.state = TRUE AND menu.gmt_delete IS NULL
		)`, filter.UserID)
	if filter.Name != "" {
		query = query.Where("lower(name) LIKE ?", "%"+strings.ToLower(strings.TrimSpace(filter.Name))+"%")
	}
	if filter.ScopeCode != "" {
		query = query.Where("scope_code = ?", filter.ScopeCode)
	}
	if filter.SchemeType.Valid() {
		query = query.Where("scheme_type = ?", filter.SchemeType)
	}
	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}
	if filter.SharedManager {
		query = query.Where("scheme_type <> ? OR owner_user_id = ?", model.QuerySchemeTypePersonal, filter.UserID)
	} else {
		query = query.Where(`
			(scheme_type = ? AND owner_user_id = ?)
			OR (scheme_type IN ? AND enabled = ?)
			OR (scheme_type = ? AND enabled = ? AND EXISTS (
				SELECT 1 FROM query_scheme_role qsr
				JOIN sys_role role ON role.id = qsr.role_id AND role.state = TRUE AND role.gmt_delete IS NULL
				WHERE qsr.scheme_id = query_scheme.id AND qsr.role_id IN ?
			))`,
			model.QuerySchemeTypePersonal, filter.UserID,
			[]model.QuerySchemeType{model.QuerySchemeTypePublic, model.QuerySchemeTypePageDefault}, true,
			model.QuerySchemeTypeRole, true, nonEmptyIDs(filter.RoleIDs),
		)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return repository.QuerySchemePage{}, err
	}
	page, num := normalizePage(filter.Page, filter.Num)
	var values []model.QueryScheme
	if err := query.Order("gmt_modify DESC, id DESC").Offset((page - 1) * num).Limit(num).Find(&values).Error; err != nil {
		return repository.QuerySchemePage{}, err
	}
	return repository.QuerySchemePage{Data: values, Total: total}, nil
}

func (repositoryImpl *QuerySchemeRepositoryImpl) RoleIDs(db *gorm.DB, schemeID int) ([]int, error) {
	var roleIDs []int
	err := db.Model(&model.QuerySchemeRole{}).Where("scheme_id = ?", schemeID).Order("role_id ASC").Pluck("role_id", &roleIDs).Error
	return roleIDs, err
}

func (repositoryImpl *QuerySchemeRepositoryImpl) ReplaceRoles(db *gorm.DB, schemeID int, roleIDs []int) error {
	if err := db.Where("scheme_id = ?", schemeID).Delete(&model.QuerySchemeRole{}).Error; err != nil {
		return err
	}
	if len(roleIDs) == 0 {
		return nil
	}
	values := make([]model.QuerySchemeRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		values = append(values, model.QuerySchemeRole{SchemeID: schemeID, RoleID: roleID})
	}
	return db.Create(&values).Error
}

func (repositoryImpl *QuerySchemeRepositoryImpl) ClearDefault(
	db *gorm.DB,
	schemeType model.QuerySchemeType,
	ownerUserID int,
	scopeCode string,
	excludeID int,
) error {
	query := db.Model(&model.QueryScheme{}).
		Where("scope_code = ? AND scheme_type = ? AND is_default = ?", scopeCode, schemeType, true)
	if schemeType == model.QuerySchemeTypePersonal {
		query = query.Where("owner_user_id = ?", ownerUserID)
	}
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	return query.Updates(map[string]any{
		"is_default": false,
		"revision":   gorm.Expr("revision + 1"),
	}).Error
}

func (repositoryImpl *QuerySchemeRepositoryImpl) DeleteByRevision(db *gorm.DB, id, revision int) (bool, error) {
	result := db.Where("id = ? AND revision = ?", id, revision).Delete(&model.QueryScheme{})
	return result.RowsAffected == 1, result.Error
}

func (repositoryImpl *QuerySchemeRepositoryImpl) FindActiveScopeMenu(
	ctx context.Context,
	userID int,
	scopeCode string,
) (model.SysMenu, error) {
	var menu model.SysMenu
	err := repositoryImpl.db.WithContext(ctx).Model(&model.SysMenu{}).
		Joins("JOIN sys_role_menu rm ON rm.menu_id = sys_menu.id").
		Joins("JOIN sys_user_role ur ON ur.role_id = rm.role_id AND ur.user_id = ?", userID).
		Joins("JOIN sys_role role ON role.id = ur.role_id AND role.state = TRUE AND role.gmt_delete IS NULL").
		Where("sys_menu.query_scope_code = ? AND sys_menu.state = TRUE AND sys_menu.gmt_delete IS NULL", scopeCode).
		Distinct("sys_menu.*").First(&menu).Error
	return menu, err
}

func (repositoryImpl *QuerySchemeRepositoryImpl) ActiveRoleIDs(ctx context.Context, userID int) ([]int, error) {
	var roleIDs []int
	err := repositoryImpl.db.WithContext(ctx).Table("sys_user_role AS ur").
		Joins("JOIN sys_role role ON role.id = ur.role_id AND role.state = TRUE AND role.gmt_delete IS NULL").
		Where("ur.user_id = ?", userID).Order("ur.role_id ASC").Pluck("ur.role_id", &roleIDs).Error
	return roleIDs, err
}

func (repositoryImpl *QuerySchemeRepositoryImpl) EmployeeID(ctx context.Context, userID int) (*int, error) {
	var employee struct{ ID int }
	err := repositoryImpl.db.WithContext(ctx).Model(&model.OrgEmployee{}).
		Select("id").Where("user_id = ? AND state = TRUE AND gmt_delete IS NULL", userID).Take(&employee).Error
	if err != nil {
		return nil, err
	}
	return &employee.ID, nil
}

func (repositoryImpl *QuerySchemeRepositoryImpl) CountActiveRoles(ctx context.Context, roleIDs []int) (int64, error) {
	if len(roleIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := repositoryImpl.db.WithContext(ctx).Model(&model.SysRole{}).
		Where("id IN ? AND state = TRUE", roleIDs).Count(&count).Error
	return count, err
}

func (repositoryImpl *QuerySchemeRepositoryImpl) HasSharedManageCapability(
	ctx context.Context,
	userID int,
	eventAction string,
) (bool, error) {
	var count int64
	err := repositoryImpl.db.WithContext(ctx).Table("sys_user_role AS ur").
		Joins("JOIN sys_role role ON role.id = ur.role_id AND role.state = TRUE AND role.gmt_delete IS NULL").
		Joins("JOIN sys_role_menu_button rmb ON rmb.role_id = ur.role_id").
		Joins("JOIN sys_menu_button button ON button.id = rmb.button_id AND button.state = TRUE AND button.gmt_delete IS NULL").
		Where("ur.user_id = ? AND button.event_action = ? AND button.is_disabled = FALSE", userID, eventAction).
		Count(&count).Error
	return count > 0, err
}

func normalizePage(page, num int) (int, int) {
	if page < 1 {
		page = 1
	}
	if num < 1 {
		num = 20
	}
	if num > 100 {
		num = 100
	}
	return page, num
}

func nonEmptyIDs(values []int) []int {
	if len(values) == 0 {
		return []int{-1}
	}
	return values
}
