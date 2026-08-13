# Sweet Platform 测试基础设施规范

## 文档信息

| 项目 | 内容 |
| --- | --- |
| 文档目的 | 统一后端测试分类、数据库使用、Gin 初始化、隔离和 CI 可见性规则 |
| 适用范围 | `backend/` 中的单元、Controller、Service、Repository、Migration、PostgreSQL 专项和 race 测试 |
| 审计日期 | 2026-08-04 |
| 审计基线 | `93f1f8e9e04f4f0bda28cc45ef6299a58f1d691c` |
| 通用测试工具 | `backend/internal/test` |

本规范采用“新增测试统一使用基础设施、存量测试按模块渐进迁移”的方式，不要求一次性重写现有夹具或改变专项测试语义。

## 1. 当前测试基础设施审计

### 1.1 审计范围

本次审计覆盖 `backend/` 下 126 个测试文件，检查 SQLite 初始化、AutoMigrate、Gin 全局模式、PostgreSQL DSN 门控、重复测试夹具和当前 CI 配置。

### 1.2 审计统计

| 检查项 | 调用数 | 文件数 | 当前结论 |
| --- | ---: | ---: | --- |
| 直接调用 GORM 打开 SQLite | 35 | 18 | 存量测试较多，后续渐进迁移 |
| 直接使用 SQLite `:memory:` | 24 | 14 | 易受连接池和数据库共享语义影响 |
| 调用 `AutoMigrate` | 45 | 20 | 需要区分夹具迁移和 Migration 行为测试 |
| 使用 `testutil.OpenSQLite` | 45 | 20 | 现有 Helper 已被新模块广泛使用 |
| `gin.SetMode` | 21 | 17 | 整改前分散在具体测试和辅助函数中 |
| 包级 `TestMain` | 0 | 0 | 整改前未统一 Gin 初始化 |
| PostgreSQL DSN 门控测试 | 4 | 4 | 缺少 DSN 时直接跳过，CI 可见性不足 |

同时直接打开 SQLite 并执行 AutoMigrate 的测试文件有 14 个。一个文件内重复 AutoMigrate 超过一次的文件有 9 个，主要集中在：

- `migrate/main_test.go`：9 处。
- `repository/impl/basic_impl_test.go`：7 处。
- `service/sys_table_service_test.go`：5 处。
- `service/application_service_test.go`：3 处。
- Organization、权限 Seed、权限投影、Basic 审计 Context 和菜单数据范围测试：各 2 处。

重复次数不直接等于可以合并。Migration 测试需要刻意构造不同初始 Schema；DryRun、连接池、共享内存和事务错误测试也可能需要专用数据库配置。

### 1.3 PostgreSQL 专项测试

当前由 `SWEET_TEST_POSTGRES_DSN` 控制的专项测试共 4 个：

1. Data Permission Migration 与 PostgreSQL 约束。
2. Organization PostgreSQL 部分唯一约束。
3. Organization 数据库注释与 Schema 不变性。
4. 旧数据权限清理 Migration。

它们验证 SQLite 无法可靠覆盖的 PostgreSQL 行为。缺少 DSN 时本地可以跳过，但 PostgreSQL CI 作业不得静默跳过。

### 1.4 当前 CI 状态

当前后端 CI 仅执行 `go test ./...`，尚未配置：

- 独立 PostgreSQL Service 和专项测试作业。
- `SWEET_REQUIRE_POSTGRES_TESTS=1` 强制门控。
- 全仓或重点包 race 作业。

这些属于 CI 配置 Backlog。本 Task 先提供统一测试 Helper 和强制门控语义，不在本次轻量整改中重写工作流。

## 2. 测试分类

### 2.1 单元测试

单元测试验证纯函数、领域对象、校验规则、DTO 映射和不依赖数据库的 Service 逻辑。

要求：

- 不启动真实网络监听器。
- 不依赖开发机数据库、Redis 或外部服务。
- 使用显式输入和确定性时间、ID 或随机源。
- 默认允许并行，但涉及全局状态时不得调用 `t.Parallel()`。

### 2.2 Repository 测试

Repository 测试验证查询、分页、排序、事务错误传播和受控筛选。

- 与数据库方言无关的行为优先使用 `testutil.OpenSQLite`。
- PostgreSQL 特有约束、锁、JSONB、部分索引和 SQL 语义必须使用真实 PostgreSQL。
- Repository 测试应验证技术错误传播，不替代 Service 领域规则测试。

### 2.3 Service 测试

Service 测试验证业务规则、事务边界、错误转换和 Repository 协作。

- 简单持久化夹具可以使用 `testutil.OpenSQLite`。
- 外部服务使用接口替身或 `testutil.NewHTTPServer`。
- 事务测试必须覆盖成功提交、错误回滚和审计失败。
- 不得依赖测试执行顺序或其他测试留下的数据。

### 2.4 Controller 与 Middleware 测试

Controller 和 Middleware 测试验证请求解析、统一响应、权限入口和安全错误映射。

- 使用 `httptest` 或 `testutil.PerformRequest`。
- Gin 模式由包级 `TestMain` 设置。
- 不在每个测试函数或路由工厂重复调用 `gin.SetMode`。
- 全局 Logger、错误处理器和权限执行器必须在测试结束时恢复。

### 2.5 Migration 测试

Migration 测试验证：

- 首次执行结果。
- 重复执行幂等。
- 表、字段、索引和约束完整性。
- 失败时事务回滚。
- 清理 Migration 不误删保留对象。

SQLite 可以覆盖通用幂等流程，但 PostgreSQL DDL、CHECK、JSONB、部分索引、注释和 Schema 行为必须由 PostgreSQL 专项测试验收。

### 2.6 PostgreSQL 专项测试

PostgreSQL 专项测试使用临时 Schema 隔离，不共用固定测试表。每个测试必须：

1. 从统一 `testutil.PostgreSQLDSN` 读取 DSN。
2. 创建唯一临时 Schema。
3. 通过 `t.Cleanup` 删除临时 Schema 和关闭连接。
4. 不使用生产 Schema 或生产数据。
5. 在 CI 专项作业中强制执行，不允许跳过。

### 2.7 Race 测试

Race 测试用于覆盖共享缓存、Registry、Gin 全局状态、异步任务、并发读取和测试替身。

- 修改共享状态或 goroutine 代码时，相关包必须执行 `go test -race`。
- 历史已知 race 必须记录具体包和复现命令，不能以此跳过所有新 race 测试。
- Race 作业应与普通测试分开显示，便于区分逻辑失败和数据竞争。

## 3. 数据库测试规范

### 3.1 默认使用 OpenSQLite

与数据库方言无关的 Repository 和 Service 测试优先使用：

```go
db := testutil.OpenSQLite(t, &model.Example{})
```

该 Helper 提供：

- 每次调用独立的命名内存数据库。
- 单连接约束，避免 `:memory:` 在多连接下出现多个数据库。
- 统一单数表命名、静默日志和平台时间函数。
- 自动 Cleanup。
- 可选 AutoMigrate。

新增测试禁止直接使用裸 `sqlite.Open(":memory:")`，除非测试目标就是验证特定连接、DryRun、迁移器或错误行为，并在代码中说明原因。

### 3.2 SQLite 适用范围

SQLite 适合：

- 基础 CRUD、分页和受控筛选。
- Service 业务校验和错误传播。
- 不依赖数据库方言的事务回滚。
- 测试夹具隔离。

SQLite 结果不得用于证明 PostgreSQL 特有约束已经生效。

### 3.3 必须使用 PostgreSQL 的场景

以下场景必须使用真实 PostgreSQL：

- JSONB 类型和 JSON 约束。
- 部分唯一索引、表达式索引和 PostgreSQL CHECK。
- Schema、注释、`search_path` 和 PostgreSQL DDL。
- PostgreSQL 锁、并发和事务隔离语义。
- 依赖 PostgreSQL 驱动错误码的行为。
- 清理 Migration 对真实 PostgreSQL 对象的影响。

### 3.4 AutoMigrate 边界

普通测试使用 `OpenSQLite(t, models...)` 统一迁移夹具模型。禁止在同一测试中重复迁移相同模型集合，除非正在验证幂等、Schema 演进或 Migration 本身。

Migration 测试可以显式调用 AutoMigrate 构造前置 Schema。不得为了减少重复而把不同历史 Schema 强行合并成一个万能夹具。

### 3.5 数据和事务隔离

- 每个测试拥有独立数据库或临时 Schema。
- 共享数据库时使用稳定清理策略，不依赖测试顺序。
- `testutil.WithRollback` 仅用于夹具隔离，不替代生产事务测试。
- 并行测试不得共享同名内存数据库、全局 GORM 句柄或可变夹具。
- 测试结束必须关闭 SQL 连接。

## 4. Gin 测试规范

### 4.1 包级初始化

使用 Gin 的测试包必须在唯一的 `TestMain` 中设置测试模式：

```go
func TestMain(m *testing.M) {
    testutil.ConfigureGinTestMode()
    os.Exit(m.Run())
}
```

禁止在测试函数、子测试、路由工厂或公共辅助函数中调用 `gin.SetMode`。Gin 模式是进程级全局状态，分散写入会增加并发测试和 race 风险。

### 4.2 依赖循环例外

`model` 包测试不能导入会反向依赖 `model` 的 `internal/test`，否则形成导入循环。该包允许在自己的 `TestMain` 中直接调用一次 `gin.SetMode(gin.TestMode)`。这是包级初始化例外，不允许恢复到每个测试函数调用。

### 4.3 全局状态恢复

以下全局状态必须使用 `t.Cleanup` 或 TestMain 恢复：

- Zap 全局 Logger。
- 可替换的 Repository、Writer 或 Registry。
- 环境变量。
- Token、时间、随机数或 Snowflake 测试替身。
- Gin 之外的其他包级配置。

修改全局变量的测试默认不得并行运行。

## 5. 测试隔离规范

### 5.1 数据库

- SQLite 使用唯一命名内存数据库。
- PostgreSQL 使用唯一临时 Schema。
- 禁止测试连接开发库或共享正式环境。
- 测试夹具使用稳定业务键，并在测试结束后释放资源。

### 5.2 缓存

- 优先使用实例级内存缓存或测试替身。
- 每个测试创建独立实例，不共享可变 map。
- 必须使用 Redis 时使用独立前缀并清理键。
- 不依赖本地运行中的 Redis 作为普通单元测试前置条件。

### 5.3 全局 Registry 和单例

- Registry 测试使用可独立构造的实例。
- 禁止测试动态注册污染生产默认 Registry。
- 测试后必须恢复全局函数指针、Hook 和默认实现。
- 并发读取测试与重复注册测试必须执行 race 检查。

### 5.4 时间、ID 和随机值

- 时间边界使用可注入时钟或明确固定时间。
- ID 生成器使用独立节点或确定性替身。
- 随机值测试必须能够稳定复现，不以概率作为断言。
- 不使用 `time.Sleep` 代替同步信号，除非测试目标就是超时行为。

## 6. PostgreSQL DSN 门控

统一 Helper 使用两个环境变量：

| 环境变量 | 作用 |
| --- | --- |
| `SWEET_TEST_POSTGRES_DSN` | PostgreSQL 测试连接串 |
| `SWEET_REQUIRE_POSTGRES_TESTS` | 设置为 `1`、`true` 或 `yes` 时，缺少 DSN 直接失败 |

本地普通测试未配置 DSN 时可以跳过 PostgreSQL 专项测试。CI PostgreSQL 作业必须同时设置：

```text
SWEET_TEST_POSTGRES_DSN=<CI PostgreSQL DSN>
SWEET_REQUIRE_POSTGRES_TESTS=1
```

这样可以区分“开发机没有 PostgreSQL”和“CI 本应执行但配置缺失”，避免专项约束测试静默消失。

## 7. CI 要求

### 7.1 普通测试作业

每次提交必须执行：

```bash
go test ./... -count=1
```

普通作业不依赖 PostgreSQL、Redis 或外部网络。使用 `-count=1` 避免缓存结果掩盖测试初始化问题。

### 7.2 PostgreSQL 专项作业

CI 应启动版本固定的 PostgreSQL Service，设置统一 DSN 和强制门控变量，然后至少执行：

```bash
go test ./migrate ./model -count=1
```

专项作业必须展示测试通过、失败和跳过数量。设置强制门控后，缺少 DSN 必须失败。

### 7.3 Race 作业

每次涉及共享状态、异步、缓存、Registry 或 Middleware 的修改，应执行相关包 race 测试。CI 至少应定期执行：

```bash
go test -race ./internal/... ./middleware/... ./service/... ./repository/...
```

若全仓 race 受历史问题阻塞，应拆出稳定包清单持续运行，并为阻塞包建立明确 Backlog，不得永久关闭 race 检查。

### 7.4 CI 可见性

- 普通、PostgreSQL 和 race 必须是独立步骤或作业。
- PostgreSQL 专项不得仅依赖测试内部 `Skip` 判断是否执行。
- 跳过测试必须带明确原因。
- 失败日志不得包含数据库密码或完整 DSN。

## 8. 本次轻量整改

本 Task 只调整测试基础设施和测试初始化，不修改业务代码：

1. 新增统一 Gin 测试模式 Helper。
2. 为 Controller、Initialize、Data Permission Internal、Middleware、Model 和 Service 六个测试包建立包级 TestMain。
3. 移除 17 个测试文件中的 21 处分散 `gin.SetMode`。
4. 新增统一 PostgreSQL DSN Helper 和 CI 强制门控开关。
5. 将 4 个 PostgreSQL 专项测试迁移到统一 DSN Helper。
6. 增加 Gin 模式和 PostgreSQL 强制开关测试。

整改后，具体测试函数和路由辅助方法中不再调用 `gin.SetMode`；PostgreSQL 专项测试不再各自读取环境变量和复制 Skip 逻辑。

## 9. 未整改范围与原因

### 9.1 SQLite 存量初始化

35 处直接 SQLite 初始化本次不做批量替换，原因包括：

- 部分测试使用 DryRun 或自定义 Dialector。
- 部分测试刻意验证多连接、共享内存或 Repository 错误行为。
- Migration 测试需要构造不同历史 Schema。
- 全量机械替换可能改变命名策略、外键迁移和连接池语义。

后续应优先迁移重复且无特殊配置的 Service 和简单 Repository 测试，再处理 SysTable、Report 和 Migration 专项测试。

### 9.2 AutoMigrate 重复

45 处 AutoMigrate 本次保留。普通夹具应逐步收口到 `OpenSQLite`，但 Migration 幂等和历史 Schema 测试必须继续显式迁移。

### 9.3 CI 工作流

当前工作流仍只有普通 `go test ./...`。PostgreSQL Service、强制门控和独立 race 作业需要结合 CI 资源、密钥和运行时长单独配置，不在本次 backend 测试工具轻量整改中直接加入。

### 9.4 已知 Controller race

本次 race 复核仍可在 `TestOrgControllerEmployeeUserBindingUsesPermissionsAndSafeResponse` 复现历史竞争：请求级 `*gin.Context` 被用作数据库 Context，`database/sql` 完成异步 Rows 关闭前 Gin 已开始复用请求 Context。

该问题不是分散调用 `gin.SetMode` 导致，也不能通过测试延时、关闭 race 或跳过测试解决。它涉及生产 Repository Context 设计，应在标准 Context 后续治理任务中处理；本 Task 不修改业务查询和 Repository 边界。

## 10. 渐进迁移顺序

建议后续按以下顺序推进：

1. `application_service_test.go` 和简单 Repository 测试迁移到 `OpenSQLite`。
2. SysTable、Report 等具有特殊数据库配置的测试按文件评审。
3. Migration 测试提取可复用 Schema 夹具，但保留不同历史状态。
4. 后端 CI 增加 PostgreSQL 专项作业并设置强制门控。
5. 后端 CI 增加稳定包 race 作业，再逐步扩大范围。

每次迁移只处理一个包或一类夹具，并执行普通测试和相关 race 测试，避免测试基础设施治理掩盖业务回归。
