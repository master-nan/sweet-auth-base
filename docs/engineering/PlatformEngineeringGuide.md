# Sweet Platform 工程架构与目录手册

- 状态：长期工程文档
- 读者：平台开发人员、项目维护人员、接手人员、架构维护者
- 依据：当前 `backend/`、`frontend/`、启动配置、迁移、测试和部署代码

本文说明 Sweet Platform 当前如何分层、各目录承担什么职责、模块如何依赖，以及新增代码应复用哪些稳定边界。具体扩展步骤见[平台扩展开发指南](ExtensionDevelopmentGuide.md)，管理员操作请阅读[平台管理员使用手册](../user-guide/PlatformAdministrationGuide.md)，部署和排错请阅读[平台部署运维指南](../operations/PlatformOperationsGuide.md)。历史设计、验收和冻结记录不属于本文正文。

## 1. 总体架构

Sweet Platform 是 Vue 3 + Quasar 前端、Go + Gin 后端、PostgreSQL 主数据库和 Redis 缓存组成的平台应用。后端通过 Wire 在启动时装配依赖；外部系统调用由 Integration Runtime 统一执行。

```text
Browser / API Client
        |
        v
Vue + Quasar
Pages -> API Services -> shared Axios
        |
        v
Gin Router and HTTP Middleware
Auth / Casbin / Audit Metadata / Error Translation
        |
        v
API or Controller             HTTP adapter
        |
        v
Service / Application         use-case orchestration and transaction boundary
        |
        +------------------------------+
        |                              |
        v                              v
internal domain capability       Repository interfaces
        |                              |
        v                              v
algorithm / contract / policy    Repository implementations
                                       |
                           +-----------+-----------+
                           |                       |
                           v                       v
                       PostgreSQL                Redis

External systems are reached through reviewed infrastructure capabilities,
especially Integration Runtime, not from controllers or repositories.
```

一个普通后台请求的实际路径是：

```text
frontend/src/api/services
  -> backend/initialize/router.go
  -> middleware (context, authentication, Casbin, audit, response)
  -> controller or api
  -> service
  -> internal capability and/or repository
  -> PostgreSQL / Redis
  -> response DTO
  -> middleware/error translation
```

层级不是为了增加转发文件。每一层只在拥有明确职责时存在：HTTP 适配在 Controller/API，业务编排在 Service，稳定算法和模块契约在 `internal`，持久化在 Repository。

## 2. 依赖原则

允许的主方向：

```text
controller/api -> service -> repository -> database
                     |
                     +-> internal domain/infrastructure contract

initialize -> all layers (composition root only)
middleware -> HTTP-facing service ports and context helpers
```

禁止反向依赖：

- Repository 不依赖 Service、Controller 或 Gin。
- Service 不依赖 Controller、HTTP ResponseWriter 或路由。
- `internal/integration` 不依赖 Organization 业务模型。
- Data Permission 不直接访问 Organization Repository 或 SysTable Repository。
- 业务模块不得绕过 Integration 自行实现正式外部 HTTP、重试或 Checkpoint。
- 前端页面不得绕过共享 API 层随意拼接后端地址。

组合根可以同时引用多个模块并完成绑定。例如 [wire.go](../../backend/initialize/wire.go) 将 Organization HR Consumer 注册为 Integration 的 `SyncConsumerRegistry`，这属于启动装配，不是 Integration 反向依赖 Organization。

## 3. 项目目录速查

### 3.1 仓库根目录

| 目录或文件 | 作用 | 可以依赖/包含 | 禁止放入 |
| --- | --- | --- | --- |
| `backend/` | Go 服务端 | HTTP、应用服务、领域能力、持久化、启动和迁移 | 前端页面、临时分析资料 |
| `frontend/` | Vue 3 + Quasar 客户端 | 页面、组件、API 封装、路由、状态 | 后端业务真值、数据库访问 |
| `docs/` | 受治理的长期文档与建设期证据 | user-guide、engineering、operations、`_construction` | 原始敏感响应、Task 临时输出 |
| `scripts/` | 部署前检查、只读 smoke、备份和文档检查 | 可独立运行的仓库工具 | 服务端运行时业务流程 |
| `Makefile` | 常用验证、构建、迁移和 Docker 命令入口 | 对已有脚本和工具的薄编排 | 业务规则 |
| `docker-compose*.yml` | 本地完整环境和外部数据库/Redis连接环境 | PostgreSQL 16、Redis、后端、前端 | 生产秘密 |
| `design/` | 早期设计遗留区 | 只作历史参考 | 新的正式文档；待 RC-001/DOC-FINAL 评审 |
| `work/` | 本地临时输出 | 日志、截图、探索材料 | 应提交的源码或正式文档 |

### 3.2 后端目录

| 目录 | 作用 | 可以依赖 | 禁止放入 |
| --- | --- | --- | --- |
| `backend/api/` | 开放 API、HMAC 客户端和认证入口的 HTTP 适配 | DTO、Service、HTTP helper | 重复认证编排、Repository、事务 |
| `backend/controller/` | 管理后台 HTTP 适配 | DTO、Service、Middleware helper | GORM、Repository、签名算法、外部 HTTP、业务事务 |
| `backend/service/` | Application Service、业务状态检查、事务、跨 Repository 编排和 DTO 投影 | Repository 接口、`internal` 契约、Model | Gin（Report 例外）、路由、HTTP 响应写入 |
| `backend/internal/` | 模块内部领域契约、算法和安全基础设施 | 标准库、必要的 Model/Repository port | 无归属的通用杂物、HTTP 页面逻辑 |
| `backend/repository/` | 数据访问接口和查询输入/结果 | `context.Context`、Model、受控查询 DTO、GORM 类型 | Gin、HTTP Error、Response DTO、外部调用、跨业务事务决策 |
| `backend/repository/impl/` | GORM 数据访问实现 | Repository、Model、数据库 helper | Controller/Service 流程、HTTP 概念 |
| `backend/repository/util/` | 元数据驱动查询等持久化辅助 | 查询 DTO、元数据模型 | 业务权限真值、HTTP 响应 |
| `backend/model/` | GORM 持久化结构、数据库枚举常量和基础 Hook | `internal/audit` 等极小基础能力 | Gin、Service、Repository、Response DTO |
| `backend/dto/request/` | HTTP 输入白名单 | 校验 tag、查询结构 | GORM 关系、秘密回显 |
| `backend/dto/response/` | HTTP 输出和列表/详情投影 | 安全业务字段 | 直接嵌入完整 Model、Credential、内部 SQL |
| `backend/middleware/` | HTTP 横切能力 | Gin、认证服务端口、审计和错误适配 | 完整业务用例、Repository 操作 |
| `backend/initialize/` | 配置加载、基础设施创建、Wire、路由和 Runner 装配 | 所有被装配模块 | 领域业务规则 |
| `backend/migrate/` | Schema 演进、幂等回填和 Seed | Model、GORM、受控配置 | 在线请求处理、长期业务调度 |
| `backend/config/` | 配置结构 | 纯配置类型 | 读取业务数据、运行流程 |
| `backend/enum/` | 共享稳定枚举 | 轻量类型和常量 | 模块内部 Reason Code、大型领域逻辑 |
| `backend/cmd/` | 容器入口、健康检查、预检、静态文件服务等独立程序 | 必要基础设施 | 主应用业务副本 |
| `backend/docs/` | `swag` 生成的 API 文档代码 | 生成结果 | 手工维护的架构文档 |

### 3.3 前端目录

| 目录 | 作用 | 可以依赖 | 禁止放入 |
| --- | --- | --- | --- |
| `frontend/src/pages/` | 页面级状态、交互和业务编排 | API services、stores、composables、components | 后端 URL、权限真值、可复用基础组件副本 |
| `frontend/src/components/` | 平台共享和布局组件 | Quasar、composables、utils | 单一页面的大量业务状态 |
| `frontend/src/api/services/` | 后端 API 封装和接口类型 | 共享 Axios | 页面 DOM、路由编排 |
| `frontend/src/boot/` | Axios、权限路由、i18n、布局等启动插件 | Quasar boot、stores | 具体业务页面逻辑 |
| `frontend/src/router/` | 常量路由、候选业务路由、受控动态路由映射 | 页面组件、权限菜单 | 任意服务端组件路径执行 |
| `frontend/src/stores/` | Pinia 跨页面状态 | API services、纯 utils | 只在一个组件使用的临时状态 |
| `frontend/src/composables/` | 可复用的交互与页面逻辑 | Vue、stores、components contracts | 第二套最终业务真值 |
| `frontend/src/modules/` | 模块专属的前端领域类型和算法 | 模块内部代码 | 全局无归属 helper；当前主要是 Report |
| `frontend/src/types/` | 通用 TypeScript 类型和枚举 | 纯类型 | API 请求执行、页面状态 |
| `frontend/src/utils/` | 纯辅助能力和元数据投影 | 纯类型、有限平台工具 | 新的全局状态、页面业务流程 |
| `frontend/src/layouts/` | 顶层应用壳 | Quasar、通用组件 | 领域页面逻辑 |
| `frontend/src/i18n/` | 中英文文案资源 | 纯文案 | 业务请求和状态 |
| `frontend/src/css/` | 全局主题和基础样式 | Quasar variables | 每个页面的局部视觉补丁 |

## 4. HTTP Adapter：Controller 与 API

`controller` 与 `api` 都是 HTTP Adapter。当前 `controller` 主要承载 `/admin` 管理接口，`api` 承载 AppToken/HMAC 保护的开放接口和用户认证接口；两者必须遵守相同边界。

它们负责：

1. 绑定 path、query、JSON 或 multipart 请求；
2. 执行 HTTP 层参数校验；
3. 从 `gin.Context` 取得 `ctx.Request.Context()`、已认证用户和页面权限事实；
4. 调用一个明确的 Application Service；
5. 设置审计动作；
6. 返回 Response DTO，或通过 `ctx.Error(err)` 交给统一中间件。

它们不得：

- 直接调用 Repository 或 GORM；
- 开启业务事务；
- 比较密码、签发 JWT、实现 HMAC；
- 直接读 Credential 或发正式外部 HTTP；
- 解析 PostgreSQL constraint、Redis 或 OS 错误；
- 返回 GORM Model。

[file_upload_controller.go](../../backend/controller/file_upload_controller.go) 展示 multipart 绑定、`FileAccessActor` 适配和 Service 调用；[integration_sync_controller.go](../../backend/controller/integration_sync_controller.go) 通过窄接口调用同步任务和批次服务。新代码应学习这些边界，而不是复制 Controller 中仍存在的历史样板。

文件 preview/download 和报表导出可以在 Controller 中设置 Header 并写入响应流，因为这属于 HTTP 适配；授权、文件查找、Content-Type 安全规则和报表生成仍必须由 Service 提供，Controller 不能自行实现。

## 5. Service / Application

Service 是业务用例和事务边界。它负责：

- 验证业务状态与稳定身份；
- 编排一个或多个 Repository；
- 调用已注册的跨模块端口；
- 把 Technical Error 转换为稳定 Application Error；
- 决定短事务范围；
- 将 Model 显式投影为 Response DTO 或内部 Runtime DTO；
- 在提交成功后失效所拥有的缓存。

Service 不负责 Gin、路由、HTTP status 或 JSON 写回。服务应按真实能力拆分，不为层次感制造 `Manager -> Facade -> DomainService` 转发链。

当前典型服务职责不同：

| 类型 | 当前职责 |
| --- | --- |
| `AuthApplicationService` | 统一认证用例，编排 Credential、账号状态、Token、Login State 和 Audit |
| `FileUploadService` | 普通上传、秒传、分片生命周期和存储补偿 |
| `FileAccessService` | 文件授权、签名 purpose、访问解析和安全流输出 |
| `MetadataRuntimeService` | 向跨模块消费者提供只读、脱离管理 DTO 的元数据投影 |
| `IntegrationExecutionService` | 管理 Execution 创建、查询和受控取消等应用能力 |
| `IntegrationSyncCoordinator` | 调度批次、创建 Slice、读取业务结果并推进 Checkpoint |

新增 Service 前先判断是否只是现有 Service 的一个清晰方法；只有职责和依赖确实独立时才新增服务。

## 6. `backend/internal` 的使用方式

Go 的 `internal` 同时提供编译期可见性限制，但它不是“暂时不知道放哪”的目录。当前子包按稳定能力组织：

| 子包 | 职责 |
| --- | --- |
| `audit` | `AuditSubject`、`RequestMetadata`、request/trace 关联信息 |
| `asynctask` | 不携带 Gin 的异步任务 Context 快照 |
| `cache` | Redis 适配、类型化缓存、Token/Login Attempt 等原子状态 |
| `database` | 主数据库句柄等基础设施边界 |
| `datapermission` | Resolver、Adapter、Subject、DataScopeResult 和 registered-field 契约 |
| `errors` | 跨 Service/HTTP 使用的稳定 Application Error 类型和定义 |
| `integration` | Runtime、Transport、Retry、Sync、Worker、Consumer 合同 |
| `metadata` | Runtime Metadata DTO、`RuntimeReader` 和只读 SQL 守卫 |
| `organization/hrsync` | 当前 HR Source DTO、Normalizer、SourceKey、Consumer 和任职解析边界 |
| `security` | Credential 加密和密码/敏感字段安全规则 |
| `storage` | Local/OSS Storage 与本地分片暂存实现 |
| `token` | JWT/HMAC 编解码器及 Claims 基础类型 |
| `reportconfig` | Report 配置解析和校验；仅供现有 Report 运行链使用 |
| `test` | SQLite、PostgreSQL Gate、Gin 和 HTTP 测试夹具 |
| `http`、`sms`、`dingtalk` | 受控基础设施适配或外部 Provider 辅助 |
| `utils` | 少量跨模块基础工具；新增内容必须证明不属于具体领域 |

领域 Reason Code、状态机原因和 Source Adapter 特有字段应留在所属模块。例如 Organization HR 的 `ReasonCode` 位于 `internal/organization/hrsync`，Integration Retry/Sync 原因位于 `internal/integration`，不能因为字符串表示失败就移入 `internal/errors`。

## 7. Repository 与 BasicRepository

Repository 接口位于 `backend/repository`，GORM 实现位于 `backend/repository/impl`。所有生产 Repository 使用 `context.Context`，不接受或保存 `*gin.Context`。

[basic.go](../../backend/repository/basic.go) 与 [basic_impl.go](../../backend/repository/impl/basic_impl.go) 提供当前共享能力：

- `WithContext` / `DBWithContext`；
- `Create`、`Update` 和按 ID/字段删除；
- `FindById`、`FindByField` 和列表读取；
- `FindByIdWithDB`、`FindByFieldWithDB` 等事务内读取；
- `FindByIdForUpdate` 行锁读取；
- `UpdateFields` 和 `UpdateFieldsByRevision`；
- `PaginateAndCountAsync`、`PaginateAndCountQuery`；
- `WithPreload`、`WithSelect`、`WithUnscoped`；
- `ExecuteTx` 这一基础事务能力。

普通 CRUD 和读取选项应复用 BasicRepository。以下场景值得增加领域 Repository 方法：

- 复合稳定业务键或幂等键；
- `SKIP LOCKED`、租约领取、原子状态变更；
- 树、有效期、权限过滤或引用完整性查询；
- 需要稳定排序或数据库专属语义的查询。

Repository 可以接收调用方已有的 `*gorm.DB`，但不自行决定跨 Repository 的业务事务。Integration Execution Repository 中围绕 Execution/Attempt 的原子操作是受控基础设施例外，不可复制成普通业务模式。

当前共享分页接口会接收 `request.Basic` 和 `model.SysTable` 作为查询描述，这是存量公共查询契约；新 Repository 方法不得继续引入 HTTP Response DTO 或 Controller 概念。

## 8. Model 与 DTO

Model 是持久化结构，不是 API 契约。[model/basic.go](../../backend/model/basic.go) 中的 `Basic` 提供主键、创建/修改/软删除审计字段、状态以及 GORM Hook。Hook 只从标准 `context.Context` 读取 `AuditSubject`。

Model 不得持有 Gin、Request、Service 或 Repository。数据库关联可以存在于 Model，但不能因此自动暴露到 HTTP。

DTO 分三类：

- Request DTO：请求白名单、校验和查询输入，位于 `dto/request`；
- Response DTO：列表、详情、编辑和访问结果，位于 `dto/response`；
- Runtime DTO：模块间只读稳定事实，位于拥有该契约的 `internal` 包，例如 `metadata.TableMetadata`。

禁止 Controller/API 直接返回 Model。列表、详情和编辑的字段目的不同，不创建一个包含所有字段的 Universal DTO。转换优先由 Service/Application 边界显式完成，避免反射映射导致 Model 新字段自动泄漏。

秘密、密码、Token、Credential 密文、物理路径、原始 Source ID、内部 SQL、软删除和内部权限关系默认不进入 Response DTO。业务主键、路由 ID 和明确需要的关联 ID 可以返回，是否内部并不只由字段名判断。

## 9. Context 与 Audit

Service、Repository 和领域能力统一使用 `context.Context`。HTTP Adapter 从 `ginCtx.Request.Context()` 进入；Middleware 将受控信息写入该标准 Context。

当前允许传播的请求事实：

- `audit.AuditSubject`：已确认用户 ID 和名称；
- `audit.RequestMetadata`：method、path、client IP、长度受限 User-Agent；
- `audit.CorrelationIDs`：`request_id`、`trace_id`；
- Data Permission 的可信 Subject 由服务端根据上述身份、角色和 Employee 绑定构造。

禁止把 `*gin.Context`、`http.Request`、`ResponseWriter`、原始密码、Token 或完整 Payload 放进 Context。异步任务使用 `internal/asynctask.Context` 的轻量快照，不捕获 Gin。

审计职责分开：

- HTTP AccessLog 记录请求结果和安全元数据；
- LoginLog/Auth Audit 记录认证入口、结果和稳定原因；
- 业务审计由 Service 在状态变更边界写入；
- IntegrationLog 记录每次技术 Attempt；
- OrgSyncBatch/Record 记录 Organization 业务处理事实。

日志和审计不得保存 Password、SMS code、Token、Credential、Authorization、完整请求/响应 Payload、物理文件路径或人员敏感信息。

## 10. 错误体系

错误流固定为：

```text
DB / Redis / OS / HTTP / parser Technical Error
                    |
                    v
Service or Application Boundary
                    |
                    v
stable ApplicationError
                    |
                    v
middleware/error_translation.go
                    |
                    v
safe HTTP status + error_code + error_message
```

### 10.1 四类概念

1. **Technical Error**：数据库、Redis、OS、第三方 HTTP、JSON 等底层失败。允许内部传播和结构化日志，不直接成为客户端消息。
2. **Application Error**：用户锁定、资源不存在、状态冲突等稳定业务/应用结果。共享定义位于 `backend/internal/errors`。
3. **HTTP Error**：Middleware 将 Application Error 的 `Kind` 映射为 HTTP status 和 `AdminError`；它不是领域真值。
4. **Reason Code**：Audit、Attempt、SyncRecord 或状态收敛的诊断分类，不等于 HTTP Error，也不一定需要构造 Go error。

`internal/errors` 按稳定业务域组织：`common.go`、`auth.go`、`file.go`、`metadata.go`、`organization.go`、`data_permission.go`、`integration.go`，基础类型在 `errors.go`。不得再按 adapter、resolver、runtime、repository、service 等技术层增加错误文件。

稳定错误标识使用 `ErrXxx`，稳定数值码使用 `ErrorCodeXxx`；模块 Reason Code 使用模块已有的 `lower_snake_case` 约定。新增错误步骤：先确认是否只是 Reason Code；若确需跨 Service/HTTP 的 Application Error，再在所属业务域文件增加唯一 code、safe message 和测试，最后由 Service 转换 Technical Error。Repository 不创建 HTTP Error，Controller 不解析底层错误。

## 11. 事务与一致性

Service 定义业务事务边界。普通短事务使用 [RunInTransaction](../../backend/service/transaction.go)：

```text
validate and external preparation (outside transaction)
  -> RunInTransaction
       -> lock/read
       -> domain writes
       -> audit/business record
  -> commit
  -> cache invalidation / external compensation
```

规则：

- Controller 不开启业务事务；
- Repository 不自行开启跨业务事务；
- 不显式嵌套事务或依赖伪嵌套语义；
- HTTP、Storage、SMS、DingTalk 等慢外部 IO 不占用数据库事务；
- Panic 保持 GORM rollback 和传播语义；
- Cache 失效发生在数据库成功提交之后；
- 数据库与 Casbin/Storage 不是同一事务资源，必须使用明确执行顺序和补偿，不能假装 DB rollback 会回滚外部状态。

受控例外包括：SysTable 的 metadata + DDL `ExecuteTx` 路径、Integration Execution/Attempt 原子持久化、Migration/Seed。例外的理由是数据库专属操作或基础设施原子性，不代表新业务可以随意调用 `ExecuteTx` 或直接 `db.Transaction`。

## 12. Cache

Redis 客户端和类型化缓存位于 `backend/internal/cache`，由 Wire 注入。现有缓存包括系统配置、用户/角色/菜单/字典、SysTable/SysTableField、Application，以及认证 Token blacklist、session、验证码和失败计数等。

缓存不是业务真值。拥有写操作的 Service 负责失效：例如 SysTable 配置提交后由 `MetadataRuntimeService.Invalidate` 清理表和字段缓存。认证使用 Redis 原子操作处理验证码消费、失败锁定、Refresh 消费和 session 状态；不能以普通 get/set 替代这些原子语义。

新增缓存前必须先证明读热点和一致性策略。不得在 Repository 中隐式加入缓存，也不得在事务提交前发布新缓存值。

## 13. Authentication

当前统一认证链是：

```text
Password / SMS / DingTalk Credential Provider
                    |
                    v
            ConfirmedIdentity
                    |
                    v
        AuthApplicationService
          |       |        |
          v       v        v
    account    Token    Login State
    policy    Service      + Audit
```

`AuthCredentialProvider` 只验证 Credential 并返回 `CredentialVerification`/`ConfirmedIdentity`，不签 Token、不写登录状态、不写 HTTP 响应。`AuthApplicationService` 统一执行账号存在性安全处理、enabled/locked、失败次数、密码状态、Token、Login State 和 Audit。后台密码、开放 API 密码、SMS、DingTalk、Refresh、Logout 均进入这条应用链。

`AuthTokenService` 复用 `internal/token` JWT 编解码和 Redis Token 状态。客户端不能构造 user ID、token type、session 或权限 Claims。认证失败对客户端不可枚举，详细原因只进入安全审计。

AppToken 是开放 API 调用方的应用身份，由 HMAC 入口校验，不是用户身份，也不进入用户 Login State。新增认证来源只能增加 Credential Provider/Identity 映射并接入 `AuthApplicationService`，不能在新的 API 中直接签 JWT。

## 14. Organization

Organization 的领域对象：

| 对象 | 语义 |
| --- | --- |
| `OrgLegalEntity` | 法律或核算主体 |
| `OrgUnit` | 可被业务稳定引用的组织主体 |
| `OrgStructure` | management 或 legal 结构定义 |
| `OrgStructureNode` | OrgUnit 在某棵结构树中的位置 |
| `OrgPosition` | 归属于 OrgUnit 的岗位 |
| `OrgEmployee` | 企业人员；可选绑定 SysUser |
| `OrgAssignment` | Employee 在 LegalEntity/OrgUnit/Position 下的任职事实 |

业务表保存 `org_unit_id` 表达组织归属，不保存 `structure_node_id`。同一 OrgUnit 可以出现在不同结构中，StructureNode 只是位置。Employee 与 SysUser 分离；同步 Employee 不创建账号，绑定由独立受控用例完成。Position 不等于 Role。

HR Adapter 位于 `internal/organization/hrsync`，固定边界是：

```text
HR Source DTO
  -> source Normalizer
  -> source-independent Canonical Sync Input
  -> OrganizationHRSyncService
  -> Organization model + OrgSyncBatch/Record
```

Source 特有字段、sendpost、ID 格式和日期解析不得进入通用 Organization Service。当前七类 HR Consumer 已静态注册但生产状态为 disabled，真实源 Gate 未关闭前不能被 SyncTask 引用。生产主任职、兼职、自动再入职和物理删除不属于当前能力。

## 15. Data Permission

功能权限与数据权限必须分离：Casbin/菜单按钮回答“能否调用”，Data Permission 回答“可访问哪些行”。

当前运行链：

```text
Resource + Operation
        |
SubjectContext (User + Roles + bound Employee + as-of date)
        |
Grant -> Policy -> Rules
        |
DimensionProvider (employee / legal entity / management org)
        |
Resolver
        |
DataScopeResult (not_applicable / all / none / filtered)
        |
registered-field or metadata Adapter
        |
business query
```

核心契约位于 `internal/datapermission`：`Resolver`、`Adapter`、`SubjectContext`、`DimensionValues`、`DataScopeResult`、`OwnershipFieldRegistry`。配置 Application Service 位于 `service/data_*_config_service.go`，运行时 Resolver 位于 `service/data_permission_policy_resolver.go`。

Subject 只能由 `SubjectContextBuilder` 从认证用户、有效角色和 Organization Employee 绑定构造，客户端不能覆盖。Organization 范围通过 `OrgPermissionProvider` 读取；registered field 通过 Metadata Runtime 的受控读取边界验证。Data Permission Core 使用 `context.Context`，不依赖 Gin。

业务模块接入时使用既有 Resource/Operation、Resolver 和 Adapter，不复制策略解析，不直接查询 Data Permission 表，也不把 `menu_id` 当作数据资源。

## 16. Metadata

Metadata 分成两个方向：

- `SysTableService`：配置写入、字段/关系/索引和现有 DDL 编排；
- `MetadataRuntimeService` + `internal/metadata.RuntimeReader`：跨模块只读事实。

Runtime DTO `TableMetadata`、`FieldMetadata`、`QueryFieldMetadata` 区分 Storage Type、Logical Type 和 UI Component，并剔除敏感或系统管理字段。跨模块字段稳定身份是 `table_code + field_code`；数字 ID 用于本地持久化和精确查询，显示名称不能作为引用身份。

Data Permission、Generalization 和 Report 已通过 Runtime Metadata 读取，不直接依赖 SysTable Repository。未来 Query Center 和前端动态列也应使用该边界。配置管理 DTO、GORM Model 和 Runtime DTO 不能混用。

查询、列表显示、标题和顺序等能力可以由 Metadata 提供；当前 `FieldMetadata` 尚未提供列宽，前端也尚未全量迁移动态列。新页面应先评估 Runtime Metadata，仍需静态列或宽度配置时应明确原因，不能声称全平台已经动态化。

## 17. File

File 模块的应用能力分为：

| 服务/组件 | 职责 |
| --- | --- |
| `FileUploadService` | 上传校验、普通上传、合法秒传、分片、合并和存储补偿 |
| `FileAccessService` | Actor 授权、preview/download 签名、签名解析和安全流响应 |
| `FileMetadataService` | 详情 DTO、metadata 生命周期和物理删除顺序 |
| `storage.Storage` | Local/OSS 保存、读取、删除和 URL 能力 |
| `LocalChunkStaging` | 当前节点本地分片暂存 |
| `FileAccessActor` | 仅携带用户 ID、超管事实等必要访问主体 |

Controller 不实现 HMAC、不拼物理路径、不直接访问 Storage。签名 Claims 强制包含 `purpose=preview|download`，两种用途不能互换，旧无 purpose 签名拒绝。物理路径、MD5 和 Storage 凭证不进入普通 Response DTO 或审计。

当前分片暂存在节点本地，多实例部署需要会话粘性或后续共享暂存能力；这是基础设施限制，不应由 Controller 自行解决。

## 18. Integration

Integration 分为五组能力：

1. **Configuration**：ExternalSystem、InterfaceDefinition、Credential、RetryPolicy；
2. **Runtime**：Execution、Input Snapshot、CredentialProvider、Transport、Attempt；
3. **Retry**：冻结策略快照、Retry Decision、`next_run_at`；
4. **Sync**：SyncTask、SyncBatch、Slice、Checkpoint、Sync Runner/Coordinator；
5. **Consumer**：成功 HTTP 响应后的注册式业务处理。

完整运行链：

```text
SyncTask
  -> IntegrationSyncRunner
  -> IntegrationSyncCoordinator
  -> IntegrationSyncBatch
  -> IntegrationExecution (one logical slice call)
  -> IntegrationWorkerRunner claims lease
  -> CredentialProvider
  -> HTTPTransportClient
  -> IntegrationLog (one immutable Attempt fact)
  -> registered SyncResultConsumer
  -> business result
  -> Execution converges
  -> Coordinator advances Batch and Checkpoint
```

`IntegrationExecution` 保存经过契约校验的非敏感输入快照和结果摘要；`IntegrationLog` 保存每次 Attempt 的状态、HTTP 摘要、结果哈希和 Retry 决策，不覆盖历史 Attempt。Credential 明文只能由 `CredentialProvider` 在调用时解密使用，不能进入 Controller、日志或 Consumer。

Worker 与 Sync Runner 由 `main.go` 启动、停止，是否运行取决于服务端配置；页面只读取状态，不能代替进程启动。Runner/Worker 的状态机、Retry、租约和 Checkpoint 是冻结能力，新业务只能使用公开契约，不能直接修改 Execution 状态或 `next_run_at`。

### 18.1 Consumer 扩展边界

`SyncResultConsumer` 在服务端静态注册，输入是 `SyncConsumptionRequest` 的受控窗口、摘要和调用栈内 Body，输出是 `SyncConsumptionResult` 的安全业务摘要。

Consumer 可以：解析响应、验证业务事实、调用业务 Domain Service、写本域 Batch/Record、返回稳定业务 Reason Code。

Consumer 不可以：

- 自行 HTTP 或读取 Credential；
- 创建 Scheduler、Retry 或 Checkpoint；
- 创建/推进 IntegrationExecution 或覆盖 Attempt；
- 修改 `next_run_at`；
- 保存完整 Response Artifact 或敏感 Payload；
- 把业务失败塞入 Integration Retry。

Organization HR 是该扩展点的当前实现样例。Integration 只持有 Consumer 契约，具体注册由 `initialize` 完成。

## 19. 核心模块依赖图

```text
Generalization ---------> Metadata Runtime <--------- Report
       |                        ^
       v                        |
Data Permission ---------------+
       |
       v
Organization Provider -> Organization Domain

Organization HR Adapter -> Integration Consumer Contract
Integration Runtime -X-> Organization models/repositories

Auth -> user repository + token/cache + audit
File -> repository + storage + signer/access policy
```

`-X->` 表示禁止依赖。跨模块依赖优先指向窄接口，例如 `metadata.RuntimeReader`、`datapermission.Resolver`、`OrgPermissionProvider`、`SyncResultConsumer`，而不是对方 Repository 或管理 DTO。

## 20. Migration、Seed 与配置

[migrate/registry.go](../../backend/migrate/registry.go) 是 Migration/Seed 步骤注册入口：[migrate/main.go](../../backend/migrate/main.go) 的默认命令执行 schema migration，`seed` 子命令执行基础数据 Seed。

- Migration 负责 Schema 演进、数据库约束、幂等回填和必要数据库注释；
- Seed 负责初始配置、字典、管理员、菜单、按钮、角色、Casbin 投影和平台元数据；
- Migration/Seed 必须可重复执行并保留 PostgreSQL 专属测试；
- 在线业务流程、Scheduler、外部接口调用不能进入 Migration；
- 新 Model 不能只加 `AutoMigrate`，还要评估约束、索引、数据回填和 Seed 权限。

配置结构位于 [config/config.go](../../backend/config/config.go)，`initialize.LoadConfig` 读取 YAML 与受控环境变量并执行生产安全校验。秘密不得提交到配置示例或文档。

## 21. Initialize、Wire 与启动

[main.go](../../backend/main.go) 的职责只有应用装配、Runner 生命周期、HTTP Server 和优雅退出。依赖装配位于：

- `initialize/wire.go`：Wire provider set、interface binding 和 `App` composition；
- `initialize/wire_gen.go`：生成文件，不手工编辑；
- `initialize/router.go`：路由和 Middleware 装配；
- `initialize/db.go`、`redis.go`、`casbin.go`：基础设施初始化；
- `initialize/integration_worker.go`、`integration_sync_runner.go`：运行器构建；
- `initialize/organization_hr_sync.go`：HR Consumer 的静态生产注册状态。

新增 Repository/Service/Controller 后，在 `wire.go` 增加 provider/bind，再运行 Wire 生成 `wire_gen.go`。业务 Service 不应在方法里自行 new 数据库、缓存、Transport 或另一个 Service。

## 22. Middleware

当前 HTTP 横切能力包括：

- `AuthHandler`：Access Token 和用户身份；
- `AuthHMACHandler`：开放 API 的应用身份；
- `CasbinHandler`：功能/API 权限；
- `LogHandler`/log context：request ID、trace ID、AccessLog 和安全元数据；
- `ResponseHandler`：统一成功响应和错误翻译；
- `CustomRecovery`：Panic 安全恢复；
- `CorsHandler`、NoRoute 和健康检查相关适配。

Middleware 适合跨请求的 HTTP 能力，不承载完整认证用例、数据权限策略解析或领域写入。它可以调用窄 Service 端口，但不能成为第二个 Service 层。

## 23. 前端架构

前端使用 Vue 3 Composition API、Quasar、TypeScript、Vue Router、Pinia、Axios、Vue I18n 和 Vitest。

启动链：

```text
quasar.config.ts
  -> boot plugins
       axios / permission / i18n / layout / bus
  -> App.vue
  -> router
  -> MainLayout
  -> authorized page routes
```

`boot/axios.ts` 是共享 HTTP 实例，统一 base URL、Access Token、loading、错误通知和未授权退出。普通页面通过 `api/services` 调用后端，不直接使用裸 axios/fetch。现有 RecordDetail、Generalization 等元数据运行时中的直接共享实例调用属于存量实现，不是新业务页面范例。

`boot/permission.ts` 登录后读取当前用户和后端权限菜单，再由 `router/utils/index.ts` 裁剪候选路由并映射受控低代码/Report 页面。服务端不能下发任意组件路径执行。按钮通过 `usePageButtons` 和菜单按钮事实控制；前端隐藏按钮不是安全边界，后端 Casbin 仍必须校验。

页面负责页面编排，跨页状态进入 Pinia，可复用交互进入 composable，通用 UI 进入 components。基础 UI 优先 Quasar，并复用 `BaseContent`、`AdvancedQuery`、`DynamicFormDialog`、`MasterDetailPage`、`TablePagination`、`OrganizationSelect` 和 `FileUpload` 等现有组件。

当前 Frontend Consistency 尚未完成：页面级 CSS 较多，部分页面过重，很多列表列仍静态定义，i18n 覆盖不完整，动态元数据只覆盖部分页面。这些是待治理现状，不是推荐架构。新代码应先评估 Metadata Runtime 和公共组件，不能继续无条件复制静态列和大段页面样式。

## 24. 测试架构

常规 Repository/Service 测试使用 `backend/internal/test`：

- `OpenSQLite` / `OpenSQLiteWithConfig`：隔离 SQLite fixture 和必要 AutoMigrate；
- `ConfigureGinTestMode`：统一 Gin 测试全局状态；
- HTTP harness：请求和权限断言；
- `PostgreSQLDSN`：真实 PostgreSQL Gate；
- `WithRollback`、`MustCreate`：测试数据辅助。

SQLite 用于普通业务和边界单测；以下语义必须使用 PostgreSQL：

- `SKIP LOCKED` 和真实并发领取；
- partial unique、CHECK、JSONB；
- DDL 和 Migration；
- Integration Runtime/Retry/Sync E2E；
- Organization HR 同步完整性；
- Metadata PostgreSQL DDL。

测试读取 `SWEET_TEST_POSTGRES_DSN`。设置 `SWEET_REQUIRE_POSTGRES_TESTS=true` 时，缺少 DSN 必须失败，不允许静默降级 SQLite。并发测试使用 channel、WaitGroup、barrier 或事件 hook，不用 sleep 掩盖竞态。生产相关包应定期执行 `go test -race ./... -count=1`。

前端使用 Vitest + Vue Test Utils；提交前按改动范围执行 `yarn test`、`yarn lint`、`yarn typecheck`、`yarn build`。文档修改运行 `make docs-check`。

当前 `make verify` 只组合 docs-check、`go test ./...` 和前端 lint/typecheck/build，不包含 Race、强制 PostgreSQL 或前端 Vitest。它不能单独代表完整发布回归；涉及并发、数据库专属语义或前端行为时必须补跑对应命令。

## 25. 关键文件职责与修改风险

| 文件/类型 | 职责 | 何时修改 | 主要风险 |
| --- | --- | --- | --- |
| [backend/main.go](../../backend/main.go) | 进程和 Runner 生命周期、HTTP Server | 新增进程级生命周期能力 | 关闭顺序、重复 Runner、资源泄漏 |
| [initialize/wire.go](../../backend/initialize/wire.go) | 依赖注入和跨模块接口绑定 | 新增/替换 provider | 依赖环、重复真值、生成文件失配 |
| [initialize/router.go](../../backend/initialize/router.go) | 路由与 Middleware 注册 | 新增 HTTP 能力 | 权限遗漏、旁路路由、API 兼容 |
| [repository/basic.go](../../backend/repository/basic.go) | Repository 共享接口 | 确有跨领域持久化共性 | 全仓影响、事务/并发语义变化 |
| [repository/impl/basic_impl.go](../../backend/repository/impl/basic_impl.go) | GORM 共享实现 | 修复基础查询或新增已评审共性 | 所有 Repository 回归、SQL 注入面 |
| [model/basic.go](../../backend/model/basic.go) | 基础字段、时间和 Audit Hook | 全平台持久化规则变化 | Migration、JSON、审计回归 |
| [service/transaction.go](../../backend/service/transaction.go) | 普通 Service 事务入口 | 事务语义统一变更 | rollback、panic、嵌套语义 |
| [internal/errors/errors.go](../../backend/internal/errors/errors.go) | ApplicationError 类型与分类 | 错误体系本身变化 | HTTP 映射和兼容码全局影响 |
| [middleware/error_translation.go](../../backend/middleware/error_translation.go) | Application Error 到 HTTP | 新 Kind 或协议映射 | 底层错误泄漏、状态码兼容 |
| [internal/audit](../../backend/internal/audit) | 标准 Context 中审计事实 | 新增真正跨层安全元数据 | 隐私泄漏、Context 滥用 |
| [internal/metadata/runtime.go](../../backend/internal/metadata/runtime.go) | 跨模块 Runtime Metadata 契约 | 新消费者需要稳定公共事实 | 破坏 Data Permission/Report/Low Code |
| [internal/integration/runtime_contract.go](../../backend/internal/integration/runtime_contract.go) | Integration 运行契约 | 正式版本化扩展 | 冻结契约、状态机和安全边界 |
| [internal/integration/sync_consumer_registry.go](../../backend/internal/integration/sync_consumer_registry.go) | Consumer 注册和调用合同 | 新业务 Consumer 能力 | 越权 HTTP/Retry/Checkpoint、Payload 泄漏 |
| [frontend/src/boot/axios.ts](../../frontend/src/boot/axios.ts) | 全局 HTTP 和错误行为 | API 协议或认证行为变化 | 全页面登录、通知、loading 回归 |
| [frontend/src/boot/permission.ts](../../frontend/src/boot/permission.ts) | 用户菜单和动态路由初始化 | 权限路由模型变化 | 无权限预加载、路由泄漏 |
| [frontend/src/router/utils/index.ts](../../frontend/src/router/utils/index.ts) | 受控动态页面映射 | 新的正式动态页面类型 | 任意组件加载、权限旁路 |
| [backend/migrate/registry.go](../../backend/migrate/registry.go) | Migration/Seed 注册顺序 | 新 Schema 或 Seed | 顺序依赖、非幂等、生产数据风险 |

## 26. 新代码放置决策

新增能力时按以下顺序判断：

1. 是 HTTP 请求/响应适配吗？放 `controller` 或 `api`。
2. 是一个业务用例、事务或跨 Repository 编排吗？放 `service`。
3. 是模块内部稳定算法、契约或安全能力吗？放该模块 `internal` 子包。
4. 是持久化查询吗？接口放 `repository`，实现放 `repository/impl`。
5. 是数据库结构吗？放 `model`，并同步 Migration/测试。
6. 是客户端输入/输出吗？放 Request/Response DTO；跨模块内部事实用所属 Runtime DTO。
7. 是前端后端调用吗？放 `frontend/src/api/services`。
8. 是页面编排、共享组件还是共享状态？分别放 pages、components/composables、stores。
9. 是长期使用、工程或运维资料吗？按 DocumentationStandard 进入对应 docs 目录；Task 证据不进入长期工程正文。

若一个 helper 只能由一个领域解释，就留在该领域；不要为了“公共”把它提前移入 `utils`。

## 27. 架构保护规则

1. Controller/API 只做 HTTP 适配，不直接 Repository、GORM、事务、签名或外部 HTTP。
2. Service 定义业务事务和 Technical Error 转换；外部 IO 不占数据库事务。
3. Repository 只做数据访问，使用 `context.Context`，不依赖 Gin 或创建 HTTP Error。
4. Model 不是 DTO；生产 HTTP 不直接返回 GORM Model。
5. `internal/errors` 只保存稳定 Application Error；Reason Code 留在所属模块。
6. Context 只传播受控身份和请求元数据，不传播 Gin/Request/ResponseWriter/秘密。
7. Cache 不是业务真值，提交成功后由拥有写入的 Service 失效。
8. Runtime Metadata 是跨模块字段事实，稳定引用使用 `table_code + field_code`。
9. 业务组织归属引用 OrgUnit；StructureNode 仅表示树位置；Employee 不等于 User，Position 不等于 Role。
10. Data Permission 通过 Resolver/Adapter/Provider 接入，不直接读 Organization 或 Metadata Repository。
11. Integration Consumer 不自行 HTTP、Credential、Retry、Checkpoint 或 Execution 状态推进。
12. 前端 API 使用共享 Axios/API service，权限按钮来自菜单事实，基础 UI 优先 Quasar 和平台公共组件。

## 28. 当前例外与延期边界

以下是已识别但未作为新代码范例的现状：

- `ReportService` 仍公开使用 `*gin.Context`，直到 Report Platform 重设计；新 Service 不得复制。
- `internal/utils/tools.go` 仍包含供 Controller 使用的 Gin 参数/Session helper；它们是 HTTP 兼容工具，不代表 Domain 可以依赖 Gin。
- SysTable 配置服务仍同时承担较多 metadata/DDL 用例；跨模块读取已经收口到 `MetadataRuntimeService`，新消费者不得直接依赖其 Repository。
- 当前 Migration 没有独立版本台账，依赖注册顺序和每个步骤的幂等性；迁移框架演进应作为独立基础设施任务处理。
- Report 目录存在 V1、V2 和 prototype 形态；其产品化边界按 Report 专项推进，不在一般页面中复用原型代码。
- 前端动态列、i18n、页面 CSS 和组件复用尚未全平台一致，留给 Frontend Consistency。
- File 分片暂存是单节点本地能力，多实例共享暂存尚未实现。
- Organization HR Adapter 能力已存在，但真实源生产 Gate 未关闭，Consumer 保持 disabled。
- Query Center 尚未实现；Metadata 中的 advanced-query 标记不代表完整 Query Center 产品能力。

这些例外不改变本手册的新代码规则。需要偏离规则时，应先形成专项设计和评审，而不是把历史例外扩散。

## 29. 文档关系

- 使用和配置：[平台管理员使用手册](../user-guide/PlatformAdministrationGuide.md)
- 扩展实施：[平台扩展开发指南](ExtensionDevelopmentGuide.md)
- 部署、环境和排错：[平台部署运维指南](../operations/PlatformOperationsGuide.md)
- 文档归类：[DocumentationStandard](../DocumentationStandard.md)
- 建设期设计、验收和冻结证据：`docs/_construction/`，仅供追溯，不作为当前代码入口手册

本文是当前长期工程架构真值。模块的具体扩展步骤由 PE-003 承载；如果代码和本文不一致，应先核实当前实现，再更新代码或文档，不能让两套规则长期并存。
