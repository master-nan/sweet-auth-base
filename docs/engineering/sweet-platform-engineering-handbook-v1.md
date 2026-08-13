# Sweet Platform 工程手册 V1

状态：工程规范 V1。后续所有 Sweet Platform 模块开发、评审、实施计划和 Codex 任务默认引用本规范。

本文是 Sweet Platform 工程宪法，约束平台统一开发方式。本文不是 Go 语法规范，不是 Vue 语法规范，也不是单个业务模块的设计文档。

依据材料：

1. `design/platform/sweet-platform-core-capabilities-overall-architecture.md`
2. `design/platform/sweet-platform-core-capabilities-overall-architecture-review-log.md`
3. `design/platform/integration-center-v1-detailed-design.md`
4. `design/platform/integration-center-v1-review-result.md`
5. `design/platform/organization-master-data-mirror-v1-detailed-design.md`
6. `design/platform/organization-master-data-mirror-v1-review-result.md`
7. `design/platform/p2a-integration-foundation-implementation-plan.md`
8. `design/platform/p2b-organization-foundation-implementation-plan.md`
9. 当前项目真实目录：`backend/`、`frontend/src/`、`docs/`、`design/`。

## 0. 规范等级

- 【必须遵守】强制规则，开发、评审和 Codex 任务不得违反。
- 【推荐】默认做法，除非已有项目模式或评审结论证明不适用。
- 【未来规划】当前不强制实现，但后续扩展应沿该方向演进。

## 1. 平台定位

### 1.1 平台目标

【必须遵守】Sweet Platform 是企业级低代码应用平台基座，不是单一业务系统。

【必须遵守】平台公共能力必须服务于低代码、Report、Workflow、Message、Job、行业模块和业务模块。

【必须遵守】新增平台能力必须复用现有功能权限、元数据、查询、分页、字典、formatter、动态表单、动态路由和审计能力。

【推荐】平台模块应优先沉淀通用能力，再被行业模块和业务模块引用。

### 1.2 平台边界

【必须遵守】功能权限和数据权限必须分离。功能权限回答“能不能进入页面、看到按钮、调用接口”；数据权限回答“能访问哪些业务数据”。

【必须遵守】组织中心是组织主数据镜像，不是 HR、OA 或组织源系统。

【必须遵守】集成中心是平台技术执行能力，不是 ESB、MQ 平台、流程引擎或同步业务模块。

【必须遵守】数据权限中心不得维护组织树，不得直接读取组织 Repository。

【必须遵守】Report 不是低代码 CRUD，不得重复普通数据表的新增、编辑、删除和列表发布能力。

【必须遵守】低代码是元数据和通用 CRUD 运行时，不得为每个业务模块硬编码组织字段、查询、分页或 formatter。

### 1.3 公共能力与业务能力

【必须遵守】公共能力包括身份与功能权限、元数据与低代码运行时、组织主数据镜像、数据权限、统一集成中心、Report 中心、审计日志、通用 UI 组件。

【必须遵守】业务能力只能依赖公共能力，不得复制公共能力内部逻辑。

【必须遵守】业务模块不得直接解析 `sys_user` 推导法人、组织、岗位或员工；必须调用组织服务。

【必须遵守】业务模块不得直接使用外部 HTTP 调用绕过统一集成中心完成正式外部接口调用，除非有明确平台例外评审。

### 1.4 禁止事项

【必须遵守】禁止另起一套 UI 框架、权限体系、元数据体系、接口返回体系或 Repository 风格。

【必须遵守】禁止用 `organization_id` 混合法人和管理组织。

【必须遵守】禁止混用 `employee_id` 和 `user_id`。

【必须遵守】禁止混用 `position_id` 和 `role_id`。

【必须遵守】禁止在新数据权限稳定前删除旧数据权限运行链路。

【必须遵守】禁止普通管理员直接配置 SQL 表达式作为数据权限规则。

## 2. 总体架构原则

### 2.1 能力分层

【必须遵守】平台能力位于最底层，行业能力依赖平台能力，业务模块依赖行业能力或平台能力。

```text
平台公共能力
  ↓
行业通用能力
  ↓
业务模块
```

【必须遵守】依赖方向只能从上层调用下层公共服务，不能反向调用业务模块。

【必须遵守】公共服务之间必须避免循环依赖。组织同步可以调用集成中心；数据权限可以调用组织服务；集成中心不得调用组织 Repository。

### 2.2 领域边界

【必须遵守】Identity & Functional Authorization 负责用户、角色、菜单、按钮、Casbin 和 usePageButtons。

【必须遵守】Metadata & Low-code Runtime 负责 `sys_table`、`sys_table_field`、`sys_table_relation`、`sys_dict`、通用查询、动态表单和低代码运行。

【必须遵守】Organization Master Data Mirror 负责法人、管理组织、员工、岗位、任职的本地只读镜像和查询服务。

【必须遵守】Data Authorization 负责数据资源、数据操作、归属字段、策略、授权和运行时过滤。

【必须遵守】Integration Center 负责外部系统、接口定义、技术执行、payload、trace、脱敏、幂等和重试。

【必须遵守】Report Center 负责只读报表、版式输出、运行、导出、发布版本和菜单化运行。

### 2.3 阶段原则

【必须遵守】大型平台能力必须按“审计 → 设计 → Review → 实施计划 → 开发 → 测试 → 交付”推进。

【必须遵守】不得跳阶段开发。P0/P1 未评审通过前，不得进入 P2 开发。

【必须遵守】不得借阶段任务扩大范围。例如 P2A 不得实现 MQ，P2B 不得实现真实组织源适配，Report UI 整改不得改后端权限模型。

## 3. Backend 开发规范

### 3.1 目录规范

【必须遵守】后端新增模块必须优先使用现有目录：

```text
backend/model
backend/repository
backend/repository/impl
backend/service
backend/controller
backend/dto/request
backend/dto/response
backend/migrate
backend/initialize
backend/enum
backend/internal
```

【必须遵守】不得为单个模块新增另一套 `dao`、`biz`、`handler`、`routes`、`schema` 等替代分层目录。

【推荐】模块内文件命名使用领域名，例如 `integration.go`、`org.go`、`report.go`。

### 3.2 Model

【必须遵守】普通持久化模型必须复用 `model.Basic`，除非表明确不需要审计字段或生命周期字段并经过评审。

【必须遵守】模型只表达数据库结构、GORM 映射和必要的基础类型，不写业务流程。

【必须遵守】敏感字段不得直接出现在普通 response DTO 中。

【推荐】时间字段优先沿用项目 `model.CustomTime` 和 `model.Now()` 的时区处理。

### 3.3 Repository

【必须遵守】Repository 只负责数据访问、查询组合、分页、状态原子更新和基础持久化。

【必须遵守】Repository 禁止调用 Service。

【必须遵守】Repository 禁止承载业务流程、权限判断、跨领域编排和外部 HTTP 调用。

【必须遵守】Repository 应优先复用 `BasicRepository`、`BasicRepositoryImpl`、`PaginateAndCountAsync` 等现有能力。

【推荐】复杂查询可以在 Repository 内封装为明确方法，但方法名必须表达查询意图。

### 3.4 Service

【必须遵守】Service 是业务规则、事务、跨 Repository 编排和跨模块服务调用的主入口。

【必须遵守】事务必须由 Service 统一控制，不得在 Controller 中开启或拼接业务事务。

【必须遵守】跨模块调用必须调用对方 Service 或已评审的公共服务接口，禁止直接调用对方 Repository。

【必须遵守】Service 必须保护字段边界。例如组织同步只能更新源字段，平台扩展操作只能更新平台字段。

【推荐】Service 文件过大时可以按职责拆分，但不得拆出新的分层风格。

### 3.5 Controller

【必须遵守】Controller 只负责参数绑定、校验、调用 Service、设置统一响应和处理错误。

【必须遵守】Controller 禁止直接写 Repository。

【必须遵守】Controller 禁止写复杂业务流程。

【必须遵守】Controller 必须使用 `utils.ValidatorBody` 或项目已有校验方式，不得绕过 DTO 校验。

【必须遵守】Controller 必须通过 `response.NewResponse()`、`SetData`、`SetTotal` 等现有响应结构返回。

### 3.6 DTO

【必须遵守】请求 DTO 放在 `backend/dto/request`，响应 DTO 放在 `backend/dto/response`。

【必须遵守】列表查询请求必须优先复用 `request.Basic`，包含 `page`、`num`、`order`、`quick_query`、`expressions`、`filters`、`menu_id` 等平台语义。

【必须遵守】Response DTO 不得直接复用 Model 暴露敏感字段。

【推荐】DTO 命名使用 `{Domain}{Action}Req`、`{Domain}Res`、`{Domain}DetailRes`。

### 3.7 Validator

【必须遵守】新增 API 必须有明确请求 DTO 和校验规则。

【必须遵守】ID、状态、枚举、必填字段、分页参数和权限上下文字段必须在 Controller 或 Service 边界校验。

【推荐】复杂业务校验放在 Service，避免 Controller 膨胀。

### 3.8 Response

【必须遵守】后台 API 返回必须沿用 `response.Response` 和 `response.AdminError`。

【必须遵守】错误必须通过 `ctx.Error(err)` 交给统一响应中间件处理。

【必须遵守】分页接口必须设置 total，不得让前端自行猜测总数。

### 3.9 依赖方向

【必须遵守】Controller → Service → Repository → DB。

【必须遵守】Controller 可以依赖 DTO、Response、Service，不得依赖 Repository 实现。

【必须遵守】Service 可以依赖 Repository、公共领域 Service、内部工具，不得依赖 Controller。

【必须遵守】Repository 不得依赖 Service、Controller 或外部模块业务逻辑。

### 3.10 事务原则

【必须遵守】需要多表一致性的写操作必须在 Service 中显式事务。

【必须遵守】外部 HTTP 调用不得被数据库事务长时间包裹。

【必须遵守】重试状态抢占、幂等记录和关键状态更新必须使用数据库原子更新或唯一约束兜底。

【推荐】批量处理不要整个大批次单事务；应按对象或小批次事务并记录失败项。

### 3.11 错误处理

【必须遵守】业务错误必须使用项目现有 `myerrors` / `AdminError` 机制。

【必须遵守】不得把底层数据库错误原样暴露给前端。

【必须遵守】安全相关错误不得泄露凭证、token、secret 或敏感 payload。

### 3.12 日志规范

【必须遵守】使用项目现有 logger 和审计机制，不另建日志框架。

【必须遵守】安全字段不得写入日志。

【必须遵守】接口执行日志、业务同步日志、审计日志必须区分职责。

【推荐】复杂任务必须记录 trace、业务 key、执行状态和错误摘要。

## 4. Frontend 开发规范

### 4.1 目录规范

【必须遵守】前端新增页面必须优先使用现有目录：

```text
frontend/src/pages
frontend/src/api/services
frontend/src/components
frontend/src/composables
frontend/src/stores
frontend/src/utils
frontend/src/modules
frontend/src/router
```

【必须遵守】平台管理页面优先放在 `frontend/src/pages/system`。

【必须遵守】低代码相关页面优先放在 `frontend/src/pages/develop`。

【必须遵守】Report 相关页面继续使用当前 `frontend/src/pages/report` 或 `frontend/src/pages/report-v2`，目录转正必须单独评审。

【必须遵守】不得为单个模块新增另一套 `views`、`services2`、`hooks2`、`widgets` 等替代目录。

### 4.2 API 目录

【必须遵守】前端 API 封装必须放在 `frontend/src/api/services`。

【必须遵守】API service 必须复用当前 axios 封装和错误处理，不得在页面内直接写 fetch/axios。

【推荐】API service 文件按领域命名，例如 `integration.ts`、`org.ts`、`report.ts`。

### 4.3 Store 目录

【必须遵守】全局状态必须放在 `frontend/src/stores`，不得用页面级全局变量替代。

【必须遵守】字典必须复用 `frontend/src/stores/dict.ts` 或既有 dict cache 机制。

【推荐】只有跨页面共享状态才进入 store；单页面状态保留在页面或 composable。

### 4.4 Hooks / Composables

【必须遵守】可复用页面逻辑放在 `frontend/src/composables` 或模块内 `composables`。

【必须遵守】composable 不得持有和 UI 组件重复的第二套最终状态。

【推荐】查询、分页、运行上下文、按钮权限等可抽 composable；一次性 UI 展示逻辑不要过度抽象。

### 4.5 Component 目录

【必须遵守】通用组件必须放在 `frontend/src/components`。

【必须遵守】模块专用组件必须放在对应页面模块下的 `components`。

【必须遵守】禁止为了单个页面复制一份 TablePagination、AdvancedQuery、DynamicFormDialog。

### 4.6 页面生命周期

【必须遵守】页面加载必须有 loading 状态。

【必须遵守】接口失败必须有 notify 或错误卡片，不得静默失败。

【必须遵守】列表页面必须明确初始化查询、分页、加载、刷新和重置流程。

【推荐】页面进入时加载元数据，再加载业务数据；元数据失败时应可提示并降级。

### 4.7 API 调用

【必须遵守】页面必须调用 `frontend/src/api/services` 中的方法。

【必须遵守】分页请求必须使用后端 `page`、`num` 和 `total` 语义。

【必须遵守】高级查询必须输出后端已有 `expressions` 结构，不得自创新查询协议。

### 4.8 错误处理

【必须遵守】用户可恢复错误使用 Quasar notify 或标准错误区域。

【必须遵守】权限错误不得伪装为空数据。

【推荐】详情加载失败显示返回按钮或关闭入口。

### 4.9 Loading

【必须遵守】列表、详情、保存、删除、导出、重试必须有明确 loading。

【必须遵守】按钮 loading 期间应避免重复提交。

### 4.10 Permission

【必须遵守】页面按钮必须接入 `usePageButtons` 或等价的现有菜单按钮能力。

【必须遵守】按钮显示和操作入口不得只靠前端硬编码角色名。

【必须遵守】前端权限只控制展示和交互，后端仍必须通过 Casbin 和业务校验保护接口。

## 5. 数据库规范

### 5.1 命名

【必须遵守】表名使用小写下划线，按领域前缀命名，例如 `integration_*`、`org_*`、`report_*`。

【必须遵守】字段名使用小写下划线。

【必须遵守】禁止使用模糊字段名表达关键业务对象，例如用 `organization_id` 同时表示法人和管理组织。

### 5.2 主键

【必须遵守】普通业务表主键沿用项目 `model.Basic.Id`。

【必须遵守】业务引用必须保存平台内部 ID，不保存名称。

【推荐】外部源 ID 与内部 ID 分开，使用 `source_id`、`source_code`、`source_system_code` 表达外部来源。

### 5.3 索引

【必须遵守】所有列表查询、高级查询、树查询、幂等定位和外键关联字段必须评估索引。

【必须遵守】幂等、源对象定位、绑定关系等关键规则必须有唯一约束或数据库兜底。

### 5.4 唯一约束

【必须遵守】唯一业务规则不得只靠前端校验。

【必须遵守】可并发写入的唯一规则必须在数据库层兜底。

【推荐】nullable 字段唯一规则使用 partial unique index 方案，但必须在设计中说明。

### 5.5 状态字段

【必须遵守】状态字段必须绑定字典或明确枚举。

【必须遵守】业务状态、同步状态、技术状态、启停状态不得混用。

【必须遵守】`enabled` 不等于当前有效；有效性如涉及时间，必须结合 `valid_from`、`valid_to` 等字段。

### 5.6 审计字段

【必须遵守】普通业务表必须保留创建、修改、删除审计能力，优先复用 `model.Basic`。

【必须遵守】敏感操作必须记录审计，但审计不得记录明文 secret、token、password。

### 5.7 软删除

【必须遵守】平台配置对象和主数据镜像默认不物理删除。

【必须遵守】源系统删除事件进入本地镜像时，应优先转换为停用或 `source_deleted`，不得破坏历史引用。

### 5.8 字典字段

【必须遵守】状态、类型、方向、协议、动作、结果等有限集合字段必须进入 `sys_dict`。

【必须遵守】前端不得散落硬编码状态文案和颜色。

### 5.9 JSON 字段

【必须遵守】JSON 字段只能用于结构灵活、查询要求低、行级管理价值低的配置或扩展信息。

【必须遵守】需要唯一约束、频繁查询、权限控制、独立生命周期的内容必须拆表。

【推荐】接口 Header/Query/Path/Body 基础配置可用 JSON；幂等记录、BusinessLink、同步记录不得合并到 JSON。

### 5.10 Version 字段

【必须遵守】外部源同步对象如存在乱序风险，必须设计 `source_version`、`source_updated_at` 或等价版本判断。

【推荐】平台自有配置对象如需发布版本，必须区分草稿态和运行态版本。

## 6. 低代码元数据规范

### 6.1 进入 sys_table 的条件

【必须遵守】需要标准列表、高级查询、字段 formatter、动态详情或低代码引用的平台对象必须登记到 `sys_table`。

【必须遵守】平台配置对象、主数据镜像对象、运行日志对象如有管理页，应评估进入 `sys_table`。

【推荐】纯技术中间表、payload content 等不适合普通列表展示的字段可登记表但隐藏敏感字段。

### 6.2 进入 sys_table_field 的条件

【必须遵守】列表展示、高级查询、动态表单、详情展示、formatter 需要识别的字段必须登记到 `sys_table_field`。

【必须遵守】敏感字段不得开放普通列表展示或普通动态表单维护。

【必须遵守】只读源字段必须标记只读，不得被 DynamicFormDialog 当作可编辑业务字段。

### 6.3 进入 dict 的条件

【必须遵守】有限枚举字段必须进入 `sys_dict` 和字典项。

【必须遵守】状态 chip、下拉选择、高级查询和表单选项必须优先使用字典。

### 6.4 不能开放的字段

【必须遵守】密码、secret、token、私钥、证书、payload 原文等敏感字段不得通过普通动态表单或普通详情接口开放。

【必须遵守】源主数据正式字段不得作为本地 CRUD 表单字段开放。

【必须遵守】系统内部字段、数据权限解析结果、Casbin policy 不得暴露给普通业务管理员直接编辑。

## 7. 权限规范

### 7.1 功能权限链路

【必须遵守】功能权限统一链路为：

```text
sys_menu
  ↓
sys_menu_button
  ↓
sys_role_menu / sys_role_menu_button
  ↓
Casbin
  ↓
usePageButtons
```

【必须遵守】新增正式页面必须有 sys_menu。

【必须遵守】新增正式按钮必须有 sys_menu_button，并同步 Casbin。

【必须遵守】前端按钮显示必须使用 `usePageButtons` 或平台等价能力。

### 7.2 Action 统一

【必须遵守】优先复用以下 action：

| action | 语义 |
| --- | --- |
| `query` | 查询列表、树或选项 |
| `detail` | 查看详情 |
| `create` | 新增 |
| `update` | 修改 |
| `delete` | 删除 |
| `enable` | 启用 |
| `disable` | 停用 |
| `refresh` | 刷新 |
| `retry` | 重试 |
| `ignore` | 忽略异常 |
| `view_payload` | 查看集成 payload |
| `view_error` | 查看错误详情 |
| `export` | 导出 |
| `test` | 测试连接或测试执行 |
| `cancel` | 取消等待任务 |
| `publish` | 发布版本 |
| `publish_menu` | 发布到菜单 |
| `unpublish_menu` | 取消发布菜单 |
| `version` | 查看版本 |

### 7.3 新增 Action 规则

【必须遵守】只有当现有 action 无法准确表达接口能力时，才允许新增 action。

【必须遵守】新增 action 必须同时说明页面位置、接口 path/method、Casbin 策略、前端按钮显示和审计要求。

【必须遵守】禁止为了一个页面文案随意新增同义 action，例如已有 `detail` 时不得新增 `view_detail`。

### 7.4 数据权限边界

【必须遵守】`menu_id` 可以作为请求来源，但不得等同于数据资源。

【必须遵守】新数据权限必须以 `resource + operation` 为核心上下文。

【必须遵守】功能权限不得表达“本人、本部门、本法人”等数据范围。

## 8. UI 规范

### 8.1 通用组件

【必须遵守】标准后台页面必须优先复用：

1. `BaseContent`
2. `AdvancedQuery`
3. `TablePagination`
4. `RecordDetail`
5. `TreeTable`
6. `MasterDetailPage`
7. `DynamicFormDialog`
8. `q-table`
9. `usePageButtons`
10. `column-format`
11. `dict store`

【必须遵守】禁止复制上述组件实现第二套同类组件。

### 8.2 列表布局

【必须遵守】标准列表使用平台已有 q-table 风格。

【必须遵守】列表必须有 loading、empty、分页和错误处理。

【必须遵守】分页必须使用 `TablePagination`。

【必须遵守】操作列必须使用 icon + tooltip 风格，不得在表格中堆一排大文字按钮。

【推荐】页面主区域使用 `BaseContent + q-pa-sm` 等现有后台布局。

### 8.3 查询布局

【必须遵守】标准查询区必须包含关键字查询、查询、重置、高级查询和刷新能力，具体按钮按页面权限显示。

【必须遵守】高级查询必须使用 `AdvancedQuery`。

【必须遵守】查询结构必须兼容后端 `request.Basic`。

### 8.4 详情布局

【必须遵守】通用详情优先使用 `RecordDetail`。

【必须遵守】复杂主从详情优先使用 `MasterDetailPage`。

【推荐】JSON、错误堆栈、payload 等长文本详情必须默认折叠或按需加载。

### 8.5 树布局

【必须遵守】树形页面优先参考菜单管理页，复用 `TreeTable` 或 `MasterDetailPage`。

【必须遵守】组织主数据树只读，不得出现拖拽调整、添加节点、删除节点等 HR 维护动作。

### 8.6 按钮位置

【必须遵守】顶部按钮、行按钮、详情按钮必须和 sys_menu_button position 对齐。

【必须遵守】危险操作必须有确认。

【推荐】刷新、新增、导出等顶部动作靠右；查询动作靠近查询条件。

### 8.7 禁止自定义风格

【必须遵守】禁止为标准列表页面自定义卡片式列表、独立分页、独立高级查询或大屏视觉。

【必须遵守】禁止在平台管理页面做营销页、仪表盘或统计卡片，除非任务明确要求。

## 9. 页面设计规范

### 9.1 Tree 页面

【必须遵守】存在天然层级且需要选中节点查看详情时，使用 Tree 或左树右表。

【推荐】菜单、法人架构、管理架构适合 Tree / MasterDetail。

【必须遵守】树节点编辑能力必须经过业务边界评审，组织镜像树默认只读。

### 9.2 List 页面

【必须遵守】标准配置、主数据列表、日志列表、人员列表、岗位列表使用 List 页面。

【必须遵守】List 页面必须复用 q-table、AdvancedQuery、TablePagination 和 usePageButtons。

### 9.3 MasterDetail 页面

【必须遵守】左侧对象列表或树，右侧详情/子表的场景优先使用 MasterDetailPage。

【推荐】人员与任职、组织详情、菜单详情等适合 MasterDetail。

### 9.4 Dialog

【必须遵守】简单创建、编辑、绑定、确认类交互使用 Dialog。

【必须遵守】Dialog 内表单应优先复用 DynamicFormDialog 或已有表单组件。

### 9.5 Drawer

【推荐】复杂详情、长表单、需要保留列表上下文的详情可使用 Drawer。

【必须遵守】Drawer 不得变成独立页面的替代品；复杂流程应设计页面。

### 9.6 禁止页面形态

【必须遵守】只读镜像对象禁止做成本地 CRUD 页面。

【必须遵守】日志页面禁止做大屏或卡片流。

【必须遵守】Report 设计器可以是专业工作台，但 Report 工作台中的列表区仍必须遵守平台表格规范。

## 10. 测试规范

### 10.1 Repository Test

【必须遵守】复杂查询、唯一定位、分页、状态抢占、树查询、数据权限过滤适配必须写 Repository 或 Service 层测试。

【推荐】简单 CRUD 且已有通用能力覆盖时，可以只做 Service/Controller 测试。

### 10.2 Service Test

【必须遵守】事务、跨表写入、状态机、同步落库、字段保护、权限解析、重试、幂等必须写 Service 测试。

【必须遵守】敏感字段不泄露必须有测试。

### 10.3 Controller Test

【必须遵守】新增关键 API 必须覆盖请求校验、错误返回和权限路径。

【推荐】Controller 只做薄层时，可通过集成测试覆盖。

### 10.4 Integration Test

【必须遵守】HTTP 出站、HTTP 入站、Report run/export、低代码通用查询等跨层能力必须有集成或 mock 测试。

【推荐】外部系统测试使用 mock server，不依赖真实第三方环境。

### 10.5 Seed Test

【必须遵守】新增字典、sys_table、sys_table_field、菜单、按钮、Casbin seed 必须幂等。

【必须遵守】重复执行 seed 不得重复创建数据。

### 10.6 Permission Test

【必须遵守】新增菜单和按钮必须测试 super_admin 或默认授权路径。

【必须遵守】受保护 API 必须测试无权限失败和有权限成功。

### 10.7 可以省略的情况

【推荐】纯文档变更可以不运行构建和测试，但必须说明未改代码。

【推荐】仅 UI 文案小改可以按风险选择 lint/typecheck/build，但涉及路由、API、权限、全局组件时必须完整验证。

## 11. 文档规范

### 11.1 目录职责

【必须遵守】`design/` 存放评审中的设计材料、详细设计、评审结果和实施计划。

【必须遵守】`docs/` 存放正式沉淀后的工程规范、用户文档、已定稿技术文档。

【必须遵守】`work/` 只放临时探索材料，不得提交正式设计成果。

### 11.2 文档流转

【必须遵守】平台级能力文档流转顺序为：

```text
audit
  ↓
proposal / architecture
  ↓
detailed design
  ↓
review result
  ↓
implementation plan
  ↓
development
  ↓
delivery notes
```

【必须遵守】未经 review result 允许，不得进入开发。

【必须遵守】实施计划不是代码，不得包含 SQL、迁移脚本或 API 实现。

### 11.3 Markdown 要求

【必须遵守】重要平台设计必须落盘为 Markdown，不能只在聊天中给结论。

【必须遵守】文档必须写明状态，例如“设计草案”“评审结果”“开发准备文档”“正式规范”。

【推荐】评审结果文档只记录结论、问题、决策和是否准入，不复制整份详细设计。

## 12. Git 规范

### 12.1 一个 PR 的修改范围

【必须遵守】一个 PR 应围绕一个阶段或一个清晰目标。

【必须遵守】不得在一个 PR 中混合平台架构、业务功能、UI 重构、权限重构和清理工作。

【必须遵守】数据库迁移、后端接口、前端页面、权限 seed 同时出现时，PR 描述必须说明兼容性和回滚风险。

### 12.2 Commit 规范

【推荐】commit 信息应包含模块和动作，例如 `integration: add execution models`、`org: add legal entity query service`。

【推荐】文档-only commit 应明确 `docs:` 前缀。

【必须遵守】不得提交无关格式化、临时文件、调试输出或 work 目录内容。

### 12.3 Review 流程

【必须遵守】涉及平台公共能力、权限、元数据、路由、迁移、数据权限、Report 运行链路的 PR 必须 Review。

【必须遵守】Review 重点先看边界、权限、安全、迁移、兼容性和测试，再看代码风格。

### 12.4 Merge 流程

【必须遵守】合并前必须确认测试结果、迁移影响、菜单权限影响和文档状态。

【推荐】大型阶段合并后补充交付说明或阶段验收记录。

## 13. Codex 工作规范

### 13.1 默认流程

【必须遵守】Codex 开发平台能力必须按以下流程：

1. 审计。
2. 设计。
3. Review。
4. 实施计划。
5. 开发。
6. 测试。
7. 交付。

【必须遵守】如果用户明确说“只读审计”“只输出方案”“不要修改代码”，Codex 不得创建代码、迁移、页面或配置。

【必须遵守】如果任务要求生成文档，Codex 必须落盘 Markdown，不能只回复聊天内容。

### 13.2 审计阶段

【必须遵守】Codex 必须先阅读相关设计文档、AGENTS.md 和现有代码目录。

【必须遵守】Codex 必须优先使用 `rg` / `rg --files` 搜索，不凭记忆猜当前项目结构。

【必须遵守】审计输出必须区分已存在、可复用、需扩展、禁止复用和未发现。

### 13.3 设计阶段

【必须遵守】Codex 不得重新设计已评审通过的 P1/P2 方案。

【必须遵守】Codex 不得提出未评审的新领域模型、新表、新权限体系或新 UI 体系。

【推荐】设计文档必须明确“做什么、不做什么、依赖什么、风险是什么、验收是什么”。

### 13.4 Review 阶段

【必须遵守】Review 文档必须回答是否通过、是否允许进入下一阶段、阻塞问题、延期决策项。

【必须遵守】Review 不得复制详细设计全文，也不得变成新的自由设计。

### 13.5 实施计划阶段

【必须遵守】实施计划只描述未来修改范围、阶段拆分、文件规划、数据库影响、权限影响和验收标准。

【必须遵守】实施计划不得生成代码、SQL、迁移、页面或接口实现。

### 13.6 开发阶段

【必须遵守】Codex 必须小步实现，每次只改一个明确阶段或一个明确页面/组件。

【必须遵守】Codex 必须复用已有架构、目录、组件、权限、元数据和响应格式。

【必须遵守】Codex 禁止自己创造新的 UI、目录、权限、分页、查询、表格、表单体系。

【必须遵守】Codex 禁止扩大开发范围、跳阶段、顺手重构无关模块。

【必须遵守】遇到用户已有改动，Codex 不得回滚或覆盖，必须先理解并兼容。

### 13.7 测试阶段

【必须遵守】代码修改后必须运行相关测试。

【必须遵守】前端功能改动默认执行 `yarn lint`、`yarn typecheck`、`yarn build`。

【必须遵守】后端功能改动默认执行 `go test ./...`。

【推荐】如果测试无法运行，必须说明原因、影响范围和未验证风险。

### 13.8 交付阶段

【必须遵守】最终回复必须列出修改文件、核心变化、验证结果、影响范围和遗留风险。

【必须遵守】文档任务最终回复不得粘贴整份文档全文，只提供路径、大小、摘要和未改代码说明。

【必须遵守】Codex 不得把未完成或未验证的工作描述为已完成。

## 14. 模块专属补充规则

### 14.1 组织中心

【必须遵守】组织中心是本地只读镜像，不提供 HR 维护能力。

【必须遵守】法人使用 `legal_entity_id`，管理组织使用 `org_unit_id`，员工使用 `employee_id`，账号使用 `user_id`。

【必须遵守】组织服务必须作为其他模块访问组织数据的唯一入口。

### 14.2 集成中心

【必须遵守】集成中心 V1 只正式支持 HTTP REST 入站和出站。

【必须遵守】`IntegrationExecution` 是技术执行主体，`IntegrationBusinessLink` 是正式业务关联。

【必须遵守】凭证必须密文存储，payload 必须脱敏后保存。

### 14.3 数据权限

【必须遵守】数据权限中心暂不进入 P2 开发。

【必须遵守】数据权限依赖组织服务稳定后再设计和实现。

【必须遵守】数据权限不得维护组织树，不得直接访问 org Repository。

### 14.4 Report

【必须遵守】Report 不做低代码普通 CRUD。

【必须遵守】已发布报表通过菜单进入通用 ReportRuntimePage。

【必须遵守】Report 不得复制数据权限解析逻辑。

### 14.5 低代码

【必须遵守】低代码能力必须继续围绕 sys_table、sys_table_field、sys_dict、AdvancedQuery、DynamicFormDialog、通用列表和动态路由。

【必须遵守】新增通用字段类型必须扩展元数据协议，而不是在每个业务页面硬编码。

## 15. 附录：当前项目基线清单

### 15.1 后端基线

【必须遵守】后端基线文件包括：

1. `backend/model/basic.go`
2. `backend/repository/basic.go`
3. `backend/dto/request/basic.go`
4. `backend/dto/response`
5. `backend/middleware/response.go`
6. `backend/middleware/casbin.go`
7. `backend/initialize/router.go`
8. `backend/migrate/main.go`
9. `backend/service`
10. `backend/controller`

### 15.2 前端基线

【必须遵守】前端基线文件和组件包括：

1. `frontend/src/components/BaseContent/BaseContent.vue`
2. `frontend/src/components/Table/TablePagination.vue`
3. `frontend/src/components/Query/AdvancedQuery.vue`
4. `frontend/src/components/FormDialog/DynamicFormDialog.vue`
5. `frontend/src/components/TreeTable/TreeTable.vue`
6. `frontend/src/components/MasterDetail/MasterDetailPage.vue`
7. `frontend/src/pages/detail/RecordDetail.vue`
8. `frontend/src/composables/page-buttons.ts`
9. `frontend/src/utils/column-format.ts`
10. `frontend/src/utils/field-metadata.ts`
11. `frontend/src/stores/dict.ts`
12. `frontend/src/api/services`

### 15.3 标准页面参考

【必须遵守】新增后台页面必须优先参考：

1. `frontend/src/pages/system/user/Index.vue`
2. `frontend/src/pages/system/role/Index.vue`
3. `frontend/src/pages/system/menu/Index.vue`
4. `frontend/src/pages/system/audit/Index.vue`
5. `frontend/src/pages/develop/database/Index.vue`
6. `frontend/src/pages/develop/generalization/Index.vue`
7. `frontend/src/pages/detail/RecordDetail.vue`

## 16. 手册执行规则

【必须遵守】本手册是 Sweet Platform 后续工程任务的默认约束。

【必须遵守】任何任务如果需要偏离本手册，必须在设计或实施计划中明确说明原因、影响范围、替代方案和评审结论。

【必须遵守】Codex 后续开发任务默认先检查本手册，再检查对应模块设计文档和 AGENTS.md。

【未来规划】本手册应在 P2A/P2B 完成交付后根据实际实现经验进入 V2 修订。
