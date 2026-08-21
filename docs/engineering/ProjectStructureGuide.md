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

## 6. 领域入口

### 6.1 Auth

后端入口：

- `service/auth_application_service.go`：登录、刷新、登出等认证用例。
- `service/auth_token_service.go`：Token生命周期。
- `service/auth_login_state_service.go`：锁定、尝试和登录状态。
- `service/auth_credential_service.go`：Credential Provider边界。
- `internal/token`、`internal/security`：Token与密码安全实现。

前端入口：`pages/system/login`、`boot/axios.ts`和认证Store。修改认证时同时验证Refresh race、
Logout、Restricted Session、SMS、DingTalk和审计脱敏。

### 6.2 System：User、Role、Menu、Application

- Model：`model/sys_user.go`、`sys_role.go`、`sys_menu.go`及关联模型。
- Service：`sys_user_service.go`、`sys_role_service.go`、`sys_menu_service.go`、`application_service.go`。
- 权限：`middleware/casbin.go`、`initialize/router.go`和菜单/按钮Seed。
- 前端：`pages/system`、`api/services/sys-user.ts`、`sys-role.ts`、`sys-menu.ts`。

菜单、按钮、API Capability和Casbin必须一起变更。前端隐藏不替代后端授权。

### 6.3 Metadata

- 管理写入：`service/sys_table_service.go`。
- 低代码发布：`service/low_code_publication_service.go`。
- Runtime读取：`service/metadata_runtime_service.go`、`internal/metadata/runtime.go`。
- 类型与校验：`enum/enum.go`、`service/metadata_field_validation.go`、`internal/querycapability`。
- 前端：`pages/develop/database`、`utils/field-metadata.ts`、Runtime Metadata composable。

Storage Type、Logical Type、Display Format和Input Type各有职责。新增类型必须贯通Migration、
DTO、Runtime、Dynamic Form、Advanced Query和跨端Contract测试。

### 6.4 Generalization

- Service：`service/generalization_service.go`。
- Repository：`repository/generalization.go`及`repository/impl/generalization_impl.go`。
- 页面：`pages/develop/generalization/Index.vue`。
- API：`api/services/generalization.ts`。

Generalization必须解析已发布菜单Capability、Runtime Metadata、Query Capability和Data Permission。
不能用table code白名单或Broad API绕过页面权限。

### 6.5 Data Permission

- 核心合同：`internal/datapermission`。
- 运行Resolver：`service/data_permission_policy_resolver.go`。
- 配置Service：`service/data_*_config_service.go`。
- Subject：`service/subject_context_builder.go`。
- 前端：`pages/system/data-permission`、`api/services/data-permission-config.ts`。

业务查询最终形态是`Business Query AND Data Permission`。新增资源时先定义稳定Resource/Operation
和Ownership，再接Resolver与Repository Query。

### 6.6 Organization

- Model：`model/org.go`。
- Service：`service/org_service.go`、`organization_hr_sync_service.go`。
- 权限Provider：`org_permission_provider.go`、`org_permission_tree_provider.go`。
- HR Adapter：`internal/organization/hrsync`。
- 前端：`pages/organization`、`api/services/org.ts`。

OrgUnit是业务主体，StructureNode是树位置，Employee不等于User，Position不等于Role。HR Source
Adapter存在，但生产Consumer保持disabled；不要猜主任职、兼职或自动再入职。

### 6.7 File

- 上传：`service/file_upload_service.go`。
- 访问：`service/file_access_service.go`。
- Metadata：`service/file_metadata_service.go`。
- Storage：`internal/storage`。
- 前端：`components/FileUpload`和`api/services/file.ts`。

修改文件能力时同时检查用途签名、所有权、分片补偿、TTL清理、路径安全和Local/OSS差异。

### 6.8 Integration

- 配置Service：ExternalSystem、Credential、InterfaceDefinition、RetryPolicy。
- 执行：`integration_execution_service.go`。
- 同步：`integration_sync_service.go`、`integration_sync_coordinator.go`。
- Runtime：`internal/integration`中的Worker、Transport、Retry、Sync和Consumer Registry。
- 前端：`pages/integration`、`api/services/integration.ts`。

Consumer只处理成功响应的业务结果，不自行HTTP、Credential、Retry、Checkpoint或Execution状态推进。

### 6.9 Query Scheme

- 领域协议：`internal/queryscheme`。
- 管理：`service/query_scheme_service.go`。
- Runtime：`service/query_scheme_runtime.go`。
- Query Scope：Registry与`sys_menu.query_scope_code`。
- 前端：`modules/query-scheme`、`components/QueryScheme`、`pages/query-scheme`。

Scope身份只来自后端菜单事实。方案Resolve必须重新验证Metadata、Operator、Sort、Binding、可见性
和页面权限，不代理业务列表查询。

### 6.10 Report

- Model：`model/report.go`。
- Service：`service/report_service.go`、`report_access.go`、`report_execution_budget.go`。
- 配置解析：`internal/reportconfig`。
- 前端：`pages/report-v2/workbench`、`designer`、`runtime`和现有Report页面。
- 类型与Sheet算法：`modules/report`。

Report保持现有定义、发布版本、运行态、后端导出、执行日志、table/sql dataset和sheet/cell/binding
能力。不要在普通平台修改中扩展外部数据源、自由SQL、复杂图表大屏或重写设计器。

## 7. 我要改什么，先看哪里

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

## 8. 修改后的验证

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
