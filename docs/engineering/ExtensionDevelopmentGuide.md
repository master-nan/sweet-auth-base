# Sweet Platform 扩展开发指南

本文面向需要在 Sweet Platform 上新增业务模块、页面或外部系统接入的开发人员。平台分层和目录职责见[平台工程架构与目录手册](PlatformEngineeringGuide.md)，管理员配置方法见[平台管理员使用手册](../user-guide/PlatformAdministrationGuide.md)。本文只回答“如何扩展”。

## 1. 开始前先判断扩展类型

先确认需求属于哪一种，避免把配置问题写成代码：

| 需求 | 首选方式 | 通常是否改代码 |
| --- | --- | --- |
| 增加用户、角色、菜单授权 | 管理页面或 Seed | 否；内置初始能力才改 Seed |
| 为现有资源配置数据范围 | Data Permission 配置 | 否 |
| 为表登记显示、查询字段 | SysTable / SysTableField | 否 |
| 调用已支持协议的外部接口 | Integration 配置 | 通常否 |
| 把外部响应写入业务模型 | 注册 `SyncResultConsumer` | 是 |
| 新增持久化业务能力 | 标准业务模块流程 | 是 |
| 新认证来源 | Credential Provider 扩展 | 是；不得自建 Token 链 |

新增代码前先阅读同模块现有 Controller、Service、Repository、DTO 和测试。只有业务复杂度确实需要时才增加专用 Domain Service；不要机械建立 Manager、Facade、ApplicationService、DomainService 多层转发。

### 1.1 扩展场景速查

| 场景 | 主要文件/边界 | 推荐顺序 | 禁止事项 | 最低验证 |
| --- | --- | --- | --- | --- |
| 普通业务模块 | `model`、`migrate`、`repository`、`service`、`dto`、`controller`、`initialize`、frontend | 身份/状态 -> DB -> Service -> HTTP -> UI | 机械套层、Controller 直连 DB | 后端全量 + 相关 Race + 前端四项 |
| 数据库表 | `backend/model/`、`backend/migrate/` | Model -> 历史数据 -> 约束 -> Registry | 只靠 AutoMigrate | Migration + PostgreSQL 16 |
| CRUD | `BasicRepository`、Service、DTO、Controller | 先复用通用 CRUD，再补领域查询 | Model 直出、重复 Repository wrapper | Repository/Service/HTTP 测试 |
| 菜单/按钮 | Migration Seed、`router.go`、`usePageButtons` | Menu -> Button/API -> Casbin -> UI | 只隐藏前端按钮、硬编码角色 | 有权/无权账号与 API |
| Data Permission | Data Resource 配置、Resolver/Adapter | Metadata -> Resource/Operation -> Ownership -> Policy/Rule/Grant -> Query | 客户端提交范围、手拼权限 SQL | 范围、拒绝和 fail-closed 测试 |
| Runtime Metadata | SysTable/SysTableField、`RuntimeReader` | 登记表 -> 字段 -> 安全校验 -> Runtime 读取 | 跨模块直读 SysTable Repository | Runtime + 受保护字段测试 |
| Application Error | `backend/internal/errors/<domain>.go`、Service、`ResponseHandler` | 分类 -> 定义 -> Service 转换 -> HTTP 映射 | 返回 `err.Error()`、按技术层拆错误文件 | errors.Is/As + HTTP 安全响应 |
| 事务 | Service、`RunInTransaction` | 事务外 IO -> 短事务 -> 提交后 Cache/Audit | Controller/Repository 业务事务、嵌套 | commit/rollback/panic/race |
| Cache | 现有 Redis Client 和所属 Service | key/TTL -> 读策略 -> 提交后失效 | Cache 当真值、另造 Client | 并发、失效、Redis 故障 |
| Integration Consumer | `internal/integration` 契约、业务 Adapter、Wire Registry | metadata -> DTO -> Normalize -> Domain -> Result -> E2E | 自行 HTTP/Retry/Checkpoint | PostgreSQL Runner/Worker/TLS E2E |
| 外部接口 | ExternalSystem、Credential、RetryPolicy、InterfaceDefinition | 系统 -> 凭证 -> 重试 -> 接口版本 | 在业务代码读 Credential、绕 Transport | TLS Mock + Retry/安全校验 |
| Migration/Seed | `backend/migrate/registry.go` 及模块 migration 文件 | Schema -> Seed -> Casbin/Cache 修复 | 运行期逻辑进 Migration | 幂等 + PostgreSQL |
| 测试 | `*_test.go`、`backend/internal/test` | 单元 -> 集成 -> Race -> PostgreSQL | 用 Skip/Sleep 掩盖失败 | 按第 17 节矩阵 |
| 前端页面 | `src/api/services`、`src/pages`、Router、i18n | API -> 页面 -> Route/Menu -> Button | 页面拼 URL、自造基础 UI | test/lint/typecheck/build |

## 2. 普通业务模块开发流程

推荐顺序如下：

1. 明确业务身份、状态、权限和事务边界。
2. 在 `backend/model/` 定义持久化模型。
3. 在 `backend/migrate/` 增加幂等 Migration，必要时补 Seed。
4. 先复用 `repository.BasicRepository`，仅为领域查询增加 Repository 方法。
5. 在 `backend/service/` 编排校验、事务、审计和 DTO 投影。
6. 在 `backend/dto/` 定义白名单 Request/Response DTO。
7. 在 `backend/controller/` 或 `backend/api/` 适配 HTTP。
8. 在 `backend/initialize/router.go` 注册路由，在 Wire 中装配依赖。
9. 为菜单、按钮和 Casbin API 权限补 Seed。
10. 在 `frontend/src/api/services/` 增加 API 封装，再增加页面和受控路由。
11. 补 Service、Repository、Controller 和 PostgreSQL 专项测试。

简单只读能力可以复用现有 Service 或 Repository，不必为一个方法新增完整目录层。任何省略都不能让 Controller 直接访问数据库、让 Repository承担业务事务，或让 HTTP 返回 Model。

## 3. Model 与 Migration

### 3.1 Model

模型位于 `backend/model/`，表示持久化结构，不是 HTTP Response。新增模型时：

- 复用项目现有 `model.Basic` 审计字段和时间语义。
- 为稳定业务键、外键和状态字段明确定义长度、可空性和索引。
- 唯一性、`CHECK`、外键等数据库不变量必须在 PostgreSQL 中成立，不能只靠 Service 校验。
- Model Hook 只能从标准 `context.Context` 读取 `AuditSubject`，不得依赖 Gin、Service 或 Repository。
- 不把密码、Token、Credential、原始外部 Payload 等写进普通业务模型。

### 3.2 Migration

正式 Schema 演进位于 `backend/migrate/`，并注册到 `migrationSteps()`。每一步都必须能安全重跑；已应用版本、key、checksum 和时间写入 `schema_migration`，严格 db-preflight 会拒绝 ledger 缺失、不完整、未知版本或 checksum 漂移。

Migration 应做到：

1. 先处理历史数据，再增加更严格约束。
2. 使用确定名称的索引和约束，执行前检查现状。
3. PostgreSQL 专属的 partial unique、JSONB、`CHECK`、DDL 必须用 PostgreSQL 16 测试。
4. 将菜单、字典和系统初始配置放入 `platformSeedSteps()`，不要混入运行时业务逻辑。
5. 不以 `AutoMigrate` 代替正式 Migration；`autoMigrateCoreSchema` 只承担现有基础模型同步。

验证至少执行：

```bash
cd backend
go test ./migrate -count=1
SWEET_REQUIRE_POSTGRES_TESTS=true SWEET_TEST_POSTGRES_DSN='<postgres-dsn>' go test ./migrate -count=1
```

## 4. Repository

先查看 `backend/repository/basic.go`。普通创建、更新、删除、按字段查询、分页、预加载、上下文 DB 和乐观修订更新应复用 `BasicRepository`。

只有以下情况才增加专用 Repository 方法：

- 复合稳定业务键解析；
- `FOR UPDATE`、`SKIP LOCKED` 等锁语义；
- 树、有效期或复杂领域查询；
- 需要稳定排序或专门投影；
- 多租户/数据权限 Adapter 已确认的查询入口。

Repository 必须：

- 接收 `context.Context`；
- 只做数据访问，并允许接收 Service 传入的 `*gorm.DB`；
- 返回技术错误给 Service 转换；
- 不依赖 Gin，不创建 HTTP Error，不调用外部 HTTP，不自行开启跨业务事务。

## 5. Service、事务、Cache 与 Audit

Service 是业务编排和事务边界。典型写入流程是：事务外完成无副作用的输入解析或外部调用，短事务内锁定和修改本地事实，提交后再失效缓存。下面的 `Order` 是本指南的假想扩展示例，展示现有 `RunInTransaction`、`FindByIdForUpdate` 和错误构造方式，不代表仓库已有订单模块。

```go
func (s *OrderService) Update(ctx context.Context, input dto.OrderEditRequest) (*dto.OrderDetail, error) {
    if err := validateOrder(input); err != nil {
        return nil, apperrors.NewParameterError("参数错误")
    }

    var result *model.Order
    err := RunInTransaction(ctx, s.db, func(tx *gorm.DB) error {
        current, err := s.repo.FindByIdForUpdate(tx, input.ID)
        if err != nil {
            return translateOrderPersistenceError(err)
        }
        result, err = s.applyUpdate(ctx, tx, current, input)
        return err
    })
    if err != nil {
        return nil, err
    }

    s.cache.Delete(ctx, orderCacheKey(result.ID)) // 提交后失效
    return toOrderDetail(result), nil
}
```

规则：

- 普通业务短事务使用 `service.RunInTransaction`。
- Controller 不开启业务事务，Repository 不决定跨 Repository 事务。
- 不显式嵌套事务。
- HTTP、短信、对象存储和长时间文件 IO 必须在数据库事务外。
- `BasicRepository` 不提供通用事务入口。SysTable 等 Application Service 使用 `RunInTransaction`；Integration 锁定型原子 Repository 方法只在其明确命名的方法内部开启短事务。
- Cache 不是业务真值。写入提交成功后再失效；缓存失败不能伪装成数据库回滚。
- 写操作沿用 `internal/audit.AuditSubject` 与 `RequestMetadata`，保留 `request_id`、`trace_id`，不记录密码、Token、Credential、完整 Payload、物理路径或 HR 敏感信息。

## 6. DTO 与 HTTP Adapter

按用途选择 DTO：

| 类型 | 用途 |
| --- | --- |
| Request DTO | Bind 和白名单输入校验 |
| List DTO | 列表扫描所需的最小字段 |
| Detail DTO | 单对象详情 |
| Edit DTO | 可编辑字段 |
| Runtime DTO | 跨模块稳定运行时事实 |

Model 到 DTO 的显式转换优先放在 Service/Application 边界。默认不返回 secret、token、credential、物理 path、内部 SQL、软删除控制字段和无业务价值的审计内部 ID。

Controller/API 只做请求 Bind、HTTP 权限、调用 Service、`ctx.Error(err)` 和返回 DTO。禁止直接访问 Repository/GORM、解析数据库约束、开启事务、签 Token/HMAC、调用外部 HTTP 或返回 GORM Model。

路由统一在 `backend/initialize/router.go` 的正确权限组中注册。新的依赖通过 `backend/initialize/wire.go` Provider Set 装配，再更新生成代码；不要在 Controller 内临时构造基础设施对象。

## 7. Application Error 与 Reason Code

新增失败前先分类：

1. **Technical Error**：数据库、Redis、OS、JSON、外部 HTTP 等底层错误，只用于内部传播和日志。
2. **Application Error**：客户端可稳定识别的业务/应用失败，定义在 `backend/internal/errors/<domain>.go`。
3. **Reason Code**：执行、审计、重试或同步记录的诊断原因，留在所属模块，不等同 HTTP Error。

标准链路：

```text
Technical Error
  -> Service/Application 转换
  -> ApplicationError
  -> ctx.Error
  -> middleware.ResponseHandler
  -> 稳定 HTTP code 与安全消息
```

错误文件按稳定业务域组织，例如 `auth.go`、`integration.go`，不得新增 `adapter_error.go`、`repository_error.go`。优先复用 `internal/errors` 的稳定错误和构造器；不要向客户端返回 `err.Error()`。

```go
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, apperrors.ErrDataNotFound
}
return nil, apperrors.WrapDatabaseError(err)
```

Integration Retry Reason、Organization HR Sync Reason 等只用于状态与诊断，应留在各自模块中。

## 8. 菜单、按钮与 Casbin

新增管理页面的推荐顺序：

1. 为页面确定稳定的前端路由名、菜单 `name` 和 `option`。
2. 在 Migration Seed 中幂等创建菜单。
3. 为每个写操作或受控查看动作创建按钮权限码和 API path/method。
4. 用现有 `menuButtonWithAPI`、`apiPermissionWithAPI` 等 Seed 方式投影 Casbin policy。
5. 后端路由进入 Auth + `CasbinHandler` 保护组。
6. 前端页面通过 `usePageButtons(routeName)` 控制按钮展示。
7. 用有权限和无权限账号同时验证页面、按钮和直达 API。

前端隐藏按钮不是安全边界。不得硬编码管理员角色名，也不得只加菜单不加后端 API 权限覆盖。生产建议保持 `security.enforce_casbin_policy_coverage=true`。

## 9. 接入 Data Permission

以下以 `order` 表的 `owner_employee_id` 和 `org_unit_id` 为例。实际能力由现有 Resource、Operation、Ownership Field、Dimension、Policy、Rule、Grant、Resolver 和 Adapter 组成。

1. 先在 Metadata 中登记 `order` 及允许参与权限过滤的字段。
2. 创建 Resource。低代码通用 CRUD 使用 `low_code_table`；专用业务 Service 使用 `business_service`。
3. 为资源注册真实需要的 Operation，如 `query`、`detail`、`update`。
4. 为 `owner_employee_id` / `org_unit_id` 创建 Ownership Field。Metadata 表字段使用 `metadata_field`；专用 Adapter 才使用经过审核的 `registered_field`。
5. 选择现有 Dimension：本人通常对应 `employee`，部门范围对应 `management_org`。不要在代码里临时发明维度。
6. 配置 Policy、Rule，并通过 Grant 授给用户或角色。
7. 业务 Service 从可信 Context 构造 Subject，调用 Resolver 得到 `DataScopeResult`。
8. 通过现有 Adapter 把范围应用到查询，必须 fail closed；不要让客户端提交用户或组织范围。
9. 测试本人、部门、无 Grant、失效 Metadata、跨角色授权和直接 API 访问。

低代码查询应复用现有 `LowCodeDataPermissionRuntime` 和 Metadata Adapter。专用业务模块不得绕过 Resolver 直接拼权限 SQL，也不得修改 Data Permission 的 Resource、Resolver、Adapter 和 `DataScopeResult` 核心语义。

## 10. 接入 Runtime Metadata

希望被 Data Permission、动态列、快捷查询或未来高级查询发现的业务表，应登记 SysTable/SysTableField。跨模块稳定字段身份是：

```text
table_code + field_code
```

显示名称不是稳定身份，数据库 column 也不应直接成为跨模块公开契约。运行时消费者依赖 `internal/metadata.RuntimeReader` 或 `MetadataRuntimeService` 的 `TableMetadata`、`FieldMetadata`、`QueryFieldMetadata`，不得直接读取 SysTable Repository 或管理页面 DTO。

密码、Token、Credential、内部 SQL、系统维护字段等受保护字段不能配置为列表展示、快捷查询、高级查询或 Data Permission 字段。当前 Runtime Metadata 提供显示、排序、查询和Query Scheme校验所需的字段事实；页面仍可保留有明确业务语义的虚拟列和受控静态列。

### 10.1 接入 Query Scheme

标准列表需要保存查询方案时：

1. 在菜单Seed中设置稳定且唯一的`query_scope_code`，前端不复制Scope常量。
2. 在Backend Query Scope Registry为该Scope声明`table_code`和必要运行配置。
3. 确认Runtime Metadata提供允许查询、排序、字典和关系事实。
4. 页面复用`useTableQueryState`、`useQuerySchemePage`和`QuerySchemeControls`。
5. 页面私有字段筛选应转换为标准Expression；View Mode或Route Context需要明确边界。
6. 只有确有业务语义时配置Quick Preset，不根据字段名猜“我的”“异常”或主时间字段。
7. 验证Quick与Advanced按AND组合、默认优先级、Dirty、保存/另存为和Data Permission不扩大。

Master-Detail、Tree、Diagnostic、Config Workbench和Report应先判断是否适合Query Scheme，不能因为存在Metadata就机械接入。

## 11. 前端新增页面

最低开发规则：

- 使用 Vue 3、Quasar 和 TypeScript，先模仿同目录现有页面。
- API 放入 `frontend/src/api/services/`，不要在页面拼后端 URL。
- 页面负责视图编排，复用 Quasar 和已有公共组件。
- 路由进入受控 Router，路由名与菜单 Seed 对齐。
- 按钮使用 `usePageButtons`；后端仍必须执行 Casbin。
- 文案按现有 i18n 方式维护。
- 新列表先评估 Runtime Metadata，不要无条件复制一套静态字段定义。

新增或修改页面时遵循当前前端页面模式，并执行：

```bash
cd frontend
yarn test
yarn lint
yarn typecheck
yarn build
```

## 12. 新增 Integration 配置

先区分“配置接入”和“业务同步接入”。

- 普通外部 HTTP 调用：创建 ExternalSystem、Credential、可选 RetryPolicy、InterfaceDefinition，调用现有 Execution 能力，通常无需新 Consumer。
- 周期业务同步：在以上配置外创建 SyncTask，并实现注册式 Consumer。

配置顺序：

1. 创建稳定 `system_code` 的 ExternalSystem。
2. 创建只写不可读的 Credential；轮换产生受控新秘密，不从日志取回。
3. 需要技术重试时创建 RetryPolicy。
4. 创建 InterfaceDefinition 版本，配置相对路径、HTTP method、输入契约、超时、响应上限和 Credential/RetryPolicy 引用。
5. 已启用的技术契约不原地改语义；创建新版本。
6. 业务同步再创建版本化 SyncTask，配置 Cron、Timezone、Checkpoint、Lookback、Slice 和 Consumer。

Integration 当前只支持已实现的 Credential/Transport 契约。OAuth client 配置存在不等于 Runtime 已支持 OAuth 请求，不能按字段存在自行宣称可用。

## 13. 新增 Integration Consumer

业务同步 Consumer 的开发流程：

1. 定义固定 `consumer_code` 和 `version`。
2. 实现 `internal/integration.SyncResultConsumer`。
3. 从只读 `SyncConsumptionRequest` 读取 Body、逻辑窗口和摘要。
4. 解码最小 Source DTO，忽略且不持久化未知字段。
5. 经 Source Normalizer 转成 source-independent Canonical Input。
6. 调用业务 Domain/Application Service，使用短事务和稳定身份幂等。
7. 返回构造受控的 `SyncConsumptionResult`。
8. 配置 `SyncConsumerMetadata` 的状态、Content-Type、最大响应、最大耗时和 Checkpoint mode。
9. 在静态 Registry/Wire Provider 中服务端注册。
10. 完成单元测试和 PostgreSQL 16 的 Runner + Worker + TLS Mock E2E。

Consumer 禁止自行：

- 发 HTTP；
- 读取 Credential；
- Retry；
- Scheduler；
- 推进 Checkpoint 或 Execution 状态；
- 保存完整 Response Artifact、Header、Attempt 或 Payload。

HTTP/Transport 失败由 Integration Retry 处理；Consumer 业务失败返回 confirmed business failure，不重新进入 Retry。Checkpoint 只有整片业务成功才推进。

## 14. Source Adapter

Organization HR 是当前可参考的真实边界：

```text
HR Source DTO
  -> Source Normalizer
  -> Organization Canonical Input
  -> Organization Domain Service
```

`sendpost`、`userType`、NCID、源字段名和源日期格式属于 HR Source Adapter。业务稳定身份、任职周期、不删除历史等属于 Organization Domain。未来更换 HR 时替换 Source DTO/Normalizer，不让 Domain 依赖新源字段，也不重写 Integration Runtime。

## 15. 新增 Cache

只有真实读热点且允许短暂重建时才增加 Cache：

1. 明确数据库仍是业务真值。
2. 定义稳定 key、TTL 和空值策略。
3. 读取失败按业务安全要求 fail closed 或回源，不能默认为授权成功。
4. 数据库提交后再失效缓存。
5. 增加并发读取、失效和 Redis 故障测试。

不要为单个查询另造 Redis Client，也不要在数据库事务提交前刷新或删除缓存。

## 16. Migration、Seed、Wire 与 Router 清单

新增模块完成前逐项检查：

- Migration 已加入 `migrationSteps()`，可重复运行。
- Seed 已加入 `platformSeedSteps()`，菜单/按钮/Casbin 关系可修复重放。
- Repository、Service、Controller 已加入 `backend/initialize/wire.go` 对应 Provider Set。
- Wire 生成结果已更新且无手工依赖漂移。
- 路由已加入正确的 public、Auth 或 Auth+Casbin 组。
- Controller API path/method 与按钮权限 Seed 一致。
- 前端路由名、菜单名和 `usePageButtons` 参数一致。

## 17. 测试矩阵

| 改动 | 最低验证 |
| --- | --- |
| 普通后端 | `cd backend && go test ./... -count=1` |
| Repository/事务/并发 | 相关包测试 + `go test -race` |
| PostgreSQL 约束/DDL | `SWEET_REQUIRE_POSTGRES_TESTS=true` + PostgreSQL 16 |
| Integration/Organization HR | Runner/Worker/TLS Mock PostgreSQL E2E + race |
| 前端 | `yarn test`、`yarn lint`、`yarn typecheck`、`yarn build` |
| 运维 Node 脚本 | `make scripts-test` |
| 文档 | `make docs-check` |

测试夹具优先用 `backend/internal/test` 的 `OpenSQLite`、HTTP helper 和有界 `Eventually`。只有至少三处真实重复且属于同一领域时才新增 fixture helper；不要建立跨领域万能工厂。SKIP LOCKED、JSONB、partial unique、`CHECK`、DDL、Migration、Integration 和 Organization HR 不能静默降级到 SQLite，并应在 PostgreSQL 16 上执行。

测试文件按长期回归价值保留：安全、事务、状态机、权限、Migration、数据库约束和产品交互优先；重复 GORM CRUD、第三方库透传、文件拆分和源码字符串测试应删除。前端页面只测试页面自己的业务组合，公共组件行为由组件测试覆盖。跨端枚举、Operator 和 Permission Contract Guard 可以保留，但同一契约不得复制多份。

`make verify`只执行docs-check、后端测试以及前端lint/typecheck/build，适合本地快速验证。Pull Request运行`make ci-check`，使用真实PostgreSQL执行一次后端测试，并完成敏感信息、文档、Node脚本和全部前端检查。代码推送到main/master后运行`make release-check`，在日常检查之外增加后端`count=3`和全仓Race。两个GitHub Actions入口复用`.github/workflows/shared-checks.yml`准备PostgreSQL 16、Redis、Go和Node环境，具体测试顺序由Makefile维护。

## 18. 常见反例

```go
// 禁止：Controller 直接访问 Repository。
func (c *OrderController) Delete(ginCtx *gin.Context) {
    _ = c.repo.Delete(ginCtx, ginCtx.Param("id"))
}
```

```go
// 禁止：Service 依赖 Gin 或向客户端暴露底层错误。
func (s *OrderService) Create(ctx *gin.Context, input model.Order) error {
    return apperrors.NewBadRequestError(s.db.Create(&input).Error.Error())
}
```

```go
// 禁止：Repository 自行决定业务事务。
func (r *OrderRepository) SaveAll(ctx context.Context, orders []model.Order) error {
    return r.db.Transaction(func(tx *gorm.DB) error { /* ... */ return nil })
}
```

```go
// 禁止：Consumer 绕过 Runtime 自行下载响应。
func (c *Consumer) Consume(ctx context.Context, req SyncConsumptionRequest) error {
    response, _ := http.Get(c.url)
    _ = response
    return nil
}
```

```ts
// 禁止：页面自行拼接后端 URL。
await fetch('/sweet_admin/admin/order/query')
```

正确方向分别是 Controller 调 Service、Service 使用 `context.Context` 并转换 Application Error、Service 定义事务、Consumer 只处理请求 Body、前端通过 API service 调用。

## 19. 提交前检查

1. 新能力是否复用了既有 Auth、Data Permission、Metadata、File 或 Integration 边界。
2. 是否出现 Model 直出、Gin 穿透、Controller 事务、Repository HTTP、动态 `err.Error()`。
3. 权限是否同时覆盖菜单/按钮、Casbin 和必要的数据权限。
4. Migration/Seed 是否幂等，PostgreSQL 约束是否真实测试。
5. Cache 是否在提交后失效，Audit 是否脱敏。
6. 测试、Race、前端四项和 docs-check 是否按改动范围执行。
7. `git status` 是否只包含本次修改，是否误纳入秘密、上传文件、日志或构建产物。
