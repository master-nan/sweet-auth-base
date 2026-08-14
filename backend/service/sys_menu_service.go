/**
 * @Author: Nan
 * @Date: 2024/7/25 下午3:29
 */

package service

import (
	"backend/dto/request"
	"backend/enum"
	myerrors "backend/internal/errors"
	platformmetadata "backend/internal/metadata"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SysMenuService struct {
	sysMenuRepo           repository.SysMenuRepository
	sysRoleMenuRepo       repository.SysRoleMenuRepository
	sysRoleRepo           repository.SysRoleRepository
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository
	sysUserRoleRepo       repository.SysUserRoleRepository
	sysMenuButtonRepo     repository.SysMenuButtonRepository
	metadataRuntime       platformmetadata.RuntimeReader
	casbinRuleRepo        repository.CasbinRuleRepository
	sf                    *utils.Snowflake
}

func NewSysMenuService(sysMenuRepo repository.SysMenuRepository, sysRoleMenuRepo repository.SysRoleMenuRepository, sysRoleRepo repository.SysRoleRepository,
	sysRoleMenuButtons repository.SysRoleMenuButtonRepository,
	sysUserRoleRepo repository.SysUserRoleRepository, sysMenuButtonRepo repository.SysMenuButtonRepository,
	metadataRuntime platformmetadata.RuntimeReader, casbinRuleRepo repository.CasbinRuleRepository,
	sf *utils.Snowflake) *SysMenuService {
	return &SysMenuService{
		sysMenuRepo,
		sysRoleMenuRepo,
		sysRoleRepo,
		sysRoleMenuButtons,
		sysUserRoleRepo,
		sysMenuButtonRepo,
		metadataRuntime,
		casbinRuleRepo,
		sf,
	}
}

func (s *SysMenuService) GetMenuById(id int) (model.SysMenu, error) {
	result, err := s.sysMenuRepo.WithPreload("MenuButtons").FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysMenu{}, nil
		}
		return model.SysMenu{}, err
	}
	return result, nil
}

// CreateMenu 新增菜单
func (s *SysMenuService) CreateMenu(ctx context.Context, req request.MenuCreateReq) error {
	var data model.SysMenu
	err := copier.Copy(&data, &req)
	if err != nil {
		zap.L().Error("结构体字段映射失败", zap.String("target", "SysMenu"), zap.Error(err))
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	data.Id = int(id)
	normalizeMenuPageBinding(&data)
	return s.sysMenuRepo.Create(s.sysMenuRepo.DBWithContext(ctx), &data)
}

// UpdateMenu 更新菜单
func (s *SysMenuService) UpdateMenu(ctx context.Context, data request.MenuUpdateReq) error {
	if data.PageType == "" {
		existing, err := s.sysMenuRepo.WithContext(ctx).FindById(data.Id)
		if err == nil {
			data.PageType = existing.PageType
			data.TableCode = existing.TableCode
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if data.PageType == "" {
		data.PageType = enum.MenuPageTypeFixed
	}
	return s.sysMenuRepo.Update(s.sysMenuRepo.DBWithContext(ctx), &data, data.Id)
}

func normalizeMenuPageBinding(menu *model.SysMenu) {
	if menu.PageType == "" {
		menu.PageType = enum.MenuPageTypeFixed
	}
}

func (s *SysMenuService) UpdateMenuOrder(ctx context.Context, data request.MenuOrderUpdateReq) error {
	seen := make(map[int]struct{}, len(data.Menus))
	return RunInTransaction(ctx, s.sysMenuRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		for _, item := range data.Menus {
			if item.Id <= 0 {
				return fmt.Errorf("菜单ID不能为空")
			}
			if _, exists := seen[item.Id]; exists {
				return fmt.Errorf("菜单ID重复")
			}
			seen[item.Id] = struct{}{}
			updateReq := model.SysMenu{Sequence: item.Sequence}
			if err := s.sysMenuRepo.WithSelect("sequence").Update(tx, updateReq, item.Id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SysMenuService) RefreshMenuCache(ctx context.Context) error {
	return nil
}

// DeleteMenuById 删除菜单
func (s *SysMenuService) DeleteMenuById(ctx context.Context, id int) error {
	_, err := s.sysMenuRepo.WithContext(ctx).FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	buttons, err := s.sysMenuButtonRepo.FindListByField("menu_id", id)
	if err != nil {
		return err
	}
	candidates, err := s.collectButtonPolicyCandidates(buttons)
	if err != nil {
		return err
	}
	identities, err := s.policyIdentities(candidates)
	if err != nil {
		return err
	}
	snapshots, err := quiesceCasbinPolicies(s.casbinRuleRepo, identities)
	if err != nil {
		return err
	}
	var cleanups []rolePolicyCleanup
	err = RunInTransaction(ctx, s.sysMenuRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		childCount, err := s.sysMenuRepo.CountByField(tx, "pid", id)
		if err != nil {
			return err
		}
		if childCount > 0 {
			return myerrors.NewBadRequestError("请先删除子菜单")
		}
		menu, err := s.sysMenuRepo.FindByIdWithDB(tx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		buttons, err := s.sysMenuButtonRepo.FindListByFieldWithDB(tx, "menu_id", id)
		if err != nil {
			return err
		}
		candidates, err := s.collectButtonPolicyCandidates(buttons)
		if err != nil {
			return err
		}
		if err := s.sysRoleMenuButtonRepo.DeleteByField(tx, "menu_id", id); err != nil {
			return err
		}
		if err := s.sysRoleMenuRepo.DeleteByField(tx, "menu_id", id); err != nil {
			return err
		}
		if err := s.sysMenuButtonRepo.DeleteByField(tx, "menu_id", id); err != nil {
			return err
		}
		if err := s.sysMenuRepo.DeleteById(tx, menu.Id); err != nil {
			return err
		}
		cleanups, err = s.orphanRolePolicyCleanups(tx, candidates)
		return err
	})
	if err != nil {
		if restoreErr := restoreCasbinPolicies(s.casbinRuleRepo, snapshots, nil); restoreErr != nil {
			return fmt.Errorf("菜单删除失败且casbin恢复失败: %v: %w", restoreErr, err)
		}
		return err
	}
	return restoreCasbinPolicies(s.casbinRuleRepo, snapshots, cleanupPolicySet(cleanups))
}

// GetMenuTree 获取菜单列表并构建树结构
func (s *SysMenuService) GetMenuTree() ([]model.SysMenu, error) {
	menus, err := s.sysMenuRepo.GetMenus()
	if err != nil {
		return nil, err
	}
	return utils.BuildMenuTree(utils.SortMenuTree(menus), 0), nil
}

// GetUserMenus 获取用户菜单权限
func (s *SysMenuService) GetUserMenus(userId int) ([]model.SysMenu, error) {
	roles, err := s.sysUserRoleRepo.GetUserRoles(userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.SysMenu{}, nil
		}
		return nil, err
	}
	var roleIds []int
	for _, role := range roles {
		roleIds = append(roleIds, role.Id)
	}
	roleMenus, err := s.sysRoleMenuRepo.GetRoleMenusByRoleIds(roleIds)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.SysMenu{}, nil
		}
		return nil, err
	}
	menuIdMap := make(map[int]bool)
	for _, rm := range roleMenus {
		menuIdMap[rm.MenuId] = true
	}
	menuIds := make([]int, 0, len(menuIdMap))
	for id := range menuIdMap {
		menuIds = append(menuIds, id)
	}
	if len(menuIds) == 0 {
		return []model.SysMenu{}, nil
	}
	allMenus, err := s.sysMenuRepo.GetMenus()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.SysMenu{}, nil
		}
		return nil, err
	}
	menuMap := make(map[int]model.SysMenu)
	for _, menu := range allMenus {
		menuMap[menu.Id] = menu
	}
	completeMenuIdMap := make(map[int]bool)
	for _, menuId := range menuIds {
		completeMenuIdMap[menuId] = true
		// 向上查找所有父级菜单
		currentId := menuId
		for {
			menu, exists := menuMap[currentId]
			if !exists || menu.Pid == 0 {
				break // 不存在或已是顶级菜单
			}
			completeMenuIdMap[menu.Pid] = true // 添加父级
			currentId = menu.Pid               // 继续向上查找
		}
	}
	completeMenuIds := make([]int, 0, len(completeMenuIdMap))
	for id := range completeMenuIdMap {
		completeMenuIds = append(completeMenuIds, id)
	}
	// 获取按照角色找出来的菜单
	myMenus, err := s.sysMenuRepo.WithPreload("MenuButtons").FindListByFieldIn("id", completeMenuIds)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.SysMenu{}, nil
		}
		return nil, err
	}
	// 获取角色按钮
	roleMenuButtons, err := s.sysRoleMenuButtonRepo.FindListByFieldIn("role_id", roleIds)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			roleMenuButtons = []model.SysRoleMenuButton{}
		} else {
			return nil, err
		}
	}
	menuButtonMap := make(map[int]map[int]bool)
	for _, rmb := range roleMenuButtons {
		if _, exists := menuButtonMap[rmb.MenuId]; !exists {
			menuButtonMap[rmb.MenuId] = make(map[int]bool)
		}
		menuButtonMap[rmb.MenuId][rmb.ButtonId] = true
	}
	for i, menu := range myMenus {
		myMenus[i].MenuButtons = filterGrantedMenuButtons(menu.MenuButtons, menuButtonMap[menu.Id])
	}
	if err := s.attachMenuDetailOpenModes(myMenus); err != nil {
		return nil, err
	}
	return utils.BuildMenuTree(utils.SortMenuTree(myMenus), 0), nil
}

func (s *SysMenuService) attachMenuDetailOpenModes(menus []model.SysMenu) error {
	tableCodes := make([]string, 0, len(menus))
	seen := make(map[string]struct{}, len(menus))
	for _, menu := range menus {
		code := strings.TrimSpace(menu.TableCode)
		if code == "" {
			continue
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		tableCodes = append(tableCodes, code)
	}
	if len(tableCodes) == 0 {
		return nil
	}
	tables, err := s.metadataRuntime.ListTables(context.Background())
	if err != nil {
		return err
	}
	applyMenuDetailOpenModes(menus, tables, seen)
	return nil
}

func applyMenuDetailOpenModes(
	menus []model.SysMenu,
	tables []platformmetadata.TableMetadata,
	wanted map[string]struct{},
) {
	modeByTableCode := make(map[string]enum.SysDetailOpenMode, len(tables))
	for _, table := range tables {
		if len(wanted) > 0 {
			if _, ok := wanted[table.Code]; !ok {
				continue
			}
		}
		mode, ok := enum.NormalizeSysDetailOpenMode(string(table.DetailOpenMode))
		if !ok {
			mode = enum.DetailOpenAuto
		}
		modeByTableCode[table.Code] = mode
	}
	for index := range menus {
		if mode, exists := modeByTableCode[menus[index].TableCode]; exists {
			menus[index].DetailOpenMode = mode
		}
	}
}

func filterGrantedMenuButtons(buttons []model.SysMenuButton, granted map[int]bool) []model.SysMenuButton {
	if len(buttons) == 0 {
		return buttons
	}
	filteredButtons := make([]model.SysMenuButton, 0, len(buttons))
	for _, button := range buttons {
		if granted[button.Id] && button.State && !button.IsDisabled {
			filteredButtons = append(filteredButtons, button)
		}
	}
	return filteredButtons
}

func (s *SysMenuService) getUserRoleIds(userId int) ([]int, error) {
	roles, err := s.sysUserRoleRepo.GetUserRoles(userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []int{}, nil
		}
		return nil, err
	}
	roleIds := make([]int, 0, len(roles))
	for _, role := range roles {
		roleIds = append(roleIds, role.Id)
	}
	return roleIds, nil
}

func (s *SysMenuService) HasUserMenuPermission(userId, menuId int) (bool, error) {
	if userId <= 0 || menuId <= 0 {
		return false, nil
	}
	roleIds, err := s.getUserRoleIds(userId)
	if err != nil || len(roleIds) == 0 {
		return false, err
	}
	roleMenus, err := s.sysRoleMenuRepo.GetRoleMenusByRoleIds(roleIds)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	for _, roleMenu := range roleMenus {
		if roleMenu.MenuId == menuId {
			return true, nil
		}
	}
	return false, nil
}

func (s *SysMenuService) GetPublishedTableMenus(tableCode string) ([]model.SysMenu, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return []model.SysMenu{}, nil
	}
	menus, err := s.sysMenuRepo.FindPublishedLowCodeMenusByTableCode(nil, tableCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.SysMenu{}, nil
		}
		return nil, err
	}
	return menus, nil
}

func (s *SysMenuService) ResolvePublishedTableMenuId(userId int, tableCode string, action enum.SysMenuButtonEventAction) (int, bool, error) {
	tableCode = strings.TrimSpace(tableCode)
	if userId <= 0 || tableCode == "" {
		return 0, false, nil
	}
	menus, err := s.GetPublishedTableMenus(tableCode)
	if err != nil || len(menus) == 0 {
		return 0, false, err
	}
	hasPublishedMenu := false
	for _, menu := range menus {
		if menu.Id <= 0 || menu.IsHidden || !menu.State {
			continue
		}
		hasPublishedMenu = true
		hasMenu, err := s.HasUserMenuPermission(userId, menu.Id)
		if err != nil {
			return 0, true, err
		}
		if !hasMenu {
			continue
		}
		hasButton, err := s.HasUserMenuButtonAction(userId, menu.Id, string(action))
		if err != nil {
			return 0, true, err
		}
		if hasButton {
			return menu.Id, true, nil
		}
	}
	if hasPublishedMenu {
		return 0, true, myerrors.ErrPermissionDenied
	}
	return 0, len(menus) > 0, myerrors.ErrPermissionDenied
}

func (s *SysMenuService) HasUserMenuButtonActionByMenuName(userId int, menuName, action string) (bool, error) {
	menuName = strings.TrimSpace(menuName)
	if menuName == "" {
		return false, nil
	}
	menus, err := s.sysMenuRepo.GetMenus()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	for _, menu := range menus {
		if menu.Name != menuName {
			continue
		}
		hasMenu, err := s.HasUserMenuPermission(userId, menu.Id)
		if err != nil || !hasMenu {
			return false, err
		}
		return s.HasUserMenuButtonAction(userId, menu.Id, action)
	}
	return false, nil
}

func (s *SysMenuService) HasUserMenuButtonAction(userId, menuId int, action string) (bool, error) {
	if userId <= 0 || menuId <= 0 {
		return false, nil
	}
	targetAction, ok := enum.NormalizeSysMenuButtonEventAction(action)
	if !ok {
		return false, nil
	}
	roleIds, err := s.getUserRoleIds(userId)
	if err != nil || len(roleIds) == 0 {
		return false, err
	}
	roleMenuButtons, err := s.sysRoleMenuButtonRepo.FindListByFieldIn("role_id", roleIds)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	buttonIDMap := make(map[int]bool)
	buttonIDs := make([]int, 0)
	for _, roleMenuButton := range roleMenuButtons {
		if roleMenuButton.MenuId != menuId || buttonIDMap[roleMenuButton.ButtonId] {
			continue
		}
		buttonIDMap[roleMenuButton.ButtonId] = true
		buttonIDs = append(buttonIDs, roleMenuButton.ButtonId)
	}
	if len(buttonIDs) == 0 {
		return false, nil
	}
	buttons, err := s.sysMenuButtonRepo.FindListByFieldIn("id", buttonIDs)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	for _, button := range buttons {
		if menuButtonAllowsAction(button, menuId, targetAction) {
			return true, nil
		}
	}
	return false, nil
}

func menuButtonAllowsAction(button model.SysMenuButton, menuId int, targetAction enum.SysMenuButtonEventAction) bool {
	if button.MenuId != menuId || !button.State || button.IsDisabled {
		return false
	}
	buttonAction := strings.TrimSpace(button.EventAction)
	if buttonAction == "" {
		return false
	}
	return buttonAction == string(targetAction)
}

// GetMenuButtonsByMenuId 获取菜单按钮列表
func (s *SysMenuService) GetMenuButtonsByMenuId(menuId int) ([]model.SysMenuButton, error) {
	return s.sysMenuButtonRepo.FindListByField("menu_id", menuId)
}

// CreateMenuButton 创建菜单按钮
func (s *SysMenuService) CreateMenuButton(ctx context.Context, req request.MenuButtonCreateReq) error {
	var data model.SysMenuButton
	err := copier.Copy(&data, &req)
	if err != nil {
		return err
	}
	data.Path = req.ApiPath
	data.Method = req.HttpMethod
	if err := applyMenuButtonType(&data, req.IsButton, req.IsHidden); err != nil {
		return err
	}
	menu, err := s.GetMenuById(req.MenuId)
	if err != nil {
		return err
	}
	if menu.Id == 0 {
		return myerrors.NewBadRequestError("菜单不存在")
	}
	if err := normalizeAndValidateMenuButton(&data, menu); err != nil {
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	data.Id = int(id)
	return s.sysMenuButtonRepo.Create(s.sysMenuButtonRepo.DBWithContext(ctx), &data)
}

// UpdateMenuButton 更新菜单按钮
func (s *SysMenuService) UpdateMenuButton(ctx context.Context, req request.MenuButtonUpdateReq) error {
	data := model.SysMenuButton{}
	err := copier.Copy(&data, &req)
	if err != nil {
		return err
	}
	data.Path = req.ApiPath
	data.Method = req.HttpMethod
	if err := applyMenuButtonType(&data, req.IsButton, req.IsHidden); err != nil {
		return err
	}
	menu, err := s.GetMenuById(req.MenuId)
	if err != nil {
		return err
	}
	if menu.Id == 0 {
		return myerrors.NewBadRequestError("菜单不存在")
	}
	if err := normalizeAndValidateMenuButton(&data, menu); err != nil {
		return err
	}
	return s.sysMenuButtonRepo.Update(s.sysMenuButtonRepo.DBWithContext(ctx), menuButtonUpdateMap(data), req.Id)
}

func menuButtonUpdateMap(button model.SysMenuButton) map[string]any {
	return map[string]any{
		"menu_id":       button.MenuId,
		"name":          button.Name,
		"code":          button.Code,
		"memo":          button.Memo,
		"position":      button.Position,
		"event_type":    button.EventType,
		"event_action":  button.EventAction,
		"icon":          button.Icon,
		"color":         button.Color,
		"display_mode":  button.DisplayMode,
		"sequence":      button.Sequence,
		"path":          button.Path,
		"method":        strings.ToUpper(button.Method),
		"params_schema": button.ParamsSchema,
		"confirm_text":  button.ConfirmText,
		"disable_when":  button.DisableWhen,
		"is_button":     button.IsButton,
		"is_hidden":     button.IsHidden,
		"is_disabled":   button.IsDisabled,
		"before_hooks":  button.BeforeHooks,
		"after_hooks":   button.AfterHooks,
	}
}

func applyMenuButtonType(button *model.SysMenuButton, isButton *bool, isHidden bool) error {
	if isButton == nil {
		return myerrors.NewBadRequestError("is_button不能为空")
	}
	button.IsButton = *isButton
	button.IsHidden = isHidden
	return nil
}

const (
	maxButtonParamsSchemaBytes = 8 * 1024
	maxButtonDisableWhenBytes  = 4 * 1024
	maxButtonHooksBytes        = 2 * 1024
	maxButtonParamFields       = 50
	maxButtonOptions           = 200
	maxButtonDisableNodes      = 50
	maxButtonDisableDepth      = 6
	maxButtonHooks             = 20
)

var (
	adminAPIPathPattern       = regexp.MustCompile(`^/admin/[A-Za-z0-9_./:-]+$`)
	buttonParamFieldPattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)
	buttonHookNamePattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,63}$`)
	buttonDisableFieldPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*){0,5}$`)
)

func normalizeAndValidateMenuButton(button *model.SysMenuButton, menu model.SysMenu) error {
	button.Code = strings.TrimSpace(button.Code)
	button.EventAction = strings.TrimSpace(button.EventAction)
	button.Path = strings.TrimSpace(button.Path)
	button.Method = strings.ToUpper(strings.TrimSpace(button.Method))
	displayMode, ok := enum.NormalizeSysMenuButtonDisplayMode(string(button.DisplayMode))
	if !ok {
		return myerrors.NewBadRequestError("按钮展示方式不支持")
	}
	button.DisplayMode = displayMode
	button.ParamsSchema = strings.TrimSpace(button.ParamsSchema)
	button.DisableWhen = strings.TrimSpace(button.DisableWhen)
	button.BeforeHooks = strings.TrimSpace(button.BeforeHooks)
	button.AfterHooks = strings.TrimSpace(button.AfterHooks)
	if button.Code == "" {
		return myerrors.NewBadRequestError("按钮编码不能为空")
	}
	if _, ok := enum.NormalizeSysMenuButtonEventAction(button.EventAction); !ok {
		return myerrors.NewBadRequestError("按钮事件动作不支持")
	}
	if err := validateMenuButtonRuntimeConfig(button); err != nil {
		return err
	}
	if err := validateMenuButtonAPIConfig(button); err != nil {
		return err
	}
	return validateLowCodeMenuButtonConfig(button, menu)
}

func validateMenuButtonRuntimeConfig(button *model.SysMenuButton) error {
	if err := validateButtonParamsSchema(button.ParamsSchema); err != nil {
		return err
	}
	if err := validateButtonDisableWhen(button.DisableWhen); err != nil {
		return err
	}
	if err := normalizeButtonHooks("前置钩子", &button.BeforeHooks); err != nil {
		return err
	}
	return normalizeButtonHooks("后置钩子", &button.AfterHooks)
}

func validateButtonParamsSchema(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > maxButtonParamsSchemaBytes {
		return myerrors.NewBadRequestError("按钮参数Schema过大")
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return myerrors.NewBadRequestError("按钮参数Schema必须是合法JSON")
	}
	switch value := parsed.(type) {
	case []any:
		return validateButtonParamFieldList(value)
	case map[string]any:
		if fields, ok := value["fields"]; ok {
			fieldList, ok := fields.([]any)
			if !ok {
				return myerrors.NewBadRequestError("按钮参数Schema fields必须是数组")
			}
			return validateButtonParamFieldList(fieldList)
		}
		props, ok := value["properties"].(map[string]any)
		if !ok {
			return myerrors.NewBadRequestError("按钮参数Schema必须包含fields或properties")
		}
		if len(props) > maxButtonParamFields {
			return myerrors.NewBadRequestError("按钮参数Schema字段过多")
		}
		for key, prop := range props {
			if !buttonParamFieldPattern.MatchString(key) {
				return myerrors.NewBadRequestError("按钮参数Schema字段名格式不正确")
			}
			if err := validateButtonParamProperty(prop); err != nil {
				return err
			}
		}
		return nil
	default:
		return myerrors.NewBadRequestError("按钮参数Schema必须是对象或数组")
	}
}

func validateButtonParamFieldList(fields []any) error {
	if len(fields) > maxButtonParamFields {
		return myerrors.NewBadRequestError("按钮参数Schema字段过多")
	}
	for _, item := range fields {
		field, ok := item.(map[string]any)
		if !ok {
			return myerrors.NewBadRequestError("按钮参数Schema字段必须是对象")
		}
		code := firstStringValue(field, "field_code", "code", "name")
		if code == "" || !buttonParamFieldPattern.MatchString(code) {
			return myerrors.NewBadRequestError("按钮参数Schema字段名格式不正确")
		}
		if options, ok := field["options"].([]any); ok && len(options) > maxButtonOptions {
			return myerrors.NewBadRequestError("按钮参数Schema选项过多")
		}
	}
	return nil
}

func validateButtonParamProperty(prop any) error {
	propMap, ok := prop.(map[string]any)
	if !ok {
		return nil
	}
	if options, ok := propMap["enum"].([]any); ok && len(options) > maxButtonOptions {
		return myerrors.NewBadRequestError("按钮参数Schema选项过多")
	}
	return nil
}

func firstStringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key].(string)
		if ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func validateButtonDisableWhen(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > maxButtonDisableWhenBytes {
		return myerrors.NewBadRequestError("按钮禁用条件过大")
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return myerrors.NewBadRequestError("按钮禁用条件必须是合法JSON")
	}
	state := disableValidationState{}
	if err := validateButtonDisableNode(parsed, 1, &state); err != nil {
		return err
	}
	return nil
}

type disableValidationState struct {
	nodes int
}

func validateButtonDisableNode(node any, depth int, state *disableValidationState) error {
	if depth > maxButtonDisableDepth {
		return myerrors.NewBadRequestError("按钮禁用条件嵌套过深")
	}
	switch value := node.(type) {
	case []any:
		if len(value) == 0 {
			return myerrors.NewBadRequestError("按钮禁用条件数组不能为空")
		}
		for _, item := range value {
			if err := validateButtonDisableNode(item, depth+1, state); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		state.nodes++
		if state.nodes > maxButtonDisableNodes {
			return myerrors.NewBadRequestError("按钮禁用条件节点过多")
		}
		if all, ok := value["all"]; ok {
			return validateButtonDisableList(all, depth, state)
		}
		if anyValue, ok := value["any"]; ok {
			return validateButtonDisableList(anyValue, depth, state)
		}
		if notValue, ok := value["not"]; ok {
			return validateButtonDisableNode(notValue, depth+1, state)
		}
		if _, ok := value["field"]; ok {
			return validateButtonDisableRule(value)
		}
		return myerrors.NewBadRequestError("按钮禁用条件结构不正确")
	default:
		return myerrors.NewBadRequestError("按钮禁用条件必须是对象或数组")
	}
}

func validateButtonDisableList(value any, depth int, state *disableValidationState) error {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return myerrors.NewBadRequestError("按钮禁用条件all/any必须是非空数组")
	}
	return validateButtonDisableNode(items, depth+1, state)
}

func validateButtonDisableRule(rule map[string]any) error {
	field, ok := rule["field"].(string)
	if !ok || !buttonDisableFieldPattern.MatchString(strings.TrimSpace(field)) {
		return myerrors.NewBadRequestError("按钮禁用条件字段格式不正确")
	}
	op := "eq"
	if rawOp, ok := rule["op"].(string); ok && strings.TrimSpace(rawOp) != "" {
		op = strings.ToLower(strings.TrimSpace(rawOp))
	}
	switch op {
	case "eq", "ne", "gt", "gte", "lt", "lte", "in", "not_in", "includes", "not_includes", "empty", "not_empty", "truthy", "falsy":
	default:
		return myerrors.NewBadRequestError("按钮禁用条件操作符不支持")
	}
	if values, ok := rule["value"].([]any); ok && len(values) > maxButtonOptions {
		return myerrors.NewBadRequestError("按钮禁用条件选项过多")
	}
	return nil
}

func normalizeButtonHooks(label string, raw *string) error {
	value := strings.TrimSpace(*raw)
	if value == "" {
		*raw = ""
		return nil
	}
	if len(value) > maxButtonHooksBytes {
		return myerrors.NewBadRequestError(label + "配置过大")
	}
	var hooks []string
	if err := json.Unmarshal([]byte(value), &hooks); err != nil {
		return myerrors.NewBadRequestError(label + "必须是JSON字符串数组")
	}
	if len(hooks) > maxButtonHooks {
		return myerrors.NewBadRequestError(label + "数量过多")
	}
	normalized := make([]string, 0, len(hooks))
	seen := make(map[string]struct{}, len(hooks))
	for _, hook := range hooks {
		hook = strings.TrimSpace(hook)
		if hook == "" || !buttonHookNamePattern.MatchString(hook) {
			return myerrors.NewBadRequestError(label + "名称格式不正确")
		}
		if _, ok := seen[hook]; ok {
			continue
		}
		seen[hook] = struct{}{}
		normalized = append(normalized, hook)
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	*raw = string(data)
	return nil
}

func validateMenuButtonAPIConfig(button *model.SysMenuButton) error {
	if button.Path == "" && button.Method == "" {
		return nil
	}
	if strings.TrimSpace(button.EventAction) == string(enum.ButtonActionNavigate) && button.Method == "" {
		if strings.Contains(button.Path, "://") || strings.ContainsAny(button.Path, "\r\n\t") {
			return myerrors.NewBadRequestError("前端跳转路径格式不正确")
		}
		return nil
	}
	if button.Path == "" || button.Method == "" {
		return myerrors.NewBadRequestError("按钮API路径和请求方法必须同时配置")
	}
	switch button.Method {
	case "GET", "POST", "PUT", "DELETE":
	default:
		return myerrors.NewBadRequestError("按钮请求方法仅支持GET/POST/PUT/DELETE")
	}
	if !adminAPIPathPattern.MatchString(button.Path) ||
		strings.Contains(button.Path, "://") ||
		strings.ContainsAny(button.Path, "?#\r\n\t ") {
		return myerrors.NewBadRequestError("按钮API路径必须是/admin开头的后端路由")
	}
	return nil
}

func validateLowCodeMenuButtonConfig(button *model.SysMenuButton, menu model.SysMenu) error {
	if !isLowCodeMenu(menu) {
		return nil
	}
	switch strings.TrimSpace(button.EventAction) {
	case string(enum.ButtonActionQuery):
		return requireLowCodeButtonAPI(button, "POST", "/admin/generalization/query/code/:code")
	case string(enum.ButtonActionCreate):
		return requireLowCodeButtonAPI(button, "POST", "/admin/generalization/create")
	case string(enum.ButtonActionUpdate):
		return requireLowCodeButtonAPI(button, "PUT", "/admin/generalization/update")
	case string(enum.ButtonActionDelete):
		return requireLowCodeButtonAPI(button, "DELETE", "/admin/generalization/delete")
	case string(enum.ButtonActionRefresh):
		if button.Path != "" || button.Method != "" {
			return myerrors.NewBadRequestError("低代码刷新按钮不能配置后端API")
		}
	}
	return nil
}

func isLowCodeMenu(menu model.SysMenu) bool {
	return menu.PageType == enum.MenuPageTypeLowCode && strings.TrimSpace(menu.TableCode) != ""
}

func requireLowCodeButtonAPI(button *model.SysMenuButton, method, path string) error {
	if button.Method != method || button.Path != path {
		return myerrors.NewBadRequestError(fmt.Sprintf("低代码%s按钮API必须为%s %s", strings.TrimSpace(button.EventAction), method, path))
	}
	return nil
}

// DeleteMenuButton 删除菜单按钮
func (s *SysMenuService) DeleteMenuButton(ctx context.Context, id int) error {
	button, err := s.sysMenuButtonRepo.FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	candidates, err := s.collectButtonPolicyCandidates([]model.SysMenuButton{button})
	if err != nil {
		return err
	}
	identities, err := s.policyIdentities(candidates)
	if err != nil {
		return err
	}
	snapshots, err := quiesceCasbinPolicies(s.casbinRuleRepo, identities)
	if err != nil {
		return err
	}
	var cleanups []rolePolicyCleanup
	err = RunInTransaction(ctx, s.sysMenuButtonRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		button, err := s.sysMenuButtonRepo.FindById(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		candidates, err := s.collectButtonPolicyCandidates([]model.SysMenuButton{button})
		if err != nil {
			return err
		}
		if err := s.sysRoleMenuButtonRepo.DeleteByField(tx, "button_id", id); err != nil {
			return err
		}
		if err := s.sysMenuButtonRepo.DeleteById(tx, id); err != nil {
			return err
		}
		cleanups, err = s.orphanRolePolicyCleanups(tx, candidates)
		return err
	})
	if err != nil {
		if restoreErr := restoreCasbinPolicies(s.casbinRuleRepo, snapshots, nil); restoreErr != nil {
			return fmt.Errorf("按钮删除失败且casbin恢复失败: %v: %w", restoreErr, err)
		}
		return err
	}
	return restoreCasbinPolicies(s.casbinRuleRepo, snapshots, cleanupPolicySet(cleanups))
}

type buttonPolicyKey struct {
	RoleID int
	Path   string
	Method string
}

type rolePolicyCleanup struct {
	RoleName string
	Path     string
	Method   string
}

func (s *SysMenuService) policyIdentities(candidates map[buttonPolicyKey]struct{}) ([]casbinPolicyIdentity, error) {
	identities := make([]casbinPolicyIdentity, 0, len(candidates))
	for candidate := range candidates {
		role, err := s.sysRoleRepo.FindById(candidate.RoleID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		identities = append(identities, casbinPolicyIdentity{Subject: role.Name, Path: candidate.Path, Method: candidate.Method})
	}
	return identities, nil
}

func cleanupPolicySet(cleanups []rolePolicyCleanup) map[casbinPolicyIdentity]struct{} {
	result := make(map[casbinPolicyIdentity]struct{}, len(cleanups))
	for _, cleanup := range cleanups {
		result[casbinPolicyIdentity{Subject: cleanup.RoleName, Path: cleanup.Path, Method: cleanup.Method}] = struct{}{}
	}
	return result
}

func (s *SysMenuService) collectButtonPolicyCandidates(buttons []model.SysMenuButton) (map[buttonPolicyKey]struct{}, error) {
	candidates := make(map[buttonPolicyKey]struct{})
	for _, button := range buttons {
		path := strings.TrimSpace(button.Path)
		method := strings.ToUpper(strings.TrimSpace(button.Method))
		if path == "" || method == "" {
			continue
		}
		roleButtons, err := s.sysRoleMenuButtonRepo.FindListByField("button_id", button.Id)
		if err != nil {
			return nil, err
		}
		for _, roleButton := range roleButtons {
			candidates[buttonPolicyKey{RoleID: roleButton.RoleId, Path: path, Method: method}] = struct{}{}
		}
	}
	return candidates, nil
}

func (s *SysMenuService) orphanRolePolicyCleanups(tx *gorm.DB, candidates map[buttonPolicyKey]struct{}) ([]rolePolicyCleanup, error) {
	cleanups := make([]rolePolicyCleanup, 0, len(candidates))
	for candidate := range candidates {
		remaining, err := s.sysRoleMenuButtonRepo.CountActiveButtonPolicy(tx, candidate.RoleID, candidate.Path, candidate.Method)
		if err != nil {
			return nil, err
		}
		if remaining > 0 {
			continue
		}
		role, err := s.sysRoleRepo.FindById(candidate.RoleID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		cleanups = append(cleanups, rolePolicyCleanup{RoleName: role.Name, Path: candidate.Path, Method: candidate.Method})
	}
	return cleanups, nil
}
