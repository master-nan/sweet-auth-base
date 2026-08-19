# Sweet Platform Query Center V1 详细设计

> Audience: 产品 Reviewer、架构 Reviewer、代码边界 Reviewer、QC-002 实施人员
>
> Lifecycle: construction
>
> Final Action: DELETE_AFTER_STABLE
>
> Removal Gate: Query Center V1 完成实现、验收，稳定规则已吸收进长期 Engineering/User Guide
>
> Audit Baseline: `291f22e231883dfab66ac32db4b40b9444fd3d66`
>
> Review Status: `DESIGN_APPROVED_FOR_QC-002A`
>
> Implementation Status: `QUERY_CENTER_V1_FROZEN`

## 1. 结论摘要

Query Center V1 保存和复用当前业务列表的查询方法。`AdvancedQuery` 继续回答“怎么查”，Query Center 只回答“如何保存、选择、设默认和管理这套查法”。它不代理业务查询，不保存 SQL，不扩大 Data Permission，也不承担列偏好、报表或跨表查询。

V1 冻结以下结论：

1. 业务查询协议继续使用现有 `Query`、`ExpressionGroup`、`QueryRule`、`Order` 和 `QuickQuery`，不新增 Query V2。
2. `sys_menu.query_scope_code` 是 Query Scope 唯一持久化身份真值；不使用 URL、中文菜单名、Vue 文件路径、Registry key、前端常量或单独 `table_code` 作为方案身份。
3. 方案类型固定为 `PERSONAL`、`PUBLIC`、`ROLE`、`PAGE_DEFAULT`。
4. 默认优先级为个人默认 > 页面默认 > 页面代码初始 Query；ROLE 不参与自动默认。
5. 一张 `query_scheme` 主表和一张 `query_scheme_role` 关联表足够；表达式存受 Schema 约束的 PostgreSQL JSONB，不拆条件 EAV 表。
6. 方案保存 expressions、quick keyword 和服务端排序；不保存 page、pageSize、列显示、列宽、密度和其他 View Preference。
7. Scheme Runtime Read 与 Scheme Management 分离。业务页面只请求当前 scope 可见摘要，并按需解析单个方案。
8. `QuerySchemeService` 只负责方案 CRUD、可见性、验证和解析；真实列表仍由原业务 Service 查询。
9. 普通用户可管理自己的 PERSONAL 方案；共享类型写操作只用一个 `query_scheme_shared_manage` Business Capability。
10. Query Center 使用隐藏路由，由业务页“管理查询方案”进入；V1 不新增左侧一级菜单。
11. 条件预览是 V1 必需；左侧条件树、字段说明面板 V1 延期。
12. 动态日期/身份不是新查询操作符。方案只可携带平台注册白名单内的强类型 value bindings，服务端解析成当前协议的具体值后才交给业务查询；Binding 不是 DSL。
13. V1 UI 最大可编辑嵌套深度为 2，后端 Query Scheme Schema 防御上限为 3；超出 UI 编辑能力的合法结构必须受限展示，不能静默降级或丢失。

## 2. 产品边界

### 2.1 属于 V1

- 业务列表页选择可见方案；
- 高级查询简单模式和高级模式；
- 保存个人方案、保存修改、另存为；
- PERSONAL、PUBLIC、ROLE、PAGE_DEFAULT 管理；
- 个人默认和页面默认；
- 方案可见性、复制、停用、删除和 revision 冲突；
- Metadata 重新校验及 `VALID`、`DEGRADED`、`INVALID` 状态；
- 动态日期和当前主体的受控解析；
- 可读条件预览；
- Query Scheme 与 Data Permission 的 AND 组合。

### 2.2 明确不属于 V1

- SQL、字符串 DSL、任意表达式脚本；
- JOIN、跨表查询、Dataset、Report 集成；
- 收藏、文件夹、标签、分享、订阅；
- 角色默认、多层默认合并；
- 方案审批、发布工作流；
- 方案 JSON 导入导出；
- 历史版本表；
- Redis Scheme Cache；
- 列显示/宽度/顺序、密度和 pageSize 偏好；
- Query Center 代理业务数据 API。

## 3. 现有基线与复用

| 现有能力 | V1 用法 |
| --- | --- |
| `AdvancedQuery.vue` | 保留为高级模式核心；增加模式壳和预览，不重写字段/值控件 |
| `AdvancedQueryRuleRow.vue` | 简单/高级模式共同复用规则行 |
| `sanitizeQueryExpressions` | Scheme 保存前值清洗的唯一前端入口 |
| `useTableQueryState` | 扩展 source、baseline、dirty、applyScheme；不新建第二个查询状态 composable |
| Runtime Metadata | 验证字段存在、启用、可高级查询、类型、字典/关系和排序 |
| `StandardTableToolbar` | 只增加三个 layout slots，不请求 Scheme API |
| `usePageButtons` | 继续处理 Business Capability；View Action 不进入 MenuButton |
| Data Permission Resolver/Adapter | 原样执行；Scheme 只增加业务 WHERE 条件 |
| `SysMenu.Name` / Menu Capability | 继续作为路由和权限关联事实，但不直接充当 Query Scope |

## 4. 总体架构

```text
Business List Page
  |- StandardTableToolbar
  |    |- QuerySchemeSelector
  |    |- QuickPresetBar
  |    |- AdvancedQuery
  |    `- Save Scheme
  |- useTableQueryState
  `- original Business API

Query Scheme Runtime API
  -> QuerySchemeService
       |- SysMenu scope identity / QueryScopeRegistry runtime config
       |- Runtime Metadata Reader
       |- Query Payload Validator
       |- Dynamic Binding Resolver
       `- QuerySchemeRepository

Resolved standard Query
  -> original Business Service
       -> Business query conditions
       +  Data Permission result
       -> Repository / SQL
```

Query Center 不位于业务 Service 与 Repository 之间，也不保存 `DataScopeResult`、Grant、角色解析结果或 SQL。

## 5. Query Scope

### 5.1 身份规则

新增 `sys_menu.query_scope_code varchar(128) null`。它是 Query Scope 的**唯一持久化身份真值**，其值是稳定、非展示性的页面查询上下文，例如：

- `system.user.list`
- `system.audit.list`
- `organization.employee.list`
- `integration.execution.list`
- `develop.dictionary.master`

空值表示该菜单资源不接 Query Center。活动菜单上的非空 scope 必须唯一。`query_scheme.scope_code` 保存该稳定值；运行时通过 scope 找到活动菜单，继承该页面的访问权限。

选择新增显式字段，而不是直接使用现有事实的原因：

- `menu_id` 在环境、Seed 和重建过程中不稳定；
- URL 和 Vue 路径可能重构；
- 中文标题是展示文本；
- `table_code` 无法区分同表多页面，也无法表达 Dictionary 主从两个查询上下文；
- `SysMenu.Name` 已承担 Router/Page Button 映射，继续叠加公开持久化契约会放大重命名风险。

`query_scope_code` 不形成第二套权限体系。它只定位现有 Menu Resource；授权仍由 Menu/Casbin 和方案可见性共同决定。后端 Registry 与前端页面配置都不得再次声明或映射另一份 scope identity。

### 5.2 Page Query Config

每个已由 `sys_menu.query_scope_code` 声明的 scope 有一份强类型运行配置，不新增配置表。固定页面由后端 `QueryScopeRegistry` 为这个**既有 scope** 注册配置，前端通过 Runtime Scope Config API 读取；低代码页面在后续批次由受控 Metadata 生成。

```text
QueryScopeConfig
  scope_code
  menu_name
  table_code (nullable for reviewed non-Metadata pages)
  quick_date_field (nullable)
  quick_presets[]
  allowed_virtual_sort_fields[]
  allowed_dynamic_bindings[]
```

`quick_date_field` 必须显式配置，不能从 `created_at` 等字段名猜测。V1 不给 SysTableField 增加 `is_primary_time_filter`；当前需求是页面级语义，放入 Page Query Config 更准确，也避免扩张 Metadata 数据模型。

`QueryScopeRegistry` 只提供 `table_code`、`quick_date_field`、`quick_presets`、Virtual Sort allowlist 和 Dynamic Binding allowlist 等运行事实。它不创建 scope、不覆盖 `sys_menu.query_scope_code`，也不把 Registry key 持久化为方案身份。前端不得维护第二份 `scope_code` 常量映射表作为业务真值；页面从 Runtime Scope Config 获取当前菜单对应的 Query Center 配置。

### 5.3 Scope 权限

所有 Runtime/Management 操作先解析 scope，再确认当前用户拥有目标菜单的 query/page access。拥有某个 Scheme 的 ID 不等于拥有其 scope。跨 scope detail、resolve、copy 一律拒绝。

## 6. 查询方案模型

### 6.1 主表 `query_scheme`

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| `id` | bigint PK | 平台 ID |
| `name` | varchar(64), not null | 同一可见命名空间内唯一，trim 后保存 |
| `scope_code` | varchar(128), not null, index | Query Scope |
| `scheme_type` | varchar(24), CHECK | PERSONAL/PUBLIC/ROLE/PAGE_DEFAULT |
| `owner_user_id` | bigint nullable FK sys_user | 仅 PERSONAL 必填，来自认证 Subject |
| `query_schema_version` | smallint not null default 1 | Payload Schema 版本，不是历史版本 |
| `query_payload` | jsonb not null | 标准 Query 子集与可选 bindings |
| `is_default` | boolean not null default false | 仅 PERSONAL/PAGE_DEFAULT 合法 |
| `enabled` | boolean not null default true | 共享方案可临时停用；PERSONAL 保持 true |
| `revision` | bigint not null default 1 | 乐观锁，每次修改 +1 |
| `gmt_create/gmt_modify/...` | Basic 审计字段 | 复用当前 Model 基线 |

不增加 `scheme_code`。V1 没有外部 Scheme API、导入导出或跨环境引用；ID 足够。未来若出现正式 Seed/外部契约，再通过独立兼容设计增加稳定 code，不能预埋无用途字段。

### 6.2 角色关联 `query_scheme_role`

| 字段 | 约束 |
| --- | --- |
| `scheme_id` | FK query_scheme, composite PK, cascade delete |
| `role_id` | FK sys_role, composite PK, cascade delete |

ROLE 至少关联一个活动角色，最多 32 个。角色删除后关联自然删除；角色停用时不可见。角色 ID 不以逗号字符串或 JSON 保存。

### 6.3 PostgreSQL 约束

- PERSONAL 必须有 owner；其他类型 owner 必须为空；
- PUBLIC/ROLE 不允许 `is_default=true`；
- PERSONAL 同 `owner_user_id + scope_code + lower(name)` 活动记录唯一；
- 共享类型同 `scope_code + scheme_type + lower(name)` 活动记录唯一；
- PERSONAL 同 owner+scope 最多一个 enabled default（partial unique）；
- PAGE_DEFAULT 同 scope 最多一个 enabled default（partial unique）；
- `octet_length(query_payload::text) <= 32768`；
- `revision >= 1`；
- `query_schema_version = 1`；
- 所有唯一约束排除软删除记录。

默认替换在短事务中先清原 default 再设新 default；partial unique 是最终防线。PUBLIC 没有默认语义，避免与 PAGE_DEFAULT 重复。ROLE V1 不支持默认。

## 7. Query Payload Schema V1

### 7.1 持久化内容

```json
{
  "expressions": [
    {
      "logic": 1,
      "rules": [
        {"field": "status", "expression_type": 5, "value": "failed", "type": 3}
      ],
      "nested": []
    }
  ],
  "quick_query": {"keyword": ""},
  "order": {"field": "gmt_create", "is_asc": false},
  "bindings": []
}
```

`expressions`、`quick_query`、`order` 的形状和枚举与当前 Query 协议一致。Payload 不保存 `page`、`num`、`table_code`、`menu_id`、`filters`、`include_deleted`。页面应用方案时将这三个语义字段合入自己的 Query，强制 `page=1`，保留页面自己的 pageSize 和固定安全 filters。

`filters` 不得进入 Scheme，因为当前它承载关联/页面约束，不是普通高级查询白名单；保存会造成跨上下文和越权风险。

### 7.2 Schema 与资源限制

| 限制 | V1 值 |
| --- | ---: |
| 名称 | 64 Unicode 字符 |
| Scope Code | 128 ASCII 字符，`^[a-z][a-z0-9_.-]*$` |
| JSONB 文本大小 | 32 KiB |
| 顶层 Expression Group | 8 |
| 总有效 Rule | 50 |
| 后端 Schema 最大嵌套深度 | 3（根 Group 深度为 1） |
| 单个多值数组 | 100 |
| Quick keyword | 256 字符 |
| ROLE 角色数 | 32 |
| 单用户单 scope 活动 PERSONAL | 100 |
| 单 scope 活动共享方案 | 200 |

限制在后端 Schema Validator 执行，数据库只承担 JSON 大小和基础 CHECK。超限请求返回稳定 Application Error，不截断、不静默降级。这里的深度 3 是防御性存储/读取上限，不代表 V1 UI 必须提供第三层编辑能力。

### 7.3 标准化与 Hash

新增 `normalizeQuerySchemePayload`，内部必须复用现有 `sanitizeQueryExpressions` 的规则和值语义。标准化固定：

1. 删除空规则/空组；
2. 按 Metadata 校正值类型；
3. 空 nested 统一为 `[]`；
4. 缺省 group logic 统一为 AND；
5. Quick keyword trim；
6. 空排序统一为 `{field:"",is_asc:false}`；
7. 保留用户条件顺序，不为了 Hash 重排 AND 条件；
8. 对标准对象键稳定序列化并计算 SHA-256，仅前端 dirty/测试使用，不新增数据库 hash 字段。

现有 `query-state.ts` 仍是前端唯一 Normalize 入口；QC-002 扩展它，而不是新建同义工具。

## 8. 动态值与快捷条件

### 8.1 两类 Quick Condition

**通用时间快捷条件**由 Page Query Config 的 `quick_date_field` 开启：今天、昨天、本周、本月、上月。点击后生成作用于现有 Query State 的条件。

**业务快捷条件**由 scope 注册的 `quick_presets` 明确定义，例如“我的”“异常”“待审核”。不根据字段名或状态文案猜测。

### 8.2 Value Binding

当前 ExpressionType 没有相对日期或当前主体操作符，业务 Query 协议不能修改。V1 在 Scheme Payload 中允许可选 `bindings`，它们只是由平台解释的强类型值解析指令：

```json
{
  "pointer": "/expressions/0/rules/1/value/0",
  "kind": "START_OF_MONTH",
  "params": {"month_offset": 0}
}
```

V1 Binding kind 白名单固定为：

- `TODAY`
- `START_OF_WEEK`
- `END_OF_WEEK`
- `START_OF_MONTH`
- `END_OF_MONTH`
- `CURRENT_USER`
- `CURRENT_EMPLOYEE`

日期偏移只允许强类型有限参数：`TODAY.day_offset` 为 `[-366, 366]` 的整数，周边界的 `week_offset` 为 `[-52, 52]` 的整数，月边界的 `month_offset` 为 `[-120, 120]` 的整数；缺省均为 0。主体 Binding 不接受参数。一个动态范围使用两个受控 Binding 分别指向 range value 的起止项，例如“本月”为 `START_OF_MONTH(month_offset=0)` 与 `END_OF_MONTH(month_offset=0)`，“上月”为 offset=-1。未知 kind、未知参数、越界数值或不匹配的 pointer/type 一律拒绝。

Payload 中对应 rule 同时保存最近一次解析得到的合法具体值，因此 expressions 本身始终是现有标准 Query。`POST .../:id/resolve` 在服务端根据应用时刻、平台时区和认证 Subject 覆盖该 value，然后只返回普通 Query 子集。业务 API 永远看不到 binding。

安全规则：

- binding kind、目标 JSON Pointer、字段和操作符必须在 scope config 白名单内；
- CURRENT_USER/CURRENT_EMPLOYEE 由服务端 Subject/Employee Binding 解析，客户端不能提交 ID；
- “我的”优先由业务 Preset 注册，不自动假定 `owner_id=current_user_id`；
- 身份缺失时 resolve 失败，不以空条件放宽结果；
- 日期边界使用平台业务时区，输出当前 DATE/DATETIME 类型可接受的具体值；
- 编辑 bound expression 的字段、操作符、值或结构时，前端清除受影响 binding 并提示“动态条件已转为当前固定值”；仅修改 quick keyword/sort 时可保留 bindings。

禁止任意变量名、自由表达式、JavaScript、SQL、模板语言、函数名字符串、动态函数调用、用户自定义 Binding 和反射式 Binding Resolver。Binding 不是 DSL，也不是通用计算引擎。

该 envelope 解决“本月方案跨月仍动态”与“不修改业务 Query 协议”的冲突。Reviewer 已批准该受控白名单机制。

### 8.3 Preset 与 Scheme

- 临时点击时间快捷条件：应用 binding + 当前解析值，Scheme Source 标记为 PRESET；
- 业务 Preset：来自受控 scope config，可包含静态条件和 bindings；
- 用户继续修改并保存：创建普通 PERSONAL Scheme；未修改的 bindings 一并保存；
- Scheme 复制：复制 Payload 和 bindings，之后与源方案无父子关系；
- 普通手工日期条件保存为固定日期，不自动推断为动态日期。

## 9. Metadata 验证与状态

### 9.1 每次保存/解析检查

- scope 存在、启用且当前用户可访问；
- field 存在、启用、非敏感、`is_advanced_search=true`；
- operator 对当前字段类型仍合法；
- value 与字段类型兼容；
- dict/linkage/organization selector 配置仍可解析；
- sort field 为 `is_sort=true` 或 scope 受控 Virtual Sort；
- binding 的字段/操作符/kind 在 scope 白名单；
- Payload 结构、大小、深度和数量限制成立。

### 9.2 状态

| 状态 | 定义 | Runtime 行为 |
| --- | --- | --- |
| `VALID` | 全部事实有效 | 可直接解析、应用 |
| `DEGRADED` | 至少一个字段/字典/关系/sort/binding 已失效，剩余结构仍可展示 | 不自动执行；返回 issues，进入编辑修复 |
| `INVALID` | JSON/Schema/Scope/结构/大小损坏，或无法安全解析 | 不返回可执行 Query；只允许有权限的管理者查看安全错误摘要/删除 |

不得静默删除失效条件，也不得仅应用“剩余有效条件”，因为那会改变过滤语义并可能扩大结果。默认方案 DEGRADED/INVALID 时跳过自动执行，回退到下一优先级并显示一次非阻断提示；服务端记录 reason code，不返回底层错误。

## 10. 默认、Dirty 与应用流程

### 10.1 默认优先级

```text
PERSONAL default for current user + scope
  > enabled PAGE_DEFAULT for scope
  > page code initial Query
```

ROLE/PUBLIC 只能手工选择。默认只定义初始查询，不覆盖 pageSize、列偏好、业务固定 filters 或 Data Permission。

### 10.2 应用方案

1. Selector 读取 Runtime Available 摘要；
2. 用户选择 Scheme；
3. 调用 resolve endpoint，后端确认 scope、可见性、revision；
4. 后端用当前 Runtime Metadata 验证，并解析 bindings；
5. 前端将 expressions、quick keyword、order 写入 `useTableQueryState`；
6. `page=1`，保留 pageSize 和页面固定 filters；
7. 调用原业务列表 API；
8. 保存 normalized baseline、source scheme id/revision/name/type/bindings。

### 10.3 Dirty

Quick keyword、Advanced expressions 或 order 任一标准化后与 resolved baseline 不同，显示“方案名称（已修改）”。分页、pageSize、刷新和列显示不参与 Dirty。

切换方案、清空查询或离开页面时，如果 dirty，使用统一确认提示：放弃修改/留在当前页。V1 不自动保存。刷新只刷新当前查询，不改变 dirty。

### 10.4 删除当前方案

删除后当前页保留已经应用的具体 Query，但 source 变为 `TEMPORARY_DELETED`，显示“临时条件（原方案已删除）”。下次进入页面按默认优先级重新加载。删除不自动清空当前结果。

## 11. 方案类型规则

### 11.1 PERSONAL

- owner 由认证 Subject 写入，客户端无 owner 字段；
- 仅本人可见、编辑、重命名、删除、设默认；
- 管理员没有日常修改他人 PERSONAL 的入口；
- 同 scope 最多 100 个活动方案；
- 保存 Dialog 只创建/修改 PERSONAL。

### 11.2 PUBLIC

- 对拥有目标 scope 页面访问权的用户可见；
- 普通用户可使用、查看安全详情、复制为 PERSONAL；
- 只有 `query_scheme_shared_manage` 可创建、编辑、停用和删除；
- PUBLIC 没有 default。

### 11.3 ROLE

- 由共享方案管理员创建，并指定 1..32 个角色；
- 用户至少有一个匹配的活动角色且拥有页面访问权时可见；
- 普通用户不能把 PERSONAL 分享给角色；
- 无角色默认；多角色匹配只去重展示，不自动选择。

### 11.4 PAGE_DEFAULT

- 每个 scope 最多一个 enabled default；
- 由共享方案管理员维护；
- 所有拥有页面访问权的用户可使用；
- 个人默认优先于它；
- 修改/替换不能影响 Data Permission。

### 11.5 复制

PUBLIC、ROLE、PAGE_DEFAULT 可复制为当前用户 PERSONAL。复制产生全新记录、revision=1、默认=false；Payload/bindings 做深复制，不保留源 ID 或依赖。PERSONAL 也可“另存为”复制。

## 12. 权限模型

V1 只新增一个共享管理 Business Capability：

| 能力 | 用途 |
| --- | --- |
| `query_scheme_shared_manage` | 创建/编辑/停用/删除 PUBLIC、ROLE、PAGE_DEFAULT，维护角色范围和页面默认 |

以下行为不新增 MenuButton：

- Runtime available/resolve：由目标 scope 页面访问权 + 方案可见性授权；
- PERSONAL CRUD/default：由目标 scope 页面访问权 + owner 授权；
- Selector、打开 AdvancedQuery、管理自己的方案：Platform Query Experience，不是独立共享管理能力。

后端仍为唯一真值。前端 Tab 和按钮隐藏只是体验。共享管理 endpoint 同时校验 Casbin capability、scope 访问和方案类型；ROLE 角色范围还要验证角色存在/活动。

## 13. Data Permission 边界

```text
Resolved Query Scheme conditions
                |
                v
Original Business Query ---- AND ---- DataScopeResult
                |                         |
                +------------+------------+
                             v
                    Repository / SQL WHERE
```

Scheme 不保存 `DataScopeResult`、Policy、Rule、Grant、角色 ID 推导结果或 ownership subject。公共方案中的 `org_unit_id=A` 只是额外过滤。Scheme 解析失败必须 fail closed；任何 Scheme 都不能减少、替换或短路 Data Permission 条件。

## 14. API 设计

### 14.1 Runtime API

| Method/Path | 用途 | 返回 |
| --- | --- | --- |
| `GET /admin/runtime/query-scopes/:scope_code` | 读取安全 Scope Config | label、quick date/presets、允许能力摘要 |
| `GET /admin/runtime/query-schemes/available?scope_code=...` | Selector 摘要 | 当前用户 PERSONAL、PUBLIC、匹配 ROLE、PAGE_DEFAULT，含 resolved default id |
| `POST /admin/runtime/query-schemes/:id/resolve` | 按需验证并解析方案 | resolved query subset、status/issues、source revision/bindings summary |

Available endpoint 不返回完整 query_payload，不分页但每类有服务端上限；总返回最多 200 条并按“个人默认、最近使用/更新时间、名称”稳定排序。若超过上限，Selector 提示去管理页搜索，不加载剩余 Payload。

Resolve body 只含 `scope_code` 和可选 `expected_revision`。不能用 ID 读取其他 scope 或其他用户 PERSONAL。

QC-002B 实施确认：Resolve 除解析后的标准 Query 与 `binding_kinds` 摘要外，还返回当前方案经过服务端校验的受控 `bindings`（pointer、白名单 kind、有限 offset 参数）。这是前端无损 Dirty、保存修改和另存为动态方案所需的最小 Runtime DTO；Available 仍不返回 Payload/Binding 明细，Scheme 协议、数据库结构与 Binding 白名单均不变。

固定页面若已有受后端支持、但因系统字段保护而不进入 Runtime Metadata `is_sort` 的默认排序，必须由 Scope Registry 的 `allowed_virtual_sort_fields` 显式声明。QC-002B 参考页验收确认 Organization 列表的既有 `gmt_modify` 排序属于该类；仅相应固定 Organization scope 获得该 allowlist，不允许前端省略排序校验或提交任意字段。

### 14.2 Management API

| Method/Path | 用途 |
| --- | --- |
| `POST /admin/query-schemes/query` | 分页查询当前用户可管理/可见方案摘要 |
| `GET /admin/query-schemes/:id` | 管理详情，按 owner/visibility/shared-manage 授权 |
| `POST /admin/query-schemes/personal` | 创建 PERSONAL |
| `PUT /admin/query-schemes/personal/:id` | revision 更新本人 PERSONAL |
| `DELETE /admin/query-schemes/personal/:id` | 删除本人 PERSONAL |
| `PUT /admin/query-schemes/personal/:id/default` | 设置/取消本人默认 |
| `POST /admin/query-schemes/:id/copy-to-personal` | 可见方案复制为 PERSONAL |
| `POST /admin/query-schemes/shared` | 创建 PUBLIC/ROLE/PAGE_DEFAULT |
| `PUT /admin/query-schemes/shared/:id` | revision 更新共享方案 |
| `DELETE /admin/query-schemes/shared/:id` | 删除共享方案 |
| `PUT /admin/query-schemes/shared/:id/enabled` | 启停共享方案 |

Management query 支持 name、scope、type、enabled、page/num；最大 pageSize 100。普通用户只看到自己的 PERSONAL 和自己可使用的共享方案；共享管理员可按 scope 管理共享类型，但仍不能修改他人 PERSONAL。

### 14.3 DTO 白名单

**AvailableSummary**：`id,name,type,is_default,status`。不返回 owner ID、角色 ID、Payload 或普通用户无需感知的 revision 技术字段；revision 仅由 Resolve/Management 编辑链路携带。

**ListItem**：上述字段 + `scope_code,scope_label,enabled,creator_display_name,role_summary`。PERSONAL 的创建者只显示“本人”。

**Detail**：ListItem + `query_payload` 安全投影、preview、issues、role IDs（仅共享管理员编辑 ROLE 时）。

**ResolveResponse**：`scheme{id,name,type,revision,is_default}`、`validation_status`、`issues[]`、`resolved_query{expressions,quick_query,order}`、`binding_summary[]`。

列表永远不返回完整 Query JSON。底层 JSON/数据库错误不进入响应。

`revision` 是 API 乐观锁字段，不是业务展示字段。前端可以在 Scheme state 中保存并随更新请求回传，但普通用户列表、Selector、Detail/Drawer 和表单均不渲染 revision 数值。

## 15. Audit 与 Reason Code

PUBLIC、ROLE、PAGE_DEFAULT 的 create/update/enable/disable/delete/default/role-scope 变更写业务 Audit。PERSONAL 只记录 action、scheme id、type、scope 和结果，不把完整 query value 写入自由文本。

建议模块内 reason code：

- `query_scheme_scope_unavailable`
- `query_scheme_metadata_degraded`
- `query_scheme_payload_invalid`
- `query_scheme_binding_unresolvable`
- `query_scheme_revision_conflict`

它们属于 Query Center 诊断，不自动等于 HTTP Error Code。稳定 Application Error 放入 `internal/errors/query_scheme.go`。

## 16. 并发与事务

- Create/Update/Default/Enable/Delete 在 Service 定义短事务；
- Update 使用 `WHERE id=? AND revision=?`，0 行返回稳定 `query_scheme_revision_conflict`；HTTP 响应不得暴露数据库条件或 revision 数值，前端统一提示“方案已被其他操作更新，请刷新后重试。”；
- Default 替换在同事务中清旧值、设新值，由 partial unique 兜底；
- ROLE 更新在同事务替换关联行；
- Metadata 读取、动态主体解析和 Payload 预校验尽量在事务外；事务内再次检查 record revision 和约束；
- 不做历史版本表，不做分布式锁；
- Cache V1 不启用，数据库是唯一真值。

## 17. 前端交互设计

### 17.1 业务列表 Toolbar

```text
[查询方案 v] [快捷条件...] [高级查询] [保存方案] ...Business Actions... [列设置] [刷新]
```

- 当前方案显示名称；dirty 时追加“（已修改）”；
- Selector 分组：我的方案、公共方案、角色方案、页面默认；底部“管理查询方案”；
- 页面默认由 badge 标识，V1 不使用收藏星标；
- 保存方案按钮不是 MenuButton；它只保存 PERSONAL；
- 管理共享方案的入口和按钮受 `query_scheme_shared_manage` 控制。

### 17.2 Simple Mode

Simple Mode 是默认 Tab：只显示字段、操作符、值；所有规则位于一个 AND Group，不显示 nested/logic。添加、删除规则后输出当前 `ExpressionGroup` 结构。切换 Advanced Mode 无损；Advanced 查询只有一个顶层 AND Group 且无 nested 时可切回 Simple，否则提示“当前包含 OR/分组条件，仅可在高级模式编辑”，不降级或丢条件。

### 17.3 Advanced Mode

继续使用现有 Group 卡片和 `AdvancedQueryRuleRow`，增加：

- Simple/Advanced tabs；
- Group 折叠；
- 底部可读条件预览；
- source scheme name/dirty 提示；
- 失效条件的显式错误样式。

V1 UI 最大可编辑嵌套深度正式冻结为 2（根 Group 深度为 1），不得创建或修改第三层。后端 Query Scheme Schema 防御上限仍为 3，以便安全识别未来版本或受控导入的合法结构；V1 不鼓励产生第三层结构，也不为此重写 `AdvancedQuery`。

如果未来加载到深度 3、仍处于后端合法上限内的结构，前端必须完整保留原 Payload，并明确显示“当前版本仅支持查看该层级，无法编辑”的受限状态，阻止会造成覆盖或丢失的保存操作；可引导用户等待后续增强能力。不得静默扁平化、截断、删除条件或伪装为可编辑。深度超过 3 则按 `INVALID` 拒绝。

### 17.4 条件树、预览和字段说明

- 左侧条件树：`V1 DEFERRED`。现有 Group 卡片已经表达结构，树会重复导航且显著增加编辑同步复杂度。
- 条件预览：`V1 REQUIRED`。使用 Field label、Operator label、Dict/Relation/Organization display label，禁止展示内部 field_code。
- 字段说明：`V1 DEFERRED`。Runtime Metadata 没有可靠 description/help_text；不虚构说明，也不在本任务扩 Metadata。

### 17.5 保存与管理

业务页 Save Dialog 只含名称、设为我的默认、取消/保存。已加载本人 PERSONAL 且有 edit 权限时显示“保存修改”；“另存为”始终创建新 PERSONAL。

Query Manager 使用隐藏 Route，四个 Tab：我的方案、公共方案、角色方案、页面默认。共享管理者看到新增/编辑/启停/删除；普通用户对可见共享方案只看到使用、详情、复制。

方案详情采用右侧 Drawer：它是管理页上下文详情，不需要独立 Route；字段包括方案名、scope、类型、状态、默认、创建/更新时间、条件预览、排序、ROLE 范围和 issues。`revision` 仅保存在前端并发控制状态，不展示给普通用户。复杂编辑使用 Dialog，避免 Manager Table 与 Detail Route 重复。

### 17.6 Empty/Error

区分：本 scope 尚无方案、筛选无结果、方案 DEGRADED、请求失败、无共享管理权限。无方案时仍提供“保存当前查询为我的方案”；没有共享管理权限不显示空白管理按钮。

## 18. 前端组件边界

| 类型 | 名称 | 职责 |
| --- | --- | --- |
| EXTEND | `StandardTableToolbar.vue` | 新增 `scheme-selector`、`quick-presets`、`save-scheme` slots；无 API |
| EXTEND | `AdvancedQuery.vue` | Simple/Advanced 壳、折叠、预览 slot/status；保留现有规则能力 |
| EXTEND | `useTableQueryState` | source、baseline、dirty、applyScheme、discard guard；不调用 API |
| EXTEND | `query-state.ts` | Scheme payload normalize/stable compare，复用 sanitize |
| NEW | `QuerySchemeSelector.vue` | 分组摘要选择、当前方案/dirty、管理入口 |
| NEW | `QueryQuickPresets.vue` | 渲染 scope config presets，emit query patch |
| NEW | `QuerySchemeSaveDialog.vue` | PERSONAL 保存/另存为 |
| NEW | `QuerySchemePreview.vue` | 条件和排序的人类可读摘要 |
| NEW | `useQuerySchemes` | Runtime 摘要、resolve、save/copy/default 的页面级编排 |
| NEW | Query Manager Page | 四 Tab 管理页，复用 Toolbar/TablePagination/StatusChip |
| NEW | `QuerySchemeDetailDrawer.vue` | 管理详情与 issues |

不新增超级 Query Page、Global Action Registry 或第二个 AdvancedQuery 协议。

## 19. 后端模块边界

| 层 | 新增/扩展 |
| --- | --- |
| Model | `QueryScheme`、`QuerySchemeRole`；`SysMenu.QueryScopeCode` |
| Migration | PostgreSQL 表、FK、CHECK、partial unique、scope seed |
| DTO | personal/shared create/update、management query、Runtime summary/resolve 白名单 |
| Repository | Scheme CRUD、visible summary、revision update、default replace、role join |
| Service | `QuerySchemeService`：scope/owner/type/capability/transaction；`QuerySchemeRuntimeService`：available/validate/resolve |
| Internal | `internal/queryscheme`：payload schema、normalize、validator、binding resolver、preview facts、scope registry |
| Controller | Bind/Context/Service/Response only |
| Router | Runtime Read 与 Management 路由分开 |
| Errors | `internal/errors/query_scheme.go` 稳定 Application Errors |
| Wire | Repository、Service、Controller 注入；业务 Service 不依赖 Query Center |

`QuerySchemeRuntimeService` 依赖 Metadata Runtime Reader、Menu/Role/User Repository 的安全只读能力，不直接调用业务列表 Service。

## 20. Eligible Page Matrix

| 域 | 页面/Scope | V1 | 说明 |
| --- | --- | --- | --- |
| System | Application / `system.application.list` | ENABLE | 标准实体列表 |
| System | User / `system.user.list` | ENABLE | Metadata + AdvancedQuery |
| System | Role / `system.role.list` | ENABLE | 方案不保存角色授权事实 |
| System | SMS / `system.sms.list` | ENABLE | 标准实体列表 |
| System | Audit / `system.audit.list` | ENABLE | 使用受控字段注册，不伪造 Metadata |
| System | Menu | NO | 树与按钮工作台 |
| System | Data Permission | NO | 多资源复杂配置工作台 |
| Organization | Employee / `organization.employee.list` | ENABLE | “我的”必须业务 Preset |
| Organization | Position / `organization.position.list` | ENABLE | 标准列表 |
| Organization | Sync Batch / `organization.sync_batch.list` | ENABLE | 诊断列表，字段注册受控 |
| Organization | Sync Error / `organization.sync_error.list` | ENABLE | 诊断列表，字段注册受控 |
| Organization | Structure | NO | 树 + 详情 Pattern |
| Integration | External System / `integration.external_system.list` | ENABLE | 标准列表 |
| Integration | Interface Definition / `integration.interface_definition.list` | ENABLE | 标准列表 |
| Integration | Credential / `integration.credential.list` | ENABLE | 禁止秘密字段进入 Payload |
| Integration | Retry Policy / `integration.retry_policy.list` | ENABLE | 标准列表 |
| Integration | Sync Task / `integration.sync_task.list` | ENABLE | 标准列表 |
| Integration | Sync Batch / `integration.sync_batch.list` | ENABLE | Runtime 列表 |
| Integration | Execution / `integration.execution.list` | ENABLE | Attempt 权限不随 Scheme 扩大 |
| Integration | Integration Log / `integration.log.list` | ENABLE | 独立权限 |
| Develop | Dictionary Master / `develop.dictionary.master` | ENABLE | Item 子表不保存父字典上下文 |
| Develop | Dictionary Item | NO | 依赖当前 master selection |
| Develop | Generalization | CONDITIONAL | 仅已发布、稳定 menu scope + Runtime Metadata 页面；QC-002C 接入 |
| Develop | Database | NO | Metadata 配置工作台 |
| Develop | Configure | NO | 表单页 |
| Detail | RecordDetail/Execution Detail | NO | 详情不是列表查询 scope |
| Special | Login/Change Password/Dashboard/404 | NO | 非业务列表 |
| Report | 全部 V1/V2 页面 | REPORT_DEFERRED | Query Center 不接 Report |

首批固定 Scope 共 18 个：System 5 个、Organization 4 个、Integration 8 个、Dictionary Master 1 个；Generalization 条件接入不计入首批门禁。

## 21. 路由与导航

新增隐藏路由 `/admin/query-schemes`，Route Name `query_scheme_manager`。它不显示为左侧导航，从 Selector 底部“管理查询方案”进入。路由 Guard 需要认证；进入后 API 仍按 scope/owner/capability 过滤。

V1 不新增一级“查询中心”菜单。共享管理员使用同一隐藏页面；若生产使用证明需要集中导航，可在不改页面/数据模型的情况下把一个管理入口菜单设为可见，这属于后续 Enablement 决策。

## 22. 逐文件实施清单

### 22.1 Frontend

| 文件 | 动作 | QC-002 内容 |
| --- | --- | --- |
| `src/types/global.ts` | EXTEND | 不改 Query；只收紧 Query value 类型时需兼容评审 |
| `src/utils/query-state.ts` | EXTEND | Payload normalize、stable serialize、preview tokens、binding invalidation helper |
| `src/composables/table-query-state.ts` | EXTEND | source/baseline/dirty/applyResolvedScheme/discard |
| `src/components/Query/AdvancedQuery.vue` | EXTEND | Simple/Advanced、折叠、preview；不重写 RuleRow |
| `src/components/Query/AdvancedQueryRuleRow.vue` | KEEP/EXTEND | 仅补失效状态和动态值显示入口 |
| `src/components/Table/StandardTableToolbar.vue` | EXTEND | 三个 slots，无 Scheme API |
| `src/api/services/query-scheme.ts` | NEW | Runtime 与 Management API 类型/封装 |
| `src/modules/query-scheme/types.ts` | NEW | Scheme/Scope/Payload/Binding/Status View Types |
| `src/composables/query-schemes.ts` | NEW | 页面级 Runtime 编排 |
| `src/components/QueryScheme/QuerySchemeSelector.vue` | NEW | 分组选择器 |
| `src/components/QueryScheme/QueryQuickPresets.vue` | NEW | 快捷条件 |
| `src/components/QueryScheme/QuerySchemeSaveDialog.vue` | NEW | PERSONAL Save/Save As |
| `src/components/QueryScheme/QuerySchemePreview.vue` | NEW | 条件摘要 |
| `src/pages/query-scheme/Index.vue` | NEW | Manager |
| `src/pages/query-scheme/QuerySchemeDetailDrawer.vue` | NEW | Detail |
| `src/router/routes.ts` | EXTEND | hidden Route |
| 18 个 Eligible 页面 | INTEGRATE | scope config + Toolbar slots + useQuerySchemes，不复制逻辑 |

### 22.2 Backend

| 文件/模块 | 动作 | QC-002 内容 |
| --- | --- | --- |
| `model/query_scheme.go` | NEW | 两个 Model |
| `model/sys.go` | EXTEND | QueryScopeCode |
| `migrate/query_scheme_schema.go` | NEW | PostgreSQL schema/constraints/indexes |
| `migrate/query_scheme_seed.go` | NEW | Scope code 和最小 capability 幂等 seed |
| `internal/queryscheme/*` | NEW | Schema、validator、normalize、binding、scope registry |
| `internal/errors/query_scheme.go` | NEW | 稳定 Application Errors |
| `dto/request/query_scheme_req.go` | NEW | 白名单请求 |
| `dto/response/query_scheme_res.go` | NEW | Summary/List/Detail/Resolve DTO |
| `repository/query_scheme.go` | NEW | Context Repository contract |
| `repository/impl/query_scheme_impl.go` | NEW | PostgreSQL query/locking |
| `service/query_scheme_service.go` | NEW | CRUD/事务/共享权限 |
| `service/query_scheme_runtime_service.go` | NEW | available/resolve/Metadata 验证 |
| `controller/query_scheme_controller.go` | NEW | HTTP Adapter |
| `initialize/router.go` | EXTEND | Runtime/Management routes |
| `wire.go` / `wire_gen.go` | EXTEND/GENERATE | 注入 |

不修改现有业务 Query DTO、Repository Query Builder、Data Permission Core、Report 或 Metadata 数据模型。

## 23. 测试矩阵

### 23.1 后端/安全

1. PERSONAL owner 隔离、客户端 owner 注入拒绝；
2. PUBLIC 只对拥有 scope 页面访问权的用户可见；
3. ROLE 仅匹配活动角色可见；
4. PAGE_DEFAULT 每 scope 唯一；
5. PERSONAL default > PAGE_DEFAULT > page initial；
6. ROLE 不自动默认；
7. Scheme 条件与 Data Permission 做 AND，不能替换/扩大；
8. 跨 scope IDOR 的 summary/detail/resolve/copy/update/delete 全拒绝；
9. 非 shared manager 无法写 PUBLIC/ROLE/PAGE_DEFAULT；
10. role IDs 必须存在且不能越权提交；
11. field/operator/type/dict/relation/sort Metadata 验证；
12. 受保护/不存在字段拒绝；
13. DEGRADED 不自动执行、不静默删条件；
14. INVALID 不返回可执行 Query；
15. JSON Schema、32 KiB、50 rules、depth 3、multi-value 100 限制；
16. 动态日期按平台时区解析；
17. CURRENT_USER/EMPLOYEE 只从 Subject/Binding 解析，缺失 fail closed；
18. revision 并发更新只有一个成功；
19. 默认替换并发保持唯一；
20. copy 成为独立 PERSONAL，源更新不影响副本；
21. 删除/停用共享方案立即从 available 消失；
22. Available 不返回 Query JSON/他人 PERSONAL；
23. Audit 不记录完整 Query value；
24. PostgreSQL partial unique、JSONB、FK、CHECK 实测；
25. Scheme Service/Repository race。

### 23.2 前端/浏览器

1. Selector 四组、当前方案名、默认 badge；
2. Simple 多条件固定 AND；
3. Simple -> Advanced 无损；复杂 Advanced 不能错误降为 Simple；
4. Preview 使用 label/operator/dict label；
5. quick/advanced/sort 改动触发 dirty，page/pageSize/refresh/columns 不触发；
6. dirty 切换/离开受控确认；
7. Save 修改和 Save As 行为；
8. 普通用户只保存 PERSONAL；
9. shared manager 四 Tab 管理，共享按钮 capability 正确；
10. read-only 用户可用 visible scheme，但看不到共享写按钮；
11. 无 detail/scope 权限不预加载 Scheme/业务 Detail API；
12. Dynamic Date 跨日/月重新 resolve；
13. DEGRADED issues 展示且不自动 Fetch Data；
14. 方案删除后当前 Query 保留并变临时；
15. 空状态、查询无结果、请求失败、无权限区分；
16. Admin 与 Limited User；
17. 亮色/深色、Toolbar wrap、Dialog maximize、键盘与 tooltip；
18. Console Error/Unhandled Promise/误 403 为 0；
19. 18 个首批 Eligible 页面集成回归；
20. Report 公共组件兼容，仍 REPORT_DEFERRED。

## 24. 性能与运维

- Selector 只加载摘要，完整 Payload 按需 resolve；
- 管理页分页，最大 100/页；
- 常用索引：scope+type+enabled、owner+scope、role_id、gmt_modify；
- V1 不使用 Redis Cache，避免 default/visibility 失效一致性债；
- Query JSON 和 issue 日志不记录敏感值；
- 指标建议只记录 available/resolve latency、validation status count、revision conflicts，不记录查询内容。

## 25. UI 原型

静态交互原型位于 [prototype/index.html](prototype/index.html)。该目录是 **Interaction Prototype / Design Review Artifact**，用于评审信息架构和交互流，不是生产页面视觉定稿。它覆盖：

1. 业务列表 + Scheme Selector；
2. AdvancedQuery Simple；
3. AdvancedQuery Advanced；
4. Save Scheme Dialog；
5. Scheme Manager；
6. Scheme Detail；
7. Empty/Degraded State。

生产 UI 必须沿用当前 Sweet Platform 正式页面视觉和 FE-001/002/003 已冻结 Pattern。业务列表使用 Query Scheme Selector、Quick Presets、Advanced Query、Save Scheme、Business Actions、Column Selector 和 Refresh；Manager 使用当前平台正式表格风格；Detail 使用正式 Drawer/Detail Pattern。

原型中的左侧“原型视图 1~7”导航、REUSE/NEW/EXTEND 技术标签、设计规则说明栏、实现说明文字、`Revision 6` 等技术控制字段和“字段说明 V1 延期”等开发提示均不得进入生产 UI。原型不进入 `frontend/src` 或正式 Router。

## 26. QC-002 拆分建议

实现规模已经超过单个安全 Task，建议拆分：

### QC-002A Backend Scheme Core

Model/Migration、Scope、Repository、Service、Runtime/Management API、Metadata Validator、binding resolver、权限、PostgreSQL/race。完成后业务 Query 协议仍不变化。

### QC-002B Frontend Runtime + Manager

AdvancedQuery Simple/Preview、Toolbar slots、Query State dirty/source、Selector/Save/Manager/Detail、API/types、组件测试与原型对照。

### QC-002C Page Integration + Browser Acceptance

18 个 Eligible 页面接入、Generalization 条件评审、Admin/Limited User、Data Permission E2E、动态日期跨边界、Report 回归和 Query Center Freeze Review。

不得把 A/B/C 压成一个超大提交。QC-001 Reviewer Gate 已通过，允许进入 QC-002A；QC-002A 仍必须遵守本设计的生产代码范围和验收要求。

## 27. Reviewer Gate 最终结论

1. **动态值与现有协议：APPROVED**。采用“标准 Query 快照 + 七类受控 Binding 白名单 + resolve 后输出标准 Query”，不增加业务 Query V2，Binding 不是 DSL。
2. **Scope Identity：APPROVED**。`sys_menu.query_scope_code` 是唯一持久化身份真值；Registry 只提供运行配置，Frontend 只消费 Runtime Scope Config。
3. **Advanced 深度：APPROVED**。V1 UI 最大可编辑深度 2，后端 Schema 防御上限 3；合法第三层必须受限展示并无损保留。
4. **共享权限粒度：APPROVED**。V1 只有一个 `query_scheme_shared_manage`，不分别制造 public/role/page-default 十余个按钮权限。
5. **Query Center 导航：APPROVED**。V1 采用 Hidden Route，不新增左侧菜单。
6. **字段说明/条件树：APPROVED_DEFERRED**。V1 延期；条件预览保留为 REQUIRED。

QC-001 Gate 状态：**`DESIGN_APPROVED_FOR_QC-002A`**。以上六项已完成产品、架构和代码边界 Review，可直接进入 QC-002A。

## 28. V1 最终实现收口

QC-002A 至 QC-002C 按本设计完成后端核心、前端运行与管理能力，以及固定页面接入。最终实现保持以下边界：

1. `sys_menu.query_scope_code` 仍是唯一 Scope Identity；前端没有 route name、Vue path 或常量到 scope 的第二映射。
2. 后端固定 Registry 共 18 个 scope；18 个 scope 均对应真实活动菜单和标准实体列表，并在 QC-002C 全部启用。
3. 页面接入统一通过 `useQuerySchemePage` 组合 `useQueryScope`、`useQuerySchemes` 和现有 `useTableQueryState`。该 composable 只收口初始化、应用、保存和一次刷新边界，不代理业务列表 API。
4. Scheme Selector、Save Dialog、Advanced Query、Quick Presets 和 Toolbar 均继续使用平台公共组件；业务按钮、Data Permission、业务 Query DTO 和 Repository 查询协议未改变。
5. Query Scheme Manager 的“使用方案”采用明确文字动作；共享方案启停仍是独立 Business Action，revision 只用于乐观锁且不向普通用户展示。
6. 18 个固定 scope 当前没有 relation/linkage 高级查询字段。Preview 已覆盖 Metadata label、operator、dictionary label 和 binding label；未来 scope 出现 relation 字段时，必须接入受控 Runtime display resolver 后才可启用该字段的方案预览，不能显示内部 ID。
7. 固定 18 个 scope 中没有 Data Permission 演示业务页。最终验收通过真实 Generalization 查询链证明同一 Scheme 条件与 Admin/Limited User 的 Data Scope 做 AND；未为验收新增生产 scope 或 Query Scheme Service 页面特例。
8. Report 仍为 `REPORT_DEFERRED`，仅做公共组件构建回归；Query Center V1 不接 Report。

最终验收证据见 [QueryCenterV1AcceptanceReport.md](QueryCenterV1AcceptanceReport.md)，冻结结论见 [QueryCenterV1FreezeReview.md](QueryCenterV1FreezeReview.md)。
