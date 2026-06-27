package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	dataPermissionValueTypeString = "string"
	dataPermissionValueTypeNumber = "number"

	dataPermissionSourceTypeNone  = "none"
	dataPermissionSourceTypeTable = "table"

	dataPermissionMatchIn = "in"
	dataPermissionMatchEq = "eq"

	dataPermissionStrategyAll       = "all"
	dataPermissionStrategyNone      = "none"
	dataPermissionStrategySpecified = "specified"
	dataPermissionStrategyTree      = "tree"
	dataPermissionStrategySelf      = "self"

	dataPermissionOverrideReplace   = "replace"
	dataPermissionOverrideUnion     = "union"
	dataPermissionOverrideIntersect = "intersect"
	dataPermissionOverrideDeny      = "deny"
)

var dataPermissionIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)

type DataPermissionService struct {
	db *gorm.DB
	sf *utils.Snowflake
}

type DataPermissionOption struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Parent string `json:"parent,omitempty"`
}

type scopeDecision struct {
	all    bool
	deny   bool
	values map[string]struct{}
}

func NewDataPermissionService(primaryDB *database.PrimaryDB, sf *utils.Snowflake) *DataPermissionService {
	return &DataPermissionService{db: primaryDB.DB, sf: sf}
}

func (s *DataPermissionService) QueryDimensions(_ *request.Basic) ([]model.SysDataDimension, int64, error) {
	var items []model.SysDataDimension
	if err := s.db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, int64(len(items)), nil
}

func (s *DataPermissionService) GetDimensionById(id int) (model.SysDataDimension, error) {
	var item model.SysDataDimension
	err := s.db.First(&item, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysDataDimension{}, nil
	}
	return item, err
}

func (s *DataPermissionService) CreateDimension(ctx *gin.Context, req request.DataPermissionDimensionCreateReq) error {
	dimension, err := s.dimensionFromCreateReq(req)
	if err != nil {
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	dimension.Id = int(id)
	return s.db.WithContext(ctx).Create(&dimension).Error
}

func (s *DataPermissionService) UpdateDimension(ctx *gin.Context, req request.DataPermissionDimensionUpdateReq) error {
	dimension, err := s.dimensionFromUpdateReq(req)
	if err != nil {
		return err
	}
	updates := map[string]interface{}{
		"code":         dimension.Code,
		"name":         dimension.Name,
		"value_type":   dimension.ValueType,
		"source_type":  dimension.SourceType,
		"source_code":  dimension.SourceCode,
		"label_field":  dimension.LabelField,
		"value_field":  dimension.ValueField,
		"parent_field": dimension.ParentField,
		"memo":         dimension.Memo,
		"state":        dimension.State,
	}
	return s.db.WithContext(ctx).Model(&model.SysDataDimension{}).Where("id = ?", req.Id).Updates(updates).Error
}

func (s *DataPermissionService) DeleteDimension(ctx *gin.Context, id int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dimension model.SysDataDimension
		if err := tx.First(&dimension, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		var count int64
		if err := tx.Model(&model.SysDataScopeBinding{}).Where("dimension_code = ? AND state = ?", dimension.Code, true).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return myerrors.NewBadRequestError("数据权限维度已被菜单绑定使用")
		}
		return tx.Delete(&dimension).Error
	})
}

func (s *DataPermissionService) GetMenuBindings(menuId int) ([]model.SysDataScopeBinding, error) {
	var bindings []model.SysDataScopeBinding
	if err := s.db.Preload("Dimension").Where("menu_id = ?", menuId).Order("id ASC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	for i := range bindings {
		bindings[i].ActionList = decodeStringList(bindings[i].Actions)
	}
	return bindings, nil
}

func (s *DataPermissionService) SaveMenuBindings(ctx *gin.Context, menuId int, req request.DataPermissionBindingSaveReq) error {
	if req.MenuId > 0 && req.MenuId != menuId {
		return myerrors.ErrParamInvalid
	}
	menu, table, err := s.boundMenuTable(menuId)
	if err != nil {
		return err
	}
	records := make([]model.SysDataScopeBinding, 0, len(req.Bindings))
	for _, item := range req.Bindings {
		record, err := s.bindingFromReq(menu, table, item)
		if err != nil {
			return err
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		record.Id = int(id)
		records = append(records, record)
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", menuId).Delete(&model.SysDataScopeBinding{}).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		return tx.Create(&records).Error
	})
}

func (s *DataPermissionService) GetRoleDataScopes(roleId int) ([]model.SysRoleDataScope, error) {
	var scopes []model.SysRoleDataScope
	err := s.db.Preload("Menu").Preload("Dimension").Where("role_id = ?", roleId).Order("menu_id ASC, id ASC").Find(&scopes).Error
	if err != nil {
		return nil, err
	}
	for i := range scopes {
		scopes[i].ScopeValueList = decodeStringList(scopes[i].ScopeValues)
	}
	return scopes, nil
}

func (s *DataPermissionService) SaveRoleDataScopes(ctx *gin.Context, roleId int, req request.RoleDataPermissionSaveReq) error {
	if req.RoleId > 0 && req.RoleId != roleId {
		return myerrors.ErrParamInvalid
	}
	records, err := s.BuildRoleDataScopeRecords(roleId, req.Permissions)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.ReplaceRoleDataScopes(tx, roleId, records)
	})
}

func (s *DataPermissionService) BuildRoleDataScopeRecords(roleId int, permissions []request.RoleDataPermissionItemReq) ([]model.SysRoleDataScope, error) {
	records := make([]model.SysRoleDataScope, 0, len(permissions))
	for _, item := range permissions {
		record, err := s.roleScopeFromReq(roleId, item)
		if err != nil {
			return nil, err
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return nil, err
		}
		record.Id = int(id)
		records = append(records, record)
	}
	return records, nil
}

func (s *DataPermissionService) ReplaceRoleDataScopes(tx *gorm.DB, roleId int, records []model.SysRoleDataScope) error {
	if tx == nil {
		tx = s.db
	}
	if err := tx.Where("role_id = ?", roleId).Delete(&model.SysRoleDataScope{}).Error; err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Create(&records).Error
}

func (s *DataPermissionService) DeleteRoleScopesByRoleId(tx *gorm.DB, roleId int) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Where("role_id = ?", roleId).Delete(&model.SysRoleDataScope{}).Error
}

func (s *DataPermissionService) DeleteRoleScopesOutsideMenus(tx *gorm.DB, roleId int, menuIds []int) error {
	if tx == nil {
		tx = s.db
	}
	if len(menuIds) == 0 {
		return s.DeleteRoleScopesByRoleId(tx, roleId)
	}
	return tx.Where("role_id = ? AND menu_id NOT IN ?", roleId, menuIds).Delete(&model.SysRoleDataScope{}).Error
}

func (s *DataPermissionService) DeleteScopesByMenuIds(tx *gorm.DB, menuIds []int) error {
	if len(menuIds) == 0 {
		return nil
	}
	if tx == nil {
		tx = s.db
	}
	if err := tx.Where("menu_id IN ?", menuIds).Delete(&model.SysDataScopeBinding{}).Error; err != nil {
		return err
	}
	if err := tx.Where("menu_id IN ?", menuIds).Delete(&model.SysRoleDataScope{}).Error; err != nil {
		return err
	}
	return tx.Where("menu_id IN ?", menuIds).Delete(&model.SysUserDataScopeOverride{}).Error
}

func (s *DataPermissionService) GetUserOverrides(userId int) ([]model.SysUserDataScopeOverride, error) {
	var overrides []model.SysUserDataScopeOverride
	err := s.db.Preload("Menu").Preload("Dimension").Where("user_id = ?", userId).Order("menu_id ASC, id ASC").Find(&overrides).Error
	if err != nil {
		return nil, err
	}
	for i := range overrides {
		overrides[i].ScopeValueList = decodeStringList(overrides[i].ScopeValues)
	}
	return overrides, nil
}

func (s *DataPermissionService) SaveUserOverrides(ctx *gin.Context, userId int, req request.UserDataPermissionOverrideSaveReq) error {
	if req.UserId > 0 && req.UserId != userId {
		return myerrors.ErrParamInvalid
	}
	records := make([]model.SysUserDataScopeOverride, 0, len(req.Overrides))
	for _, item := range req.Overrides {
		record, err := s.userOverrideFromReq(userId, item)
		if err != nil {
			return err
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		record.Id = int(id)
		records = append(records, record)
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userId).Delete(&model.SysUserDataScopeOverride{}).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		return tx.Create(&records).Error
	})
}

func (s *DataPermissionService) DimensionOptions(code string) ([]DataPermissionOption, error) {
	code, err := normalizeDataPermissionCode("维度编码", code)
	if err != nil {
		return nil, err
	}
	var dimension model.SysDataDimension
	if err := s.db.Where("code = ? AND state = ?", code, true).First(&dimension).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []DataPermissionOption{}, nil
		}
		return nil, err
	}
	if dimension.SourceType != dataPermissionSourceTypeTable {
		return []DataPermissionOption{}, nil
	}
	if err := validateIdentifier("来源表", dimension.SourceCode); err != nil {
		return nil, err
	}
	if err := validateIdentifier("展示字段", dimension.LabelField); err != nil {
		return nil, err
	}
	if err := validateIdentifier("值字段", dimension.ValueField); err != nil {
		return nil, err
	}
	selects := []string{dimension.LabelField, dimension.ValueField}
	if strings.TrimSpace(dimension.ParentField) != "" {
		if err := validateIdentifier("父级字段", dimension.ParentField); err != nil {
			return nil, err
		}
		selects = append(selects, dimension.ParentField)
	}
	var rows []map[string]interface{}
	if err := s.db.Table(dimension.SourceCode).Select(strings.Join(selects, ",")).Limit(1000).Find(&rows).Error; err != nil {
		return nil, err
	}
	options := make([]DataPermissionOption, 0, len(rows))
	for _, row := range rows {
		option := DataPermissionOption{
			Label: fmt.Sprintf("%v", row[dimension.LabelField]),
			Value: fmt.Sprintf("%v", row[dimension.ValueField]),
		}
		if dimension.ParentField != "" && row[dimension.ParentField] != nil {
			option.Parent = fmt.Sprintf("%v", row[dimension.ParentField])
		}
		options = append(options, option)
	}
	return options, nil
}

func (s *DataPermissionService) ResolveDataScope(user model.SysUser, menuId int, table model.SysTable, action enum.SysMenuButtonEventAction) (*request.DataScope, error) {
	if menuId <= 0 || table.Id == 0 {
		return &request.DataScope{AllowAll: true}, nil
	}
	bindings, err := s.activeBindingsFor(menuId, table.TableCode, action)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return &request.DataScope{AllowAll: true}, nil
	}
	roleIds, err := s.userRoleIds(user.Id)
	if err != nil {
		return nil, err
	}
	if len(roleIds) == 0 {
		return &request.DataScope{DenyAll: true}, nil
	}
	conditions := make([]request.DataScopeCondition, 0, len(bindings))
	for _, binding := range bindings {
		if !utils.HasTableField(table, binding.FieldCode) {
			return &request.DataScope{DenyAll: true}, nil
		}
		dimension, err := s.activeDimension(binding.DimensionCode)
		if err != nil {
			return nil, err
		}
		if dimension.Code == "" {
			return &request.DataScope{DenyAll: true}, nil
		}
		decision, err := s.resolveBindingDecision(user, roleIds, binding)
		if err != nil {
			return nil, err
		}
		if decision.deny {
			return &request.DataScope{DenyAll: true}, nil
		}
		if decision.all {
			continue
		}
		values := sortedStringSet(decision.values)
		if len(values) == 0 {
			if binding.Required {
				return &request.DataScope{DenyAll: true}, nil
			}
			continue
		}
		conditions = append(conditions, request.DataScopeCondition{
			DimensionCode: binding.DimensionCode,
			Field:         binding.FieldCode,
			MatchType:     normalizeMatchTypeValue(binding.MatchType),
			ValueType:     dimension.ValueType,
			Values:        values,
		})
	}
	if len(conditions) == 0 {
		return &request.DataScope{AllowAll: true}, nil
	}
	return &request.DataScope{Conditions: conditions}, nil
}

// ResolveDataScopeForTableAction resolves data scope for fixed/list pages that may not
// post menu_id. If no active binding exists, data permission is not configured for
// the table/action and the query stays unrestricted.
func (s *DataPermissionService) ResolveDataScopeForTableAction(user model.SysUser, menuId int, table model.SysTable, action enum.SysMenuButtonEventAction) (*request.DataScope, error) {
	if menuId > 0 {
		return s.ResolveDataScope(user, menuId, table, action)
	}
	menuIds, err := s.activeBoundMenuIdsFor(table.TableCode, action)
	if err != nil {
		return nil, err
	}
	if len(menuIds) == 0 {
		return &request.DataScope{AllowAll: true}, nil
	}
	if len(menuIds) > 1 {
		return &request.DataScope{DenyAll: true}, nil
	}
	return s.ResolveDataScope(user, menuIds[0], table, action)
}

func (s *DataPermissionService) dimensionFromCreateReq(req request.DataPermissionDimensionCreateReq) (model.SysDataDimension, error) {
	code, err := normalizeDataPermissionCode("维度编码", req.Code)
	if err != nil {
		return model.SysDataDimension{}, err
	}
	valueType, err := normalizeValueType(req.ValueType)
	if err != nil {
		return model.SysDataDimension{}, err
	}
	sourceType, err := normalizeSourceType(req.SourceType)
	if err != nil {
		return model.SysDataDimension{}, err
	}
	if err := validateDimensionSource(sourceType, req.SourceCode, req.LabelField, req.ValueField, req.ParentField); err != nil {
		return model.SysDataDimension{}, err
	}
	state := true
	if req.State != nil {
		state = *req.State
	}
	return model.SysDataDimension{
		Basic:       model.Basic{State: state},
		Code:        code,
		Name:        strings.TrimSpace(req.Name),
		ValueType:   valueType,
		SourceType:  sourceType,
		SourceCode:  strings.TrimSpace(req.SourceCode),
		LabelField:  strings.TrimSpace(req.LabelField),
		ValueField:  strings.TrimSpace(req.ValueField),
		ParentField: strings.TrimSpace(req.ParentField),
		Memo:        strings.TrimSpace(req.Memo),
	}, nil
}

func (s *DataPermissionService) dimensionFromUpdateReq(req request.DataPermissionDimensionUpdateReq) (model.SysDataDimension, error) {
	createReq := request.DataPermissionDimensionCreateReq{
		Code:        req.Code,
		Name:        req.Name,
		ValueType:   req.ValueType,
		SourceType:  req.SourceType,
		SourceCode:  req.SourceCode,
		LabelField:  req.LabelField,
		ValueField:  req.ValueField,
		ParentField: req.ParentField,
		Memo:        req.Memo,
		State:       req.State,
	}
	item, err := s.dimensionFromCreateReq(createReq)
	item.Id = req.Id
	return item, err
}

func (s *DataPermissionService) bindingFromReq(menu model.SysMenu, table model.SysTable, req request.DataPermissionBindingItemReq) (model.SysDataScopeBinding, error) {
	dimensionCode, err := normalizeDataPermissionCode("维度编码", req.DimensionCode)
	if err != nil {
		return model.SysDataScopeBinding{}, err
	}
	if _, err := s.requireActiveDimension(dimensionCode); err != nil {
		return model.SysDataScopeBinding{}, err
	}
	fieldCode := strings.TrimSpace(req.FieldCode)
	if err := validateIdentifier("字段编码", fieldCode); err != nil {
		return model.SysDataScopeBinding{}, err
	}
	if !utils.HasTableField(table, fieldCode) {
		return model.SysDataScopeBinding{}, myerrors.NewBadRequestError(fmt.Sprintf("字段%s不存在", fieldCode))
	}
	matchType, err := normalizeMatchType(req.MatchType)
	if err != nil {
		return model.SysDataScopeBinding{}, err
	}
	actions, err := normalizeActions(req.Actions)
	if err != nil {
		return model.SysDataScopeBinding{}, err
	}
	required := true
	if req.Required != nil {
		required = *req.Required
	}
	state := true
	if req.State != nil {
		state = *req.State
	}
	return model.SysDataScopeBinding{
		Basic:         model.Basic{State: state},
		MenuId:        menu.Id,
		TableCode:     table.TableCode,
		DimensionCode: dimensionCode,
		FieldCode:     fieldCode,
		MatchType:     matchType,
		Required:      required,
		Actions:       encodeStringList(actions),
	}, nil
}

func (s *DataPermissionService) roleScopeFromReq(roleId int, req request.RoleDataPermissionItemReq) (model.SysRoleDataScope, error) {
	binding, err := s.requireBinding(req.MenuId, req.TableCode, req.DimensionCode)
	if err != nil {
		return model.SysRoleDataScope{}, err
	}
	strategy, values, err := normalizeScopeStrategyValues(req.Strategy, req.ScopeValues, roleId)
	if err != nil {
		return model.SysRoleDataScope{}, err
	}
	state := true
	if req.State != nil {
		state = *req.State
	}
	return model.SysRoleDataScope{
		Basic:         model.Basic{State: state},
		RoleId:        roleId,
		MenuId:        binding.MenuId,
		TableCode:     binding.TableCode,
		DimensionCode: binding.DimensionCode,
		Strategy:      strategy,
		ScopeValues:   encodeStringList(values),
	}, nil
}

func (s *DataPermissionService) userOverrideFromReq(userId int, req request.UserDataPermissionOverrideItemReq) (model.SysUserDataScopeOverride, error) {
	binding, err := s.requireBinding(req.MenuId, req.TableCode, req.DimensionCode)
	if err != nil {
		return model.SysUserDataScopeOverride{}, err
	}
	strategy, values, err := normalizeScopeStrategyValues(req.Strategy, req.ScopeValues, userId)
	if err != nil {
		return model.SysUserDataScopeOverride{}, err
	}
	overrideMode, err := normalizeOverrideMode(req.OverrideMode)
	if err != nil {
		return model.SysUserDataScopeOverride{}, err
	}
	var expireAt *model.CustomTime
	if strings.TrimSpace(req.ExpireAt) != "" {
		t, err := time.ParseInLocation(time.DateTime, strings.TrimSpace(req.ExpireAt), time.Local)
		if err != nil {
			return model.SysUserDataScopeOverride{}, myerrors.NewBadRequestError("过期时间格式应为 YYYY-MM-DD HH:mm:ss")
		}
		value := model.CustomTime(t)
		expireAt = &value
	}
	state := true
	if req.State != nil {
		state = *req.State
	}
	return model.SysUserDataScopeOverride{
		Basic:         model.Basic{State: state},
		UserId:        userId,
		MenuId:        binding.MenuId,
		TableCode:     binding.TableCode,
		DimensionCode: binding.DimensionCode,
		Strategy:      strategy,
		ScopeValues:   encodeStringList(values),
		OverrideMode:  overrideMode,
		ExpireAt:      expireAt,
	}, nil
}

func (s *DataPermissionService) boundMenuTable(menuId int) (model.SysMenu, model.SysTable, error) {
	var menu model.SysMenu
	if err := s.db.First(&menu, menuId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysMenu{}, model.SysTable{}, myerrors.ErrDataNotFound
		}
		return model.SysMenu{}, model.SysTable{}, err
	}
	if strings.TrimSpace(menu.TableCode) == "" || menu.IsHidden || !menu.State {
		return model.SysMenu{}, model.SysTable{}, myerrors.NewBadRequestError("只能为已绑定数据表的可用菜单配置数据权限")
	}
	var table model.SysTable
	err := s.db.Preload("TableFields").Where("table_code = ?", menu.TableCode).First(&table).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysMenu{}, model.SysTable{}, myerrors.NewBadRequestError("菜单绑定的数据表不存在")
	}
	return menu, table, err
}

func (s *DataPermissionService) activeBoundMenuIdsFor(tableCode string, action enum.SysMenuButtonEventAction) ([]int, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return nil, nil
	}
	var bindings []model.SysDataScopeBinding
	if err := s.db.Preload("Menu").
		Where("table_code = ? AND state = ?", tableCode, true).
		Order("menu_id ASC, id ASC").
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	menuIDSet := map[int]struct{}{}
	menuIDs := make([]int, 0)
	for _, binding := range bindings {
		if binding.Menu.Id == 0 || binding.Menu.IsHidden || !binding.Menu.State {
			continue
		}
		binding.ActionList = decodeStringList(binding.Actions)
		if !actionApplies(binding.ActionList, action) {
			continue
		}
		if _, exists := menuIDSet[binding.MenuId]; exists {
			continue
		}
		menuIDSet[binding.MenuId] = struct{}{}
		menuIDs = append(menuIDs, binding.MenuId)
	}
	return menuIDs, nil
}

func (s *DataPermissionService) requireActiveDimension(code string) (model.SysDataDimension, error) {
	dimension, err := s.activeDimension(code)
	if err != nil {
		return model.SysDataDimension{}, err
	}
	if dimension.Code == "" {
		return model.SysDataDimension{}, myerrors.NewBadRequestError(fmt.Sprintf("维度%s不存在或已停用", code))
	}
	return dimension, nil
}

func (s *DataPermissionService) activeDimension(code string) (model.SysDataDimension, error) {
	var dimension model.SysDataDimension
	err := s.db.Where("code = ? AND state = ?", code, true).First(&dimension).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysDataDimension{}, nil
	}
	return dimension, err
}

func (s *DataPermissionService) requireBinding(menuId int, tableCode, dimensionCode string) (model.SysDataScopeBinding, error) {
	dimensionCode, err := normalizeDataPermissionCode("维度编码", dimensionCode)
	if err != nil {
		return model.SysDataScopeBinding{}, err
	}
	query := s.db.Where("menu_id = ? AND dimension_code = ? AND state = ?", menuId, dimensionCode, true)
	if strings.TrimSpace(tableCode) != "" {
		query = query.Where("table_code = ?", strings.TrimSpace(tableCode))
	}
	var binding model.SysDataScopeBinding
	err = query.First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SysDataScopeBinding{}, myerrors.NewBadRequestError("菜单未绑定该数据权限维度")
	}
	return binding, err
}

func (s *DataPermissionService) activeBindingsFor(menuId int, tableCode string, action enum.SysMenuButtonEventAction) ([]model.SysDataScopeBinding, error) {
	var bindings []model.SysDataScopeBinding
	err := s.db.
		Where("menu_id = ? AND table_code = ? AND state = ?", menuId, tableCode, true).
		Order("id ASC").
		Find(&bindings).Error
	if err != nil {
		return nil, err
	}
	result := make([]model.SysDataScopeBinding, 0, len(bindings))
	for _, binding := range bindings {
		binding.ActionList = decodeStringList(binding.Actions)
		if actionApplies(binding.ActionList, action) {
			result = append(result, binding)
		}
	}
	return result, nil
}

func (s *DataPermissionService) userRoleIds(userId int) ([]int, error) {
	var rows []model.SysUserRole
	if err := s.db.Where("user_id = ?", userId).Find(&rows).Error; err != nil {
		return nil, err
	}
	roleIds := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.RoleId > 0 {
			roleIds = append(roleIds, row.RoleId)
		}
	}
	return roleIds, nil
}

func (s *DataPermissionService) resolveBindingDecision(user model.SysUser, roleIds []int, binding model.SysDataScopeBinding) (scopeDecision, error) {
	decision := scopeDecision{values: map[string]struct{}{}}
	var roleScopes []model.SysRoleDataScope
	err := s.db.
		Where("role_id IN ? AND menu_id = ? AND table_code = ? AND dimension_code = ? AND state = ?", roleIds, binding.MenuId, binding.TableCode, binding.DimensionCode, true).
		Find(&roleScopes).Error
	if err != nil {
		return decision, err
	}
	if len(roleScopes) == 0 {
		if binding.Required {
			decision.deny = true
		} else {
			decision.all = true
		}
		return s.applyUserOverrides(user, binding, decision)
	}
	hasSpecified := false
	for _, scope := range roleScopes {
		next := strategyDecision(scope.Strategy, decodeStringList(scope.ScopeValues), user.Id)
		if next.deny {
			continue
		}
		if next.all {
			decision.all = true
			decision.values = map[string]struct{}{}
			return s.applyUserOverrides(user, binding, decision)
		}
		hasSpecified = true
		for value := range next.values {
			decision.values[value] = struct{}{}
		}
	}
	if !hasSpecified && len(decision.values) == 0 {
		decision.deny = true
	}
	return s.applyUserOverrides(user, binding, decision)
}

func (s *DataPermissionService) applyUserOverrides(user model.SysUser, binding model.SysDataScopeBinding, base scopeDecision) (scopeDecision, error) {
	var overrides []model.SysUserDataScopeOverride
	err := s.db.
		Where("user_id = ? AND menu_id = ? AND table_code = ? AND dimension_code = ? AND state = ?", user.Id, binding.MenuId, binding.TableCode, binding.DimensionCode, true).
		Find(&overrides).Error
	if err != nil {
		return base, err
	}
	now := time.Now()
	for _, override := range overrides {
		if override.ExpireAt != nil && time.Time(*override.ExpireAt).Before(now) {
			continue
		}
		overrideDecision := strategyDecision(override.Strategy, decodeStringList(override.ScopeValues), user.Id)
		switch normalizeOverrideModeValue(override.OverrideMode) {
		case dataPermissionOverrideDeny:
			return scopeDecision{deny: true, values: map[string]struct{}{}}, nil
		case dataPermissionOverrideUnion:
			base = unionScopeDecision(base, overrideDecision)
		case dataPermissionOverrideIntersect:
			base = intersectScopeDecision(base, overrideDecision)
		default:
			base = overrideDecision
		}
	}
	return base, nil
}

func strategyDecision(strategy string, values []string, userId int) scopeDecision {
	decision := scopeDecision{values: map[string]struct{}{}}
	switch normalizeStrategyValue(strategy) {
	case dataPermissionStrategyAll:
		decision.all = true
	case dataPermissionStrategySelf:
		decision.values[strconv.Itoa(userId)] = struct{}{}
	case dataPermissionStrategySpecified, dataPermissionStrategyTree:
		for _, value := range normalizeStringValues(values) {
			decision.values[value] = struct{}{}
		}
	default:
		decision.deny = true
	}
	return decision
}

func unionScopeDecision(left, right scopeDecision) scopeDecision {
	if left.deny {
		return right
	}
	if right.deny {
		return left
	}
	if left.all || right.all {
		return scopeDecision{all: true, values: map[string]struct{}{}}
	}
	result := scopeDecision{values: map[string]struct{}{}}
	for value := range left.values {
		result.values[value] = struct{}{}
	}
	for value := range right.values {
		result.values[value] = struct{}{}
	}
	return result
}

func intersectScopeDecision(left, right scopeDecision) scopeDecision {
	if left.deny || right.deny {
		return scopeDecision{deny: true, values: map[string]struct{}{}}
	}
	if left.all {
		return right
	}
	if right.all {
		return left
	}
	result := scopeDecision{values: map[string]struct{}{}}
	for value := range left.values {
		if _, exists := right.values[value]; exists {
			result.values[value] = struct{}{}
		}
	}
	if len(result.values) == 0 {
		result.deny = true
	}
	return result
}

func normalizeDataPermissionCode(label, raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" || !dataPermissionIdentifierPattern.MatchString(value) {
		return "", myerrors.NewBadRequestError(label + "格式不正确")
	}
	return value, nil
}

func validateIdentifier(label, raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" || !dataPermissionIdentifierPattern.MatchString(value) {
		return myerrors.NewBadRequestError(label + "格式不正确")
	}
	return nil
}

func normalizeValueType(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return dataPermissionValueTypeString, nil
	}
	switch value {
	case dataPermissionValueTypeString, dataPermissionValueTypeNumber:
		return value, nil
	default:
		return "", myerrors.NewBadRequestError("维度值类型仅支持 string 或 number")
	}
}

func normalizeSourceType(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return dataPermissionSourceTypeNone, nil
	}
	switch value {
	case dataPermissionSourceTypeNone, dataPermissionSourceTypeTable:
		return value, nil
	default:
		return "", myerrors.NewBadRequestError("维度来源类型仅支持 none 或 table")
	}
}

func normalizeMatchType(raw string) (string, error) {
	value := normalizeMatchTypeValue(raw)
	switch value {
	case dataPermissionMatchIn, dataPermissionMatchEq:
		return value, nil
	default:
		return "", myerrors.NewBadRequestError("数据权限匹配方式仅支持 in 或 eq")
	}
}

func normalizeMatchTypeValue(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return dataPermissionMatchIn
	}
	return value
}

func normalizeStrategyValue(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return dataPermissionStrategyNone
	}
	return value
}

func normalizeScopeStrategyValues(rawStrategy string, rawValues []string, subjectId int) (string, []string, error) {
	strategy := normalizeStrategyValue(rawStrategy)
	values := normalizeStringValues(rawValues)
	switch strategy {
	case dataPermissionStrategyAll, dataPermissionStrategyNone:
		return strategy, []string{}, nil
	case dataPermissionStrategySelf:
		if len(values) == 0 {
			values = []string{strconv.Itoa(subjectId)}
		}
		return strategy, values, nil
	case dataPermissionStrategySpecified, dataPermissionStrategyTree:
		if len(values) == 0 {
			return "", nil, myerrors.NewBadRequestError("指定范围策略必须填写范围值")
		}
		return strategy, values, nil
	default:
		return "", nil, myerrors.NewBadRequestError("数据权限策略不合法")
	}
}

func normalizeOverrideMode(raw string) (string, error) {
	value := normalizeOverrideModeValue(raw)
	switch value {
	case dataPermissionOverrideReplace, dataPermissionOverrideUnion, dataPermissionOverrideIntersect, dataPermissionOverrideDeny:
		return value, nil
	default:
		return "", myerrors.NewBadRequestError("用户覆盖模式不合法")
	}
}

func normalizeOverrideModeValue(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return dataPermissionOverrideReplace
	}
	return value
}

func validateDimensionSource(sourceType, sourceCode, labelField, valueField, parentField string) error {
	if sourceType != dataPermissionSourceTypeTable {
		return nil
	}
	if err := validateIdentifier("来源表", sourceCode); err != nil {
		return err
	}
	if err := validateIdentifier("展示字段", labelField); err != nil {
		return err
	}
	if err := validateIdentifier("值字段", valueField); err != nil {
		return err
	}
	if strings.TrimSpace(parentField) != "" {
		return validateIdentifier("父级字段", parentField)
	}
	return nil
}

func normalizeActions(actions []string) ([]string, error) {
	seen := make(map[string]struct{}, len(actions))
	result := make([]string, 0, len(actions))
	for _, item := range actions {
		action, ok := enum.NormalizeSysMenuButtonEventAction(item)
		if !ok {
			return nil, myerrors.NewBadRequestError("数据权限动作不合法")
		}
		value := string(action)
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func actionApplies(actions []string, action enum.SysMenuButtonEventAction) bool {
	if len(actions) == 0 || strings.TrimSpace(string(action)) == "" {
		return true
	}
	for _, item := range actions {
		if item == string(action) {
			return true
		}
	}
	return false
}

func encodeStringList(values []string) string {
	values = normalizeStringValues(values)
	if len(values) == 0 {
		return "[]"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func decodeStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return normalizeStringValues(values)
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' '
	})
	return normalizeStringValues(parts)
}

func normalizeStringValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
