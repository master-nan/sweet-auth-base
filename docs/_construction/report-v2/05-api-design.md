# Report V2 API 设计

## 总体原则

后端 API 不使用 `/report-v2`。

正式 API 继续使用 `/admin/report` 系列。

原因：

1. 后端 API 表达的是领域能力，不是前端版本。
2. V1-A / V1-B 已完成的发布、运行、导出、版本能力本身是正式能力。
3. V2 主要重做前端产品体验，不需要为了前端重构复制一套 `/report-v2` 后端 API。
4. 避免长期维护两套后端 API。

如果未来出现完全不兼容的新后端 API，再考虑版本化；当前 Report V2 不需要。

## 第一阶段继续使用的正式 API

### 报表定义 API

```http
POST /admin/report/query
GET  /admin/report/:id
POST /admin/report
PUT  /admin/report/:id
DELETE /admin/report/:id
```

职责：

1. 查询报表列表。
2. 获取报表详情。
3. 创建报表草稿。
4. 更新报表草稿。
5. 删除报表。

### 数据源发现 API

```http
GET /admin/report/data-sources
```

第一阶段继续返回 `sys_table / sys_table_field`。

该接口是数据源发现，不是独立 `report_datasource` 管理。

### 设计时预览 API

```http
POST /admin/report/:id/design-preview
```

规则：

1. 读取 `report_definition` 草稿。
2. 允许 draft / published。
3. 不允许 disabled。
4. action = `design_preview`。

### 发布 API

```http
POST /admin/report/:id/publish
```

规则：

1. 校验配置。
2. 校验 SQL 安全。
3. 创建 `report_definition_version` 快照。
4. 更新 `published_version_id`。
5. 更新状态为 published。

### 运行 API

```http
POST /admin/report/:id/run
```

规则：

1. 只允许 published 报表。
2. 读取发布版本快照。
3. 不读取草稿配置。
4. 应用权限和数据权限。
5. 分页返回。
6. action = `runtime_run`。

### 导出 API

```http
POST /admin/report/:id/export
```

规则：

1. 只允许 published 报表。
2. 读取发布版本快照。
3. 不读取草稿配置。
4. 应用权限和数据权限。
5. 限制最大导出行数。
6. action = `runtime_export`。

### 版本 API

```http
GET /admin/report/:id/versions
```

职责：

1. 查询发布版本列表。
2. 标记当前版本。
3. 第一阶段只读，不做回滚。

### 执行日志 API

第一阶段可以复用现有日志中心或后台通用查询能力。

未来建议补充：

```http
POST /admin/report-execution-log/query
GET  /admin/report-execution-log/:id
```

职责：

1. 查询设计预览、运行、导出日志。
2. 按报表、用户、时间、action、success 过滤。
3. 排查失败原因和慢查询。

## 数据集 API 预留

第一阶段不强行实现数据集 API。

未来建议：

```http
POST /admin/report-dataset/query
POST /admin/report-dataset
PUT  /admin/report-dataset/:id
GET  /admin/report-dataset/:id
POST /admin/report-dataset/:id/fields
```

职责：

1. 管理可复用数据集。
2. 解析字段。
3. 管理参数 schema。
4. 管理数据集权限。

## 数据源 API 预留

第一阶段不实现外部数据源。

未来建议：

```http
POST /admin/report-datasource/query
POST /admin/report-datasource
PUT  /admin/report-datasource/:id
GET  /admin/report-datasource/:id
POST /admin/report-datasource/:id/test
```

职责：

1. 管理外部数据库连接。
2. 加密存储凭据。
3. 测试连接。
4. 控制数据源可见范围。

## V2 前端对 API 的使用策略

V2 前端第一阶段：

1. 继续调用 `/admin/report/query`。
2. 继续调用 `/admin/report/data-sources`。
3. 继续调用 `/admin/report/:id/design-preview`。
4. 继续调用 `/admin/report/:id/publish`。
5. 继续调用 `/admin/report/:id/run`。
6. 继续调用 `/admin/report/:id/export`。
7. 继续调用 `/admin/report/:id/versions`。

不新增 `/admin/report-v2`。
