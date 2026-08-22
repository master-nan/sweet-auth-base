package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"context"
	"encoding/json"
	"strings"
)

const (
	reportListPath   = "/admin/report/query"
	reportDetailPath = "/admin/report/:id"
	reportRunPath    = "/admin/report/:id/run"
	reportExportPath = "/admin/report/:id/export"
)

var reportManagementMenuNames = []string{"report_manage", "report_v2_workbench"}

type reportRoutePermission struct {
	Path   string
	Method string
}

type reportMenuSubjectGrant struct {
	MenuID   int    `gorm:"column:menu_id"`
	RoleName string `gorm:"column:role_name"`
}

func reportDefinitionListResponse(data model.ReportDefinition) response.ReportDefinitionListRes {
	return response.ReportDefinitionListRes{
		BasicRes:            response.NewBasicRes(data.Basic),
		Code:                data.Code,
		Name:                data.Name,
		Description:         data.Description,
		Category:            data.Category,
		Status:              data.Status,
		PublishedVersionId:  data.PublishedVersionId,
		SourceType:          data.SourceType,
		SourceCode:          data.SourceCode,
		PermissionMenuId:    data.PermissionMenuId,
		PermissionTableCode: data.PermissionTableCode,
		Remark:              data.Remark,
	}
}

func reportDefinitionDetailResponse(data model.ReportDefinition) response.ReportDefinitionDetailRes {
	return response.ReportDefinitionDetailRes{
		ReportDefinitionListRes: reportDefinitionListResponse(data),
		QueryConfig:             json.RawMessage(data.QueryConfig),
		LayoutConfig:            json.RawMessage(data.LayoutConfig),
	}
}

func (s *ReportService) GetReportDefinitionListResponse(ctx context.Context, user model.SysUser, basic *request.Basic, table model.SysTable) (response.ListResult[response.ReportDefinitionListRes], error) {
	result, err := s.GetAuthorizedReportDefinitionList(ctx, user, basic, table)
	if err != nil {
		return response.ListResult[response.ReportDefinitionListRes]{}, err
	}
	items := make([]response.ReportDefinitionListRes, 0, len(result.Data))
	for _, item := range result.Data {
		items = append(items, reportDefinitionListResponse(item))
	}
	return response.ListResult[response.ReportDefinitionListRes]{Data: items, Total: result.Total}, nil
}

func (s *ReportService) GetReportDefinitionByIdResponse(ctx context.Context, user model.SysUser, id int, requestedMenuID int) (response.ReportDefinitionDetailRes, error) {
	data, err := s.GetReportDefinitionByIdWithContext(ctx, id)
	if err != nil {
		return response.ReportDefinitionDetailRes{}, err
	}
	if data.Id == 0 {
		return response.ReportDefinitionDetailRes{}, myerrors.ErrDataNotFound
	}
	if err := s.AuthorizeReportDetail(ctx, user, data, requestedMenuID); err != nil {
		return response.ReportDefinitionDetailRes{}, err
	}
	return reportDefinitionDetailResponse(data), nil
}

func (s *ReportService) GetAuthorizedReportDefinitionList(
	ctx context.Context,
	user model.SysUser,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.ReportDefinition], error) {
	managementAllowed, err := s.currentReportSubjectHasNamedMenuRoute(ctx, user, reportManagementMenuNames, reportRoutePermission{
		Path: reportListPath, Method: "POST",
	})
	if err != nil {
		return response.ListResult[model.ReportDefinition]{}, err
	}
	if managementAllowed {
		return s.GetReportDefinitionList(basic, table)
	}

	menuIDs, err := s.currentReportSubjectMenuIDsForRoute(ctx, user, reportRoutePermission{Path: reportRunPath, Method: "POST"})
	if err != nil {
		return response.ListResult[model.ReportDefinition]{}, err
	}
	if len(menuIDs) == 0 {
		return response.ListResult[model.ReportDefinition]{Data: []model.ReportDefinition{}, Total: 0}, nil
	}
	scoped := cloneReportListQuery(basic)
	scoped.Expressions = append(scoped.Expressions,
		request.ExpressionGroup{Logic: enum.And, Rules: []request.QueryRule{{
			Field: "permission_menu_id", ExpressionType: enum.In, Value: menuIDs,
		}}},
		request.ExpressionGroup{Logic: enum.And, Rules: []request.QueryRule{{
			Field: "status", ExpressionType: enum.Eq, Value: reportStatusPublished,
		}}},
	)
	return s.GetReportDefinitionList(&scoped, table)
}

func (s *ReportService) AuthorizeReportDetail(ctx context.Context, user model.SysUser, report model.ReportDefinition, requestedMenuID int) error {
	managementAllowed, err := s.currentReportSubjectHasNamedMenuRoute(ctx, user, reportManagementMenuNames, reportRoutePermission{
		Path: reportDetailPath, Method: "GET",
	})
	if err != nil {
		return err
	}
	if managementAllowed {
		return nil
	}
	if !report.State || normalizeReportStatus(report.Status) != reportStatusPublished {
		return myerrors.ErrPermissionDenied
	}
	return s.authorizePublishedReportObject(ctx, user, report.PermissionMenuId, requestedMenuID,
		reportRoutePermission{Path: reportDetailPath, Method: "GET"},
		// 部分现有Report菜单没有详情按钮投影；复用既有run能力可保持可用且不扩大访问范围。
		reportRoutePermission{Path: reportRunPath, Method: "POST"},
	)
}

func (s *ReportService) authorizePublishedReportRun(ctx context.Context, user model.SysUser, permissionMenuID, requestedMenuID int) error {
	return s.authorizePublishedReportAccess(ctx, user, permissionMenuID, requestedMenuID,
		reportRoutePermission{Path: reportRunPath, Method: "POST"})
}

func (s *ReportService) authorizePublishedReportExport(ctx context.Context, user model.SysUser, permissionMenuID, requestedMenuID int) error {
	return s.authorizePublishedReportAccess(ctx, user, permissionMenuID, requestedMenuID,
		reportRoutePermission{Path: reportExportPath, Method: "POST"})
}

func (s *ReportService) authorizePublishedReportAccess(
	ctx context.Context,
	user model.SysUser,
	permissionMenuID int,
	requestedMenuID int,
	route reportRoutePermission,
) error {
	managementAllowed, err := s.currentReportSubjectHasNamedMenuRoute(ctx, user, reportManagementMenuNames, route)
	if err != nil {
		return err
	}
	if managementAllowed {
		return nil
	}
	return s.authorizePublishedReportObject(ctx, user, permissionMenuID, requestedMenuID, route)
}

func (s *ReportService) authorizePublishedReportObject(
	ctx context.Context,
	user model.SysUser,
	permissionMenuID int,
	requestedMenuID int,
	routes ...reportRoutePermission,
) error {
	if permissionMenuID <= 0 || (requestedMenuID > 0 && requestedMenuID != permissionMenuID) {
		return myerrors.ErrPermissionDenied
	}
	for _, route := range routes {
		allowed, err := s.currentReportSubjectHasMenuRoute(ctx, user, permissionMenuID, route)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}
	}
	return myerrors.ErrPermissionDenied
}

func (s *ReportService) currentReportSubjectHasMenuRoute(
	ctx context.Context,
	user model.SysUser,
	menuID int,
	route reportRoutePermission,
) (bool, error) {
	if menuID <= 0 {
		return false, nil
	}
	grants, err := s.reportMenuSubjectGrants(ctx, user, []int{menuID}, nil, route)
	if err != nil {
		return false, err
	}
	return len(grants) > 0, nil
}

func (s *ReportService) currentReportSubjectHasNamedMenuRoute(
	ctx context.Context,
	user model.SysUser,
	menuNames []string,
	route reportRoutePermission,
) (bool, error) {
	if utils.IsSuperAdmin(user) {
		return true, nil
	}
	grants, err := s.reportMenuSubjectGrants(ctx, user, nil, menuNames, route)
	if err != nil {
		return false, err
	}
	return len(grants) > 0, nil
}

func (s *ReportService) currentReportSubjectMenuIDsForRoute(
	ctx context.Context,
	user model.SysUser,
	route reportRoutePermission,
) ([]int, error) {
	if utils.IsSuperAdmin(user) {
		var menuIDs []int
		err := s.reportRepo.DBWithContext(ctx).Model(&model.SysMenu{}).
			Where("state = ? AND page_type = ?", true, enum.MenuPageTypeReport).
			Pluck("id", &menuIDs).Error
		return uniquePositiveInts(menuIDs), err
	}
	grants, err := s.reportMenuSubjectGrants(ctx, user, nil, nil, route)
	if err != nil {
		return nil, err
	}
	menuIDs := make([]int, 0, len(grants))
	for _, grant := range grants {
		menuIDs = append(menuIDs, grant.MenuID)
	}
	return uniquePositiveInts(menuIDs), nil
}

func (s *ReportService) reportMenuSubjectGrants(
	ctx context.Context,
	user model.SysUser,
	menuIDs []int,
	menuNames []string,
	route reportRoutePermission,
) ([]reportMenuSubjectGrant, error) {
	roleIDs := reportCurrentRoleIDs(user)
	if len(roleIDs) == 0 || s.reportRepo == nil || s.casbinRuleRepo == nil {
		return nil, nil
	}
	db := s.reportRepo.DBWithContext(ctx)
	roleTable := db.NamingStrategy.TableName("SysRole")
	menuTable := db.NamingStrategy.TableName("SysMenu")
	buttonTable := db.NamingStrategy.TableName("SysMenuButton")
	roleMenuTable := db.NamingStrategy.TableName("SysRoleMenu")
	roleMenuButtonTable := db.NamingStrategy.TableName("SysRoleMenuButton")

	query := db.Table(roleTable+" AS r").
		Select("DISTINCT m.id AS menu_id, r.name AS role_name").
		Joins("JOIN "+roleMenuTable+" AS rm ON rm.role_id = r.id").
		Joins("JOIN "+menuTable+" AS m ON m.id = rm.menu_id").
		Joins("JOIN "+roleMenuButtonTable+" AS rmb ON rmb.role_id = r.id AND rmb.menu_id = m.id").
		Joins("JOIN "+buttonTable+" AS b ON b.id = rmb.button_id AND b.menu_id = m.id").
		Where("r.id IN ?", roleIDs).
		Where("r.gmt_delete IS NULL AND r.state = ?", true).
		Where("m.gmt_delete IS NULL AND m.state = ?", true).
		Where("b.gmt_delete IS NULL AND b.state = ? AND b.is_disabled = ?", true, false).
		Where("b.path = ? AND UPPER(b.method) = ?", strings.TrimSpace(route.Path), strings.ToUpper(strings.TrimSpace(route.Method)))
	if len(menuIDs) > 0 {
		query = query.Where("m.id IN ? AND m.page_type = ?", menuIDs, enum.MenuPageTypeReport)
	} else if len(menuNames) > 0 {
		query = query.Where("m.name IN ?", menuNames)
	} else {
		query = query.Where("m.page_type = ?", enum.MenuPageTypeReport)
	}
	var candidates []reportMenuSubjectGrant
	if err := query.Scan(&candidates).Error; err != nil {
		return nil, err
	}
	grants := make([]reportMenuSubjectGrant, 0, len(candidates))
	allowedByRole := make(map[string]bool, len(roleIDs))
	for _, candidate := range candidates {
		allowed, resolved := allowedByRole[candidate.RoleName]
		if !resolved {
			var err error
			allowed, err = s.reportCasbinSubjectAllows(candidate.RoleName, route)
			if err != nil {
				return nil, err
			}
			allowedByRole[candidate.RoleName] = allowed
		}
		if allowed {
			grants = append(grants, candidate)
		}
	}
	return grants, nil
}

func (s *ReportService) reportCasbinSubjectAllows(subject string, route reportRoutePermission) (bool, error) {
	policies, err := s.casbinRuleRepo.GetFilteredPolicy(0, strings.TrimSpace(subject))
	if err != nil {
		return false, err
	}
	path := strings.TrimSpace(route.Path)
	method := strings.ToUpper(strings.TrimSpace(route.Method))
	for _, policy := range policies {
		if len(policy) >= 3 && policy[1] == path && strings.ToUpper(policy[2]) == method {
			return true, nil
		}
	}
	return false, nil
}

func reportCurrentRoleIDs(user model.SysUser) []int {
	roleIDs := make([]int, 0, len(user.Roles))
	for _, role := range user.Roles {
		if role.Id > 0 && role.State {
			roleIDs = append(roleIDs, role.Id)
		}
	}
	return uniquePositiveInts(roleIDs)
}

func cloneReportListQuery(basic *request.Basic) request.Basic {
	if basic == nil {
		return request.Basic{}
	}
	cloned := *basic
	cloned.Expressions = append([]request.ExpressionGroup(nil), basic.Expressions...)
	if basic.Filters != nil {
		cloned.Filters = make(map[string]any, len(basic.Filters))
		for key, value := range basic.Filters {
			cloned.Filters[key] = value
		}
	}
	return cloned
}
