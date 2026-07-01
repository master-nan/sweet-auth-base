package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"encoding/json"
	"errors"
	"fmt"
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
			Fields:      reportPreviewColumns(fullTable),
		})
	}
	return items, nil
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
	return s.reportRepo.Update(s.reportRepo.DBWithContext(ctx), &report, req.Id)
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
	sourceTable, permissionTable, err := s.resolveReportTables(report)
	if err != nil {
		_ = s.writeExecutionLog(ctx, report, "preview", req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	query := req.Query
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
	preview := response.ReportPreviewRes{
		Columns: reportPreviewColumns(sourceTable),
		Rows:    result.Data,
		Total:   result.Total,
		Meta: response.ReportPreviewMeta{
			ReportId:    report.Id,
			ReportCode:  report.Code,
			SourceCode:  sourceTable.TableCode,
			AppliedMenu: query.MenuId,
		},
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
	report.SourceType = normalizeReportSourceType(req.SourceType)
	report.SourceCode = strings.TrimSpace(req.SourceCode)
	report.PermissionTableCode = strings.TrimSpace(req.PermissionTableCode)
	if report.SourceType == "" {
		return report, myerrors.NewBadRequestError("报表数据源类型不合法")
	}
	if report.Code == "" || report.Name == "" || report.SourceCode == "" {
		return report, myerrors.ErrParamInvalid
	}
	if len(report.QueryConfig) == 0 {
		report.QueryConfig = datatypes.JSON([]byte("{}"))
	}
	if len(report.LayoutConfig) == 0 {
		report.LayoutConfig = datatypes.JSON([]byte("{}"))
	}
	report.State = true
	return report, s.validateReportTables(report)
}

func (s *ReportService) reportFromUpdateReq(req request.ReportDefinitionUpdateReq) (model.ReportDefinition, error) {
	report := model.ReportDefinition{}
	if err := copier.Copy(&report, &req); err != nil {
		return report, err
	}
	report.Code = strings.TrimSpace(req.Code)
	report.Name = strings.TrimSpace(req.Name)
	report.SourceType = normalizeReportSourceType(req.SourceType)
	report.SourceCode = strings.TrimSpace(req.SourceCode)
	report.PermissionTableCode = strings.TrimSpace(req.PermissionTableCode)
	if req.Id <= 0 || report.Code == "" || report.Name == "" || report.SourceCode == "" || report.SourceType == "" {
		return report, myerrors.ErrParamInvalid
	}
	if len(report.QueryConfig) == 0 {
		report.QueryConfig = datatypes.JSON([]byte("{}"))
	}
	if len(report.LayoutConfig) == 0 {
		report.LayoutConfig = datatypes.JSON([]byte("{}"))
	}
	return report, s.validateReportTables(report)
}

func (s *ReportService) validateReportTables(report model.ReportDefinition) error {
	_, _, err := s.resolveReportTables(report)
	return err
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
		permissionTable, err = s.sysTableService.GetTableByTableCode(strings.TrimSpace(report.PermissionTableCode))
		if err != nil {
			return model.SysTable{}, model.SysTable{}, err
		}
		if permissionTable.Id == 0 {
			return model.SysTable{}, model.SysTable{}, myerrors.NewBadRequestError("报表权限表不存在")
		}
	}
	return sourceTable, permissionTable, nil
}

func (s *ReportService) injectReportDataScope(ctx *gin.Context, query *request.Basic, permissionTable model.SysTable) error {
	if s.dataPermissionService == nil || query == nil {
		return nil
	}
	value, exists := ctx.Get("user")
	if !exists {
		return nil
	}
	user, ok := value.(model.SysUser)
	if !ok {
		return nil
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
	columns := make([]response.ReportPreviewColumn, 0, len(table.TableFields))
	for _, field := range table.TableFields {
		if !field.IsListShow || strings.TrimSpace(field.FieldCode) == "" {
			continue
		}
		columns = append(columns, response.ReportPreviewColumn{
			Name:  field.FieldCode,
			Field: field.FieldCode,
			Label: field.FieldName,
			Type:  fmt.Sprintf("%d", field.FieldType),
		})
	}
	return columns
}

func reportTableTypeLabel(tableType enum.SysTableType) string {
	switch tableType {
	case enum.View:
		return reportSourceTypeView
	default:
		return reportSourceTypeTable
	}
}
