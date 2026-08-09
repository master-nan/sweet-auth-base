# Sweet Platform Integration Retry V1 冻结评审

## 1. 冻结背景和依据

Integration Runtime 第一期已通过 [IntegrationRuntimeAcceptanceReport.md](IntegrationRuntimeAcceptanceReport.md) 验收并由 [IntegrationRuntimeFreezeReview.md](IntegrationRuntimeFreezeReview.md) 冻结。INT-004A 至 INT-004C-2 在该执行链上增加 RetryPolicy、RetryDecision 和 retry_waiting 调度，INT-004D 以 `b4acfe58b06b5d3abb717889e854921d71f4c0b5` 为基线完成 PostgreSQL 16、Runner + TLS、race、前端和真实浏览器验收。

正式验收依据为 [IntegrationRetryAcceptanceReport.md](IntegrationRetryAcceptanceReport.md)。本评审只冻结已经通过的 Retry V1，不把立即重试、人工重放、SyncTask 或 OAuth 纳入当前能力。

## 2. 冻结范围

本次冻结：

- `RetryPolicy` 版本模型和参数边界。
- `RetryPolicySnapshot` 的 Execution 创建时冻结语义。
- `RetryDecision` 唯一决策入口。
- fixed / exponential Backoff 与 none / full Jitter。
- Retry-After 解析和候选时间规则。
- `next_run_at` 持久化与到期领取。
- `retry_waiting -> running` 的 PostgreSQL 原子领取。
- 同一 Execution 下 Attempt 单调追加。
- Runner、租约、ConcurrencyGuard、取消与恢复边界。
- RetryPolicy、Execution、Attempt 页面、权限、DTO 和脱敏边界。

Runtime 冻结保护的 `IntegrationExecution`、`IntegrationLog / Attempt`、`ExecutionInputSnapshot`、`IntegrationRuntimeLimits`、`CredentialProvider`、`TransportClient`、`IntegrationExecutionEngine`、`IntegrationWorkerRunner` 继续有效，Retry 不改变其核心语义。

## 3. 唯一运行链

```text
Application Service
  -> IntegrationExecution + RetryPolicySnapshot
  -> IntegrationWorkerRunner
  -> ClaimReadyExecutions
  -> IntegrationLog / running Attempt
  -> ExecutionInputSnapshot 复核
  -> CredentialProvider
  -> TransportClient
  -> RetryDecision
  -> CompleteAttemptAndExecution
```

自动重试是原 Execution 的继续执行，不是新 Execution 或人工重放。所有 Attempt 共用冻结输入、接口版本和策略快照，每次重新解析当前有效 Credential，再通过唯一 Transport 发出请求。

## 4. RetryPolicy 版本模型

一个逻辑策略由稳定 `policy_code` 标识，技术版本由 `policy_code + version` 唯一。状态仅为 draft、enabled、disabled；draft 可编辑，enabled 的技术字段不可原地修改，变化必须创建新版本。同一 code 同时最多一个 enabled 版本，由 Service 事务和 PostgreSQL partial unique index 共同保护。

V1 不提供删除。enabled InterfaceDefinition 引用时禁止非法停用策略。InterfaceDefinition 引用明确版本，已启用接口不能原地修改策略引用。

## 5. RetryPolicySnapshot

Execution 创建时由服务端把引用策略的最小执行语义冻结为 Snapshot，包括 code/version、最大 Attempt、Backoff、Jitter、Retry Window、允许错误分类、HTTP 状态和 Retry-After 规则。客户端不能提交或覆盖 Snapshot。

源策略停用、修改草稿或创建新版本不改变已有 Execution。Retry Worker 不读取“当前最新策略”；Snapshot 缺失或损坏时安全失败，不套用默认策略。

## 6. Attempt 次数语义

`max_attempts` 包含首次 Attempt。值为 1 表示不自动重试；值为 3 表示最多 Attempt1、Attempt2、Attempt3。Decision 和 Claim 均校验 `current_attempt < max_attempts`，数据库唯一约束保护 `execution_id + attempt_no`。

Attempt 只追加、不覆盖、不复用编号。已经完成的 Attempt 是不可变历史事实，新的重试只能在同一 Execution 下追加下一 Attempt。

## 7. RetryDecision 唯一真值

RetryDecision 是独立纯决策能力，输入服务端事实和受控时钟，输出 retryable、final state、reason、next time、delay、remaining、Retry-After source 和 determinacy。Decision 不读数据库、不发送 HTTP、不修改 Model。

Engine 只能通过 RetryDecision 决定失败后进入 `retry_waiting` 或 `failed`。Runner 和 Repository 只消费已经持久化的 `next_run_at`，不得重新解析 Retry-After、重算 Backoff 或形成第二套资格规则。

## 8. 平台硬禁止优先级

决策优先级冻结为：

1. Runtime 硬安全禁止。
2. 结果确定性和远端幂等。
3. Attempt 次数和 Retry Window。
4. Policy ErrorCategory / HTTPStatus 白名单。
5. Backoff / Jitter / Retry-After。

Policy 不能放开配置错误、Credential、SSRF、TLS 安全错误、输入快照错误、响应超限、Content-Type 错误、401、403、cancelled 或内部安全错误。

## 9. confirmed、unknown 与远端幂等

confirmed failure 可以在策略、次数和窗口均允许时重试。unknown 不是普通失败：GET 可继续评估；PUT/DELETE 必须明确 `idempotent_method`；POST/PATCH 必须为 `remote_key_header` 且 Execution 已冻结远端幂等键。

本地 Execution idempotency key 不构成远端幂等证据。unknown 非幂等写最终 failed。远端幂等键由接口契约定义，客户端普通 Header 不可覆盖，每个 Attempt 注入同一冻结键。

## 10. Backoff 与 Jitter

V1 Backoff 仅 fixed、exponential。首次失败调度 Attempt2 的 retry index 为 1，exponential 按 multiplier 增长并受 `max_delay_ms` 约束。时间使用整数毫秒和向上取整，平台最小延迟为 1 秒。

Jitter 仅 none、full。full 使用可注入、并发安全的 RandomSource，结果不小于 0 且不超过 base delay。产生后的 delay 和 schedule 持久化到 Attempt，Worker 重启不得重新抽随机。

## 11. Retry-After

V1 支持 delta-seconds 和 HTTP-date。仅在 Policy `respect_retry_after=true` 且场景允许时使用，候选延迟固定为 `max(local_backoff_after_jitter, retry_after_delay)`。

非法、负数和溢出值不形成无限等待。Retry-After 超过 max delay 或 Retry Window 时不静默截断后提前执行，而是停止自动调度。Runner 不读取原始 Header，只消费 Decision 持久化的调度结果和安全 source 编码。

## 12. Retry Window

Retry Window 从首次 Attempt `started_at` 起算。Execution 创建时间、当前 retry_waiting 时间和 Worker 领取时间都不是窗口起点。当前时间达到截止点，或计算出的 `next_run_at` 达到或超过截止点，均不得创建后续 Attempt。

即使 Attempt 次数还有剩余，Window 耗尽也必须收敛 failed。

## 13. next_run_at 与数据库时间

`next_run_at` 是 RetryDecision 计算并由完成事务持久化的唯一当前调度时间，只允许 retry_waiting 持有。领取进入 running 后清空；上一轮调度事实保留在 Attempt 的 `retry_scheduled_at` 和 `retry_delay_ms`。

到期、领取、租约完成和恢复均使用 PostgreSQL `CURRENT_TIMESTAMP AT TIME ZONE 'UTC'`，与 UTC 语义的 `timestamp without time zone` 字段匹配。应用 `time.Now()` 不作为多实例调度真值。

## 14. retry_waiting 原子领取

Runner 统一调度 created 和到期 retry_waiting。PostgreSQL `FOR UPDATE SKIP LOCKED` 与状态、时间、revision、次数、窗口和 Snapshot 条件确保多实例只有一个领取者。

领取短事务原子写 running、lease owner/expiry、current Attempt、revision 和 last Attempt time，追加 running Attempt，随后提交事务。Credential、输入重建和 HTTP 全部在事务外执行。

## 15. Attempt 单调追加与完成

第一次调用是 Attempt1，第一次自动重试是 Attempt2。完成事务锁定 Execution，校验 running、lease owner、revision 和 running Attempt，再同时完成 Attempt、更新 Execution 和清理租约。

成功收敛 succeeded；Decision 允许时收敛 retry_waiting 并写 `next_run_at`；次数、窗口或安全规则终止时收敛 failed。完成失败不能返回成功。

## 16. Cancel / Claim

管理端只允许取消 created 或 retry_waiting。Cancel 和 Claim 通过行锁、状态和 revision 竞争：Cancel 先成功时 Worker 无候选；Claim 先成功时状态已 running，Cancel 稳定拒绝。

不得通过前端隐藏替代后端竞争保护，也不得出现 cancelled 与新 Retry Attempt 已发送 HTTP 同时成立。

## 17. 租约恢复与 unknown

Retry Attempt 使用 Runtime 冻结租约和完成余量。租约真正过期时，RecoverExpiredLease 将 running Attempt 与 Execution 收敛为 failed + unknown，不再放回 retry_waiting，也不自动重发。

应用关闭只停止新领取并有限等待当前任务；未完成任务交由租约恢复，不批量改 cancelled，不伪造成功。

## 18. ConcurrencyGuard

Retry 与首次调用共享 Runner 的实例并发上限，并继续通过平台、ExternalSystem 和 InterfaceDefinition 三级 ConcurrencyGuard。进程内 semaphore 只优化本实例资源，数据库租约和 SKIP LOCKED 才是多实例唯一领取基础。

到期 Retry 不得因优先级或数量绕过 Guard，也不得另启不受控 goroutine。

## 19. 权限、DTO 与脱敏

RetryPolicy 使用 `sys_menu_button` 和 Casbin 提供 query、detail、create、edit draft、create version、enable、disable，不按角色名称硬编码。Execution 和 IntegrationLog 继续使用独立功能权限和 Data Permission query/detail。

Execution DTO 只暴露安全调度摘要，Attempt DTO 只暴露安全 Retry 诊断。禁止返回 RetryPolicySnapshot、ExecutionInputSnapshot、CredentialMaterial、Header 原值、Authorization、Cookie、API Key、Token、Payload、密文、nonce、存储引用或底层错误正文。

管理员配置动作写标准 Audit；自动 Retry 只记录 IntegrationLog 技术事实，不伪造管理员 AuditSubject。

## 20. 页面管理边界

集成中心菜单顺序冻结为：外部系统、接口定义、集成凭证、重试策略、执行记录、调用日志。RetryPolicy 页面只提供查询、详情、创建、编辑 draft、创建版本、启用和停用。

Execution 和 Attempt 页面显示 policy、Attempt 次数、next time、reason、delay、scheduled time 和安全 source，不显示完整 Snapshot 或 Payload。V1 页面禁止立即重试、跳过等待、修改 next time、修改 Execution 策略、修改最大次数、人工重放、start、complete、fail 和在线调试。

## 21. V1 明确不支持

Retry V1 不支持立即重试、人工重放、跳过退避、独立 Retry Dashboard、linear backoff、equal jitter、自定义错误表达式、动态限流配置、SyncTask/SyncBatch、HR 同步、OAuth，以及前端高级查询和一致性治理。

这些能力不得通过修改数据库状态、临时 Controller API 或复制执行链实现。

## 22. SyncTask 扩展边界

后续 INT-005 SyncTask 只能作为触发、批次、业务映射和结果关联层：

- 通过现有 Application Service 创建 IntegrationExecution。
- 使用冻结的 InterfaceDefinition、ExecutionInputSnapshot 和 RetryPolicySnapshot。
- 由现有 Runner、Engine、CredentialProvider 和 TransportClient 执行。
- 通过 Execution / Attempt 查询结果，不直接更新运行状态。

SyncTask 不得修改 Retry 状态机，不得直接修改 `next_run_at`，不得自行调用 Transport，不得解密 Credential，不得自行实现重试，也不得覆盖历史 Attempt。

## 23. 最终冻结结论

**冻结结论：通过冻结。**

INT-004D 已完成 RetryPolicy、Snapshot、Decision、幂等安全、Backoff/Jitter、Retry-After、Window、数据库调度、SKIP LOCKED、Cancel/Claim、租约恢复、ConcurrencyGuard、页面、权限和脱敏的正式验收。PostgreSQL 16 Runner + TLS、后端全量、专项 race、前端全量和真实浏览器验收均通过。

当前没有代码、安全、权限、事务、重试真实性或自动重发风险阻塞项。允许进入 INT-005 SyncTask，但必须遵守本评审冻结的唯一执行链和扩展边界。
