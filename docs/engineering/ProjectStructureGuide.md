# Sweet Platform 项目结构说明

本文帮助开发者快速定位“应该改哪里”。它只列稳定入口和领域边界，不逐一罗列仓库文件。
工程规则见[平台工程架构指南](PlatformEngineeringGuide.md)，具体扩展步骤见
[平台扩展开发指南](ExtensionDevelopmentGuide.md)，前端页面模式见
[前端架构与页面模式指南](FrontendArchitectureGuide.md)。

## 1. 根目录

```text
sweet-auth-base/
|- backend/                 Go后端、Migration和Swagger
|- frontend/                Vue 3 + Quasar前端
|- scripts/                 发布、预检、备份、只读Smoke和安全扫描
|- docs/                    用户、工程和运维长期文档
|- .github/workflows/       GitHub发布门禁
|- Makefile                 本地与CI共享命令
|- docker-compose.yml       本地完整环境
|- docker-compose.external.yml  连接外部PostgreSQL/Redis的环境
|- .env.external.example    外部环境变量模板，不包含真实秘密
`- .nvmrc                   前端Node major/minor真值
```

根目录README只提供快速启动和文档入口。业务设计、实现记录和个人配置不进入tracked根目录。

## 2. 后端目录

```text
backend/
|- main.go                  进程生命周期、HTTP和Runner启动关闭
|- api/                     Gin Handler适配与参数绑定
|- controller/              HTTP Controller和响应入口
|- service/                 Application Service、事务和用例编排
|- repository/              Repository接口与查询契约
|  `- impl/                 GORM实现
|- model/                   持久化模型
|- dto/                     Request/Response白名单DTO
|- enum/                    稳定协议枚举
|- internal/                模块内部契约、安全边界和算法
|- initialize/              配置、DB/Redis、Router、Wire和启动资源
|- migrate/                 Migration Registry、Schema步骤和Seed
|- middleware/              认证、Casbin、日志和错误转换
|- config/                  配置结构与解析
|- cmd/                     容器入口、Preflight、Health和静态服务
`- docs/                    生成的Swagger资产
```

### 2.1 Controller、Service与Repository

标准调用链：

```text
Router -> Controller/API -> Application Service -> Repository -> PostgreSQL
                                  |
                                  `-> internal领域能力
```

- Controller/API负责HTTP绑定、调用Service和返回DTO。
- Service负责业务规则、Capability、事务、Audit和跨Repository编排。
- Repository只负责`context.Context`下的数据访问、锁和批量查询。
- Model不是Response DTO，不能直接从Controller返回。
- `internal`只承载真实模块边界，不是通用工具回收站。

### 2.2 `backend/internal`关键区域

| 目录 | 职责 |
| --- | --- |
| `audit` | 请求身份、审计事实和安全日志边界 |
| `cache` | Redis选项、基础缓存与专属安全状态 |
| `database` | PostgreSQL DSN和TLS配置 |
| `datapermission` | Resolver合同、Subject、Dimension、Ownership和DataScope |
| `errors` | 稳定Application Error |
| `integration` | Transport、Execution、Retry、Sync、Worker与Consumer合同 |
| `metadata` | Runtime Metadata只读契约与value contract |
| `migration` | Migration Catalog、Ledger、Checksum和advisory lock |
| `organization/hrsync` | HR Source DTO、Normalizer、SourceKey和Consumer适配 |
| `querycapability` | 字段类型与Operator安全真值 |
| `queryscheme` | Query Payload、Binding、Scope和Validation合同 |
| `reportconfig` | 当前Report配置解析和校验 |
| `security` | 密码、签名和安全配置基础能力 |
| `storage` | Local/OSS、分片暂存和路径安全 |
| `test` | 跨多个后端包复用的最小测试基础设施 |

## 3. 前端目录

```text
frontend/src/
|- boot/                    Axios、权限、i18n等启动插件
|- router/                  静态候选路由与受控动态组件映射
|- layouts/                 主布局
|- pages/                   按业务领域组织的页面
|- components/              稳定公共交互组件
|- composables/             可复用状态与页面机制
|- api/services/            后端API封装
|- modules/                 Query Scheme、Report等模块内类型和算法
|- stores/                  Pinia跨页状态
|- types/                   平台通用协议类型
|- utils/                   稳定跨页纯函数
|- css/                     Theme Token和全局样式
|- i18n/                    中英文资源
`- test/                    前端测试基础设施
```

### 3.1 页面与公共组件

- `pages`负责业务页面编排、业务动作和领域Dialog。
- `components`只放跨页面稳定交互，例如AdvancedQuery、QueryScheme、Table、Detail和FormDialog。
- `composables`负责可复用状态，不直接成为第二套API或页面Store。
- 页面通过`api/services`访问后端，不直接使用裸Axios URL。
- 普通列表复用`StandardTableToolbar`、`TablePagination`、`useTableQueryState`和Runtime Metadata。
- Tree、Master-Detail、Diagnostic和Report保留专属布局，不机械套普通列表。

### 3.2 Query Scheme前端入口

| 入口 | 职责 |
| --- | --- |
| `composables/query-scheme-page.ts` | 页面Scope、初始化、应用、默认、Dirty和保存动作 |
| `components/QueryScheme/QuerySchemeControls.vue` | 标准页面查询方案UI组合，不请求业务列表 |
| `components/QueryScheme/QuerySchemeSelector.vue` | 可用方案分组、当前来源和方案动作入口 |
| `components/QueryScheme/QuerySchemeSaveDialog.vue` | 保存个人方案和另存为 |
| `pages/query-scheme/` | 查询方案管理、编辑与详情 |
| `api/services/query-scheme.ts` | Runtime与Management API |

页面仍持有业务列表请求、分页、列、Business Capability和领域筛选，不创建页面专属Scheme Store。

## 4. 脚本、文档与CI

### 4.1 `scripts`

| 脚本 | 用途 |
| --- | --- |
| `check-tracked-secrets.mjs` | 扫描Git tracked秘密和敏感配置 |
| `check-docs.mjs` | 检查最终docs结构、空文档和相对链接 |
| `preflight-external.mjs` | 外部环境配置和目标安全预检 |
| `db-backup-external.mjs` | 外部PostgreSQL备份、Manifest、Checksum和恢复验证 |
| `smoke-readonly.mjs` | 发布后只读HTTP Smoke |

每个Node脚本旁保留对应`node:test`。脚本属于发布链路时，由`make scripts-test`和
`make release-check`执行。

### 4.2 `docs`

- `user-guide`：用户操作和平台配置。
- `engineering`：架构、目录和扩展规则。
- `operations`：部署、配置、迁移、备份和排错。
- `docs/README.md`：唯一文档导航。

临时分析、原始响应、截图、实现记录和个人环境资料不进入tracked docs。

### 4.3 GitHub Workflow

`.github/workflows/release.yml`是PR/main发布门禁，直接调用`make release-check`。CI提供
PostgreSQL 16和Redis health service，本地与CI不维护两套测试清单。

## 5. 配置与Compose

| 文件 | 用途 |
| --- | --- |
| `backend/config-dev.yaml` | 本地直接运行 |
| `backend/config-docker.yaml` | 仓库内置Compose |
| `backend/config-pro.yaml` | 生产配置结构和安全默认 |
| `docker-compose.yml` | 本地PostgreSQL、Redis、backend、frontend |
| `docker-compose.external.yml` | 外部数据库/Redis部署 |
| `.env.external.example` | 外部变量名模板 |

配置通过`APP_`环境变量覆盖。生产秘密只从部署平台或受保护环境注入，不提交Git。

## 6. 模块技术地图

本节回答四个问题：用户能做什么、请求会经过哪些边界、状态保存在哪里、修改时哪些约束不能绕过。
文件名均以`backend/`或`frontend/src/`为基准。

### 6.1 Auth

**模块功能与用户能力**

Auth负责密码、短信和DingTalk登录，Access/Refresh Token签发与刷新，登出、账号锁定、受限会话、
首次或重置密码后的强制改密，以及认证安全审计。用户看到登录页、验证码、改密提示和登录失败反馈；
不会看到Token黑名单、密码摘要或Credential内部错误。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `middleware/auth.go` | 从HTTP请求提取Token，调用统一认证链并把可信身份放入请求上下文 |
| `service/auth_application_service.go` | 编排登录、刷新、登出和Access Token认证用例，协调Credential、登录状态、Token与审计 |
| `service/auth_credential_service.go` | 实现密码、短信、DingTalk Credential Provider，只负责身份凭据校验 |
| `service/auth_token_service.go` | 签发、校验、刷新、撤销Token，并关闭Refresh与Logout并发窗口 |
| `service/auth_login_state_service.go` | 管理登录失败计数、账号锁定和成功登录状态 |
| `service/auth_audit_service.go` | 把认证结果转换为脱敏审计事实，不保存凭据 |
| `internal/token/` | JWT/HMAC编码、Claims和Token底层合同 |
| `internal/security/password_policy.go` | 密码强度、摘要及重置后的安全规则 |
| `frontend/src/pages/Login.vue` | 登录交互入口 |
| `frontend/src/stores/user.ts` | 持有当前用户与Token状态，登出时统一清理本地会话 |
| `frontend/src/boot/axios.ts` | 附加Token、处理稳定认证错误和Refresh协作 |

**典型链路**

`HTTP登录 -> AuthHandler -> AuthApplicationService -> Credential Provider -> AuthLoginStateService -> AuthTokenService -> AuthAuditService`

**核心对象、权限与扩展**

- 核心对象是`SysUser`、Token Claims、登录尝试计数和认证审计记录；User是账号，不等同Employee。
- 登录端点不依赖业务菜单权限；登录后的每个业务请求仍由认证Middleware与Casbin分别判断身份和能力。
- 新认证方式实现`AuthCredentialProvider`并接入既有编排，不复制Token、锁定或审计流程。
- 修改时必须覆盖Refresh race、Logout、锁定边界、受限会话和错误脱敏。
- 当前不提供任意外部Identity Provider编排器，也不会把第三方凭据写入Token或日志。

### 6.2 System：User、Role、Menu、Application、SMS与AccessLog

**模块功能与用户能力**

System提供用户、角色、菜单、按钮权限、应用凭证、短信模板和访问审计管理。用户可以维护账号与角色、
配置菜单和按钮、轮换应用Secret、发送或查询短信状态，并查看脱敏后的访问日志。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `controller/sys_user_controller.go` | User HTTP绑定、重置密码通知和安全Response输出 |
| `service/sys_user_service.go` | User CRUD、密码变更、角色分配、Token失效与缓存一致性 |
| `service/sys_role_service.go` | Role CRUD及菜单、按钮授权事务 |
| `service/sys_menu_service.go` | 菜单树、用户菜单、MenuButton Capability和已发布页面解析 |
| `service/application_service.go` | Application CRUD、AppKey/Secret生成轮换与认证读取 |
| `service/sms_service.go` | 短信发送、状态查询和模板管理，隔离供应商错误 |
| `service/log_service.go` | LoginLog、AccessLog和事务内Audit写入；读取只返回安全摘要 |
| `model/sys.go` | User、Role、Menu、MenuButton、Dictionary和Metadata基础持久化模型 |
| `model/application.go`、`model/sms.go`、`model/log.go` | 应用、短信和日志状态对象 |
| `initialize/casbin.go` | Casbin Enforcer初始化和持久化规则装载 |
| `initialize/router.go` | 路由分组、认证与权限Middleware注册 |
| `frontend/src/pages/system/` | User、Role、Menu、Application、SMS和Audit页面 |
| `frontend/src/api/services/sys-*.ts` | System领域API封装；Application、SMS和AccessLog使用各自Service文件 |

**典型链路**

`用户列表 -> SysUserController -> SysUserService -> SysUserRepository -> PostgreSQL -> 安全Response DTO`

`菜单加载 -> Auth身份 -> SysMenuService.GetUserMenus -> Role/Menu Repository -> 动态路由与按钮Capability`

**核心对象、权限与扩展**

- `SysRoleMenu`控制页面，`SysRoleMenuButton`控制业务动作；前端按钮隐藏不替代后端授权。
- `query_scope_code`是菜单托管的Query Scope身份，普通菜单请求不能写入固定Scope。
- Application Secret只在创建或轮换结果中返回一次；AccessLog不返回请求/响应Payload。
- 新页面必须同时处理Router、Menu Seed、MenuButton、Casbin和前端动态组件映射。
- Role不是硬编码管理员名称；所有能力按菜单、按钮和后端策略判断。

### 6.3 Metadata：Table、Field、Relation、Index、Runtime与发布

**模块功能与用户能力**

Metadata定义低代码表、字段、关系和索引，并把配置发布为可授权菜单。开发管理员可维护Schema、同步物理结构、
设置字段查询/展示/输入属性并预览Runtime Metadata；业务页面只消费经过安全投影的Runtime合同。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `service/sys_table_service.go` | Table、Field、Relation、Index与DDL/View生命周期；拒绝无法证明无损的关系结构更新 |
| `service/metadata_field_validation.go` | 字段创建和更新共享的Storage/Logical/Display/Input约束校验 |
| `service/low_code_publication_service.go` | 将已配置Table发布或撤销为菜单及按钮权限投影，不管理DDL |
| `service/metadata_runtime_service.go` | 为运行时消费者提供安全Metadata读取、关系选项和缓存生命周期 |
| `internal/metadata/runtime.go` | 定义稳定只读Runtime投影，隔离SysTable持久化模型和管理字段 |
| `internal/metadata/value_contract.go` | SmallInt、Decimal等值合同与精度安全规范化 |
| `internal/querycapability/operators.go` | 后端字段Operator允许集合的安全真值 |
| `repository/sys_table*.go` | Metadata实体读取、锁、批量关系与索引持久化合同 |
| `migrate/metadata_value_contract.go` | Metadata类型编号、约束及历史数据到Canonical合同的迁移 |
| `frontend/src/pages/develop/database/Index.vue` | Metadata配置工作台 |
| `frontend/src/composables/runtime-table-metadata.ts` | 页面生命周期内加载并缓存Runtime Metadata |
| `frontend/src/utils/field-metadata.ts` | 把Runtime字段合同转换为表单和查询可消费的受控能力 |
| `frontend/src/api/services/runtime-relation.ts` | 按已配置Relation读取显示选项，不允许客户端指定任意目标表 |

**典型链路**

`Metadata配置 -> SysTableController -> SysTableService -> Transaction -> Metadata Repository -> PostgreSQL DDL/Metadata -> Runtime缓存失效`

`低代码发布 -> LowCodePublicationService -> Menu/MenuButton/Casbin投影 -> 已发布动态页面`

**核心对象、权限与扩展**

- 核心对象是`SysTable`、`SysTableField`、`SysTableRelation`、`SysTableIndex`和索引字段顺序。
- Storage Type决定数据库存储；Logical Type表达业务值；Display Format控制受控展示；Input Type选择组件。
- Relation由目标表、value field、display field和可选parent/filter映射组成，同时服务List、Detail、Form与Query。
- 扩展字段类型必须贯通Enum、Migration、Runtime DTO、Dynamic Form、Query Capability和跨端Contract测试。
- Runtime Metadata不是权限替代品；DDL和关系物理结构变化不能通过普通Update隐式丢数据。

```text
Metadata Schema -> Runtime Projection -> Dynamic Form / Table / Query
        |
        `-> Low-code Publication -> Menu / Button / Casbin
```

### 6.4 Generalization

**模块功能与用户能力**

Generalization是已发布低代码页面的通用CRUD与查询入口。用户看到由Metadata生成的列表、表单、详情、
高级查询、关系选择和按钮；它不是任意table code数据库浏览器。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `controller/generalization_controller.go` | 从当前用户、菜单和请求构造低代码用例输入 |
| `service/generalization_service.go` | 解析Runtime Table、页面Capability和Data Permission，编排动态CRUD |
| `repository/generalization.go` | 动态字段Query、CRUD和DataScope执行合同 |
| `repository/impl/generalization_impl.go` | 受Metadata字段白名单约束的GORM动态查询实现 |
| `frontend/src/pages/develop/generalization/Index.vue` | 动态列表、查询、表单与详情页面 |
| `frontend/src/api/services/generalization.ts` | Generalization HTTP API封装 |
| `frontend/src/components/FormDialog/DynamicFormDialog.vue` | 表单生命周期、布局、联动、校验与提交编排 |
| `frontend/src/components/FormDialog/DynamicFieldControl.vue` | 单字段Input Type到Quasar控件的渲染边界 |

**典型链路**

`动态列表 -> GeneralizationController -> Published Menu Capability -> MetadataRuntimeService -> DataPermissionResolver -> GeneralizationRepository -> PostgreSQL`

**核心对象、权限与扩展**

- 查询使用统一`QuickQuery + ExpressionGroup + Order`协议，字段和Operator必须经过Runtime Metadata验证。
- 列表、详情、更新、删除均使用同一页面Capability与Data Permission事实，不能只保护列表。
- 业务特殊行为通过固定页面或受控Override实现，不在Generalization中加入table code特例。
- 当前只支持单表Runtime Metadata及已配置Relation，不支持任意JOIN、前端SQL或自由表达式。

### 6.5 Data Permission

**模块功能与用户能力**

Data Permission在功能权限之后限制用户可见和可操作的数据行。管理员配置Resource、Operation、Ownership、
Dimension、Policy、Rule和Grant；业务用户只看到其授权范围内的列表、数量与详情。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `internal/datapermission/resolver_contract.go` | 定义可信Subject、Resource、Operation输入和失败关闭的Resolver合同 |
| `internal/datapermission/data_scope_result.go` | 表达ALL、NONE、FILTERED和NOT_APPLICABLE结果及条件组 |
| `internal/datapermission/ownership_field_registry.go` | 注册固定业务资源可用于Ownership的字段与校验器 |
| `internal/datapermission/metadata_field_adapter.go` | 将低代码Metadata字段适配为数据权限可执行字段 |
| `service/subject_context_builder.go` | 从当前User、Role、Employee和组织事实构造可信SubjectContext |
| `service/data_permission_policy_resolver.go` | 加载Grant、Policy、Rule、Ownership、Dimension并计算DataScopeResult |
| `service/dimension_provider_runtime.go` | 解析本人、角色、管理组织等受控Dimension值 |
| `service/data_permission_config_preflight_service.go` | 配置保存前批量验证跨对象引用和可执行性 |
| `service/data_resource_config_service.go` | Resource和Operation配置写入 |
| `service/data_ownership_config_service.go` | Ownership字段与资源绑定配置写入 |
| `service/data_policy_config_service.go` | Policy和Rule配置写入 |
| `service/data_grant_config_service.go` | Role/User Grant生命周期与审计身份 |
| `frontend/src/pages/system/data-permission/Index.vue` | 数据权限配置工作台 |
| `frontend/src/api/services/data-permission-config.ts` | 配置与Preflight API封装 |

**典型链路**

`业务查询 -> SubjectContextBuilder -> DataPermissionPolicyResolver -> Dimension Provider -> DataScopeResult -> Repository Adapter -> Business Query AND Data Permission`

**核心对象、权限与扩展**

- Resource和Operation是稳定业务身份；Ownership说明业务行归属字段；Dimension提供授权值集合。
- Resolver只接收可信身份，不允许调用方注入Policy、Grant或SQL；解析失败默认拒绝。
- 固定业务新增Ownership需注册字段和校验器；低代码资源使用Metadata Adapter，不写字段白名单旁路。
- 配置管理权限与业务数据读取权限相互独立；详情、更新、删除必须和列表使用同一范围。
- 当前不支持前端自由字段、自由SQL条件或用角色名代替Grant。

```text
Subject + Resource/Operation
            |
     Grant -> Policy -> Rule
                       |
Ownership Field <- Dimension Values
            |
      DataScopeResult -> Repository Query
```

### 6.6 Organization

**模块功能与用户能力**

Organization管理法人、组织单元、管理/法人结构、岗位、人员、账号绑定、任职和HR同步结果。用户可从人员档案
或组织结构两个视角读取信息；Employee与平台登录User保持分离。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `model/org.go` | 法人、OrgUnit、Structure、Position、Employee、Assignment和同步记录持久化模型 |
| `controller/org_controller.go` | Organization查询、详情、Selector、账号绑定和任职HTTP入口 |
| `service/org_service.go` | Organization读取、树、循环校验、人员/岗位/任职和账号绑定用例 |
| `service/org_permission_provider.go` | 把Organization事实投影为数据权限Dimension值 |
| `service/org_permission_tree_provider.go` | 为管理组织范围提供树形后代解析 |
| `service/organization_hr_sync_service.go` | HR Canonical输入幂等写入、版本比较、批次与错误记录 |
| `internal/organization/hrsync/source_dto.go` | 外部HR Source DTO，只表达源协议 |
| `internal/organization/hrsync/normalizer.go` | 将源字段、日期和状态转换为Canonical Organization输入 |
| `internal/organization/hrsync/source_key.go` | 生成稳定SourceKey，防止跨对象或跨来源身份碰撞 |
| `internal/organization/hrsync/consumer.go` | Integration Consumer到Organization同步用例的适配边界 |
| `frontend/src/pages/organization/` | 结构、人员、岗位、同步批次与错误页面 |
| `frontend/src/api/services/org.ts` | Organization统一API封装 |

**典型链路**

`员工详情 -> OrgController -> OrgService -> Employee/Assignment/User Repository -> 安全Organization DTO`

`HR响应 -> Integration Consumer -> HR Source DTO -> Normalizer -> OrganizationHRSyncService -> SourceKey/Version检查 -> PostgreSQL`

**核心对象、权限与扩展**

- OrgUnit是业务主体，StructureNode是某种结构中的位置；不得把节点ID当稳定组织ID。
- Position不是Role；Employee不是User；Assignment表达员工与法人、组织、岗位在有效期内的关系。
- 结构类型必须显式区分management与legal，树查询执行循环和父子合法性校验。
- 新HR来源在Source Adapter层转换，不把sendpost、userType等源字段扩散到Domain。
- 生产HR Consumer默认未启用；平台不猜主任职/兼职，不根据模糊事件自动处理再入职。

```text
HR Source -> Source DTO -> Normalizer -> Canonical Input
                                      |
                                      v
                           OrganizationHRSyncService
                                      |
                 LegalEntity / OrgUnit / Position / Employee / Assignment
```

### 6.7 File

**模块功能与用户能力**

File提供普通上传、分片上传、合并、进度查询、详情、删除、预览和下载。用户通过业务页面或文件组件访问文件；
服务端按所有权、业务引用、签名用途和存储路径共同授权。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `controller/file_upload_controller.go` | multipart、分片初始化/上传/合并和Actor适配 |
| `controller/file_access_controller.go` | Preview/Download签名URL和文件流HTTP响应 |
| `controller/file_metadata_controller.go` | 文件详情与删除HTTP入口 |
| `service/file_upload_service.go` | 普通上传、分片会话、校验、合并补偿和过期分片清理 |
| `service/file_access_service.go` | 所有权授权、purpose签名、过期校验和安全响应Header |
| `service/file_metadata_service.go` | Metadata读取、删除生命周期和物理存储补偿 |
| `internal/storage/storage.go` | Local与OSS共享的最小Storage接口 |
| `internal/storage/chunk_staging.go` | 受控暂存目录、合并和TTL清理，不能访问持久文件目录 |
| `internal/storage/local.go`、`internal/storage/oss.go` | Local/OSS存储实现和路径安全 |
| `frontend/src/components/FileUpload/` | 上传、进度、显示和Preview Dialog |
| `frontend/src/api/services/file.ts` | 文件上传、访问和Metadata API封装 |

**典型链路**

`普通上传 -> FileUploadController -> FileUploadService -> Storage.Put -> FileRepository -> FileDetail DTO`

`分片上传 -> Init Session -> LocalChunkStaging.Write -> MergeChunks -> Storage.Put -> Transaction写File -> 清理暂存目录`

`文件预览 -> FileAccessController -> FileAccessService校验Actor/purpose签名 -> Storage.Get -> 安全Content-Type/CSP`

**核心对象、权限与扩展**

- `File`保存稳定UUID、路径、大小、类型和创建者；Upload Session保存分片归属和状态。
- Preview与Download签名purpose不可互换；签名不等于业务记录授权，私有文件仍需Actor检查。
- 新Storage实现只实现既有接口，不绕过路径安全、补偿和Metadata事务。
- 本地分片暂存依赖实例粘性；当前不提供共享Chunk Storage或大型取消协议，放弃会话由TTL清理兜底。

### 6.8 Integration

**模块功能与用户能力**

Integration管理外部系统、加密Credential、接口版本、RetryPolicy、SyncTask、Batch、Execution、Attempt和调用日志。
用户可配置并启停接口、手工执行同步、查看执行状态与失败原因；Worker负责真实网络调用和重试推进。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `service/external_system_service.go` | ExternalSystem配置、启停和引用校验 |
| `service/credential_service.go` | Credential加密、轮换、吊销、脱敏读取与审计 |
| `service/interface_definition_service.go` | InterfaceDefinition版本、输入合同、启停和系统关联校验 |
| `service/retry_policy_service.go` | RetryPolicy版本、退避参数、错误分类和启停规则 |
| `service/integration_sync_service.go` | SyncTask版本与启停、手工Batch创建、Batch读取 |
| `service/integration_sync_coordinator.go` | 从到期SyncTask创建Batch，并把Batch协调为Execution |
| `service/integration_execution_service.go` | 创建Execution、冻结接口/输入/Retry快照，提供查询和取消用例 |
| `internal/integration/execution_engine.go` | 单次Attempt状态推进、Transport结果归类和Retry决策 |
| `internal/integration/worker_runner.go` | 周期Claim可执行记录，受并发上限和Context取消控制 |
| `internal/integration/sync_runner.go` | 周期调度到期SyncTask并协调Batch，停止后不再Claim |
| `internal/integration/transport_client.go` | 执行受EndpointPolicy、Timeout和大小限制的HTTP调用 |
| `internal/integration/credential_provider.go` | 调用时解密Credential并生成受控认证材料 |
| `internal/integration/sync_consumer_registry.go` | 按稳定Consumer code解析成功响应的业务Consumer |
| `internal/integration/retry_decision.go` | 根据冻结策略、Attempt结果和随机源计算下一次Retry |
| `repository/integration_execution.go` | Execution/Attempt的锁、Claim、lease和状态更新合同 |
| `frontend/src/pages/integration/` | 配置、同步、执行和日志页面 |
| `frontend/src/api/services/integration.ts` | Integration领域API封装 |

**典型链路**

`外部接口执行 -> IntegrationExecutionService创建冻结快照 -> Worker Claim -> ExecutionEngine -> CredentialProvider -> Transport -> Attempt结果 -> Retry/Terminal状态`

`同步任务 -> SyncRunner -> IntegrationSyncCoordinator -> Batch -> Execution -> Worker -> Consumer -> Checkpoint/Batch汇总`

**核心对象、权限与扩展**

- 配置对象使用revision和版本语义；Execution冻结接口、输入和Retry快照，运行中不读取易变配置替代历史事实。
- Credential明文只在调用瞬间进入Provider，不能出现在Controller、Execution快照、日志或Consumer。
- 新业务Consumer注册稳定code，只处理成功Transport结果；不自行发HTTP、重试、推进Execution或写Checkpoint。
- Worker/Runner必须响应Context取消，Claim、lease、重试和终态更新保持幂等。
- 当前不提供任意脚本Consumer、Response Artifact长期存储或跨系统分布式工作流编排。

```text
ExternalSystem + Credential + InterfaceDefinition + RetryPolicy
                              |
SyncTask -> Coordinator -> Batch -> Execution / Attempt
                                      |
                                    Worker
                                      |
                       CredentialProvider -> Transport
                                      |
                                  Consumer
                                      |
                              Checkpoint / Summary
```

### 6.9 Query Scheme

**模块功能与用户能力**

Query Scheme让用户保存、应用和管理个人、公共、角色及页面默认查询方案。方案保存Quick Query、Expression、
Order和Binding，不保存分页或列偏好；业务查询仍由所属页面发起。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `internal/queryscheme/scope.go` | 固定Scope Registry、QuickPreset和允许Binding声明 |
| `internal/queryscheme/types.go` | Scheme类型、Payload、Validation结果和ResolvedQuery合同 |
| `internal/queryscheme/validator.go` | 按Runtime Metadata验证字段、Operator、排序、深度和Binding引用 |
| `internal/queryscheme/binding.go` | 在Resolve时把动态日期、当前用户和当前员工绑定写入表达式副本 |
| `service/query_scheme_service.go` | Personal/Shared创建、更新、默认、启停、删除及revision事务 |
| `service/query_scheme_runtime.go` | Scope配置、Available、Visibility、Resolve、Detail与CopyToPersonal |
| `service/query_scheme_projection.go` | 安全Runtime/Management DTO投影和Scope Label批量装配 |
| `repository/query_scheme.go` | Scheme、Role关系、默认和revision锁定读写合同 |
| `controller/query_scheme_controller.go` | Runtime与Management HTTP入口及Capability适配 |
| `frontend/src/composables/query-scheme-page.ts` | 页面初始化、默认应用、Dirty、保存、切换和临时条件状态 |
| `frontend/src/components/QueryScheme/` | Selector、Controls、Preset、Preview和Save Dialog |
| `frontend/src/pages/query-scheme/` | Hidden Route管理页、详情Drawer和Shared编辑 |
| `frontend/src/modules/query-scheme/` | 前端集中类型、名称和跨页面事件 |

**典型链路**

`选择方案 -> Runtime Available摘要 -> Resolve -> Visibility + Metadata Validator + BindingResolver -> ResolvedQuery -> useTableQueryState -> 业务列表查询`

**核心对象、权限与扩展**

- Scope身份只来自`sys_menu.query_scope_code`；Scope Config提供运行配置，不是第二身份来源。
- PERSONAL仅Owner可见；PUBLIC、ROLE、PAGE_DEFAULT按页面权限、角色和启用状态解析。
- 个人默认优先于页面默认；ROLE/PUBLIC不自动成为默认；revision冲突拒绝覆盖。
- 新Binding Kind必须进入后端白名单、Scope声明、Validator、Resolver和前端受控标签，不能保存解析后的真实用户ID。
- Resolve只返回查询状态，不调用业务列表API；Data Permission在业务查询阶段继续AND叠加。

### 6.10 Report

**模块功能与用户能力**

当前Report支持报表定义管理、table/sql dataset配置、参数、sheet/cell/binding设计、设计预览、发布版本、
运行态查询、后端导出、执行日志和发布为菜单。设计端与运行端共享受控配置解析，不允许任意外部数据源。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `model/report.go` | ReportDefinition、发布版本与ExecutionLog持久化对象 |
| `service/report_service.go` | 定义CRUD、发布、菜单发布、Preview、Run和Export用例编排 |
| `service/report_access.go` | Report列表/详情授权、菜单Capability和安全DTO读取 |
| `service/report_execution_budget.go` | 运行Deadline、行数/大小预算和终态执行日志 |
| `internal/reportconfig/config.go` | query_config、layout_config、dataset、sheet/cell/binding解析校验 |
| `repository/report.go` | 定义、版本、执行日志和发布状态持久化合同 |
| `controller/report_controller.go` | Report HTTP入口、文件导出和当前Gin上下文适配 |
| `frontend/src/pages/report/` | 当前报表管理、设计、版本和Runtime预览入口 |
| `frontend/src/pages/report-v2/` | 当前轻量Sheet工作台、Designer和Runtime Shell |
| `frontend/src/modules/report/` | Sheet Schema、单元格算法、选项和类型合同 |
| `frontend/src/api/services/report.ts` | Report API及导出封装 |

**典型链路**

`运行报表 -> ReportController -> ReportAccess授权 -> 发布版本配置 -> reportconfig校验 -> Dataset Query + Data Permission -> Execution Budget -> Preview/Export + ExecutionLog`

**核心对象、权限与扩展**

- 运行态读取已发布版本，不直接使用正在编辑的草稿；菜单发布与Report发布状态共同决定入口。
- table dataset复用SysTable/Field发现能力；SQL dataset必须经过只读SQL守卫、参数绑定和执行预算。
- 扩展应围绕现有sheet/cell/binding、发布隔离、导出和日志演进，不另建数据源或设计器体系。
- 当前暂不支持外部数据库数据源、自由画布、图表大屏、打印分页、填报、调度订阅和多数据集复杂联动。

### 6.11 Migration与Seed

**模块功能与用户能力**

Migration在应用发布前把数据库升级到当前Canonical Schema；Seed补齐平台必需字典、菜单、按钮和固定配置。
普通用户不可操作Migration，运维人员通过正式命令执行、adopt既有Canonical数据库并运行Preflight。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `migrate/main.go` | Migration/Seed命令入口、执行模式和共享初始化 |
| `migrate/registry.go` | 按Catalog顺序绑定实际Migration函数，并在事务成功后写Ledger |
| `internal/migration/catalog.go` | Version、Key、Contract、Checksum的唯一有序目录 |
| `internal/migration/ledger.go` | Ledger建表、读取、Checksum验证和PostgreSQL advisory lock |
| `migrate/*_schema.go` | 各领域Schema与Canonical数据迁移，必须幂等且可从旧状态升级 |
| `migrate/*_seed.go` | 当前字典、菜单、按钮和业务基线Seed |
| `cmd/db-preflight/main.go` | 只读检查Schema、关键约束、Seed与Ledger完整性，不执行Migration |

**典型链路**

`migrate命令 -> Catalog校验 -> PostgreSQL advisory lock -> 读取Ledger -> 校验Checksum -> 单步事务 -> 写Ledger -> Seed -> Preflight`

**核心对象、权限与扩展**

- 已执行Version的Checksum变化立即失败；Migration失败不得写成功Ledger；并发实例只有持锁者执行。
- Fresh DB直接按Catalog执行；既有Canonical DB必须显式adopt；部分升级DB不能盲目标记完成。
- 新Migration只追加Catalog与Runner，不改已发布Migration合同；SQLite测试不能证明PostgreSQL DDL正确。
- Seed必须幂等并使用稳定code，不能依赖本地自增ID或覆盖管理员业务数据。

### 6.12 Cache与Redis

**模块功能与用户能力**

Cache为配置读取、Token撤销、验证码、登录锁定和第三方身份映射提供运行时加速或原子安全状态。
用户不会直接操作Redis；缓存不可成为权限、持久业务状态或Migration事实的唯一来源。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `internal/cache/cache.go` | 最小Cacher合同 |
| `internal/cache/basic_cache.go` | 适用于配置模型的泛型读写与失效实现 |
| `internal/cache/redis.go` | Redis基础操作及验证码、锁定等Lua原子操作 |
| `internal/cache/redis_options.go` | Runtime、Migration和Preflight共享的Redis/TLS连接合同 |
| `internal/cache/token_black_cache.go` | Token/Session撤销状态 |
| `internal/cache/login_attempt_cache.go` | 登录失败计数和锁定原子边界 |
| `internal/cache/send_code_cache.go` | 验证码写入、消费和猜测限制 |
| `initialize/redis.go` | Redis客户端创建、连通性校验和关闭注册 |

**典型链路**

`Service读取 -> 专属Cache -> Redis命中/Repository回源 -> Cache写入`；安全状态使用专属原子方法，不套普通缓存。

**核心对象、权限与扩展**

- 配置写入成功后显式失效；缓存失败不能覆盖数据库主错误，也不能制造成功假象。
- Token黑名单、验证码和登录锁定具有专属TTL与原子语义，不能合并成通用Key操作。
- 新缓存必须先证明重复读取价值，并定义Key、TTL、失效所有者和Redis不可用时的失败策略。

### 6.13 Audit与Request Metadata

**模块功能与用户能力**

Audit记录登录、访问、管理变更和关键状态操作；Request Metadata携带request ID、trace ID、客户端地址和
安全User-Agent摘要。管理员只能读取脱敏事实，不能修改审计记录。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `middleware/log.go` | 在HTTP边界采集请求结果并写AccessLog |
| `service/log_service.go` | 异步普通日志与事务内业务Audit的统一写入边界 |
| `internal/audit/subject.go` | 与Gin解耦的可信操作者快照 |
| `internal/audit/request_metadata.go` | 可进入事务和后台终态记录的安全请求元数据 |
| `internal/audit/correlation.go` | request/trace关联标识提取和传播 |
| `internal/asynctask/context.go` | 从请求派生不携带Gin对象或敏感Payload的后台Context |
| `dto/response/stock_boundary_res.go` | AccessLog安全摘要DTO，排除Payload和内部关系ID |

**典型链路**

`HTTP Middleware -> Request Metadata/Audit Subject -> Application Service -> TransactionalAuditWriter -> AccessLog Repository`

**核心对象、权限与扩展**

- 事务内变更审计必须和业务写入同成败；取消后的终态日志使用有界脱离Context。
- 不记录密码、Token、Credential、原始HR响应或完整请求/响应Payload。
- 新审计动作使用稳定resource/action和安全changes摘要，不从`err.Error()`构造客户端内容。

### 6.14 Runtime Lifecycle与Shutdown

**模块功能与用户能力**

Runtime Lifecycle负责HTTP、Cron、Integration Worker、Sync Runner、Chunk清理和外部连接的启动与有序关闭。
用户只感知服务可用性；运维通过日志、healthz和readyz判断状态。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `main.go` | 组装HTTP Server与Runner，接收SIGINT/SIGTERM并执行统一Shutdown顺序 |
| `initialize/wire.go` | Composition Root，装配Controller、Service、Repository和Runtime依赖 |
| `initialize/corn.go` | 启停Cron并等待在途任务 |
| `initialize/integration_worker.go` | 构造Execution Worker及其Runtime依赖 |
| `initialize/integration_sync_runner.go` | 构造Sync Runner与Coordinator |
| `initialize/lifecycle.go` | 在HTTP和后台任务停止后关闭Redis、sql.DB等外部资源 |
| `cmd/container-entrypoint/main.go` | 容器内先Migration/Preflight再`exec`主进程，保留Signal语义 |

**典型链路**

`启动 -> InitializeApp -> Cron/Worker/SyncRunner -> HTTP Serve -> Signal -> 停止接收HTTP -> cancel Runtime -> 停Cron/Runner -> HTTP Shutdown -> 等Chunk清理 -> 关Redis/DB -> flush日志`

**核心对象、权限与扩展**

- Runner必须在Context取消后停止Claim，并在Shutdown timeout内收敛在途任务。
- DB/Redis只能在HTTP和后台任务停止后关闭；不能依赖进程退出让OS代为回收。
- 新后台任务必须进入统一生命周期和关闭等待，不允许裸goroutine永久运行。

### 6.15 Preflight、Backup与Release

**模块功能与用户能力**

发布链路在启动前验证配置、数据库、Ledger和关键Seed，提供外部数据库备份/恢复闭环，并由本地与CI共享
同一`release-check`门禁。该模块面向开发和运维，不向业务用户暴露页面。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `cmd/db-preflight/main.go` | 检查核心表、列、索引、约束、Seed、Ledger与TLS配置，输出脱敏问题 |
| `scripts/preflight-external.mjs` | 校验外部部署变量和目标环境安全条件 |
| `scripts/db-backup-external.mjs` | 生成/校验Manifest与Checksum，编排备份和恢复后Preflight |
| `scripts/smoke-readonly.mjs` | 对已部署服务执行不改数据的health、ready和只读Smoke |
| `scripts/check-tracked-secrets.mjs` | 扫描tracked文件中的Secret和本机敏感配置 |
| `Makefile` | 本地与CI共享的test、docs、scripts和release命令真值 |
| `.github/workflows/release.yml` | 提供PostgreSQL 16/Redis并调用共享发布门禁 |
| `docker-compose.yml`、`docker-compose.external.yml` | 本地完整环境和外部基础设施部署拓扑 |

**典型链路**

`clean build -> PostgreSQL/Redis ready -> migrate -> seed -> db-preflight -> startup -> smoke -> SIGTERM shutdown -> restart smoke`

**核心对象、权限与扩展**

- Preflight只检查，不修Schema；错误可说明缺失对象，但不能输出DSN、密码或Credential。
- Restore先验证Manifest和目标环境，再恢复并运行Preflight；备份文件不进入Git。
- CI和本地必须复用Makefile目标，不能维护两套不同门禁。

### 6.16 Frontend公共能力

**模块功能与用户能力**

前端公共层提供标准表格、统一查询、Query Scheme、动态表单、详情展示、按钮Capability、Runtime Metadata
和Theme。它统一机制，但允许Master-Detail、Tree、Diagnostic、Workbench和Report保留领域布局。

**核心入口与文件职责**

| 入口 | 职责 |
| --- | --- |
| `components/Table/StandardTableToolbar.vue` | 查询、业务动作和视图动作的窄职责布局，不调用业务API |
| `components/Table/TablePagination.vue` | 平台分页交互和稳定尺寸 |
| `composables/table-query-state.ts` | Quick、Expression、Order和Pagination状态；Quick与Advanced按AND组合 |
| `components/Query/AdvancedQuery.vue` | Simple/Advanced共享ExpressionGroup编辑器和业务/方案两种提交语义 |
| `components/Query/AdvancedQueryRuleRow.vue` | 单条字段、Operator、值和Binding编辑 |
| `composables/query-scope.ts` | 从菜单Runtime事实读取Scope身份并加载Scope Config |
| `composables/query-scheme-page.ts` | 页面Query Scheme初始化、默认、Dirty、保存和切换流程 |
| `components/QueryScheme/QuerySchemeControls.vue` | 标准页面Selector、Preset和Advanced入口组合 |
| `components/FormDialog/DynamicFormDialog.vue` | Dynamic Form整体生命周期、布局、联动、校验和提交 |
| `components/FormDialog/DynamicFieldControl.vue` | 单字段控件渲染，复用Metadata、Decimal、Dictionary和Relation真值 |
| `components/FormDialog/FormDialogShell.vue` | Dialog标题、导航、滚动Body、Preview和Footer结构 |
| `components/Detail/DetailFieldGrid.vue` | 详情字段语义化网格与空值展示 |
| `composables/page-buttons.ts` | 将当前菜单按钮投影为页面Capability |
| `utils/button-handlers.ts` | 平台按钮Action Handler与Hook Registry |
| `composables/runtime-table-metadata.ts` | 页面Runtime Metadata加载、错误和缓存状态 |
| `utils/field-metadata.ts` | 字段类型、控件、Operator和Relation能力的前端消费入口 |
| `stores/theme.ts`、`css/app.scss` | Theme状态和跨页面Semantic Token |

**典型链路**

`页面进入 -> 动态菜单/Capability -> Runtime Metadata -> useTableQueryState/useQuerySchemePage -> Toolbar/AdvancedQuery -> API Service -> 业务列表`

**核心对象、权限与扩展**

- 页面持有业务请求、分页、列和业务Dialog；公共Composable不直接变成第二套API或超级Store。
- Query只使用`QuickQuery + ExpressionGroup + Order`；页面不能维护不进入方案Payload的隐藏字段查询。
- Capability决定按钮显示，但后端仍必须授权；Metadata决定可呈现字段，但不授予数据访问权。
- 新公共组件需有跨页稳定语义；单页面布局留在页面，不创建BaseCrudPage或UniversalTable。
- 当前不提供移动端专属布局引擎，也不强制特殊工作台套用标准列表DOM。

## 7. 典型调用链

以下链路用于定位断点和判断职责归属，箭头表示主要调用方向，不表示所有错误、缓存和审计分支。

1. **登录**：`POST /login -> AuthHandler -> AuthApplicationService.Authenticate -> Credential Provider -> AuthLoginStateService -> AuthTokenService -> AuthAuditService`。
2. **用户列表查询**：`System User页面 -> sys-user API -> SysUserController -> SysUserService -> SysUserRepository -> PostgreSQL -> SysUserRes`。
3. **菜单权限加载**：`认证用户 -> SysMenuService.GetUserMenus -> UserRole/RoleMenu/RoleMenuButton Repository -> 动态路由 + usePageButtons`。
4. **Metadata表配置**：`Database Workbench -> SysTableController -> SysTableService -> Transaction -> Metadata Repository + PostgreSQL DDL -> MetadataRuntimeService.Invalidate`。
5. **低代码页面发布**：`发布动作 -> LowCodePublicationService -> SysMenu/SysMenuButton/Casbin事务投影 -> 动态菜单 -> Generalization页面`。
6. **Generalization列表查询**：`动态页面 -> GeneralizationController -> ResolvePublishedTableMenuId -> MetadataRuntimeService -> Query Capability -> Data Permission -> GeneralizationRepository`。
7. **Data Permission过滤**：`SubjectContextBuilder -> ResolverInput -> DataPermissionPolicyResolver -> Grant/Policy/Rule/Ownership/Dimension -> DataScopeResult -> Repository条件`。
8. **员工详情**：`Employee页面 -> org API -> OrgController -> OrgService.GetEmployeeDetail -> Employee/User/Assignment批量读取 -> Organization DTO`。
9. **HR同步**：`Transport成功 -> Sync Consumer -> Source DTO -> Normalizer -> Canonical Input -> OrganizationHRSyncService -> SourceKey/版本/幂等写入 -> Batch摘要`。
10. **普通文件上传**：`FileUpload -> multipart API -> FileUploadController -> FileUploadService -> Storage.Put -> FileRepository -> FileDetailRes`。
11. **分片上传与Merge**：`InitChunkUpload -> UploadChunk -> LocalChunkStaging -> MergeChunks -> Storage.Put -> File事务 -> 清理Session目录`。
12. **文件Preview**：`Preview请求 -> FileAccessController -> Actor/签名purpose校验 -> FileAccessService -> Storage.Get -> CSP/Content-Disposition响应`。
13. **外部接口执行**：`CreateExecution -> 冻结Interface/Input/Retry快照 -> Worker Claim -> ExecutionEngine -> CredentialProvider -> Transport -> Attempt -> Retry或终态`。
14. **同步任务执行**：`SyncRunner -> ScheduleDueTasks -> Batch -> CoordinateBatch -> Execution -> Worker -> Consumer -> Checkpoint/Batch汇总`。
15. **Query Scheme应用**：`Selector -> Available -> Resolve -> Visibility + Validator + BindingResolver -> ResolvedQuery -> table-query-state -> 页面列表API`。
16. **高级查询**：`AdvancedQuery -> ExpressionGroup -> normalize/sanitize -> 页面Query State -> 后端Query Capability校验 -> Query Builder -> PostgreSQL`。
17. **Report运行**：`Runtime页面 -> ReportController -> ReportAccess -> 发布版本 -> reportconfig -> Dataset Query/Data Permission -> Budget -> Preview/Export + ExecutionLog`。
18. **Migration启动与应用Shutdown**：`Container entrypoint -> migrate/ledger/preflight -> main.runRuntime`；停止时执行`Signal -> stop HTTP intake -> cancel Runner -> wait in-flight -> close Redis/DB -> flush logger`。

## 8. 我要改什么，先看哪里

| 需求 | 先看 | 同步检查 |
| --- | --- | --- |
| 新增固定页面 | 同领域`pages`、`api/services`、Router、菜单Seed | Page Button、Casbin、Runtime Metadata、测试 |
| 新增低代码表 | SysTable管理、LowCodeManual、Generalization | 字段/关系/索引、发布菜单、Data Permission |
| 修改查询 | `types/global.ts`、AdvancedQuery、table-query-state、query builder | querycapability、Metadata、Query Scheme normalize |
| 新增字段类型 | enum、Metadata validation、Migration | DTO精度、Dynamic Form、Operator、跨端Contract |
| 修改功能权限 | SysMenu/SysMenuButton、Casbin、usePageButtons | Router、Seed、API权限覆盖、只读账号 |
| 修改数据权限 | internal/datapermission、Resolver、Resource配置 | Subject、Ownership、列表/数量/详情一致性 |
| 修改组织 | Org Model/Service、Organization页面 | 管理/法人结构、Employee/User、Data Permission |
| 接外部系统 | Integration配置与Runtime合同 | Credential、TLS、Retry、Execution、Consumer |
| 修改Migration | `internal/migration/catalog.go`、`migrate/registry.go` | checksum、Fresh/Upgrade/并发、Preflight |
| 修改部署配置 | config、Compose、entrypoint、Operations Guide | TLS、Secret、health/readiness、shutdown |
| 修改Query Scheme | internal/queryscheme、Service、前端模块 | Scope、权限、Metadata、Dirty、Data Permission |
| 修改Report | Report Service、reportconfig、Report模块 | 发布隔离、SQL安全、权限、Deadline、执行日志 |

## 9. 修改后的验证

按风险选择测试，提交前至少执行相关单元测试。完整发布门禁：

```bash
SWEET_TEST_POSTGRES_DSN=postgresql://... make release-check
```

前端修改至少执行：

```bash
cd frontend
yarn test
yarn lint
yarn typecheck
yarn build
```

Migration、JSONB、partial unique、CHECK、SKIP LOCKED、Integration Runtime和Organization HR完整性
必须使用PostgreSQL 16测试，SQLite不能替代。发布与运维步骤见
[平台部署运维指南](../operations/PlatformOperationsGuide.md)。
