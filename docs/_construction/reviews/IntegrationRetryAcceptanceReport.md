# Sweet Platform Integration Retry V1 正式验收报告

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| Task | INT-004D |
| 验收范围 | RetryPolicy、Snapshot、Decision、Backoff、Jitter、Retry-After、调度、Runner、页面、权限与脱敏 |
| 验收日期 | 2026-08-09 |
| 验收基线 | `b4acfe58b06b5d3abb717889e854921d71f4c0b5`（实现集成自动重试调度） |
| INT-004A | `3a1c71e8d3859c137412ba53b9eb30587b07fac1` |
| INT-004B | `fe2e87bce00783e7e42cf132512210d1784ab3d8` |
| INT-004C-1 | `c534b02cc99a202c13959bc31a150b698ef50f78` |
| INT-004C-2 | `b4acfe58b06b5d3abb717889e854921d71f4c0b5` |
| 验收环境 | macOS arm64、Go 1.26.2、Node.js 24.14.0、PostgreSQL 16.14、Redis 6.2.7、Docker Compose、本地 TLS Server |
| 正式设计 | [IntegrationRetryDesign.md](../design/IntegrationRetryDesign.md) |
| Runtime 冻结依据 | [IntegrationRuntimeAcceptanceReport.md](IntegrationRuntimeAcceptanceReport.md)、[IntegrationRuntimeFreezeReview.md](IntegrationRuntimeFreezeReview.md) |

本报告以当前仓库真实代码、Migration、Router、Seed、页面、测试和浏览器操作为依据，不以历史 Task 回复代替验收证据。

## 2. 正式结论

**最终验收结论：通过。**

Retry V1 已沿 Runtime 冻结链路完成策略配置、策略快照、统一决策、到期领取、Attempt 追加和自动调度。PostgreSQL 16 强制门控、Runner + TLS 端到端、后端全量、专项 race、前端全量及真实登录态页面验收均通过。当前没有代码、安全、权限、事务、重试真实性或自动重发风险阻塞项。

允许冻结 Retry V1，并允许进入 INT-005 SyncTask。SyncTask 只能通过既有 Application Service 创建 Execution，不能修改 Retry 状态机、直接修改 `next_run_at`、自行调用 Transport 或建立第二套重试链。

## 3. 唯一执行链

```text
Application Service
  -> IntegrationExecution + RetryPolicySnapshot
  -> IntegrationWorkerRunner
  -> PostgreSQL SKIP LOCKED Claim
  -> IntegrationLog / running Attempt
  -> ExecutionInputSnapshot 复核
  -> CredentialProvider
  -> TransportClient
  -> RetryDecision
  -> Attempt + Execution 原子收敛
```

确认不存在 `RetryExecution`、独立 Retry Engine、独立 Retry Runner、Controller HTTP 调用或进程内睡眠调度。首次调用和自动重试共享同一个 Execution、Engine、Runner、CredentialProvider、TransportClient、租约和完成事务。

## 4. RetryPolicy 配置中心

| 验收项 | 结果 | 说明 |
| --- | --- | --- |
| 创建、编辑 draft | 通过 | `policy_code` 不可修改，技术字段仅 draft 可编辑 |
| 版本递增 | 通过 | 同一 code 从 version 1 单调创建下一版本 |
| enabled 技术字段只读 | 通过 | 修改必须创建新版本 |
| 单 enabled 版本 | 通过 | Service 事务与 PostgreSQL partial unique index 双重保护 |
| enabled / disabled | 通过 | 受 revision、引用保护和 Audit 约束 |
| enabled 引用保护 | 通过 | InterfaceDefinition 引用时禁止非法停用 |
| `max_attempts` | 通过 | 1-10，包含首次 Attempt，默认 3 |
| delay / window | 通过 | 毫秒单位，完整尝试计划必须落在 Window 内 |
| backoff / jitter | 通过 | V1 仅 fixed、exponential、none、full |
| ErrorCategory | 通过 | 仅受控 Runtime 分类 |
| HTTP 状态 | 通过 | 仅允许 429、502、503、504 |
| 数据库约束 | 通过 | CHECK、外键、code+version 和 partial unique 实测有效 |

本次补强 InterfaceDefinition 引用校验：不仅要求策略 enabled，还复核完整参数组合与尝试计划，防止绕过 RetryPolicy Service 写入的脏策略被接口引用。

## 5. 策略冻结

- InterfaceDefinition 引用明确 RetryPolicy 版本，已启用接口不能原地修改引用。
- 创建接口新版本复制原引用，并可按版本规则切换到另一 enabled 策略。
- Execution 创建时由服务端构造最小 `RetryPolicySnapshot`，请求 DTO 没有客户端覆盖入口。
- 源策略后续 disabled 或创建新版本，不改变已有 Execution。
- 无策略接口不自动重试，不套用默认策略。
- Snapshot 缺失或损坏在 HTTP 前安全失败；历史无合法 Snapshot 的 `retry_waiting` 不会被套用新策略。
- Response DTO 和页面不返回 Snapshot 完整 JSON。

## 6. RetryDecision

Engine 在 Attempt 完成前调用独立 RetryDecision。Decision 不依赖 Gin、GORM、Repository、HTTP Client 或 Credential，只返回安全建议；真正状态变更仍由 Engine 完成事务执行。旧的 429/5xx 简单判断没有作为第二套真值保留。

平台硬禁止集合优先于 Policy。`invalid_config`、Credential 失败、SSRF、TLS 安全失败、输入快照错误、`response_too_large`、`unsupported_content_type`、401、403、cancelled 和内部安全错误均不能被脏 Policy 放开。受控候选仅包括策略明确允许的临时 network、timeout、429、502、503、504。

## 7. confirmed、unknown 与远端幂等

| 场景 | 结果 |
| --- | --- |
| GET confirmed / unknown | 满足策略、次数和窗口时可重试 |
| PUT / DELETE unknown | 仅接口明确 `idempotent_method` 时继续评估 |
| POST / PATCH unknown | 仅 `remote_key_header` 且 Execution 冻结远端幂等键时继续评估 |
| 本地 Execution 幂等键 | 不作为远端幂等证据 |
| unknown 非幂等写 | 最终 failed，不自动再次发送 |
| 远端幂等键 | 客户端普通 Header 不可覆盖；所有 Attempt 使用同一冻结键 |

Runner + TLS 端到端验证 unknown 非幂等 POST 只发送一次，也验证 remote key POST 可以重试且两个 Attempt 的 `Idempotency-Key` 完全一致。Credential 仍在输入复核后由 Provider 最后适配。

## 8. Attempt 次数与 Retry Window

- `max_attempts` 包含首次调用：1 只允许 Attempt1；2 最多 Attempt1/2；3 最多 Attempt1/2/3。
- `attempts_remaining = max_attempts - current_attempt`，不产生 Attempt4。
- Decision 和 Repository Claim 都检查次数。
- Retry Window 从首次 Attempt `started_at` 起算，不使用 Execution `created_at` 或当前 retry_waiting 时间。
- 当前时间等于截止点、`next_run_at` 等于或超过截止点时均不调度。
- 次数尚有剩余但 Window 耗尽时收敛 failed。
- PostgreSQL 唯一约束保护 `execution_id + attempt_no`，Attempt 只追加不覆盖。

## 9. Backoff、Jitter 与 Retry-After

- fixed 使用 `initial_delay_ms`；exponential 从首次 Retry 的 index 1 开始按 multiplier 增长，并受 `max_delay_ms` 限制。
- 延迟按整数毫秒向上取整，平台最小值 1 秒。
- none 不修改 base delay；full jitter 在 `[0, base_delay]` 范围内取值。
- RandomSource 可注入且通过 race；已持久化 delay 不会在 Worker 重启后重新抽随机。
- Retry-After 支持严格 delta-seconds 和 HTTP-date。
- invalid、negative、overflow 按正式设计安全回退本地 Backoff；`respect=false` 时忽略。
- 候选延迟为 `max(local_backoff_after_jitter, retry_after_delay)`。
- Retry-After 超出 max delay 或 Window 时不静默截断后提前执行，而是终止自动调度。

## 10. retry_waiting 调度与数据库时间

- Runner 统一领取 created 和已到期 retry_waiting，不启动第二个 Retry Runner。
- 候选按 `COALESCE(next_run_at, gmt_create), gmt_create, id` 排序，避免长期饥饿。
- 到期条件使用 `next_run_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'`。
- PostgreSQL `FOR UPDATE SKIP LOCKED` 保证多实例同一 Execution 只有一个领取者。
- 领取事务原子写 running、lease、`current_attempt + 1`、revision、`last_attempt_at`，追加 Attempt 并清空 `next_run_at`。
- 首次 `started_at` 不覆盖，上一轮调度事实保留在 Attempt。

本次修复时间真值的最后一处漂移：Claim、完成、租约校验、RetryDecision CurrentTime 和 RecoverExpiredLease 统一取 Repository 数据库时钟。测试故意给应用时钟增加 8 小时，仍按 PostgreSQL UTC 时间正确领取、完成和恢复。

## 11. Cancel / Claim、租约与并发

| 验收项 | 结果 |
| --- | --- |
| Cancel 先提交 | cancelled，Worker 不领取，不新增 HTTP |
| Claim 先提交 | running，Cancel 稳定冲突 |
| 两 Worker同时 Claim | 仅一个成功，仅一个新 Attempt |
| revision / current_attempt | 每次领取仅增加一次 |
| 实例并发 | 受 Runner `instance_concurrency` 限制 |
| System / Interface 并发 | 继续进入现有 ConcurrencyGuard |
| Retry Attempt 租约 | 复用 Runtime 默认安全租约与 owner/revision |
| 过期恢复 | failed + unknown，不回 retry_waiting |
| 优雅关闭 | 停止新领取，有限等待，未完成交租约恢复 |

## 12. PostgreSQL Runner + TLS 端到端

真实 PostgreSQL 16.14、常驻 Runner 和本地 TLS Server 完成以下链路；测试没有手工调用第二次 `RunExecution`：

| 场景 | 最终结果 |
| --- | --- |
| 503 -> 200 | succeeded，Attempts=2，HTTP=2 |
| 503 -> 503，max_attempts=2 | failed，Attempts=2，无 Attempt3 |
| 429 + Retry-After -> 200 | 正确计算 next time，到期后 Runner 自动领取 |
| unknown 非幂等 POST | failed，HTTP=1 |
| unknown + remote key POST | 可重试，两个 Attempt 使用同一远端键 |
| max_attempts=1 | 只存在 Attempt1 |

PostgreSQL 门控还覆盖 RetryPolicy CHECK、partial unique、InterfaceDefinition 外键、Snapshot JSONB、Attempt retry 字段、status+next_run_at 索引、Migration 幂等、database time、SKIP LOCKED、Cancel/Claim 和租约恢复。

## 13. DTO、页面与脱敏

Execution DTO 只新增 `max_attempts`、`attempts_remaining`、`next_run_at`、reason、policy code/version。Attempt DTO 只返回 retryable、reason、delay、scheduled time 和安全 source。DTO 不返回 RetryPolicySnapshot、ExecutionInputSnapshot、CredentialMaterial、Authorization、Cookie、API Key、Token、正文、密文、nonce、存储引用或底层错误。

真实管理员浏览器验收结果：

- 菜单顺序为外部系统、接口定义、集成凭证、重试策略、执行记录、调用日志。
- 创建临时 draft，完成编辑、启用 v1、创建 v2、停用 v1。
- InterfaceDefinition 仅在具备 RetryPolicy query 权限时加载 enabled 策略。
- Execution 列表和详情显示安全策略、Attempt、next time、中文 reason 和输入摘要。
- Attempt 通过独立 Log API 加载，显示 retry delay、scheduled time 和 source。
- Worker 显示当前实例状态，未启用为中性状态。
- 深色模式可读，无重叠或独立亮色滚动区域。
- 页面没有立即重试、跳过等待、修改 next time、修改策略、重放、start、complete、fail、Payload 查看或在线调试。

无权限账号 `dp_acceptance_ungranted` 实际登录后为 0 个菜单；直达 RetryPolicy 和 Execution 路由均进入 404，不加载或泄露数据。验收后临时 Policy、InterfaceDefinition、Execution 和 Attempt 已删除，账号字段已恢复；管理员操作审计作为不可变历史保留。

## 14. 权限、Audit 与日志

- RetryPolicy 菜单和 query/detail/create/edit/create_version/enable/disable 来自 `sys_menu_button` 与 Casbin，不按角色名硬编码。
- RetryPolicy 是平台级配置，不伪造 Organization Ownership。
- Execution 和 IntegrationLog 继续使用独立功能权限及 Data Permission query/detail。
- 无日志权限时前端不请求 Attempt，后端直接访问稳定拒绝且不泄露存在性。
- 管理员策略动作和取消 retry_waiting 写标准 Audit。
- 自动 Retry 只写 IntegrationLog 技术事实，不伪造管理员 AuditSubject。
- 结构化日志不记录 Snapshot、Payload、Credential 或认证材料。

## 15. 自动化结果

```text
SWEET_TEST_POSTGRES_DSN=postgres://sweet_admin:***@127.0.0.1:15432/sweet_admin?sslmode=disable
SWEET_REQUIRE_POSTGRES_TESTS=true
cd backend && go test ./... -count=1
结果：通过，PostgreSQL 门控实际执行，未跳过。

go test -race ./internal/integration/... ./repository/impl ./service ./initialize -count=1
结果：通过

go test -race ./controller -run 'Retry|Integration' -count=1
结果：通过（backend/controller 1.831s）

yarn test
结果：通过，33 个测试文件、122 个测试

yarn lint / yarn typecheck / yarn build
结果：全部通过
```

Build 仅保留既有大 chunk 提示，不影响产物或本次验收。

## 16. 本次发现并修复的问题

1. 运行时数据库时钟未完全统一：完成、租约和 Decision 仍可能使用应用时钟；已统一为 Repository 数据库时钟。
2. InterfaceDefinition 原先只检查策略 enabled；已增加完整参数与尝试计划复核。
3. Retry-After 缺少 negative、overflow、respect=false 覆盖；已补齐。
4. Cancel / Claim 缺少确定性双方胜出测试；已补 cancel-first、claim-first。
5. E2E 缺少 429、unknown 非幂等、remote key 和 max_attempts=1；已补齐。
6. Execution/Attempt 页面安全 Retry 摘要不足；已补摘要、中文 reason 和一致时间格式。
7. InterfaceDefinition 无权限仍请求策略 API；已按动态按钮权限阻止请求。
8. 前端 Retry Window 只校验初始延迟；已与后端一致校验完整尝试计划。

## 17. 非阻塞限制

V1 不支持立即重试、人工重放、跳过退避、独立 Retry Dashboard、linear backoff、equal jitter、自定义表达式、SyncTask/SyncBatch、HR 同步、OAuth，以及前端高级查询和一致性治理。

这些是后续能力边界，不构成本次失败理由。既有前端大 chunk 提示和历史 Organization/Gin race 也不是 Retry V1 代码阻塞。

## 18. 冻结与后续入口

**冻结结论：Retry V1 通过冻结。**

冻结对象和扩展约束见 [IntegrationRetryFreezeReview.md](IntegrationRetryFreezeReview.md)。允许进入 INT-005 SyncTask，但必须复用冻结的 Snapshot、Decision、Runner、Engine、Credential 和 Transport 链路。
