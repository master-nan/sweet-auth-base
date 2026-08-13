# 报表模块 V1-A 实现证据包

## 版本信息

- 范围：V1-A 发布版本 + 运行态隔离。
- 不包含：`report_datasource`、`report_dataset`、外部数据库数据源、后端导出、Excel/CSV 导出、完整 AST SQL 白名单、前端复杂 UI 改造。
- 证据来源：当前工作区实现代码。

---

## 1. 数据模型证据

### 1.1 `ReportDefinition.published_version_id`

- 文件：`backend/model/report.go`
- 结构体：`ReportDefinition`
- 说明：主表继续承载草稿；`PublishedVersionId` 指向当前发布版本快照。

```go
type ReportDefinition struct {
	Basic
	Code                string         `gorm:"size:128;uniqueIndex:uni_report_definition_code;comment:报表编码" json:"code"`
	Name                string         `gorm:"size:128;comment:报表名称" json:"name"`
	Description         string         `gorm:"size:512;comment:报表说明" json:"description"`
	Category            string         `gorm:"size:128;comment:报表分类" json:"category"`
	Status              string         `gorm:"size:32;default:draft;index:idx_report_definition_status;comment:报表状态（draft:草稿,published:已发布,disabled:已停用）" json:"status"`
	PublishedVersionId  int            `gorm:"index:idx_report_definition_published_version;comment:当前发布版本ID" json:"published_version_id"`
	SourceType          string         `gorm:"size:32;default:table;comment:数据源类型" json:"source_type"`
	SourceCode          string         `gorm:"size:128;index:idx_report_definition_source;comment:数据源表/视图编码" json:"source_code"`
	PermissionMenuId    int            `gorm:"comment:数据权限菜单ID" json:"permission_menu_id"`
	PermissionTableCode string         `gorm:"size:128;comment:数据权限表编码" json:"permission_table_code"`
	QueryConfig         datatypes.JSON `gorm:"type:jsonb;comment:查询配置JSON" json:"query_config"`
	LayoutConfig        datatypes.JSON `gorm:"type:jsonb;comment:布局配置JSON" json:"layout_config"`
	Remark              string         `gorm:"size:256;comment:备注" json:"remark"`
}
```

### 1.2 `ReportDefinitionVersion` model、表名和索引设计

- 文件：`backend/model/report.go`
- 结构体：`ReportDefinitionVersion`
- 说明：GORM 当前配置使用 `SingularTable`，结构体映射为 `report_definition_version`；`report_id + version_no` 使用唯一索引防止同一报表版本号重复。

```go
type ReportDefinitionVersion struct {
	Basic
	ReportId            int            `gorm:"index:idx_report_definition_version_report;uniqueIndex:uni_report_definition_version_no;comment:报表ID" json:"report_id"`
	VersionNo           int            `gorm:"uniqueIndex:uni_report_definition_version_no;comment:版本号" json:"version_no"`
	ReportCode          string         `gorm:"size:128;comment:报表编码" json:"report_code"`
	ReportName          string         `gorm:"size:128;comment:报表名称" json:"report_name"`
	Description         string         `gorm:"size:512;comment:报表说明" json:"description"`
	Category            string         `gorm:"size:128;comment:报表分类" json:"category"`
	SourceType          string         `gorm:"size:32;comment:数据源类型" json:"source_type"`
	SourceCode          string         `gorm:"size:128;comment:数据源表/视图编码" json:"source_code"`
	PermissionMenuId    int            `gorm:"comment:数据权限菜单ID" json:"permission_menu_id"`
	PermissionTableCode string         `gorm:"size:128;comment:数据权限表编码" json:"permission_table_code"`
	QueryConfig         datatypes.JSON `gorm:"type:jsonb;comment:查询配置JSON快照" json:"query_config"`
	LayoutConfig        datatypes.JSON `gorm:"type:jsonb;comment:布局配置JSON快照" json:"layout_config"`
	Status              string         `gorm:"size:32;default:published;index:idx_report_definition_version_status;comment:版本状态（published:当前发布,archived:历史归档）" json:"status"`
	PublishedAt         CustomTime     `gorm:"type:timestamp;index:idx_report_definition_version_published_at;comment:发布时间" json:"published_at"`
	PublishedBy         int            `gorm:"comment:发布人ID" json:"published_by"`
	PublishedName       string         `gorm:"size:128;comment:发布人名称" json:"published_name"`
	ChangeLog           string         `gorm:"size:512;comment:发布说明" json:"change_log"`
}
```

### 1.3 迁移注册

- 文件：`backend/migrate/main.go`
- 函数：`migrateSchema`
- 说明：AutoMigrate 注册 `ReportDefinitionVersion`；`ReportDefinition` 新增字段由 AutoMigrate 增量迁移。

```go
err := db.AutoMigrate(
	&model.SysConfigure{},
	&model.SysTable{},
	&model.SysTableField{},
	&model.SysTableRelation{},
	&model.SysTableIndex{},
	&model.SysTableIndexField{},
	&model.SysDict{},
	&model.SysDictItem{},
	&model.AccessLog{},
	&model.LoginLog{},
	&model.SysUser{},
	&model.SysUserRole{},
	&model.SysMenu{},
	&model.SysMenuButton{},
	&model.SysMenuButtonTemplate{},
	&model.SysRole{},
	&model.SysRoleMenu{},
	&model.SysRoleMenuButton{},
	&model.SysDataDimension{},
	&model.SysDataScopeBinding{},
	&model.SysRoleDataScope{},
	&model.SysUserDataScopeOverride{},
	&model.SysUserDimensionValue{},
	&model.ReportDefinition{},
	&model.ReportDefinitionVersion{},
	&model.ReportExecutionLog{},
	&model.Application{},
)
```

---

## 2. 发布流程证据

### 2.1 `PublishReport` 核心流程、事务和 `FOR UPDATE`

- 文件：`backend/service/report_service.go`
- 函数：`PublishReport`
- 说明：发布流程整体在数据库事务内；`clause.Locking{Strength: "UPDATE"}` 锁定 `report_definition` 行。

```go
func (s *ReportService) PublishReport(ctx *gin.Context, reportId int, req request.ReportPublishReq) (response.ReportPublishRes, error) {
	if reportId <= 0 {
		return response.ReportPublishRes{}, myerrors.ErrParamInvalid
	}
	var published model.ReportDefinitionVersion
	err := s.reportRepo.DBWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.ReportDefinition
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.ErrDataNotFound
			}
			return err
		}
		if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
			return myerrors.NewBadRequestError("报表已停用，不能发布")
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
```

### 2.2 `version_no` 生成、创建版本、更新发布指针

- 文件：`backend/service/report_service.go`
- 函数：`PublishReport`
- 说明：版本号在事务内从当前最大版本号生成；创建版本后才更新 `published_version_id`。

```go
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
```

### 2.3 发布失败不更新 `published_version_id` 的保证

- 文件：`backend/service/report_service.go`
- 函数：`PublishReport`
- 说明：校验、归档、创建版本、更新主表指针都在同一个事务闭包内；任一 `return err` 会回滚，事务成功后才返回发布结果。

```go
err := s.reportRepo.DBWithContext(ctx).Transaction(func(tx *gorm.DB) error {
	// validate report/config/sql permission
	// archive old published versions
	// create report_definition_version
	// update report_definition.published_version_id
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
```

### 2.4 版本号查询 repository

- 文件：`backend/repository/impl/report_impl.go`
- 函数：`GetMaxVersionNo`
- 说明：发布事务传入 `tx`，在锁定主表后读取最大版本号。

```go
func (r *ReportDefinitionVersionRepositoryImpl) GetMaxVersionNo(tx *gorm.DB, reportId int) (int, error) {
	var maxVersionNo int
	err := tx.Model(&model.ReportDefinitionVersion{}).
		Where("report_id = ?", reportId).
		Select("COALESCE(MAX(version_no), 0)").
		Scan(&maxVersionNo).Error
	return maxVersionNo, err
}
```

---

## 3. 运行态隔离证据

### 3.1 `ReportExecutionSnapshot` 结构

- 文件：`backend/service/report_service.go`
- 结构体：`ReportExecutionSnapshot`
- 说明：公共执行方法只接收 snapshot，不直接接收 `ReportDefinition`。

```go
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
```

### 3.2 `DesignPreview` 从草稿构造 snapshot

- 文件：`backend/service/report_service.go`
- 函数：`DesignPreview`
- 说明：设计时预览读取 `report_definition` 草稿，构造 `runtime_type = design_preview`、`version_id = 0` 的 snapshot。

```go
func (s *ReportService) DesignPreview(ctx *gin.Context, reportId int, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	start := time.Now()
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: reportRuntimeDesignPreview}
	report, err := s.GetReportDefinitionById(reportId)
	if err != nil {
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if report.Id == 0 {
		err = myerrors.ErrDataNotFound
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	snapshot = reportSnapshotFromDefinition(report, reportRuntimeDesignPreview)
	if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
		err = myerrors.NewBadRequestError("报表已停用")
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	return s.executeReportSnapshot(ctx, snapshot, req, start)
}
```

### 3.3 `RunReport` 从发布版本构造 snapshot

- 文件：`backend/service/report_service.go`
- 函数：`RunReport`
- 说明：运行接口仅通过 `published_version_id` 读取 `report_definition_version`，随后从版本构造 snapshot。

```go
func (s *ReportService) RunReport(ctx *gin.Context, reportId int, req request.ReportPreviewReq) (response.ReportPreviewRes, error) {
	start := time.Now()
	snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: reportRuntimeRun}
	report, err := s.GetReportDefinitionById(reportId)
	if err != nil {
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if report.Id == 0 {
		err = myerrors.ErrDataNotFound
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	snapshot.ReportId = report.Id
	snapshot.Code = report.Code
	if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
		err = myerrors.NewBadRequestError("报表已停用")
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if normalizeReportStatus(report.Status) != reportStatusPublished || report.PublishedVersionId <= 0 {
		err = myerrors.NewBadRequestError("报表未发布，请先调用发布接口")
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	version, err := s.reportVersionRepo.FindByReportAndId(report.Id, report.PublishedVersionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = myerrors.NewBadRequestError("报表发布版本不存在")
		}
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if !version.State {
		err = myerrors.NewBadRequestError("报表发布版本不可用")
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	snapshot = reportSnapshotFromVersion(version, reportRuntimeRun)
	return s.executeReportSnapshot(ctx, snapshot, req, start)
}
```

### 3.4 snapshot 构造函数

- 文件：`backend/service/report_service.go`
- 函数：`reportSnapshotFromDefinition`、`reportSnapshotFromVersion`
- 说明：设计时 snapshot 从草稿 JSON 克隆；运行态 snapshot 从版本 JSON 克隆。

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

### 3.5 公共执行方法签名与隔离证明

- 文件：`backend/service/report_service.go`
- 函数：`executeReportSnapshot`
- 说明：公共执行方法签名只接收 `ReportExecutionSnapshot`，并通过 `snapshot.QueryConfig` / `snapshot.LayoutConfig` 执行。

```go
func (s *ReportService) executeReportSnapshot(ctx *gin.Context, snapshot ReportExecutionSnapshot, req request.ReportPreviewReq, start time.Time) (response.ReportPreviewRes, error) {
	config, err := reportconfig.Parse(snapshot.QueryConfig, snapshot.LayoutConfig)
	if err != nil {
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if err := validateReportSQLDatasets(config); err != nil {
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	if err := ensureSQLDatasetRole(ctx, config); err != nil {
		_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
		return response.ReportPreviewRes{}, err
	}
	// 后续查询统一使用 snapshot 派生的 config、权限字段和 source 字段。
}
```

证明点：

- `RunReport` 只使用 `report_definition` 的 `status`、`state`、`published_version_id` 来定位发布版本。
- `RunReport` 的执行配置来自 `reportSnapshotFromVersion(version, reportRuntimeRun)`。
- `executeReportSnapshot` 不接收 `model.ReportDefinition`，因此运行态公共执行链路不会重新读取草稿 `query_config` / `layout_config`。

---

## 4. 状态保护证据

### 4.1 禁止直接改成 `published`

- 文件：`backend/service/report_service.go`
- 函数：`UpdateReportDefinitionStatus`
- 说明：状态接口不能绕过发布流程。

```go
func (s *ReportService) UpdateReportDefinitionStatus(ctx *gin.Context, id int, status string) error {
	status = normalizeReportStatus(status)
	if id <= 0 || !isValidReportStatus(status) {
		return myerrors.ErrParamInvalid
	}
	if status == reportStatusPublished {
		return myerrors.NewBadRequestError("发布报表必须调用 /admin/report/:id/publish")
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
```

### 4.2 `disabled` 报表不能 design-preview

- 文件：`backend/service/report_service.go`
- 函数：`DesignPreview`

```go
snapshot = reportSnapshotFromDefinition(report, reportRuntimeDesignPreview)
if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
	err = myerrors.NewBadRequestError("报表已停用")
	_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
	return response.ReportPreviewRes{}, err
}
return s.executeReportSnapshot(ctx, snapshot, req, start)
```

### 4.3 `disabled` / `draft` 报表不能 run

- 文件：`backend/service/report_service.go`
- 函数：`RunReport`

```go
if !report.State || normalizeReportStatus(report.Status) == reportStatusDisabled {
	err = myerrors.NewBadRequestError("报表已停用")
	_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
	return response.ReportPreviewRes{}, err
}
if normalizeReportStatus(report.Status) != reportStatusPublished || report.PublishedVersionId <= 0 {
	err = myerrors.NewBadRequestError("报表未发布，请先调用发布接口")
	_ = s.writeExecutionLog(ctx, snapshot, req, false, 0, start, err)
	return response.ReportPreviewRes{}, err
}
```

---

## 5. SQL dataset 权限证据

### 5.1 判断是否包含 SQL dataset

- 文件：`backend/service/report_service.go`
- 函数：`validateReportSQLDatasets`、`reportConfigHasSQLDataset`
- 说明：先校验 SQL 安全，再判断是否需要限制角色。

```go
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
```

### 5.2 SQL dataset 仅允许 super_admin

- 文件：`backend/service/report_service.go`
- 函数：`ensureSQLDatasetRole`
- 说明：`design-preview`、`publish`、`run` 都会进入 `ensureSQLDatasetRole`；项目已有 `utils.IsSuperAdmin`，未凭空创造新的 super admin 判断。

```go
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

### 5.3 项目已有 super_admin 判断方式

- 文件：`backend/internal/utils/tools.go`
- 函数：`IsSuperAdmin`

```go
func IsSuperAdmin(user model.SysUser) bool {
	for _, role := range user.Roles {
		if role.Name == "super_admin" {
			return true
		}
	}
	return false
}
```

调用位置：

- `PublishReport`：发布前调用 `ensureSQLDatasetRole`
- `executeReportSnapshot`：design-preview 和 run 的公共执行入口调用 `ensureSQLDatasetRole`

---

## 6. 执行日志证据

### 6.1 `writeExecutionLog` 固定 params JSON 结构

- 文件：`backend/service/report_service.go`
- 函数：`writeExecutionLog`
- 说明：`Params` 固定写入 `request` 和 `runtime`；`runtime` 包含 `runtime_type`、`version_id`、`version_no`。

```go
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
	if runErr != nil {
		log.ErrorMessage = runErr.Error()
	}
	return s.reportLogRepo.Create(s.reportLogRepo.DBWithContext(ctx), &log)
}
```

### 6.2 `design_preview` / `runtime_run` action 来源

- 文件：`backend/service/report_service.go`
- 常量：`reportRuntimeDesignPreview`、`reportRuntimeRun`
- 说明：action 直接来自 snapshot 的 `RuntimeType`。

```go
const (
	reportRuntimeDesignPreview = "design_preview"
	reportRuntimeRun           = "runtime_run"
)

// DesignPreview
snapshot = reportSnapshotFromDefinition(report, reportRuntimeDesignPreview)

// RunReport
snapshot = reportSnapshotFromVersion(version, reportRuntimeRun)
```

日志结构示例：

```json
{
  "request": {},
  "runtime": {
    "runtime_type": "runtime_run",
    "version_id": 123,
    "version_no": 2
  }
}
```

---

## 7. 历史数据回填证据

### 7.1 查找历史 published 且未回填的报表

- 文件：`backend/migrate/main.go`
- 函数：`backfillReportDefinitionVersions`

```go
func backfillReportDefinitionVersions(db *gorm.DB, sf *utils.Snowflake) error {
	if !db.Migrator().HasTable(&model.ReportDefinition{}) || !db.Migrator().HasTable(&model.ReportDefinitionVersion{}) {
		return nil
	}
	var reports []model.ReportDefinition
	if err := db.Where("status = ? AND COALESCE(published_version_id, 0) = 0", "published").Find(&reports).Error; err != nil {
		return err
	}
	for _, report := range reports {
		if !json.Valid(report.QueryConfig) || !json.Valid(report.LayoutConfig) {
			log.Printf("skip report version backfill: report_id=%d code=%s has invalid query_config/layout_config", report.Id, report.Code)
			continue
		}
```

### 7.2 幂等策略和异常跳过

- 文件：`backend/migrate/main.go`
- 函数：`backfillReportDefinitionVersions`
- 说明：已有版本则只回填指针；没有版本才创建 `version_no = 1`；JSON 异常记录日志并跳过；唯一索引冲突时记录并跳过。

```go
var existing model.ReportDefinitionVersion
err := db.Where("report_id = ?", report.Id).Order("version_no DESC").First(&existing).Error
if err == nil {
	if err := db.Model(&model.ReportDefinition{}).
		Where("id = ? AND COALESCE(published_version_id, 0) = 0", report.Id).
		Update("published_version_id", existing.Id).Error; err != nil {
		return err
	}
	continue
}
if err != gorm.ErrRecordNotFound {
	return err
}
id, err := newMigrationID(sf)
if err != nil {
	return err
}
publishedAt := model.CustomTime(model.Now())
version := model.ReportDefinitionVersion{
	Basic:               model.Basic{Id: id, State: true},
	ReportId:            report.Id,
	VersionNo:           1,
	ReportCode:          report.Code,
	ReportName:          report.Name,
	QueryConfig:         report.QueryConfig,
	LayoutConfig:        report.LayoutConfig,
	Status:              "published",
	PublishedAt:         publishedAt,
	PublishedName:       "migration",
	ChangeLog:           "历史 published 报表初始版本回填",
}
if err := db.Transaction(func(tx *gorm.DB) error {
	if err := tx.Create(&version).Error; err != nil {
		return err
	}
	return tx.Model(&model.ReportDefinition{}).
		Where("id = ? AND COALESCE(published_version_id, 0) = 0", report.Id).
		Update("published_version_id", version.Id).Error
}); err != nil {
	if strings.Contains(err.Error(), "uni_report_definition_version_no") {
		log.Printf("skip duplicate report version backfill: report_id=%d code=%s", report.Id, report.Code)
		continue
	}
	return err
}
```

幂等依据：

- 查询条件只处理 `COALESCE(published_version_id, 0) = 0`。
- 已有版本时不创建新版本。
- `report_id + version_no` 唯一索引防止重复创建 `version_no = 1`。

---

## 8. 路由和权限证据

### 8.1 新增路由位于 admin 认证组

- 文件：`backend/initialize/router.go`
- 函数：`InitRouter`
- 说明：新增路由位于 `adminGroup` 中，继承 `AuthHandler` 和 `CasbinHandler`。

```go
// report
adminGroup.POST("/report/query", app.ReportController.QueryReportDefinitions)
adminGroup.GET("/report/data-sources", app.ReportController.GetReportDataSources)
adminGroup.POST("/report/sql-fields", app.ReportController.InferSQLFields)
adminGroup.GET("/report/:id/versions", app.ReportController.GetReportVersions)
adminGroup.GET("/report/:id", app.ReportController.GetReportDefinitionById)
adminGroup.POST("/report", app.ReportController.CreateReportDefinition)
adminGroup.PUT("/report/:id", app.ReportController.UpdateReportDefinition)
adminGroup.POST("/report/:id/status", app.ReportController.UpdateReportDefinitionStatus)
adminGroup.POST("/report/:id/publish", app.ReportController.PublishReport)
adminGroup.POST("/report/:id/design-preview", app.ReportController.DesignPreviewReport)
adminGroup.POST("/report/:id/run", app.ReportController.RunReport)
adminGroup.DELETE("/report/:id", app.ReportController.DeleteReportDefinitionById)
adminGroup.POST("/report/:id/preview", app.ReportController.PreviewReport)
```

### 8.2 菜单按钮权限种子

- 文件：`backend/migrate/main.go`
- 函数：`seedReportCenterMenuButtons`、`seedReportManageMenuButtons`、`seedReportDesignMenuButtons`
- 说明：新增运行、发布、版本列表、设计时预览按钮或 API 权限。

```go
buttons := []model.SysMenuButton{
	menuButtonWithAPI(702, menuID, "运行报表", "report_center_preview", enum.Line, "preview", "play_arrow", "primary", 1, "/admin/report/:id/preview", "POST"),
	menuButtonWithAPI(703, menuID, "运行报表V1", "report_center_run", enum.Line, "run", "play_arrow", "primary", 2, "/admin/report/:id/run", "POST"),
	apiPermissionWithAPI(704, menuID, "报表列表", "report_center_query", enum.Top, "query", "search", "primary", 90, "/admin/report/query", "POST"),
}

buttons := []model.SysMenuButton{
	menuButtonWithAPI(730, menuID, "发布", "report_manage_publish", enum.Line, "publish", "published_with_changes", "primary", 6, "/admin/report/:id/publish", "POST"),
	menuButtonWithAPI(731, menuID, "运行", "report_manage_run", enum.Line, "run", "play_arrow", "primary", 7, "/admin/report/:id/run", "POST"),
	apiPermissionWithAPI(732, menuID, "版本列表", "report_manage_versions", enum.Line, "versions", "history", "primary", 94, "/admin/report/:id/versions", "GET"),
}

buttons := []model.SysMenuButton{
	menuButtonWithAPI(714, menuID, "设计时预览", "report_design_design_preview", enum.Top, "preview", "preview", "primary", 3, "/admin/report/:id/design-preview", "POST"),
	menuButtonWithAPI(715, menuID, "发布", "report_design_publish", enum.Top, "publish", "published_with_changes", "primary", 4, "/admin/report/:id/publish", "POST"),
	apiPermissionWithAPI(716, menuID, "版本列表", "report_design_versions", enum.Top, "versions", "history", "primary", 95, "/admin/report/:id/versions", "GET"),
}
```

### 8.3 Casbin policy 种子

- 文件：`backend/migrate/main.go`
- 函数：`seedSuperAdminRoutePolicies`
- 说明：super_admin 路由策略包含新增接口。

```go
{"/admin/report/:id/preview", "POST"},
{"/admin/report/:id/design-preview", "POST"},
{"/admin/report/:id/publish", "POST"},
{"/admin/report/:id/run", "POST"},
{"/admin/report/:id/versions", "GET"},
{"/admin/report/sql-fields", "POST"},
```

---

## 9. 测试结果

### 9.1 已执行命令

```bash
gofmt -w backend/model/report.go backend/dto/request/report_req.go backend/dto/response/report_res.go backend/repository/report.go backend/repository/impl/report_impl.go backend/service/report_service.go backend/controller/report_controller.go backend/initialize/router.go backend/initialize/wire.go backend/migrate/main.go
go run github.com/google/wire/cmd/wire ./initialize
go test ./...
```

### 9.2 `go test ./...` 结果摘要

执行目录：`/Users/nan/project/sweet-auth-base/backend`

结果：通过。

关键摘要：

```text
?   	backend	[no test files]
ok  	backend/api	0.576s
ok  	backend/cmd/db-preflight	1.665s
ok  	backend/controller	0.582s
ok  	backend/initialize	2.162s
ok  	backend/internal/reportconfig	(cached)
ok  	backend/middleware	2.781s
ok  	backend/migrate	3.555s
ok  	backend/repository/impl	(cached)
ok  	backend/service	4.323s
```

### 9.3 自动化测试覆盖说明

- 本次未新增 V1-A 专项单元测试文件。
- 现有 `go test ./...` 能覆盖编译、wire 生成后的依赖注入、迁移包编译、已有 service/controller/middleware 测试。
- 建议 V1-B 前补充 V1-A 专项测试：
  - draft 可以 design-preview 但不能 run。
  - published run 使用 version 快照。
  - 修改草稿不影响 run。
  - disabled 不可 design-preview/run。
  - execution_log action 和 params.runtime 结构。

---

## 10. 已知风险

### 10.1 V1-A 尚未覆盖

- 未实现后端导出。
- 未实现 Excel / CSV 导出。
- 未引入 `report_datasource`。
- 未引入 `report_dataset`。
- 未支持外部数据库数据源。
- 未实现完整 AST SQL 白名单。
- 未新增 `report_execution_log.version_id` 字段。
- 未做前端复杂 UI 改造。
- 未做自由画布增强、图表大屏、打印分页、填报、定时调度、邮件订阅。

### 10.2 留到 V1-B / V1-C

V1-B 建议：

- 后端受控导出。
- 查询超时配置。
- 执行日志增加 `version_id` 字段。
- V1-A 专项自动化测试。
- 前端最小入口调整：publish、design-preview、run、versions。

V1-C 建议：

- `report_datasource`。
- `report_dataset`。
- 外部数据库只读连接。
- SQL AST 白名单。
- 数据集复用和版本。
- 异步导出任务。
