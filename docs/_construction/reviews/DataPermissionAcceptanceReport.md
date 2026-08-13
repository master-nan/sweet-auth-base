# Sweet Platform Data Permission V1 正式验收报告

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| 文档目的 | 对 Sweet Platform Data Permission V1 的当前实现、数据库结构、运行链路、安全边界和验收结果进行正式确认 |
| 验收范围 | DP-1 基础层、DP-2 配置层、DP-3 Resolver、DP-4 Adapter 与 Generalization 接入、DP-5 Demo 和旧实现清理 |
| 验收日期 | 2026-08-03 |
| 验收环境 | macOS Darwin 25.5.0 arm64；Go 1.26.2；Node.js 24.14.0；Yarn 1.22.21；PostgreSQL 16.14 |
| 验收基线 Commit | `2390e95ce9dd79b5d43da52bfa0dafe94ec13b4d`（删除旧数据权限并统一新权限链路） |
| 适用版本 | Sweet Platform Data Permission V1 |
| 报告提交 | 报告自身最终 Commit 以 Git 历史中的“完成数据权限正式验收”为准 |

本报告依据当前仓库代码、本次实际执行的测试、PostgreSQL 临时 Schema 验证和 Demo 数据复核编写，不以历史任务回复代替验收证据。

## 2. 模块定位

功能权限回答“用户是否可以进入菜单或访问接口”，由 `sys_menu`、`sys_menu_button` 和 Casbin 负责。

数据权限回答“用户通过功能权限后，可以查看、修改、删除或导出哪些数据”，由本报告验收的 Data Permission 运行链负责。

Data Permission 是 Sweet Platform 的平台底座能力。它提供通用权限配置、主体解析、事实维度解析、结构化结果和执行适配契约，不等同于 TMS、WMS 或 SRM 的具体业务权限实现。

## 3. 最终架构

当前平台只保留以下一条数据权限运行链：

```mermaid
flowchart LR
    A["可信认证上下文"] --> B["SubjectContextBuilder"]
    B --> C["SubjectContext"]
    C --> D["DataResource / DataResourceOperation"]
    D --> E["DataGrant"]
    E --> F["DataPolicy / DataPolicyRule"]
    F --> G["DataOwnershipField"]
    G --> H["Dimension Provider"]
    H --> I["Resolver"]
    I --> J["DataScopeResult"]
    J --> K["MetadataFieldAdapter / RegisteredFieldAdapter"]
    K --> L["Generalization 或固定业务 Repository"]
```

职责边界如下：

- `SubjectContextBuilder` 只从服务端可信来源构建主体上下文。
- Provider 只提供主体在某一 Dimension 下的事实值。
- Resolver 读取配置并组合权限语义，不生成 SQL。
- Adapter 将结构化结果转换为受控执行描述，不重新解释 Policy。
- Repository 或查询构建器执行最终过滤或受控写操作。

仓库中只存在上述运行链。已删除对象的名称仅由清理 Migration 及其回归测试用于识别历史数据库对象。

## 4. 数据模型验收

### 4.1 DataDimensionDefinition

- 职责：定义平台可识别的数据维度、值类型和 Provider 来源。
- 稳定业务键：`code`，API 中表达为 `dimension_code`。
- 关键约束：维度类别和值类型受 CHECK 约束；当前基础值类型为 `bigint` 和 `string`。
- 生命周期：使用平台 `Basic` 状态和软删除字段；基础 Seed 按稳定编码幂等写入，不覆盖管理员维护内容。
- 关系：被 Ownership 和 PolicyRule 通过 `dimension_id` 引用。
- 验收结论：通过。

### 4.2 DataResource

- 职责：定义受数据权限保护的数据资源，不以菜单或路由作为资源身份。
- 稳定业务键：`resource_code`，全局唯一且创建后不可修改。
- 关键约束：资源目标在 table、service、report 中只能选择一个；`permission_enabled` 默认 `false`。
- 生命周期：支持创建、受控修改、停用和启停数据权限；启用前必须通过 Preflight。
- 关系：拥有 Operation、Ownership 和 Grant。
- 验收结论：通过。

### 4.3 DataResourceOperation

- 职责：声明 Resource 支持并可独立启用的数据操作。
- 稳定业务键：活动记录的 `resource_id + operation`。
- 关键约束：操作值限定为 `query`、`detail`、`create`、`update`、`delete`、`export`、`run`；重复操作受唯一约束保护。
- 生命周期：支持批量配置；已被引用的操作不得物理删除，未被引用的操作按现有模型执行停用或删除。
- 关系：Grant 通过相同 Resource 和 Operation 建立授权。
- 验收结论：通过。

### 4.4 DataOwnershipField

- 职责：描述一条 Resource 记录的归属值位于哪个受控绑定位置，以及该值属于哪个 Dimension。
- 稳定业务键：活动记录的 `resource_id + ownership_code`。
- 关键约束：`dimension_id` 必填；`binding_type` 只允许 `metadata_field` 或 `registered_field`；两种绑定目标互斥。
- 生命周期：`resource_id`、`ownership_code`、`dimension_id`、`binding_type`、绑定目标和值类型属于身份字段；普通更新只允许状态变更；被有效 PolicyRule 引用时禁止停用或删除。
- 关系：PolicyRule 必须通过 `ownership_code + dimension_id` 精确匹配。
- 验收结论：通过。

### 4.5 DataPolicy

- 职责：定义可复用的数据范围策略，不直接绑定 Resource。
- 稳定业务键：`code`，API 中表达为 `policy_code`。
- 关键约束：策略类型受控；策略编码创建后不可修改。
- 生命周期：支持创建、详情、分页、名称和描述修改、状态管理及 Rule 配置。
- 关系：PolicyRule 归属于 Policy，Grant 将 Policy 绑定到具体 Subject、Resource 和 Operation。
- 验收结论：通过。

### 4.6 DataPolicyRule

- 职责：描述 Policy 内一条明确的 Ownership、Dimension、范围来源、关系和 Operator 规则。
- 稳定业务键：活动记录的 `policy_id + sequence`。
- 关键约束：`dimension_id` 必填；`scope_source`、`relation`、`operator` 受控；`specified_values` 使用 JSONB 并校验值类型。
- 生命周期：通过 Policy Service 增加、替换或停用；身份字段不能被普通更新改写。
- 关系：必须与目标 Resource 上相同 `ownership_code + dimension_id` 的有效 Ownership 匹配。
- 验收结论：通过。

### 4.7 DataGrant

- 职责：建立 `Subject + Resource + Operation + Policy` 的授权关系。
- 稳定业务键：活动记录的 `subject_type + subject_id + resource_id + operation + policy_id`。
- 关键约束：V1 主体类型只允许 `role` 和 `user`；有效期、主体、Resource、Operation、Policy 和 Ownership 兼容性均由 Service 校验。
- 生命周期：支持创建、批量授权、查询、停用、恢复和软删除；批量失败整体回滚。
- 关系：是主体与可复用 Policy 在具体 Resource Operation 上的绑定。
- 验收结论：通过。

七类 Model 均复用平台 `Basic`，没有引入独立删除模型、SQL 字段、菜单主键或组织实体字段。

## 5. 数据库验收

当前 PostgreSQL 中的新数据权限表为：

1. `sys_data_dimension_definition`
2. `sys_data_resource`
3. `sys_data_resource_operation`
4. `sys_data_ownership_field`
5. `sys_data_policy`
6. `sys_data_policy_rule`
7. `sys_data_grant`

Migration 使用现有注册体系和事务执行，包含必要外键、活动记录唯一索引、枚举 CHECK 和 JSONB 字段。PostgreSQL 专项测试验证了 Migration 重复执行、默认值、唯一约束、CHECK 和外键约束。

清理 Migration 为 `backend/migrate/legacy_data_permission_cleanup.go`。本次在隔离 Schema 中连续执行清理两次，确认旧表安全删除且新七表全部保留。实际开发数据库复核结果：

- 旧表数量：`0`
- 新表数量：`7`
- 旧兼容视图数量：`0`

已删除的旧表为 `sys_data_dimension`、`sys_data_scope_binding`、`sys_role_data_scope`、`sys_user_data_scope_override` 和 `sys_user_dimension_value`。旧名称只保留在清理 Migration 及其测试中，用于幂等识别和回归验证，不构成运行时依赖。

验收结论：通过。

## 6. Ownership 验收

一个 Resource 可以配置多个 Ownership。例如运输订单可以同时具备管理组织、法人主体和企业人员归属。

每条 PolicyRule 必须使用 `ownership_code + dimension_id` 精确选择 Ownership。Resolver 和配置预检均不允许：

- 使用隐式主归属。
- 自动选择第一项。
- 只按 Dimension 猜测字段。
- 跨 Resource 复用 Ownership。

两种绑定方式的边界：

| 绑定类型 | 可信输入 | 校验与执行 |
| --- | --- | --- |
| `metadata_field` | `table_field_id` | 由服务端读取 `sys_table`、`sys_table_field`，校验 Resource、字段状态、可过滤性和类型 |
| `registered_field` | 持久化的 `adapter_field_code` | 由服务端注册表通过 `adapter_code + adapter_field_code` 定位执行能力，并再次校验 Resource、Dimension、Operation、Operator 和 ValueType |

客户端不能提交 SQL、表名、数据库字段名、JOIN 或字段表达式。验收结论：通过。

## 7. Policy 与 Grant 验收

Policy 是可复用权限策略，不保存 `resource_id`、菜单信息或数据表信息。Grant 负责将 Subject、Resource、Operation 和 Policy 绑定起来，并在创建时检查：

- 主体存在且类型合法。
- Resource 与 Operation 存在并允许授权。
- Policy 有效且 Rule 完整。
- 每条 Rule 都能在目标 Resource 中精确匹配有效 Ownership。

V1 支持 `role` 和 `user` 主体，不支持岗位、任职、组织或客户端自定义主体类型。角色授权和用户直接授权在 Resolver 中均为有效授权来源，角色名称没有特殊语义。

当前 `SubjectContext` 要求至少存在一个有效角色。因此，没有任何有效角色的账号即使配置了用户直接 Grant，也无法构建 Resolver 上下文。该边界已列入当前限制，未被写成已支持场景。

验收结论：在当前“平台账号具备至少一个有效角色”的验收范围内通过。

## 8. 决策语义验收

| decision | 最终语义 |
| --- | --- |
| `not_applicable` | 当前 Resource 或 Operation 未启用新数据权限；不表示已经授权全部 |
| `all` | 有效授权明确允许当前 Resource Operation 的全部数据 |
| `none` | 明确无授权，或安全收敛为无数据 |
| `filtered` | 必须完整应用结构化 ConditionGroup 后才能访问 |

非 `filtered` 结果不能携带过滤条件；`filtered` 必须至少包含一个非空组和一个非空条件。空值过滤不会变成全量访问，而是拒绝构造或收敛为 `none`。

无授权、配置异常、Provider 异常、Resolver 异常或 Adapter 异常均不存在 `error -> all` 路径。验收结论：通过。

## 9. 条件组合验收

当前真实组合规则为：

- 同一 Policy 内多个有效 Rule：AND，放入同一 ConditionGroup。
- 不同 Grant、不同角色 Grant 与用户直接 Grant：OR，保留为不同 ConditionGroup。
- `all OR X = all`。
- `none OR X = X`。
- `not_applicable` 不参与授权结果合并，参与合并会返回稳定错误。
- 完全相同的条件和条件组会去重并稳定排序。

过滤结构保持“组内 AND、组间 OR”。当前结果模型不执行复杂布尔分配律展开；`filtered AND filtered` 只在双方各有一个组时提供基础合并，复杂组合由 Resolver 按 Policy 和 Grant 层级直接构造。

验收结论：通过。

## 10. Organization Provider 验收

| Dimension | 值类型 | 当前 relation | 事实来源 |
| --- | --- | --- | --- |
| `legal_entity` | `bigint` | `exact` | Organization Permission Provider 返回员工当前有效法人集合 |
| `management_org` | `bigint` | `exact`、`self_and_descendants` | Organization Permission Provider 返回当前有效管理组织，并按显式 `structure_code` 查询后代 |
| `employee` | `bigint` | `exact` | SubjectContext 中已绑定的 `employee_id` |

`management_org + self_and_descendants` 的处理为：取得全部当前有效管理组织，逐个调用组织树后代能力并包含自身，最终去重和升序排序。循环、孤儿、无效组织或 Provider 错误均安全失败，不退化为当前节点或全部组织。

Data Permission 运行时没有直接读取 `org_*` Repository；组织事实只通过 Organization Permission Provider 获取。验收结论：通过。

## 11. Resolver 验收

### 11.1 SubjectContext 可信来源

- `user_id` 来自 Gin 认证上下文中的服务端身份。
- `role_ids` 来自现有用户角色关系，只保留启用且未删除角色，并去重、排序。
- `employee_id` 来自 Organization 账号绑定服务，客户端不能覆盖。
- `as_of_date` 由服务端当前日期生成，客户端不能提交。

SubjectContext 字段私有，通过构造函数创建；切片访问返回副本；JSON 只序列化 `user_id`、`role_ids`、`employee_id` 和 `as_of_date`。

### 11.2 解析流程

Resolver 依次验证 Resource、Operation、有效 Grant、Policy、全部有效 Rule、精确 Ownership 和 Dimension Provider 结果。同一 Policy 多 Rule 组合为 AND，不同 Grant 的结果按 OR 合并。

每次解析请求可缓存：

- Resource 查询结果。
- Policy 与 Rule 查询结果。
- 相同 Dimension 请求的 DimensionValues。

未实现跨请求全局权限缓存。

### 11.3 诊断摘要

`ResolverSummary` 记录 `resource_code`、`operation`、`decision`、`checked_grant_count` 和 `checked_policy_count`。摘要不包含 SQL、表名、数据库字段、角色详情或内部 Grant/Policy ID。

Resolver 不负责 SQL、GORM Scope、业务数据查询或 Adapter 执行。验收结论：通过。

## 12. Adapter 验收

### 12.1 MetadataFieldAdapter

MetadataFieldAdapter 通过服务端可信的 `table_id + table_field_id` 读取元数据，校验字段属于 Resource 对应表、字段有效、可用于高级查询且类型未漂移。

当前值类型映射：

- `bigint`：元数据 `bigint`、`int`、`tinyint`。
- `string`：元数据 `varchar`、`text`。

主键、表达式字段、非普通字段、文件、富文本及技术字段不能作为过滤字段。

### 12.2 RegisteredFieldAdapter

RegisteredFieldAdapter 使用线程安全、实例隔离的服务端注册表。注册项以 `adapter_code + adapter_field_code` 唯一定位，并校验 `resource_code`、Dimension、ValueType、Operation 和 Operator。重复或冲突注册不会覆盖原项。

公共框架已经完成，但尚未接入真实 TMS、WMS 或 SRM Repository。

两种 Adapter 均：

- 不接收或输出 SQL、表名、数据库列名、JOIN、函数表达式或 ORM Clause。
- 不读取 Grant、Policy、PolicyRule 或 Organization Provider。
- 完整保留组内 AND 和组间 OR。
- 任一 Condition 转换失败时整体失败，不返回部分执行结果。
- 明确区分 `not_applicable`、`allow_all`、`deny_all` 和 `apply_filter`。

验收结论：MetadataFieldAdapter 通过；RegisteredFieldAdapter 公共契约和测试实现通过，真实业务接入未验收。

## 13. 查询链接入验收

Generalization Controller 不解析数据权限，只将可信请求上下文、服务端定位的表和操作交给统一运行时。`LowCodeDataPermissionRuntime` 构建一次 SubjectContext，调用 Resolver 并将结果交给 MetadataFieldAdapter。

| 入口 | operation | 权限应用位置 | 失败处理 | 结论 |
| --- | --- | --- | --- | --- |
| 列表 rows | `query` | Repository 查询条件 | `deny_all` 返回空集合；Resolver/Adapter 失败不执行原查询 | 通过 |
| total | `query` | 与 rows 相同的已过滤查询，移除分页后 count | 与 rows 保持相同过滤语义 | 通过 |
| 详情 | `detail` | 权限条件与业务 ID 同时进入查询 | 无权统一返回数据不存在，不泄露记录是否存在 | 通过 |
| update | `update` | 权限条件与业务 ID 进入受控更新 | 无权或条件不匹配时不写数据 | 通过 |
| delete | `delete` | 权限条件与业务 ID 进入受控软删除 | 无权时不写数据 | 通过 |
| 批量删除 | `delete` | 同一事务内按 ID 集合和权限条件执行 | 影响行数不完整时整体回滚 | 通过 |
| export | `export` | 后端导出查询使用独立 operation | Resolver/Adapter 失败不导出原始全量数据 | 通过 |

rows 和 total 在一次 Service 调用内复用同一权限执行结果。用户查询和数据权限条件的组合为：

```text
UserQuery
AND
(
  PermissionGroup1
  OR
  PermissionGroup2
)
```

通用更新当前拒绝修改 Ownership 对应字段，尚未提供通用的新归属写入能力。

## 14. not_applicable 最终规则

未配置或未启用新数据权限的 Resource 或 Operation 返回 `not_applicable`，查询和写操作保持原业务条件，不追加数据权限过滤。

`not_applicable`：

- 不表示已授权全部。
- 不记录为 `all`。
- 不调用任何旧数据权限链路。
- 不参与 Grant 授权结果合并。

已启用 Resource 若 Resolver 非法返回 `not_applicable`，运行时按配置冲突安全失败，不执行原始全量查询。

验收结论：通过。

## 15. 前端配置能力验收

当前菜单位于“系统管理 → 数据权限”，菜单组件为 `pages/system/data-permission/Index.vue`，页面使用一个正式路由和以下五个标签：

1. 数据资源
2. 归属定义
3. 权限策略
4. 权限授权
5. 配置检查

当前真实能力：

- 数据资源：分页和高级查询、详情、创建、修改、Operation 配置、数据权限启停。
- 归属定义：分页和高级查询、详情、创建、状态变更；身份字段不提供自由修改。
- 权限策略：分页和高级查询、详情、创建、修改、Rule 配置、启停。
- 权限授权：分页和高级查询、详情、创建、启停。
- 配置检查：按 Resource、Policy 或 Grant 调用 Preflight，并展示结构化 `code`、`message` 和对象信息。

页面复用 `BaseContent`、Quasar `q-table`、分页、`AdvancedQuery`、配置弹框和详情弹框。按钮通过 `usePageButtons('system_data_permission')` 读取 `sys_menu_button` 的 `event_action`，接口由 Casbin 权限投影保护。前端没有角色名称或 `admin` 特判，也没有发现旧数据权限页面、API Service 或弹窗残留。

本次前端测试、lint、类型检查和构建均通过。验收结论：通过。

## 16. Demo 验收场景

开发环境验收脚本 `scripts/DataPermissionDemoData.sql` 使用稳定业务键并限制为 development 环境，可重复执行，不进入生产 Seed。

### 16.1 验收数据

- 组织：华东物流中心、上海运输部（华东下级）、华南物流中心。
- 用户：验收用户 A、验收用户 B、验收无授权用户。
- 角色：物流经理、数据权限验收无授权角色。
- Resource：`transport_order`，目标低代码表为 `demo_transport_order`。
- Ownership：`owner_org`，Dimension 为 `management_org`，绑定类型为 `metadata_field`。
- Policy：本组织及下级组织。
- Rule：`effective_org_units + self_and_descendants + in`，显式指定管理组织架构编码。
- Grant：物流经理角色在 `query` 和 `detail` Operation 上绑定上述 Policy。
- 运单：ORD001 属于华东物流中心，ORD002 属于上海运输部，ORD003 属于华南物流中心。

### 16.2 实际结果

| 主体 | rows | total | 详情 |
| --- | --- | --- | --- |
| 用户 A | ORD001、ORD002 | 2 | 可查看 ORD001/ORD002；ORD003 返回数据不存在 |
| 用户 B | ORD003 | 1 | 可查看 ORD003；ORD001/ORD002 返回数据不存在 |
| 无授权用户 | 空 | 0 | 三张运单均返回数据不存在 |

PostgreSQL Demo 脚本连续执行两次成功；自动化端到端测试覆盖 SubjectContext、Grant、Policy、Organization 范围、Resolver、Metadata Adapter、rows、total 和 detail，并通过上述断言。

验收结论：通过。

## 17. 测试验收

### 17.1 后端

| 命令 | 本次结果 |
| --- | --- |
| `cd backend && go test ./... -count=1` | 通过，所有后端包完成 |
| `go test -race ./internal/datapermission ./repository/... ./migrate ./service ./controller -count=1` | 部分通过；Data Permission、Repository、Migration、Controller 通过，Service 因历史 Gin 测试竞态失败 |
| `go test -race ./service -run 'DataPermission\|DimensionProvider\|SubjectContext\|Generalization' -count=1` | 通过，Data Permission 与 Generalization 相关 Service 测试无竞态 |
| `go test ./service -run '^TestDataPermissionDemoAcceptance' -count=1 -v` | 通过，后代关系与完整 Demo 场景全部通过 |

全 Service race 的失败定位为 `backend/service/org_employee_binding_service_test.go:323`：并发测试 goroutine 同时调用 Gin 全局 `gin.SetMode`。该竞态位于 Organization 测试辅助代码，不在 Data Permission Resolver、Provider、Adapter 或 Generalization 运行链中。本 Task 按范围只记录，不修改。

### 17.2 PostgreSQL

使用 PostgreSQL 16.14 实际执行：

| 验证 | 本次结果 |
| --- | --- |
| `TestDataPermissionDomainMigrationPostgreSQLConstraints` | 通过；在隔离 Schema 中执行 Migration 两次并验证约束 |
| `TestRemoveLegacyDataPermissionSchemaPostgreSQL` | 通过；在包含旧表的隔离 Schema 中执行清理两次 |
| 旧表数量 | 0 |
| 新表数量 | 7 |
| 旧兼容视图数量 | 0 |
| `DataPermissionDemoData.sql` 连续执行两次 | 通过 |
| Demo 数据复核 | 3 个用户、3 张运单均存在且稳定 |

PostgreSQL 脚本验证真实数据库对象和幂等性；完整 Resolver 到查询结果的自动化验收使用隔离 SQLite 测试库运行。这两部分共同覆盖数据库方言约束和运行链语义，但尚未建立启动完整 HTTP 服务后针对 PostgreSQL 的黑盒验收套件。

### 17.3 前端

所有命令使用工作区随附 Node.js 24.14.0：

| 命令 | 本次结果 |
| --- | --- |
| `cd frontend && yarn test` | 通过，17 个测试文件、77 个测试用例 |
| `yarn lint` | 通过 |
| `yarn typecheck` | 通过 |
| `yarn build` | 通过 |

构建存在既有的部分 chunk 大于 900 kB 警告及 Node 子进程参数弃用警告，不影响本次构建成功，也不是 Data Permission 专属问题。

## 18. 安全验收

| 检查项 | 结果 | 依据 |
| --- | --- | --- |
| 无 Grant 不返回 all | 通过 | Resolver 返回 `none`，Demo 无授权用户 rows=0 |
| 配置缺失不返回 all | 通过 | Preflight 和 Resolver 稳定失败或返回 `none` |
| Provider 异常不返回 all | 通过 | Dimension Provider 与 Resolver 失败安全测试 |
| Resolver 异常不放行 | 通过 | LowCode Runtime 失败时不执行原查询 |
| Adapter 异常不执行原查询 | 通过 | Metadata/Registered Adapter 整体失败测试 |
| deny_all 不被忽略 | 通过 | Repository 转换为恒假受控条件 |
| not_applicable 不被记录为 all | 通过 | 独立 decision 和执行模式，诊断保留原值 |
| 客户端不能覆盖 resource_code | 通过 | 低代码运行时根据服务端 `table_id` 定位 Resource |
| 客户端不能提交权限条件 | 通过 | Controller 请求 DTO 不包含权限过滤树入口 |
| 客户端不能提交 SQL、表名、字段表达式 | 通过 | Ownership DTO、Adapter Contract 和注册表均拒绝此类输入 |
| Ownership 不跨 Resource 使用 | 通过 | `resource_id + ownership_code + dimension_id` 精确校验 |
| 无权详情不泄露存在性 | 通过 | 权限和 ID 同查，无权统一为数据不存在 |
| update/delete 无权限时不写数据 | 通过 | 权限条件进入受控原子写操作 |
| 批量删除部分无权时整体回滚 | 通过 | 事务内校验受影响行数，数量不符回滚 |
| 旧权限运行链路已不存在 | 通过 | 全仓引用和 Generalization 调用链审计 |

安全验收结论：当前已实现范围通过。

## 19. 性能与复杂度边界

当前代码冻结的上限为：

| 项目 | 上限 |
| --- | --- |
| 单 Policy Rule 数 | 8 |
| 单个 `specified_values` 数量 | 5000 |
| ConditionGroup 数 | 32 |
| 每组 Condition 数 | 8 |
| 总 Condition 数 | 256 |
| 单 Condition values 数 | 5000 |
| 总参数数 | 5000 |

超过上限时配置或运行时稳定失败，不返回截断结果，也不扩大为 `all`。

Resolver 使用请求级 Resource、Policy/Rule 和 DimensionValues 缓存；没有全局权限结果缓存。MetadataFieldAdapter 只在单次调用内缓存元数据读取结果。Ownership 注册表和 Registered Adapter 执行注册表使用读写锁保护，并支持测试实例隔离和并发读取。

## 20. 当前限制

1. RegisteredFieldAdapter 公共框架已经完成，但尚未接入真实 TMS、WMS、SRM Repository。
2. 当前端到端业务验收以 `metadata_field` 为主；`registered_field` 通过公共框架单元测试和竞态测试验收。
3. `create` Operation 尚未形成通用 Ownership 赋值和新归属校验能力；Generalization 创建仍保持现有业务创建链。
4. `legal_entity` 和 `employee` 当前只支持 `exact`；只有 `management_org` 支持 `self_and_descendants`。
5. 当前没有独立 Resolver 诊断页面，诊断能力以结构化摘要和日志为主。
6. 全 Service race 仍存在 Organization 并发测试辅助代码调用全局 `gin.SetMode` 的历史竞态；Data Permission 相关 race 已通过。
7. 尚未完成面向管理员的数据权限配置操作手册。
8. SubjectContext 当前要求至少一个有效角色；无角色账号不能仅凭用户直接 Grant 进入 Resolver。
9. PostgreSQL 已完成真实 Migration、清理和 Demo 数据验证，但完整运行链自动化目前使用 SQLite 隔离库，尚无 PostgreSQL HTTP 黑盒验收套件。
10. 前端五项配置能力当前集中在一个页面的五个标签中，不是五个独立路由。

## 21. 验收结论

**最终结论：有条件通过。**

判断理由：

- 新七类领域模型、数据库约束、配置 Service、Preflight、后台 API 和配置页面均已实现并通过本次测试。
- SubjectContext、Organization Dimension Provider、Resolver、DataScopeResult、MetadataFieldAdapter 和 RegisteredFieldAdapter 的职责边界清晰，失败路径不会扩大权限。
- Generalization 的 rows、total、detail、update、delete、批量删除和 export 已统一接入新运行链。
- PostgreSQL 新七表、旧五表清理、幂等 Migration 和 Demo 脚本已实测通过。
- Demo 对用户 A、用户 B 和无授权用户的 rows、total、detail 结果符合预期。
- 前端测试、lint、类型检查和构建通过。

### 21.1 阻塞项

当前验收范围内没有阻止 Data Permission V1 作为平台底座交付的代码阻塞项。

如果 V1 对外承诺“无任何角色的账号可仅凭 user Grant 获权”，则 SubjectContext 的非空角色约束必须在承诺前调整并补充验收；当前报告未把该场景判定为已支持。

### 21.2 非阻塞后续项

- 修复 Organization 并发测试中的 Gin 全局模式竞态，使全 Service race 门禁完全通过。
- 为真实 TMS、WMS、SRM 注册并验收 RegisteredFieldAdapter 执行器。
- 设计并实现 `create` Operation 的通用归属赋值规则。
- 增加 PostgreSQL 完整 HTTP 黑盒验收。
- 编写管理员操作手册并完成 Data Permission 最终冻结评审。

## 22. 后续建议

1. 编写面向平台管理员的数据权限资源、归属、策略、授权和配置检查操作手册。
2. 组织 Data Permission V1 最终冻结评审，确认无角色用户直接授权和 `create` 归属规则。
3. 选择一个真实业务模块完成 RegisteredFieldAdapter 首次接入，并复用本报告的安全检查项验收。
4. 在 Data Permission 冻结后单独开展平台代码治理审计，包括历史测试竞态和构建包体积告警。

## 附录：正式文档引用检查

本次检查了：

- `docs/_construction/design/DataPermissionDesign.md`
- `docs/_construction/design/DataPermissionOwnershipDesign.md`
- `docs/_construction/reviews/DataPermissionAcceptanceGuide.md`

上述正式设计文档之间的引用均有效。
