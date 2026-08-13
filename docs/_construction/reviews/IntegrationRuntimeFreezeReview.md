# Sweet Platform Integration Runtime 第一期冻结评审

## 1. 冻结背景和依据

Integration Runtime 第一期在首次验收和 INT-003B-7R 复验中先后暴露状态入口、输入快照、运行上限、租约余量、权限隔离和 PostgreSQL JSONB 语义校验问题。INT-003B-7A 至 7D 已逐项整改，INT-003B-7R2 以 `4b5302c269243a8e3b4987edc20dd6de7e540be2` 为基线完成 PostgreSQL 16、常驻 Runner + TLS、race、前端和真实浏览器复验。

正式验收依据为 `docs/_construction/reviews/IntegrationRuntimeAcceptanceReport.md`。本评审只冻结已经通过验收的 Runtime 第一期核心，不把后续 Retry、同步任务或业务集成能力纳入当前版本。

## 2. 冻结范围

本次冻结以下核心对象和边界：

- `IntegrationExecution`：一次逻辑调用及其冻结配置、输入摘要和状态。
- `IntegrationLog / Attempt`：一次真实调用尝试及其技术事实。
- `ExecutionInputSnapshot`：受控 Path、Query、普通 Header 和 JSON Body 输入。
- `IntegrationRuntimeLimits`：配置、Transport、Engine 和 Runner 的统一运行参数契约。
- `CredentialProvider`：运行期凭证校验、解密和一次性认证材料适配。
- `TransportClient`：受控 HTTP/HTTPS 调用、安全限制和结构化结果。
- `IntegrationExecutionEngine`：领取、Attempt 编排、执行和状态收敛。
- `IntegrationWorkerRunner`：应用生命周期、轮询、实例并发和租约恢复调度。

## 3. 唯一运行链

```text
Application Service
  -> IntegrationExecution
  -> IntegrationWorkerRunner
  -> IntegrationExecutionEngine / Claim
  -> IntegrationLog / running Attempt
  -> ExecutionInputSnapshot 复核
  -> CredentialProvider
  -> TransportClient
  -> Attempt + Execution 原子状态收敛
```

Controller 只负责请求适配和合法管理命令，不执行 HTTP、不读取秘密、不领取任务，也不直接推进运行状态。运行时不存在管理状态旁路、双跑或兼容回退。

## 4. Execution 与 Attempt 语义

Execution 表示一次逻辑调用，保存接口版本、幂等信息、受控输入摘要、当前 Attempt 和最终结果摘要。Attempt 表示一次真实调用尝试，按 `execution_id + attempt_no` 唯一并单调追加，保存开始结束时间、技术状态、HTTP 摘要、错误分类和结果确定性。

Execution 不能覆盖历史 Attempt，Attempt 不能脱离 Execution 和租约独立完成。完整请求、完整响应、Authorization、Token、密文和内部堆栈不属于这两个对象的持久化边界。

## 5. 状态机唯一入口

`created -> running` 只能由 Worker 的原子领取完成，并在同一短事务内建立 running Attempt。`running -> succeeded / failed / retry_waiting` 只能由 Engine 在校验 running Attempt、lease_owner 和 revision 后通过完成事务推进。过期租约恢复只收敛为 `failed + unknown`，不会重发远端请求。

管理端仅保留创建、查询以及对 `created`、`retry_waiting` 的合法取消。`start`、`complete`、`fail` 管理 API 已移除；running 和终态不能通过通用状态更新取消或改写。

## 6. ExecutionInputSnapshot

快照版本 1 只支持接口契约明确声明的 Path、Query、普通 Header 和 JSON Body。客户端不能覆盖 scheme、Host、端口、完整 URL、代理、TLS、Credential、Authorization、Cookie、API Key、Token 或其他认证与 Hop-by-Hop 字段，也不能提交脚本、SQL、模板或表达式。

Application Service 负责加载冻结接口版本、执行契约校验、规范化、计算 Hash 并持久化结构化 JSONB。Worker 不依赖原始 HTTP 请求或 Gin Context，执行前从数据库独立加载并复核快照。

## 7. JSONB 语义校验

PostgreSQL JSONB 只保证 JSON 语义，不保证原始字节、键顺序、空白或等价转义表达。完整性校验因此固定为：严格解码 JSONB，校验快照版本和接口契约，调用与创建端相同的规范化入口，再比较 semantic_size 和 Hash。

`input_snapshot_size` 表示规范化语义字节长度，语义上限为 384 KiB。数据库返回 JSONB 另有 512 KiB 独立存储防御上限；两者具有不同错误分类，不得混用或通过删除校验放宽安全边界。

## 8. semantic_size 和 Hash

规范化规则固定包括 Header 名称小写、Query 多值稳定排序、JSON 对象键稳定排序和 JSON 数组顺序保留。Hash 使用 SHA-256，信封按固定顺序包含明确 `interface_version` 和规范化快照语义字节。

数据库 ID、request_id、trace_id、Worker ID、创建时间和 Credential 秘密不参与 Hash。客户端提交的 input_hash 不是可信来源，只能用于可选比对，数据库始终保存服务端计算结果。

## 9. Runtime Limits

统一运行参数冻结为：接口请求超时最小 1 秒、最大 120 秒、默认 30 秒；响应限制最小 1 KiB、最大 64 MiB、默认 10 MiB。InterfaceDefinition 草稿、启用、Execution 创建、Worker 执行和 Transport 构造均使用同一服务端边界。

Transport 不静默截断，不为兼容配置放宽硬上限。超限接口不能启用，绕过配置产生的不兼容记录在发送 HTTP 前安全失败。

## 10. Worker、租约和并发

一期采用固定平台租约：最大请求时长 120 秒 + completion_margin 30 秒 + claim_safety_margin 15 秒 = 最小安全租约 165 秒；默认租约 180 秒，最大 600 秒。Runner 和 Engine 在启动或执行前校验该契约，客户端不能修改租约。

PostgreSQL `FOR UPDATE SKIP LOCKED` 和条件更新负责多实例唯一领取，进程内并发限制只用于本实例资源控制，不替代数据库原语。租约和恢复条件使用数据库时间。Runner 默认关闭，显式启用后才轮询；停止时停止新领取并按有限超时等待当前任务，未完成任务交由租约恢复，不伪造终态。

## 11. Credential 与 Transport 安全边界

CredentialProvider 支持 Basic、API Key 和 Bearer Token，OAuth Client 稳定返回不支持。Provider 校验系统归属、接口关联、active 状态、有效期、安全引用和密文信封，复用现有 AES-GCM 组件，不提供 Controller 解密入口，不缓存或回退历史秘密。

TransportClient 只接收服务端确认的基础地址、相对路径、受控输入和认证适配结果。默认只允许 HTTPS，并执行 Endpoint Policy、DNS 解析后 IP 复核、DNS Rebinding 防护、重定向拒绝、Header 黑名单、超时、响应大小和 Content-Type 限制。秘密不进入 Context、结果摘要或日志。

## 12. 短事务边界

Execution 创建、Worker 领取与 Attempt 创建、Attempt 与 Execution 完成分别使用短事务。Credential 解密、请求构造、DNS/TLS 和 HTTP 调用全部在事务外执行。Repository 提供原子领取、完成和恢复原语，但不判断凭证、重试资格或 HTTP 业务成功语义。

同一事务对象不得跨 goroutine 使用，Runner 不保存事务 DB，完成事务失败不得返回成功。

## 13. 权限与脱敏

Execution 和 IntegrationLog 分别使用独立 Casbin 功能权限及 Data Permission query/detail。Execution detail 通过不代表 Log detail 通过；无日志权限时前端不请求 Attempt API，后端直接请求仍稳定拒绝且不泄露记录存在性。

Response DTO 仅返回白名单摘要。输入快照默认只展示版本、semantic_size、参数数量、是否含 JSON Body 和 input_hash；Attempt 详情只展示受控技术摘要，不返回 Payload、Header 原值、Authorization、Cookie、Token、API Key、密文、nonce、安全存储引用或堆栈。

## 14. 页面与管理能力

集成中心固定包含外部系统、接口定义、集成凭证、执行记录和调用日志。执行详情使用独立 ID 路由，Attempt 通过独立日志接口加载；Worker 页面只显示当前实例只读状态，不提供启动、停止或修改配置能力。

页面不提供 start、complete、fail、Retry、Payload 查看、在线调试或重新发送 Attempt。动态按钮来自 `sys_menu_button` 和 Casbin，不按角色名称硬编码；深色模式和无权限路由已经通过真实浏览器验收。

## 15. V1 不支持项

Runtime 第一期不支持 Retry Worker、RetryPolicy 配置中心、SyncTask/SyncBatch、HR Organization 业务同步、OAuth Token 获取与刷新、敏感或大型输入、动态输入表单完整产品化、完整 Payload 查看、在线调试代理、集群 Worker 汇总，以及前端高级筛选和完整 i18n 治理。

这些是后续能力边界，不构成本次冻结缺陷。历史 Organization/Gin race 仍由平台治理任务处理，不在本次冻结中宣称已解决。

## 16. Retry 与 SyncTask 扩展边界

INT-004 Retry 只能调度现有 `retry_waiting` Execution，并在同一 Execution 下追加新的 Attempt。它必须继续复用 ExecutionInputSnapshot、Runtime Limits、CredentialProvider、TransportClient、租约领取、结果确定性和完成事务，不能修改历史 Attempt，不能把 unknown 的非幂等写操作直接自动重发。

后续 SyncTask/SyncBatch 只能作为触发、批次和业务结果关联层调用现有 Application Service，不得直接执行 HTTP、读取 Credential 秘密、访问组织表或复制一条新的 Execution 状态机。

## 17. 最终冻结结论

**冻结结论：通过冻结。**

INT-003B-7R2 已关闭首次验收五项阻塞和 JSONB 语义完整性阻塞，后端全量、Integration 专项 race、PostgreSQL 16、常驻 Runner + TLS、前端全量和真实浏览器验收均通过。当前没有代码、安全、权限、事务或执行真实性阻塞项。

Integration Runtime 第一期核心架构正式冻结。后续模块不得修改或绕过本评审保护的八个核心对象；扩展必须沿 Resource、Execution、Attempt、Credential Provider、Transport、Engine 和 Runner 的既有边界进行。允许进入 INT-004 Retry。
