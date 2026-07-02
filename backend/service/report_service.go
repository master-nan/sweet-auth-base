package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/reportconfig"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	queryutil "backend/repository/util"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	reportSourceTypeTable = "table"
	reportSourceTypeView  = "view"
)

var reportSQLForbiddenPattern = regexp.MustCompile(`(?i)\b(insert|update|delete|merge|truncate|drop|alter|create|grant|revoke|replace|call|execute|exec|copy|vacuum|reindex|attach|detach|pragma)\b`)

type ReportService struct {
	reportRepo            repository.ReportDefinitionRepository
	reportLogRepo         repository.ReportExecutionLogRepository
	generalizationService *GeneralizationService
	sysTableService       *SysTableService
	dataPermissionService *DataPermissionService
	sf                    *utils.Snowflake
}

func NewReportService(
	reportRepo repository.ReportDefinitionRepository,
	reportLogRepo repository.ReportExecutionLogRepository,
	generalizationService *GeneralizationService,
	sysTableService *SysTableService,
	dataPermissionService *DataPermissionService,
	sf *utils.Snowflake,
) *ReportService {
	return &ReportService{
		reportRepo:            reportRepo,
		reportLogRepo:         reportLogRepo,
		generalizationService: generalizationService,
		sysTableService:       sysTableService,
		dataPermissionService: dataPermissionService,
		sf:                    sf,
	}
}

func (s *ReportService) GetReportDefinitionList(basic *request.Basic, table model.SysTable) (response.ListResult[model.ReportDefinition], error) {
	return s.reportRepo.GetReportDefinitionList(basic, table)
}

func (s *ReportService) GetReportDefinitionById(id int) (model.ReportDefinition, error) {
	report, err := s.reportRepo.FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ReportDefinition{}, nil
		}
		return model.ReportDefinition{}, err
	}
	return report, nil
}

func (s *ReportService) GetDataSources() ([]response.ReportDataSourceRes, error) {
	query := &request.Basic{
		Page:      1,
		Num:       500,
		TableCode: "sys_table",
		Order: request.Order{
			Field: "gmt_modify",
			IsAsc: false,
		},
	}
	result, err := s.sysTableService.GetTableList(query)
	if err != nil {
		return nil, err
	}
	items := make([]response.ReportDataSourceRes, 0, len(result.Data))
	for _, table := range result.Data {
		fullTable, err := s.sysTableService.GetTableByTableCode(table.TableCode)
		if err != nil {
			return nil, err
		}
		if fullTable.Id == 0 {
			continue
		}
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

func (s *ReportService) InferSQLFields(ctx *gin.Context, req request.ReportSQLFieldsReq) ([]response.ReportPreviewColumn, error) {
	if s.reportRepo == nil {
		return nil, myerrors.NewBadRequestError("报表数据仓储未初始化")
	}
	sqlText, err := safeReportPreviewSQL(req.SQL)
	if err != nil {
		return nil, err
	}
	wrapped := fmt.Sprintf("SELECT * FROM (%s) AS report_sql_dataset_fields LIMIT 0", sqlText)
	sqlRows, err := s.reportRepo.DBWithContext(ctx).Raw(wrapped).Rows()
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()
	return reportSQLColumns(sqlRows)
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
		return 0, myerrors.NewBadRequestError("报表编码已存在")
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
		return myerrors.NewBadRequestError("报表编码已存在")
	}
	return s.reportRepo.DBWithContext(ctx).Model(&model.ReportDefinition{}).Where("id = ?", req.Id).Updates(map[string]any{
		"code":                  report.Code,
		"name":                  report.Name,
		"description":           report.Description,
		"category":              report.Category,
		"source_type":           report.SourceType,
		"source_code":           report.SourceCode,
		"permission_menu_id":    report.PermissionMenuId,
		"permission_table_code": report.PermissionTableCode,
		"query_config":          report.QueryConfig,
		"layout_config":         report.LayoutConfig,
		"remark":                report.Remark,
		"state":                 report.State,
	}).Error
}

func (s *ReportService) DeleteReportDefinitionById(ctx *gin.Context, id int) error {
	return s.reportRepo.DeleteById(s.reportRepo.DBWithContext(ctx), id)
}

func (s *ReportService) Preview(ctx *gin.Context, reportId int, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	start := time.Now()
	report, err := s.GetReportDefinitionById(reportId)
	if err != nil {
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if report.Id == 0 {
		err = myerrors.ErrDataNotFound
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if !report.State {
		err = myerrors.NewBadRequestError("报表已停用")
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	config, err := reportconfig.Parse(report.QueryConfig, report.LayoutConfig)
	if err != nil {
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	var selectedDataset reportconfig.Dataset
	if strings.TrimSpace(req.DatasetId) != "" {
		var ok bool
		selectedDataset, ok = config.DatasetByID(req.DatasetId)
		if !ok {
			err = myerrors.NewBadRequestError("报表数据集不存在")
			_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
			return response.ReportPreviewRes{}, err
		}
		if selectedDataset.Type == reportconfig.SourceTypeSQL {
			preview, err := s.previewSQLDataset(ctx, report, config, selectedDataset, req)
			if err != nil {
				_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
				return response.ReportPreviewRes{}, err
			}
			_ = s.writeExecutionLog(ctx, report, "preview", req, true, len(preview.Rows), start, nil)
			return preview, nil
		}
	}
	if selectedDataset.Id == "" && reportShouldUseJoinedPreview(config) {
		preview, err := s.previewJoinedTableDatasets(ctx, report, config, req)
		if err != nil {
			_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
			return response.ReportPreviewRes{}, err
		}
		_ = s.writeExecutionLog(ctx, report, "preview", req, true, len(preview.Rows), start, nil)
		return preview, nil
	}
	activeDatasetID := reportDatasetIdForPreview(config, selectedDataset)
	sourceTable, permissionTable, err := s.resolveReportPreviewTable(report, selectedDataset)
	if err != nil {
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	query := req.Query
	if err := applyReportParameterValues(&query, config, activeDatasetID, req.Parameters); err != nil {
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Num <= 0 || query.Num > 200 {
		query.Num = 20
	}
	query.TableCode = sourceTable.TableCode
	query.MenuId = req.MenuId
	if report.PermissionMenuId > 0 {
		query.MenuId = report.PermissionMenuId
	}
	if err := s.injectReportDataScope(ctx, &query, permissionTable); err != nil {
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	result, err := s.generalizationService.Query(&query, sourceTable)
	if err != nil {
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	columns := reportPreviewColumnsFromConfig(sourceTable, report.QueryConfig)
	preview := response.ReportPreviewRes{
		Columns: columns,
		Rows:    filterReportRows(result.Data, columns),
		Total:   result.Total,
		Meta: response.ReportPreviewMeta{
			ReportId:    report.Id,
			ReportCode:  report.Code,
			SourceCode:  sourceTable.TableCode,
			DatasetId:   activeDatasetID,
			DatasetType: reportSourceTypeTable,
			AppliedMenu: query.MenuId,
		},
		Datasets: reportPreviewDatasets(config, report, columns),
		Joins:    reportConfigDatasetJoins(config),
	}
	_ = s.writeExecutionLog(ctx, report, "preview", req, true, len(result.Data), start, nil)
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
	if report.Code == "" || report.Name == "" {
		return report, myerrors.ErrParamInvalid
	}
	if len(report.QueryConfig) == 0 {
		report.QueryConfig = datatypes.JSON([]byte("{}"))
	}
	if len(report.LayoutConfig) == 0 {
		report.LayoutConfig = datatypes.JSON([]byte("{}"))
	}
	report.State = true
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
	if req.Id <= 0 || report.Code == "" || report.Name == "" {
		return report, myerrors.ErrParamInvalid
	}
	if len(report.QueryConfig) == 0 {
		report.QueryConfig = datatypes.JSON([]byte("{}"))
	}
	if len(report.LayoutConfig) == 0 {
		report.LayoutConfig = datatypes.JSON([]byte("{}"))
	}
	if err := s.applyReportPrimaryDataset(&report); err != nil {
		return report, err
	}
	return report, s.validateReportTables(report)
}

func (s *ReportService) applyReportPrimaryDataset(report *model.ReportDefinition) error {
	config, err := reportconfig.Parse(report.QueryConfig, report.LayoutConfig)
	if err != nil {
		return err
	}
	if config.HasDatasets() {
		dataset, ok := config.PrimaryTableDataset()
		if !ok {
			return myerrors.NewBadRequestError("报表必须配置 primary table 数据集")
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
		return myerrors.NewBadRequestError("报表数据源类型不合法")
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
	if normalizeReportSourceType(report.SourceType) == "" {
		return model.SysTable{}, model.SysTable{}, myerrors.NewBadRequestError("报表数据源类型不合法")
	}
	sourceTable, err := s.sysTableService.GetTableByTableCode(strings.TrimSpace(report.SourceCode))
	if err != nil {
		return model.SysTable{}, model.SysTable{}, err
	}
	if sourceTable.Id == 0 {
		return model.SysTable{}, model.SysTable{}, myerrors.NewBadRequestError("报表数据源表不存在")
	}
	permissionTable := sourceTable
	if strings.TrimSpace(report.PermissionTableCode) != "" && report.PermissionTableCode != sourceTable.TableCode {
		return model.SysTable{}, model.SysTable{}, myerrors.NewBadRequestError("报表权限表暂仅支持与数据源表一致")
	}
	return sourceTable, permissionTable, nil
}

func (s *ReportService) resolveReportPreviewTable(report model.ReportDefinition, selectedDataset reportconfig.Dataset) (model.SysTable, model.SysTable, error) {
	if selectedDataset.Id == "" {
		return s.resolveReportTables(report)
	}
	sourceTable, err := s.sysTableService.GetTableByTableCode(strings.TrimSpace(selectedDataset.SourceCode))
	if err != nil {
		return model.SysTable{}, model.SysTable{}, err
	}
	if sourceTable.Id == 0 {
		return model.SysTable{}, model.SysTable{}, myerrors.NewBadRequestError("报表数据集表不存在")
	}
	return sourceTable, sourceTable, nil
}

func (s *ReportService) validateReportDatasetTables(config reportconfig.Config) error {
	if s.sysTableService == nil {
		return myerrors.NewBadRequestError("报表表结构服务未初始化")
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
		table, err := s.sysTableService.GetTableByTableCode(dataset.SourceCode)
		if err != nil {
			return err
		}
		if table.Id == 0 {
			return myerrors.NewBadRequestError(fmt.Sprintf("报表数据集表不存在: %s", dataset.SourceCode))
		}
	}
	for _, join := range config.DatasetJoins() {
		left := datasetByID[strings.TrimSpace(join.LeftDatasetId)]
		right := datasetByID[strings.TrimSpace(join.RightDatasetId)]
		if left.Type != reportconfig.SourceTypeTable || right.Type != reportconfig.SourceTypeTable {
			return myerrors.NewBadRequestError("报表数据集关联暂仅支持表数据集")
		}
	}
	return nil
}

func (s *ReportService) previewSQLDataset(ctx *gin.Context, report model.ReportDefinition, config reportconfig.Config, dataset reportconfig.Dataset, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	sqlText, err := safeReportPreviewSQL(dataset.SQL)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	whereClause, args, err := reportSQLParameterWhere(config, dataset.Id, req.Parameters)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	limit := req.Query.Num
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	rows, columns, err := s.queryReportSQL(ctx, sqlText, whereClause, args, limit)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	return response.ReportPreviewRes{
		Columns:  columns,
		Rows:     rows,
		Total:    len(rows),
		Datasets: reportConfigDatasetMetadata(config),
		Joins:    reportConfigDatasetJoins(config),
		Meta: response.ReportPreviewMeta{
			ReportId:    report.Id,
			ReportCode:  report.Code,
			SourceCode:  report.SourceCode,
			DatasetId:   dataset.Id,
			DatasetType: reportconfig.SourceTypeSQL,
			AppliedMenu: req.MenuId,
		},
	}, nil
}

func (s *ReportService) previewJoinedTableDatasets(ctx *gin.Context, report model.ReportDefinition, config reportconfig.Config, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	primaryDataset, ok := config.PrimaryTableDataset()
	if !ok {
		return response.ReportPreviewRes{}, myerrors.NewBadRequestError("报表必须配置 primary table 数据集")
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
		table, err := s.sysTableService.GetTableByTableCode(dataset.SourceCode)
		if err != nil {
			return response.ReportPreviewRes{}, err
		}
		if table.Id == 0 {
			return response.ReportPreviewRes{}, myerrors.NewBadRequestError(fmt.Sprintf("报表数据集表不存在: %s", dataset.SourceCode))
		}
		tableByDatasetID[dataset.Id] = table
	}
	primaryTable, ok := tableByDatasetID[primaryDataset.Id]
	if !ok {
		return response.ReportPreviewRes{}, myerrors.NewBadRequestError("报表主数据集表不存在")
	}

	aliasByDatasetID := reportDatasetAliases(config, primaryDataset.Id, primaryTable.TableCode)
	query := s.reportRepo.DBWithContext(ctx).Table(quoteReportIdentifier(primaryTable.TableCode))
	if _, ok := reportFindTableField(primaryTable, "gmt_delete"); ok {
		query = query.Where(fmt.Sprintf("%s IS NULL", reportDatasetFieldExpr(primaryDataset.Id, "gmt_delete", primaryDataset.Id, primaryTable.TableCode, aliasByDatasetID)))
	}
	if err := s.applyJoinedReportDataScope(ctx, &query, req, report, primaryTable); err != nil {
		return response.ReportPreviewRes{}, err
	}
	joinedDatasetIDs := make(map[string]struct{})
	for _, join := range config.DatasetJoins() {
		targetID := reportJoinTargetDatasetID(join, primaryDataset.Id)
		if targetID != "" {
			if _, exists := joinedDatasetIDs[targetID]; exists {
				continue
			}
			joinedDatasetIDs[targetID] = struct{}{}
		}
		joinExpr, err := reportJoinSQL(join, primaryDataset.Id, primaryTable.TableCode, datasetByID, tableByDatasetID, aliasByDatasetID)
		if err != nil {
			return response.ReportPreviewRes{}, err
		}
		if joinExpr != "" {
			query = query.Joins(joinExpr)
		}
	}
	var err error
	query, err = reportApplyJoinedParameters(query, config, primaryDataset.Id, primaryTable.TableCode, tableByDatasetID, aliasByDatasetID, req.Parameters)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}

	selections, columns, err := reportJoinedPreviewSelections(config, primaryDataset.Id, primaryTable, tableByDatasetID, aliasByDatasetID)
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	if len(selections) == 0 {
		return response.ReportPreviewRes{}, myerrors.NewBadRequestError("报表未配置可预览字段")
	}
	keyword := ""
	if req.Query.QuickQuery != nil {
		keyword = req.Query.QuickQuery.Keyword
	}
	query = reportApplyJoinedQuickSearch(query, keyword, selections)

	page := req.Query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.Query.Num
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return response.ReportPreviewRes{}, err
	}
	sqlRows, err := query.
		Select(strings.Join(reportSelectExprs(selections), ", ")).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Rows()
	if err != nil {
		return response.ReportPreviewRes{}, err
	}
	rows, _, err := scanReportSQLRows(sqlRows)
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
			ReportId:    report.Id,
			ReportCode:  report.Code,
			SourceCode:  primaryTable.TableCode,
			DatasetId:   primaryDataset.Id,
			DatasetType: reportSourceTypeTable,
			AppliedMenu: reportAppliedMenu(report, req.MenuId),
		},
	}, nil
}

func (s *ReportService) applyJoinedReportDataScope(ctx *gin.Context, query **gorm.DB, req request.ReportPreviewReq, report model.ReportDefinition, primaryTable model.SysTable) error {
	if s.dataPermissionService == nil {
		return myerrors.NewBadRequestError("报表数据权限服务未初始化")
	}
	value, exists := ctx.Get("user")
	if !exists {
		return myerrors.NewBadRequestError("报表运行缺少当前用户上下文")
	}
	user, ok := value.(model.SysUser)
	if !ok {
		return myerrors.NewBadRequestError("报表运行用户上下文不合法")
	}
	menuID := reportAppliedMenu(report, req.MenuId)
	scope, err := s.dataPermissionService.ResolveDataScopeForTableAction(user, menuID, primaryTable, enum.ButtonActionQuery)
	if err != nil {
		return err
	}
	*query = queryutil.ApplyDataScope(*query, scope, primaryTable)
	return nil
}

func safeReportPreviewSQL(raw string) (string, error) {
	sqlText := strings.TrimSpace(raw)
	if sqlText == "" {
		return "", myerrors.NewBadRequestError("SQL 数据集未配置 SQL")
	}
	if strings.Contains(sqlText, ";") {
		return "", myerrors.NewBadRequestError("SQL 数据集预览仅允许单条 SELECT/WITH 查询，禁止使用分号")
	}
	lower := strings.ToLower(sqlText)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return "", myerrors.NewBadRequestError("SQL 数据集预览仅允许 SELECT/WITH 只读查询")
	}
	if reportSQLForbiddenPattern.MatchString(sqlText) {
		return "", myerrors.NewBadRequestError("SQL 数据集预览禁止写操作或 DDL 关键字")
	}
	return sqlText, nil
}

func (s *ReportService) queryReportSQL(ctx *gin.Context, sqlText string, whereClause string, args []any, limit int) ([]map[string]interface{}, []response.ReportPreviewColumn, error) {
	if s.reportRepo == nil {
		return nil, nil, myerrors.NewBadRequestError("报表数据仓储未初始化")
	}
	wrapped := fmt.Sprintf("SELECT * FROM (%s) AS report_sql_dataset_preview", sqlText)
	if strings.TrimSpace(whereClause) != "" {
		wrapped += " WHERE " + whereClause
	}
	wrapped += " LIMIT ?"
	args = append(args, limit)
	sqlRows, err := s.reportRepo.DBWithContext(ctx).Raw(wrapped, args...).Rows()
	if err != nil {
		return nil, nil, err
	}
	columnMeta, err := reportSQLColumns(sqlRows)
	if err != nil {
		return nil, nil, err
	}
	records, columnNames, err := scanReportSQLRows(sqlRows)
	if err != nil {
		return nil, nil, err
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
	return records, columnMeta, nil
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
	if len(config.DatasetJoins()) > 0 {
		return true
	}
	tableDatasetCount := 0
	for _, dataset := range config.Datasets() {
		if reportconfig.NormalizeDataset(dataset).Type == reportconfig.SourceTypeTable {
			tableDatasetCount++
		}
	}
	return tableDatasetCount > 1
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
		return "", myerrors.NewBadRequestError("报表数据集关联缺少数据集")
	}
	if leftDataset.Type != reportconfig.SourceTypeTable || rightDataset.Type != reportconfig.SourceTypeTable {
		return "", myerrors.NewBadRequestError("报表数据集关联暂仅支持表数据集")
	}
	leftTable, leftTableOK := tableByDatasetID[leftID]
	rightTable, rightTableOK := tableByDatasetID[rightID]
	if !leftTableOK || !rightTableOK {
		return "", myerrors.NewBadRequestError("报表数据集关联表不存在")
	}
	if _, ok := reportFindTableField(leftTable, join.LeftField); !ok {
		return "", myerrors.NewBadRequestError("报表数据集关联左字段不存在")
	}
	if _, ok := reportFindTableField(rightTable, join.RightField); !ok {
		return "", myerrors.NewBadRequestError("报表数据集关联右字段不存在")
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
		return "", myerrors.NewBadRequestError("报表数据集关联别名不合法")
	}
	leftExpr := reportDatasetFieldExpr(leftID, join.LeftField, primaryDatasetID, primaryTableCode, aliasByDatasetID)
	rightExpr := reportDatasetFieldExpr(rightID, join.RightField, primaryDatasetID, primaryTableCode, aliasByDatasetID)
	if leftExpr == "" || rightExpr == "" {
		return "", myerrors.NewBadRequestError("报表数据集关联字段不合法")
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
			return nil, nil, myerrors.NewBadRequestError("报表绑定字段不合法")
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
				return nil, nil, myerrors.NewBadRequestError("报表绑定字段不合法")
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
			return query, myerrors.NewBadRequestError("报表参数绑定数据集不存在")
		}
		field, ok := reportFindTableField(table, param.Field)
		if !ok {
			return query, myerrors.NewBadRequestError("报表参数绑定字段不存在")
		}
		fieldExpr := reportDatasetFieldExpr(datasetID, field.FieldCode, primaryDatasetID, primaryTableCode, aliasByDatasetID)
		if fieldExpr == "" {
			return query, myerrors.NewBadRequestError("报表参数绑定字段不合法")
		}
		switch strings.ToLower(strings.TrimSpace(param.Operator)) {
		case "", "eq":
			query = query.Where(fmt.Sprintf("%s = ?", fieldExpr), value)
		case "like":
			query = query.Where(fmt.Sprintf("%s ILIKE ?", reportTextSearchExpr(fieldExpr)), "%"+fmt.Sprint(value)+"%")
		case "between":
			items, ok := reportRangeValues(value)
			if !ok {
				return query, myerrors.NewBadRequestError("报表区间参数必须传入两个值")
			}
			query = query.Where(fmt.Sprintf("%s BETWEEN ? AND ?", fieldExpr), items[0], items[1])
		case "gte":
			query = query.Where(fmt.Sprintf("%s >= ?", fieldExpr), value)
		case "lte":
			query = query.Where(fmt.Sprintf("%s <= ?", fieldExpr), value)
		default:
			return query, myerrors.NewBadRequestError("报表参数操作符不支持")
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

func reportAppliedMenu(report model.ReportDefinition, requestMenuID int) int {
	if report.PermissionMenuId > 0 {
		return report.PermissionMenuId
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
			return myerrors.NewBadRequestError("报表参数缺少绑定字段")
		}
		switch strings.ToLower(strings.TrimSpace(param.Operator)) {
		case "", "eq":
			query.Filters[field] = value
		case "like":
			rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Like, Value: value})
		case "between":
			if !reportValueIsRange(value) {
				return myerrors.NewBadRequestError("报表区间参数必须传入两个值")
			}
			rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Between, Value: value})
		case "gte":
			rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Gte, Value: value})
		case "lte":
			rules = append(rules, request.QueryRule{Field: field, ExpressionType: enum.Lte, Value: value})
		default:
			return myerrors.NewBadRequestError("报表参数操作符不支持")
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
			return "", nil, myerrors.NewBadRequestError("报表 SQL 参数绑定字段不合法")
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
				return "", nil, myerrors.NewBadRequestError("报表区间参数必须传入两个值")
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
			return "", nil, myerrors.NewBadRequestError("报表参数操作符不支持")
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
		return v.Format(time.DateTime)
	default:
		return value
	}
}

func (s *ReportService) injectReportDataScope(ctx *gin.Context, query *request.Basic, permissionTable model.SysTable) error {
	if s.dataPermissionService == nil || query == nil {
		return myerrors.NewBadRequestError("报表数据权限服务未初始化")
	}
	value, exists := ctx.Get("user")
	if !exists {
		return myerrors.NewBadRequestError("报表运行缺少当前用户上下文")
	}
	user, ok := value.(model.SysUser)
	if !ok {
		return myerrors.NewBadRequestError("报表运行用户上下文不合法")
	}
	scope, err := s.dataPermissionService.ResolveDataScopeForTableAction(user, query.MenuId, permissionTable, enum.ButtonActionQuery)
	if err != nil {
		return err
	}
	query.DataScope = scope
	return nil
}

func (s *ReportService) writeExecutionLog(ctx *gin.Context, report model.ReportDefinition, action string, req request.ReportPreviewReq, success bool, rowCount int, start time.Time, runErr error) error {
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	user := model.SysUser{}
	if value, exists := ctx.Get("user"); exists {
		if parsed, ok := value.(model.SysUser); ok {
			user = parsed
		}
	}
	params, _ := json.Marshal(req)
	log := model.ReportExecutionLog{
		Basic:        model.Basic{Id: int(id), State: true},
		ReportId:     report.Id,
		ReportCode:   report.Code,
		UserId:       user.Id,
		UserName:     user.UserName,
		Action:       action,
		Params:       datatypes.JSON(params),
		Success:      success,
		DurationMs:   time.Since(start).Milliseconds(),
		RowCount:     rowCount,
		ErrorMessage: "",
	}
	if runErr != nil {
		log.ErrorMessage = runErr.Error()
	}
	return s.reportLogRepo.Create(s.reportLogRepo.DBWithContext(ctx), &log)
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

func reportPreviewDatasets(config reportconfig.Config, report model.ReportDefinition, columns []response.ReportPreviewColumn) []response.ReportPreviewDataset {
	datasets := reportConfigDatasetMetadata(config)
	if len(datasets) > 0 {
		return datasets
	}
	return []response.ReportPreviewDataset{
		{
			Id:         "primary",
			Name:       report.Name,
			Type:       reportSourceTypeTable,
			SourceCode: report.SourceCode,
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
