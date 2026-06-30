# 通用数据权限设计与实现说明

## 背景

底座的数据权限不能写死为公司、组织或某个行业对象。不同项目可能按租户、项目、范围、课程、负责人、创建人或自定义业务字段隔离数据。

## 结论

`sweet-auth-base` 已按“通用数据权限模型”落地，不再把公司、部门、HRDB 这类业务实体写进底座。

当前模型的核心是：

1. 权限维度可配置，例如租户、项目、课程、业务范围、负责人、创建人。
2. 已绑定数据表的菜单声明自己受哪个维度控制，以及维度落在哪个字段上。
3. 角色或用户只保存结构化的数据范围，不保存 SQL。
4. 低代码列表、详情、新增、编辑、删除都走统一的数据权限解析；已接入的固定系统页复用同一套规则。
5. 控制器不自己拼 `tenant_id`、`owner_id` 之类的条件。

这套模型能做 SaaS、企业中台、在线学习、CRM 或其他业务系统。区别只在配置的数据权限维度和绑定字段不同，底座代码不需要知道具体业务是什么。

## 当前状态

当前系统已经实现的是菜单权限、按钮权限、接口权限，以及通用数据权限基础模型：

- 维度由 `sys_data_dimension` 定义，不写死具体业务实体。
- 菜单通过 `sys_data_scope_binding` 绑定维度、表编码、字段编码和生效动作。
- 角色通过 `sys_role_data_scope` 保存常规范围策略。
- 用户通过 `sys_user_data_scope_override` 保存临时覆盖或收窄/扩展。
- 低代码列表、详情、创建、更新、删除会复用解析后的结构化范围做行级检查。
- 已接入的固定系统页只要菜单绑定了数据表，也可以配置数据权限；列表查询和记录检查会应用同一套结构化范围。
- 角色权限保存时可以同时保存菜单、按钮和角色数据权限，避免权限保存到一半的中间态。

树形范围使用 `tree` 策略保存根节点，并按维度来源表的 `value_field` / `parent_field` 展开下级。底座仍不内置公司、部门或其他具体业务实体。

不要在文件控制器或某个业务控制器里再临时写一套数据归属判断来替代通用数据权限。导出、批量操作、文件预览和下载如果涉及业务记录归属，也应带上业务上下文并复用同一套规则。

文件访问权限负责判断文件本身是否存在、是否允许预览或下载；当请求携带业务记录上下文时，会回到统一数据权限层判断对应业务记录是否可访问。

## 固定字段权限与当前模型

第一版容易做成固定业务字段权限：

```text
用户 -> 租户列表 -> 查询时拼 tenant_id IN (...)
```

这种做法适合单个业务系统，但不适合底座。因为在线学习平台可能按 `tenant_id`、`course_id` 或 `school_id` 控制，CRM 可能按 `owner_id` 或 `region_id` 控制，通用项目管理也可能按 `scope_id` 或 `project_id` 控制。

当前模型改成配置化维度：

```text
用户/角色 -> 数据权限范围 -> 菜单绑定维度 -> 字段 -> 统一查询条件
```

例如：

| 项目场景 | 维度 | 菜单绑定字段 | 范围值 |
| --- | --- | --- | --- |
| 示例中心 | 业务范围 | `scope_id` | 1、2 |
| 示例中心 | 项目 | `project_id` | 1001、1002 |
| 在线学习 | 学校 | `school_id` | 2001 |
| 在线学习 | 课程 | `course_id` | 3001、3002 |
| CRM | 负责人 | `owner_id` | 当前用户 ID |
| 多租户 SaaS | 租户 | `tenant_id` | 当前租户 ID |

这样底座只理解“维度、字段、范围值”，不理解具体业务。

## 核心模型

### 1. 权限维度

`sys_data_dimension` 描述某类数据边界，例如租户、项目、课程、业务范围、负责人。

主要字段：

- `code`：维度编码，例如 `tenant`、`project`、`demo_scope`、`owner`。
- `name`：维度名称。
- `value_type`：值类型，目前支持 `string`、`number`。
- `source_type`：取值来源，目前支持 `none`、`table`。
- `source_code`：来源表编码。
- `label_field` / `value_field`：展示字段和取值字段。
- `parent_field`：树形维度的父级字段。
- `memo`：备注。
- `state`：是否启用。

示例：

```json
{
  "code": "demo_scope",
  "name": "业务范围",
  "value_type": "number",
  "source_type": "table",
  "source_code": "demo_scope",
  "label_field": "scope_name",
  "value_field": "id",
  "state": true
}
```

### 2. 菜单绑定

`sys_data_scope_binding` 声明某个已绑定数据表的菜单受哪些数据权限维度控制，以及该维度落在哪个业务字段上。当前实现以 `menu_id` 为保存和解析入口，`table_code` 从菜单绑定表推导或随绑定保存，用于运行时校验和排查。

示例：

- 用户管理菜单绑定 `department`，字段是 `dept_id`。
- 订单管理菜单绑定 `tenant`，字段是 `tenant_id`。
- 示例事项菜单绑定 `demo_scope`，字段是 `scope_id`。

主要字段：

- `menu_id`
- `table_code`
- `dimension_code`
- `field_code`
- `match_type`：匹配方式，目前支持 `in`、`eq`。
- `required`：是否必须配置授权范围。
- `actions`：生效动作 JSON，例如 `query`、`detail`、`create`、`update`、`delete`、`export`、`batch_delete`。
- `state`：是否启用。

示例：

```json
{
  "menu_id": 1201,
  "table_code": "demo_ticket",
  "dimension_code": "demo_scope",
  "field_code": "scope_id",
  "match_type": "in",
  "required": true,
  "actions": ["query", "detail", "create", "update", "delete"]
}
```

如果一张表需要多维度控制，例如事项既要限制业务范围，又要限制项目，可以绑定多条规则。运行时默认取交集：

```text
scope_id IN (...) AND project_id IN (...)
```

如果同一张表发布或绑定到多个菜单，请求应携带明确的 `menu_id`。菜单没有数据权限绑定时不追加数据权限条件；同表多菜单且无法唯一定位时，后端会拒绝解析，避免误用另一份菜单授权。

### 3. 角色授权

`sys_role_data_scope` 保存角色在某个菜单和维度上的数据范围。

当前策略：

- `all`：全部数据。
- `none`：无数据。
- `specified`：指定范围，`scope_values` 保存 JSON 数组。
- `tree`：指定树节点并展开下级，要求维度来源表配置 `value_field` 和 `parent_field`。
- `self`：本人数据，解析为当前用户 ID。
- `user_dimension`：从 `sys_user_dimension_value` 读取当前用户在该维度上的归属值。

指定范围保存为结构化值，例如 JSON 数组，而不是 SQL。

运行时以“角色授权”为主，“用户授权”为补充：

- 角色数据权限：常规权限来源。
- 用户数据权限覆盖：用于在已有角色、菜单和按钮权限基础上临时扩大、收窄、替换或拒绝特殊人员范围；它不能单独授予一个没有角色权限的用户访问页面。
- 多角色合并：同一维度取并集，不同维度按绑定条件形成交集。

例子：

```json
{
  "role_id": 8,
  "menu_id": 1201,
  "dimension_code": "demo_scope",
  "strategy": "specified",
  "scope_values": ["1001", "1002"]
}
```

如果策略是本人数据，不需要手工填值：

```json
{
  "role_id": 9,
  "menu_id": 1301,
  "dimension_code": "owner",
  "strategy": "self",
  "scope_values": []
}
```

### 4. 用户维度归属与覆盖

`sys_user_dimension_value` 保存用户在某个维度上的默认归属值，供 `user_dimension` 策略使用。例如某个用户归属业务范围 `1`、`2`，角色策略选 `user_dimension` 后运行时会读取这组值。

`sys_user_data_scope_override` 保存用户级覆盖，字段包括 `user_id`、`menu_id`、`table_code`、`dimension_code`、`strategy`、`scope_values`、`override_mode`、`expire_at` 和 `state`。

这两类配置都不是角色权限的替代品。用户必须先通过角色获得菜单、按钮和接口权限，数据权限解析才会继续计算用户归属和特殊授权。

覆盖模式：

- `replace`：用用户覆盖替换角色解析结果。
- `union`：用户覆盖与角色结果取并集。
- `intersect`：用户覆盖与角色结果取交集。
- `deny`：直接拒绝该维度访问。

### 5. 运行时解析

请求进入 service 后，根据当前用户、菜单、表编码和接口动作解析数据范围。

解析输出统一结构：

```go
type DataScope struct {
	AllowAll   bool
	DenyAll    bool
	Conditions []DataScopeCondition
}

type DataScopeCondition struct {
	DimensionCode string
	Field         string
	MatchType     string
	ValueType     string
	Values        []string
}
```

repository 层只接收结构化条件并转成 GORM 查询，避免控制器里拼查询条件。

运行时流程：

1. 中间件只负责登录和基础接口权限。
2. 控制器解析当前访问的是哪个菜单、表和动作。
3. service 调用数据权限解析器，得到结构化范围。
4. repository 的查询构建器统一追加数据权限条件。
5. 详情、编辑、删除先按 ID 查询时也追加同一套条件，避免只拦列表不拦详情。

新增和编辑的校验语义略有不同：

- 新增没有历史行可查，后端会校验本次提交的受控字段是否落在当前可访问范围内。
- 编辑会先校验原行是否可访问，再校验本次提交里出现的受控字段；未提交的受控字段按原行值处理。
- 删除、详情和导出按当前菜单、表编码、动作和记录 ID 解析范围后再访问数据。

示意：

```text
HTTP 请求
  -> 登录校验
  -> 菜单/按钮/接口权限
  -> 解析数据权限
  -> repository 追加查询条件
  -> 返回数据
```

不要在 `FileController`、`GeneralizationController`、业务控制器里各写一套数据权限判断。文件预览和下载如果要判断业务归属，应通过“文件引用的业务记录”回到统一数据权限层判断，而不是在文件控制器里按 `table_code` 临时拼条件。

## 数据表

### 数据权限维度表

`sys_data_dimension`

保存“有哪些可授权维度”。

| 字段 | 含义 |
| --- | --- |
| `id` | 主键 |
| `code` | 维度编码 |
| `name` | 维度名称 |
| `value_type` | 值类型 |
| `source_type` | 数据来源 |
| `source_code` | 来源表编码 |
| `label_field` | 展示字段 |
| `value_field` | 值字段 |
| `parent_field` | 树形父级字段 |
| `memo` | 备注 |
| `state` | 是否启用 |

### 权限绑定表

`sys_data_scope_binding`

保存“某个已绑定数据表的菜单按哪个字段受控”。

| 字段 | 含义 |
| --- | --- |
| `id` | 主键 |
| `menu_id` | 菜单 ID |
| `table_code` | 表编码 |
| `dimension_code` | 维度编码 |
| `field_code` | 当前表里的受控字段 |
| `match_type` | 匹配方式 |
| `required` | 是否必须配置范围 |
| `actions` | 生效动作 JSON |
| `state` | 是否启用 |

### 角色数据权限授权表

`sys_role_data_scope`

保存“角色在某个菜单上的数据范围”。

| 字段 | 含义 |
| --- | --- |
| `id` | 主键 |
| `role_id` | 角色 ID |
| `menu_id` | 菜单 ID |
| `table_code` | 表编码 |
| `dimension_code` | 维度编码 |
| `strategy` | 全部、本人、指定范围、无数据等 |
| `scope_values` | JSON 数组，保存指定范围 |
| `state` | 是否启用 |

### 用户数据权限覆盖表

`sys_user_data_scope_override`

用于给某个用户临时扩大、缩小、替换或拒绝范围。

| 字段 | 含义 |
| --- | --- |
| `id` | 主键 |
| `user_id` | 用户 ID |
| `menu_id` | 菜单 ID |
| `table_code` | 表编码 |
| `dimension_code` | 维度编码 |
| `strategy` | 覆盖策略 |
| `scope_values` | JSON 数组 |
| `override_mode` | 覆盖模式：replace、union、intersect、deny |
| `expire_at` | 过期时间，可选 |
| `state` | 是否启用 |

### 用户维度归属表

`sys_user_dimension_value`

用于保存用户在某个维度上的默认归属值。

| 字段 | 含义 |
| --- | --- |
| `id` | 主键 |
| `user_id` | 用户 ID |
| `dimension_code` | 维度编码 |
| `scope_values` | JSON 数组 |
| `state` | 是否启用 |

## 配置示例

### 通用 Demo：事项只看指定业务范围

1. 建维度 `demo_scope`，来源表 `demo_scope`。
2. 事项菜单绑定 `demo_scope` 维度，字段为 `scope_id`。
3. 角色“范围 A 用户”授权 `demo_scope` 范围为 `[1]`。
4. 查询事项时自动追加：

```text
demo_ticket.scope_id IN (1)
```

### 在线学习：老师只看自己课程

1. 建维度 `course`，来源表 `edu_course`。
2. 课程内容菜单绑定 `course` 维度，字段为 `course_id`。
3. 老师角色授权策略为“指定范围”，范围来自老师被分配的课程。
4. 课程内容列表、详情、编辑和文件预览都只允许访问授权课程下的数据。

### 多租户：业务菜单按租户隔离

1. 建维度 `tenant`。
2. 每个需要隔离的菜单绑定 `tenant` 维度，字段为 `tenant_id`。
3. 登录后从用户上下文得到当前租户范围。
4. repository 统一追加 `tenant_id` 条件。

当前实现还没有独立的全局表级绑定开关。如果后续大量菜单都要按同一租户字段隔离，可以在现有模型上增加“批量绑定/模板绑定”能力，但运行时仍应落到菜单绑定和结构化范围解析上。

## 配置入口

当前配置围绕“维度、菜单绑定、角色授权、用户覆盖、权限排查”展开：

1. 数据权限维度：维护维度、来源表、展示字段、值字段、父级字段。
2. 权限绑定：把菜单绑定到表、维度、字段和生效动作。
3. 角色管理：保存菜单权限、按钮权限、接口权限和角色数据权限。
4. 用户维度归属和用户覆盖：保存特殊人员的默认归属或临时覆盖。
5. 权限排查：按用户、菜单、表编码和动作查看最终解析结果。

权限排查展示：

- 当前用户有哪些角色。
- 当前菜单绑定了哪些数据权限维度。
- 每个维度最终解析出的范围值。
- 解析后的 `scope` JSON、绑定/角色/用户归属/特殊授权计数和诊断备注。

权限排查用于解释“为什么当前账号在这个菜单、表和动作下能看到或看不到数据”。它不是 SQL 预览器，不保证展示最终 GORM 生成的完整 SQL。

## 与已有固定字段权限的迁移

如果当前已有固定字段数据权限，可以按下面过渡：

1. 初始化一个对应的数据权限维度。
2. 将已有使用固定字段隔离的菜单绑定到 `tenant_id`、`project_id` 或现有业务字段。
3. 将原来的角色范围迁移成“指定范围”。
4. 完成迁移后移除旧固定字段逻辑，运行态只保留新数据权限模型。

## 当前边界

已经明确的规则：

- 用户级覆盖已经支持，用于临时扩展、收窄、替换或拒绝范围。
- 多角色在同一维度取并集，不同维度通过多条绑定形成交集。
- 低代码列表、详情、新增、编辑、删除、导出、批量删除应走数据权限。
- 已接入的固定系统页记录检查应走数据权限；未接入的固定控制器不能在文档里当作已覆盖。
- 文件预览和下载在有业务记录上下文时应回到统一数据权限层检查。
- 树形范围支持按来源表展开下级；大规模树形维度后续可以再评估缓存和刷新策略。

仍然不要做的事：

- 不要把数据权限写死为公司、部门或 HRDB。
- 不要允许前端提交原始 SQL。
- 不要在每个控制器里各写一套字段判断。
- 不要只限制列表，不限制详情、编辑、删除和导出。
