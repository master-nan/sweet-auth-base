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
| `docs/` | 当前系统的长期文档 | user-guide、engineering、operations | 原始敏感响应、临时实现记录 |
| `scripts/` | 部署前检查、只读 smoke、备份和文档检查 | 可独立运行的仓库工具 | 服务端运行时业务流程 |
| `Makefile` | 常用验证、构建、迁移和 Docker 命令入口 | 对已有脚本和工具的薄编排 | 业务规则 |
| `.nvmrc` | Node.js 唯一版本真值，当前为 22.23.0 | CI、本地前端工具链 | 第二份 Node 版本文件 |
| `docker-compose*.yml` | 本地完整环境和外部数据库/Redis连接环境 | PostgreSQL 16、Redis、后端、前端 | 生产秘密 |
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

受控例外包括：Integration Execution/Attempt 的锁定型原子 Repository 方法，以及 Migration/Seed。这些边界可以在明确命名的方法内部直接使用数据库事务；普通 Application Service（包括 SysTable metadata + DDL）统一使用 `RunInTransaction`。

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

Data Permission、Generalization、Query Scheme和Report通过Runtime Metadata读取，不直接依赖SysTable Repository。配置管理DTO、GORM Model和Runtime DTO不能混用。

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

## 19. Query Scheme

Query Scheme保存和复用标准列表的查询状态。`sys_menu.query_scope_code`是Query Scope唯一持久化身份；Backend Registry只为既有Scope提供`table_code`、快捷日期字段、业务Preset、虚拟排序和Binding白名单等运行配置，前端不维护第二份Scope映射。

Payload只保存`quick_query.keyword`、`expressions`、`order`和`bindings`。分页、每页条数、列显示、列宽、密度、Menu ID、Table Code和DataScopeResult不属于方案。Runtime Resolve按当前Scope权限、方案可见性、Payload Schema、Metadata、Operator、Sort和Binding重新校验，再返回标准Query；Scheme Service不代理业务数据查询。

PERSONAL由认证用户拥有；PUBLIC、ROLE和PAGE_DEFAULT写操作使用共享管理Capability。默认优先级是个人默认、页面默认、页面初始Query，ROLE/PUBLIC不自动参与。业务查询最终执行：

```text
Resolved Scheme Query
  AND
Data Permission
  -> Business Repository
```

方案不能保存或扩大Data Permission。动态Binding仅接受Query Scheme领域注册的强类型白名单，不执行SQL、脚本、模板或反射调用。

## 20. Report

Report当前保留`ReportDefinition`、发布版本和`ReportExecutionLog`，支持table/sql dataset、参数、sheet/cell/binding设计、发布版本、运行和CSV导出。设计预览读取草稿配置，正式运行和导出读取已发布快照，草稿修改不影响已发布版本。

权限由报表菜单、按钮、Casbin和服务端对象授权共同控制。table dataset沿用Metadata与Data Permission边界；SQL dataset执行只读安全守卫、超时和结果限制，但不会自动获得任意业务表的Data Permission过滤，管理员必须把它视为受控报表数据集而不是通用查询入口。

Report不接入Query Center，也不提供外部数据库数据源、自由跨表设计器、复杂图表大屏、打印分页、填报、定时调度、邮件订阅或多数据集复杂联动。新平台能力不得依赖Report内部布局或把Report Service作为通用查询服务。

## 21. 核心模块依赖图

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

## 22. Migration、Seed 与配置

[migrate/registry.go](../../backend/migrate/registry.go) 是 Migration/Seed 步骤注册入口：[migrate/main.go](../../backend/migrate/main.go) 的默认命令执行 schema migration，`seed` 子命令执行基础数据 Seed。

- Migration 负责 Schema 演进、数据库约束、幂等回填和必要数据库注释；
- Seed 负责初始配置、字典、管理员、菜单、按钮、角色、Casbin 投影和平台元数据；
- Migration/Seed 必须可重复执行并保留 PostgreSQL 专属测试；
- 在线业务流程、Scheduler、外部接口调用不能进入 Migration；
- 新 Model 不能只加 `AutoMigrate`，还要评估约束、索引、数据回填和 Seed 权限。

Migration registry 的已应用事实写入 `schema_migration(version, key, checksum, applied_at)`。checksum 绑定已发布步骤内容，严格 db-preflight 负责检查 ledger 是否缺失、不完整、含未知版本或发生 checksum 漂移；不得手改 ledger 来绕过迁移失败。外部部署必须显式关闭启动期 Migration/Seed，再通过受控 Make 目标分别执行。

配置结构位于 [config/config.go](../../backend/config/config.go)，`initialize.LoadConfig` 读取 YAML 与受控环境变量并执行生产安全校验。PostgreSQL 与 Redis 的 TLS 配置分别覆盖 mode/CA/client cert/key 和 enabled/server name/CA/client cert/key。秘密不得提交到配置示例或文档，external compose 也不得提供 production secret fallback。

## 23. Initialize、Wire 与启动

[main.go](../../backend/main.go) 的职责只有应用装配、Runner 生命周期、HTTP Server 和优雅退出。依赖装配位于：

- `initialize/wire.go`：Wire provider set、interface binding 和 `App` composition；
- `initialize/wire_gen.go`：生成文件，不手工编辑；
- `initialize/router.go`：路由和 Middleware 装配；
- `initialize/db.go`、`redis.go`、`casbin.go`：基础设施初始化；
- `initialize/integration_worker.go`、`integration_sync_runner.go`：运行器构建；
- `initialize/organization_hr_sync.go`：HR Consumer 的静态生产注册状态。

新增 Repository/Service/Controller 后，在 `wire.go` 增加 provider/bind，再运行 Wire 生成 `wire_gen.go`。业务 Service 不应在方法里自行 new 数据库、缓存、Transport 或另一个 Service。

## 24. Middleware

当前 HTTP 横切能力包括：

- `AuthHandler`：Access Token 和用户身份；
- `AuthHMACHandler`：开放 API 的应用身份；
- `CasbinHandler`：功能/API 权限；
- `LogHandler`/log context：request ID、trace ID、AccessLog 和安全元数据；
- `ResponseHandler`：统一成功响应和错误翻译；
- `CustomRecovery`：Panic 安全恢复；
- `CorsHandler`、NoRoute 和健康检查相关适配。

Middleware 适合跨请求的 HTTP 能力，不承载完整认证用例、数据权限策略解析或领域写入。它可以调用窄 Service 端口，但不能成为第二个 Service 层。

## 25. 前端架构

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

前端按标准列表、Master-Detail、Tree、Diagnostic和Report等页面类型复用适合的公共机制。新代码应先评估Metadata Runtime和公共组件，不能无条件复制静态列和大段页面样式，也不能用万能CRUD组件抹平业务布局。

## 26. 测试架构

常规 Repository/Service 测试使用 `backend/internal/test`：

- `OpenSQLite` / `OpenSQLiteWithConfig`：隔离 SQLite fixture 和必要 AutoMigrate；
- `ConfigureGinTestMode`：统一 Gin 测试全局状态；
- HTTP harness：请求和权限断言；
- `PostgreSQLDSN`：真实 PostgreSQL Gate；
- `MustCreate`：显式测试数据写入；
- `Eventually`：异步状态的有界轮询，替代各包重复的 `time.Sleep` 循环。

SQLite 用于普通业务和边界单测；以下语义必须使用 PostgreSQL：

- `SKIP LOCKED` 和真实并发领取；
- partial unique、CHECK、JSONB；
- DDL 和 Migration；
- Integration Runtime/Retry/Sync E2E；
- Organization HR 同步完整性；
- Metadata PostgreSQL DDL。

测试读取 `SWEET_TEST_POSTGRES_DSN`。设置 `SWEET_REQUIRE_POSTGRES_TESTS=true` 时，缺少 DSN 必须失败，不允许静默降级 SQLite。并发测试使用 channel、WaitGroup、barrier、事件 hook 或有界 `Eventually`，不用开放式 sleep 掩盖竞态。仅当测试本身需要模拟远端延迟时保留明确的 `time.Sleep`。生产相关包应定期执行 `go test -race ./... -count=1`。

前端使用 Vitest + Vue Test Utils。组件和页面测试应验证点击、emit、权限、请求、状态、表单或布局行为；不为某轮整改长期保存 class、import、组件名或源码字符串断言。只允许少量真正稳定的跨端 Contract Guard 和 Architecture Guard，且一个契约只保留一处真值测试。

运维 Node 脚本使用 `node:test`，通过 `make scripts-test` 执行；tracked 凭据扫描通过 `make secret-scan` 执行，只输出规则和位置，不回显命中值。提交前按改动范围执行 `yarn test`、`yarn lint`、`yarn typecheck`、`yarn build`；文档修改运行 `make docs-check`。浏览器验收用于主题、真实权限、Console、布局和跨页面流程，不由 source-string 页面测试替代。

当前 `make verify` 只组合 docs-check、`go test ./...` 和前端 lint/typecheck/build，不包含 Race、强制 PostgreSQL、前端 Vitest或 Node 运维脚本测试。它不能单独代表完整发布回归。`make release-check` 是唯一发布门禁，包含 secret scan、docs、scripts、强制 PostgreSQL、Race、Frontend Vitest、lint、typecheck 和 build；`.github/workflows/release.yml` 提供 PostgreSQL 16/Redis health service 并直接调用该目标，本地与 CI 不复制步骤。Node 版本只从根 `.nvmrc` 读取，`package.json` 仅允许 Node 22 major。

## 27. 关键文件职责与修改风险

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

## 28. 新代码放置决策

新增能力时按以下顺序判断：

1. 是 HTTP 请求/响应适配吗？放 `controller` 或 `api`。
2. 是一个业务用例、事务或跨 Repository 编排吗？放 `service`。
3. 是模块内部稳定算法、契约或安全能力吗？放该模块 `internal` 子包。
4. 是持久化查询吗？接口放 `repository`，实现放 `repository/impl`。
5. 是数据库结构吗？放 `model`，并同步 Migration/测试。
6. 是客户端输入/输出吗？放 Request/Response DTO；跨模块内部事实用所属 Runtime DTO。
7. 是前端后端调用吗？放 `frontend/src/api/services`。
8. 是页面编排、共享组件还是共享状态？分别放 pages、components/composables、stores。
9. 是长期使用、工程或运维资料吗？分别进入`docs/user-guide`、`docs/engineering`或`docs/operations`；临时证据不进入长期工程正文。

若一个 helper 只能由一个领域解释，就留在该领域；不要为了“公共”把它提前移入 `utils`。

## 29. 架构保护规则

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

## 30. 当前例外与限制

以下是已识别但未作为新代码范例的现状：

- `ReportService` 仍公开使用 `*gin.Context`，直到 Report Platform 重设计；新 Service 不得复制。
- `internal/utils/tools.go` 仍包含供 Controller 使用的 Gin 参数/Session helper；它们是 HTTP 兼容工具，不代表 Domain 可以依赖 Gin。
- SysTable 配置服务仍同时承担较多 metadata/DDL 用例；跨模块读取已经收口到 `MetadataRuntimeService`，新消费者不得直接依赖其 Repository。
- Report保留当前工作台、设计器、运行页、发布版本、后端导出和执行日志边界，不在一般页面中复用其专属布局。
- 前端动态列、i18n和页面专属样式按页面类型维护，不构成复制新实现的理由。
- File 分片暂存是单节点本地能力，多实例共享暂存尚未实现。
- Organization HR Adapter 能力已存在，但真实源生产 Gate 未关闭，Consumer 保持 disabled。
- Query Center V1 已通过 Runtime Scope、Query Scheme 和现有 Advanced Query 协议接入标准列表页；Report 仍不接 Query Center。
- `TableMetadata.QueryModel()` 是 Runtime Metadata 到现有动态 Query Engine 的唯一过渡桥，目前仅供 Generalization 与 Report 使用；禁止新增调用方。待 Report 专项稳定后，应让 Query Engine 直接消费 Runtime Metadata，并一次性删除该桥，不得再造第二个 adapter。

这些例外不改变本手册的新代码规则。需要偏离规则时，应明确记录当前约束，不能把局部例外扩散。

## 31. 文档关系

- 使用和配置：[平台管理员使用手册](../user-guide/PlatformAdministrationGuide.md)
- 日常系统操作：[系统使用手册](../user-guide/PlatformUserGuide.md)
- 项目结构与修改入口：[项目结构说明](ProjectStructureGuide.md)
- 扩展实施：[平台扩展开发指南](ExtensionDevelopmentGuide.md)
- 部署、环境和排错：[平台部署运维指南](../operations/PlatformOperationsGuide.md)

本文是当前长期工程架构真值。如果代码和本文不一致，应先核实当前实现，再更新代码或文档，不能让两套规则长期并存。

## 32. Notification Center

消息通知中心是当前已实现的平台级站内消息基础能力。本节记录数据模型、发送入口、Runtime API、前端生命周期和安全边界的当前真值。

### 32.1 实现边界

| 对象 | 当前事实 | 对通知中心的结论 |
| --- | --- | --- |
| `ToolbarItem.vue` | Header 铃铛读取 `notificationStore.unreadCount`，0 不显示，超过 99 显示 `99+` | 铃铛只承载入口和 Badge，列表交互由 `NotificationPopover` 承担 |
| `stores/user.ts` | 认证主体是 `SysUser`；Logout/换号通过 `session_generation` 隔离旧请求 | `stores/notification.ts` 复用该代次，停止轮询并清理旧用户状态 |
| `boot/axios.ts` | 每个请求携带 Token 与 Session Generation 快照，旧 Session 响应会转成 `StaleSessionResponseError` | 轮询结果必须复用该边界，禁止旧用户未读数写入新用户状态 |
| Router / TagView | `/admin/notifications` 是 `showTag=true` 的静态 Hidden Route | 从 Header 进入，不增加左侧菜单或 MenuButton Seed |
| Casbin | Notification Runtime API 是 authenticated common route | 不要求 MenuButton，Service 仍必须按当前 Audit Subject 隔离收件箱 |
| SMS | 管理短信模板、发送和发送日志，是外部通道 | 不复用其模板、Provider 或发送记录模型 |
| Integration | 管理外部系统、执行、重试、同步和 Consumer 状态机 | 可作为通知生产者，但 Notification 不进入 Integration 状态机 |
| Audit / AccessLog | 记录请求、安全主体和受控摘要 | Send 记录安全业务摘要；MarkRead 不产生重型审计事件 |
| User / Role / Organization | 登录主体是 `SysUser`；Employee 可选绑定 User，Position/Role 语义不同 | Recipient 只保存 `user_id`，不保存 Role、Employee、Position 或 OrgUnit |
| Query / Pagination | 已有显式分页 DTO、`TablePagination` 和 `StandardTableToolbar` | 页面复用分页与工具栏，不接 Query Center |
| Runtime Metadata | 服务低代码、动态表单、查询和跨模块字段事实 | 通知中心是显式平台 Runtime 页面，不由 Metadata 驱动 |
| Migration | Ledger Version 17 `notification_center_schema` 创建两张表、CHECK、FK 和索引 | PostgreSQL 是通知真值，不使用 AutoMigrate 代替正式升级 |

正式产品名为“消息通知中心”，英文领域名为 `Notification`。它是平台级站内消息基础设施：业务模块产生“某些用户需要被提醒”的事实，Notification 负责消息持久化、收件人分发、用户可见性、已读状态、未读数量、受控跳转和消息生命周期。

Notification 不是短信、邮件、聊天、待办流程引擎或系统日志。V1 只提供站内通知、未读数、消息列表、单条/全部已读、点击跳转、用户隔离和程序化发送。

### 32.2 核心对象与受控枚举

V1 使用 `Notification` 与 `NotificationRecipient` 两个对象，不能合并为一张表：

- `Notification` 保存一次发送共享的不可变消息事实。标题、内容、来源、级别和 Action 不应为每个用户复制。
- `NotificationRecipient` 保存每个用户自己的投递和阅读状态。同一消息对用户 A 已读、对用户 B 未读，属于两条独立收件状态。
- 两表分离后，单次发送一千名用户不会复制一千份长内容，主消息也不会因某个用户已读而影响其他用户。

受控 Category：

| 值 | 用途 |
| --- | --- |
| `SYSTEM` | 平台配置、维护和通用系统事实 |
| `BUSINESS` | 一般业务事实，如计划发布、成绩发布 |
| `TASK` | 明确需要用户处理的事项，但不承担工作流状态 |
| `REMINDER` | 截止时间、开始时间等提醒 |
| `SECURITY` | 登录、凭据或账号安全事实 |
| `INTEGRATION` | 外部系统执行、同步等运行事实 |

`SECURITY` 和 `INTEGRATION` 保留，是因为它们具有稳定的平台展示语义和安全处置差异；它们不控制权限、重试或优先级。学习和考试不得新增专属 Category，应通过 `source_module` 区分。

受控 Level 为 `INFO`、`SUCCESS`、`WARNING`、`ERROR`。Level 只控制图标、颜色和辅助文案，不控制投递顺序、权限、业务状态或调度。

### 32.3 数据库合同

V1 不使用 `expires_at`、`action_label`、附件、富文本、任意 JSON 业务快照或用户删除字段。消息长期保留；未来归档策略另行增加 Migration，不能在 V1 暗藏 Cleanup Worker。

当前 PostgreSQL 16 DDL 由 Migration Ledger Version 17 维护，核心合同如下：

```sql
CREATE TABLE notification (
    id               bigint PRIMARY KEY,
    category         varchar(24)  NOT NULL,
    level            varchar(16)  NOT NULL,
    title            varchar(160) NOT NULL,
    content          text         NOT NULL,
    source_module    varchar(64)  NOT NULL,
    source_type      varchar(64)  NOT NULL,
    source_id        varchar(128) NOT NULL DEFAULT '',
    action_menu_name varchar(32)  NOT NULL DEFAULT '',
    action_path      varchar(512) NOT NULL DEFAULT '',
    dedup_key        varchar(128),
    created_at       timestamptz  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_notification_category
        CHECK (category IN ('SYSTEM','BUSINESS','TASK','REMINDER','SECURITY','INTEGRATION')),
    CONSTRAINT chk_notification_level
        CHECK (level IN ('INFO','SUCCESS','WARNING','ERROR')),
    CONSTRAINT chk_notification_title
        CHECK (char_length(btrim(title)) BETWEEN 1 AND 160),
    CONSTRAINT chk_notification_content CHECK (
        char_length(content) BETWEEN 1 AND 4000 AND octet_length(content) <= 16384
    ),
    CONSTRAINT chk_notification_source CHECK (
        char_length(btrim(source_module)) BETWEEN 1 AND 64
        AND char_length(btrim(source_type)) BETWEEN 1 AND 64
    ),
    CONSTRAINT chk_notification_action
        CHECK (action_path = '' OR action_menu_name <> '')
);

CREATE UNIQUE INDEX ux_notification_source_dedup
    ON notification (source_module, dedup_key)
    WHERE dedup_key IS NOT NULL AND dedup_key <> '';

CREATE INDEX idx_notification_created
    ON notification (created_at DESC, id DESC);

CREATE TABLE notification_recipient (
    notification_id bigint      NOT NULL REFERENCES notification(id) ON DELETE RESTRICT,
    user_id          bigint      NOT NULL REFERENCES sys_user(id) ON DELETE RESTRICT,
    read_at          timestamptz,
    created_at       timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (notification_id, user_id)
);

CREATE INDEX idx_notification_recipient_user_created
    ON notification_recipient (user_id, created_at DESC, notification_id DESC);

CREATE INDEX idx_notification_recipient_user_unread
    ON notification_recipient (user_id, created_at DESC, notification_id DESC)
    WHERE read_at IS NULL;
```

字段归属：

| 对象 | 字段 | 语义 |
| --- | --- | --- |
| Notification | `id` | Snowflake bigint 消息身份 |
| Notification | `category` / `level` | 受控展示分类和级别 |
| Notification | `title` / `content` | 纯文本共享消息事实 |
| Notification | `source_module` / `source_type` / `source_id` | 产生消息的业务对象身份，不保存对象快照 |
| Notification | `action_menu_name` / `action_path` | 菜单 name 是稳定权限身份；Path 是该页面下的受控内部跳转信息 |
| Notification | `dedup_key` | 由来源模块定义的可选幂等身份 |
| Notification | `created_at` | 消息事实创建时间和默认排序时间 |
| Recipient | `notification_id` / `user_id` | 消息和最终认证主体的复合身份 |
| Recipient | `read_at` | 首次阅读时间；为空表示未读 |
| Recipient | `created_at` | 投递时间，用于当前用户列表和未读索引 |

两个模型都不嵌入完整 `model.Basic`：Notification 是不可变消息事实，Recipient 使用复合主键表达用户投递。V1 不建设 Sender 产品体系，也不暴露草稿、发布、撤回或禁用状态机。

### 32.4 来源、Action 与收件人合同

`source_module`、`source_type`、`source_id` 只保存稳定来源身份。例如 `learning / learning_plan / 123`。`source_module` 和 `source_type` 必填，`source_id` 对无法落到单个业务对象的系统消息可为空。禁止把完整业务对象、Credential、Token、考试答案或用户隐私放入自由 JSON。

Action 冻结为“受控内部路径 + 所属菜单身份”：

```go
menu_name = 稳定页面/权限身份
action_path = 该页面下的受控内部跳转信息
```

约束如下：

1. `ActionPath` 非空时必须以 `/admin/` 开头，最大 512 字符；拒绝 scheme、host、`//`、反斜线、控制字符、query 和 fragment。
2. 有 Action 时 `ActionMenuName` 必须是发送时已存在的活动菜单；Hidden Detail Route 使用其业务归属页面的菜单 name。`ActionPath` 可为空，表示只有消息内容而无跳转。
3. Recent/List 返回 Action 时批量计算 `allowed`，不逐条查询菜单。
4. 消息内容始终可查看；`allowed=false` 时前端明确提示“暂无目标页面访问权限”，不得导航。
5. `allowed=true` 只是用户体验预检，目标页面和 API 仍执行 Router、Menu、Casbin 和 Data Permission；通知不能成为授权凭证。
6. V1 不接受外部 URL。未来外链需求必须新增受控 Action 类型，不能放宽本字段。

V1 不保存 `action_label`。Popover 通过点击整行执行 Action，详情 Dialog 使用固定“前往相关页面”文案，避免业务模块产生不一致按钮语言。

收件主体固定为 `SysUser`。唯一发送命令接受单个或多个 `user_id`，但持久化模型不支持 Role、OrgUnit、Position、Employee 等多态 Subject。按 Role/Organization 发消息时，由产生消息的 Application Service 使用其既有授权/组织能力先解析并去重为用户集合，再调用 Notification。

Employee 未绑定 User 时不创建 Recipient，也绝不自动创建账号。上游业务必须把“未绑定账号”作为可观测的分发结果处理；若最终收件集合为空，`Send` 返回稳定校验错误。

### 32.5 发送、幂等与事务

V1 只暴露一个正式 Go Application 入口，不提供普通管理员 HTTP Send API：

```go
type NotificationCommand struct {
    Recipients     []int
    Category       NotificationCategory
    Level          NotificationLevel
    Title          string
    Content        string
    SourceModule   string
    SourceType     string
    SourceId       string
    ActionMenuName string
    ActionPath     string
    DedupKey       string
}
```

业务模块直接注入 `*NotificationService` 并调用 `Send(ctx, command)`。不建立 `NotificationSender`、Manager、Facade 或对管理员开放的 Send HTTP API；单用户发送也传一个元素的 `Recipients`。

输入上限：

- 收件人去重后 1 到 1000 个，所有 ID 必须对应有效 `SysUser`；任一无效时整条命令失败。
- Title 为 1 到 160 个 Unicode 字符。
- Content 为 1 到 4000 个 Unicode 字符且 UTF-8 字节数不超过 16 KiB。
- Source Module/Type 各 1 到 64 字符，Source ID 不超过 128 字符。
- Dedup Key 为空表示每次创建新消息；非空最大 128 字符。

幂等由 `ux_notification_source_dedup(source_module, dedup_key)` 部分唯一索引保证，不使用 Redis 锁。首次 Send 在短事务中创建主消息和所有 Recipient。重复 Send：

1. 读取唯一约束对应的既有消息；
2. 比较 Category、Level、Title、Content、Source 和 Action 等不可变事实；
3. 事实不一致时返回 Dedup Conflict，禁止静默覆盖；
4. 事实一致时复用原消息，仅以 `ON CONFLICT DO NOTHING` 补齐缺少的 Recipient；
5. 返回 `deduplicated`、`created_recipient_count` 和 `existing_recipient_count`，不重复投递已有用户。

主消息与 Recipient 写入必须在一个 `RunInTransaction` 短事务内完成。有效用户批量校验、主消息写入、Dedup 冲突处理和 Recipient 批量写入均使用同一 `context.Context`；不在事务中执行外部 IO。

跨模块业务事务采用：

```text
业务事务提交
  -> NotificationService.Send
  -> 发送失败记录安全摘要并由调用方决定重试
```

Notification 失败不能回滚已经成功的学习、考试、审批或集成业务事实。V1 不建设 Outbox，因此承认“业务提交后进程崩溃、通知尚未发送”的小窗口；需要可靠异步投递时再建设通用 Outbox，不能用跨模块大事务掩盖。

### 32.6 Runtime API 与 DTO

所有 API 经过 `AuthHandler`。它们是登录用户基础能力，不要求 MenuButton，并已纳入 `allowAuthenticatedIdentityRoute`；Service 只从可信 Audit Subject 取得用户 ID，Request/Path 中不允许出现可选择的 `user_id`。

| Method | Path | 用途 |
| --- | --- | --- |
| GET | `/admin/runtime/notifications/unread-count` | 当前用户未读总数 |
| GET | `/admin/runtime/notifications/recent?limit=8` | 当前用户最近通知，limit 默认 8，范围 1 到 10 |
| POST | `/admin/runtime/notifications/:id/read` | 幂等标记当前用户的一条消息已读 |
| POST | `/admin/runtime/notifications/read-all` | 标记当前用户全部有效未读消息已读 |
| POST | `/admin/runtime/notifications/query` | 当前用户完整消息分页查询 |
| GET | `/admin/runtime/notifications/:id` | 当前用户消息详情和完整纯文本内容 |

Unread Count 响应只返回真实整数 `unread_count`；`99+` 是前端展示规则。查询使用 `notification_recipient` 的未读 partial index，不拉取列表后计数。

Recent 默认按“未读优先，再按投递时间倒序”返回 8 条，上限 10；完整页面固定按投递时间倒序，不接受任意排序字段。

分页 Query Request：

```text
page: 1..n，默认 1
num: 1..50，默认 15
keyword: 标题/内容受控模糊查询，最大 100 字符
read_status: ALL | UNREAD | READ
category: 可选受控 Category
```

Summary DTO 只包含：

```text
id, title, content_preview, category, level,
read, read_at, created_at,
action { path, allowed }
```

`content_preview` 由后端按 Unicode 安全截取并规范换行，最大 120 字符。Detail DTO 增加完整 `content` 和受控 `source { module, type, id }`。任何 DTO 都不得返回 Recipient User ID、Dedup Key、Created By、内部状态字段或 GORM Model。

MarkRead 只执行：

```sql
UPDATE notification_recipient
SET read_at = CURRENT_TIMESTAMP
WHERE notification_id = $1 AND user_id = $2 AND read_at IS NULL;
```

重复调用不更新时间，首次 `read_at` 保持不变。目标消息不属于当前用户时统一返回 Not Found，不能区分“存在但属于别人”。MarkAllRead 只更新当前用户、`read_at IS NULL` 且主消息有效的记录；与单条 MarkRead 并发时依靠幂等条件收敛，不加 Redis 锁。

### 32.7 前端状态、轮询与会话隔离

`stores/notification.ts` 是 Header 通知状态的唯一所有者，保存 unread count、recent、loading/error 和轮询句柄。完整页的分页、筛选和 Dialog 状态仍属于页面。`ToolbarItem` 只渲染 Badge 和 Popover，不自行拼 API。

刷新规则：

1. 登录成功且 Header 挂载后立即读取未读数。
2. 页面可见时每 60 秒轮询一次未读数；`document.hidden=true` 时不发请求。
3. Tab 从后台恢复、用户打开通知 Popover 时立即刷新未读数和 Recent。
4. Logout、Token 更换或 `MainLayout` 卸载时调用 `reset`，停止定时器并清空用户通知态。
5. 每个请求沿用 Axios 的 Token/Session Generation 快照；`StaleSessionResponseError` 只丢弃，不改新 Session 状态。
6. 后台轮询失败保留最后一次成功计数，错误保存在 Store 中，不由定时任务弹出高频通知。

成功 MarkRead 后先本地幂等更新对应行和计数，再以后台刷新校正；失败则恢复原状态。MarkAllRead 成功后 unread count 立即置零，Recent 和当前页面记录同步为已读。

### 32.8 UI 详细合同

临时原型已按当前 Header、TagView、Semantic Token 和 Quasar 密度核验两个视图；原型不进入 tracked 仓库。实现不新增营销式页面标题或卡片堆叠。

**Header Popover**

- 使用 `q-menu`，从铃铛右下方展开，建议宽度 380px、最大高度 480px。
- Header 显示“通知”、未读摘要和“全部已读”；主体显示最近 8 条；Footer 只有“查看全部通知”。
- 每行包含未读点、Category/Level 图标、单行标题、两行纯文本摘要和相对时间。
- 未读行使用现有 primary soft surface，已读行使用普通 surface；不靠粗大彩色卡片表达状态。
- Badge 为 0 时不渲染，1 到 99 显示数字，大于 99 显示 `99+`。
- Loading 使用固定高度 Skeleton，Empty 显示 `notifications_none` 与“暂无通知”，Error 显示简短错误和 Retry，不关闭 Popover。
- 长标题 Ellipsis + Tooltip，长内容两行截断；完整内容进入详情 Dialog。

**消息中心页面**

- Hidden Route：`/admin/notifications`，Tag 标题“消息通知中心”，不建立左侧一级菜单。
- 使用 `BaseContent`、`StandardTableToolbar` 和 `TablePagination`，不接 Query Scheme、AdvancedQuery 或 Runtime Metadata。
- 顶部保持单行紧凑布局：全部/未读/已读分段控件、关键词、Category 和搜索；“全部已读”位于右侧动作区。
- 主体使用无额外卡片嵌套的 `q-list` 收件箱行，而不是普通 CRUD 表格。行展示状态、标题与摘要、Category、时间和打开图标。
- 默认时间倒序；点击行标记已读并打开详情 Dialog。Dialog 展示纯文本完整内容、Category、时间和固定 Action 按钮。
- Action 无权限时 Dialog 保留内容并显示“暂无相关页面访问权限”，按钮 Disabled。

**适配与主题**

- 1366 宽度下 Popover 不遮住头像或超出屏幕；页面 Toolbar 不换成两行，必要时先收缩日期和关键词宽度。
- 常用宽屏保持内容全宽工作台，不人为放大标题或增加装饰空白。
- 亮色使用 `--app-surface`、`--app-surface-muted`、`--app-border`、`--app-text-*`；深色复用现有 Token，不在页面新增 `body--dark` 色值补丁。
- Loading、Empty 和 Error 的容器尺寸稳定，避免 Popover 开合跳动。
- 所有图标按钮提供 `aria-label` 和 `q-tooltip`；Popover、Dialog 支持 Escape 关闭和明确关闭按钮。

### 32.9 权限、Data Permission、Audit 与内容安全

Notification 的读取所有权是 `recipient.user_id = 当前认证 user_id`，不经过 Data Permission Resolver。Data Permission 继续保护 Action 指向页面的数据；消息里出现某业务对象 ID 不代表用户可以读取该对象。

程序化 Send 记录结构化安全日志：Notification ID、Source Identity、Recipient Count、Dedup Outcome、Request/Trace ID；不记录 Content、完整 Recipient 列表、Token、Credential 或业务 Payload。普通 MarkRead/MarkAllRead 不写平台重审计日志，避免制造高频 AccessLog 之外的噪声。未来管理员公告能力再独立设计审计。

Title 和 Content 全程按纯文本处理。前端使用文本插值或 `white-space: pre-wrap`，禁止 `v-html`、Markdown HTML passthrough、Script、Style 和任意模板表达式。数据库与 Service 同时限制长度，避免巨型 Payload。

Notification 与外部通道的边界固定为：

```text
Notification = 消息事实 + 站内收件人 + 已读状态
SMS / Email / DingTalk = 外部 Delivery Channel
Integration = 外部系统配置、执行和状态机
```

V1 三者不联动，也不预建 `notification_delivery`、Channel、Provider 或 Template 表。

### 32.10 学习与考试复用示例

| 场景 | Category | Source | Action | Recipients | Dedup Key 示例 |
| --- | --- | --- | --- | --- | --- |
| 学习计划发布 | `BUSINESS` | `learning / learning_plan / {planId}` | `/admin/learning/plans/{planId}` | 计划分配对象最终绑定的用户 | `plan:{planId}:published:{version}` |
| 学习任务即将截止 | `REMINDER` | `learning / learning_task / {taskId}` | `/admin/learning/tasks/{taskId}` | 尚未完成任务的用户 | `task:{taskId}:due:24h` |
| 考试即将开始 | `REMINDER` | `exam / exam / {examId}` | `/admin/exam/exams/{examId}` | 已安排且可登录的考生用户 | `exam:{examId}:starts:{startAt}` |
| 考试即将截止 | `REMINDER` | `exam / exam / {examId}` | `/admin/exam/exams/{examId}` | 尚未交卷用户 | `exam:{examId}:closes:{endAt}` |
| 补考通知 | `TASK` | `exam / exam_retake / {retakeId}` | `/admin/exam/retakes/{retakeId}` | 获得补考资格的用户 | `retake:{retakeId}:assigned` |
| 成绩发布 | `BUSINESS` | `exam / exam_result / {resultId}` | `/admin/exam/results/{resultId}` | 成绩所属用户 | `result:{resultId}:published:{revision}` |

这些示例只验证通用合同；Notification 不新增 Learning/Exam 枚举，也不负责计算完成状态、考生资格或组织分配。

### 32.11 容量、性能与生命周期

- 单次 Command 最多 1000 名用户，Repository 使用批量查询和批量 Insert，不逐用户 N+1。
- 十万人广播超出 V1 同步 Fan-out 边界；未来采用异步分片/Outbox，不把单次上限简单调大。
- Unread Count 只做当前用户 partial index count；Recent 只读取当前用户最多 10 条，不 Join `sys_user`。
- Action Menu 权限对当前批次的不同 Menu Name 一次批量解析，不能每条消息查一次权限。
- 页面单页最大 50，默认 15；Keyword 最大 100 字符。若数据规模证明 `ILIKE` 成为瓶颈，再评审受控全文索引，不在 V1预建搜索引擎。
- 消息和 Recipient V1 长期保留。Operations 后续可按 90/180/365 天策略设计归档 Migration 或维护命令，但当前不运行自动删除 Worker。
- Redis 不保存未读真值，不新增 Notification Cache；PostgreSQL 是唯一读写真值。

### 32.12 文件与 Migration

后端实现文件：

```text
backend/model/notification.go
backend/dto/request/notification_req.go
backend/dto/response/notification_res.go
backend/repository/notification.go
backend/repository/impl/notification_impl.go
backend/service/notification_service.go
backend/controller/notification_controller.go
```

`NotificationCommand`、校验、发送和收件箱用例均位于 `notification_service.go`；没有 `internal/notification`、sender、reader、runtime、mapper、validator 或 manager 附加层。

前端实现文件：

```text
frontend/src/api/services/notification.ts
frontend/src/stores/notification.ts
frontend/src/components/Notification/NotificationPopover.vue
frontend/src/pages/notification/Index.vue
```

详情 Dialog 直接位于 Popover 和 `Index.vue`，不新增 Detail Route 或 MessageItem/Badge/Empty 子组件。类型与展示 Map 由 API Service 集中维护。

Migration Ledger Version 17，key 为 `notification_center_schema`，已将两张表加入 Managed Tables。该 Migration 创建表、CHECK、FK 和索引；V1 没有字典 Seed、可见菜单 Seed、MenuButton Seed 或管理员权限 Seed。

Wire 只增加 Repository、Service、Controller provider 和 interface binding。业务模块注入 `*NotificationService` 调用 `Send`，不直接操作 Repository。

### 32.13 测试与安全矩阵

长期测试矩阵：

| 层级 | 必须保护的行为 |
| --- | --- |
| Backend Unit | Category/Level、长度、Recipient 去重、Action Path、纯文本、Dedup Conflict |
| Repository PostgreSQL | 两表 FK、partial unique、批量 Recipient、用户索引查询、并发 Dedup、并发 MarkRead/MarkAllRead |
| Service | 用户隔离、无效用户整单失败、首次 read_at 不变、Action 权限批量解析、后台发送主体 |
| Controller | 未登录拒绝、伪造 user_id 无入口、跨用户 ID 返回 Not Found、分页/时间参数校验 |
| Frontend Store | 首次加载、60 秒可见轮询、暂停/恢复、Logout 停止、Session 切换丢弃旧响应、乐观已读回滚 |
| Frontend Component | Badge 0/99/99+、Loading/Empty/Error/Retry、长标题、全部已读、详情 Dialog、Action Disabled |
| Browser | Header Popover、完整页面、1366/宽屏、亮色/深色、键盘关闭、无权限 Action、Console 无异常 |

安全矩阵：

| 风险 | 防线 |
| --- | --- |
| IDOR / 跨用户读写 | User ID 只取 Audit Subject；Recipient 条件写入每条查询和更新；不存在与无权统一 Not Found |
| 伪造 `user_id` | Runtime Request/Path/Query 不提供该字段 |
| Open Redirect | Action 只允许规范化 `/admin/...` Path，拒绝 Origin、Query、Fragment、Scheme 和协议相对路径 |
| 无权限 Action | Action 绑定 Menu Name，服务端批量判定；目标 API 继续 Casbin/Data Permission |
| HTML/XSS | Content 纯文本，禁止 `v-html`，长度双重校验 |
| 巨型 Content/Recipient | 4000 字符/16 KiB、1000 Recipient 硬上限 |
| Dedup 竞态 | PostgreSQL partial unique + 同事务冲突读取 + 不可变字段比较 |
| 已归档消息 | V1 无归档入口，不存在半实现语义 |
| 旧 Session 返回 | Axios Session Generation + Notification Store Generation 检查和 Reset |
| 敏感信息泄漏 | DTO 不返回用户 ID/Dedup/审计字段；日志不记录 Content、Recipient 列表和业务 Payload |

### 32.14 明确不做

V1 明确不做：WebSocket、SSE、MQ、邮件、短信、钉钉、企业微信、移动 Push、模板中心、变量渲染、订阅偏好、广播后台、公告管理、审批、群组、多态 Recipient、附件、富文本、评论/回复、聊天、物理删除、自动过期、归档 Worker 和管理员全站消息管理。

当前实现边界：

- Action 选择受控内部 Path + Menu Identity，不使用自由外链、Route Params JSON 或开放 Action Registry。
- Role/Organization 不进入 Recipient；上游先解析为 User。
- V1 不保留 `expires_at`、`action_label`、`archived_at`，避免半成品生命周期。
- 保留 `SECURITY`、`INTEGRATION` Category，但它们只表达展示语义。
- 保留 Detail GET 和 Dialog，用于长纯文本及 Action 权限结果；不增加 Detail Route。
- Programmatic Send 不开放 HTTP Admin API。
- Popover 未读优先，完整页面时间倒序；两者排序语义有意不同。
- Notification 不接 Query Center、Runtime Metadata 或 Data Permission。
