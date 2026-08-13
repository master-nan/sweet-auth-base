# V1-B 后端受控导出实现证据包

本文档用于外部架构审查，基于当前 V1-B 后端受控导出实现整理。本文档只摘录关键代码片段，不贴整文件。

## 1. 接口和 DTO 证据

### 1.1 ReportExportReq

- 文件路径：`backend/dto/request/report_req.go`
- 结构体：`ReportExportReq`
- 说明：导出请求支持 `csv`，兼容 `max_rows` / `maxRows`，并沿用报表运行的 `dataset_id`、`parameters`、`params`、`query` 参数形态。

```go
type ReportSQLFieldsReq struct {
	SQL string `json:"sql" binding:"required" example:"select id, name from sys_user"`
}

type ReportStatusUpdateReq struct {
	Status string `json:"status" binding:"required" example:"published"`
}

type ReportPublishReq struct {
	ChangeLog string `json:"change_log" example:"首次发布"`
}

type ReportExportReq struct {
	Format     string         `json:"format" example:"csv"`
	MenuId     int            `json:"menu_id" example:"401"`
	DatasetId  string         `json:"dataset_id" example:"main"`
	Parameters map[string]any `json:"parameters"`
	Params     map[string]any `json:"params"`
	Query      Basic          `json:"query"`
	MaxRows    *int           `json:"max_rows" example:"5000"`
	MaxRowsAlt *int           `json:"maxRows" example:"5000"`
}
```

### 1.2 ReportExportFile

- 文件路径：`backend/dto/response/report_res.go`
- 结构体：`ReportExportFile`
- 说明：Service 返回文件名、Content-Type、文件内容和导出行数；Controller 负责写出文件流。

```go
type ReportVersionRes struct {
	Id            int    `json:"id"`
	VersionNo     int    `json:"version_no"`
	Status        string `json:"status"`
	PublishedAt   string `json:"published_at"`
	PublishedBy   int    `json:"published_by"`
	PublishedName string `json:"published_name"`
	ChangeLog     string `json:"change_log"`
	IsCurrent     bool   `json:"is_current"`
}

type ReportExportFile struct {
	FileName    string
	ContentType string
	Content     []byte
	RowCount    int
}
```

### 1.3 Controller 和 Router

- 文件路径：`backend/controller/report_controller.go`
- 函数名：`(*ReportController).ExportReport`
- 文件路径：`backend/initialize/router.go`
- 路由：`POST /admin/report/:id/export`
- 说明：接口位于 `adminGroup` 下，请求通过 `ValidatorBody` 绑定，成功时直接返回 CSV 文件流；错误时仍走 `ctx.Error` 统一错误链路。

```go
func (r *ReportController) ExportReport(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	var data request.ReportExportReq
	translator := r.translators["zh"]
	if err := utils.ValidatorBody[request.ReportExportReq](ctx, &data, translator); err != nil {
		_ = ctx.Error(err)
		return
	}
	file, err := r.reportService.ExportReport(ctx, id, data)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	ctx.Header("Content-Type", file.ContentType)
	ctx.Header("Content-Disposition", contentDisposition("attachment", file.FileName))
	ctx.Data(200, file.ContentType, file.Content)
}

// backend/initialize/router.go
adminGroup.GET("/report/:id/versions", app.ReportController.GetReportVersions)
adminGroup.GET("/report/:id", app.ReportController.GetReportDefinitionById)
adminGroup.POST("/report", app.ReportController.CreateReportDefinition)
adminGroup.PUT("/report/:id", app.ReportController.UpdateReportDefinition)
adminGroup.POST("/report/:id/status", app.ReportController.UpdateReportDefinitionStatus)
adminGroup.POST("/report/:id/publish", app.ReportController.PublishReport)
adminGroup.POST("/report/:id/design-preview", app.ReportController.DesignPreviewReport)
adminGroup.POST("/report/:id/run", app.ReportController.RunReport)
adminGroup.POST("/report/:id/export", app.ReportController.ExportReport)
adminGroup.DELETE("/report/:id", app.ReportController.DeleteReportDefinitionById)
adminGroup.POST("/report/:id/preview", app.ReportController.PreviewReport)
```

## 2. 发布版本读取证据

### 2.1 ExportReport 读取 published_version_id 和版本快照

- 文件路径：`backend/service/report_service.go`
- 函数名：`(*ReportService).ExportReport`
- 函数名：`(*ReportService).loadPublishedReportSnapshot`
- 说明：`ExportReport` 调用 `loadPublishedReportSnapshot`；该函数只通过 `report.PublishedVersionId` 读取 `report_definition_version`，并校验 report 与 version 状态。

```go
func (s *ReportService) ExportReport(ctx *gin.Context, reportId int, req request.ReportExportReq) (response.ReportExportFile, error) {
	start := time.Now()
	previewReq := reportExportPreviewReq(req)
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: reportRuntimeExport}
	fail := func(err error) (response.ReportExportFile, error) {
		_ = s.writeExecutionLog(ctx, snapshot, previewReq, false, 0, start, err)
		return response.ReportExportFile{}, err
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = reportExportFormatCSV
	}
	if format == "xlsx" {
		return fail(myerrors.NewBadRequestError("报表导出暂不支持 xlsx 格式，仅支持 csv"))
	}
	if format != reportExportFormatCSV {
		return fail(myerrors.NewBadRequestError("报表导出格式不支持，仅支持 csv"))
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
	preview, err := s.executeReportSnapshotWithOptions(ctx, snapshot, previewReq, start, ReportExecutionOptions{
		MaxRows:              effectiveMaxRows,
		PageSizeLimit:        effectiveMaxRows,
		DefaultPageSize:      effectiveMaxRows,
		ExportMode:           true,
		WriteLog:             false,
		DataPermissionAction: enum.ButtonActionExport,
	})
```

```go
func (s *ReportService) loadPublishedReportSnapshot(ctx *gin.Context, reportId int, runtimeType string) (ReportExecutionSnapshot, error) {
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: runtimeType}
	report, err := s.GetReportDefinitionById(reportId)
	if err != nil {
		return snapshot, err
	}
	if report.Id == 0 {
		return snapshot, myerrors.ErrDataNotFound
	}
	snapshot.ReportId = report.Id
	snapshot.Code = report.Code
	if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
		return snapshot, myerrors.NewBadRequestError("报表已停用")
	}
	if normalizeReportStatus(report.Status) != reportStatusPublished || report.PublishedVersionId <= 0 {
		return snapshot, myerrors.NewBadRequestError("报表未发布，请先调用发布接口")
	}
	version, err := s.reportVersionRepo.FindByReportAndId(report.Id, report.PublishedVersionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = myerrors.NewBadRequestError("报表发布版本不存在")
		}
		return snapshot, err
	}
	if !version.State {
		return snapshot, myerrors.NewBadRequestError("报表发布版本不可用")
	}
	if normalizeReportStatus(version.Status) != reportVersionPublished {
		return snapshot, myerrors.NewBadRequestError("报表发布版本状态不可运行")
	}
	snapshot = reportSnapshotFromVersion(version, runtimeType)
	return snapshot, nil
}
```

### 2.2 从 version 构造 ReportExecutionSnapshot

- 文件路径：`backend/service/report_service.go`
- 函数名：`reportSnapshotFromVersion`
- 说明：运行态快照的 `QueryConfig` 和 `LayoutConfig` 来自 `ReportDefinitionVersion`，不是草稿 `ReportDefinition`。这证明导出不会读取草稿 `report_definition.query_config / layout_config`。

```go
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
```

## 3. 导出执行流程证据

### 3.1 ReportExecutionOptions 和 runtime_export

- 文件路径：`backend/service/report_service.go`
- 常量：`reportRuntimeExport`
- 结构体：`ReportExecutionOptions`
- 说明：V1-B 新增 `runtime_export`，并通过 `ReportExecutionOptions` 控制导出模式、行数上限、页大小上限、是否由公共执行方法自动写日志、数据权限 action。

```go
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

	defaultReportExportMaxRows = 5000
	maxReportExportRows        = 10000
)

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

type ReportExecutionOptions struct {
	MaxRows              int
	PageSizeLimit        int
	DefaultPageSize      int
	ExportMode           bool
	WriteLog             bool
	DataPermissionAction enum.SysMenuButtonEventAction
}
```

### 3.2 公共执行方法支持导出模式

- 文件路径：`backend/service/report_service.go`
- 函数名：`executeReportSnapshot`
- 函数名：`executeReportSnapshotWithOptions`
- 说明：普通运行仍通过 `executeReportSnapshot` 保持 `pageSizeLimit=200`；导出调用 `executeReportSnapshotWithOptions`，将 `PageSizeLimit` 和 `DefaultPageSize` 设置为有效导出上限，避免直接复用 RunReport 导致只能导出 200 行。

```go
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
			_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		}
	}
	writeSuccess := func(rowCount int) {
		if options.WriteLog {
			_ = s.writeExecutionLog(ctx, snapshot, req, true, rowCount, start, nil)
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
```

### 3.3 table 查询路径的导出补齐

- 文件路径：`backend/service/report_service.go`
- 函数名：`executeReportSnapshotWithOptions`
- 函数名：`completeReportTableExportRows`
- 说明：table dataset 继续复用现有 `generalizationService.Query` 和数据权限注入；导出模式下先校验 `total <= MaxRows`，再分页补齐数据，避免静默只导出第一页或前 N 行。

```go
if err := s.injectReportDataScopeForAction(ctx, &query, permissionTable, options.DataPermissionAction); err != nil {
	writeFailure(err)
	return response.ReportPreviewRes{}, err
}
result, err := s.generalizationService.Query(&query, sourceTable)
if err != nil {
	writeFailure(err)
	return response.ReportPreviewRes{}, err
}
if options.ExportMode {
	if result.Total > options.MaxRows {
		err = myerrors.NewBadRequestError("导出行数超过系统限制，请缩小查询条件后重试")
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	if err := s.completeReportTableExportRows(query, sourceTable, &result, options.MaxRows); err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
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
```

```go
func (s *ReportService) completeReportTableExportRows(query request.Basic, table model.SysTable, result *repository.GeneralizationListResult, maxRows int) error {
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
		nextResult, err := s.generalizationService.Query(&nextQuery, table)
		if err != nil {
			return err
		}
		if len(nextResult.Data) == 0 {
			return myerrors.NewBadRequestError("报表导出查询结果不完整")
		}
		result.Data = append(result.Data, nextResult.Data...)
		if len(result.Data) > maxRows {
			return myerrors.NewBadRequestError("导出行数超过系统限制，请缩小查询条件后重试")
		}
	}
	if len(result.Data) > result.Total {
		result.Data = result.Data[:result.Total]
	}
	return nil
}
```

## 4. 最大导出行数证据

### 4.1 max_rows / maxRows 归一化

- 文件路径：`backend/service/report_service.go`
- 函数名：`normalizeReportExportMaxRows`
- 说明：不传或非正数使用默认 5000；同时传入冲突值报错；大于 10000 直接报错。

```go
func normalizeReportExportMaxRows(req request.ReportExportReq) (int, error) {
	hasSnake := req.MaxRows != nil
	hasCamel := req.MaxRowsAlt != nil
	if hasSnake && hasCamel && *req.MaxRows != *req.MaxRowsAlt {
		return 0, myerrors.NewBadRequestError("max_rows 与 maxRows 不能同时传入不同值")
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
		return 0, myerrors.NewBadRequestError("导出行数超过系统最大限制")
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
```

### 4.2 total 超限直接失败

- 文件路径：`backend/service/report_service.go`
- 函数名：`(*ReportService).ExportReport`
- 说明：导出查询结束后再次校验 `preview.Total > effectiveMaxRows`，超限直接返回错误，不调用 `buildReportCSV`，因此不会生成部分文件。

```go
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
	return fail(myerrors.NewBadRequestError("导出行数超过系统限制，请缩小查询条件后重试"))
}
content, err := buildReportCSV(preview.Columns, preview.Rows)
if err != nil {
	return fail(err)
}
if err := s.writeExecutionLog(ctx, snapshot, previewReq, true, len(preview.Rows), start, nil); err != nil {
	return response.ReportExportFile{}, err
}
return response.ReportExportFile{
	FileName:    reportExportFileName(snapshot),
	ContentType: "text/csv; charset=utf-8",
	Content:     content,
	RowCount:    len(preview.Rows),
}, nil
```

## 5. CSV 生成和安全证据

### 5.1 CSV 构建、BOM、表头和数据行

- 文件路径：`backend/service/report_service.go`
- 函数名：`buildReportCSV`
- 说明：使用 `encoding/csv`，先写 UTF-8 BOM，再写表头和数据行；表头和数据值都经过 `safeReportCSVCell`。

```go
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
```

### 5.2 Excel 公式注入防护

- 文件路径：`backend/service/report_service.go`
- 函数名：`safeReportCSVCell`
- 说明：字符串或 `[]byte` 转字符串后，trim 后若以 `=`、`+`、`-`、`@` 开头，则在原始文本前追加单引号，避免 Excel 公式注入。

```go
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
		code = fmt.Sprintf("report_%d", snapshot.ReportId)
	}
	if snapshot.VersionNo > 0 {
		return fmt.Sprintf("%s_v%d.csv", code, snapshot.VersionNo)
	}
	return code + ".csv"
}
```

## 6. 权限和数据权限证据

### 6.1 adminGroup、按钮权限和 Casbin 种子

- 文件路径：`backend/initialize/router.go`
- 文件路径：`backend/migrate/main.go`
- 说明：导出接口挂在 `adminGroup`；迁移种子中新增报表中心和报表管理导出按钮，并为 super_admin 种入 `/admin/report/:id/export` 的 Casbin policy。项目已有 `enum.ButtonActionExport`，所以导出数据权限 action 使用 export。

```go
// backend/initialize/router.go
adminGroup.POST("/report/:id/publish", app.ReportController.PublishReport)
adminGroup.POST("/report/:id/design-preview", app.ReportController.DesignPreviewReport)
adminGroup.POST("/report/:id/run", app.ReportController.RunReport)
adminGroup.POST("/report/:id/export", app.ReportController.ExportReport)
adminGroup.DELETE("/report/:id", app.ReportController.DeleteReportDefinitionById)
adminGroup.POST("/report/:id/preview", app.ReportController.PreviewReport)

// backend/migrate/main.go
var sysMenuButtonEventActionSeeds = []dictSeed{
	{name: "查询", code: "sys_menu_button_event_action_query", value: string(enum.ButtonActionQuery)},
	{name: "页面元数据", code: "sys_menu_button_event_action_metadata", value: string(enum.ButtonActionMetadata)},
	{name: "详情", code: "sys_menu_button_event_action_detail", value: string(enum.ButtonActionDetail)},
	{name: "新增", code: "sys_menu_button_event_action_create", value: string(enum.ButtonActionCreate)},
	{name: "编辑", code: "sys_menu_button_event_action_update", value: string(enum.ButtonActionUpdate)},
	{name: "删除", code: "sys_menu_button_event_action_delete", value: string(enum.ButtonActionDelete)},
	{name: "导出", code: "sys_menu_button_event_action_export", value: string(enum.ButtonActionExport)},
}
```

```go
buttons := []model.SysMenuButton{
	menuButtonWithAPI(702, menuID, "运行报表", "report_center_preview", enum.Line, "preview", "play_arrow", "primary", 1, "/admin/report/:id/preview", "POST"),
	menuButtonWithAPI(703, menuID, "运行报表V1", "report_center_run", enum.Line, "run", "play_arrow", "primary", 2, "/admin/report/:id/run", "POST"),
	menuButtonWithAPI(733, menuID, "导出报表", "report_center_export", enum.Line, "export", "download", "primary", 3, "/admin/report/:id/export", "POST"),
	apiPermissionWithAPI(704, menuID, "报表列表", "report_center_query", enum.Top, "query", "search", "primary", 90, "/admin/report/query", "POST"),
}

buttons := []model.SysMenuButton{
	menuButtonWithAPI(720, menuID, "新建报表", "report_manage_create", enum.Top, "create", "add", "primary", 1, "/admin/report", "POST"),
	menuButtonWithAPI(730, menuID, "发布", "report_manage_publish", enum.Line, "publish", "published_with_changes", "primary", 6, "/admin/report/:id/publish", "POST"),
	menuButtonWithAPI(731, menuID, "运行", "report_manage_run", enum.Line, "run", "play_arrow", "primary", 7, "/admin/report/:id/run", "POST"),
	menuButtonWithAPI(734, menuID, "导出", "report_manage_export", enum.Line, "export", "download", "primary", 8, "/admin/report/:id/export", "POST"),
}

policies := []apiPolicy{
	{"/admin/report/:id/preview", "POST"},
	{"/admin/report/:id/design-preview", "POST"},
	{"/admin/report/:id/publish", "POST"},
	{"/admin/report/:id/run", "POST"},
	{"/admin/report/:id/export", "POST"},
	{"/admin/report/:id/versions", "GET"},
}
```

### 6.2 SQL dataset 和 table dataset 数据权限

- 文件路径：`backend/service/report_service.go`
- 函数名：`ensureSQLDatasetRole`
- 函数名：`injectReportDataScopeForAction`
- 说明：SQL dataset 继续仅允许 super_admin；table dataset 通过 `ResolveDataScopeForTableAction` 继续应用现有数据权限，并在导出时传入 `enum.ButtonActionExport`。

```go
func (s *ReportService) injectReportDataScopeForAction(ctx *gin.Context, query *request.Basic, permissionTable model.SysTable, action enum.SysMenuButtonEventAction) error {
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
	scope, err := s.dataPermissionService.ResolveDataScopeForTableAction(user, query.MenuId, permissionTable, action)
	if err != nil {
		return err
	}
	query.DataScope = scope
	return nil
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
```

## 7. 执行日志证据

### 7.1 runtime_export 日志结构

- 文件路径：`backend/service/report_service.go`
- 常量：`reportRuntimeExport`
- 函数名：`writeExecutionLog`
- 说明：导出调用 `writeExecutionLog`，`Action` 取 `snapshot.RuntimeType`；`params.runtime` 固定包含 `runtime_type`、`version_id`、`version_no`。

```go
const (
	reportRuntimeDesignPreview = "design_preview"
	reportRuntimeRun           = "runtime_run"
	reportRuntimeExport        = "runtime_export"
)

func (s *ReportService) writeExecutionLog(ctx *gin.Context, snapshot ReportExecutionSnapshot, req request.ReportPreviewReq, success bool, rowCount int, start time.Time, runErr error) error {
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	user := reportUserFromContext(ctx)
	action := snapshot.RuntimeType
	if action == "" {
		action = reportRuntimeDesignPreview
	}
	params, _ := json.Marshal(map[string]any{
		"request": req,
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
		UserId:       user.Id,
		UserName:     user.UserName,
		Action:       action,
		Params:       datatypes.JSON(params),
		Success:      success,
		DurationMs:   time.Since(start).Milliseconds(),
		RowCount:     rowCount,
		ErrorMessage: "",
	}
```

```go
if runErr != nil {
	log.ErrorMessage = runErr.Error()
}
db := s.reportLogRepo.DBWithContext(ctx)
if err := db.Create(&log).Error; err != nil {
	return err
}
if !success {
	return db.Model(&model.ReportExecutionLog{}).Where("id = ?", log.Id).Update("success", false).Error
}
return nil
```

### 7.2 导出成功和失败均写日志

- 文件路径：`backend/service/report_service.go`
- 函数名：`(*ReportService).ExportReport`
- 说明：失败路径统一走 `fail`，写 `success=false,row_count=0`；成功路径在 CSV 构建完成后写 `success=true,row_count=len(preview.Rows)`。

```go
func (s *ReportService) ExportReport(ctx *gin.Context, reportId int, req request.ReportExportReq) (response.ReportExportFile, error) {
	start := time.Now()
	previewReq := reportExportPreviewReq(req)
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: reportRuntimeExport}
	fail := func(err error) (response.ReportExportFile, error) {
		_ = s.writeExecutionLog(ctx, snapshot, previewReq, false, 0, start, err)
		return response.ReportExportFile{}, err
	}
	// format / max_rows / version / query checks ...
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
		return fail(myerrors.NewBadRequestError("导出行数超过系统限制，请缩小查询条件后重试"))
	}
	content, err := buildReportCSV(preview.Columns, preview.Rows)
	if err != nil {
		return fail(err)
	}
	if err := s.writeExecutionLog(ctx, snapshot, previewReq, true, len(preview.Rows), start, nil); err != nil {
		return response.ReportExportFile{}, err
	}
```

## 8. 错误处理证据

### 8.1 draft / disabled / version.status / xlsx / max_rows / total 超限

- 文件路径：`backend/service/report_service.go`
- 函数名：`loadPublishedReportSnapshot`
- 函数名：`ExportReport`
- 函数名：`normalizeReportExportMaxRows`
- 说明：draft 和 disabled 在快照读取阶段拒绝；version 非 published 拒绝；xlsx、max rows 超限、total 超限均返回明确错误，并通过 `fail` 写失败日志。

```go
if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
	return snapshot, myerrors.NewBadRequestError("报表已停用")
}
if normalizeReportStatus(report.Status) != reportStatusPublished || report.PublishedVersionId <= 0 {
	return snapshot, myerrors.NewBadRequestError("报表未发布，请先调用发布接口")
}
version, err := s.reportVersionRepo.FindByReportAndId(report.Id, report.PublishedVersionId)
if err != nil {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = myerrors.NewBadRequestError("报表发布版本不存在")
	}
	return snapshot, err
}
if !version.State {
	return snapshot, myerrors.NewBadRequestError("报表发布版本不可用")
}
if normalizeReportStatus(version.Status) != reportVersionPublished {
	return snapshot, myerrors.NewBadRequestError("报表发布版本状态不可运行")
}
```

```go
if format == "xlsx" {
	return fail(myerrors.NewBadRequestError("报表导出暂不支持 xlsx 格式，仅支持 csv"))
}
if format != reportExportFormatCSV {
	return fail(myerrors.NewBadRequestError("报表导出格式不支持，仅支持 csv"))
}
effectiveMaxRows, err := normalizeReportExportMaxRows(req)
if err != nil {
	return fail(err)
}
// ...
if preview.Total > effectiveMaxRows {
	return fail(myerrors.NewBadRequestError("导出行数超过系统限制，请缩小查询条件后重试"))
}
content, err := buildReportCSV(preview.Columns, preview.Rows)
if err != nil {
	return fail(err)
}
```

### 8.2 查询失败和 CSV 构建失败日志

- 文件路径：`backend/service/report_service.go`
- 函数名：`executeReportSnapshotWithOptions`
- 函数名：`ExportReport`
- 说明：公共执行方法内部查询失败会返回 error；导出流程设置 `WriteLog=false`，再由 `ExportReport.fail` 统一写 `runtime_export` 失败日志，避免重复写日志。

```go
writeFailure := func(err error) {
	if options.WriteLog {
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
	}
}
// ...
result, err := s.generalizationService.Query(&query, sourceTable)
if err != nil {
	writeFailure(err)
	return response.ReportPreviewRes{}, err
}
if options.ExportMode {
	if result.Total > options.MaxRows {
		err = myerrors.NewBadRequestError("导出行数超过系统限制，请缩小查询条件后重试")
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
	if err := s.completeReportTableExportRows(query, sourceTable, &result, options.MaxRows); err != nil {
		writeFailure(err)
		return response.ReportPreviewRes{}, err
	}
}

// ExportReport 中统一记录失败
preview, err := s.executeReportSnapshotWithOptions(ctx, snapshot, previewReq, start, ReportExecutionOptions{
	ExportMode: true,
	WriteLog:   false,
})
if err != nil {
	return fail(err)
}
content, err := buildReportCSV(preview.Columns, preview.Rows)
if err != nil {
	return fail(err)
}
```

## 9. 测试证据

### 9.1 V1-B 测试列表

- 文件路径：`backend/service/report_service_v1b_test.go`
- 测试函数：
  - `TestReportV1BExportStatusRulesAndLogs`
  - `TestReportV1BExportUsesPublishedVersionSnapshot`
  - `TestReportV1BExportRejectsVersionAndSQLPermission`
  - `TestReportV1BExportMaxRowsAndFormatValidation`
  - `TestReportV1BCSVSafety`
- 说明：覆盖 draft/disabled 禁止导出、版本快照隔离、再次发布后使用新版本、SQL dataset 权限、max rows、xlsx 错误、CSV 公式注入和 runtime_export 日志。

```go
func TestReportV1BExportStatusRulesAndLogs(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_status_rules", reportStatusDraft, queryConfig, layoutConfig)

	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{}); err == nil || !strings.Contains(err.Error(), "未发布") {
		t.Fatalf("draft report should not export, got %v", err)
	}
	draftLog := env.lastExecutionLog(t, reportRuntimeExport)
	if draftLog.Success {
		t.Fatalf("draft export failure log should be unsuccessful: %#v", draftLog)
	}
	assertReportRuntimeLog(t, draftLog, reportRuntimeExport, 0, 0)

	disabled := env.createReport(t, "v1b_disabled", reportStatusDisabled, queryConfig, layoutConfig)
	if _, err := env.svc.ExportReport(env.ctx, disabled.Id, request.ReportExportReq{}); err == nil || !strings.Contains(err.Error(), "已停用") {
		t.Fatalf("disabled report should not export, got %v", err)
	}
	disabledLog := env.lastExecutionLog(t, reportRuntimeExport)
	if disabledLog.Success {
		t.Fatalf("disabled export failure log should be unsuccessful: %#v", disabledLog)
	}
}
```

### 9.2 修改草稿不影响导出、再次发布后导出新版本

- 文件路径：`backend/service/report_service_v1b_test.go`
- 函数名：`TestReportV1BExportUsesPublishedVersionSnapshot`
- 说明：发布 v1 后导出“名称”；修改草稿为“金额”但不发布，导出仍为“名称”；再次发布 v2 后导出变为“金额”。

```go
func TestReportV1BExportUsesPublishedVersionSnapshot(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	nameQuery, nameLayout := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_snapshot", reportStatusDraft, nameQuery, nameLayout)

	published1, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{ChangeLog: "publish name"})
	if err != nil {
		t.Fatalf("publish report first version: %v", err)
	}
	file1, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
	if err != nil {
		t.Fatalf("export first published version: %v", err)
	}
	records1 := parseReportCSV(t, file1.Content)
	assertCSVHeader(t, records1, "名称")

	amountQuery, amountLayout := reportV1ATableConfig("amount")
	if err := env.db.Model(&model.ReportDefinition{}).
		Where("id = ?", report.Id).
		Updates(map[string]any{
			"query_config":  amountQuery,
			"layout_config": amountLayout,
		}).Error; err != nil {
		t.Fatalf("modify draft config after publish: %v", err)
	}
	fileAfterDraftEdit, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
	if err != nil {
		t.Fatalf("export after draft edit: %v", err)
	}
	recordsAfterDraftEdit := parseReportCSV(t, fileAfterDraftEdit.Content)
	assertCSVHeader(t, recordsAfterDraftEdit, "名称")
```

```go
	published2, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{ChangeLog: "publish amount"})
	if err != nil {
		t.Fatalf("publish report second version: %v", err)
	}
	file2, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{})
	if err != nil {
		t.Fatalf("export second published version: %v", err)
	}
	records2 := parseReportCSV(t, file2.Content)
	assertCSVHeader(t, records2, "金额")
	if !strings.Contains(file2.FileName, "_v2.csv") {
		t.Fatalf("export file should include version: %s", file2.FileName)
	}
	exportLog := env.lastExecutionLog(t, reportRuntimeExport)
	if !exportLog.Success || exportLog.RowCount != 2 {
		t.Fatalf("export success log should record rows: %#v", exportLog)
	}
	assertReportRuntimeLog(t, exportLog, reportRuntimeExport, published2.VersionId, 2)
	if published1.VersionId == published2.VersionId {
		t.Fatalf("publish versions should differ: first=%#v second=%#v", published1, published2)
	}
}
```

### 9.3 max rows、CSV 安全和 runtime_export 日志

- 文件路径：`backend/service/report_service_v1b_test.go`
- 函数名：`TestReportV1BExportMaxRowsAndFormatValidation`
- 函数名：`TestReportV1BCSVSafety`
- 说明：测试默认 5000、非正数默认、双参数冲突、超过 10000、xlsx 不支持、total 超限、失败日志，以及 CSV 公式注入防护。

```go
func TestReportV1BExportMaxRowsAndFormatValidation(t *testing.T) {
	env := newReportV1ATestEnv(t, reportV1AUser(false))
	queryConfig, layoutConfig := reportV1ATableConfig("name")
	report := env.createReport(t, "v1b_limits", reportStatusDraft, queryConfig, layoutConfig)
	published, err := env.svc.PublishReport(env.ctx, report.Id, request.ReportPublishReq{})
	if err != nil {
		t.Fatalf("publish report: %v", err)
	}

	if got, err := normalizeReportExportMaxRows(request.ReportExportReq{}); err != nil || got != defaultReportExportMaxRows {
		t.Fatalf("default max rows = %d/%v, want %d", got, err, defaultReportExportMaxRows)
	}
	if got, err := normalizeReportExportMaxRows(request.ReportExportReq{MaxRows: reportIntPtr(0)}); err != nil || got != defaultReportExportMaxRows {
		t.Fatalf("non-positive max rows = %d/%v, want default", got, err)
	}
	if _, err := normalizeReportExportMaxRows(request.ReportExportReq{MaxRows: reportIntPtr(10), MaxRowsAlt: reportIntPtr(20)}); err == nil || !strings.Contains(err.Error(), "max_rows") {
		t.Fatalf("conflicting max rows should fail, got %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{MaxRows: reportIntPtr(maxReportExportRows + 1)}); err == nil || !strings.Contains(err.Error(), "最大限制") {
		t.Fatalf("max rows above hard limit should fail, got %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{Format: "xlsx"}); err == nil || !strings.Contains(err.Error(), "不支持 xlsx") {
		t.Fatalf("xlsx export should fail clearly, got %v", err)
	}
	if _, err := env.svc.ExportReport(env.ctx, report.Id, request.ReportExportReq{MaxRows: reportIntPtr(1)}); err == nil || !strings.Contains(err.Error(), "导出行数超过系统限制") {
		t.Fatalf("total greater than max rows should fail, got %v", err)
	}
	limitLog := env.lastExecutionLog(t, reportRuntimeExport)
	if limitLog.Success || limitLog.RowCount != 0 {
		t.Fatalf("limit failure log should be unsuccessful with row_count=0: %#v", limitLog)
	}
	assertReportRuntimeLog(t, limitLog, reportRuntimeExport, published.VersionId, 1)
}
```

```go
func TestReportV1BCSVSafety(t *testing.T) {
	content, err := buildReportCSV([]response.ReportPreviewColumn{
		{Field: "name", Label: "=危险表头"},
		{Field: "amount", Label: "金额"},
	}, []map[string]interface{}{
		{"name": "+SUM(1,1)", "amount": "-10"},
		{"name": "@cmd", "amount": 20},
	})
	if err != nil {
		t.Fatalf("build csv: %v", err)
	}
	records := parseReportCSV(t, content)
	if records[0][0] != "'=危险表头" {
		t.Fatalf("header should be formula-safe: %#v", records[0])
	}
	if records[1][0] != "'+SUM(1,1)" || records[1][1] != "'-10" || records[2][0] != "'@cmd" {
		t.Fatalf("data cells should be formula-safe: %#v", records)
	}
}
```

### 9.4 go test ./... 结果摘要

- 执行目录：`backend`
- 命令：`go test ./...`
- 结果：通过
- 摘要：`backend/api`、`backend/controller`、`backend/dto/response`、`backend/initialize`、`backend/internal/*`、`backend/middleware`、`backend/migrate`、`backend/repository/*`、`backend/service` 等包均通过；无失败包。

## 10. 已知风险

1. 同步内存 CSV 风险：当前 CSV 直接构建为 `[]byte` 后由 Controller 返回。V1-B 通过 10000 行硬上限控制风险，但不适合大批量、长耗时导出。异步任务、文件落盘、对象存储、任务中心留到 V1-C / V1-D。
2. 客户端断连日志风险：当前日志在查询和 CSV 构建成功后写入，随后再 `ctx.Data` 输出文件。如果客户端在响应写出阶段断连，日志仍可能记录为成功。更精细的传输态日志留到后续导出任务化阶段。
3. SQL dataset 风险：V1-B 仍只做 super_admin 限制，并复用 V1-A SQL 安全校验；未实现完整 SQL AST 白名单，也未承诺 SQL dataset 自动数据权限注入。AST 白名单、外部数据源、数据集实体化留到 V1-C / V1-D。
4. table dataset 数据权限依赖现有体系：导出已使用 `enum.ButtonActionExport` 调用现有 `ResolveDataScopeForTableAction`，但具体权限规则仍依赖菜单、按钮、数据权限配置完整性。
5. 第一版仅 CSV：`xlsx`、PDF、打印分页、邮件订阅、定时任务、导出任务中心均明确不在 V1-B 范围内。
