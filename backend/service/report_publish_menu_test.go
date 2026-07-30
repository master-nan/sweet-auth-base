package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	"backend/model"
	"backend/repository/impl"
	"encoding/json"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
)

type reportMenuTestEnv struct {
	*reportV1ATestEnv
	enforcer *casbin.Enforcer
}

func newReportMenuTestEnv(t *testing.T) *reportMenuTestEnv {
	t.Helper()
	env := newReportV1ATestEnv(t, reportV1AUser(true))
	if err := env.db.AutoMigrate(
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.CasbinRule{},
	); err != nil {
		t.Fatalf("migrate report menu test schema: %v", err)
	}
	enforcer, err := casbin.NewEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("create casbin enforcer: %v", err)
	}
	primaryDB := &database.PrimaryDB{DB: env.db}
	env.svc.sysMenuRepo = impl.NewSysMenuRepositoryImpl(primaryDB)
	env.svc.sysMenuButtonRepo = impl.NewSysMenuButtonRepositoryImpl(primaryDB)
	env.svc.sysRoleRepo = impl.NewSysRoleRepositoryImpl(primaryDB)
	env.svc.sysRoleMenuRepo = impl.NewSysRoleMenuRepositoryImpl(primaryDB)
	env.svc.sysRoleMenuButtonRepo = impl.NewSysRoleMenuButtonRepositoryImpl(primaryDB)
	env.svc.casbinRuleRepo = impl.NewCasbinRuleRepositoryImpl(primaryDB, enforcer)
	seedReportMenuParent(t, env)
	seedReportSuperAdminRole(t, env)
	return &reportMenuTestEnv{reportV1ATestEnv: env, enforcer: enforcer}
}

func TestReportPublishMenuRejectsUnpublishedReport(t *testing.T) {
	env := newReportMenuTestEnv(t)
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	draft := env.createReport(t, "menu_draft_report", reportStatusDraft, queryConfig, layoutConfig)

	_, err := env.svc.PublishReportAsMenu(env.ctx, draft.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700,
		Visible:      reportMenuBoolPtr(true),
	})
	if err == nil || !strings.Contains(err.Error(), "未发布") {
		t.Fatalf("draft report should not publish to menu, got %v", err)
	}
}

func TestReportPublishMenuRejectsMissingPublishedVersion(t *testing.T) {
	env := newReportMenuTestEnv(t)
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "menu_missing_version", reportStatusPublished, queryConfig, layoutConfig)

	_, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700,
		Visible:      reportMenuBoolPtr(true),
	})
	if err == nil || !strings.Contains(err.Error(), "发布版本") {
		t.Fatalf("report without published version should not publish to menu, got %v", err)
	}
}

func TestReportPublishMenuRejectsInvalidParent(t *testing.T) {
	env := newReportMenuTestEnv(t)
	report := env.createPublishedMenuReport(t, "menu_invalid_parent")
	_, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 999,
		Visible:      reportMenuBoolPtr(true),
	})
	if err == nil || !strings.Contains(err.Error(), "父级菜单不存在") {
		t.Fatalf("missing parent should be rejected, got %v", err)
	}
	if err := env.db.Create(&model.SysMenu{
		Basic:     model.Basic{Id: 701, State: true},
		Name:      "fixed_page",
		Path:      "fixed/page",
		Component: "pages/fixed/Index.vue",
		Title:     "固定页面",
		PageType:  enum.MenuPageTypeFixed,
	}).Error; err != nil {
		t.Fatalf("seed fixed menu: %v", err)
	}

	_, err = env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 701,
		Visible:      reportMenuBoolPtr(true),
	})
	if err == nil || !strings.Contains(err.Error(), "目录菜单") {
		t.Fatalf("fixed parent should be rejected, got %v", err)
	}
}

func TestReportPublishMenuCreatesMenuButtonsPermissionsAndOption(t *testing.T) {
	env := newReportMenuTestEnv(t)
	report := env.createPublishedMenuReport(t, "supplier_statement")

	result, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700,
		Title:        "供应商月度对账单",
		Sort:         36,
		Visible:      reportMenuBoolPtr(true),
	})
	if err != nil {
		t.Fatalf("publish report menu: %v", err)
	}
	if result.MenuId == 0 || !result.PublishedToMenu {
		t.Fatalf("unexpected publish menu response: %+v", result)
	}

	var menu model.SysMenu
	if err := env.db.First(&menu, result.MenuId).Error; err != nil {
		t.Fatalf("query menu: %v", err)
	}
	if menu.PageType != enum.MenuPageTypeReport || menu.Component != reportRuntimeComponent {
		t.Fatalf("unexpected menu binding: pageType=%s component=%s", menu.PageType, menu.Component)
	}
	if menu.Path != "report/runtime/supplier_statement" || menu.Name != "report_supplier_statement" {
		t.Fatalf("unexpected route: path=%s name=%s", menu.Path, menu.Name)
	}
	if menu.TableCode != report.PermissionTableCode {
		t.Fatalf("menu table_code=%q want %q", menu.TableCode, report.PermissionTableCode)
	}
	var option reportMenuOption
	if err := json.Unmarshal([]byte(menu.Option), &option); err != nil {
		t.Fatalf("parse report menu option: %v", err)
	}
	if option.ReportId != report.Id || option.ReportCode != report.Code {
		t.Fatalf("unexpected option: %+v", option)
	}

	var storedReport model.ReportDefinition
	if err := env.db.First(&storedReport, report.Id).Error; err != nil {
		t.Fatalf("query report: %v", err)
	}
	if storedReport.PermissionMenuId != menu.Id {
		t.Fatalf("permission_menu_id=%d want %d", storedReport.PermissionMenuId, menu.Id)
	}

	buttons := env.reportMenuButtons(t, menu.Id)
	wantActions := map[string]bool{"query": false, "export": false, "refresh": false}
	for _, button := range buttons {
		if _, ok := wantActions[button.EventAction]; ok {
			wantActions[button.EventAction] = true
		}
		if strings.Contains(button.EventAction, "create") || strings.Contains(button.EventAction, "delete") || strings.Contains(button.EventAction, "update") {
			t.Fatalf("CRUD button should not be generated: %+v", button)
		}
	}
	for action, found := range wantActions {
		if !found {
			t.Fatalf("missing default report menu button action %s, buttons=%+v", action, buttons)
		}
	}

	var roleMenuCount int64
	if err := env.db.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", 1, menu.Id).Count(&roleMenuCount).Error; err != nil {
		t.Fatalf("count role menu: %v", err)
	}
	if roleMenuCount != 1 {
		t.Fatalf("super_admin menu permission count=%d", roleMenuCount)
	}
	if ok, err := env.enforcer.Enforce(reportSuperAdminRoleName, "/admin/report/:id/run", "POST"); err != nil || !ok {
		t.Fatalf("super_admin should have run policy, ok=%v err=%v", ok, err)
	}
	if ok, err := env.enforcer.Enforce(reportSuperAdminRoleName, "/admin/report/:id/export", "POST"); err != nil || !ok {
		t.Fatalf("super_admin should have export policy, ok=%v err=%v", ok, err)
	}
}

func TestReportPublishMenuRejectsPathConflict(t *testing.T) {
	env := newReportMenuTestEnv(t)
	report := env.createPublishedMenuReport(t, "path_conflict_report")
	if err := env.db.Create(&model.SysMenu{
		Basic:     model.Basic{Id: 702, State: true},
		Name:      "another_report",
		Path:      "report/runtime/path_conflict_report",
		Component: reportRuntimeComponent,
		Title:     "其他报表",
		PageType:  enum.MenuPageTypeReport,
	}).Error; err != nil {
		t.Fatalf("seed conflicting menu: %v", err)
	}

	_, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700,
		Visible:      reportMenuBoolPtr(true),
	})
	if err == nil || !strings.Contains(err.Error(), "路径已被占用") {
		t.Fatalf("path conflict should be rejected, got %v", err)
	}
}

func TestReportPublishMenuIsIdempotentAndUpdatesExistingMenu(t *testing.T) {
	env := newReportMenuTestEnv(t)
	report := env.createPublishedMenuReport(t, "repeat_publish_report")

	first, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700,
		Title:        "首次标题",
		Visible:      reportMenuBoolPtr(true),
	})
	if err != nil {
		t.Fatalf("first publish menu: %v", err)
	}
	second, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700,
		Title:        "更新标题",
		Path:         "report/runtime/repeat-publish-custom",
		Visible:      reportMenuBoolPtr(true),
	})
	if err != nil {
		t.Fatalf("second publish menu: %v", err)
	}
	if second.MenuId != first.MenuId {
		t.Fatalf("repeated publish should update existing menu, first=%d second=%d", first.MenuId, second.MenuId)
	}
	var count int64
	if err := env.db.Model(&model.SysMenu{}).
		Where("page_type = ? AND component = ?", enum.MenuPageTypeReport, reportRuntimeComponent).
		Count(&count).Error; err != nil {
		t.Fatalf("count report menus: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one report menu after repeated publish, got %d", count)
	}
	var menu model.SysMenu
	if err := env.db.First(&menu, first.MenuId).Error; err != nil {
		t.Fatalf("query menu: %v", err)
	}
	if menu.Title != "更新标题" || menu.Path != "report/runtime/repeat-publish-custom" {
		t.Fatalf("menu was not updated: title=%q path=%q", menu.Title, menu.Path)
	}
}

func TestReportUnpublishMenuHidesMenuAndCleansPermissions(t *testing.T) {
	env := newReportMenuTestEnv(t)
	report := env.createPublishedMenuReport(t, "unpublish_report")
	published, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700,
		Visible:      reportMenuBoolPtr(true),
	})
	if err != nil {
		t.Fatalf("publish report menu: %v", err)
	}

	result, err := env.svc.UnpublishReportMenu(env.ctx, report.Id)
	if err != nil {
		t.Fatalf("unpublish report menu: %v", err)
	}
	if result.PublishedToMenu || result.Visible {
		t.Fatalf("unexpected unpublish response: %+v", result)
	}
	var menu model.SysMenu
	if err := env.db.First(&menu, published.MenuId).Error; err != nil {
		t.Fatalf("query menu: %v", err)
	}
	if !menu.IsHidden || menu.State {
		t.Fatalf("menu should be hidden and disabled after unpublish: hidden=%v state=%v", menu.IsHidden, menu.State)
	}
	var storedReport model.ReportDefinition
	if err := env.db.First(&storedReport, report.Id).Error; err != nil {
		t.Fatalf("query report: %v", err)
	}
	if storedReport.PermissionMenuId != published.MenuId {
		t.Fatalf("unpublish should keep permission_menu_id=%d, got %d", published.MenuId, storedReport.PermissionMenuId)
	}
	var roleMenuCount int64
	if err := env.db.Model(&model.SysRoleMenu{}).Where("menu_id = ?", published.MenuId).Count(&roleMenuCount).Error; err != nil {
		t.Fatalf("count role menu: %v", err)
	}
	if roleMenuCount != 0 {
		t.Fatalf("role menu permission should be cleaned, got %d", roleMenuCount)
	}
	if ok, err := env.enforcer.Enforce(reportSuperAdminRoleName, "/admin/report/:id/run", "POST"); err != nil || ok {
		t.Fatalf("run policy should be removed after unpublish, ok=%v err=%v", ok, err)
	}
}

func (env *reportMenuTestEnv) createPublishedMenuReport(t *testing.T, code string) model.ReportDefinition {
	t.Helper()
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, code, reportStatusPublished, queryConfig, layoutConfig)
	version := env.createVersion(t, report, 1, reportVersionPublished, true, queryConfig, layoutConfig)
	if err := env.db.Model(&model.ReportDefinition{}).
		Where("id = ?", report.Id).
		Update("published_version_id", version.Id).Error; err != nil {
		t.Fatalf("set published version pointer: %v", err)
	}
	report.PublishedVersionId = version.Id
	return report
}

func (env *reportMenuTestEnv) reportMenuButtons(t *testing.T, menuID int) []model.SysMenuButton {
	t.Helper()
	var buttons []model.SysMenuButton
	if err := env.db.Where("menu_id = ?", menuID).Find(&buttons).Error; err != nil {
		t.Fatalf("query report menu buttons: %v", err)
	}
	return buttons
}

func seedReportMenuParent(t *testing.T, env *reportV1ATestEnv) {
	t.Helper()
	menu := model.SysMenu{
		Basic:     model.Basic{Id: 700, State: true},
		Pid:       0,
		Name:      "report",
		Path:      "report",
		Component: "src/components/Layout/Layout.vue",
		Title:     "报表",
		PageType:  enum.MenuPageTypeDirectory,
	}
	if err := env.db.Create(&menu).Error; err != nil {
		t.Fatalf("seed report menu parent: %v", err)
	}
}

func seedReportSuperAdminRole(t *testing.T, env *reportV1ATestEnv) {
	t.Helper()
	role := model.SysRole{
		Basic: model.Basic{Id: 1, State: true},
		Name:  reportSuperAdminRoleName,
		Memo:  "超级管理员",
	}
	if err := env.db.Create(&role).Error; err != nil {
		t.Fatalf("seed super admin role: %v", err)
	}
}

func reportMenuBoolPtr(value bool) *bool {
	return &value
}
