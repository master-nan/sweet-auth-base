# Sweet Platform Integration Sync V1 正式验收报告

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| Task | INT-005D |
| 验收范围 | SyncTask、SyncBatch、SyncRunner、Coordinator、Checkpoint、顺序切片、Execution 生成、Consumer、页面、权限与脱敏 |
| 验收日期 | 2026-08-12 |
| 验收基线 | `4604d00a7f69591a80b76ed8b685c64c6ee6fc9b`（实现集成同步结果交付能力） |
| INT-005A | `b309c7c`（完成集成同步任务体系设计） |
| INT-005B | `fcbecb3`（实现集成同步任务配置中心） |
| INT-005C-1 | `3f5582e`（实现集成同步调度协调能力） |
| INT-005C-2 | `4604d00`（实现集成同步结果交付能力） |
| 验收环境 | macOS arm64、Go 1.26.2、Node.js 22.23.0、Yarn 1.22.21、PostgreSQL 16.14、Redis 6.2.7、Docker Compose、本地 TLS Server |
| 正式设计 | [IntegrationSyncDesign.md](../design/IntegrationSyncDesign.md) |
| 前置冻结 | [IntegrationRuntimeFreezeReview.md](IntegrationRuntimeFreezeReview.md)、[IntegrationRetryFreezeReview.md](IntegrationRetryFreezeReview.md) |

本报告以当前仓库真实代码、Migration、Service、Runner、Registry、页面、PostgreSQL 和测试结果为依据，不以历史 Task 回复代替证据。

## 2. 正式结论

**最终验收结论：通过。**

Sync V1 已在 Runtime 与 Retry 冻结执行链上完成版本化 Task、manual/cron 调度、Batch、timestamp Checkpoint、Lookback、顺序切片、Execution 幂等生成和 registered Consumer 业务结果交付。PostgreSQL 16 强制门控、双 Sync Runner、Integration Worker + TLS + Consumer 端到端、后端全量、专项 race、前端全量和真实浏览器登录态验收均通过。

当前没有重复 Batch、重复 Execution、Checkpoint 错推进、Consumer 失败误报成功、业务失败自动 Retry、Sync 自行 HTTP/Retry、多实例重复调度或敏感响应泄露等冻结阻塞项。允许冻结 Sync V1，并允许进入 INT-006 Organization HR；HR 只能实现注册 Consumer 和业务映射，不得复制 Scheduler、HTTP、Retry、Checkpoint 或 Execution 状态链。

## 3. 唯一同步链

```text
IntegrationSyncRunner
  -> PostgreSQL due Task claim / manual Application Service
  -> IntegrationSyncBatch
  -> IntegrationSyncCoordinator
  -> IntegrationExecutionService.CreateExecution
  -> IntegrationExecution + RetryPolicySnapshot + Sync source
  -> IntegrationWorkerRunner / RetryDecision
  -> TransportClient
  -> SyncResultConsumer
  -> Attempt + Execution 收敛
  -> Coordinator 连续推进 Checkpoint / 下一 Slice / Batch 终态
```

Sync 不执行 HTTP、不读取 Credential、不计算 Retry、不修改 `next_run_at`，也不直接写 IntegrationExecution。所有远程执行继续复用冻结 Runtime/Retry 链。

## 4. SyncTask 配置中心

| 验收项 | 结果 | 说明 |
| --- | --- | --- |
| draft 创建和编辑 | 通过 | `task_code` 创建后不可修改，仅 draft 可编辑 |
| code + version | 通过 | PostgreSQL 唯一约束和 Service 事务保证单调版本 |
| enabled/disabled 只读 | 通过 | 技术变化只能创建新版本 |
| 单 enabled | 通过 | partial unique index 与事务双重保护 |
| active Batch 停用保护 | 通过 | 有 created/running Batch 时拒绝停用 |
| InterfaceDefinition 固定版本 | 通过 | 新接口版本不漂移既有 Task |
| Consumer code/version | 通过 | 启用时验证 Registry 元数据与兼容性 |
| Retry 边界 | 通过 | Task 无 Retry 配置，继续由 InterfaceDefinition 冻结策略 |
| revision | 通过 | 编辑、版本、启停和手工运行均校验乐观锁 |

Schedule 支持 `none` 和五段 cron；六段 cron 与非法 IANA timezone 稳定拒绝。`next_scheduled_at`、`last_scheduled_at` 仅由服务端维护，Cron 按业务 timezone 解释后以 UTC 持久化。DST spring-forward/fall-back 和停机后 `coalesce_one` 均有测试覆盖。

## 5. Scheduler 与多实例

到期扫描使用 PostgreSQL `FOR UPDATE SKIP LOCKED`。候选条件使用 `next_scheduled_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'`，与 UTC `timestamp without time zone` 字段一致，不依赖数据库会话时区。

两个真实 `IntegrationSyncRunner` 同时扫描同一 Task 时只创建一个 Batch；`trigger_key`、`task_code + scheduled_for` 和同 task_code 单 active Batch partial unique 作为最终数据库防线。`next_scheduled_at` 只推进一次，`last_scheduled_at` 与创建的 Batch 一致。manual 与 scheduled 并发也只能一个获得活动 Batch。

## 6. Checkpoint、Window 与顺序切片

- V1 Checkpoint 仅 `none`、`timestamp`。none 模式所有 Checkpoint/窗口切片字段为空。
- 首版 timestamp 使用 `initial_checkpoint_at`；新版本在启用事务内继承同 code 最新有效 Checkpoint，draft 创建时不冻结陈旧值。
- 逻辑窗口为 `[checkpoint, scheduled_for/database_now)`；Lookback 只影响首片请求起点，不改变逻辑 Checkpoint。
- 空窗口产生零 Execution 并成功收敛；非法窗口安全失败。
- Slice 从 1 开始，按 UTC 时间升序，一次只创建当前 Slice；最后一片可以短于 slice duration。
- `created/running/retry_waiting` 保持 Batch running；成功才创建下一片；failed/cancelled 立即 stop-on-failure。
- Checkpoint 只在当前连续 Slice 的技术与 Consumer 业务结果均成功且 Task revision 匹配时推进。重复协调不重复推进。

端到端成功场景两片均成功后 Checkpoint 到达窗口终点；业务失败场景第二片失败后 Batch failed，Checkpoint 保持第一片终点，不创建第三片。

## 7. Execution 生成与幂等

Coordinator 通过 `IntegrationExecutionService.CreateExecution` 创建 Execution，Input Plan 先生成受控参数，再由正式 `ExecutionInputSnapshot` 契约校验、规范化与 Hash。RetryPolicySnapshot 和远端幂等能力仍按 InterfaceDefinition 冻结。

Execution 保存 `sync_batch_id`、`sync_slice_no`、窗口和 Consumer code/version。普通 Execution 的 Sync 字段为空。PostgreSQL partial unique `(sync_batch_id, sync_slice_no)` 保证重复协调与崩溃恢复只得到一个 Execution，未发现 Repository 直接 INSERT Execution 的旁路。

Batch 已提交但 Execution 尚未创建时重启 Coordinator，会恢复创建同一 Slice；不会产生 Execution2。

## 8. Consumer 与技术/业务成功

Registry 以 code/version 唯一注册服务端 Consumer，不支持动态加载、脚本、SQL、反射方法或插件。Task 启用和执行前验证状态、Content-Type、response size、Checkpoint mode、max duration 与租约预算。

`SyncConsumptionRequest` 仅在调用栈内传递 execution、batch、task、slice、window、content type、size、hash 和 body；不包含 Credential、Authorization、Cookie、Token、完整 Header 或数据库对象，body 不长期保存。

真实测试结论：

| 场景 | Execution | Retry | Checkpoint |
| --- | --- | --- | --- |
| HTTP 200 + Consumer 成功 | succeeded | 不触发 | 可连续推进 |
| HTTP 200 + Consumer 业务失败 | failed | 不触发 | 不推进 |
| HTTP 200 + Consumer panic | failed | 不触发 | 不推进 |
| HTTP 200 + Consumer timeout | failed | 不触发 | 不推进 |

业务失败稳定归类 `business_processing_failed`，不是 Runtime 临时技术失败。Test Consumer 具备幂等断言；Consumer 业务事务已提交但 Execution 完成失败时不伪造 succeeded，也不自动重新发送远端请求。

租约预算冻结为 `interface_timeout + consumer_max_duration + completion_margin + claim_margin`。合法值和边界通过，超预算 Task 不能启用；未新增续租 goroutine，也不静默截断 Consumer。

## 9. Retry 联动

真实场景第一片首次 HTTP 503 后，Execution 进入 `retry_waiting`，Batch 保持 running，Checkpoint 不推进且第二片不创建。既有 Integration Worker 到期后自动追加 Retry Attempt，第二次 HTTP 200 且 Consumer 成功；Coordinator 下一轮才推进第一片 Checkpoint 并创建第二片。

Sync 没有解析 Retry-After、计算 Backoff、修改 `next_run_at`、直接再次发送 HTTP 或建立第二套 Retry Runner。Retry 最终失败或 Execution 被合法取消时，Batch 收敛 failed。

## 10. 手工触发

新增 `POST /admin/integration/sync-task/:id/run`。只允许 enabled Task，输入只含 revision；客户端不能提交时间范围、Checkpoint、Dry Run、补数、Execution 输入或 Payload。

Application Service 在短事务内锁 Task，取数据库 UTC 当前时间作为窗口终点，复用定时 Batch 快照构造器，并与 scheduled trigger 竞争同一 active Batch 约束。成功返回安全 Batch DTO，并记录真实管理员 AuditSubject。手工触发只创建 Batch，Coordinator 负责后续 Execution。

## 11. 页面、权限与脱敏

管理员真实浏览器验收确认菜单顺序为：外部系统、接口定义、集成凭证、重试策略、同步任务、同步批次、执行记录、调用日志。

- SyncTask 页面提供列表、详情、draft 创建/编辑、版本、启停和 enabled + `run` 权限下的“运行一次”。受控表单没有自由 JSON、脚本、补数或自定义窗口。
- SyncBatch 页面显示 Batch、Task/接口/Consumer 摘要、触发、状态、逻辑窗口、Checkpoint、切片进度、技术/业务计数、reason 和时间。
- Batch 到 Execution 仅在有 Execution query 权限时请求，使用精确 `sync_batch_id`；详情跳转还需 Execution detail，并继续由后端执行 Data Permission。
- Execution 详情仅显示安全 Sync 业务摘要，不返回 Consumer body。
- 页面没有 Batch cancel、Checkpoint 修改、Dry Run、重新运行 Batch、立即重试、Payload 查看、Consumer 脚本或在线调试。
- 公共深色主题实际切换后页面可读，无文字重叠或独立亮色区域。

无授权账号 `dp_acceptance_ungranted` 实际登录后为 0 个可访问菜单；直达 SyncTask 路由进入 404，不加载同步数据。临时密码字段和缓存已恢复，未留下验收业务数据。

SyncTask 的 query/detail/create/edit/create_version/enable/disable/run 与 SyncBatch query/detail 均来自 `sys_menu_button` 和 Casbin，不硬编码角色。V1 技术对象不伪造 Organization Ownership；Execution/Attempt 仍有独立功能权限和 Data Permission。

DTO、Audit 和结构化日志不返回或记录完整 Response Body、ExecutionInputSnapshot、HR 数据、Credential、Authorization、Cookie、Token、密文、Consumer 内部错误或数据库错误。自动 Scheduler/Consumer 不伪造管理员 AuditSubject。

## 12. PostgreSQL 16 与端到端结果

强制设置 `SWEET_REQUIRE_POSTGRES_TESTS=true`，实际执行 PostgreSQL 16.14，未跳过门控：

| 场景 | 结果 |
| --- | --- |
| Migration 重复执行、CHECK、FK、partial unique | 通过 |
| Task 版本、单 enabled、active Batch | 通过 |
| trigger/scheduled/Execution slice 唯一 | 通过 |
| SKIP LOCKED、数据库 UTC、Checkpoint revision | 通过 |
| 双 Sync Runner | 仅一个 Batch，调用次数一次 |
| manual/scheduled 竞争 | 仅一个 active Batch |
| 两片 Runner + TLS + Consumer | Batch succeeded，Checkpoint 到终点 |
| 503 -> Retry -> 200 -> 下一片 | 通过，首片 Attempts=2 |
| 第二片 Consumer 失败 | Batch failed，Checkpoint 停在第一片 |

完整 E2E 由真实 IntegrationSyncRunner 和 IntegrationWorkerRunner 自动推进，没有在测试中手工推进下一 Slice 或第二次 Retry HTTP。

## 13. 自动化结果

```text
SWEET_TEST_POSTGRES_DSN=postgresql://sweet_admin:***@127.0.0.1:15432/sweet_admin?sslmode=disable
SWEET_REQUIRE_POSTGRES_TESTS=true
cd backend && go test ./... -count=1
结果：通过，PostgreSQL 门控实际执行。

go test -race ./internal/integration/... ./repository/impl ./service ./initialize -count=1
结果：通过。

go test -race ./controller -run 'Sync|Integration' -count=1
结果：通过（backend/controller 4.949s）。

yarn test
结果：通过，36 个测试文件、130 个测试。

yarn lint / yarn typecheck / yarn build
结果：全部通过。
```

Build 仅有既有大 chunk 提示，不影响产物或本次验收。

## 14. 本次发现并修复的问题

1. 缺少正式手工运行闭环：新增权限、Router、Controller、Service、事务、Audit、页面按钮和测试；不接受自定义窗口或输入。
2. PostgreSQL 会话时区不是 UTC 时，`timestamp without time zone` 与裸 `CURRENT_TIMESTAMP` 比较可能提前领取：改为显式数据库 UTC 表达式并增加 Asia/Shanghai 回归。
3. SyncBatch 页面缺少 Task/接口/Consumer、Checkpoint、结果和 Execution 明细：补齐安全摘要与独立权限加载。
4. Batch 到 Execution 缺少精确关联查询：新增 `sync_batch_id` 受控过滤和安全 Sync source DTO。
5. SyncTask 页面在无引用权限时仍预加载 metadata/ref API：改为按动态功能权限和打开表单时加载。
6. 缺少 DST 和 manual/scheduled 竞争验收：补齐确定性测试。
7. 原 E2E 未同时覆盖 Retry 后继续切片与第二片业务失败：扩展为 PostgreSQL + 双 Runner + TLS + Consumer 完整场景。

## 15. 非阻塞限制

V1 不支持 Batch cancel、高级补数、Dry Run、自定义时间范围、并行 Slice、DAG、continue-on-failure、cursor Checkpoint、Response Artifact、异步 Consumer、Organization HR 真实 Consumer、OAuth、高级查询或前端一致性治理。

生产环境当前不注册虚构 Consumer；测试 Consumer 仅存在于正式测试 Registry。浏览器页面没有用伪 Consumer 或伪 Batch 构造成功数据，完整运行真实性由强制 PostgreSQL E2E 证明。既有前端大 chunk 提示与历史 Organization/Gin race 不是 Sync V1 阻塞。

## 16. 最终判断

**验收结论：通过。**

**冻结结论：允许冻结。**

**后续阶段：允许进入 INT-006 Organization HR。**

进入下一阶段后必须遵守 [IntegrationSyncFreezeReview.md](IntegrationSyncFreezeReview.md)：Organization 只实现服务端注册 Consumer、业务校验/映射、业务事务与幂等，不得自行执行 HTTP、Retry、Scheduler、Checkpoint 或 IntegrationExecution 状态推进。
