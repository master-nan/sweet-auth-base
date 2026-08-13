# Integration Sync Source Contract 冻结后变更评审

## 1. 评审信息

| 项目 | 内容 |
| --- | --- |
| Task | INT-006B |
| 变更类型 | Integration Sync V1 冻结后受控兼容扩展 |
| 评审对象 | `SyncExecutionInputPlan` V2 Source Window Contract |
| 原冻结基线 | `IntegrationSyncFreezeReview.md` |
| 业务依据 | `OrganizationHRSyncDesign.md` 与脱敏 HR 源分析 |
| 结论 | 通过；允许使用 V2 `lower_bound_only`，但不关闭源时间与响应规模生产 Gate |

## 2. 变更原因与真实证据

真实 HR timestamp 接口只声明一个 `{time}` 下界参数，没有结束时间参数。脱敏采集证明公司接口在当前样本环境使用包含式下界，并观察到响应未按 `changeTime` 排序；Swagger 和样本仍不能证明时区、正式精度及同秒完整性。

冻结的 V1 要求 `window_start_binding` 和 `window_end_binding` 都存在。把逻辑终点绑定到不存在或无关的参数会伪造源契约，因此真实接口无法合法配置。该问题属于平台 Source Contract 表达缺口，不能通过 Organization 自建 HTTP、Scheduler、Checkpoint 或 Consumer 私改 Execution 规避。

## 3. 向后兼容方案

V1 保持逐字语义兼容：

- `version=1`；
- 不允许 `window_mode`；
- timestamp 模式必须同时具有 start/end binding；
- 物化请求同时写入真实起止参数。

新增 `version=2`，`window_mode` 必填且只能为：

1. `bounded_window`：与 V1 相同，必须双绑定；
2. `lower_bound_only`：必须只有 start binding，出现 end binding 即拒绝。

旧 Task JSONB 不迁移、不重写。DTO 接受 1/2，PostgreSQL CHECK 接受 V1 或合法 V2；统一规范化器仍执行 InterfaceDefinition input_contract 和 ExecutionInputSnapshot 最终复核。前端 timestamp Task 使用 V2 受控选择，编辑 V1 时推导为 `bounded_window`。

## 4. Checkpoint 与逻辑窗口

两种模式都由 Sync Coordinator 冻结逻辑半开区间 `[logical_start, logical_end)`，Execution 继续保存 `sync_window_start/end`。第一片 Lookback 只改变请求下界，不改变逻辑 Checkpoint 起点。

`lower_bound_only` 的 HTTP 请求只发送：

```text
changed_since = logical_start - first_slice_lookback
```

`logical_end` 不进入 HTTP。Consumer 必须使用受控请求中的 WindowStart/WindowEnd 和已确认的 source change timestamp 分类：

- `< logical_start`：仅稳定键幂等重放；
- `logical_start <= value < logical_end`：当前 Slice 可处理；
- `value >= logical_end`：future，禁止写业务对象和当前 Slice 成功记录。

Coordinator 仍只在 Execution succeeded、Consumer success、切片连续且 Task revision 匹配后推进到 `logical_end`。下载到 future 数据不构成提前推进依据。

## 5. 响应大小限制

`lower_bound_only` 不保证响应有上界。它不能：

- 限制源端返回到 `logical_end`；
- 通过缩短逻辑 Slice 必然减少 Body；
- 解决初始化或历史积压；
- 绕过 InterfaceDefinition、Consumer 或 Transport Response Limit；
- 提高冻结的 64 MiB Transport 上限；
- 引入 Response Artifact、磁盘 Payload 或旁路流式处理。

超限继续按 Runtime 安全失败。Catch-up 与初始化必须以实际响应量做准入；人员按公司分区仅允许未来经服务端批准的固定 static input，不支持任意 ID、动态 fan-out 或 Organization Scheduler。

## 6. 安全与架构边界

本变更不修改 Runtime 状态机、Attempt、RetryDecision、RetryPolicy、CredentialProvider、Transport、IntegrationExecution 状态语义、Scheduler 架构、Checkpoint 连续推进或 ConcurrencyGuard。Sync 仍只创建/观察 Execution，Organization 仍只能实现注册 Consumer。

输入计划不接受模板、脚本、SQL、任意 Header 或认证字段。Consumer 收到的一次性 Body 不持久化；future 记录不进入 Organization 业务表；日志不记录原始 Source ID、人员 Payload 或 Credential。

## 7. 数据库与验证

PostgreSQL 迁移幂等替换原 V1-only JSONB CHECK，新约束明确接受：

- version 1 且无 `window_mode`；
- version 2 且 `window_mode` 为固定白名单。

应用规范化器进一步校验绑定组合和 InterfaceDefinition 契约。测试覆盖：

1. V1 bounded 不回归；
2. V2 bounded 双绑定；
3. V2 lower-bound 单绑定；
4. lower-bound 伪 end 拒绝；
5. 逻辑 end 继续进入 Execution/Consumer；
6. Lookback/current/future 半开区间分类；
7. future 不落 Organization；
8. Checkpoint 只推进 logical end；
9. PostgreSQL JSONB CHECK 与 Migration 重复执行；
10. PostgreSQL 16 + SyncRunner + WorkerRunner + TLS + Organization test consumer 完整链路；
11. Transport Response Limit 不变。

## 8. 冻结结论

本受控扩展通过的条件是上述 PostgreSQL 与 E2E 门控全部实际通过。通过后，V2 `lower_bound_only` 成为 Integration Sync 正式 Source Window Contract；V1 继续冻结不变。

该结论只关闭单下界的“平台无法表达”问题。BIP ID 永久性、changeTime 权威性/时区/精度/同秒完整性、人员大响应、主任职、`sendpost`、兼职 ID、开放任职日期等 P0 继续打开。任何生产 Organization Consumer 仍须按各自 Gate 决定是否注册和启用。
