# Platform Capability Backlog V1

状态：平台规划阶段最终能力地图。完成评审后，后续 Sprint、Roadmap 和开发任务应从本文件拆解。

本文档不是设计文档、不是编码规范、不是接口实现方案、不是数据库脚本。本文档用于定义 Sweet Platform 的平台能力地图，明确哪些能力应优先作为平台公共能力建设，而不是按单个业务模块临时实现。

## 1. 文档依据

本 Backlog 依据以下已形成的工程和平台文档整理：

- `docs/engineering/PlatformEngineeringGuide.md`
- `design/platform/sweet-platform-core-capabilities-overall-architecture.md`
- `design/platform/sweet-platform-core-capabilities-overall-architecture-review-log.md`
- `design/platform/integration-center-v1-detailed-design.md`
- `design/platform/integration-center-v1-review-result.md`
- `design/platform/organization-master-data-mirror-v1-detailed-design.md`
- `design/platform/organization-master-data-mirror-v1-review-result.md`
- `design/platform/p2a-integration-foundation-implementation-plan.md`
- `design/platform/p2b-organization-foundation-implementation-plan.md`
- 当前项目真实目录结构：`backend/model`、`backend/repository`、`backend/service`、`backend/controller`、`backend/dto`、`backend/migrate`、`frontend/src/pages`、`frontend/src/components`、`frontend/src/api/services`、`frontend/src/composables`、`frontend/src/stores`、`frontend/src/utils`。

## 2. Capability 定义

Capability 是平台可复用能力，必须同时满足：

1. 可独立开发。
2. 可独立测试。
3. 可被多个模块复用。
4. 不依赖具体行业。

符合 Capability 的例子：

- Tree
- Selector
- Metadata
- Permission
- Formatter
- BusinessLink
- Execution
- Runtime Context

不属于通用 Capability 的例子：

- 组织树
- 员工树
- 客户树
- TMS 运单列表
- 某个具体报表页面

这些具体对象或页面应建立在平台 Capability 之上，不应反向定义平台架构。

## 3. 能力分层

| 层级 | 名称 | 定位 |
| --- | --- | --- |
| Layer1 | Infrastructure | 后端基础工程、权限、响应、事务、审计、迁移、配置等基础设施能力。 |
| Layer2 | Metadata | 低代码元数据、字段、关系、字典、formatter、动态查询和动态表单能力。 |
| Layer3 | Platform Capability | Tree、Selector、MasterDetail、BusinessLink、Execution、Organization、Integration、Report 等平台公共能力。 |
| Layer4 | Platform Runtime | CRUD Runtime、Report Runtime、Permission Runtime、Integration Runtime 等可执行运行时。 |
| Layer5 | Business Capability | TMS、WMS、MES、ERP、CRM 等行业或业务能力消费者，不作为平台优先开发对象。 |

## 4. 优先级和工作量口径

| 标记 | 含义 |
| --- | --- |
| P0 | 当前 P2A/P2B/P4 前必须优先具备或补强的基础能力。 |
| P1 | V1 平台能力闭环需要完成的能力。 |
| P2 | V2 扩展能力，依赖 V1 稳定。 |
| P3 | 后续生态能力或较复杂公共能力。 |
| P4 | 业务行业能力或长期演进能力。 |

| 工作量 | 含义 |
| --- | --- |
| S | 小范围增强或规范化。 |
| M | 单个能力可独立完成，涉及前后端或元数据部分改造。 |
| L | 跨多个模块，需要后端、前端、权限、元数据、测试配合。 |
| XL | 平台级长期能力，需单独规划。 |

## 5. Layer1 Infrastructure Capability

| 能力编号 | 能力名称 | 能力层级 | 能力描述 | 主要职责 | 依赖Capability | 被哪些模块引用 | 是否已经存在 | 是否需要开发 | 优先级 | 建议Sprint | 预计工作量 | 开发阶段 | 验收标准 | 未来扩展 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| INF-001 | Repository Foundation | Layer1 Infrastructure | 统一 Repository 访问模式。 | 复用 `BasicRepositoryImpl`、分页查询、状态更新、幂等定位。 | 无 | 低代码、Report、组织、集成、数据权限、行业模块 | 已存在 | 规范化增强 | P0 | Sprint1 | S | 基线能力 | 新模块 Repository 不自造风格，Repository 不调用 Service。 | 扩展批量操作、原子抢占模板。 |
| INF-002 | Transaction Boundary | Layer1 Infrastructure | Service 层统一事务边界。 | 控制业务事务、避免 Controller 写业务、避免跨模块大事务。 | INF-001 | 所有后端模块 | 已存在 | 规范化增强 | P0 | Sprint1 | S | 基线能力 | 新模块事务在 Service 层声明，外部 HTTP 调用不被长事务包裹。 | 事务模板和一致性检查工具。 |
| INF-003 | Unified Response & Error | Layer1 Infrastructure | 统一响应和错误结构。 | 复用 `response.NewResponse`、`SetData`、`SetTotal`、`myerrors`、`ctx.Error`。 | 无 | 所有 API | 已存在 | 规范化增强 | P0 | Sprint1 | S | 基线能力 | 新接口不返回第二套 response，不自定义错误格式。 | 标准错误码目录。 |
| INF-004 | Validation | Layer1 Infrastructure | 请求参数校验能力。 | 复用 `utils.ValidatorBody` 和 DTO 校验。 | INF-003 | 所有 API | 已存在 | 规范化增强 | P0 | Sprint1 | S | 基线能力 | Controller 只做 DTO 校验和调用 Service。 | 参数校验错误统一本地化。 |
| INF-005 | Functional Permission | Layer1 Infrastructure | 功能权限体系。 | 菜单、按钮、角色、Casbin、前端 `usePageButtons`。 | INF-003 | 所有后台页面、低代码、Report、组织、集成 | 已存在 | 持续补齐 | P0 | Sprint1 | M | 基线能力 | 页面和接口均有菜单按钮权限，功能权限不混入数据权限。 | 权限审计和权限差异检查。 |
| INF-006 | Logger & Audit | Layer1 Infrastructure | 审计和操作日志基础能力。 | 操作日志、登录日志、报表执行日志、后续集成执行日志。 | INF-003 | 系统管理、Report、集成、组织同步 | 部分存在 | 增强 | P0 | Sprint1 | M | P2A/P2B | 关键操作可追踪，敏感 payload 不明文暴露。 | Trace 贯穿、审计留存策略。 |
| INF-007 | Migration & Seed | Layer1 Infrastructure | 幂等迁移和种子能力。 | `backend/migrate/main.go`、菜单、按钮、字典、元数据 seed。 | INF-005、META-001、META-004 | 所有平台模块 | 已存在 | 增强 | P0 | Sprint1 | M | P2A/P2B | 重复执行不重复创建，sys_table、dict、menu、button 可幂等补齐。 | Seed 校验报告。 |
| INF-008 | Config & Environment | Layer1 Infrastructure | 平台配置与环境能力。 | 环境变量、系统配置、集成环境标记。 | INF-003 | 集成中心、组织同步、Report、行业模块 | 部分存在 | 增强 | P1 | Sprint4 | M | P2A | 集成系统可标记 dev/test/prod，不引入第二套配置中心。 | 多租户、多环境凭证隔离。 |
| INF-009 | Security & Credential | Layer1 Infrastructure | 凭证安全基础能力。 | 加密、脱敏、密钥版本、凭证不回显。 | INF-006、INF-008 | 集成中心、外部系统、后续 Open API | 部分存在 | 需要开发 | P0 | Sprint4 | L | P2A | token、secret、password 不通过普通接口明文返回，日志脱敏失败不存明文。 | 密钥轮换、证书管理。 |
| INF-010 | Lightweight Worker | Layer1 Infrastructure | 轻量后台执行能力。 | 数据库 waiting 状态扫描、重试抢占、避免 MQ 依赖。 | INF-001、INF-002 | 集成重试、组织同步重试、后续 Job | 未建设 | 需要开发 | P0 | Sprint4 | M | P2A | 不引入 MQ 即可完成基础自动重试，支持单实例锁或数据库锁。 | 正式 Job Center。 |
| INF-011 | File & Attachment Storage | Layer1 Infrastructure | 附件和文件基础能力。 | 文件上传、预览、下载、附件引用。 | INF-006、INF-009 | 低代码、业务模块、Report、消息 | 部分存在 | 增强 | P2 | Sprint7 | L | V2 | 常用文件预览能力保留，重型预览按需加载，权限可控。 | 对象存储、版本、病毒扫描。 |
| INF-012 | Test Harness | Layer1 Infrastructure | 平台测试基线能力。 | Repository、Service、Controller、Seed、Permission、前端构建验证。 | INF-001、INF-003、INF-007 | 所有模块 | 部分存在 | 增强 | P0 | Sprint1 | M | 基线能力 | 新平台模块具备最小后端测试、seed 幂等测试和前端 lint/typecheck/build。 | 契约测试和可视化回归。 |

## 6. Layer2 Metadata Capability

| 能力编号 | 能力名称 | 能力层级 | 能力描述 | 主要职责 | 依赖Capability | 被哪些模块引用 | 是否已经存在 | 是否需要开发 | 优先级 | 建议Sprint | 预计工作量 | 开发阶段 | 验收标准 | 未来扩展 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| META-001 | Table Metadata | Layer2 Metadata | `sys_table` 表级元数据。 | 声明平台配置对象、低代码对象、数据资源候选。 | INF-007 | 低代码、Report、组织、集成、数据权限 | 已存在 | 增强 | P0 | Sprint1 | M | 基线能力 | 新平台表按规范登记，不自造元数据体系。 | 表级能力标签、资源映射。 |
| META-002 | Field Metadata | Layer2 Metadata | `sys_table_field` 字段级元数据。 | 列表显示、查询、表单、字典、输入控件、只读字段声明。 | META-001、META-004 | 低代码、Report 参数、组织选择器、集成页面 | 已存在 | 增强 | P0 | Sprint1 | M | 基线能力 | 字段渲染和高级查询尽量从元数据读取。 | 组织字段类型、数据归属字段。 |
| META-003 | Relation Metadata | Layer2 Metadata | `sys_table_relation` 关系元数据。 | 关联对象显示、关联查询、后续 MasterDetail。 | META-001、META-002 | 低代码、RecordDetail、组织、业务模块 | 部分存在 | 增强 | P1 | Sprint2 | M | V1 | 常见关联显示不硬编码。 | 多级关联、级联查询。 |
| META-004 | Dict | Layer2 Metadata | 字典能力。 | `sys_dict`、前端 dict store、状态和枚举渲染。 | INF-007 | 所有后台页面、低代码、Report、组织、集成 | 已存在 | 持续补齐 | P0 | Sprint1 | M | 基线能力 | 状态、类型、方向、动作不在页面硬编码。 | 字典分组、有效期、多语言。 |
| META-005 | Column Formatter | Layer2 Metadata | 集中列格式化。 | 字典、日期、布尔、状态、关联对象统一展示。 | META-002、META-004 | 列表页、RecordDetail、Report、组织、集成 | 已存在 | 增强 | P0 | Sprint2 | M | 基线能力 | 新列表字段渲染集中处理，不散落模板判断。 | 复杂对象 formatter。 |
| META-006 | Dynamic Query | Layer2 Metadata | 高级查询能力。 | `AdvancedQuery` 根据字段元数据生成查询表达式。 | META-002、META-004、META-008 | 用户、低代码、Report 工作台、组织、集成 | 已存在 | 增强 | P0 | Sprint2 | L | 基线能力 | 新标准列表复用 AdvancedQuery，不自造查询组件。 | 组织选择器查询、范围查询。 |
| META-007 | Dynamic Form | Layer2 Metadata | 动态表单能力。 | `DynamicFormDialog` 按元数据渲染新增、编辑、详情部分字段。 | META-002、META-004、META-008 | 低代码、平台配置页、组织扩展字段、集成配置 | 已存在 | 增强 | P1 | Sprint2 | L | V1 | 适合普通配置，不渲染敏感凭证明文。 | 分组表单、只读详情。 |
| META-008 | Selector Metadata | Layer2 Metadata | 选择器元数据协议。 | legal_entity_select、org_unit_select、employee_select、position_select 等组件声明。 | META-002、CAP-002 | 组织、低代码、Report 参数、数据权限、业务模块 | 规划中 | 需要开发 | P0 | Sprint3 | L | P2B | 四类组织选择器可通过同一元数据协议接入查询、表单、列表、Report 参数。 | 通用远程选择器协议。 |
| META-009 | Menu Metadata | Layer2 Metadata | 动态菜单和组件映射元数据。 | `sys_menu`、page_type、component、option、动态 route meta。 | INF-005 | 低代码发布、Report 发布、平台菜单 | 已存在 | 增强 | P0 | Sprint1 | M | 基线能力 | 固定页、低代码页、报表运行页动态路由不混淆。 | 更多 page_type 规范化。 |
| META-010 | Data Ownership Metadata | Layer2 Metadata | 数据归属字段元数据。 | 标记 legal_entity_id、org_unit_id、employee_id、owner_user_id 等归属字段。 | META-001、META-002、CAP-007 | 数据权限、低代码、Report、业务模块 | 规划中 | 需要开发 | P0 | Sprint5 | L | P4 | 数据权限可按 resource + operation 解析业务表归属字段，不依赖 menu_id 当资源。 | 多归属、多操作策略。 |

## 7. Layer3 Platform Capability

| 能力编号 | 能力名称 | 能力层级 | 能力描述 | 主要职责 | 依赖Capability | 被哪些模块引用 | 是否已经存在 | 是否需要开发 | 优先级 | 建议Sprint | 预计工作量 | 开发阶段 | 验收标准 | 未来扩展 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| CAP-001 | Tree | Layer3 Platform Capability | 通用树能力。 | 树数据结构、节点选择、搜索、展开、左树右表基础能力。 | META-002、META-004 | 菜单、组织架构、分类、低代码树表 | 部分存在 | 增强 | P0 | Sprint2 | L | P2B | 法人架构和管理架构可复用平台树，不写组织专属树框架。 | 懒加载、展开记忆、树权限。 |
| CAP-002 | Selector | Layer3 Platform Capability | 通用选择器能力。 | 远程搜索、树选择、单选多选、编码名称展示、返回标准 ID。 | META-008、META-004、CAP-001 | 组织、低代码、Report 参数、动态表单、业务页面 | 部分存在 | 需要开发 | P0 | Sprint3 | L | P2B | LegalEntitySelect、OrgUnitSelect、EmployeeSelect、PositionSelect 统一协议，不混用 ID。 | 选择器缓存、历史选择。 |
| CAP-003 | MasterDetail | Layer3 Platform Capability | 主从详情页面能力。 | 左树右表、列表详情、RecordDetail、tabs 详情。 | CAP-001、META-005 | 菜单、组织、集成日志、业务主从页 | 部分存在 | 增强 | P1 | Sprint2 | M | P2B | 定制页面内部列表仍复用平台表格和详情。 | 主从联动模板。 |
| CAP-004 | Runtime Context | Layer3 Platform Capability | 运行上下文能力。 | route meta、menu_id、resource、operation、user、trace 上下文统一传递。 | INF-005、META-009 | 低代码、Report、数据权限、集成、业务接口 | 部分存在 | 增强 | P0 | Sprint5 | L | P4 | Report/低代码运行能传递 menuId 作为来源，但数据资源使用 resource + operation。 | 全链路 TraceContext。 |
| CAP-005 | BusinessLink | Layer3 Platform Capability | 技术执行和业务对象关联能力。 | `IntegrationBusinessLink` 关联 execution、batch、record、业务对象。 | CAP-006 | 集成中心、组织同步、未来业务集成 | 规划中 | 需要开发 | P0 | Sprint4 | M | P2A | Execution 不重复保存正式业务关联，BusinessLink 支持一对多业务对象。 | 业务对象跳转、反查日志。 |
| CAP-006 | Execution | Layer3 Platform Capability | 技术执行实例能力。 | execution_no、trace_id、request_id、状态机、payload、重试关系。 | INF-006、INF-009 | 集成中心、组织同步、未来接口调用 | 规划中 | 需要开发 | P0 | Sprint4 | XL | P2A | IntegrationExecution 表达技术执行，技术状态和业务处理状态分离。 | Job 执行、Workflow 执行复用。 |
| CAP-007 | Organization Service | Layer3 Platform Capability | 组织主数据镜像服务。 | 法人、管理组织、人员、岗位、任职只读查询和选择器服务。 | CAP-001、CAP-002、META-008 | 低代码、Report、数据权限、流程、业务模块 | 规划中 | 需要开发 | P0 | Sprint3 | XL | P2B | 提供统一组织查询服务，其他模块不直连 org Repository。 | 历史组织、负责人范围。 |
| CAP-008 | Integration Service | Layer3 Platform Capability | 统一集成中心能力。 | 外部系统、接口、凭证、出入站 REST、payload、重试、日志。 | CAP-005、CAP-006、INF-009、INF-010 | 组织同步、TMS/WMS/MES/ERP/OA/SAP 接口 | 规划中 | 需要开发 | P0 | Sprint4 | XL | P2A | 业务模块正式外部调用经 ExecuteOutbound，不绕开集成中心。 | SOAP/MQ/文件协议预留。 |
| CAP-009 | Report Capability | Layer3 Platform Capability | 报表中心能力。 | 报表定义、设计、运行、导出、发布到菜单、执行日志。 | META-001、META-005、CAP-004 | Report、低代码、业务模块 | 部分存在 | 持续增强 | P1 | Sprint5 | XL | 已有能力延续 | Report 不复制数据权限逻辑，不把报表工作台变成低代码 CRUD。 | 高级 Sheet、复杂布局。 |
| CAP-010 | Data Permission Service | Layer3 Platform Capability | 新数据权限能力。 | resource + operation、策略、授权、解析、查询过滤。 | CAP-007、META-010、CAP-004 | 低代码、Report、业务接口 | 规划中 | 需要开发 | P0 | Sprint5 | XL | P4 | 数据权限调用组织服务，不维护组织树；无法解析时安全失败。 | 解析测试、双跑迁移。 |
| CAP-011 | Workflow Capability | Layer3 Platform Capability | 工作流平台能力。 | 流程定义、任务、审批、状态流转。 | INF-005、CAP-004、CAP-007 | 业务流程、低代码、通知 | 未建设 | 未来开发 | P2 | Sprint6 | XL | V2 | 不在 P2A/P2B 实现，不提前引入流程引擎。 | BPMN、表单联动。 |
| CAP-012 | Message Capability | Layer3 Platform Capability | 消息平台能力。 | 站内信、消息模板、消息记录。 | CAP-014、INF-006 | Workflow、Job、业务提醒 | 未建设 | 未来开发 | P3 | Sprint6 | L | V2 | 不自造 MQ 平台，不替代集成中心。 | 多渠道消息。 |
| CAP-013 | Job Capability | Layer3 Platform Capability | 任务调度能力。 | 定时任务、失败重试、执行记录。 | INF-010、CAP-006 | 集成自动重试、报表订阅、业务任务 | 未建设 | 未来开发 | P2 | Sprint6 | XL | V2 | P2A 仅做轻量 worker，不提前建设完整调度中心。 | 分布式调度。 |
| CAP-014 | Notification Capability | Layer3 Platform Capability | 通知能力。 | 消息聚合、通知规则、通知状态。 | CAP-012、CAP-013 | Workflow、Report、集成异常、组织异常 | 未建设 | 未来开发 | P3 | Sprint6 | L | V2 | 不在当前阶段加入统计卡片或大屏通知。 | 邮件、短信、Webhook。 |
| CAP-015 | Attachment Capability | Layer3 Platform Capability | 附件能力。 | 文件预览、下载、附件关联、权限控制。 | INF-011、INF-005 | 低代码、业务模块、Report、Workflow | 部分存在 | 增强 | P2 | Sprint7 | L | V2 | 保留 open-file-viewer 等正式预览能力，重型插件按需加载。 | 对象存储、审批附件。 |
| CAP-016 | Open API Capability | Layer3 Platform Capability | 平台开放接口能力。 | 对外 API、应用凭证、限流、审计。 | CAP-008、INF-009、INF-006 | 外部系统、生态应用 | 未建设 | 未来开发 | P3 | Sprint7 | XL | V3 | 不在 V1 集成中心中扩成完整 Open API 平台。 | API Marketplace。 |
| CAP-017 | Plugin Extension Capability | Layer3 Platform Capability | 插件扩展能力。 | 插件注册、能力发现、隔离、版本。 | CAP-016、INF-008 | 行业模块、外部扩展 | 未建设 | 未来开发 | P4 | Sprint8 | XL | V3 | 当前不引入插件运行时。 | 插件市场。 |
| CAP-018 | AI Capability | Layer3 Platform Capability | AI 平台能力。 | 智能生成、问答、辅助配置、智能审计。 | CAP-016、META-001、INF-006 | 低代码、Report、运维、业务模块 | 未建设 | 未来规划 | P4 | Sprint8 | XL | V3 | 当前不将 AI 作为 P2 开发范围。 | Agent、RAG、智能表单。 |

## 8. Layer4 Platform Runtime Capability

| 能力编号 | 能力名称 | 能力层级 | 能力描述 | 主要职责 | 依赖Capability | 被哪些模块引用 | 是否已经存在 | 是否需要开发 | 优先级 | 建议Sprint | 预计工作量 | 开发阶段 | 验收标准 | 未来扩展 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| RUN-001 | CRUD Runtime | Layer4 Platform Runtime | 低代码 CRUD 运行时。 | 通用列表、查询、新增、编辑、详情、权限。 | META-001、META-002、META-006、META-007、INF-005 | 低代码、平台配置对象、行业模块 | 已存在 | 增强 | P0 | Sprint2 | L | 基线能力 | 新标准对象优先复用 CRUD Runtime，不重写列表页。 | 树表、主从 CRUD。 |
| RUN-002 | Report Runtime | Layer4 Platform Runtime | 报表运行时。 | 参数、run、export、发布菜单、执行日志。 | CAP-009、CAP-004、CAP-010 | Report、业务报表、菜单报表 | 部分存在 | 增强 | P1 | Sprint5 | L | Report V2 延续 | run/export 后续接入新数据权限，不复制权限解析。 | 高级 Sheet 运行。 |
| RUN-003 | Permission Runtime | Layer4 Platform Runtime | 功能权限和数据权限运行时。 | Casbin 功能权限、数据范围解析、record access check。 | INF-005、CAP-010、CAP-004 | 低代码、Report、业务接口 | 部分存在 | 新数据权限待建 | P0 | Sprint5 | XL | P4 | 功能权限和数据权限严格分离。 | 解析缓存、双跑监控。 |
| RUN-004 | Integration Runtime | Layer4 Platform Runtime | 集成执行运行时。 | 入站接收、出站调用、payload、retry、BusinessLink。 | CAP-008、CAP-006、CAP-005 | 组织同步、外部系统对接、行业模块 | 规划中 | 需要开发 | P0 | Sprint4 | XL | P2A | V1 只支持 HTTP REST 入站和出站。 | 多协议运行时。 |
| RUN-005 | Organization Query Runtime | Layer4 Platform Runtime | 组织查询运行时。 | 有效组织、任职、祖先后代、选择器查询。 | CAP-007、CAP-002、CAP-001 | 数据权限、低代码、Report、Workflow | 规划中 | 需要开发 | P0 | Sprint3 | XL | P2B | 其他模块调用组织服务，不访问 org Repository。 | 历史任职查询。 |
| RUN-006 | Workflow Runtime | Layer4 Platform Runtime | 工作流运行时。 | 流程实例、任务、审批、状态。 | CAP-011、RUN-003 | Workflow、业务模块 | 未建设 | 未来开发 | P2 | Sprint6 | XL | V2 | 不在 P2A/P2B 开发。 | BPMN 和规则引擎。 |
| RUN-007 | Message Runtime | Layer4 Platform Runtime | 消息运行时。 | 消息发送、状态、重试、渠道。 | CAP-012、CAP-014 | Workflow、Job、业务提醒 | 未建设 | 未来开发 | P3 | Sprint6 | L | V2 | 不替代集成中心。 | 站内信、邮件、短信。 |
| RUN-008 | Job Runtime | Layer4 Platform Runtime | 任务运行时。 | 定时任务、执行记录、失败重试。 | CAP-013、INF-010 | 集成、Report 订阅、业务任务 | 未建设 | 未来开发 | P2 | Sprint6 | XL | V2 | P2A 仅做轻量重试，不提前扩展完整 Job Runtime。 | 分布式 Job。 |

## 9. Layer5 Business Capability

Layer5 是业务能力消费者清单，用于说明平台能力最终服务哪些业务方向。Layer5 不应反向定义平台公共架构，也不应在平台能力尚未完成时优先开发具体行业功能。

| 能力编号 | 能力名称 | 能力层级 | 能力描述 | 主要职责 | 依赖Capability | 被哪些模块引用 | 是否已经存在 | 是否需要开发 | 优先级 | 建议Sprint | 预计工作量 | 开发阶段 | 验收标准 | 未来扩展 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BIZ-001 | TMS Capability | Layer5 Business Capability | 运输管理业务能力。 | 运单、车辆、司机、调度、对账等业务对象。 | RUN-001、RUN-003、CAP-007、CAP-008、RUN-002 | TMS 行业模块 | 部分存在 | 后续业务开发 | P4 | Sprint8 | XL | 业务阶段 | 不绕过平台权限、元数据、组织和集成能力。 | 行业模板。 |
| BIZ-002 | WMS Capability | Layer5 Business Capability | 仓储管理业务能力。 | 仓库、库位、入库、出库、库存。 | RUN-001、RUN-003、CAP-008 | WMS 行业模块 | 未建设 | 后续业务开发 | P4 | Sprint8 | XL | 业务阶段 | 优先使用平台 CRUD、选择器和数据权限。 | 仓储行业包。 |
| BIZ-003 | MES Capability | Layer5 Business Capability | 制造执行业务能力。 | 工单、产线、设备、工序、质量。 | RUN-001、RUN-003、CAP-008、CAP-013 | MES 行业模块 | 未建设 | 后续业务开发 | P4 | Sprint8 | XL | 业务阶段 | 不为 MES 单独发明元数据和权限体系。 | 制造行业包。 |
| BIZ-004 | ERP Finance Capability | Layer5 Business Capability | ERP/财务业务能力。 | 往来、凭证、核算主体、报表。 | CAP-007、RUN-002、RUN-003、CAP-008 | ERP、财务模块 | 未建设 | 后续业务开发 | P4 | Sprint8 | XL | 业务阶段 | legal_entity_id 作为法人字段，不使用 organization_id。 | 财务行业包。 |
| BIZ-005 | CRM Capability | Layer5 Business Capability | 客户关系业务能力。 | 客户、联系人、商机、合同、回款。 | RUN-001、RUN-003、CAP-002、CAP-008 | CRM 行业模块 | 未建设 | 后续业务开发 | P4 | Sprint8 | XL | 业务阶段 | 客户树不是平台 Tree 的替代物，只是 Tree 的应用。 | 销售行业包。 |
| BIZ-006 | Industry App Template | Layer5 Business Capability | 行业应用模板能力。 | 基于平台能力组合行业模板。 | RUN-001、RUN-002、RUN-003、CAP-016、CAP-017 | 行业解决方案 | 未建设 | 未来规划 | P4 | Sprint8 | XL | V3 | 模板必须复用平台能力，不携带第二套 UI 或权限。 | 模板市场。 |

## 10. Capability Dependency

以下 Mermaid 图使用调用和依赖方向：调用方或上层能力 --> 被调用方或下层能力。

```mermaid
flowchart TD
  "Business Capabilities\nTMS/WMS/MES/ERP/CRM" --> "Platform Runtime\nCRUD/Report/Permission/Integration"
  "Platform Runtime\nCRUD/Report/Permission/Integration" --> "Platform Capabilities\nTree/Selector/BusinessLink/Execution/Organization/Integration"
  "Platform Capabilities\nTree/Selector/BusinessLink/Execution/Organization/Integration" --> "Metadata\nTable/Field/Dict/Query/Form/Formatter"
  "Metadata\nTable/Field/Dict/Query/Form/Formatter" --> "Infrastructure\nRepository/Transaction/Response/Permission/Seed/Audit"
```

```mermaid
flowchart LR
  "Selector" --> "Selector Metadata"
  "Selector Metadata" --> "Field Metadata"
  "Field Metadata" --> "Table Metadata"
  "Field Metadata" --> "Dict"
  "AdvancedQuery" --> "Field Metadata"
  "DynamicFormDialog" --> "Field Metadata"
  "Column Formatter" --> "Dict"
  "Column Formatter" --> "Field Metadata"
  "Report Parameters" --> "Selector"
  "Low-code Forms" --> "Selector"
```

```mermaid
flowchart TD
  "Organization Sync" --> "Integration Runtime"
  "Integration Runtime" --> "Execution"
  "Execution" --> "Payload"
  "Execution" --> "BusinessLink"
  "BusinessLink" --> "Organization Sync Batch"
  "BusinessLink" --> "Organization Sync Record"
  "Data Permission Runtime" --> "Organization Query Runtime"
  "Report Runtime" --> "Data Permission Runtime"
  "CRUD Runtime" --> "Data Permission Runtime"
```

## 11. Sprint 能力规划

Sprint 以 Capability 为拆分单位，不以单个业务模块为拆分单位。

| Sprint | 主题 | 能力范围 | 目标 | 不做内容 |
| --- | --- | --- | --- | --- |
| Sprint1 | 工程和元数据基线 | INF-001、INF-002、INF-003、INF-004、INF-005、INF-006、INF-007、INF-012、META-001、META-002、META-004、META-009 | 固化 Repository、事务、响应、权限、seed、元数据和测试基线。 | 不做新业务模块，不重构既有功能权限。 |
| Sprint2 | 标准列表和通用页面能力 | META-003、META-005、META-006、META-007、CAP-001、CAP-003、RUN-001 | 统一 q-table、AdvancedQuery、TablePagination、formatter、动态表单、Tree 和 MasterDetail。 | 不做卡片式管理页，不另造查询分页。 |
| Sprint3 | 组织查询和选择器基础 | META-008、CAP-002、CAP-007、RUN-005 | 建立组织镜像查询服务、四类选择器和选择器元数据协议。 | 不做真实源系统适配，不做 HR 维护。 |
| Sprint4 | 集成中心基础运行能力 | INF-008、INF-009、INF-010、CAP-005、CAP-006、CAP-008、RUN-004 | 建立 HTTP REST 入站/出站、Execution、Payload、BusinessLink、凭证、脱敏、轻量重试。 | 不做 ESB、MQ、流程编排、字段映射平台。 |
| Sprint5 | Report 和数据权限运行闭环 | META-010、CAP-004、CAP-009、CAP-010、RUN-002、RUN-003 | Report 接入统一运行上下文，新数据权限以 resource + operation 为核心进入设计和开发。 | 不清理旧数据权限，不复制组织树。 |
| Sprint6 | Workflow/Message/Job V2 基础 | CAP-011、CAP-012、CAP-013、CAP-014、RUN-006、RUN-007、RUN-008 | 在平台基础稳定后设计并建设流程、消息、任务、通知能力。 | 不提前在 P2A/P2B 中实现完整调度中心。 |
| Sprint7 | 附件、Open API 和生态能力 | INF-011、CAP-015、CAP-016 | 完善附件和预览，规划 Open API。 | 不把 Integration Center 扩成完整 Open API 平台。 |
| Sprint8 | 行业能力承载 | BIZ-001、BIZ-002、BIZ-003、BIZ-004、BIZ-005、BIZ-006、CAP-017、CAP-018 | 基于平台能力组合行业模板、插件和 AI 能力。 | 不让行业模块反向破坏平台规范。 |

## 12. Roadmap

### V1：平台基础闭环

V1 的目标是让 Sweet Platform 具备可复用的低代码企业平台基座。

重点完成：

- Infrastructure 基线：Repository、Transaction、Response、Permission、Seed、Audit、Test。
- Metadata 基线：Table Metadata、Field Metadata、Dict、Formatter、AdvancedQuery、DynamicForm。
- Tree 和 MasterDetail 能力统一。
- Selector Metadata 和四类组织选择器协议。
- Organization Mirror V1：法人、管理组织、人员、岗位、任职只读镜像和查询服务。
- Integration Center V1：HTTP REST 入站/出站、Execution、Payload、BusinessLink、凭证、脱敏、重试。
- Report Runtime 延续完善。
- Data Permission V1：在组织服务稳定后进入 resource + operation 模型。

### V2：平台运行增强

V2 的目标是扩展平台运行时能力，覆盖更多企业应用公共场景。

重点规划：

- Workflow Capability。
- Message Capability。
- Job Capability。
- Notification Capability。
- Attachment Capability 增强。
- Report 高级 Sheet 和复杂布局增强。
- Integration Center 协议能力扩展，但仍避免变成 ESB。

### V3：平台生态能力

V3 的目标是面向生态、开放和智能化能力。

重点规划：

- Open API Capability。
- Plugin Extension Capability。
- AI Capability。
- 行业模板和应用市场。
- 更完整的可观测性、审计、插件隔离和扩展治理。

## 13. 平台能力统计

| 层级 | 能力数量 |
| --- | ---: |
| Layer1 Infrastructure | 12 |
| Layer2 Metadata | 10 |
| Layer3 Platform Capability | 18 |
| Layer4 Platform Runtime | 8 |
| Layer5 Business Capability | 6 |
| 合计 | 54 |

按优先级统计：

| 优先级 | 能力数量 | 说明 |
| --- | ---: | --- |
| P0 | 26 | P2A/P2B/P4 前必须优先具备或补强的公共能力。 |
| P1 | 7 | V1 平台闭环增强能力。 |
| P2 | 6 | V2 平台运行增强能力。 |
| P3 | 5 | V2/V3 生态和消息开放能力。 |
| P4 | 10 | 业务行业能力和长期生态能力。 |

## 14. 后续使用规则

【必须遵守】后续新模块开发前，先判断需求属于哪个 Capability，优先补平台能力，再实现业务模块。

【必须遵守】不得为单个业务模块新增第二套 UI、权限、元数据、查询、分页、formatter、选择器或动态表单体系。

【必须遵守】组织中心、集成中心、数据权限、Report、低代码、Workflow、Message、Job、行业模块均应从本 Backlog 拆解任务。

【必须遵守】Layer5 业务能力不得反向驱动 Layer1 到 Layer4 破坏平台规范。

【必须遵守】组织树、员工树、客户树等具体对象不是 Capability，它们应复用 Tree、Selector、Metadata、Runtime 等平台能力。

【推荐】每个 Sprint 应至少交付一组可独立验收的 Capability，并包含后端测试、前端验证、seed 幂等和权限验证。

【推荐】Roadmap 调整时先更新本 Backlog，再拆实施计划。

【未来规划】V2/V3 能力进入开发前，应按工程手册继续走审计、设计、评审、实施计划、开发、测试、交付流程。
