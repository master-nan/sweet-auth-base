# Sweet Platform Frozen Backlog 最终稳定性评审

> Audience: 平台架构、工程维护和后续专项负责人
>
> Lifecycle: construction
>
> Final Action: DELETE_AFTER_STABLE
>
> Review Baseline: `4f88860385bc1289f9899c0e62a47b64f0d3a4e1`

## 1. 评审范围

本评审以 AF-001 至 AF-006 的真实代码、路由、测试、PostgreSQL 16、Race、前端构建和文档检查为依据，不复用历史审计数字作为结论。

当前基线包括 358 个 Go 生产文件、193 个 Go 测试文件、39 个后端 Package 和 866 个顶层 Go Test。AF-006 不新增产品能力，集中完成 Data Permission Context 收口、测试夹具隔离、历史 Race 复核、死代码确认删除和完整稳定性回归。

## 2. AF-001 至 AF-006 结果

| Task | 已收口能力 | 最终状态 |
| --- | --- | --- |
| AF-001 | Gin 仅留在 HTTP Adapter，Repository、Model 和核心 Service 使用 `context.Context` | 通过 |
| AF-002 | 密码、开放 API、短信、Refresh、Logout 进入统一认证 Application Service | 通过 |
| AF-003 | File Query、Upload、Access、Lifecycle 职责和签名 purpose 边界 | 通过 |
| AF-004 | Metadata Definition、Runtime Read、Security Metadata 边界 | 通过 |
| AF-005 | DTO 白名单、稳定 Error、事务边界和补偿语义 | 通过 |
| AF-006 | 测试基础设施、Race、死代码和全量回归 | 通过 |

## 3. Context 与历史 Race

Data Permission Controller 现在只负责从 Gin 提取标准请求 Context，Core Service、Resolver、Subject Builder、Generalization 和 Integration Permission Adapter 均不再依赖 `*gin.Context`。Resolver Summary 和 AuditSubject 通过标准 Context 传播，并保留并发安全的摘要读取。

生产 `*gin.Context` 最终分布如下：Controller 276、API 11、Middleware 24、Service 23、Repository 0、Model 0、Internal HTTP Helper 6、Initialize Router 3。Service 的 23 处全部位于 Report Service；Report 已进入后续专项重做范围，本轮只增加标准 Context 适配，不改变报表领域语义。

历史 Employee User Binding / Gin 测试竞态通过扩大的 Controller、Service 定向 Race 和全仓 Race 复核，未再出现。测试 Gin Mode 由公共测试运行时统一设置；Model Package 因测试辅助包反向依赖 Model 而在独立 `TestMain` 单点设置，未在并行测试中动态修改。

## 4. 测试基础设施

- 普通 SQLite Fixture 统一使用 `testutil.OpenSQLite`；调用共 101 处。
- 直接 SQLite 初始化剩余 7 处：2 处 DryRun、2 处 Model Hook、3 处 Migration/历史结构测试，均保留其特殊语义。
- `AutoMigrate` 测试调用剩余 40 处，属于局部 Model Fixture、Migration 或历史 Schema 测试；未把真实 Migration 测试降级为模拟 Schema。
- 新增 `OpenSQLiteWithConfig`，为需要定制 GORM 配置或多连接行为的测试提供唯一且隔离的共享内存数据库。
- 三次全仓测试曾发现 Organization HR、Report、Basic Repository 和 Generalization Fixture 名称复用导致的跨轮污染；改为唯一数据库名后通过。
- PostgreSQL 强制门控仍由统一测试辅助函数控制：普通模式可明确跳过，`SWEET_REQUIRE_POSTGRES_TESTS=true` 且缺少 DSN 时直接失败。
- 新增真实 PostgreSQL Metadata DDL 验证，覆盖列增删改、唯一索引和事务回滚。

测试中的 `time.After` 均作为失败超时保护；剩余 `time.Sleep` 仅用于真实轮询、租约或传输延迟语义。Worker/Sync Runner 中原有固定等待已改为 Channel/Barrier 或显式数据库状态，不再依赖“睡够时间”。

## 5. Dead Code 与 Swagger

经 Router、Swagger、接口实现、认证调用、测试和仓库引用交叉确认，删除以下无外部契约实现：

- `SysUserService.GetAll`
- `SysRoleService.CreateRoleMenu`、`DeleteRoleMenu`
- `ApplicationController.GetApplicationByAppKey`
- `UserController.GetUserByUserName`
- `TableController.DeleteTableIndexByTableId`
- 对应 Request/Response DTO、Repository Helper 和 Generalization/Data Policy/Metadata 的无调用包装
- 旧 Refresh Request DTO、Basic Repository `WithOmit`、无调用 SMS/Role/Menu Helper

认证核心仍使用的 `ApplicationService.GetApplicationByAppKey` 和 `SysUserService.GetByUserName` 明确保留。Swagger 已重新生成，三个无路由历史 Path 不再存在；孤立注解不再作为永久保留理由。

## 6. DTO、Error 与 Transaction 防回归

- 生产 HTTP Controller/API 直接返回 GORM Model：0。
- `NewBadRequestError(err.Error())`：0。
- `fmt.Print*`：0。
- 生产 Controller 业务事务：0。
- 生产显式嵌套事务：0。
- `RunInTransaction` 共 84 处（含 Helper 定义），用于 Service 短事务。
- `ExecuteTx` 共 24 处（含接口和实现）：SysTable 保留 DDL/Metadata 特殊补偿边界，Integration Execution Repository 保留 Claim、Attempt 和 Completion 的单 Repository 原子更新。
- 直接 GORM `Transaction` 共 15 处：Service Helper、Basic Repository 基础设施和 Migration/Seed；无 Controller 直接事务。

AF-006 额外修复短信状态查询的 P0 边界：查询必须匹配当前 Application 和手机号；响应仅返回稳定状态，不返回供应商内容或手机号；空明细安全失败；持久化模板参数统一脱敏。数据库、第三方和供应商错误继续只在内部诊断，不进入客户端动态消息。

## 7. 完整验证

以下验证均在本 Task 代码上执行：

- `go test ./... -count=1`：通过。
- `go test ./... -count=3`：通过，且已用于发现并修复 Fixture 隔离问题。
- `go test -race ./... -count=1`：通过。
- PostgreSQL 16 强制门控全仓回归：通过；覆盖 Integration、Retry、Sync、Organization HR、Migration 和 Metadata DDL。
- 前端 `yarn test`：36 个测试文件、136 个测试通过。
- 前端 `yarn lint`、`yarn typecheck`、`yarn build`：通过。
- `make docs-check`：通过，0 断链。

前端构建保留两个非阻塞告警：Node 子进程 `shell=true` 弃用提示，以及部分 Chunk 超过 900 kB。它们属于后续 Frontend Consistency/Build Optimization，不影响本轮后端冻结遗留关闭。

## 8. 安全与边界

受跟踪源码、测试 Fixture 和文档未新增真实 HR 原始响应、真实人员信息、内网地址、Token、Cookie、Authorization 或 Credential Secret。`docs/development` 和 HR Raw 资料继续不进入 Git。

Organization HR Production Gate 未因本评审关闭；Report Gin Context 未被伪装为已治理；Metadata 跨数据库 DDL 补偿、Casbin/Cache 对账也没有被包装成分布式事务保证。

## 9. 剩余 Backlog

以下事项已转入明确专项，不阻塞 Frozen Backlog 关闭：

| 分类 | 剩余事项 |
| --- | --- |
| A. Platform Enablement | 最终用户、工程和运维文档重写及 DOC-FINAL |
| B. Frontend Consistency | 动态列、Quasar/CSS、页面和构建分包一致性 |
| C. Metadata 专项 | 跨数据库 DDL 补偿、对象级元数据授权 |
| D. Operations/Observability | Cache Retry、Casbin Reconciliation、监控告警 |
| E. Report Platform | Report Service Context 和产品架构专项重做 |
| F. Production Enablement | HR Source ID、时间、人员响应量等生产 Gate |
| G. Query Center | 高级查询、方案和统一查询中心 |

## 10. 最终结论

**Frozen Backlog 可以正式关闭。**

已知 P0/P1 架构债已完成治理，或被明确转入具有独立边界的后续专项；全量测试、全仓 Race、PostgreSQL 强制门控、前端四项和文档检查均通过。后续工作应进入 Platform Enablement，并按上述分类推进，不再以 Frozen Backlog 名义继续扩大架构整改范围。
