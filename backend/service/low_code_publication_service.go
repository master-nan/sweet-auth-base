package service

import (
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"

	"gorm.io/gorm"
)

const lowCodeCrudButtonTemplateScene = "lowcode_crud"

// LowCodePublicationService publishes configured metadata into the platform menu
// and permission projection. Schema lifecycle remains owned by SysTableService.
type LowCodePublicationService struct {
	sysTableRepo          repository.SysTableRepository
	sysMenuRepo           repository.SysMenuRepository
	sysMenuButtonRepo     repository.SysMenuButtonRepository
	sysMenuButtonTplRepo  repository.SysMenuButtonTemplateRepository
	sysRoleRepo           repository.SysRoleRepository
	sysRoleMenuRepo       repository.SysRoleMenuRepository
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository
	sf                    *utils.Snowflake
	metadataRuntime       *MetadataRuntimeService
}

func NewLowCodePublicationService(
	sysTableRepo repository.SysTableRepository,
	sysMenuRepo repository.SysMenuRepository,
	sysMenuButtonRepo repository.SysMenuButtonRepository,
	sysMenuButtonTplRepo repository.SysMenuButtonTemplateRepository,
	sysRoleRepo repository.SysRoleRepository,
	sysRoleMenuRepo repository.SysRoleMenuRepository,
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository,
	sf *utils.Snowflake,
	metadataRuntime *MetadataRuntimeService,
) *LowCodePublicationService {
	return &LowCodePublicationService{
		sysTableRepo:          sysTableRepo,
		sysMenuRepo:           sysMenuRepo,
		sysMenuButtonRepo:     sysMenuButtonRepo,
		sysMenuButtonTplRepo:  sysMenuButtonTplRepo,
		sysRoleRepo:           sysRoleRepo,
		sysRoleMenuRepo:       sysRoleMenuRepo,
		sysRoleMenuButtonRepo: sysRoleMenuButtonRepo,
		sf:                    sf,
		metadataRuntime:       metadataRuntime,
	}
}

func (s *LowCodePublicationService) PublishTableAsMenu(ctx context.Context, tableCode string, parentID int) error {
	table, err := s.metadataRuntime.configTableByCode(ctx, tableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrTableNotFound
	}
	return RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if err := s.ensureTableCanPublishLowCode(tx, table); err != nil {
			return err
		}
		targetParentID, err := s.resolvePublishParentMenu(tx, parentID)
		if err != nil {
			return err
		}
		menuID, err := s.ensureLowCodeMenu(tx, table, targetParentID)
		if err != nil {
			return err
		}
		if err := s.hideDuplicateLowCodeMenus(tx, table.TableCode, menuID); err != nil {
			return err
		}
		buttonIDs, err := s.ensureDefaultCrudButtons(tx, table.TableCode, menuID)
		if err != nil {
			return err
		}
		return s.ensureSuperAdminMenuPermissions(tx, menuID, buttonIDs)
	})
}

func (s *LowCodePublicationService) UnpublishTableMenu(ctx context.Context, tableCode string) error {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return myerrors.ErrParamInvalid
	}
	table, err := s.metadataRuntime.configTableByCode(ctx, tableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrTableNotFound
	}
	return RunInTransaction(ctx, s.sysTableRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		menus, err := s.findPublishedLowCodeMenus(tx, table.TableCode)
		if err != nil {
			return err
		}
		if len(menus) == 0 {
			return nil
		}
		menuIDs := make([]int, 0, len(menus))
		for _, menu := range menus {
			menuIDs = append(menuIDs, menu.Id)
		}
		if err := s.sysMenuRepo.HideMenusByIds(tx, menuIDs); err != nil {
			return err
		}
		if err := s.sysRoleMenuRepo.DeleteByMenuIds(tx, menuIDs); err != nil {
			return err
		}
		return s.sysRoleMenuButtonRepo.DeleteByMenuIds(tx, menuIDs)
	})
}

func (s *LowCodePublicationService) ensureTableCanPublishLowCode(tx *gorm.DB, table model.SysTable) error {
	menus, err := s.sysMenuRepo.FindFixedMenusByTableCode(tx, table.TableCode)
	if err != nil {
		return err
	}
	if len(menus) == 0 {
		return nil
	}
	menu := menus[0]
	title := strings.TrimSpace(menu.Title)
	if title == "" {
		title = menu.Name
	}
	return myerrors.NewValidationError(fmt.Sprintf("表 %s 已绑定固定菜单 %s，不能发布成低代码页面", table.TableCode, title))
}

func (s *LowCodePublicationService) resolvePublishParentMenu(tx *gorm.DB, parentID int) (int, error) {
	if parentID <= 0 {
		return s.ensureDevelopMenu(tx)
	}
	menu, err := s.sysMenuRepo.FindByIdWithDB(tx, parentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, myerrors.NewValidationError("发布目录不存在")
		}
		return 0, err
	}
	if menu.IsHidden || !menu.State {
		return 0, myerrors.NewValidationError("发布目录不可用")
	}
	if !isLowCodePublishParentMenu(menu) {
		return 0, myerrors.NewValidationError("低代码页面只能发布到目录菜单下")
	}
	return menu.Id, nil
}

func isLowCodePublishParentMenu(menu model.SysMenu) bool {
	if menu.IsHidden || !menu.State || strings.TrimSpace(menu.TableCode) != "" {
		return false
	}
	return menu.PageType == "" || menu.PageType == enum.MenuPageTypeDirectory
}

func (s *LowCodePublicationService) ensureDevelopMenu(tx *gorm.DB) (int, error) {
	menu, err := s.sysMenuRepo.FindByFieldWithDB(tx, "name", "develop")
	if err == nil {
		return menu.Id, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	menu = model.SysMenu{
		Basic:     model.Basic{Id: 300, State: true},
		Pid:       0,
		Name:      "develop",
		Path:      "develop",
		Component: "src/components/Layout/Layout.vue",
		Title:     "router.develop.default",
		Sequence:  3,
		PageType:  enum.MenuPageTypeDirectory,
		Icon:      utils.StringPtr("developer_mode"),
	}
	return menu.Id, s.sysMenuRepo.Create(tx, &menu)
}

func (s *LowCodePublicationService) ensureLowCodeMenu(tx *gorm.DB, table model.SysTable, parentID int) (int, error) {
	name := lowCodeMenuName(table.TableCode)
	menu, err := s.findPublishedLowCodeMenu(tx, table.TableCode)
	if err == nil {
		update := map[string]any{
			"pid":        parentID,
			"name":       name,
			"path":       "generalization/" + table.TableCode,
			"component":  "pages/develop/generalization/Index.vue",
			"page_type":  enum.MenuPageTypeLowCode,
			"table_code": table.TableCode,
			"option":     "",
			"is_hidden":  false,
			"state":      true,
			"gmt_delete": nil,
		}
		return menu.Id, s.sysMenuRepo.UpdateMenuFields(tx, menu.Id, update)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, err
	}
	menu = model.SysMenu{
		Basic:     model.Basic{Id: int(id), State: true},
		Pid:       parentID,
		Name:      name,
		Path:      "generalization/" + table.TableCode,
		Component: "pages/develop/generalization/Index.vue",
		Title:     table.TableName,
		Sequence:  uint8(30 + (table.Id % 100)),
		Icon:      utils.StringPtr("dynamic_form"),
		PageType:  enum.MenuPageTypeLowCode,
		TableCode: table.TableCode,
	}
	return menu.Id, s.sysMenuRepo.Create(tx, &menu)
}

func (s *LowCodePublicationService) hideDuplicateLowCodeMenus(tx *gorm.DB, tableCode string, keepMenuID int) error {
	menus, err := s.findPublishedLowCodeMenus(tx, tableCode)
	if err != nil {
		return err
	}
	duplicateIDs := make([]int, 0, len(menus))
	for _, menu := range menus {
		if menu.Id != keepMenuID {
			duplicateIDs = append(duplicateIDs, menu.Id)
		}
	}
	if len(duplicateIDs) == 0 {
		return nil
	}
	if err := s.sysMenuRepo.HideMenusByIds(tx, duplicateIDs); err != nil {
		return err
	}
	if err := s.sysRoleMenuRepo.DeleteByMenuIds(tx, duplicateIDs); err != nil {
		return err
	}
	return s.sysRoleMenuButtonRepo.DeleteByMenuIds(tx, duplicateIDs)
}

func (s *LowCodePublicationService) findPublishedLowCodeMenu(tx *gorm.DB, tableCode string) (model.SysMenu, error) {
	menus, err := s.findPublishedLowCodeMenus(tx, tableCode)
	if err != nil {
		return model.SysMenu{}, err
	}
	if len(menus) == 0 {
		return model.SysMenu{}, gorm.ErrRecordNotFound
	}
	return menus[0], nil
}

func (s *LowCodePublicationService) findPublishedLowCodeMenus(tx *gorm.DB, tableCode string) ([]model.SysMenu, error) {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return nil, nil
	}
	return s.sysMenuRepo.FindPublishedLowCodeMenus(tx, tableCode)
}

func lowCodeMenuName(tableCode string) string {
	raw := strings.TrimSpace(tableCode)
	name := "lowcode_" + raw
	if len(name) <= 32 {
		return name
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(raw))
	hash := fmt.Sprintf("%08x", hasher.Sum32())
	prefixLen := 32 - len("lowcode_") - len("_") - len(hash)
	if prefixLen < 1 {
		prefixLen = 1
	}
	prefix := raw
	if len(prefix) > prefixLen {
		prefix = prefix[:prefixLen]
	}
	return "lowcode_" + prefix + "_" + hash
}

func (s *LowCodePublicationService) ensureDefaultCrudButtons(tx *gorm.DB, tableCode string, menuID int) ([]int, error) {
	templates, err := s.sysMenuButtonTplRepo.FindEnabledBySceneWithDB(tx, lowCodeCrudButtonTemplateScene)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, myerrors.NewValidationError("低代码默认按钮模板未初始化")
	}
	defaults := lowCodeDefaultMenuButtons(tableCode, templates)
	buttonIDs := make([]int, 0, len(defaults))
	for _, item := range defaults {
		button, err := s.sysMenuButtonRepo.FindByMenuIdAndCode(tx, menuID, item.Code)
		if err == nil {
			updates := map[string]any{
				"name": item.Name, "memo": item.Memo, "position": item.Position,
				"event_type": item.EventType, "event_action": item.EventAction,
				"icon": item.Icon, "color": item.Color, "display_mode": item.DisplayMode,
				"sequence": item.Sequence, "path": item.Path, "method": strings.ToUpper(item.Method),
				"params_schema": item.ParamsSchema, "confirm_text": item.ConfirmText,
				"disable_when": item.DisableWhen, "before_hooks": item.BeforeHooks,
				"after_hooks": item.AfterHooks, "is_button": item.IsButton,
				"is_hidden": false, "is_disabled": item.IsDisabled, "state": true,
			}
			if err := s.sysMenuButtonRepo.UpdateMenuButtonFields(tx, button.Id, updates); err != nil {
				return nil, err
			}
			buttonIDs = append(buttonIDs, button.Id)
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return nil, err
		}
		item.Basic = model.Basic{Id: int(id), State: true}
		item.MenuId = menuID
		if err := s.sysMenuButtonRepo.Create(tx, &item); err != nil {
			return nil, err
		}
		buttonIDs = append(buttonIDs, item.Id)
	}
	return buttonIDs, nil
}

func lowCodeDefaultMenuButtons(tableCode string, templates []model.SysMenuButtonTemplate) []model.SysMenuButton {
	buttons := make([]model.SysMenuButton, 0, len(templates))
	for _, template := range templates {
		displayMode, ok := enum.NormalizeSysMenuButtonDisplayMode(string(template.DisplayMode))
		if !ok {
			displayMode = enum.ButtonDisplayAuto
		}
		buttons = append(buttons, model.SysMenuButton{
			Name: template.Name, Code: tableCode + template.CodeSuffix, Memo: template.Memo,
			Position: template.Position, EventType: template.EventType, EventAction: template.EventAction,
			Icon: template.Icon, Color: template.Color, DisplayMode: displayMode, Sequence: template.Sequence,
			Path: template.Path, Method: strings.ToUpper(template.Method), ParamsSchema: template.ParamsSchema,
			ConfirmText: template.ConfirmText, DisableWhen: template.DisableWhen,
			IsButton: template.IsButton, IsDisabled: template.IsDisabled,
			BeforeHooks: template.BeforeHooks, AfterHooks: template.AfterHooks,
		})
	}
	return buttons
}

func (s *LowCodePublicationService) ensureSuperAdminMenuPermissions(tx *gorm.DB, menuID int, buttonIDs []int) error {
	role, err := s.sysRoleRepo.FindByFieldWithDB(tx, "name", "super_admin")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if err := s.sysRoleMenuRepo.CreateIfNotExists(tx, model.SysRoleMenu{RoleId: role.Id, MenuId: menuID}); err != nil {
		return err
	}
	for _, buttonID := range buttonIDs {
		if err := s.sysRoleMenuButtonRepo.CreateIfNotExists(tx, model.SysRoleMenuButton{RoleId: role.Id, MenuId: menuID, ButtonId: buttonID}); err != nil {
			return err
		}
	}
	return nil
}
