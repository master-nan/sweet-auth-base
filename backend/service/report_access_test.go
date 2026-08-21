package service

import (
	"backend/dto/request"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"testing"
)

func TestPublishedReportAccessIsScopedToPermissionMenuAndCasbinSubject(t *testing.T) {
	env := newReportMenuTestEnv(t)
	role := model.SysRole{Basic: model.Basic{Id: 20, State: true}, Name: "report_runtime_viewer"}
	if err := env.db.Create(&role).Error; err != nil {
		t.Fatalf("seed runtime role: %v", err)
	}

	reportA := env.createPublishedMenuReport(t, "runtime_scope_a")
	publishedA, err := env.svc.PublishReportAsMenu(env.ctx, reportA.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700, Visible: reportMenuBoolPtr(true), PermissionRoleIds: []int{role.Id},
	})
	if err != nil {
		t.Fatalf("publish report A menu: %v", err)
	}
	reportB := env.createPublishedMenuReport(t, "runtime_scope_b")
	publishedB, err := env.svc.PublishReportAsMenu(env.ctx, reportB.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700, Visible: reportMenuBoolPtr(true),
	})
	if err != nil {
		t.Fatalf("publish report B menu: %v", err)
	}
	reportA = reloadReportDefinition(t, env, reportA.Id)
	reportB = reloadReportDefinition(t, env, reportB.Id)
	actor := model.SysUser{
		Basic: model.Basic{Id: 200, State: true}, UserName: "runtime-viewer",
		Roles: []model.SysRole{role},
	}
	env.ctx.Set("user", actor)

	list, err := env.svc.GetAuthorizedReportDefinitionList(context.Background(), actor, &request.Basic{Page: 1, Num: 20}, reportDefinitionListTestTable(env))
	if err != nil {
		t.Fatalf("list authorized reports: %v", err)
	}
	if list.Total != 1 || len(list.Data) != 1 || list.Data[0].Id != reportA.Id {
		t.Fatalf("object-scoped list = %+v total=%d, want report A only", list.Data, list.Total)
	}
	if err := env.svc.AuthorizeReportDetail(context.Background(), actor, reportA, publishedA.MenuId); err != nil {
		t.Fatalf("detail report A: %v", err)
	}
	if err := env.svc.AuthorizeReportDetail(context.Background(), actor, reportB, publishedB.MenuId); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("detail report B error = %v, want permission denied", err)
	}
	if _, err := env.svc.RunReport(env.ctx, reportA.Id, request.ReportPreviewReq{MenuId: publishedA.MenuId}); err != nil {
		t.Fatalf("run report A: %v", err)
	}
	if _, err := env.svc.RunReport(env.ctx, reportB.Id, request.ReportPreviewReq{MenuId: publishedB.MenuId}); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("run report B error = %v, want permission denied", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, reportA.Id, request.ReportExportReq{MenuId: publishedA.MenuId}); err != nil {
		t.Fatalf("export report A: %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, reportB.Id, request.ReportExportReq{MenuId: publishedB.MenuId}); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("export report B error = %v, want permission denied", err)
	}

	if _, err := env.enforcer.RemovePolicy(role.Name, reportRunPath, "POST"); err != nil {
		t.Fatalf("remove runtime Casbin policy: %v", err)
	}
	if _, err := env.svc.RunReport(env.ctx, reportA.Id, request.ReportPreviewReq{MenuId: publishedA.MenuId}); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("run without current Subject Casbin policy = %v, want permission denied", err)
	}
}

func TestPublishedReportDetailSupportsHistoricalRunPolicy(t *testing.T) {
	env := newReportMenuTestEnv(t)
	role := model.SysRole{Basic: model.Basic{Id: 21, State: true}, Name: "historical_report_viewer"}
	if err := env.db.Create(&role).Error; err != nil {
		t.Fatalf("seed historical runtime role: %v", err)
	}
	report := env.createPublishedMenuReport(t, "historical_runtime_detail")
	published, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
		ParentMenuId: 700, Visible: reportMenuBoolPtr(true), PermissionRoleIds: []int{role.Id},
	})
	if err != nil {
		t.Fatalf("publish historical fixture: %v", err)
	}
	report = reloadReportDefinition(t, env, report.Id)
	var detailButton model.SysMenuButton
	if err := env.db.Where("menu_id = ? AND path = ? AND method = ?", published.MenuId, reportDetailPath, "GET").First(&detailButton).Error; err != nil {
		t.Fatalf("load detail button: %v", err)
	}
	if err := env.db.Where("role_id = ? AND menu_id = ? AND button_id = ?", role.Id, published.MenuId, detailButton.Id).
		Delete(&model.SysRoleMenuButton{}).Error; err != nil {
		t.Fatalf("remove detail relation: %v", err)
	}
	if _, err := env.enforcer.RemovePolicy(role.Name, reportDetailPath, "GET"); err != nil {
		t.Fatalf("remove detail policy: %v", err)
	}
	actor := model.SysUser{Basic: model.Basic{Id: 201, State: true}, UserName: "historical-viewer", Roles: []model.SysRole{role}}
	if err := env.svc.AuthorizeReportDetail(context.Background(), actor, report, published.MenuId); err != nil {
		t.Fatalf("historical run policy should authorize runtime detail: %v", err)
	}
}

func TestReportMenuSubjectGrantsLoadsCasbinPolicyOncePerRole(t *testing.T) {
	env := newReportMenuTestEnv(t)
	role := model.SysRole{Basic: model.Basic{Id: 22, State: true}, Name: "multi_report_viewer"}
	if err := env.db.Create(&role).Error; err != nil {
		t.Fatalf("seed multi-report role: %v", err)
	}
	for _, code := range []string{"multi_report_a", "multi_report_b"} {
		report := env.createPublishedMenuReport(t, code)
		if _, err := env.svc.PublishReportAsMenu(env.ctx, report.Id, request.ReportPublishMenuReq{
			ParentMenuId: 700, Visible: reportMenuBoolPtr(true), PermissionRoleIds: []int{role.Id},
		}); err != nil {
			t.Fatalf("publish %s menu: %v", code, err)
		}
	}
	counter := &countingReportCasbinRepository{CasbinRuleRepository: env.svc.casbinRuleRepo}
	env.svc.casbinRuleRepo = counter
	actor := model.SysUser{Basic: model.Basic{Id: 203, State: true}, Roles: []model.SysRole{role}}

	menuIDs, err := env.svc.currentReportSubjectMenuIDsForRoute(context.Background(), actor, reportRoutePermission{Path: reportRunPath, Method: "POST"})
	if err != nil {
		t.Fatalf("resolve report menus: %v", err)
	}
	if len(menuIDs) != 2 {
		t.Fatalf("authorized menu ids = %v, want two menus", menuIDs)
	}
	if counter.filteredPolicyCalls != 1 {
		t.Fatalf("GetFilteredPolicy calls = %d, want once for one role", counter.filteredPolicyCalls)
	}
}

func TestReportManagementMenuKeepsExistingObjectAccess(t *testing.T) {
	env := newReportMenuTestEnv(t)
	role := seedReportManagementAccess(t, env)
	actor := model.SysUser{Basic: model.Basic{Id: 202, State: true}, UserName: "report-manager", Roles: []model.SysRole{role}}
	env.ctx.Set("user", actor)
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	draft := env.createReport(t, "management_draft", reportStatusDraft, queryConfig, layoutConfig)
	published := env.createPublishedMenuReport(t, "management_published")

	list, err := env.svc.GetAuthorizedReportDefinitionList(context.Background(), actor, &request.Basic{Page: 1, Num: 20}, reportDefinitionListTestTable(env))
	if err != nil {
		t.Fatalf("management list: %v", err)
	}
	if list.Total < 2 || !reportListContains(list.Data, draft.Id) || !reportListContains(list.Data, published.Id) {
		t.Fatalf("management list should preserve draft and published visibility: %+v", list.Data)
	}
	if err := env.svc.AuthorizeReportDetail(context.Background(), actor, draft, 0); err != nil {
		t.Fatalf("management detail for draft: %v", err)
	}
	if _, err := env.svc.RunReport(env.ctx, published.Id, request.ReportPreviewReq{}); err != nil {
		t.Fatalf("management run without report menu assignment: %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, published.Id, request.ReportExportReq{}); err != nil {
		t.Fatalf("management export without report menu assignment: %v", err)
	}
}

func seedReportManagementAccess(t *testing.T, env *reportMenuTestEnv) model.SysRole {
	t.Helper()
	role := model.SysRole{Basic: model.Basic{Id: 30, State: true}, Name: "report_manager"}
	menu := model.SysMenu{
		Basic: model.Basic{Id: 800, State: true}, Name: "report_manage", Path: "report/manage",
		Component: "pages/report/manage/Index.vue", PageType: enum.MenuPageTypeFixed,
	}
	if err := env.db.Create(&role).Error; err != nil {
		t.Fatalf("seed management role: %v", err)
	}
	if err := env.db.Create(&menu).Error; err != nil {
		t.Fatalf("seed management menu: %v", err)
	}
	if err := env.db.Create(&model.SysRoleMenu{RoleId: role.Id, MenuId: menu.Id}).Error; err != nil {
		t.Fatalf("seed management role menu: %v", err)
	}
	routes := []reportRoutePermission{
		{Path: reportListPath, Method: "POST"},
		{Path: reportDetailPath, Method: "GET"},
		{Path: reportRunPath, Method: "POST"},
		{Path: reportExportPath, Method: "POST"},
	}
	for index, route := range routes {
		button := model.SysMenuButton{
			Basic: model.Basic{Id: 810 + index, State: true}, MenuId: menu.Id,
			Name: route.Path, Code: "management_" + route.Method + route.Path, Path: route.Path, Method: route.Method,
		}
		if err := env.db.Create(&button).Error; err != nil {
			t.Fatalf("seed management button: %v", err)
		}
		if err := env.db.Create(&model.SysRoleMenuButton{RoleId: role.Id, MenuId: menu.Id, ButtonId: button.Id}).Error; err != nil {
			t.Fatalf("seed management button relation: %v", err)
		}
		if _, err := env.enforcer.AddPolicy(role.Name, route.Path, route.Method); err != nil {
			t.Fatalf("seed management Casbin policy: %v", err)
		}
	}
	return role
}

func reportDefinitionListTestTable(env *reportMenuTestEnv) model.SysTable {
	return model.SysTable{
		TableCode: env.db.NamingStrategy.TableName("ReportDefinition"),
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsListShow: true},
			{FieldCode: "permission_menu_id", FieldType: enum.BigIntFieldType, IsListShow: true},
			{FieldCode: "status", FieldType: enum.VarcharFieldType, IsListShow: true},
		},
	}
}

func reloadReportDefinition(t *testing.T, env *reportMenuTestEnv, reportID int) model.ReportDefinition {
	t.Helper()
	var report model.ReportDefinition
	if err := env.db.First(&report, reportID).Error; err != nil {
		t.Fatalf("reload report %d: %v", reportID, err)
	}
	return report
}

func reportListContains(reports []model.ReportDefinition, reportID int) bool {
	for _, report := range reports {
		if report.Id == reportID {
			return true
		}
	}
	return false
}

type countingReportCasbinRepository struct {
	repository.CasbinRuleRepository
	filteredPolicyCalls int
}

func (r *countingReportCasbinRepository) GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error) {
	r.filteredPolicyCalls++
	return r.CasbinRuleRepository.GetFilteredPolicy(fieldIndex, fieldValues...)
}
