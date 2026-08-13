# 报表设计器第一阶段 API 设计

## 设计原则

第一阶段 API 以增量演进为主，不破坏现有报表管理和设计器基本能力。

原则：

1. 保留现有报表定义 CRUD。
2. 新增设计时预览接口。
3. 新增发布接口。
4. 新增运行接口。
5. 新增后端导出接口。
6. 新增版本列表接口。
7. 不新增数据源管理接口。
8. 不新增数据集管理接口。

## 现有接口保留

当前接口继续保留：

- `POST /admin/report/query`
- `GET /admin/report/data-sources`
- `POST /admin/report/sql-fields`
- `GET /admin/report/:id`
- `POST /admin/report`
- `PUT /admin/report/:id`
- `POST /admin/report/:id/status`
- `DELETE /admin/report/:id`

说明：

- `/admin/report/data-sources` 继续返回 `sys_table` / `sys_table_field` 派生的数据源。
- `/admin/report/sql-fields` 继续用于 SQL 字段推断，但必须复用 SQL 安全守卫。

## 新增接口

### 1. 设计时预览

```http
POST /admin/report/:id/design-preview
```

用途：

- 设计器内预览草稿配置。

读取：

- `report_definition.query_config`
- `report_definition.layout_config`

允许状态：

- `draft`
- `published`

禁止状态：

- `disabled`

日志 action：

- `design_preview`

请求示例：

```json
{
  "page": 1,
  "pageSize": 20,
  "dataset_id": "main",
  "params": {
    "keyword": "admin"
  }
}
```

响应示例：

```json
{
  "columns": [],
  "rows": [],
  "total": 0,
  "page": 1,
  "pageSize": 20,
  "meta": {
    "report_id": 1,
    "runtime_type": "design_preview"
  }
}
```

### 2. 发布

```http
POST /admin/report/:id/publish
```

用途：

- 将当前草稿发布为版本快照。

处理：

1. 校验 `query_config`。
2. 校验 `layout_config`。
3. 校验 SQL 安全。
4. 创建 `report_definition_version`。
5. 更新 `report_definition.published_version_id`。
6. 更新 `report_definition.status = published`。

请求示例：

```json
{
  "change_log": "首次发布"
}
```

响应示例：

```json
{
  "report_id": 1,
  "version_id": 10,
  "version_no": 3,
  "status": "published"
}
```

### 3. 运行

```http
POST /admin/report/:id/run
```

用途：

- 报表中心运行已发布版本。

读取：

- `report_definition.published_version_id`
- `report_definition_version.query_config`
- `report_definition_version.layout_config`

允许状态：

- `published`

日志 action：

- `runtime_run`

请求示例：

```json
{
  "page": 1,
  "pageSize": 20,
  "params": {
    "keyword": "admin"
  }
}
```

响应示例：

```json
{
  "columns": [],
  "rows": [],
  "total": 0,
  "page": 1,
  "pageSize": 20,
  "meta": {
    "report_id": 1,
    "version_id": 10,
    "runtime_type": "runtime_run"
  }
}
```

### 4. 导出

```http
POST /admin/report/:id/export
```

用途：

- 后端受控导出已发布报表。

读取：

- `report_definition.published_version_id`
- `report_definition_version.query_config`
- `report_definition_version.layout_config`

允许状态：

- `published`

日志 action：

- `runtime_export`

请求示例：

```json
{
  "format": "xlsx",
  "params": {
    "keyword": "admin"
  },
  "maxRows": 5000
}
```

响应方式：

- 第一阶段可直接返回文件流。
- 如果后续导出耗时较长，再升级为异步任务。

限制：

- 必须限制最大导出行数。
- 必须校验权限。
- 必须应用数据权限。

### 5. 版本列表

```http
GET /admin/report/:id/versions
```

用途：

- 查看报表发布版本历史。

响应示例：

```json
{
  "list": [
    {
      "id": 10,
      "report_id": 1,
      "version_no": 3,
      "status": "published",
      "published_at": "2026-07-03T10:00:00+08:00",
      "published_by": 1,
      "published_name": "admin",
      "change_log": "首次发布"
    }
  ]
}
```

## 不新增接口

第一阶段不新增：

- `report_datasource` CRUD 接口
- `report_dataset` CRUD 接口
- 外部数据库连接测试接口
- 数据集版本接口
- 图表大屏接口
- 打印接口
- 填报接口
- 调度订阅接口

## 兼容策略

当前已有：

```http
POST /admin/report/:id/preview
```

建议第一阶段保留兼容，但不再作为报表中心运行接口。

兼容策略：

- 设计器逐步改用 `/design-preview`。
- 报表中心改用 `/run`。
- 导出改用 `/export`。
- `/preview` 后续只保留为兼容入口或映射为设计预览。

## 错误码建议

建议补充以下业务错误：

| 错误 | 场景 |
|---|---|
| `REPORT_DISABLED` | 报表已停用 |
| `REPORT_NOT_PUBLISHED` | 报表未发布 |
| `REPORT_VERSION_NOT_FOUND` | 发布版本不存在 |
| `REPORT_SQL_UNSAFE` | SQL 安全校验失败 |
| `REPORT_EXPORT_LIMIT_EXCEEDED` | 超过导出行数限制 |
| `REPORT_PERMISSION_DENIED` | 没有报表权限 |
| `REPORT_DATA_SCOPE_DENIED` | 数据权限不足 |

## 日志 action

接口和日志 action 对应关系：

| 接口 | action |
|---|---|
| `/admin/report/:id/design-preview` | `design_preview` |
| `/admin/report/:id/run` | `runtime_run` |
| `/admin/report/:id/export` | `runtime_export` |
