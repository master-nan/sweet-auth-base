# 报表设计器第一阶段版本与运行态设计

## 设计目标

第一阶段需要解决当前报表草稿和运行态混用的问题。

核心目标：

1. 草稿可以继续编辑。
2. 设计器可以读取草稿做设计时预览。
3. 发布时生成不可变版本快照。
4. 报表中心只运行已发布版本。
5. 草稿修改不影响已发布运行结果。

## 数据模型

### `report_definition`

继续作为报表主表和草稿表。

保留字段：

- `code`
- `name`
- `description`
- `category`
- `status`
- `source_type`
- `source_code`
- `permission_menu_id`
- `permission_table_code`
- `query_config`
- `layout_config`
- `remark`

新增字段：

- `published_version_id`

说明：

- `query_config` / `layout_config` 保存当前草稿。
- `published_version_id` 指向当前线上运行版本。
- `status = published` 表示存在有效发布版本。
- `status = disabled` 表示禁止运行和导出。

### `report_definition_version`

新增发布版本快照表。

建议字段：

| 字段 | 说明 |
|---|---|
| `id` | 主键 |
| `report_id` | 关联 `report_definition.id` |
| `version_no` | 版本号，按报表递增 |
| `report_code` | 发布时的报表编码 |
| `report_name` | 发布时的报表名称 |
| `description` | 发布时的描述 |
| `category` | 发布时的分类 |
| `source_type` | 发布时的来源类型 |
| `source_code` | 发布时的来源编码 |
| `permission_menu_id` | 发布时的权限菜单上下文 |
| `permission_table_code` | 发布时的数据权限表编码 |
| `query_config` | 发布时查询配置快照 |
| `layout_config` | 发布时布局配置快照 |
| `status` | 版本状态，例如 `published`、`archived` |
| `published_at` | 发布时间 |
| `published_by` | 发布人 ID |
| `published_name` | 发布人名称 |
| `change_log` | 发布说明 |
| `state` | 通用状态 |
| 审计字段 | 与现有 `Basic` 风格保持一致 |

## 状态规则

### `draft`

- 可保存草稿。
- 可设计时预览。
- 不可在报表中心运行。
- 不可导出。
- 可以发布。

### `published`

- 可保存新的草稿修改。
- 可设计时预览草稿。
- 可在报表中心运行当前 `published_version_id`。
- 可导出当前 `published_version_id`。
- 再次发布时生成新版本，并更新 `published_version_id`。

### `disabled`

- 不可设计时预览。
- 不可运行。
- 不可导出。
- 可以在管理端重新启用或重新发布，具体交互由业务决定。

## 流程设计

### 1. 保存草稿

输入：

- 报表基础信息
- `query_config`
- `layout_config`

处理：

1. 校验基础字段。
2. 校验 `query_config` 基本结构。
3. 校验 `layout_config` 基本结构。
4. 保存到 `report_definition`。

输出：

- 当前草稿定义。

注意：

- 保存草稿不生成版本。
- 保存草稿不影响 `published_version_id`。

### 2. 设计时预览

读取来源：

- `report_definition.query_config`
- `report_definition.layout_config`

允许状态：

- `draft`
- `published`

禁止状态：

- `disabled`

处理规则：

1. 校验用户有设计或管理权限。
2. 读取草稿配置。
3. 校验 SQL 安全。
4. 应用现有数据权限。
5. 强制分页。
6. 写执行日志。

日志 action：

- `design_preview`

### 3. 发布

读取来源：

- `report_definition`

处理规则：

1. 校验用户有发布权限。
2. 校验 `query_config`。
3. 校验 `layout_config`。
4. 校验 SQL 安全。
5. 创建 `report_definition_version` 快照。
6. 更新 `report_definition.published_version_id`。
7. 更新 `report_definition.status = published`。

结果：

- 新版本成为报表中心运行版本。
- 历史版本保留。
- 草稿后续修改不影响当前发布版本。

### 4. 运行

读取来源：

- `report_definition.published_version_id`
- `report_definition_version.query_config`
- `report_definition_version.layout_config`

允许状态：

- `published`

禁止状态：

- `draft`
- `disabled`

处理规则：

1. 校验用户有运行权限。
2. 校验报表状态是 `published`。
3. 读取发布版本快照。
4. 校验 SQL 安全。
5. 应用数据权限。
6. 强制分页。
7. 执行查询。
8. 写执行日志。

日志 action：

- `runtime_run`

### 5. 导出

读取来源：

- `report_definition.published_version_id`
- `report_definition_version.query_config`
- `report_definition_version.layout_config`

允许状态：

- `published`

处理规则：

1. 校验用户有导出权限。
2. 校验报表状态是 `published`。
3. 读取发布版本快照。
4. 校验 SQL 安全。
5. 应用数据权限。
6. 限制最大导出行数。
7. 生成后端导出文件。
8. 写执行日志。

日志 action：

- `runtime_export`

## 兼容现有 preview

当前已有 `/admin/report/:id/preview`。第一阶段可以保留兼容，但不应继续作为报表中心生产运行接口。

建议：

- 新设计时预览使用 `/admin/report/:id/design-preview`。
- 报表中心运行使用 `/admin/report/:id/run`。
- 后端导出使用 `/admin/report/:id/export`。
- 旧 `/preview` 可作为兼容入口，后续逐步下线或只映射到设计预览。

## 不变量

以下规则必须始终成立：

1. 运行态不读取草稿配置。
2. 导出不读取草稿配置。
3. 修改草稿不改变已发布运行结果。
4. disabled 报表不能运行。
5. disabled 报表不能导出。
6. 每次运行必须写日志。
7. 每次导出必须写日志。
