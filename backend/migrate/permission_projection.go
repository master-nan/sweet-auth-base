package main

import (
	"backend/enum"
	"backend/internal/utils"
	"backend/model"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

type functionalPermissionSpec struct {
	id       int
	name     string
	code     string
	suffix   string
	action   string
	icon     string
	sequence uint8
	path     string
	method   string
}

var filePermissionSpecs = []functionalPermissionSpec{
	{id: 500, name: "文件上传", code: "platform_file_upload", suffix: "_file_upload", action: "create", icon: "upload_file", sequence: 110, path: "/admin/file/upload", method: "POST"},
	{id: 501, name: "文件详情", code: "platform_file_detail", suffix: "_file_detail", action: "detail", icon: "description", sequence: 111, path: "/admin/file/:id", method: "GET"},
	{id: 502, name: "文件删除", code: "platform_file_delete", suffix: "_file_delete", action: "delete", icon: "delete", sequence: 112, path: "/admin/file/:id", method: "DELETE"},
	{id: 503, name: "文件预览地址", code: "platform_file_preview_url", suffix: "_file_preview_url", action: "detail", icon: "preview", sequence: 113, path: "/admin/file/preview-url/:uuid", method: "GET"},
	{id: 504, name: "文件下载地址", code: "platform_file_download_url", suffix: "_file_download_url", action: "detail", icon: "download", sequence: 114, path: "/admin/file/download-url/:uuid", method: "GET"},
	{id: 505, name: "文件预览", code: "platform_file_preview", suffix: "_file_preview", action: "detail", icon: "visibility", sequence: 115, path: "/admin/file/preview/:uuid", method: "GET"},
	{id: 506, name: "文件下载", code: "platform_file_download", suffix: "_file_download", action: "detail", icon: "download", sequence: 116, path: "/admin/file/download/:uuid", method: "GET"},
	{id: 507, name: "初始化分片上传", code: "platform_file_upload_init", suffix: "_file_upload_init", action: "create", icon: "upload", sequence: 117, path: "/admin/file/upload/init", method: "POST"},
	{id: 508, name: "上传文件分片", code: "platform_file_upload_chunk", suffix: "_file_upload_chunk", action: "create", icon: "upload", sequence: 118, path: "/admin/file/upload/chunk", method: "POST"},
	{id: 509, name: "合并文件分片", code: "platform_file_upload_merge", suffix: "_file_upload_merge", action: "create", icon: "merge", sequence: 119, path: "/admin/file/upload/merge/:upload_id", method: "POST"},
	{id: 510, name: "查询上传进度", code: "platform_file_upload_progress", suffix: "_file_upload_progress", action: "query", icon: "progress_activity", sequence: 120, path: "/admin/file/upload/progress/:upload_id", method: "GET"},
}

type retiredPermissionSpec struct {
	menuName string
	code     string
}

var retiredOrganizationPermissions = []retiredPermissionSpec{
	{menuName: "organization_structure", code: "organization_unit_ancestors"},
	{menuName: "organization_structure", code: "organization_unit_descendants"},
	{menuName: "organization_sync_error", code: "organization_sync_error_retry"},
}

// seedFunctionalPermissionProjection 先修复权限元数据，再将角色 Policy 重建为角色按钮授权的纯投影。
func seedFunctionalPermissionProjection(db *gorm.DB, sf *utils.Snowflake) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var superAdmin model.SysRole
		if err := tx.Where("name = ?", "super_admin").First(&superAdmin).Error; err != nil {
			return fmt.Errorf("query super_admin role: %w", err)
		}

		if err := seedRoleAPIPermissions(tx, sf, superAdmin); err != nil {
			return err
		}
		if err := seedLowCodeSharedAPIPermissions(tx, sf, superAdmin); err != nil {
			return err
		}
		if err := seedLowCodeFilePermissionTemplates(tx, sf); err != nil {
			return err
		}
		if err := backfillLowCodeFilePermissions(tx, sf, superAdmin); err != nil {
			return err
		}
		if err := retireUnimplementedOrganizationPermissions(tx); err != nil {
			return err
		}
		if err := rebuildFunctionalPermissionPolicies(tx); err != nil {
			return err
		}
		return nil
	})
}

func seedRoleAPIPermissions(db *gorm.DB, sf *utils.Snowflake, role model.SysRole) error {
	menu, err := findPermissionMenu(db, "system_role")
	if err != nil {
		return err
	}
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(494, menu.Id, "角色菜单详情", "system_role_permission_menu_detail", enum.Line, "query_permission_menu", "account_tree", "primary", 96, "/admin/role/menu/:id", "GET"),
		apiPermissionWithAPI(495, menu.Id, "角色菜单按钮查询", "system_role_permission_button_query", enum.Line, "query_button", "smart_button", "primary", 97, "/admin/role/menu/buttons/:roleId/:menuId", "GET"),
	}
	return seedMenuButtons(db, sf, role.Id, role.Name, buttons)
}

func seedLowCodeSharedAPIPermissions(db *gorm.DB, sf *utils.Snowflake, role model.SysRole) error {
	menu, err := findPermissionMenu(db, "develop_generalization")
	if err != nil {
		return err
	}
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(511, menu.Id, "低代码批量删除", "develop_generalization_batch_delete", enum.Top, "batch_delete", "delete_sweep", "negative", 108, "/admin/generalization/batch-delete", "DELETE"),
		apiPermissionWithAPI(512, menu.Id, "低代码导出", "develop_generalization_export", enum.Top, "export", "download", "primary", 109, "/admin/generalization/export", "POST"),
	}
	for _, spec := range filePermissionSpecs {
		buttons = append(buttons, apiPermissionWithAPI(
			spec.id,
			menu.Id,
			spec.name,
			spec.code,
			enum.Top,
			spec.action,
			spec.icon,
			"primary",
			spec.sequence,
			spec.path,
			spec.method,
		))
	}
	return seedMenuButtons(db, sf, role.Id, role.Name, buttons)
}

func seedLowCodeFilePermissionTemplates(db *gorm.DB, sf *utils.Snowflake) error {
	for i, spec := range filePermissionSpecs {
		template := apiPermissionTemplate(
			608+i,
			spec.name,
			spec.suffix,
			enum.Top,
			spec.action,
			spec.icon,
			"primary",
			spec.sequence,
			spec.path,
			spec.method,
		)
		if err := seedLowCodeMenuButtonTemplate(db, sf, template); err != nil {
			return fmt.Errorf("seed low-code file permission template %s: %w", spec.suffix, err)
		}
	}
	return nil
}

func backfillLowCodeFilePermissions(db *gorm.DB, sf *utils.Snowflake, superAdmin model.SysRole) error {
	var menus []model.SysMenu
	if err := db.
		Where("page_type = ? AND state = ?", enum.MenuPageTypeLowCode, true).
		Order("id").
		Find(&menus).Error; err != nil {
		return fmt.Errorf("query published low-code menus: %w", err)
	}

	for _, menu := range menus {
		tableCode := strings.TrimSpace(menu.TableCode)
		if tableCode == "" {
			continue
		}
		for _, spec := range filePermissionSpecs {
			button := apiPermissionWithAPI(
				0,
				menu.Id,
				spec.name,
				tableCode+spec.suffix,
				enum.Top,
				spec.action,
				spec.icon,
				"primary",
				spec.sequence,
				spec.path,
				spec.method,
			)
			if err := seedMenuButton(db, sf, superAdmin.Id, superAdmin.Name, button); err != nil {
				return fmt.Errorf("seed file permission %s for menu %s: %w", spec.suffix, menu.Name, err)
			}
		}
	}
	return nil
}

func findPermissionMenu(db *gorm.DB, name string) (model.SysMenu, error) {
	var menu model.SysMenu
	if err := db.Where("name = ?", name).First(&menu).Error; err != nil {
		return model.SysMenu{}, fmt.Errorf("query permission menu %s: %w", name, err)
	}
	return menu, nil
}

func retireUnimplementedOrganizationPermissions(db *gorm.DB) error {
	for _, spec := range retiredOrganizationPermissions {
		menu, err := findPermissionMenu(db, spec.menuName)
		if err != nil {
			return err
		}
		var button model.SysMenuButton
		err = db.Where("menu_id = ? AND code = ?", menu.Id, spec.code).First(&button).Error
		if err == gorm.ErrRecordNotFound {
			continue
		}
		if err != nil {
			return fmt.Errorf("query retired organization permission %s: %w", spec.code, err)
		}
		if err := db.Where("menu_id = ? AND button_id = ?", menu.Id, button.Id).
			Delete(&model.SysRoleMenuButton{}).Error; err != nil {
			return fmt.Errorf("delete retired organization role permission %s: %w", spec.code, err)
		}
		if err := db.Model(&model.SysMenuButton{}).
			Where("id = ? AND menu_id = ?", button.Id, menu.Id).
			Updates(map[string]interface{}{
				"is_button":   false,
				"is_hidden":   true,
				"is_disabled": true,
				"state":       false,
				"gmt_modify":  model.Now(),
			}).Error; err != nil {
			return fmt.Errorf("retire unimplemented organization permission %s: %w", spec.code, err)
		}
	}
	return nil
}

func rebuildFunctionalPermissionPolicies(db *gorm.DB) error {
	var roles []model.SysRole
	if err := db.Order("id").Find(&roles).Error; err != nil {
		return fmt.Errorf("query roles for policy projection: %w", err)
	}
	if err := validateFunctionalPermissionSubjects(roles); err != nil {
		return err
	}
	if err := db.Where("ptype = ?", "p").Delete(&model.CasbinRule{}).Error; err != nil {
		return fmt.Errorf("clear functional permission policies: %w", err)
	}
	for _, role := range roles {
		if !role.State {
			continue
		}

		buttons, err := activeRolePermissionButtons(db, role.Id)
		if err != nil {
			return err
		}
		policies := projectedButtonPolicies(buttons)
		for _, policy := range policies {
			if err := db.Create(&model.CasbinRule{
				PType: "p",
				V0:    role.Name,
				V1:    policy.path,
				V2:    policy.method,
			}).Error; err != nil {
				return fmt.Errorf("create projected policy for role %s: %w", role.Name, err)
			}
		}
	}
	return nil
}

func validateFunctionalPermissionSubjects(roles []model.SysRole) error {
	seen := make(map[string]int, len(roles))
	for _, role := range roles {
		subject := strings.TrimSpace(role.Name)
		if subject == "" {
			return fmt.Errorf("role %d has empty functional permission subject", role.Id)
		}
		if subject != role.Name {
			return fmt.Errorf("role %d functional permission subject contains surrounding whitespace", role.Id)
		}
		if utf8.RuneCountInString(subject) > 100 {
			return fmt.Errorf("role %d functional permission subject exceeds Casbin v0 length", role.Id)
		}
		if existingRoleID, exists := seen[subject]; exists {
			return fmt.Errorf("roles %d and %d share functional permission subject %q", existingRoleID, role.Id, subject)
		}
		seen[subject] = role.Id
	}
	return nil
}

func activeRolePermissionButtons(db *gorm.DB, roleID int) ([]model.SysMenuButton, error) {
	var buttons []model.SysMenuButton
	if err := db.Model(&model.SysMenuButton{}).
		Select("sys_menu_button.*").
		Joins("JOIN sys_role_menu_button ON sys_role_menu_button.button_id = sys_menu_button.id AND sys_role_menu_button.menu_id = sys_menu_button.menu_id").
		Joins("JOIN sys_role_menu ON sys_role_menu.role_id = sys_role_menu_button.role_id AND sys_role_menu.menu_id = sys_role_menu_button.menu_id").
		Joins("JOIN sys_menu ON sys_menu.id = sys_role_menu_button.menu_id").
		Where("sys_role_menu_button.role_id = ?", roleID).
		Where("sys_menu_button.state = ? AND sys_menu_button.is_disabled = ?", true, false).
		Where("sys_menu.state = ? AND sys_menu.gmt_delete IS NULL", true).
		Order("sys_menu_button.id").
		Find(&buttons).Error; err != nil {
		return nil, fmt.Errorf("query active permission buttons for role %d: %w", roleID, err)
	}
	return buttons, nil
}

type projectedPermission struct {
	path   string
	method string
}

func projectedButtonPolicies(buttons []model.SysMenuButton) []projectedPermission {
	seen := make(map[string]struct{})
	result := make([]projectedPermission, 0, len(buttons))
	for _, button := range buttons {
		path := strings.TrimSpace(button.Path)
		method := strings.ToUpper(strings.TrimSpace(button.Method))
		action, validAction := enum.NormalizeSysMenuButtonEventAction(button.EventAction)
		if path == "" && method == "" && validAction && action == enum.ButtonActionDetail {
			path = "/admin/generalization/detail/code/:code/:id"
			method = "GET"
		}
		if path == "" || method == "" {
			continue
		}
		key := method + " " + path
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, projectedPermission{path: path, method: method})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].path == result[j].path {
			return result[i].method < result[j].method
		}
		return result[i].path < result[j].path
	})
	return result
}
