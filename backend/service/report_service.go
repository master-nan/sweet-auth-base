package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/audit"
	myerrors "backend/internal/errors"
	platformmetadata "backend/internal/metadata"
	"backend/internal/reportconfig"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	queryutil "backend/repository/util"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	reportSourceTypeTable      = "table"
	reportSourceTypeView       = "view"
	reportStatusDraft          = "draft"
	reportStatusPublished      = "published"
	reportStatusDisabled       = "disabled"
	reportVersionPublished     = "published"
	reportVersionArchived      = "archived"
	reportRuntimeDesignPreview = "design_preview"
	reportRuntimeRun           = "runtime_run"
	reportRuntimeExport        = "runtime_export"
	reportExportFormatCSV      = "csv"
	reportRuntimeComponent     = "pages/report/runtime/ReportRuntimePage.vue"
	reportDefaultMenuIcon      = "assessment"
	reportSuperAdminRoleName   = "super_admin"

	defaultReportExportMaxRows = 5000
	maxReportExportRows        = 10000
)

// ReportExecutionSnapshot 固定一次Preview、Run或Export使用的发布版本和非敏感运行事实。
type ReportExecutionSnapshot struct {
	ReportId            int
	VersionId           int
	VersionNo           int
	Code                string
	Name                string
	SourceType          string
	SourceCode          string
	PermissionMenuId    int
	PermissionTableCode string
	QueryConfig         datatypes.JSON
	LayoutConfig        datatypes.JSON
	RuntimeType         string
}

// ReportExecutionOptions 是服务端运行预算，不接受客户端直接覆盖。
type ReportExecutionOptions struct {
	MaxRows              int
	PageSizeLimit        int
	DefaultPageSize      int
	ExportMode           bool
	WriteLog             bool
	DataPermissionAction enum.SysMenuButtonEventAction
}

// ReportService 编排定义、发布版本、菜单绑定、运行、导出和执行日志；
// 配置结构由reportconfig校验，运行读取仍受菜单与Data Permission约束。
type ReportService struct {
	reportRepo            repository.ReportDefinitionRepository
	reportVersionRepo     repository.ReportDefinitionVersionRepository
	reportLogRepo         repository.ReportExecutionLogRepository
	sysMenuRepo           repository.SysMenuRepository
	sysMenuButtonRepo     repository.SysMenuButtonRepository
	sysRoleRepo           repository.SysRoleRepository
	sysRoleMenuRepo       repository.SysRoleMenuRepository
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository
	casbinRuleRepo        repository.CasbinRuleRepository
	generalizationService *GeneralizationService
	metadataRuntime       platformmetadata.RuntimeReader
	sf                    *utils.Snowflake
}

func NewReportService(
	reportRepo repository.ReportDefinitionRepository,
	reportVersionRepo repository.ReportDefinitionVersionRepository,
	reportLogRepo repository.ReportExecutionLogRepository,
	sysMenuRepo repository.SysMenuRepository,
	sysMenuButtonRepo repository.SysMenuButtonRepository,
	sysRoleRepo repository.SysRoleRepository,
	sysRoleMenuRepo repository.SysRoleMenuRepository,
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository,
	casbinRuleRepo repository.CasbinRuleRepository,
	generalizationService *GeneralizationService,
	metadataRuntime platformmetadata.RuntimeReader,
	sf *utils.Snowflake,
) *ReportService {
	return &ReportService{
		reportRepo:            reportRepo,
		reportVersionRepo:     reportVersionRepo,
		reportLogRepo:         reportLogRepo,
		sysMenuRepo:           sysMenuRepo,
		sysMenuButtonRepo:     sysMenuButtonRepo,
		sysRoleRepo:           sysRoleRepo,
		sysRoleMenuRepo:       sysRoleMenuRepo,
		sysRoleMenuButtonRepo: sysRoleMenuButtonRepo,
		casbinRuleRepo:        casbinRuleRepo,
		generalizationService: generalizationService,
		metadataRuntime:       metadataRuntime,
		sf:                    sf,
	}
}

func (s *ReportService) GetReportDefinitionList(basic *request.Basic, table model.SysTable) (response.ListResult[model.ReportDefinition], error) {
	return s.reportRepo.GetReportDefinitionList(basic, table)
}

func (s *ReportService) GetReportDefinitionById(id int) (model.ReportDefinition, error) {
	return s.GetReportDefinitionByIdWithContext(context.Background(), id)
}

func (s *ReportService) GetReportDefinitionByIdWithContext(ctx context.Context, id int) (model.ReportDefinition, error) {
	report, err := s.reportRepo.WithContext(ctx).FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ReportDefinition{}, nil
		}
		return model.ReportDefinition{}, err
	}
	return report, nil
}

func (s *ReportService) GetDataSources() ([]response.ReportDataSourceRes, error) {
	result, err := s.metadataRuntime.ListTables(context.Background())
	if err != nil {
		return nil, err
	}
	items := make([]response.ReportDataSourceRes, 0, len(result))
	for _, tableMetadata := range result {
		fullTable := tableMetadata.QueryModel()
		items = append(items, response.ReportDataSourceRes{
			Id:          fullTable.Id,
			Name:        fullTable.TableName,
			Code:        fullTable.TableCode,
			Type:        reportTableTypeLabel(fullTable.TableType),
			Description: fmt.Sprintf("%s (%s)", fullTable.TableName, fullTable.TableCode),
			Fields:      reportDataSourceColumns(fullTable),
		})
	}
	return items, nil
}

func (s *ReportService) ResolveRuntimeTable(ctx context.Context, tableCode string) (model.SysTable, error) {
	if s == nil || s.metadataRuntime == nil {
		return model.SysTable{}, myerrors.NewValidationError("报表元数据服务未初始化")
	}
	table, err := s.metadataRuntime.GetTable(ctx, tableCode)
	if errors.Is(err, myerrors.ErrDataNotFound) {
		return model.SysTable{}, nil
	}
	if err != nil {
		return model.SysTable{}, err
	}
	return table.QueryModel(), nil
}

func (s *ReportService) InferSQLFields(ctx *gin.Context, req request.ReportSQLFieldsReq) ([]response.ReportPreviewColumn, error) {
	executionCtx, cancel := withReportExecutionDeadline(reportRequestContext(ctx), reportDesignPreviewDeadline)
	defer cancel()
	if s.reportRepo == nil {
		return nil, myerrors.NewValidationError("报表数据仓储未初始化")
	}
	sqlText, err := safeReportPreviewSQL(req.SQL)
	if err != nil {
		return nil, err
	}
	wrapped := fmt.Sprintf("SELECT * FROM (%s) AS report_sql_dataset_fields LIMIT 0", sqlText)
	var columns []response.ReportPreviewColumn
	err = withReportExecutionTransaction(executionCtx, s.reportRepo.DBWithContext(executionCtx), func(tx *gorm.DB) error {
		sqlRows, queryErr := tx.Raw(wrapped).Rows()
		if queryErr != nil {
			return queryErr
		}
		defer sqlRows.Close()
		columns, queryErr = reportSQLColumns(sqlRows)
		return queryErr
	})
	return columns, normalizeReportExecutionError(err)
}

func (s *ReportService) CreateReportDefinition(ctx *gin.Context, req request.ReportDefinitionCreateReq) (int, error) {
	report, err := s.reportFromCreateReq(req)
	if err != nil {
		return 0, err
	}
	existing, err := s.reportRepo.FindByField("code", report.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if existing.Id != 0 {
		return 0, myerrors.NewValidationError("报表编码已存在")
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, err
	}
	report.Id = int(id)
	return report.Id, s.reportRepo.Create(s.reportRepo.DBWithContext(ctx), &report)
}

func (s *ReportService) UpdateReportDefinition(ctx *gin.Context, req request.ReportDefinitionUpdateReq) error {
	report, err := s.reportFromUpdateReq(req)
	if err != nil {
		return err
	}
	existing, err := s.reportRepo.FindByField("code", report.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing.Id != 0 && existing.Id != req.Id {
		return myerrors.NewValidationError("报表编码已存在")
	}
	current, err := s.reportRepo.FindById(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrDataNotFound
		}
		return err
	}
	if report.Status == reportStatusPublished && current.PublishedVersionId <= 0 {
		return myerrors.NewValidationError("发布报表必须调用 /admin/report/:id/publish")
	}
	tx := s.reportRepo.DBWithContext(ctx).Model(&model.ReportDefinition{}).Where("id = ?", req.Id).Updates(map[string]any{
		"code":                  report.Code,
		"name":                  report.Name,
		"description":           report.Description,
		"category":              report.Category,
		"status":                report.Status,
		"source_type":           report.SourceType,
		"source_code":           report.SourceCode,
		"permission_menu_id":    report.PermissionMenuId,
		"permission_table_code": report.PermissionTableCode,
		"query_config":          report.QueryConfig,
		"layout_config":         report.LayoutConfig,
		"remark":                report.Remark,
		"state":                 report.State,
	})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return myerrors.ErrDataNotFound
	}
	return nil
}

func (s *ReportService) DeleteReportDefinitionById(ctx *gin.Context, id int) error {
	return s.reportRepo.DeleteById(s.reportRepo.DBWithContext(ctx), id)
}

func (s *ReportService) UpdateReportDefinitionStatus(ctx *gin.Context, id int, status string) error {
	status = normalizeReportStatus(status)
	if id <= 0 || !isValidReportStatus(status) {
		return myerrors.ErrParamInvalid
	}
	if status == reportStatusPublished {
		return myerrors.NewValidationError("发布报表必须调用 /admin/report/:id/publish")
	}
	tx := s.reportRepo.DBWithContext(ctx).Model(&model.ReportDefinition{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"state":      status != reportStatusDisabled,
		"gmt_modify": model.Now(),
	})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return myerrors.ErrDataNotFound
	}
	return nil
}

func (s *ReportService) PublishReport(ctx *gin.Context, reportId int, req request.ReportPublishReq) (response.ReportPublishRes, error) {
	if reportId <= 0 {
		return response.ReportPublishRes{}, myerrors.ErrParamInvalid
	}
	var published model.ReportDefinitionVersion
	err := RunInTransaction(reportRequestContext(ctx), s.reportRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		var report model.ReportDefinition
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.ErrDataNotFound
			}
			return err
		}
		if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
			return myerrors.NewValidationError("报表已停用，不能发布")
		}
		config, err := reportconfig.Parse(report.QueryConfig, report.LayoutConfig)
		if err != nil {
			return err
		}
		if err := s.validateReportTables(report); err != nil {
			return err
		}
		if err := validateReportSQLDatasets(config); err != nil {
			return err
		}
		if err := ensureSQLDatasetRole(ctx, config); err != nil {
			return err
		}
		maxVersionNo, err := s.reportVersionRepo.GetMaxVersionNo(tx, report.Id)
		if err != nil {
			return err
		}
		id, err := s.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		user := reportUserFromContext(ctx)
		publishedAt := model.CustomTime(model.Now())
		version := model.ReportDefinitionVersion{
			Basic:               model.Basic{Id: int(id), State: true},
			ReportId:            report.Id,
			VersionNo:           maxVersionNo + 1,
			ReportCode:          report.Code,
			ReportName:          report.Name,
			Description:         report.Description,
			Category:            report.Category,
			SourceType:          report.SourceType,
			SourceCode:          report.SourceCode,
			PermissionMenuId:    report.PermissionMenuId,
			PermissionTableCode: report.PermissionTableCode,
			QueryConfig:         cloneReportJSON(report.QueryConfig),
			LayoutConfig:        cloneReportJSON(report.LayoutConfig),
			Status:              reportVersionPublished,
			PublishedAt:         publishedAt,
			PublishedBy:         user.Id,
			PublishedName:       user.UserName,
			ChangeLog:           strings.TrimSpace(req.ChangeLog),
		}
		if err := s.reportVersionRepo.ArchiveByReportId(tx, report.Id); err != nil {
			return err
		}
		if err := s.reportVersionRepo.Create(tx, &version); err != nil {
			return err
		}
		if err := tx.Model(&model.ReportDefinition{}).
			Where("id = ?", report.Id).
			Updates(map[string]any{
				"published_version_id": version.Id,
				"status":               reportStatusPublished,
				"state":                true,
				"gmt_modify":           model.Now(),
			}).Error; err != nil {
			return err
		}
		published = version
		return nil
	})
	if err != nil {
		return response.ReportPublishRes{}, err
	}
	return response.ReportPublishRes{
		ReportId:  published.ReportId,
		VersionId: published.Id,
		VersionNo: published.VersionNo,
		Status:    reportStatusPublished,
	}, nil
}

func (s *ReportService) GetReportVersions(reportId int) ([]response.ReportDefinitionVersionRes, error) {
	report, err := s.GetReportDefinitionById(reportId)
	if err != nil {
		return nil, err
	}
	if report.Id == 0 {
		return nil, myerrors.ErrDataNotFound
	}
	versions, err := s.reportVersionRepo.ListByReportId(reportId)
	if err != nil {
		return nil, err
	}
	result := make([]response.ReportDefinitionVersionRes, 0, len(versions))
	for _, version := range versions {
		result = append(result, response.ReportDefinitionVersionRes{
			Id:            version.Id,
			ReportId:      version.ReportId,
			VersionNo:     version.VersionNo,
			Status:        version.Status,
			PublishedAt:   version.PublishedAt.String(),
			PublishedBy:   version.PublishedBy,
			PublishedName: version.PublishedName,
			ChangeLog:     version.ChangeLog,
			IsCurrent:     version.Id == report.PublishedVersionId,
		})
	}
	return result, nil
}

func (s *ReportService) PublishReportAsMenu(ctx *gin.Context, reportId int, req request.ReportPublishMenuReq) (response.ReportPublishMenuRes, error) {
	if reportId <= 0 {
		return response.ReportPublishMenuRes{}, myerrors.ErrParamInvalid
	}
	var result response.ReportPublishMenuRes
	var policies []reportMenuPolicy
	err := RunInTransaction(reportRequestContext(ctx), s.reportRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		var report model.ReportDefinition
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.ErrDataNotFound
			}
			return err
		}
		if normalizeReportStatus(report.Status) != reportStatusPublished || !report.State {
			return myerrors.NewValidationError("报表未发布，不能发布到菜单")
		}
		if report.PublishedVersionId <= 0 {
			return myerrors.NewValidationError("报表缺少发布版本，不能发布到菜单")
		}
		var version model.ReportDefinitionVersion
		if err := tx.First(&version, "id = ? AND report_id = ?", report.PublishedVersionId, report.Id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.NewValidationError("报表发布版本不存在")
			}
			return err
		}
		if !version.State || normalizeReportStatus(version.Status) != reportVersionPublished {
			return myerrors.NewValidationError("报表发布版本状态不可发布到菜单")
		}
		parent, err := s.validateReportMenuParent(tx, req.ParentMenuId)
		if err != nil {
			return err
		}
		menuPath := normalizeReportMenuPath(req.Path, report.Code)
		existingMenu, err := s.findReportMenuForUpdate(tx, report)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		existingMenuID := 0
		if err == nil {
			existingMenuID = existingMenu.Id
		}
		if err := s.ensureReportMenuPathAvailable(tx, menuPath, existingMenuID); err != nil {
			return err
		}
		menu, err := s.upsertReportMenu(tx, report, parent.Id, existingMenu, req, menuPath)
		if err != nil {
			return err
		}
		buttons, err := s.ensureReportRuntimeButtons(tx, report, menu.Id)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.ReportDefinition{}).
			Where("id = ?", report.Id).
			Updates(map[string]any{
				"permission_menu_id": menu.Id,
				"gmt_modify":         model.Now(),
			}).Error; err != nil {
			return err
		}
		policies, err = s.grantReportMenuRoles(tx, menu.Id, buttons, req.PermissionRoleIds)
		if err != nil {
			return err
		}
		for _, policy := range policies {
			if err := s.casbinRuleRepo.UpsertPolicyWithDB(tx, policy.RoleName, policy.Path, policy.Method); err != nil {
				return err
			}
		}
		result = reportMenuResponse(report, menu, true)
		return nil
	})
	if err != nil {
		return response.ReportPublishMenuRes{}, err
	}
	if err := s.casbinRuleRepo.ReloadPolicy(); err != nil {
		return response.ReportPublishMenuRes{}, err
	}
	return result, nil
}

func (s *ReportService) UnpublishReportMenu(ctx *gin.Context, reportId int) (response.ReportPublishMenuRes, error) {
	if reportId <= 0 {
		return response.ReportPublishMenuRes{}, myerrors.ErrParamInvalid
	}
	db := s.reportRepo.DBWithContext(ctx)
	var result response.ReportPublishMenuRes
	var cleanups []reportMenuPolicy
	err := RunInTransaction(reportRequestContext(ctx), db, func(tx *gorm.DB) error {
		var report model.ReportDefinition
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.ErrDataNotFound
			}
			return err
		}
		result = response.ReportPublishMenuRes{
			ReportId:        report.Id,
			ReportCode:      report.Code,
			MenuId:          report.PermissionMenuId,
			PublishedToMenu: false,
		}
		if report.PermissionMenuId <= 0 {
			return nil
		}
		var menu model.SysMenu
		if err := tx.First(&menu, report.PermissionMenuId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if menu.PageType != enum.MenuPageTypeReport {
			return myerrors.NewValidationError("报表运行菜单类型不匹配")
		}
		if option, ok := parseReportMenuOption(menu.Option); ok && option.ReportId != report.Id {
			return myerrors.NewValidationError("报表运行菜单绑定信息不匹配")
		}
		buttons, err := s.reportMenuButtons(tx, menu.Id)
		if err != nil {
			return err
		}
		candidates, err := s.reportButtonPolicyCandidates(tx, buttons)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.SysMenu{}).
			Where("id = ?", menu.Id).
			Updates(map[string]any{
				"is_hidden":  true,
				"state":      false,
				"gmt_modify": model.Now(),
			}).Error; err != nil {
			return err
		}
		if err := s.sysRoleMenuRepo.DeleteByMenuIds(tx, []int{menu.Id}); err != nil {
			return err
		}
		if err := s.sysRoleMenuButtonRepo.DeleteByMenuIds(tx, []int{menu.Id}); err != nil {
			return err
		}
		cleanups, err = s.reportOrphanRolePolicyCleanups(tx, candidates)
		if err != nil {
			return err
		}
		for _, cleanup := range cleanups {
			if err := s.casbinRuleRepo.RemovePolicyWithDB(tx, cleanup.RoleName, cleanup.Path, cleanup.Method); err != nil {
				return err
			}
		}
		result = reportMenuResponse(report, menu, false)
		result.Visible = false
		result.PublishedToMenu = false
		return nil
	})
	if err != nil {
		return response.ReportPublishMenuRes{}, err
	}
	if err := s.casbinRuleRepo.ReloadPolicy(); err != nil {
		return response.ReportPublishMenuRes{}, err
	}
	return result, nil
}

func (s *ReportService) DesignPreview(ctx *gin.Context, reportId int, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	start := time.Now()
	executionCtx, cancel := withReportExecutionDeadline(reportRequestContext(ctx), reportDesignPreviewDeadline)
	defer cancel()
	ctx = reportGinContextWithExecutionContext(ctx, executionCtx)
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: reportRuntimeDesignPreview}
	report, err := s.GetReportDefinitionByIdWithContext(reportRequestContext(ctx), reportId)
	if err != nil {
		_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, false, 0, start, err)
		return response.ReportPreviewRes{}, normalizeReportExecutionError(err)
	}
	if report.Id == 0 {
		err = myerrors.ErrDataNotFound
		_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	snapshot = reportSnapshotFromDefinition(report, reportRuntimeDesignPreview)
	if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
		err = myerrors.NewValidationError("报表已停用")
		_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	result, err := s.executeReportSnapshot(ctx, snapshot, req, start)
	return result, normalizeReportExecutionError(err)
}

func (s *ReportService) RunReport(ctx *gin.Context, reportId int, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	start := time.Now()
	executionCtx, cancel := withReportExecutionDeadline(reportRequestContext(ctx), reportRuntimeRunDeadline)
	defer cancel()
	ctx = reportGinContextWithExecutionContext(ctx, executionCtx)
	snapshot, err := s.loadPublishedReportSnapshot(ctx, reportId, reportRuntimeRun)
	if err != nil {
		_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, false, 0, start, err)
		return response.ReportPreviewRes{}, normalizeReportExecutionError(err)
	}
	if err := s.authorizePublishedReportRun(reportRequestContext(ctx), reportUserFromContext(ctx), snapshot.PermissionMenuId, req.MenuId); err != nil {
		_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	result, err := s.executeReportSnapshot(ctx, snapshot, req, start)
	return result, normalizeReportExecutionError(err)
}

func (s *ReportService) ExportReport(ctx *gin.Context, reportId int, req request.ReportExportReq) (response.ReportExportFile, error) {
	start := time.Now()
	executionCtx, cancel := withReportExecutionDeadline(reportRequestContext(ctx), reportRuntimeExportDeadline)
	defer cancel()
	ctx = reportGinContextWithExecutionContext(ctx, executionCtx)
	previewReq := reportExportPreviewReq(req)
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: reportRuntimeExport}
	fail := func(err error) (response.ReportExportFile, error) {
		err = normalizeReportExecutionError(err)
		_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, false, 0, start, err)
		return response.ReportExportFile{}, err
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = reportExportFormatCSV
	}
	if format == "xlsx" {
		return fail(myerrors.NewValidationError("报表导出暂不支持 xlsx 格式，仅支持 csv"))
	}
	if format != reportExportFormatCSV {
		return fail(myerrors.NewValidationError("报表导出格式不支持，仅支持 csv"))
	}
	effectiveMaxRows, err := normalizeReportExportMaxRows(req)
	if err != nil {
		return fail(err)
	}
	previewReq.Query.Page = 1
	previewReq.Query.Num = effectiveMaxRows
	snapshot, err = s.loadPublishedReportSnapshot(ctx, reportId, reportRuntimeExport)
	if err != nil {
		return fail(err)
	}
	if err := s.authorizePublishedReportExport(reportRequestContext(ctx), reportUserFromContext(ctx), snapshot.PermissionMenuId, req.MenuId); err != nil {
		return fail(err)
	}
	preview, err := s.executeReportSnapshotWithOptions(ctx, snapshot, previewReq, start, ReportExecutionOptions{
		MaxRows:              effectiveMaxRows,
		PageSizeLimit:        effectiveMaxRows,
		DefaultPageSize:      effectiveMaxRows,
		ExportMode:           true,
		WriteLog:             false,
		DataPermissionAction: enum.ButtonActionExport,
	})
	if err != nil {
		return fail(err)
	}
	if preview.Total > effectiveMaxRows {
		return fail(myerrors.NewValidationError("导出行数超过系统限制，请缩小查询条件后重试"))
	}
	content, err := buildReportCSV(preview.Columns, preview.Rows)
	if err != nil {
		return fail(err)
	}
	if err := s.writeExecutionLog(reportRequestContext(ctx), snapshot, true, len(preview.Rows), start, nil); err != nil {
		return response.ReportExportFile{}, err
	}
	return response.ReportExportFile{
		FileName:    reportExportFileName(snapshot),
		ContentType: "text/csv; charset=utf-8",
		Content:     content,
		RowCount:    len(preview.Rows),
	}, nil
}

func (s *ReportService) loadPublishedReportSnapshot(ctx *gin.Context, reportId int, runtimeType string) (ReportExecutionSnapshot, error) {
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: runtimeType}
	report, err := s.GetReportDefinitionByIdWithContext(reportRequestContext(ctx), reportId)
	if err != nil {
		return snapshot, err
	}
	if report.Id == 0 {
		return snapshot, myerrors.ErrDataNotFound
	}
	snapshot.ReportId = report.Id
	snapshot.Code = report.Code
	if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
		return snapshot, myerrors.NewValidationError("报表已停用")
	}
	if normalizeReportStatus(report.Status) != reportStatusPublished || report.PublishedVersionId <= 0 {
		return snapshot, myerrors.NewValidationError("报表未发布，请先调用发布接口")
	}
	version, err := s.reportVersionRepo.WithContext(reportRequestContext(ctx)).FindById(report.PublishedVersionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = myerrors.NewValidationError("报表发布版本不存在")
		}
		return snapshot, err
	}
	if version.ReportId != report.Id {
		return snapshot, myerrors.NewValidationError("报表发布版本不存在")
	}
	if !version.State {
		return snapshot, myerrors.NewValidationError("报表发布版本不可用")
	}
	if normalizeReportStatus(version.Status) != reportVersionPublished {
		return snapshot, myerrors.NewValidationError("报表发布版本状态不可运行")
	}
	snapshot = reportSnapshotFromVersion(version, runtimeType)
	// 菜单发布是版本发布后的运行时绑定；授权必须依据当前Report与菜单关系，不能使用过期快照值。
	snapshot.PermissionMenuId = report.PermissionMenuId
	return snapshot, nil
}

func (s *ReportService) Preview(ctx *gin.Context, reportId int, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	return s.DesignPreview(ctx, reportId, req)
}

func (s *ReportService) executeReportSnapshot(ctx *gin.Context, snapshot ReportExecutionSnapshot, req request.ReportPreviewReq, start time.Time) (response.ReportPreviewRes, error) {
	return s.executeReportSnapshotWithOptions(ctx, snapshot, req, start, ReportExecutionOptions{
		PageSizeLimit:        200,
		DefaultPageSize:      20,
		WriteLog:             true,
		DataPermissionAction: enum.ButtonActionQuery,
	})
}

func (s *ReportService) executeReportSnapshotWithOptions(ctx *gin.Context, snapshot ReportExecutionSnapshot, req request.ReportPreviewReq, start time.Time, options ReportExecutionOptions) (response.ReportPreviewRes, error) {
	options = normalizeReportExecutionOptions(options)
	writeFailure := func(err error) {
		if options.WriteLog {
			_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, false, 0, start, normalizeReportExecutionError(err))
		}
	}
	writeSuccess := func(rowCount int) {
		if options.WriteLog {
			_ = s.writeExecutionLog(reportRequestContext(ctx), snapshot, true, rowCount, start, nil)
		}
	}
	config, err := reportconfig.Parse(snapshot.QueryConfig, snapshot.LayoutConfig)
	if err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	if err := validateReportSQLDatasets(config); err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	if err := ensureSQLDatasetRole(ctx, config); err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	var selectedDataset reportconfig.Dataset
	if strings.TrimSpace(req.DatasetId) != "" {
		var ok bool
		selectedDataset, ok = config.DatasetByID(req.DatasetId)
		if !ok {
			err = myerrors.NewValidationError("报表数据集不存在")
			writeFailure(err)
			return response.ReportPreviewRes{}, err
		}
		if selectedDataset.Type == reportconfig.SourceTypeSQL {
			preview, err := s.previewSQLDataset(ctx, snapshot, config, selectedDataset, req, options)
			if err != nil {
				writeFailure(err)
				return response.ReportPreviewRes{}, err
			}
			writeSuccess(len(preview.Rows))
			return preview, nil
		}
	}
	if selectedDataset.Id == "" && reportShouldUseJoinedPreview(config) {
		preview, err := s.previewJoinedTableDatasets(ctx, snapshot, config, req, options)
		if err != nil {
			writeFailure(err)
			return response.ReportPreviewRes{}, err
		}
		writeSuccess(len(preview.Rows))
		return preview, nil
	}
	activeDatasetID := reportDatasetIdForPreview(config, selectedDataset)
	sourceTable, _, err := s.resolveReportPreviewTable(reportRequestContext(ctx), snapshot, selectedDataset)
	if err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	sourceTable = reportTableWithPreviewFields(sourceTable, config, activeDatasetID)
	query := req.Query
	if err := applyReportParameterValues(&query, config, activeDatasetID, req.Parameters); err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	query.Num = normalizeReportPageSize(query.Num, options)
	query.TableCode = sourceTable.TableCode
	query.MenuId = req.MenuId
	if snapshot.PermissionMenuId > 0 {
		query.MenuId = snapshot.PermissionMenuId
	}
	permission, err := s.generalizationService.ResolveDataPermission(reportRequestContext(ctx), sourceTable, reportDataPermissionOperation(options.DataPermissionAction))
	if err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	var result repository.GeneralizationListResult
	err = withReportExecutionTransaction(reportRequestContext(ctx), s.reportRepo.DBWithContext(reportRequestContext(ctx)), func(tx *gorm.DB) error {
		var queryErr error
		result, queryErr = s.generalizationService.QueryWithResolvedDataPermissionDB(tx, &query, sourceTable, permission)
		if queryErr != nil {
			return queryErr
		}
		if !options.ExportMode {
			return nil
		}
		if result.Total > options.MaxRows {
			return myerrors.NewValidationError("导出行数超过系统限制，请缩小查询条件后重试")
		}
		return s.completeReportTableExportRows(tx, query, sourceTable, permission, &result, options.MaxRows)
	})
	if err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	columns := reportPreviewColumnsFromConfig(sourceTable, snapshot.QueryConfig)
	preview := response.ReportPreviewRes{
		Columns: columns,
		Rows:    filterReportRows(result.Data, columns),
		Total:   result.Total,
		Meta: response.ReportPreviewMeta{
			ReportId:    snapshot.ReportId,
			VersionId:   snapshot.VersionId,
			VersionNo:   snapshot.VersionNo,
			RuntimeType: snapshot.RuntimeType,
			ReportCode:  snapshot.Code,
			SourceCode:  sourceTable.TableCode,
			DatasetId:   activeDatasetID,
			DatasetType: reportSourceTypeTable,
			AppliedMenu: query.MenuId,
		},
		Datasets: reportPreviewDatasets(config, snapshot, columns),
		Joins:    reportConfigDatasetJoins(config),
	}
	writeSuccess(len(result.Data))
	return preview, nil
}

func (s *ReportService) reportFromCreateReq(req request.ReportDefinitionCreateReq) (model.ReportDefinition, error) {
	report := model.ReportDefinition{}
	if err := copier.Copy(&report, &req); err != nil {
		return report, err
	}
	report.Code = strings.TrimSpace(req.Code)
	report.Name = strings.TrimSpace(req.Name)
	report.SourceType = strings.TrimSpace(req.SourceType)
	report.SourceCode = strings.TrimSpace(req.SourceCode)
	report.PermissionTableCode = strings.TrimSpace(req.PermissionTableCode)
	report.Status = normalizeReportStatus(req.Status)
	if report.Code == "" || report.Name == "" {
		return report, myerrors.ErrParamInvalid
	}
	if !isValidReportStatus(report.Status) {
		return report, myerrors.ErrParamInvalid
	}
	if report.Status == reportStatusPublished {
		return report, myerrors.NewValidationError("发布报表必须调用 /admin/report/:id/publish")
	}
	if len(report.QueryConfig) == 0 {
		report.QueryConfig = datatypes.JSON([]byte("{}"))
	}
	if len(report.LayoutConfig) == 0 {
		report.LayoutConfig = datatypes.JSON([]byte("{}"))
	}
	report.State = report.Status != reportStatusDisabled
	if err := s.applyReportPrimaryDataset(&report); err != nil {
		return report, err
	}
	return report, s.validateReportTables(report)
}

func (s *ReportService) reportFromUpdateReq(req request.ReportDefinitionUpdateReq) (model.ReportDefinition, error) {
	report := model.ReportDefinition{}
	if err := copier.Copy(&report, &req); err != nil {
		return report, err
	}
	report.Code = strings.TrimSpace(req.Code)
	report.Name = strings.TrimSpace(req.Name)
	report.SourceType = strings.TrimSpace(req.SourceType)
	report.SourceCode = strings.TrimSpace(req.SourceCode)
	report.PermissionTableCode = strings.TrimSpace(req.PermissionTableCode)
	report.Status = normalizeReportStatus(req.Status)
	if req.Id <= 0 || report.Code == "" || report.Name == "" {
		return report, myerrors.ErrParamInvalid
	}
	if !isValidReportStatus(report.Status) {
		return report, myerrors.ErrParamInvalid
	}
	if len(report.QueryConfig) == 0 {
		report.QueryConfig = datatypes.JSON([]byte("{}"))
	}
	if len(report.LayoutConfig) == 0 {
		report.LayoutConfig = datatypes.JSON([]byte("{}"))
	}
	report.State = report.Status != reportStatusDisabled
	if err := s.applyReportPrimaryDataset(&report); err != nil {
		return report, err
	}
	return report, s.validateReportTables(report)
}

func normalizeReportStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return reportStatusDraft
	}
	return status
}

func isValidReportStatus(status string) bool {
	switch status {
	case reportStatusDraft, reportStatusPublished, reportStatusDisabled:
		return true
	default:
		return false
	}
}

func (s *ReportService) applyReportPrimaryDataset(report *model.ReportDefinition) error {
	config, err := reportconfig.Parse(report.QueryConfig, report.LayoutConfig)
	if err != nil {
		return err
	}
	if config.HasDatasets() {
		dataset, ok := config.PrimaryTableDataset()
		if !ok {
			return myerrors.NewValidationError("报表必须配置 primary table 数据集")
		}
		report.SourceType = reportSourceTypeTable
		report.SourceCode = dataset.SourceCode
		if strings.TrimSpace(report.PermissionTableCode) == "" {
			report.PermissionTableCode = dataset.SourceCode
		}
		return nil
	}
	report.SourceType = normalizeReportSourceType(report.SourceType)
	report.SourceCode = strings.TrimSpace(report.SourceCode)
	if report.SourceType == "" {
		return myerrors.NewValidationError("报表数据源类型不合法")
	}
	if report.SourceCode == "" {
		return myerrors.ErrParamInvalid
	}
	if strings.TrimSpace(report.PermissionTableCode) == "" {
		report.PermissionTableCode = report.SourceCode
	}
	return nil
}

func (s *ReportService) validateReportTables(report model.ReportDefinition) error {
	if _, _, err := s.resolveReportTables(report); err != nil {
		return err
	}
	config, err := reportconfig.Parse(report.QueryConfig, report.LayoutConfig)
	if err != nil {
		return err
	}
	return s.validateReportDatasetTables(config)
}

func (s *ReportService) resolveReportTables(report model.ReportDefinition) (model.SysTable, model.SysTable, error) {
	return s.resolveReportTablesWithContext(context.Background(), report)
}

func (s *ReportService) resolveReportTablesWithContext(ctx context.Context, report model.ReportDefinition) (model.SysTable, model.SysTable, error) {
	if normalizeReportSourceType(report.SourceType) == "" {
		return model.SysTable{}, model.SysTable{}, myerrors.NewValidationError("报表数据源类型不合法")
	}
	sourceTable, err := s.ResolveRuntimeTable(ctx, strings.TrimSpace(report.SourceCode))
	if err != nil {
		return model.SysTable{}, model.SysTable{}, err
	}
	if sourceTable.Id == 0 {
		return model.SysTable{}, model.SysTable{}, myerrors.NewValidationError("报表数据源表不存在")
	}
	permissionTable := sourceTable
	if strings.TrimSpace(report.PermissionTableCode) != "" && report.PermissionTableCode != sourceTable.TableCode {
		return model.SysTable{}, model.SysTable{}, myerrors.NewValidationError("报表权限表暂仅支持与数据源表一致")
	}
	return sourceTable, permissionTable, nil
}

func (s *ReportService) resolveReportPreviewTable(ctx context.Context, snapshot ReportExecutionSnapshot, selectedDataset reportconfig.Dataset) (model.SysTable, model.SysTable, error) {
	if selectedDataset.Id == "" {
		return s.resolveReportTablesWithContext(ctx, reportDefinitionFromSnapshot(snapshot))
	}
	sourceTable, err := s.ResolveRuntimeTable(ctx, strings.TrimSpace(selectedDataset.SourceCode))
	if err != nil {
		return model.SysTable{}, model.SysTable{}, err
	}
	if sourceTable.Id == 0 {
		return model.SysTable{}, model.SysTable{}, myerrors.NewValidationError("报表数据集表不存在")
	}
	return sourceTable, sourceTable, nil
}

func (s *ReportService) validateReportDatasetTables(config reportconfig.Config) error {
	if s.metadataRuntime == nil {
		return myerrors.NewValidationError("报表表结构服务未初始化")
	}
	datasetByID := make(map[string]reportconfig.Dataset, len(config.Datasets()))
	for _, dataset := range config.Datasets() {
		dataset = reportconfig.NormalizeDataset(dataset)
		if dataset.Id != "" {
			datasetByID[dataset.Id] = dataset
		}
		if dataset.Type != reportconfig.SourceTypeTable {
			continue
		}
		table, err := s.ResolveRuntimeTable(context.Background(), dataset.SourceCode)
		if err != nil {
			return err
		}
		if table.Id == 0 {
			return myerrors.NewValidationError(fmt.Sprintf("报表数据集表不存在: %s", dataset.SourceCode))
		}
	}
	for _, join := range config.DatasetJoins() {
		left := datasetByID[strings.TrimSpace(join.LeftDatasetId)]
		right := datasetByID[strings.TrimSpace(join.RightDatasetId)]
		if left.Type != reportconfig.SourceTypeTable || right.Type != reportconfig.SourceTypeTable {
			return myerrors.NewValidationError("报表数据集关联暂仅支持表数据集")
		}
	}
	if len(config.DatasetJoins()) > 0 {
		primary, ok := config.PrimaryTableDataset()
		if !ok {
			return myerrors.NewValidationError("报表必须配置 primary table 数据集")
		}
		if _, err := reportOrderedDatasetJoins(config, primary.Id); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReportService) previewSQLDataset(ctx *gin.Context, snapshot ReportExecutionSnapshot, config reportconfig.Config, dataset reportconfig.Dataset, req request.ReportPreviewReq, options ReportExecutionOptions) (response.ReportPreviewRes, error) {
	sqlText, err := safeReportPreviewSQL(dataset.SQL)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	whereClause, args, err := reportSQLParameterWhere(config, dataset.Id, req.Parameters)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	limit := normalizeReportPageSize(req.Query.Num, options)
	page := req.Query.Page
	if page <= 0 {
		page = 1
	}
	rows, columns, total, err := s.queryReportSQL(ctx, sqlText, whereClause, args, page, limit)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	return response.ReportPreviewRes{
		Columns:  columns,
		Rows:     rows,
		Total:    total,
		Datasets: reportConfigDatasetMetadata(config),
		Joins:    reportConfigDatasetJoins(config),
		Meta: response.ReportPreviewMeta{
			ReportId:    snapshot.ReportId,
			VersionId:   snapshot.VersionId,
			VersionNo:   snapshot.VersionNo,
			RuntimeType: snapshot.RuntimeType,
			ReportCode:  snapshot.Code,
			SourceCode:  snapshot.SourceCode,
			DatasetId:   dataset.Id,
			DatasetType: reportconfig.SourceTypeSQL,
			AppliedMenu: reportAppliedMenu(snapshot, req.MenuId),
		},
	}, nil
}

func (s *ReportService) previewJoinedTableDatasets(ctx *gin.Context, snapshot ReportExecutionSnapshot, config reportconfig.Config, req request.ReportPreviewReq, options ReportExecutionOptions) (response.ReportPreviewRes, error) {
	primaryDataset, ok := config.PrimaryTableDataset()
	if !ok {
		return response.ReportPreviewRes{}, myerrors.NewValidationError("报表必须配置 primary table 数据集")
	}
	datasets := config.Datasets()
	datasetByID := make(map[string]reportconfig.Dataset, len(datasets))
	tableByDatasetID := make(map[string]model.SysTable, len(datasets))
	for _, dataset := range datasets {
		dataset = reportconfig.NormalizeDataset(dataset)
		if dataset.Id == "" {
			continue
		}
		datasetByID[dataset.Id] = dataset
		if dataset.Type != reportconfig.SourceTypeTable {
			continue
		}
		table, err := s.ResolveRuntimeTable(reportRequestContext(ctx), dataset.SourceCode)
		if err != nil {
			return response.ReportPreviewRes{}, err
		}
		if table.Id == 0 {
			return response.ReportPreviewRes{}, myerrors.NewValidationError(fmt.Sprintf("报表数据集表不存在: %s", dataset.SourceCode))
		}
		tableByDatasetID[dataset.Id] = table
	}
	primaryTable, ok := tableByDatasetID[primaryDataset.Id]
	if !ok {
		return response.ReportPreviewRes{}, myerrors.NewValidationError("报表主数据集表不存在")
	}

	aliasByDatasetID := reportDatasetAliases(config, primaryDataset.Id, primaryTable.TableCode)
	selections, columns, err := reportJoinedPreviewSelections(config, primaryDataset.Id, primaryTable, tableByDatasetID, aliasByDatasetID)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	if len(selections) == 0 {
		return response.ReportPreviewRes{}, myerrors.NewValidationError("报表未配置可预览字段")
	}
	keyword := ""
	if req.Query.QuickQuery != nil {
		keyword = req.Query.QuickQuery.Keyword
	}
	page := req.Query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := normalizeReportPageSize(req.Query.Num, options)
	permission, err := s.generalizationService.ResolveDataPermission(reportRequestContext(ctx), primaryTable, reportDataPermissionOperation(options.DataPermissionAction))
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	var total int64
	var rows []map[string]interface{}
	err = withReportExecutionTransaction(reportRequestContext(ctx), s.reportRepo.DBWithContext(reportRequestContext(ctx)), func(tx *gorm.DB) error {
		query := tx.Table(quoteReportIdentifier(primaryTable.TableCode))
		if _, ok := reportFindTableField(primaryTable, "gmt_delete"); ok {
			query = query.Where(fmt.Sprintf("%s IS NULL", reportDatasetFieldExpr(primaryDataset.Id, "gmt_delete", primaryDataset.Id, primaryTable.TableCode, aliasByDatasetID)))
		}
		query, err = queryutil.ApplyGeneralizationPermission(query, permission, primaryTable)
		if err != nil {
			return err
		}
		orderedJoins, joinOrderErr := reportOrderedDatasetJoins(config, primaryDataset.Id)
		if joinOrderErr != nil {
			return joinOrderErr
		}
		joinedDatasetIDs := make(map[string]struct{})
		for _, join := range orderedJoins {
			targetID := reportJoinTargetDatasetID(join, primaryDataset.Id)
			if targetID != "" {
				if _, exists := joinedDatasetIDs[targetID]; exists {
					continue
				}
				joinedDatasetIDs[targetID] = struct{}{}
			}
			joinExpr, joinErr := reportJoinSQL(join, primaryDataset.Id, primaryTable.TableCode, datasetByID, tableByDatasetID, aliasByDatasetID)
			if joinErr != nil {
				return joinErr
			}
			if joinExpr != "" {
				query = query.Joins(joinExpr)
			}
		}
		query, err = reportApplyJoinedParameters(query, config, primaryDataset.Id, primaryTable.TableCode, tableByDatasetID, aliasByDatasetID, req.Parameters)
		if err != nil {
			return err
		}
		query = reportApplyJoinedQuickSearch(query, keyword, selections)
		if err = query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
			return err
		}
		sqlRows, queryErr := query.
			Select(strings.Join(reportSelectExprs(selections), ", ")).
			Limit(pageSize).
			Offset((page - 1) * pageSize).
			Rows()
		if queryErr != nil {
			return queryErr
		}
		rows, _, queryErr = scanReportSQLRows(sqlRows)
		return queryErr
	})
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	return response.ReportPreviewRes{
		Columns:  columns,
		Rows:     rows,
		Total:    int(total),
		Datasets: reportConfigDatasetMetadata(config),
		Joins:    reportConfigDatasetJoins(config),
		Meta: response.ReportPreviewMeta{
			ReportId:    snapshot.ReportId,
			VersionId:   snapshot.VersionId,
			VersionNo:   snapshot.VersionNo,
			RuntimeType: snapshot.RuntimeType,
			ReportCode:  snapshot.Code,
			SourceCode:  primaryTable.TableCode,
			DatasetId:   primaryDataset.Id,
			DatasetType: reportSourceTypeTable,
			AppliedMenu: reportAppliedMenu(snapshot, req.MenuId),
		},
	}, nil
}

func reportRequestContext(ctx *gin.Context) context.Context {
	if ctx != nil {
		if value, exists := ctx.Get(reportExecutionContextKey); exists {
			if executionCtx, ok := value.(context.Context); ok && executionCtx != nil {
				return executionCtx
			}
		}
	}
	if ctx != nil && ctx.Request != nil {
		return ctx.Request.Context()
	}
	return context.Background()
}

func reportGinContextWithExecutionContext(ctx *gin.Context, executionCtx context.Context) *gin.Context {
	if ctx == nil {
		return nil
	}
	cloned := ctx.Copy()
	cloned.Set(reportExecutionContextKey, executionCtx)
	if ctx.Request != nil {
		cloned.Request = ctx.Request.Clone(executionCtx)
	}
	return cloned
}

func safeReportPreviewSQL(raw string) (string, error) {
	sqlText, err := platformmetadata.ValidateReadOnlyQuery(raw)
	if err != nil {
		return "", myerrors.NewValidationError("SQL 数据集仅允许单条 SELECT/WITH 只读查询")
	}
	return sqlText, nil
}

func (s *ReportService) queryReportSQL(ctx *gin.Context, sqlText string, whereClause string, args []any, page int, limit int) ([]map[string]interface{}, []response.ReportPreviewColumn, int, error) {
	if s.reportRepo == nil {
		return nil, nil, 0, myerrors.NewValidationError("报表数据仓储未初始化")
	}
	wrapped := fmt.Sprintf("SELECT * FROM (%s) AS report_sql_dataset_preview", sqlText)
	if strings.TrimSpace(whereClause) != "" {
		wrapped += " WHERE " + whereClause
	}
	var total int64
	var columnMeta []response.ReportPreviewColumn
	var records []map[string]interface{}
	var columnNames []string
	err := withReportExecutionTransaction(reportRequestContext(ctx), s.reportRepo.DBWithContext(reportRequestContext(ctx)), func(tx *gorm.DB) error {
		if queryErr := tx.Raw("SELECT COUNT(1) FROM ("+wrapped+") AS report_sql_dataset_count", args...).Scan(&total).Error; queryErr != nil {
			return queryErr
		}
		queryArgs := append([]any{}, args...)
		dataSQL := wrapped + " LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, limit, (page-1)*limit)
		sqlRows, queryErr := tx.Raw(dataSQL, queryArgs...).Rows()
		if queryErr != nil {
			return queryErr
		}
		columnMeta, queryErr = reportSQLColumns(sqlRows)
		if queryErr != nil {
			_ = sqlRows.Close()
			return queryErr
		}
		records, columnNames, queryErr = scanReportSQLRows(sqlRows)
		return queryErr
	})
	if err != nil {
		return nil, nil, 0, err
	}
	if len(columnMeta) == 0 {
		columnMeta = make([]response.ReportPreviewColumn, 0, len(columnNames))
		for _, name := range columnNames {
			columnMeta = append(columnMeta, response.ReportPreviewColumn{
				Name:  name,
				Field: name,
				Label: name,
				Type:  "string",
			})
		}
	}
	return records, columnMeta, int(total), nil
}

type reportJoinedSelection struct {
	Expr      string
	Alias     string
	Label     string
	Type      string
	Role      string
	Field     model.SysTableField
	DatasetID string
}

func scanReportSQLRows(sqlRows *sql.Rows) ([]map[string]interface{}, []string, error) {
	defer sqlRows.Close()
	columnNames, err := sqlRows.Columns()
	if err != nil {
		return nil, nil, err
	}
	records := make([]map[string]interface{}, 0)
	for sqlRows.Next() {
		values := make([]interface{}, len(columnNames))
		pointers := make([]interface{}, len(columnNames))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := sqlRows.Scan(pointers...); err != nil {
			return nil, nil, err
		}
		record := make(map[string]interface{}, len(columnNames))
		for i, name := range columnNames {
			record[name] = normalizeReportSQLValue(values[i])
		}
		records = append(records, record)
	}
	if err := sqlRows.Err(); err != nil {
		return nil, nil, err
	}
	return records, columnNames, nil
}

func reportSQLColumns(sqlRows *sql.Rows) ([]response.ReportPreviewColumn, error) {
	columnNames, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	columnTypes, _ := sqlRows.ColumnTypes()
	typeByName := make(map[string]string, len(columnTypes))
	for _, columnType := range columnTypes {
		typeByName[columnType.Name()] = reportSQLColumnType(columnType.DatabaseTypeName())
	}
	columns := make([]response.ReportPreviewColumn, 0, len(columnNames))
	for _, name := range columnNames {
		columns = append(columns, response.ReportPreviewColumn{
			Name:  name,
			Field: name,
			Label: name,
			Type:  typeByName[name],
		})
		if columns[len(columns)-1].Type == "" {
			columns[len(columns)-1].Type = "string"
		}
	}
	return columns, nil
}

func reportSQLColumnType(dbType string) string {
	value := strings.ToLower(strings.TrimSpace(dbType))
	switch {
	case strings.Contains(value, "int"), strings.Contains(value, "numeric"), strings.Contains(value, "decimal"), strings.Contains(value, "real"), strings.Contains(value, "double"), strings.Contains(value, "float"):
		return "number"
	case strings.Contains(value, "bool"):
		return "boolean"
	case strings.Contains(value, "date"), strings.Contains(value, "time"):
		return "datetime"
	case strings.Contains(value, "json"):
		return "json"
	default:
		return "string"
	}
}

func reportShouldUseJoinedPreview(config reportconfig.Config) bool {
	return len(config.DatasetJoins()) > 0
}

func reportOrderedDatasetJoins(config reportconfig.Config, primaryDatasetID string) ([]reportconfig.DatasetJoin, error) {
	primaryDatasetID = strings.TrimSpace(primaryDatasetID)
	if primaryDatasetID == "" {
		return nil, myerrors.NewValidationError("报表主数据集不存在")
	}
	remaining := append([]reportconfig.DatasetJoin(nil), config.DatasetJoins()...)
	ordered := make([]reportconfig.DatasetJoin, 0, len(remaining))
	connected := map[string]struct{}{primaryDatasetID: {}}
	for len(remaining) > 0 {
		nextIndex := -1
		shouldReverse := false
		for index, join := range remaining {
			_, leftConnected := connected[strings.TrimSpace(join.LeftDatasetId)]
			_, rightConnected := connected[strings.TrimSpace(join.RightDatasetId)]
			if leftConnected != rightConnected {
				nextIndex = index
				shouldReverse = rightConnected
				break
			}
		}
		if nextIndex < 0 {
			return nil, myerrors.NewValidationError("报表数据集关联必须从主数据集逐层连接，不能存在断开或循环关联")
		}
		join := remaining[nextIndex]
		if shouldReverse {
			join.LeftDatasetId, join.RightDatasetId = join.RightDatasetId, join.LeftDatasetId
			join.LeftField, join.RightField = join.RightField, join.LeftField
		}
		ordered = append(ordered, join)
		connected[strings.TrimSpace(join.LeftDatasetId)] = struct{}{}
		connected[strings.TrimSpace(join.RightDatasetId)] = struct{}{}
		remaining = append(remaining[:nextIndex], remaining[nextIndex+1:]...)
	}
	return ordered, nil
}

func reportDatasetAliases(config reportconfig.Config, primaryDatasetID string, primaryTableCode string) map[string]string {
	aliases := make(map[string]string)
	for index, dataset := range config.Datasets() {
		id := strings.TrimSpace(dataset.Id)
		if id == "" {
			continue
		}
		if id == primaryDatasetID {
			aliases[id] = primaryTableCode
			continue
		}
		aliases[id] = fmt.Sprintf("rds_%d", index+1)
	}
	return aliases
}

func reportJoinSQL(join reportconfig.DatasetJoin, primaryDatasetID string, primaryTableCode string, datasetByID map[string]reportconfig.Dataset, tableByDatasetID map[string]model.SysTable, aliasByDatasetID map[string]string) (string, error) {
	leftID := strings.TrimSpace(join.LeftDatasetId)
	rightID := strings.TrimSpace(join.RightDatasetId)
	leftDataset, leftOK := datasetByID[leftID]
	rightDataset, rightOK := datasetByID[rightID]
	if !leftOK || !rightOK {
		return "", myerrors.NewValidationError("报表数据集关联缺少数据集")
	}
	if leftDataset.Type != reportconfig.SourceTypeTable || rightDataset.Type != reportconfig.SourceTypeTable {
		return "", myerrors.NewValidationError("报表数据集关联暂仅支持表数据集")
	}
	leftTable, leftTableOK := tableByDatasetID[leftID]
	rightTable, rightTableOK := tableByDatasetID[rightID]
	if !leftTableOK || !rightTableOK {
		return "", myerrors.NewValidationError("报表数据集关联表不存在")
	}
	if _, ok := reportFindTableField(leftTable, join.LeftField); !ok {
		return "", myerrors.NewValidationError("报表数据集关联左字段不存在")
	}
	if _, ok := reportFindTableField(rightTable, join.RightField); !ok {
		return "", myerrors.NewValidationError("报表数据集关联右字段不存在")
	}
	targetID := reportJoinTargetDatasetID(join, primaryDatasetID)
	targetTable := rightTable
	if rightID == primaryDatasetID {
		targetTable = leftTable
	}
	if targetID == primaryDatasetID {
		return "", nil
	}
	targetAlias := aliasByDatasetID[targetID]
	if !isSafeReportIdentifier(targetAlias) {
		return "", myerrors.NewValidationError("报表数据集关联别名不合法")
	}
	leftExpr := reportDatasetFieldExpr(leftID, join.LeftField, primaryDatasetID, primaryTableCode, aliasByDatasetID)
	rightExpr := reportDatasetFieldExpr(rightID, join.RightField, primaryDatasetID, primaryTableCode, aliasByDatasetID)
	if leftExpr == "" || rightExpr == "" {
		return "", myerrors.NewValidationError("报表数据集关联字段不合法")
	}
	joinType := "LEFT"
	if strings.EqualFold(strings.TrimSpace(join.JoinType), "inner") {
		joinType = "INNER"
	}
	return fmt.Sprintf("%s JOIN %s AS %s ON %s = %s", joinType, quoteReportIdentifier(targetTable.TableCode), quoteReportIdentifier(targetAlias), leftExpr, rightExpr), nil
}

func reportJoinTargetDatasetID(join reportconfig.DatasetJoin, primaryDatasetID string) string {
	leftID := strings.TrimSpace(join.LeftDatasetId)
	rightID := strings.TrimSpace(join.RightDatasetId)
	if rightID == primaryDatasetID {
		return leftID
	}
	if leftID == primaryDatasetID {
		return rightID
	}
	return rightID
}

func reportDatasetFieldExpr(datasetID string, fieldCode string, primaryDatasetID string, primaryTableCode string, aliasByDatasetID map[string]string) string {
	fieldCode = strings.TrimSpace(fieldCode)
	if !isSafeReportIdentifier(fieldCode) {
		return ""
	}
	tableExpr := primaryTableCode
	if datasetID != primaryDatasetID {
		tableExpr = aliasByDatasetID[datasetID]
	}
	if !isSafeReportIdentifier(tableExpr) {
		return ""
	}
	return quoteReportIdentifier(tableExpr) + "." + quoteReportIdentifier(fieldCode)
}

func reportJoinedPreviewSelections(config reportconfig.Config, primaryDatasetID string, primaryTable model.SysTable, tableByDatasetID map[string]model.SysTable, aliasByDatasetID map[string]string) ([]reportJoinedSelection, []response.ReportPreviewColumn, error) {
	selections := make([]reportJoinedSelection, 0)
	seen := make(map[string]struct{})
	for _, cell := range config.Layout.Sheet.Cells {
		binding := cell.Binding
		if strings.TrimSpace(binding.Field) == "" {
			continue
		}
		switch strings.TrimSpace(binding.Type) {
		case "", "field", "group", "sum", "count":
		default:
			continue
		}
		datasetID := strings.TrimSpace(binding.DatasetId)
		if datasetID == "" {
			datasetID = primaryDatasetID
		}
		table, ok := tableByDatasetID[datasetID]
		if !ok {
			continue
		}
		field, ok := reportFindTableField(table, binding.Field)
		if !ok {
			continue
		}
		alias := reportColumnAlias(datasetID, field.FieldCode)
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		expr := reportDatasetFieldExpr(datasetID, field.FieldCode, primaryDatasetID, primaryTable.TableCode, aliasByDatasetID)
		if expr == "" {
			return nil, nil, myerrors.NewValidationError("报表绑定字段不合法")
		}
		label := strings.TrimSpace(cell.Value)
		if label == "" {
			label = field.FieldName
		}
		if label == "" {
			label = field.FieldCode
		}
		selections = append(selections, reportJoinedSelection{Expr: expr, Alias: alias, Label: label, Type: fmt.Sprintf("%d", field.FieldType), Role: strings.ToLower(strings.TrimSpace(binding.Type)), Field: field, DatasetID: datasetID})
	}
	if len(selections) == 0 {
		for _, column := range reportDataSourceColumns(primaryTable) {
			field, ok := reportFindTableField(primaryTable, column.Field)
			if !ok {
				continue
			}
			alias := reportColumnAlias(primaryDatasetID, column.Field)
			expr := reportDatasetFieldExpr(primaryDatasetID, column.Field, primaryDatasetID, primaryTable.TableCode, aliasByDatasetID)
			if expr == "" {
				return nil, nil, myerrors.NewValidationError("报表绑定字段不合法")
			}
			selections = append(selections, reportJoinedSelection{Expr: expr, Alias: alias, Label: column.Label, Type: column.Type, Field: field, DatasetID: primaryDatasetID})
		}
	}
	columns := make([]response.ReportPreviewColumn, 0, len(selections))
	for _, selection := range selections {
		columns = append(columns, response.ReportPreviewColumn{Name: selection.Label, Field: selection.Alias, Label: selection.Label, Type: selection.Type})
	}
	return selections, columns, nil
}

func reportSelectExprs(selections []reportJoinedSelection) []string {
	items := make([]string, 0, len(selections))
	for _, selection := range selections {
		items = append(items, fmt.Sprintf("%s AS %s", selection.Expr, quoteReportIdentifier(selection.Alias)))
	}
	return items
}

func reportApplyJoinedParameters(query *gorm.DB, config reportconfig.Config, primaryDatasetID string, primaryTableCode string, tableByDatasetID map[string]model.SysTable, aliasByDatasetID map[string]string, values map[string]any) (*gorm.DB, error) {
	for _, param := range config.Parameters() {
		value, ok := reportParameterRuntimeValue(param, values)
		if !ok {
			continue
		}
		datasetID := strings.TrimSpace(param.DatasetId)
		if datasetID == "" {
			datasetID = primaryDatasetID
		}
		table, ok := tableByDatasetID[datasetID]
		if !ok {
			return query, myerrors.NewValidationError("报表参数绑定数据集不存在")
		}
		field, ok := reportFindTableField(table, param.Field)
		if !ok {
			return query, myerrors.NewValidationError("报表参数绑定字段不存在")
		}
		fieldExpr := reportDatasetFieldExpr(datasetID, field.FieldCode, primaryDatasetID, primaryTableCode, aliasByDatasetID)
		if fieldExpr == "" {
			return query, myerrors.NewValidationError("报表参数绑定字段不合法")
		}
		switch strings.ToLower(strings.TrimSpace(param.Operator)) {
		case "", "eq":
			query = query.Where(fmt.Sprintf("%s = ?", fieldExpr), value)
		case "like":
			query = query.Where(fmt.Sprintf("%s ILIKE ?", reportTextSearchExpr(fieldExpr)), "%"+fmt.Sprint(value)+"%")
		case "between":
			items, ok := reportRangeValues(value)
			if !ok {
				return query, myerrors.NewValidationError("报表区间参数必须传入两个值")
			}
			query = query.Where(fmt.Sprintf("%s BETWEEN ? AND ?", fieldExpr), items[0], items[1])
		case "gte":
			query = query.Where(fmt.Sprintf("%s >= ?", fieldExpr), value)
		case "lte":
			query = query.Where(fmt.Sprintf("%s <= ?", fieldExpr), value)
		default:
			return query, myerrors.NewValidationError("报表参数操作符不支持")
		}
	}
	return query, nil
}

func reportApplyJoinedQuickSearch(query *gorm.DB, keyword string, selections []reportJoinedSelection) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return query
	}
	conditions := make([]string, 0)
	args := make([]any, 0)
	for _, selection := range selections {
		if !reportFieldLooksText(selection.Field) {
			continue
		}
		conditions = append(conditions, fmt.Sprintf("%s ILIKE ?", reportTextSearchExpr(selection.Expr)))
		args = append(args, "%"+keyword+"%")
	}
	if len(conditions) == 0 {
		return query
	}
	return query.Where("("+strings.Join(conditions, " OR ")+")", args...)
}

func reportFieldLooksText(field model.SysTableField) bool {
	typeLabel := strings.ToLower(fmt.Sprint(field.FieldType))
	return strings.Contains(typeLabel, "string") || strings.Contains(typeLabel, "text") || int(field.FieldType) == 1
}

func reportColumnAlias(datasetID string, fieldCode string) string {
	raw := strings.Trim(strings.TrimSpace(datasetID)+"__"+strings.TrimSpace(fieldCode), "_")
	if raw == "" {
		raw = strings.TrimSpace(fieldCode)
	}
	builder := strings.Builder{}
	for _, ch := range raw {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			builder.WriteRune(ch)
		} else {
			builder.WriteByte('_')
		}
	}
	result := builder.String()
	if result == "" || (result[0] >= '0' && result[0] <= '9') {
		result = "c_" + result
	}
	return result
}

func reportFindTableField(table model.SysTable, fieldCode string) (model.SysTableField, bool) {
	fieldCode = strings.TrimSpace(fieldCode)
	for _, field := range table.TableFields {
		if strings.EqualFold(strings.TrimSpace(field.FieldCode), fieldCode) {
			return field, true
		}
	}
	if strings.EqualFold(fieldCode, "id") {
		return model.SysTableField{FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType}, true
	}
	return model.SysTableField{}, false
}

func reportAppliedMenu(snapshot ReportExecutionSnapshot, requestMenuID int) int {
	if snapshot.PermissionMenuId > 0 {
		return snapshot.PermissionMenuId
	}
	return requestMenuID
}

func applyReportParameterValues(query *request.Basic, config reportconfig.Config, datasetID string, values map[string]any) error {
	if query == nil || len(config.Parameters()) == 0 {
		return nil
	}
	if query.Filters == nil {
		query.Filters = map[string]any{}
	}
	rules := make([]request.QueryRule, 0)
	for _, param := range config.Parameters() {
		if !reportParameterApplies(param, datasetID) {
			continue
		}
		value, ok := reportParameterRuntimeValue(param, values)
		if !ok {
			continue
		}
		field := strings.TrimSpace(param.Field)
		if field == "" {
			return myerrors.NewValidationError("报表参数缺少绑定字段")
		}
		switch strings.ToLower(strings.TrimSpace(param.Operator)) {
		case "", "eq":
			query.Filters[field] = value
		case "like":
			if reportParameterFieldAllowsLike(config, param) {
				rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Like, Value: value})
			} else {
				query.Filters[field] = value
			}
		case "between":
			if !reportValueIsRange(value) {
				return myerrors.NewValidationError("报表区间参数必须传入两个值")
			}
			rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Between, Value: value})
		case "gte":
			rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Gte, Value: value})
		case "lte":
			rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Lte, Value: value})
		default:
			return myerrors.NewValidationError("报表参数操作符不支持")
		}
	}
	if len(rules) > 0 {
		query.Expressions = append(query.Expressions, request.ExpressionGroup{
			Logic:  enum.And,
			Rules:  rules,
			Nested: []request.ExpressionGroup{},
		})
	}
	return nil
}

func reportParameterFieldAllowsLike(config reportconfig.Config, param reportconfig.Parameter) bool {
	field, ok := reportParameterField(config, param)
	if !ok {
		return true
	}
	role := strings.ToLower(strings.TrimSpace(field.Role))
	if role == "metric" || role == "time" {
		return false
	}
	typ := strings.ToLower(strings.TrimSpace(field.Type))
	if typ == "" {
		return true
	}
	if typ == "date" || typ == "datetime" || typ == "timestamp" || typ == "time" {
		return false
	}
	for _, token := range []string{"int", "number", "numeric", "decimal", "float", "double", "bigint", "smallint", "serial"} {
		if strings.Contains(typ, token) {
			return false
		}
	}
	return true
}

func reportParameterField(config reportconfig.Config, param reportconfig.Parameter) (reportconfig.Field, bool) {
	paramDatasetID := strings.TrimSpace(param.DatasetId)
	paramField := strings.TrimSpace(param.Field)
	if paramField == "" {
		return reportconfig.Field{}, false
	}
	for _, dataset := range config.Datasets() {
		if paramDatasetID != "" && strings.TrimSpace(dataset.Id) != paramDatasetID {
			continue
		}
		for _, field := range dataset.Fields {
			if strings.TrimSpace(field.Code) == paramField || strings.TrimSpace(field.Field) == paramField {
				return field, true
			}
		}
	}
	return reportconfig.Field{}, false
}

func reportSQLParameterWhere(config reportconfig.Config, datasetID string, values map[string]any) (string, []any, error) {
	conditions := make([]string, 0)
	args := make([]any, 0)
	for _, param := range config.Parameters() {
		if !reportParameterApplies(param, datasetID) {
			continue
		}
		value, ok := reportParameterRuntimeValue(param, values)
		if !ok {
			continue
		}
		field := strings.TrimSpace(param.Field)
		if !isSafeReportIdentifier(field) {
			return "", nil, myerrors.NewValidationError("报表 SQL 参数绑定字段不合法")
		}
		fieldExpr := quoteReportIdentifier(field)
		switch strings.ToLower(strings.TrimSpace(param.Operator)) {
		case "", "eq":
			conditions = append(conditions, fmt.Sprintf("%s = ?", fieldExpr))
			args = append(args, value)
		case "like":
			conditions = append(conditions, fmt.Sprintf("%s ILIKE ?", reportTextSearchExpr(fieldExpr)))
			args = append(args, "%"+fmt.Sprint(value)+"%")
		case "between":
			items, ok := reportRangeValues(value)
			if !ok {
				return "", nil, myerrors.NewValidationError("报表区间参数必须传入两个值")
			}
			conditions = append(conditions, fmt.Sprintf("%s BETWEEN ? AND ?", fieldExpr))
			args = append(args, items[0], items[1])
		case "gte":
			conditions = append(conditions, fmt.Sprintf("%s >= ?", fieldExpr))
			args = append(args, value)
		case "lte":
			conditions = append(conditions, fmt.Sprintf("%s <= ?", fieldExpr))
			args = append(args, value)
		default:
			return "", nil, myerrors.NewValidationError("报表参数操作符不支持")
		}
	}
	return strings.Join(conditions, " AND "), args, nil
}

func reportTextSearchExpr(fieldExpr string) string {
	return fmt.Sprintf("CAST(%s AS TEXT)", fieldExpr)
}

func reportParameterApplies(param reportconfig.Parameter, datasetID string) bool {
	paramDatasetID := strings.TrimSpace(param.DatasetId)
	return paramDatasetID == "" || datasetID == "" || paramDatasetID == datasetID
}

func reportParameterRuntimeValue(param reportconfig.Parameter, values map[string]any) (any, bool) {
	if values != nil {
		if value, ok := values[strings.TrimSpace(param.Id)]; ok && !reportValueIsEmpty(value) {
			return value, true
		}
		if value, ok := values[strings.TrimSpace(param.Field)]; ok && !reportValueIsEmpty(value) {
			return value, true
		}
	}
	if !reportValueIsEmpty(param.DefaultValue) {
		return param.DefaultValue, true
	}
	return nil, false
}

func reportValueIsEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case []string:
		return len(v) == 0
	case []int:
		return len(v) == 0
	case []int64:
		return len(v) == 0
	case []float64:
		return len(v) == 0
	default:
		return false
	}
}

func reportValueIsRange(value any) bool {
	_, ok := reportRangeValues(value)
	return ok
}

func reportRangeValues(value any) ([]any, bool) {
	switch v := value.(type) {
	case []any:
		if len(v) != 2 || reportValueIsEmpty(v[0]) || reportValueIsEmpty(v[1]) {
			return nil, false
		}
		return v, true
	case []string:
		if len(v) != 2 || reportValueIsEmpty(v[0]) || reportValueIsEmpty(v[1]) {
			return nil, false
		}
		return []any{v[0], v[1]}, true
	default:
		return nil, false
	}
}

var reportIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isSafeReportIdentifier(value string) bool {
	return reportIdentifierPattern.MatchString(strings.TrimSpace(value))
}

func quoteReportIdentifier(value string) string {
	parts := strings.Split(strings.TrimSpace(value), ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || !isSafeReportIdentifier(part) {
			return ""
		}
		quoted = append(quoted, `"`+strings.ReplaceAll(part, `"`, `""`)+`"`)
	}
	return strings.Join(quoted, ".")
}

func normalizeReportSQLValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.In(model.AppLocation()).Format(time.DateTime)
	default:
		return value
	}
}

func normalizeReportExecutionOptions(options ReportExecutionOptions) ReportExecutionOptions {
	if options.PageSizeLimit <= 0 {
		options.PageSizeLimit = 200
	}
	if options.DefaultPageSize <= 0 {
		options.DefaultPageSize = 20
	}
	if options.DataPermissionAction == "" {
		options.DataPermissionAction = enum.ButtonActionQuery
	}
	return options
}

func normalizeReportPageSize(raw int, options ReportExecutionOptions) int {
	options = normalizeReportExecutionOptions(options)
	if raw <= 0 || raw > options.PageSizeLimit {
		return options.DefaultPageSize
	}
	return raw
}

func (s *ReportService) completeReportTableExportRows(db *gorm.DB, query request.Basic, table model.SysTable, permission repository.GeneralizationPermission, result *repository.GeneralizationListResult, maxRows int) error {
	if result == nil || maxRows <= 0 || result.Total <= len(result.Data) || result.Total > maxRows {
		return nil
	}
	pageSize := query.Num
	if pageSize <= 0 || pageSize > 5000 {
		pageSize = 5000
	}
	for len(result.Data) < result.Total {
		nextQuery := query
		nextQuery.Page = len(result.Data)/pageSize + 1
		nextQuery.Num = pageSize
		nextResult, err := s.generalizationService.QueryWithResolvedDataPermissionDB(db, &nextQuery, table, permission)
		if err != nil {
			return err
		}
		if len(nextResult.Data) == 0 {
			return myerrors.NewValidationError("报表导出查询结果不完整")
		}
		result.Data = append(result.Data, nextResult.Data...)
		if len(result.Data) > maxRows {
			return myerrors.NewValidationError("导出行数超过系统限制，请缩小查询条件后重试")
		}
	}
	if len(result.Data) > result.Total {
		result.Data = result.Data[:result.Total]
	}
	return nil
}

func reportDataPermissionOperation(action enum.SysMenuButtonEventAction) string {
	if action == enum.ButtonActionExport {
		return model.DataPermissionOperationExport
	}
	return model.DataPermissionOperationQuery
}

func normalizeReportExportMaxRows(req request.ReportExportReq) (int, error) {
	hasSnake := req.MaxRows != nil
	hasCamel := req.MaxRowsAlt != nil
	if hasSnake && hasCamel && *req.MaxRows != *req.MaxRowsAlt {
		return 0, myerrors.NewValidationError("max_rows 与 maxRows 不能同时传入不同值")
	}
	value := defaultReportExportMaxRows
	switch {
	case hasSnake:
		value = *req.MaxRows
	case hasCamel:
		value = *req.MaxRowsAlt
	}
	if value <= 0 {
		return defaultReportExportMaxRows, nil
	}
	if value > maxReportExportRows {
		return 0, myerrors.NewValidationError("导出行数超过系统最大限制")
	}
	return value, nil
}

func reportExportPreviewReq(req request.ReportExportReq) request.ReportPreviewReq {
	parameters := req.Parameters
	if len(parameters) == 0 && len(req.Params) > 0 {
		parameters = req.Params
	}
	return request.ReportPreviewReq{
		MenuId:     req.MenuId,
		DatasetId:  req.DatasetId,
		Parameters: parameters,
		Query:      req.Query,
	}
}

func buildReportCSV(columns []response.ReportPreviewColumn, rows []map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buf)
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		label := strings.TrimSpace(column.Label)
		if label == "" {
			label = strings.TrimSpace(column.Name)
		}
		if label == "" {
			label = strings.TrimSpace(column.Field)
		}
		header = append(header, safeReportCSVCell(label))
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, row := range rows {
		record := make([]string, 0, len(columns))
		for _, column := range columns {
			record = append(record, safeReportCSVCell(row[column.Field]))
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func safeReportCSVCell(value interface{}) string {
	if value == nil {
		return ""
	}
	var text string
	switch v := value.(type) {
	case string:
		text = v
	case []byte:
		text = string(v)
	default:
		return fmt.Sprint(value)
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + text
	default:
		return text
	}
}

func reportExportFileName(snapshot ReportExecutionSnapshot) string {
	code := strings.TrimSpace(snapshot.Code)
	if code == "" {
		code = "report"
	}
	code = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, code)
	if snapshot.VersionNo > 0 {
		return fmt.Sprintf("%s_v%d.csv", code, snapshot.VersionNo)
	}
	return code + ".csv"
}

func (s *ReportService) writeExecutionLog(requestCtx context.Context, snapshot ReportExecutionSnapshot, success bool, rowCount int, start time.Time, runErr error) error {
	logCtx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), 2*time.Second)
	defer cancel()
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	subject, _ := audit.GetAuditSubject(logCtx)
	action := snapshot.RuntimeType
	if action == "" {
		action = reportRuntimeDesignPreview
	}
	params, _ := json.Marshal(map[string]any{
		"runtime": map[string]any{
			"runtime_type": action,
			"version_id":   snapshot.VersionId,
			"version_no":   snapshot.VersionNo,
		},
	})
	log := model.ReportExecutionLog{
		Basic:        model.Basic{Id: int(id), State: true},
		ReportId:     snapshot.ReportId,
		ReportCode:   snapshot.Code,
		UserId:       subject.UserID,
		UserName:     subject.UserName,
		Action:       action,
		Params:       datatypes.JSON(params),
		Success:      success,
		DurationMs:   time.Since(start).Milliseconds(),
		RowCount:     rowCount,
		ErrorMessage: "",
	}
	if runErr != nil {
		log.ErrorMessage = myerrors.SafeMessageOf(runErr)
	}
	db := s.reportLogRepo.DBWithContext(logCtx)
	if err := db.Create(&log).Error; err != nil {
		return err
	}
	if !success {
		return db.Model(&model.ReportExecutionLog{}).Where("id = ?", log.Id).Update("success", false).Error
	}
	return nil
}

func reportSnapshotFromDefinition(report model.ReportDefinition, runtimeType string) ReportExecutionSnapshot {
	return ReportExecutionSnapshot{
		ReportId:            report.Id,
		VersionId:           0,
		VersionNo:           0,
		Code:                report.Code,
		Name:                report.Name,
		SourceType:          report.SourceType,
		SourceCode:          report.SourceCode,
		PermissionMenuId:    report.PermissionMenuId,
		PermissionTableCode: report.PermissionTableCode,
		QueryConfig:         cloneReportJSON(report.QueryConfig),
		LayoutConfig:        cloneReportJSON(report.LayoutConfig),
		RuntimeType:         runtimeType,
	}
}

func reportSnapshotFromVersion(version model.ReportDefinitionVersion, runtimeType string) ReportExecutionSnapshot {
	return ReportExecutionSnapshot{
		ReportId:            version.ReportId,
		VersionId:           version.Id,
		VersionNo:           version.VersionNo,
		Code:                version.ReportCode,
		Name:                version.ReportName,
		SourceType:          version.SourceType,
		SourceCode:          version.SourceCode,
		PermissionMenuId:    version.PermissionMenuId,
		PermissionTableCode: version.PermissionTableCode,
		QueryConfig:         cloneReportJSON(version.QueryConfig),
		LayoutConfig:        cloneReportJSON(version.LayoutConfig),
		RuntimeType:         runtimeType,
	}
}

func reportDefinitionFromSnapshot(snapshot ReportExecutionSnapshot) model.ReportDefinition {
	return model.ReportDefinition{
		Basic:               model.Basic{Id: snapshot.ReportId, State: true},
		Code:                snapshot.Code,
		Name:                snapshot.Name,
		SourceType:          snapshot.SourceType,
		SourceCode:          snapshot.SourceCode,
		PermissionMenuId:    snapshot.PermissionMenuId,
		PermissionTableCode: snapshot.PermissionTableCode,
		QueryConfig:         cloneReportJSON(snapshot.QueryConfig),
		LayoutConfig:        cloneReportJSON(snapshot.LayoutConfig),
	}
}

func cloneReportJSON(raw datatypes.JSON) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON([]byte("{}"))
	}
	cloned := make([]byte, len(raw))
	copy(cloned, raw)
	return datatypes.JSON(cloned)
}

func reportUserFromContext(ctx *gin.Context) model.SysUser {
	if ctx == nil {
		return model.SysUser{}
	}
	if value, exists := ctx.Get("user"); exists {
		if user, ok := value.(model.SysUser); ok {
			return user
		}
	}
	return model.SysUser{}
}

func validateReportSQLDatasets(config reportconfig.Config) error {
	for _, dataset := range config.Datasets() {
		dataset = reportconfig.NormalizeDataset(dataset)
		if dataset.Type != reportconfig.SourceTypeSQL {
			continue
		}
		if _, err := safeReportPreviewSQL(dataset.SQL); err != nil {
			return err
		}
	}
	return nil
}

func reportConfigHasSQLDataset(config reportconfig.Config) bool {
	for _, dataset := range config.Datasets() {
		if reportconfig.NormalizeDataset(dataset).Type == reportconfig.SourceTypeSQL {
			return true
		}
	}
	return false
}

func ensureSQLDatasetRole(ctx *gin.Context, config reportconfig.Config) error {
	if !reportConfigHasSQLDataset(config) {
		return nil
	}
	user := reportUserFromContext(ctx)
	if utils.IsSuperAdmin(user) {
		return nil
	}
	return myerrors.ErrPermissionDenied
}

func normalizeReportSourceType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", reportSourceTypeTable:
		return reportSourceTypeTable
	case reportSourceTypeView:
		return reportSourceTypeView
	default:
		return ""
	}
}

func reportPreviewColumns(table model.SysTable) []response.ReportPreviewColumn {
	return reportDataSourceColumns(table)
}

func reportTableWithPreviewFields(table model.SysTable, config reportconfig.Config, datasetID string) model.SysTable {
	needed := reportPreviewFieldCodes(config, datasetID)
	if len(needed) == 0 {
		return table
	}
	hasID := false
	for index := range table.TableFields {
		code := strings.TrimSpace(table.TableFields[index].FieldCode)
		if strings.EqualFold(code, "id") {
			hasID = true
		}
		if _, ok := needed[code]; ok {
			table.TableFields[index].IsListShow = true
		}
	}
	if _, ok := needed["id"]; ok && !hasID {
		table.TableFields = append([]model.SysTableField{{
			FieldName:    "ID",
			FieldCode:    "id",
			FieldType:    enum.BigIntFieldType,
			IsPrimaryKey: true,
			IsListShow:   true,
		}}, table.TableFields...)
	}
	return table
}

func reportPreviewFieldCodes(config reportconfig.Config, datasetID string) map[string]struct{} {
	datasetID = strings.TrimSpace(datasetID)
	result := make(map[string]struct{})
	add := func(code string) {
		code = strings.TrimSpace(code)
		if code != "" && !reportFieldIsSensitive(code) {
			result[code] = struct{}{}
		}
	}
	add("id")
	for _, dataset := range config.Datasets() {
		if datasetID != "" && strings.TrimSpace(dataset.Id) != datasetID {
			continue
		}
		for _, field := range dataset.Fields {
			add(field.Code)
			add(field.Field)
		}
	}
	for _, field := range config.Query.Fields {
		add(field.Code)
		add(field.Field)
	}
	for _, param := range config.Parameters() {
		if reportParameterApplies(param, datasetID) {
			add(param.Field)
		}
	}
	for _, cell := range config.Layout.Sheet.Cells {
		if cell.Binding.Field == "" {
			continue
		}
		if datasetID == "" || strings.TrimSpace(cell.Binding.DatasetId) == "" || strings.TrimSpace(cell.Binding.DatasetId) == datasetID {
			add(cell.Binding.Field)
		}
	}
	return result
}

func reportDataSourceColumns(table model.SysTable) []response.ReportPreviewColumn {
	columns := make([]response.ReportPreviewColumn, 0, len(table.TableFields)+1)
	seen := map[string]struct{}{}
	appendField := func(field model.SysTableField) {
		code := strings.TrimSpace(field.FieldCode)
		if code == "" || reportFieldIsSensitive(code) {
			return
		}
		if _, ok := seen[code]; ok {
			return
		}
		seen[code] = struct{}{}
		columns = append(columns, response.ReportPreviewColumn{
			Name:  code,
			Field: code,
			Label: field.FieldName,
			Type:  fmt.Sprintf("%d", field.FieldType),
		})
	}
	appendField(model.SysTableField{FieldName: "ID", FieldCode: "id", FieldType: enum.BigIntFieldType})
	for _, field := range table.TableFields {
		appendField(field)
	}
	return columns
}

func reportFieldIsSensitive(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "password", "access_token", "access_tokens", "refresh_token", "salt":
		return true
	default:
		return false
	}
}

func reportPreviewColumnsFromConfig(table model.SysTable, raw datatypes.JSON) []response.ReportPreviewColumn {
	type fieldConfig struct {
		Name string `json:"name"`
		Code string `json:"code"`
		Type string `json:"type"`
	}
	var config struct {
		Fields []fieldConfig `json:"fields"`
	}
	allColumns := reportPreviewColumns(table)
	if len(raw) == 0 || json.Unmarshal(raw, &config) != nil || len(config.Fields) == 0 {
		return allColumns
	}
	columnByField := make(map[string]response.ReportPreviewColumn, len(allColumns))
	for _, column := range allColumns {
		columnByField[column.Field] = column
	}
	columns := make([]response.ReportPreviewColumn, 0, len(config.Fields))
	for _, item := range config.Fields {
		code := strings.TrimSpace(item.Code)
		if code == "" {
			continue
		}
		column, ok := columnByField[code]
		if !ok {
			continue
		}
		if strings.TrimSpace(item.Name) != "" {
			column.Label = strings.TrimSpace(item.Name)
			column.Name = column.Label
		}
		if strings.TrimSpace(item.Type) != "" {
			column.Type = strings.TrimSpace(item.Type)
		}
		columns = append(columns, column)
	}
	if len(columns) == 0 {
		return allColumns
	}
	return columns
}

func reportDatasetIdForPreview(config reportconfig.Config, selectedDataset reportconfig.Dataset) string {
	if selectedDataset.Id != "" {
		return selectedDataset.Id
	}
	if dataset, ok := config.PrimaryTableDataset(); ok {
		return dataset.Id
	}
	return ""
}

func reportPreviewDatasets(config reportconfig.Config, snapshot ReportExecutionSnapshot, columns []response.ReportPreviewColumn) []response.ReportPreviewDataset {
	datasets := reportConfigDatasetMetadata(config)
	if len(datasets) > 0 {
		return datasets
	}
	return []response.ReportPreviewDataset{
		{
			Id:         "primary",
			Name:       snapshot.Name,
			Type:       reportSourceTypeTable,
			SourceCode: snapshot.SourceCode,
			Primary:    true,
			Fields:     columns,
		},
	}
}

func reportConfigDatasetMetadata(config reportconfig.Config) []response.ReportPreviewDataset {
	datasets := config.Datasets()
	if len(datasets) == 0 {
		return nil
	}
	result := make([]response.ReportPreviewDataset, 0, len(datasets))
	for _, dataset := range datasets {
		dataset = reportconfig.NormalizeDataset(dataset)
		result = append(result, response.ReportPreviewDataset{
			Id:         dataset.Id,
			Name:       dataset.Name,
			Type:       dataset.Type,
			SourceCode: dataset.SourceCode,
			Primary:    dataset.Primary,
			Fields:     reportColumnsFromConfigFields(dataset.Fields),
		})
	}
	return result
}

func reportConfigDatasetJoins(config reportconfig.Config) []response.ReportPreviewJoin {
	joins := config.DatasetJoins()
	if len(joins) == 0 {
		return nil
	}
	result := make([]response.ReportPreviewJoin, 0, len(joins))
	for _, join := range joins {
		joinType := strings.ToLower(strings.TrimSpace(join.JoinType))
		if joinType == "" {
			joinType = "left"
		}
		result = append(result, response.ReportPreviewJoin{
			Id:             strings.TrimSpace(join.Id),
			LeftDatasetId:  strings.TrimSpace(join.LeftDatasetId),
			LeftField:      strings.TrimSpace(join.LeftField),
			RightDatasetId: strings.TrimSpace(join.RightDatasetId),
			RightField:     strings.TrimSpace(join.RightField),
			JoinType:       joinType,
		})
	}
	return result
}

func reportColumnsFromConfigFields(fields []reportconfig.Field) []response.ReportPreviewColumn {
	columns := make([]response.ReportPreviewColumn, 0, len(fields))
	for _, field := range fields {
		code := strings.TrimSpace(field.Code)
		if code == "" {
			code = strings.TrimSpace(field.Field)
		}
		if code == "" {
			continue
		}
		label := strings.TrimSpace(field.Name)
		if label == "" {
			label = strings.TrimSpace(field.Label)
		}
		if label == "" {
			label = code
		}
		columns = append(columns, response.ReportPreviewColumn{
			Name:  label,
			Field: code,
			Label: label,
			Type:  strings.TrimSpace(field.Type),
		})
	}
	return columns
}

func filterReportRows(rows []map[string]interface{}, columns []response.ReportPreviewColumn) []map[string]interface{} {
	if len(rows) == 0 || len(columns) == 0 {
		return rows
	}
	filtered := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		item := make(map[string]interface{}, len(columns)+1)
		if value, ok := row["id"]; ok {
			item["id"] = value
		}
		for _, column := range columns {
			item[column.Field] = row[column.Field]
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func reportTableTypeLabel(tableType enum.SysTableType) string {
	switch tableType {
	case enum.View:
		return reportSourceTypeView
	default:
		return reportSourceTypeTable
	}
}

type reportMenuOption struct {
	ReportId   int    `json:"report_id"`
	ReportCode string `json:"report_code"`
}

type reportMenuPolicy struct {
	RoleID   int
	RoleName string
	Path     string
	Method   string
}

func (s *ReportService) validateReportMenuParent(tx *gorm.DB, parentID int) (model.SysMenu, error) {
	if parentID <= 0 {
		return model.SysMenu{}, myerrors.NewValidationError("请选择发布父级菜单")
	}
	var menu model.SysMenu
	if err := tx.First(&menu, parentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysMenu{}, myerrors.NewValidationError("发布父级菜单不存在")
		}
		return model.SysMenu{}, err
	}
	if !isReportPublishParentMenu(menu) {
		return model.SysMenu{}, myerrors.NewValidationError("报表只能发布到目录菜单下")
	}
	return menu, nil
}

func isReportPublishParentMenu(menu model.SysMenu) bool {
	if menu.IsHidden || !menu.State || strings.TrimSpace(menu.TableCode) != "" {
		return false
	}
	if menu.PageType == "" {
		return true
	}
	return menu.PageType == enum.MenuPageTypeDirectory
}

func (s *ReportService) findReportMenuForUpdate(tx *gorm.DB, report model.ReportDefinition) (model.SysMenu, error) {
	if report.PermissionMenuId > 0 {
		var menu model.SysMenu
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&menu, report.PermissionMenuId).Error
		if err == nil {
			return menu, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysMenu{}, err
		}
	}
	var menus []model.SysMenu
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("page_type = ?", enum.MenuPageTypeReport).
		Find(&menus).Error; err != nil {
		return model.SysMenu{}, err
	}
	for _, menu := range menus {
		option, ok := parseReportMenuOption(menu.Option)
		if ok && option.ReportId == report.Id {
			return menu, nil
		}
	}
	return model.SysMenu{}, gorm.ErrRecordNotFound
}

func parseReportMenuOption(raw string) (reportMenuOption, bool) {
	var option reportMenuOption
	if strings.TrimSpace(raw) == "" {
		return option, false
	}
	if err := json.Unmarshal([]byte(raw), &option); err != nil {
		return reportMenuOption{}, false
	}
	return option, option.ReportId > 0
}

func (s *ReportService) ensureReportMenuPathAvailable(tx *gorm.DB, path string, currentMenuID int) error {
	var existing model.SysMenu
	query := tx.Where("path = ?", path)
	if currentMenuID > 0 {
		query = query.Where("id <> ?", currentMenuID)
	}
	err := query.First(&existing).Error
	if err == nil {
		return myerrors.NewValidationError("菜单路径已被占用")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func (s *ReportService) upsertReportMenu(tx *gorm.DB, report model.ReportDefinition, parentID int, existing model.SysMenu, req request.ReportPublishMenuReq, path string) (model.SysMenu, error) {
	optionBytes, err := json.Marshal(reportMenuOption{ReportId: report.Id, ReportCode: report.Code})
	if err != nil {
		return model.SysMenu{}, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(report.Name)
	}
	icon := strings.TrimSpace(req.Icon)
	if icon == "" {
		icon = reportDefaultMenuIcon
	}
	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}
	menu := model.SysMenu{
		Basic:     model.Basic{Id: existing.Id, State: true},
		Pid:       parentID,
		Name:      reportMenuName(report.Code),
		Path:      path,
		Component: reportRuntimeComponent,
		Title:     title,
		IsHidden:  !visible,
		Sequence:  normalizeReportMenuSequence(req.Sort, report.Id),
		PageType:  enum.MenuPageTypeReport,
		TableCode: strings.TrimSpace(report.PermissionTableCode),
		Option:    string(optionBytes),
		Icon:      utils.StringPtr(icon),
	}
	if existing.Id > 0 {
		updates := map[string]any{
			"pid":        menu.Pid,
			"name":       menu.Name,
			"path":       menu.Path,
			"component":  menu.Component,
			"title":      menu.Title,
			"is_hidden":  menu.IsHidden,
			"sequence":   menu.Sequence,
			"page_type":  menu.PageType,
			"table_code": menu.TableCode,
			"option":     menu.Option,
			"icon":       menu.Icon,
			"state":      true,
			"gmt_delete": nil,
			"gmt_modify": model.Now(),
		}
		if err := tx.Model(&model.SysMenu{}).Unscoped().Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return model.SysMenu{}, err
		}
		menu.Basic.Id = existing.Id
		return menu, nil
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return model.SysMenu{}, err
	}
	menu.Basic.Id = int(id)
	if err := s.sysMenuRepo.Create(tx, &menu); err != nil {
		return model.SysMenu{}, err
	}
	return menu, nil
}

func normalizeReportMenuPath(path string, reportCode string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	if path != "" {
		return path
	}
	return "report/runtime/" + strings.TrimSpace(reportCode)
}

func reportMenuName(reportCode string) string {
	raw := strings.TrimSpace(reportCode)
	name := "report_" + raw
	if len(name) <= 32 {
		return name
	}
	hash := shortHash(raw)
	prefixLen := 32 - len("report_") - len("_") - len(hash)
	if prefixLen < 1 {
		prefixLen = 1
	}
	prefix := raw
	if len(prefix) > prefixLen {
		prefix = prefix[:prefixLen]
	}
	return "report_" + prefix + "_" + hash
}

func reportMenuButtonCode(reportCode string, action string) string {
	raw := strings.TrimSpace(reportCode)
	suffix := "_" + strings.TrimSpace(action)
	code := raw + suffix
	if len(code) <= 128 {
		return code
	}
	hash := shortHash(raw + suffix)
	prefixLen := 128 - len(suffix) - len("_") - len(hash)
	if prefixLen < 1 {
		prefixLen = 1
	}
	if len(raw) > prefixLen {
		raw = raw[:prefixLen]
	}
	return raw + "_" + hash + suffix
}

func shortHash(value string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return fmt.Sprintf("%08x", hasher.Sum32())
}

func normalizeReportMenuSequence(sortValue int, reportID int) uint8 {
	if sortValue < 0 {
		return 0
	}
	if sortValue > 255 {
		return 255
	}
	if sortValue > 0 {
		return uint8(sortValue)
	}
	return uint8(30 + (reportID % 100))
}

func (s *ReportService) ensureReportRuntimeButtons(tx *gorm.DB, report model.ReportDefinition, menuID int) ([]model.SysMenuButton, error) {
	defaults := reportRuntimeDefaultButtons(report.Code)
	buttons := make([]model.SysMenuButton, 0, len(defaults))
	for _, item := range defaults {
		button, err := s.sysMenuButtonRepo.FindByMenuIdAndCode(tx, menuID, item.Code)
		if err == nil {
			updates := map[string]any{
				"name":         item.Name,
				"memo":         item.Memo,
				"position":     item.Position,
				"event_type":   item.EventType,
				"event_action": item.EventAction,
				"icon":         item.Icon,
				"color":        item.Color,
				"display_mode": item.DisplayMode,
				"sequence":     item.Sequence,
				"path":         item.Path,
				"method":       strings.ToUpper(item.Method),
				"is_button":    item.IsButton,
				"is_hidden":    item.IsHidden,
				"is_disabled":  false,
				"state":        true,
				"gmt_modify":   model.Now(),
			}
			if err := s.sysMenuButtonRepo.UpdateMenuButtonFields(tx, button.Id, updates); err != nil {
				return nil, err
			}
			item.Basic.Id = button.Id
			item.MenuId = menuID
			buttons = append(buttons, item)
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
		buttons = append(buttons, item)
	}
	return buttons, nil
}

func reportRuntimeDefaultButtons(reportCode string) []model.SysMenuButton {
	return []model.SysMenuButton{
		{
			Name:        "详情",
			Code:        reportMenuButtonCode(reportCode, "detail"),
			Position:    enum.Top,
			EventAction: string(enum.ButtonActionDetail),
			Sequence:    90,
			Path:        "/admin/report/:id",
			Method:      "GET",
			IsButton:    false,
			IsHidden:    true,
		},
		{
			Name:        "查询",
			Code:        reportMenuButtonCode(reportCode, "query"),
			Position:    enum.Top,
			EventAction: string(enum.ButtonActionQuery),
			Icon:        "search",
			Color:       "primary",
			DisplayMode: enum.ButtonDisplayIconText,
			Sequence:    0,
			Path:        "/admin/report/:id/run",
			Method:      "POST",
			IsButton:    true,
		},
		{
			Name:        "导出",
			Code:        reportMenuButtonCode(reportCode, "export"),
			Position:    enum.Top,
			EventAction: string(enum.ButtonActionExport),
			Icon:        "file_download",
			Color:       "primary",
			DisplayMode: enum.ButtonDisplayIconText,
			Sequence:    1,
			Path:        "/admin/report/:id/export",
			Method:      "POST",
			IsButton:    true,
		},
		{
			Name:        "刷新",
			Code:        reportMenuButtonCode(reportCode, "refresh"),
			Position:    enum.Top,
			EventAction: string(enum.ButtonActionRefresh),
			Icon:        "refresh",
			Color:       "primary",
			DisplayMode: enum.ButtonDisplayIconText,
			Sequence:    2,
			IsButton:    true,
		},
	}
}

func (s *ReportService) grantReportMenuRoles(tx *gorm.DB, menuID int, buttons []model.SysMenuButton, permissionRoleIDs []int) ([]reportMenuPolicy, error) {
	roles, err := s.reportMenuGrantRoles(permissionRoleIDs)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, nil
	}
	policies := make([]reportMenuPolicy, 0, len(roles)*len(buttons))
	for _, role := range roles {
		if err := s.sysRoleMenuRepo.CreateIfNotExists(tx, model.SysRoleMenu{RoleId: role.Id, MenuId: menuID}); err != nil {
			return nil, err
		}
		for _, button := range buttons {
			if err := s.sysRoleMenuButtonRepo.CreateIfNotExists(tx, model.SysRoleMenuButton{RoleId: role.Id, MenuId: menuID, ButtonId: button.Id}); err != nil {
				return nil, err
			}
			path := strings.TrimSpace(button.Path)
			method := strings.ToUpper(strings.TrimSpace(button.Method))
			if path == "" || method == "" {
				continue
			}
			policies = append(policies, reportMenuPolicy{RoleID: role.Id, RoleName: role.Name, Path: path, Method: method})
		}
	}
	return uniqueReportMenuPolicies(policies), nil
}

func (s *ReportService) reportMenuGrantRoles(permissionRoleIDs []int) ([]model.SysRole, error) {
	roleMap := make(map[int]model.SysRole)
	superAdmin, err := s.sysRoleRepo.FindByField("name", reportSuperAdminRoleName)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if superAdmin.Id > 0 {
		roleMap[superAdmin.Id] = superAdmin
	}
	for _, roleID := range permissionRoleIDs {
		if roleID <= 0 {
			continue
		}
		if _, ok := roleMap[roleID]; ok {
			continue
		}
		role, err := s.sysRoleRepo.FindById(roleID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, myerrors.NewValidationError("授权角色不存在")
			}
			return nil, err
		}
		roleMap[role.Id] = role
	}
	ids := make([]int, 0, len(roleMap))
	for id := range roleMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	roles := make([]model.SysRole, 0, len(ids))
	for _, id := range ids {
		roles = append(roles, roleMap[id])
	}
	return roles, nil
}

func uniqueReportMenuPolicies(policies []reportMenuPolicy) []reportMenuPolicy {
	seen := make(map[string]struct{}, len(policies))
	result := make([]reportMenuPolicy, 0, len(policies))
	for _, policy := range policies {
		if policy.RoleName == "" || policy.Path == "" || policy.Method == "" {
			continue
		}
		key := fmt.Sprintf("%s|%s|%s", policy.RoleName, policy.Path, policy.Method)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, policy)
	}
	return result
}

func (s *ReportService) reportMenuButtons(tx *gorm.DB, menuID int) ([]model.SysMenuButton, error) {
	var buttons []model.SysMenuButton
	if err := tx.Where("menu_id = ?", menuID).Find(&buttons).Error; err != nil {
		return nil, err
	}
	return buttons, nil
}

func (s *ReportService) reportButtonPolicyCandidates(tx *gorm.DB, buttons []model.SysMenuButton) (map[buttonPolicyKey]struct{}, error) {
	candidates := make(map[buttonPolicyKey]struct{})
	for _, button := range buttons {
		path := strings.TrimSpace(button.Path)
		method := strings.ToUpper(strings.TrimSpace(button.Method))
		if path == "" || method == "" {
			continue
		}
		var roleButtons []model.SysRoleMenuButton
		if err := tx.Where("button_id = ?", button.Id).Find(&roleButtons).Error; err != nil {
			return nil, err
		}
		for _, roleButton := range roleButtons {
			candidates[buttonPolicyKey{RoleID: roleButton.RoleId, Path: path, Method: method}] = struct{}{}
		}
	}
	return candidates, nil
}

func (s *ReportService) reportOrphanRolePolicyCleanups(tx *gorm.DB, candidates map[buttonPolicyKey]struct{}) ([]reportMenuPolicy, error) {
	cleanups := make([]reportMenuPolicy, 0, len(candidates))
	for candidate := range candidates {
		remaining, err := s.sysRoleMenuButtonRepo.CountActiveButtonPolicy(tx, candidate.RoleID, candidate.Path, candidate.Method)
		if err != nil {
			return nil, err
		}
		if remaining > 0 {
			continue
		}
		var role model.SysRole
		if err := tx.First(&role, candidate.RoleID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		cleanups = append(cleanups, reportMenuPolicy{RoleID: role.Id, RoleName: role.Name, Path: candidate.Path, Method: candidate.Method})
	}
	return cleanups, nil
}

func reportMenuResponse(report model.ReportDefinition, menu model.SysMenu, published bool) response.ReportPublishMenuRes {
	return response.ReportPublishMenuRes{
		ReportId:        report.Id,
		ReportCode:      report.Code,
		MenuId:          menu.Id,
		MenuName:        menu.Name,
		MenuTitle:       menu.Title,
		Path:            menu.Path,
		Component:       menu.Component,
		PageType:        string(menu.PageType),
		Visible:         !menu.IsHidden && menu.State,
		PublishedToMenu: published && !menu.IsHidden && menu.State,
	}
}
