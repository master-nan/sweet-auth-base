# Sweet Platform Integration Sync V1 冻结评审

## 1. 冻结背景和依据

Integration Runtime 与 Retry V1 已分别通过正式验收并冻结。INT-005A 至 INT-005C-2 在唯一执行链之上增加 SyncTask、SyncBatch、Scheduler、Checkpoint、顺序切片和 registered Consumer。INT-005D 以 `4604d00a7f69591a80b76ed8b685c64c6ee6fc9b` 为实现基线完成 PostgreSQL 16、双 Runner、TLS、Consumer、race、前端和真实浏览器验收。

正式证据见 [IntegrationSyncAcceptanceReport.md](IntegrationSyncAcceptanceReport.md)，设计依据见 [IntegrationSyncDesign.md](IntegrationSyncDesign.md)。本评审只冻结已经验收通过的 Sync V1，不把 HR 映射、补数、并行切片或工作流纳入当前能力。

## 2. 冻结范围

本次冻结：

- `IntegrationSyncTask` 版本、状态和技术配置边界。
- `IntegrationSyncBatch` 技术运行实例和结果摘要。
- `IntegrationSyncRunner` 的 manual/cron 调度、多实例领取和生命周期。
- `IntegrationSyncCoordinator` 的窗口、Lookback、顺序切片和状态协调。
- SyncExecutionInputPlan 与 Integration Application Service 的 Execution 生成边界。
- timestamp/none Checkpoint 的连续推进规则。
- `SyncResultConsumerRegistry`、Consumer 交付协议和业务成功边界。
- Sync/Retry 联动、租约预算、权限、DTO、Audit、日志和页面边界。

Runtime 与 Retry 冻结对象继续受保护：`IntegrationExecution`、`IntegrationLog / Attempt`、`ExecutionInputSnapshot`、`IntegrationRuntimeLimits`、`RetryPolicySnapshot`、`RetryDecision`、`CredentialProvider`、`TransportClient`、`IntegrationExecutionEngine` 和 `IntegrationWorkerRunner`。

## 3. 唯一运行链

```text
Application Service / IntegrationSyncRunner
  -> IntegrationSyncBatch
  -> IntegrationSyncCoordinator
  -> IntegrationExecutionService
  -> IntegrationExecution + RetryPolicySnapshot
  -> IntegrationWorkerRunner
  -> Attempt + ExecutionInputSnapshot
  -> CredentialProvider
  -> TransportClient
  -> RetryDecision
  -> SyncResultConsumer
  -> Attempt + Execution 收敛
  -> Coordinator Checkpoint / Slice / Batch 收敛
```

Sync 只组织 Execution 并观察其结果。Sync 包不得直接 HTTP、读取 Credential、实现 Retry、修改 `next_run_at`、覆盖 Attempt 或直接写 Execution 状态。

## 4. SyncTask 版本模型

一个逻辑任务由稳定 `task_code` 标识，技术版本由 `task_code + version` 唯一。状态仅 draft、enabled、disabled；只有 draft 可编辑，enabled/disabled 的技术变化必须创建新版本。同一 code 同时最多一个 enabled 版本。

Task 固定 ExternalSystem、InterfaceDefinition 明确版本及 Consumer code/version。接口或 Consumer 新版本不自动漂移已有 Task。Retry 继续由 InterfaceDefinition 的 RetryPolicySnapshot 决定，Task 不提供 Retry 配置覆盖。

## 5. SyncBatch

Batch 表示一次 Task 的技术运行实例，状态仅 created、running、succeeded、failed。Batch 冻结 Task、接口、Consumer、trigger、逻辑窗口、Checkpoint、Lookback 和 slice 计划摘要。

V1 不使用 BatchItem 或独立 Checkpoint 表；Execution 以 `sync_batch_id + sync_slice_no` 直接关联。Batch 不保存完整响应、输入或业务对象，不等于 Organization 的业务 Batch。

同 task_code 只允许一个 active Batch。`trigger_key`、scheduled key 和 active partial unique 既是幂等约束，也是 manual/cron 与多实例竞争的最终防线。

## 6. Scheduler、Manual 与 Cron

SyncRunner 独立于 IntegrationWorkerRunner，只发现到期 Task、创建 Batch 并协调 Batch；它不执行远程调用。生产默认 `enabled=false`，显式启用后支持 Start/Run/Stop/Status、panic 恢复和优雅关闭。

Cron 固定五段式，timezone 为 IANA 名称。日历计算按 Task timezone，`scheduled_at`、窗口、Checkpoint 和数据库字段统一 UTC。missed schedule 固定 `coalesce_one`，停机恢复只产生一个合并 Batch，不逐个补历史触发点。

手工运行只允许 enabled Task，使用当前 Checkpoint 和数据库当前 UTC 时间，不接受自定义窗口、补数、Dry Run、Checkpoint 或 Execution 输入。manual 与 scheduled 竞争同一个 active Batch 约束。

## 7. PostgreSQL 多实例语义

到期 Task 领取使用短事务和 `FOR UPDATE SKIP LOCKED`。到期判断显式使用 `CURRENT_TIMESTAMP AT TIME ZONE 'UTC'`，不依赖会话时区。创建 Batch、推进 Task schedule 字段和唯一约束在同一事务内完成。

数据库唯一约束保证同 Task/同 scheduled time、同 trigger key 和同 task_code active Batch 不能重复。双 Runner 只能一个获胜；冲突方按稳定幂等语义退出，不产生第二个 Batch。

## 8. Checkpoint

V1 仅支持 none 和 timestamp。timestamp 首版从 `initial_checkpoint_at` 启用；新版本在启用事务内继承同 code 最新有效 Checkpoint，draft 创建时不得复制为永久运行起点。

Checkpoint 代表连续完成的逻辑时间边界。推进必须同时满足：当前 Slice Execution succeeded、Consumer business success、Slice 连续、Task code/version/revision 匹配。推进使用 Task 行锁与 revision；重复协调不重复推进，陈旧 Coordinator 不得推进新版本。

Lookback 只扩展请求起点，不回退逻辑 Checkpoint。中间 Slice 失败时 Checkpoint 最多到失败 Slice 前一边界，不能越过失败区间。

## 9. Window 与 Lookback

timestamp 逻辑窗口为 `[checkpoint, scheduled_for/database_now)`。请求第一片可从 `checkpoint - lookback` 开始，后续 Slice 不重复应用 Lookback。重复数据由业务 Consumer 的稳定源 ID 幂等处理。

空窗口允许零 Execution 并成功收敛；start > end 为非法窗口并失败。时间值按 UTC 存储和比较。

## 10. 顺序时间切片与 stop-on-failure

Slice 编号从 1 开始并按时间升序。Coordinator 一次只创建当前 Slice，不预创建全部 Execution，不并行执行。当前 Slice 为 created/running/retry_waiting 时 Batch 保持 running；只有 succeeded 且 Consumer 成功后才推进 Checkpoint并创建下一 Slice。

failed 或 cancelled 固定 stop-on-failure，Batch failed，后续 Slice 不创建。V1 不支持 continue-on-failure、失败片补跑、DAG 或并行 Slice。

## 11. Execution 生成与幂等

Coordinator 必须调用 Integration Application Service。SyncExecutionInputPlan 只生成契约声明的静态参数和窗口 binding，随后进入正式 ExecutionInputSnapshot 校验、规范化与 Hash；禁止模板、脚本、SQL、任意 Header、Credential 或客户端输入旁路。

PostgreSQL partial unique `(sync_batch_id, sync_slice_no)` 保证崩溃恢复和重复协调只得到一个 Execution。Execution 冻结 Sync window、Consumer、接口版本、RetryPolicySnapshot 和远端幂等能力。普通 Execution 的 Sync 字段保持空。

## 12. Registered Consumer

Consumer 由服务端静态注册，以 code/version 唯一定位；不允许动态加载、反射方法、脚本、SQL 或插件。Registry 元数据冻结 Content-Type、最大响应大小、最大耗时和支持的 Checkpoint mode。

响应 body 只在调用栈内交给 Consumer，不长期保存在 Sync、Execution、Attempt、DTO、Audit 或日志。Request 不含 Credential、Authorization、Cookie、Token、完整 Header 或数据库 Model。

业务 Consumer 自己定义业务事务、校验、映射和业务幂等。Integration 不包裹业务数据库事务，也不直接写 Organization 数据。

## 13. 技术成功与业务成功

HTTP 2xx 只是技术响应成功，不足以让 Sync Execution succeeded。只有注册 Consumer 返回业务成功后，Engine 才完成 Execution succeeded；业务失败、panic 或 timeout 均为 confirmed `business_processing_failed`，Execution failed，且默认不进入 retry_waiting。

Consumer 业务事务已提交但 Execution 完成失败时，平台不得伪造 succeeded 或自动重发远端请求。业务 Consumer 必须以 source stable ID 和 Sync 上下文实现幂等，以承受受控重复交付。

## 14. Retry 联动

技术失败是否重试的唯一真值仍是 RetryDecision。Execution 为 retry_waiting 时 Batch 保持 running、Checkpoint 不推进、下一 Slice 不创建。Retry 成功且 Consumer 成功后，Coordinator 才继续。

Sync 不解析 Retry-After、不计算 Backoff/Jitter、不修改 `next_run_at`、不新增 Retry Worker。Retry 最终失败、租约恢复 failed + unknown 或 Execution cancelled 时，Batch 按 stop-on-failure 收敛。

## 15. 租约与事务边界

Sync Consumer 的安全租约预算为：

```text
required_lease = interface_timeout
               + consumer_max_duration
               + completion_margin
               + claim_margin
```

Task 启用和 Execution 运行前均复核。超预算配置稳定拒绝，不静默截断 Consumer，不新增无审计续租 goroutine。

Task claim/Batch 创建、Execution claim/Attempt 创建、结果完成和 Checkpoint 推进各自是短事务。Cron 等待、HTTP、Credential、Consumer 和轮询不处于数据库事务中；Consumer 业务事务由 Consumer 自己管理。

## 16. 权限、页面、DTO 与脱敏

SyncTask 使用 query、detail、create、edit、create_version、enable、disable、run；SyncBatch 使用 query、detail。权限来自 `sys_menu_button` 和 Casbin，不按角色名硬编码。V1 技术对象不伪造 Organization Data Permission。

Batch 到 Execution 的查询必须同时满足 Execution query；详情跳转还需 detail，后端继续执行 Execution Data Permission。前端无权限时不发请求，直达 Sync 路由稳定拒绝。

页面只提供 Task 配置、启停、版本、手工运行和 Batch 查询。禁止 Batch cancel、Checkpoint 修改、补数、自定义窗口、Dry Run、重新运行 Batch、立即重试、Payload 查看、Consumer 脚本和在线调试。

DTO、Audit 和日志只返回安全摘要，禁止 Response Body、ExecutionInputSnapshot、HR 数据、Credential、Authorization、Cookie、Token、密文、Consumer 内部错误和数据库错误。Scheduler/Consumer 不伪造管理员 AuditSubject。

## 17. V1 明确不支持

Sync V1 不支持 Batch cancel、高级补数、Dry Run、自定义时间范围、并行 Slice、DAG、continue-on-failure、cursor Checkpoint、Response Artifact、异步 Consumer、HR 业务实现、OAuth、高级查询和前端一致性治理。

这些能力不得通过直接写数据库状态、复制 Runner/Engine、动态脚本或绕过 registered Consumer 临时实现。

## 18. Organization HR 扩展边界

INT-006 Organization HR 只能提供：

- 明确 code/version 的服务端注册 Consumer。
- HR 响应解析、字段校验和业务错误分类。
- Organization 字段映射和业务事务。
- stable source ID、任职关系和依赖数据的业务幂等。
- Organization 自己的业务 Batch/Record 事实，并与 Integration batch/execution 建安全关联。

Organization 不得自行执行 HTTP、读取 Integration Credential、实现 Scheduler/Retry/Checkpoint、修改 `next_run_at`、直接创建或推进 IntegrationExecution、覆盖 Attempt，或持久化 Runtime 禁止的完整 Response Artifact。

## 19. 最终冻结结论

**冻结结论：通过冻结。**

INT-005D 已完成 SyncTask、Batch、manual/cron、数据库 UTC、Checkpoint、Lookback、顺序切片、Execution 幂等、registered Consumer、技术/业务成功、Retry 联动、租约、多实例、权限和脱敏的正式验收。

当前没有代码、安全、权限、事务、调度唯一性、Checkpoint 连续性或业务成功真实性阻塞项。允许进入 INT-006 Organization HR，但必须遵守本文冻结的唯一同步链和 Consumer 扩展边界。
