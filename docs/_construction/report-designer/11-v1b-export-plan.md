# 报表模块 V1-B 后端受控导出实现计划

## 一、阶段目标

V1-B 只做后端受控导出，目标是在 V1-A 已完成的发布版本和运行态隔离基础上，新增：

1. `POST /admin/report/:id/export`
2. 只允许 `published` 报表导出。
3. 导出只读取 `report_definition.published_version_id`。
4. 导出只读取 `report_definition_version` 快照。
5. 导出不读取草稿 `query_config` / `layout_config`。
6. 导出复用 V1-A 的 `ReportExecutionSnapshot` 或等价快照结构。
7. 导出复用现有 SQL 安全校验。
8. 导出复用现有 table dataset 数据权限。
9. 导出限制最大行数。
10. 导出写 `report_execution_log`，`action = runtime_export`。
11. 第一版只支持 CSV。

## 二、V1-B 暂不做

本阶段明确不做：

1. 不做前端复杂 UI。
2. 不做 `report_datasource`。
3. 不做 `report_dataset`。
4. 不做外部数据库。
5. 不做异步导出任务。
6. 不做导出任务中心。
7. 不做 PDF。
8. 不做 Excel 模板。
9. 不做打印分页。
10. 不做邮件发送。
11. 不做定时订阅。
12. 不做 SQL AST 白名单。
13. 不做数据集实体化。
14. 不做 XLSX。

## 三、当前代码可复用点

### 1. 复用 ReportExecutionSnapshot

V1-A 已新增 `ReportExecutionSnapshot`，导出继续复用同一结构：

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

V1-B 只需新增运行态常量：

```go
reportRuntimeExport = "runtime_export"
```

导出必须从发布版本构造快照：

```go
snapshot := reportSnapshotFromVersion(version, reportRuntimeExport)
```

### 2. 复用 RunReport 的发布版本读取逻辑

`RunReport` 已完成以下校验：

1. `report_definition` 存在。
2. `report_definition.state = true`。
3. `report_definition.status = published`。
4. `published_version_id > 0`。
5. `report_definition_version` 存在。
6. `version.State = true`。
7. `version.Status = published`。

建议抽出内部 helper：

```go
func (s *ReportService) loadPublishedReportSnapshot(
    ctx *gin.Context,
    reportId int,
    runtimeType string,
) (ReportExecutionSnapshot, error)
```

`RunReport` 和 `ExportReport` 共用该 helper，避免两处状态判断漂移。

### 3. executeReportSnapshot 不适合原样直接复用

当前 `executeReportSnapshot` 同时负责：

1. 解析配置。
2. SQL 安全校验。
3. SQL dataset 权限校验。
4. table / SQL / join 查询。
5. pageSize 限制。
6. 构造 preview response。
7. 写 execution log。

导出不能原样直接调用，原因：

1. 运行接口 pageSize 上限是 200，导出需要 5000 或 10000。
2. 导出日志必须在 CSV 构建成功后才写成功。
3. 导出失败包括状态错误、权限错误、超过行数限制、查询失败、CSV 构建失败。
4. 导出 action 必须是 `runtime_export`。

### 4. 新增 options 以最小化重构

建议新增内部 options：

```go
type ReportExecutionOptions struct {
    MaxRows              int
    PageSizeLimit        int
    DefaultPageSize      int
    ExportMode           bool
    WriteLog             bool
    DataPermissionAction enum.SysMenuButtonEventAction
}
```

保留现有方法签名作为 wrapper：

```go
func (s *ReportService) executeReportSnapshot(
    ctx *gin.Context,
    snapshot ReportExecutionSnapshot,
    req request.ReportPreviewReq,
    start time.Time,
) (response.ReportPreviewRes, error)
```

内部调用 options 版本：

```go
executeReportSnapshotWithOptions(ctx, snapshot, req, start, runOptions)
```

导出调用同一查询路径，但关闭自动日志：

```go
executeReportSnapshotWithOptions(ctx, snapshot, req, start, exportOptions)
```

### 5. 避免复制三套查询逻辑

不要复制 table / SQL / join 三套查询逻辑。

V1-B 应在当前函数上做小幅参数化：

1. `previewSQLDataset` 增加 options 参数。
2. `previewJoinedTableDatasets` 增加 options 参数。
3. 普通 table dataset 查询使用 options 控制 `Page` / `Num`。
4. SQL 和 join 路径使用 options 控制 limit。
5. 数据权限注入继续复用现有方法。

## 四、文件修改计划

预计修改以下文件：

1. `backend/dto/request/report_req.go`
   - 新增 `ReportExportReq`。

2. `backend/dto/response/report_res.go`
   - 如需要新增内部返回结构 `ReportExportFile`。
   - 也可以只在 service 包内定义非 DTO 结构，避免扩大 API JSON 响应面。

3. `backend/service/report_service.go`
   - 新增 `reportRuntimeExport`。
   - 新增 `ExportReport`。
   - 新增 `ReportExecutionOptions`。
   - 新增导出请求归一化。
   - 新增发布版本 snapshot 读取 helper。
   - 新增 CSV 构建函数。
   - 调整查询路径以支持导出行数限制。
   - 写 `runtime_export` execution log。

4. `backend/controller/report_controller.go`
   - 新增 `ExportReport` controller 方法。
   - 返回 CSV 文件流。

5. `backend/initialize/router.go`
   - 新增：

```go
adminGroup.POST("/report/:id/export", app.ReportController.ExportReport)
```

6. `backend/migrate/main.go`
   - 新增导出按钮权限。
   - 新增 super_admin Casbin policy。

7. `backend/service/report_service_v1b_test.go`
   - 新增 V1-B service 层测试。

## 五、请求设计

新增 `ReportExportReq`：

```go
type ReportExportReq struct {
    Format     string         `json:"format"`
    MenuId     int            `json:"menu_id"`
    DatasetId  string         `json:"dataset_id"`
    Parameters map[string]any `json:"parameters"`
    Params     map[string]any `json:"params"`
    Query      Basic          `json:"query"`
    MaxRows    *int           `json:"max_rows"`
    MaxRowsAlt *int           `json:"maxRows"`
}
```

说明：

1. `format` 第一版只支持 `csv`，空值默认 `csv`。
2. `dataset_id` 沿用运行逻辑。
3. `parameters` 沿用 `ReportPreviewReq`。
4. `params` 用于兼容文档中已有命名，但进入 service 后必须归一化到 `parameters`。
5. `query` 复用 `Basic` 的 filters、expressions、quick_query、order、menu_id 等。
6. 导出忽略用户传入的分页，后端强制 `Page = 1`。
7. 后续逻辑只使用 `effectiveMaxRows`，不继续传递 `max_rows` 和 `maxRows` 两个字段。

### max_rows / maxRows 归一化规则

必须新增内部归一化函数，例如：

```go
func normalizeReportExportMaxRows(req request.ReportExportReq) (int, error)
```

规则：

1. 如果 `max_rows` 和 `maxRows` 都传了，且值不同，返回参数错误。
2. 如果只传一个，就使用该值。
3. 如果都不传，默认 `5000`。
4. 如果传入值 `<= 0`，默认 `5000`。
5. 如果传入值 `> 10000`，直接返回参数错误。
6. 归一化后只使用 `effectiveMaxRows`。

建议常量：

```go
const defaultReportExportMaxRows = 5000
const maxReportExportRows = 10000
```

## 六、响应设计

### 1. CSV 文件流

Controller 直接返回文件流，不走 `resp.SetData`：

```go
ctx.Header("Content-Type", "text/csv; charset=utf-8")
ctx.Header("Content-Disposition", contentDisposition("attachment", fileName))
ctx.Data(http.StatusOK, "text/csv; charset=utf-8", file.Content)
```

### 2. Content-Type

```http
Content-Type: text/csv; charset=utf-8
```

### 3. Content-Disposition

建议复用 `controller/file_controller.go` 中同 package 的 `contentDisposition`：

```go
contentDisposition("attachment", fileName)
```

文件名建议：

```text
{report_code}_v{version_no}.csv
```

中文报表名如果进入文件名，必须使用 `filename*=UTF-8''...`。

### 4. 错误响应

状态错误、权限错误、参数错误、超过行数限制、查询失败、CSV 构建失败，必须在写文件 header 前返回错误，并继续走统一 JSON 错误响应。

### 5. 导出元信息

第一版不返回 JSON 元信息。导出行数、版本、runtime_type 写入 `report_execution_log`。

## 七、导出读取来源硬约束

`ExportReport` 必须遵守：

1. 只能读取 `report_definition.published_version_id`。
2. 只能读取 `report_definition_version.query_config`。
3. 只能读取 `report_definition_version.layout_config`。
4. 不能读取 `report_definition.query_config`。
5. 不能读取 `report_definition.layout_config`。
6. 必须复用 V1-A 的 `ReportExecutionSnapshot` 或等价快照结构。
7. 公共执行方法只能接收 snapshot，不能在执行过程中重新读取草稿配置。

导出快照构造只能来自：

```go
snapshot := reportSnapshotFromVersion(version, reportRuntimeExport)
```

## 八、导出执行流程

新增：

```go
func (s *ReportService) ExportReport(
    ctx *gin.Context,
    reportId int,
    req request.ReportExportReq,
) (ReportExportFile, error)
```

流程：

1. 初始化 `start := time.Now()`。
2. 初始化 `snapshot := ReportExecutionSnapshot{ReportId: reportId, RuntimeType: reportRuntimeExport}`，用于早期失败日志。
3. 校验 `format`，空值默认 `csv`，非 `csv` 返回参数错误。
4. 归一化 `max_rows` / `maxRows` 为 `effectiveMaxRows`。
5. 读取 `report_definition`。
6. 校验报表存在。
7. 校验 `report.State = true`。
8. 校验 `report.Status = published`。
9. 校验 `published_version_id > 0`。
10. 读取 `report_definition_version`。
11. 校验 `version.State = true`。
12. 校验 `version.Status = published`。
13. 使用 `reportSnapshotFromVersion(version, reportRuntimeExport)` 构造 snapshot。
14. 将 `ReportExportReq` 转换为内部 `ReportPreviewReq`。
15. 使用导出 options 执行查询。
16. 校验总行数不超过 `effectiveMaxRows`。
17. 构建 CSV。
18. 写成功 execution log。
19. 返回文件内容。

## 九、导出行数限制策略

不要静默截断。

规则：

1. 不传 `max_rows`：默认 `5000`。
2. `max_rows <= 0`：默认 `5000`。
3. `max_rows > 10000`：直接返回参数错误。
4. 查询总数 `total > effectiveMaxRows`：直接返回错误：

```text
导出行数超过系统限制，请缩小查询条件后重试
```

5. 不允许只导出前 N 行然后让用户误以为导出了完整数据。

### total 获取策略

优先使用现有查询结果中的 `Total`：

1. 普通 table dataset：`generalizationService.Query` 当前返回 `Total`。
2. SQL dataset：`queryReportSQL` 当前会先 count，返回 `total`。
3. join dataset：`previewJoinedTableDatasets` 当前会先 count，返回 `total`。

如果后续某条查询路径无法可靠拿到 `total`，使用 `effectiveMaxRows + 1` 的查询方式判断是否超限：

1. 请求 `limit = effectiveMaxRows + 1`。
2. 如果返回行数大于 `effectiveMaxRows`，直接报错。
3. 不生成部分 CSV。

## 十、CSV 安全

第一版只支持 CSV，必须防 Excel 公式注入。

规则：

1. 使用标准库 `encoding/csv`。
2. CSV 使用 UTF-8。
3. 可以加 UTF-8 BOM 以兼容 Excel 中文。
4. 表头也必须经过安全处理。
5. 字符串值 `strings.TrimSpace` 后如果以以下字符开头，前置单引号：
   - `=`
   - `+`
   - `-`
   - `@`

建议函数：

```go
func safeReportCSVCell(value any) string {
    text := fmt.Sprint(value)
    if text == "<nil>" {
        return ""
    }
    trimmed := strings.TrimSpace(text)
    if trimmed != "" && strings.ContainsAny(trimmed[:1], "=+-@") {
        return "'" + text
    }
    return text
}
```

表头生成时也调用同一安全函数。

CSV 列顺序以 `response.ReportPreviewColumn` 顺序为准：

1. 表头使用 `column.Label`，为空则用 `column.Name`，再为空用 `column.Field`。
2. 数据行使用 `row[column.Field]`。
3. 不额外导出未在 columns 中声明的字段。

## 十一、数据权限 action

当前项目已经存在：

```go
enum.ButtonActionExport
```

V1-B 导出应优先使用它：

```go
DataPermissionAction: enum.ButtonActionExport
```

规则：

1. 如果项目已有 `enum.ButtonActionExport` 或等价值，则使用它。
2. 如果未来分支中没有该值，不要因为 V1-B 随意新增并扩大权限模型。
3. 没有 export action 时，可以暂时复用 query action，并在文档和代码注释中标记后续完善。
4. 不允许因为 V1-B 导出破坏现有 table dataset 数据权限逻辑。

## 十二、SQL dataset 处理

SQL dataset 导出继续沿用 V1-A 的严格策略：

1. 复用 `validateReportSQLDatasets`。
2. 复用 `safeReportPreviewSQL`。
3. 复用 `ensureSQLDatasetRole`。
4. 非 `super_admin` 不能导出 SQL dataset。
5. 第一版不承诺 SQL dataset 自动数据权限注入。
6. 不实现 SQL AST 白名单。

## 十三、导出日志

导出必须写 `report_execution_log`。

### 成功日志

规则：

1. `action = runtime_export`
2. `success = true`
3. `row_count = 导出行数`
4. `error_message = ""`

`params.runtime` 固定包含：

```json
{
  "runtime_type": "runtime_export",
  "version_id": 123,
  "version_no": 2
}
```

### 失败日志

以下失败都必须写日志：

1. 状态错误。
2. 权限错误。
3. 参数错误。
4. 超过行数限制。
5. 查询失败。
6. CSV 构建失败。

失败日志规则：

1. `action = runtime_export`
2. `success = false`
3. `error_message = err.Error()`
4. `row_count = 0`

V1-B 统一规定失败时 `row_count = 0`。如果后续要记录实际查询到的行数，需要在单独字段或后续版本中明确语义，避免不同失败路径不一致。

### 日志参数结构

`writeExecutionLog` 当前参数类型是 `ReportPreviewReq`。V1-B 可选两种方式：

1. 将 `ReportExportReq` 归一化为 `ReportPreviewReq` 后写入 request。
2. 将 `writeExecutionLog` 扩展为接收 `any` request，但保持 `runtime` 结构不变。

为最小改动，建议第一版使用方案 1。

## 十四、权限和路由

### 路由

新增路由必须位于 `adminGroup` 下：

```go
adminGroup.POST("/report/:id/export", app.ReportController.ExportReport)
```

因此会继续走：

1. `AuthHandler`
2. `CasbinHandler`

### Casbin policy

`backend/migrate/main.go` 的 super_admin policy 种子需要新增：

```go
{"/admin/report/:id/export", "POST"}
```

### 菜单按钮权限

建议新增按钮：

1. `report_center_export`
   - 菜单：报表中心
   - 路由：`POST /admin/report/:id/export`
   - action：`export`

2. `report_manage_export`
   - 菜单：报表管理
   - 路由：`POST /admin/report/:id/export`
   - action：`export`

是否给设计器页增加导出按钮，V1-B 不做复杂 UI，后续再定。

### 导出频率限制

V1-B 暂不做导出频率限制。

原因：

1. 当前阶段没有异步导出任务中心。
2. 没有导出任务表和用户级限流配置。
3. 第一版先通过最大行数限制和权限控制降低风险。

## 十五、测试计划

新增 `backend/service/report_service_v1b_test.go`，优先复用 `report_service_v1a_test.go` 中测试环境 helper。

至少覆盖：

1. draft 报表不能 export。
2. disabled 报表不能 export。
3. published 报表可以 export。
4. export 读取 `report_definition_version` 快照。
5. 修改草稿后 export 仍导出旧版本。
6. 再次 publish 后 export 导出新版本。
7. `version.status != published` 时 export 拒绝。
8. SQL dataset 非 super_admin 不能 export。
9. export 写 `report_execution_log`，`action = runtime_export`。
10. `execution_log.params.runtime` 包含 `runtime_type`、`version_id`、`version_no`。
11. `max_rows` 默认值生效。
12. `max_rows > 10000` 返回参数错误。
13. `total > effectiveMaxRows` 返回：

```text
导出行数超过系统限制，请缩小查询条件后重试
```

14. 不发生静默截断。
15. CSV 表头正确。
16. CSV 数据行正确。
17. CSV 表头和数据都防公式注入。
18. `go test ./...` 在 `backend` module 下通过。

## 十六、风险点

1. 当前查询逻辑和日志写入耦合较强，V1-B 需要小幅重构 `executeReportSnapshot`。
2. 必须保留原 wrapper，避免破坏 V1-A 测试和行为。
3. CSV 第一版一次性生成 `[]byte`，存在内存风险，但通过 5000 / 10000 行限制控制。
4. SQL dataset 的 count 可能较慢，V1-B 不做复杂 SQL 优化。
5. 文件写出阶段如果客户端断开，Gin `ctx.Data` 不易可靠写失败日志；V1-B 成功日志以查询和 CSV 构建成功为准。
6. 异步导出、任务中心、文件落库、下载审计、导出频率限制留到后续阶段。

## 十七、验收标准

V1-B 完成后必须满足：

1. `POST /admin/report/:id/export` 可用。
2. draft 报表不能导出。
3. disabled 报表不能导出。
4. 只有 published 报表能导出。
5. 导出只读取发布版本快照。
6. 修改草稿不会影响导出。
7. 再次发布后导出新版本。
8. SQL dataset 非 super_admin 不能导出。
9. table dataset 导出继续应用数据权限。
10. 导出行数超过限制时直接报错，不静默截断。
11. CSV 防 Excel 公式注入。
12. 导出成功和失败都写 `runtime_export` 日志。
13. 不引入 V1-B 明确暂不做的能力。
