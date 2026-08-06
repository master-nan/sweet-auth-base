# Sweet Platform Integration Runtime 第一期正式验收报告

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| 文档目的 | 复核 Integration Runtime 第一期真实实现，记录自动化、PostgreSQL 与浏览器验收结果，并判断是否具备正式冻结条件 |
| 验收范围 | IntegrationExecution、IntegrationLog/Attempt、Execution Engine、常驻 Worker、HTTP Transport、Credential Provider、Runtime 管理页面 |
| 验收日期 | 2026-08-06 |
| 首次验收基线 | `6412875ab0a603a777f3fb16bcc6c5e671a25f4a`（实现集成运行管理页面） |
| INT-003B-7R 复验基线 | `d6a28b5a4b98521c44097e103bc54aa2754c64f1`（统一集成运行参数与租约契约） |
| INT-003B-7D 整改基线 | `44ad3b2fd77c69ad2a500c21fcb7dff6db518fbb`（完成集成运行时复验与冻结） |
| 验收环境 | macOS arm64、Go 1.26.2、Node.js 22、PostgreSQL 16、Redis 6.2.7、Docker Compose 本地环境 |
| 适用阶段 | Integration Runtime 第一期验收，不包含 Retry Worker、SyncTask/SyncBatch、OAuth Token 流程和业务同步规则 |

报告提交记录以 Git 历史为准。本报告以当前仓库真实代码、实际测试和本地浏览器操作为依据，不沿用历史 Task 回复作为通过证据。

## 2. 验收结论摘要

**INT-003B-7R 最终结论：不通过。**

首次验收识别的五项阻塞中，状态机唯一入口、配置与 Transport 上限、租约安全余量、Execution 与 Attempt 权限边界已经通过本次代码、测试、PostgreSQL 与浏览器复核。受控输入快照的结构、契约、服务端 Hash 和 Worker 重建代码也已经存在。

但真实 PostgreSQL 16 + 常驻 Runner + 本地 TLS Server 复验发现新的核心阻塞：输入快照以 JSONB 保存后，PostgreSQL 会按 JSONB 语义重新序列化字节；Worker 当前却要求数据库读取的原始字节长度和内容与创建时规范化字节完全相等。因此合法快照在真实 PostgreSQL 往返后被稳定判定为 `execution_input_too_large`，Execution 在发出 HTTP 前收敛为 `failed`。这意味着 INT-003B-7B 的“Worker 可独立重建并执行真实请求”结论未成立。

本 Task 按正式规则不修改该核心快照语义，也不生成冻结评审文档。详细复验证据见第 16 节；第 3 至 15 节保留首次验收和历次整改记录，用于追溯结论变化。

## 3. 当前实现链路

当前 Worker 执行主链路为：

```text
IntegrationWorkerRunner
  -> IntegrationExecutionEngine.ClaimCreatedExecutions
  -> PostgreSQL 行锁 / SKIP LOCKED + 租约 + running Attempt
  -> CredentialProvider.Resolve
  -> TransportClient.Execute
  -> CompleteAttemptAndExecution
  -> IntegrationLog / IntegrationExecution 终态
```

正确实现的边界包括：

- Worker 和 Engine 使用标准 `context.Context`，不依赖 Gin。
- 领取与 Attempt 创建位于短事务内；Credential 解析和 HTTP 位于事务外；结果收敛使用新的短事务。
- Credential Provider 只在调用栈内产生一次性认证材料，不向 Controller 提供读取秘密的入口。
- Transport 不读取数据库、不解释业务规则，并保留 URL、SSRF、Header、超时和响应大小安全边界。
- 常驻 Worker 默认关闭，仅在服务端明确配置后启动；`retry_waiting` 不会被一期 Worker 自动再次领取。

## 4. Execution 与 Attempt 验收

### 4.1 已通过能力

| 验收项 | 结果 | 依据 |
| --- | --- | --- |
| created 原子领取 | 通过 | Engine/Repository 测试及 PostgreSQL 并发领取专项 |
| 领取时创建 running Attempt | 通过 | 领取、Attempt 唯一约束和事务回滚测试 |
| Attempt 序号单调增加 | 通过 | Repository 领取与恢复测试 |
| 成功调用收敛 succeeded | 通过 | 本地 TLS Server 完整 Engine 调用测试 |
| 429、500、503 分类并进入 retry_waiting 候选 | 通过 | 本次补充 Engine 回归断言 |
| 配置或 Credential 失败不发 HTTP | 通过 | Engine 配置失败测试和 Provider 测试 |
| 租约丢失、revision 冲突和重复完成拒绝 | 通过 | Repository 与 Engine 测试 |
| 过期租约恢复为 failed + unknown | 通过 | Engine 租约恢复测试 |
| 历史 Attempt 不覆盖 | 通过 | 唯一约束和完成原语测试 |

完整调用测试覆盖 Basic、API Key、Bearer Token 的解析和注入，使用本地测试 Server，不访问公网。该测试完成 Worker 领取、Credential 解析、Transport 执行、Execution 终态与 Attempt 持久化闭环。

### 4.2 阻塞问题

#### 4.2.1 管理 API 绕过 Worker 与 Attempt

当前 Router 暴露 `start`、`complete`、`fail` 管理接口，Controller 直接调用状态 Service。该路径只校验状态和 revision，不校验租约所有者，也不创建或完成 Attempt。因此授权调用者能够把 `created` Execution 直接推进到 `running`，并进一步推进到 `succeeded` 或 `failed`，形成没有真实 HTTP 调用和 Attempt 的成功记录。

这与正式设计中“`created -> running` 只能由 Worker 原子领取”和“终态必须通过 Attempt + 租约完成”的规则冲突，属于安全与审计一致性阻塞项。

#### 4.2.2 缺少可执行输入快照

当前 Execution 创建请求和数据库模型只保存 `input_hash`，Engine 构造 TransportRequest 时没有 Execution 的 Path、Query、Header 或 Body 输入。异步 Worker 因此只能执行完全静态、无动态输入的接口，也无法验证 `input_hash` 与实际请求输入一致。

正式设计已经要求受控输入快照或受控引用，但具体大小上限、敏感分级、加密方式和留存期仍列为实施前待确认项。该项需要先补充设计决策，再实现和重新验收；本 Task 不以保存明文 Payload 的方式临时处理。

## 5. Worker 验收

| 验收项 | 结果 | 说明 |
| --- | --- | --- |
| 默认关闭 | 通过 | Docker 开发配置为 `enabled: false`，页面显示中性“未启用” |
| 启停与重复 Start | 通过 | Runner 生命周期测试 |
| 可取消轮询、非零间隔和错误退避 | 通过 | Runner 轮询与恢复测试 |
| 实例并发上限 | 通过 | 并发边界测试与 race 测试 |
| 单任务 panic 恢复 | 通过 | Runner panic 测试 |
| 优雅关闭和超时 | 通过 | Runner shutdown 测试 |
| 自动领取 created | 通过 | Runner 自动执行测试 |
| 不领取 retry_waiting | 通过 | Runner 状态边界测试 |
| 定期恢复过期租约 | 通过 | Runner 恢复调度测试 |
| WorkerStatus 并发读取 | 通过 | WorkerStatus race 测试 |

### 5.1 租约时长风险

Engine 默认租约为 2 分钟，Transport 允许的整体请求上限也是 2 分钟。当前没有为响应读取、Attempt 完成和 Execution 完成事务预留安全余量。接近超时上限的合法调用可能在本地仍执行时失去租约，导致结果进入 unknown 或由恢复流程提前收敛。

该项在正式启用常驻 Worker 前必须统一配置校验：租约应大于接口请求上限，并包含明确的完成余量。

## 6. Credential Provider 验收

| 场景 | 结果 |
| --- | --- |
| Basic 正常解析 | 通过 |
| API Key 正常解析 | 通过 |
| Bearer Token 正常解析 | 通过 |
| OAuth Client 稳定拒绝 | 通过 |
| Credential 不存在 | 通过，安全失败 |
| 跨系统或接口系统不匹配 | 通过，安全失败 |
| draft、disabled、revoked | 通过，安全失败 |
| expired | 通过，安全失败 |
| 密文损坏、主密钥缺失、解密失败 | 通过，安全失败 |
| 轮换后使用新版本且不回退历史秘密 | 通过 |
| 并发 Resolve | 通过 race |

Provider 复用现有 AES-GCM 安全存储能力，没有复制新的加密实现。CredentialMaterial 不支持 JSON 暴露，String/GoString 不输出秘密，不进入数据库、Context 或全局缓存。Controller 不存在读取秘密的 API。

## 7. Transport 验收

| 场景 | 结果 |
| --- | --- |
| HTTPS 正常请求 | 通过 |
| 开发模式显式允许 HTTP | 通过 |
| 非法 Method、完整 URL、路径逃逸 | 通过，拒绝 |
| loopback、link-local、云元数据、未批准私网 | 通过，拒绝 |
| DNS 解析后校验与 DNS Rebinding | 通过，拒绝 |
| 自动重定向 | 通过，默认拒绝 |
| Header 黑名单与认证隔离 | 通过 |
| Query 标准编码和 JSON Body | 通过 |
| 请求超时与 Context 取消 | 通过 |
| Content-Length 超限和流读取超限 | 通过，完整失败 |
| Content-Type 非法 | 通过，拒绝 |
| 429、500、其他远端错误 | 通过，结构化分类 |
| 响应 Hash 与敏感 Header 隔离 | 通过 |
| 并发调用 | 通过 race |

Transport 使用自建受控 `http.Client`，不使用无边界 `http.DefaultClient`。测试全部使用本地 TLS Server 或 Stub，不访问公网。

### 7.1 配置上限冲突

InterfaceDefinition 当前允许最大 300 秒超时和 100 MiB 响应；Transport 最大只接受 120 秒和 64 MiB。配置中心可以启用 Transport 必然判定为 `invalid_config` 的配置。该问题必须通过统一配置契约解决，不能让运行时静默放宽安全限制。

## 8. Runtime 页面人工验收

### 8.1 实际执行方式

在本地 Docker 环境中使用浏览器真实登录管理员账号，逐项打开页面。为了检查有数据时的详情与导航，临时建立一条不含 Payload 和秘密的 Execution/Attempt 验收记录；检查完成后已经从本地数据库删除。

### 8.2 结果

| 页面或交互 | 结果 |
| --- | --- |
| 集成中心菜单顺序 | 通过：外部系统、接口定义、集成凭证、执行记录、调用日志 |
| 执行记录列表 | 通过，分页表格和空状态正常 |
| Worker 当前实例状态 | 通过，未启用显示为中性状态 |
| Worker 零时间显示 | 本次修复后通过，显示 `-` |
| Execution 独立详情页 | 通过，独立 ID 路由和页签正常 |
| Attempt 列表与跳转 | 通过，可从 Execution 详情进入调用日志 |
| Attempt 详情弹框 | 通过，显示受控摘要，不显示 Payload、Token 或 Authorization |
| 调用日志列表 | 本次修复后通过，不再返回系统异常 |
| 深色模式 | 通过，Execution、Log、Worker 状态区域可读且布局正常 |
| Worker 状态权限失败对列表的影响 | 本次修复：Worker 状态读取失败不再阻断 Execution 列表初始化 |

### 8.3 页面与权限遗留

1. Execution 详情直接携带 Attempt 摘要，未独立校验 `integration_log_detail` 权限，需明确“执行详情是否包含 Attempt 摘要”的权限契约。
2. Execution 和 Log 页面尚未完整开放设计要求中的精确编号、系统、接口、错误分类和时间范围筛选，AdvancedQuery 也未接入。
3. 部分 Runtime 页面文案仍为硬编码中文，尚未完整进入平台 i18n。
4. Execution 详情和 Log 详情展示字段少于现有 DTO 可安全提供的技术摘要。
5. 前端详情入口主要依赖后端拒绝，没有完全按详情按钮权限隐藏入口；不构成当前数据越权，但交互不完整。

## 9. Data Permission、Casbin 与脱敏

### 9.1 已确认

- Execution query/detail Controller 使用统一 Data Permission Runtime，不复制 Grant、Policy 或 Resolver 逻辑。
- 解析失败会停止查询，不回退全量。
- Runtime 菜单和按钮由 `sys_menu_button` 与 Casbin Seed 管理，不按角色名称硬编码。
- Response DTO 为字段白名单，不直接返回 Model。
- API 和页面不返回完整 Payload、Authorization、Cookie、API Key、Token、密文、nonce 或安全存储引用。
- Worker 自动执行记录技术事实，不伪造管理员 AuditSubject。

### 9.2 未完成验证或缺口

- Log query/detail、Worker status 的 Controller 测试未覆盖完整 Casbin 拒绝矩阵。
- Runtime Resource 当前按真实模型边界使用明确策略，没有伪造组织 Ownership；但 `not_applicable` 场景仍需在后续业务归属设计完成后复核。
- Execution 详情中的 Attempt 摘要存在前述跨功能权限边界问题。

## 10. 本次修复的问题

### 10.1 调用日志分页查询 500

原实现把 `Preload("Execution")` 提前附加到分页查询，公共 Count 在没有 Model 的计数语句上继承 Preload，PostgreSQL 返回 `model value required when using preload`。本次调整为：

1. 先对完全相同的权限和筛选条件执行 Count。
2. 再仅对 rows 查询附加 Execution Preload。
3. 补充 rows、total 和关联预加载回归测试。

修复后浏览器调用日志页面正常返回空列表或验收记录，不再显示“系统异常”。

### 10.2 Worker 零时间展示

未启用 Worker 的零值时间此前被浏览器格式化为公元 1 年。本次新增 Runtime 日期格式化边界，将空值、Go 零时间和非法时间统一显示为 `-`，并补充前端单元测试。

### 10.3 Worker 状态权限与 Execution 列表解耦

Execution 列表和 Worker 状态是两个独立权限资源。页面现在先完成 Execution 列表加载；Worker 状态查询失败不再使列表初始化和分页监听整体中断。前端 Worker 时间字段同时调整为可空契约。

### 10.4 429、500、503 收敛回归

补充 Engine 测试，明确 429、500 和 503 均按已冻结分类形成 confirmed 的重试候选，并收敛到 `retry_waiting`；一期仍不自动重试。

## 11. 自动化测试结果

### 11.1 后端

```text
cd backend && go test ./... -count=1
结果：通过，所有 backend 包通过。
```

### 11.2 Race

```text
go test -race ./internal/integration/... ./repository/impl ./service ./initialize -count=1
结果：通过。

go test -race ./controller -run Integration -count=1
结果：通过。
```

全量 `backend/controller` race 另会触发现有 Organization/Gin 测试的历史竞争问题，位置为员工账号绑定查询链；该问题不在 Integration Runtime 修改范围内，Integration Controller 专项 race 已单独通过。

### 11.3 PostgreSQL

本次实际使用 Docker PostgreSQL 16 DSN 执行，并非跳过：

```text
TestIntegrationExecutionPostgreSQLClaimUsesRowLock
结果：通过，两个并发 Worker 只有一个成功领取同一 created Execution。

TestIntegrationRuntimeSchemaPostgreSQLConstraints
结果：通过，Migration、CHECK、外键、Execution/Attempt 唯一约束及幂等重复执行通过。
```

### 11.4 前端

```text
yarn test
结果：27 个测试文件、104 个测试通过。

yarn lint
结果：通过。

yarn typecheck
结果：通过。

yarn build
结果：通过；仅存在既有大 chunk 提示，不影响构建产物。
```

## 12. 当前限制

1. Retry Worker、自动退避和 `retry_waiting` 调度未实现。
2. SyncTask、SyncBatch 和业务转换未实现。
3. OAuth Client Token 获取、缓存和刷新未实现。
4. Worker 状态只表示当前应用实例，不是集群汇总。
5. 常驻 Worker 默认关闭，尚不应在生产启用，直至本报告阻塞项关闭。
6. 结果 `unknown` 不会被自动重放，符合安全失败原则。
7. TransportAuthentication 虽未进入 DTO 或日志，但尚缺显式 String/JSON 防泄露测试，建议补强。
8. Attempt 尚未持久化 Method、目标 Host 摘要、请求大小/Hash 和最终重试资格摘要等完整诊断字段。
9. 历史 Organization/Gin Controller race 仍存在，与 Integration Runtime 专项无关。

## 13. 阻塞项与后续动作

### 13.1 阻塞项

1. 删除或重新设计可绕过 Worker/Attempt 的 `start`、`complete`、`fail` 管理 API，并补充 Casbin 与状态机回归。
2. 冻结并实现 Execution 受控输入快照或受控引用，确保 Worker 可独立重建请求且 Hash 可验证。
3. 统一 InterfaceDefinition 与 Transport 的超时、响应大小契约。
4. 建立租约时长大于请求上限加完成余量的配置校验。
5. 明确 Execution 详情携带 Attempt 摘要时的 Log 功能权限规则并实现。

### 13.2 非阻塞后续项

1. 补齐 Runtime 高级筛选、页面 i18n 和安全摘要展示。
2. 补齐 Log/Worker Controller 的 Casbin 与 Data Permission 测试矩阵。
3. 增加从提交、常驻 Runner、真实 Provider/Transport 到最终查询 DTO 的单一端到端测试。
4. 补充 TransportAuthentication 显式防序列化和防 String 泄露边界。

## 14. 最终结论

**不通过。**

当前实现已经具备可复用的 Runtime 技术基础，且主要安全组件和数据库原子原语通过了实际测试；本次发现的页面查询缺陷也已修复。然而管理状态 API、异步输入快照、配置上限、租约余量和 Attempt 权限边界仍会影响执行真实性、审计完整性或生产安全，不能作为非阻塞限制处理。

因此本次不冻结 Integration Runtime 第一期。完成阻塞项后，应基于本报告场景重新执行 INT-003B-7，只有完整调用链、状态机唯一入口、输入可重建、权限边界和生产配置契约同时通过后，方可进入正式冻结。

## 15. 阻塞项整改状态补充

### 15.1 INT-003B-7A 状态机唯一入口

2026-08-06 已完成本项整改：

1. 管理 Router、Controller、Application Service 和请求 DTO 不再暴露 `start`、`complete`、`fail` 命令。
2. 对应 `sys_menu_button`、角色按钮关联和 Casbin 路径由 Seed 幂等清理，不保留失效授权入口。
3. 创建后的 Execution 保持 `created`；`created -> running` 仅由 Worker 领取原语在创建 running Attempt 的同一事务内完成。
4. `running -> succeeded / failed / retry_waiting` 仅由 Engine 通过租约所有者、revision、running Attempt 和完成事务收敛。
5. 管理端仅保留 `created`、`retry_waiting` 的取消命令；`running` 和终态均稳定拒绝。

回归测试确认：旧状态路由不存在，无租约或 Attempt 不能完成 Execution，Engine 正常领取和完成路径保持通过。

### 15.2 INT-003B-7A Attempt 权限边界

Execution 详情响应已移除内嵌 Attempt 技术摘要，只保留 Execution 自身白名单字段与 `current_attempt`。Attempt 必须通过独立调用日志接口查询，并分别执行调用日志功能权限与 Data Permission `query/detail` 校验。

前端仅在拥有 `integration_log_query` 时请求 Attempt 列表；仅在拥有 `integration_log_detail` 时提供详情入口。无调用日志权限时不发起 Attempt 请求，并显示中性权限提示。调用日志路由参数同样受详情按钮权限约束，不会先请求再根据 403 隐藏内容。

### 15.3 本项实际验证

```text
cd backend && go test ./... -count=1
结果：通过。

go test -race ./controller ./service ./repository/impl ./internal/integration/... -count=1
结果：Service、Repository、Integration Engine 通过；全量 Controller 仍命中第 11.2 节记录的既有 Organization/Gin 测试竞争。

go test -race ./controller -run Integration -count=1
结果：通过。

cd frontend && yarn test
结果：29 个测试文件、111 个测试通过。

yarn lint
yarn typecheck
yarn build
结果：全部通过；构建仅有既有产物体积提示。
```

### 15.4 验收结论保持

本补充只关闭第 13.1 节中的第 1 项和第 5 项，不改变本报告“不通过”的正式结论。仍待独立 Task 完成：

1. Execution 受控输入快照或受控引用。
2. InterfaceDefinition 与 Transport 的超时、响应大小契约统一。
3. 租约时长大于请求上限并包含完成余量的配置校验。

全部阻塞项关闭后，由 INT-003B-7R 重新执行正式验收并决定是否冻结。

### 15.5 INT-003B-7B Execution 输入快照设计与存储

2026-08-06 已完成本项整改。正式设计已在 `IntegrationRuntimeDesign.md` 冻结版本 1 输入快照：只包含接口契约允许的 Path、Query、普通 Header 和 JSON Body，不包含 Host、完整 URL、Credential、Authorization、代理、TLS、脚本、SQL、模板或表达式。

InterfaceDefinition 新增版本化 `input_contract`，定义参数编码、位置、数据类型、必填性、最大长度、是否多值和敏感标记。路径占位符与 Path 契约必须精确一致；已启用版本不能原地修改技术契约，创建下一版本时完整复制契约。

V1 采用 PostgreSQL JSONB 保存规范化、非敏感输入，数据库同时保存快照版本和大小摘要。当前没有接入 KMS，也没有复用 Credential AES-GCM 密钥语义；契约声明为敏感的字段、敏感控制名称、二进制、Multipart 和超过限制的输入均被拒绝。Execution API 和管理页面不提供完整快照读取能力。

服务端限制为：Path 32 项/4 KiB、Query 64 项/16 KiB、Header 16 项/8 KiB、JSON Body 256 KiB、完整快照 384 KiB、JSON 深度 16、单数组 256 项、字段总数 256、单字符串 4 KiB。

### 15.6 规范化、幂等与 Worker 请求重建

Execution 创建 Service 现在加载明确的 InterfaceDefinition 版本，对输入执行契约校验和规范化，再对“明确接口版本 + 规范化快照”计算服务端 SHA-256。客户端 `input_hash` 只作为可选比对值，不能覆盖服务端结果。

规范化规则包括：Header 名称小写、Query 多值稳定排序、JSON 对象稳定键顺序、JSON 数组保持顺序。`request_id`、`trace_id`、Worker ID、Credential 秘密和时间字段不参与 Hash。相同幂等键且规范化输入相同返回原 Execution；输入不同返回稳定冲突；并发创建最终只保留一条记录。

Worker 在 Credential 解析和 Transport 调用之前完成：加载快照、校验版本和大小、按冻结契约重新规范化、重算 Hash、校验规范化字节与 Hash，然后重建 Path、Query、普通 Header 和 JSON Body。Credential Provider 最后注入认证。快照缺失、损坏或 Hash 不一致时不发送 HTTP，并按配置错误完成 Attempt。

Repository 领取条件只接受版本 1 且大小有效的待执行记录。Migration 将没有快照的历史 `created` / `retry_waiting` Execution 收敛为 `failed`，不伪造空输入继续调用。

### 15.7 DTO、日志与页面边界

Execution 详情只新增输入摘要：快照版本、大小、Path/Query/Header 数量和是否包含 Body。DTO 白名单测试确认不返回 Path、Query、Header 和 JSON Body 原值。

动态 Engine 测试实际携带业务 Path、Header 和 Body，并捕获结构化日志确认这些原值及认证 Token 不进入日志。Audit 继续只记录 Execution 创建和合法取消，不记录输入快照。

当前 Runtime 页面没有创建 Execution 的动态输入表单，本 Task 未增加自由 JSON、任意 Header 或完整 URL 编辑器。管理页面仍只展示输入 Hash 与安全摘要；动态契约配置和受控提交入口属于后续产品化范围。

### 15.8 INT-003B-7B 实际验证

```text
cd backend && go test ./... -count=1
结果：通过。

go test -race ./internal/integration/... ./service ./repository/impl -count=1
结果：通过。

SWEET_TEST_POSTGRES_DSN=<本地 Docker PostgreSQL 16>
go test ./migrate -run TestIntegrationRuntimeSchemaPostgreSQLConstraints -count=1
结果：通过；实际验证 JSONB 字段、CHECK、Migration 重复执行和无效契约/快照约束。

go test ./repository/impl -run TestIntegrationExecutionPostgreSQLClaimUsesRowLock -count=1
结果：通过；真实 PostgreSQL 行锁领取回归未受输入快照条件影响。
```

前端代码未修改，因此本整改未重复执行 Yarn。前端边界通过现有页面代码审计确认：当前没有创建 Execution 的自由输入入口，也没有完整 Payload 查看能力。

### 15.9 当前剩余阻塞项

本补充关闭第 13.1 节中的第 2 项，但不改变本报告“不通过”的正式结论。当前仅剩 INT-003B-7C 范围内两项：

1. 统一 InterfaceDefinition 与 Transport 的超时、响应大小契约。
2. 校验 Worker 租约时长大于最大请求时长并包含状态完成安全余量。

完成上述整改后，仍须由 INT-003B-7R 重新执行完整 Runtime 验收并决定是否冻结。

### 15.10 INT-003B-7C 统一运行参数契约

2026-08-06 已完成最后一组代码阻塞整改。配置中心、Transport、Execution Service、Engine 与 Runner 现统一使用服务端 `IntegrationRuntimeLimits`：接口请求超时为 1 至 120 秒，默认 30 秒；响应限制为 1 KiB 至 64 MiB，默认 10 MiB。接口草稿创建/编辑、启用、Execution 创建、Worker 执行前和 Transport 构造请求均复核同一边界，不存在“配置通过、Transport 必然 invalid_config”的合法配置。

一期采用固定平台租约。计算公式为：

```text
minimum_lease_duration
= max_request_timeout 120 秒
  + completion_margin 30 秒
  + claim_safety_margin 15 秒
= 165 秒
```

默认租约为 180 秒，最大为 600 秒。Runner 与 Engine 启动时均校验该预算，余量不足稳定拒绝启动；本项未引入续租 goroutine。恢复器仍只处理真实过期的 `running` Execution，并收敛为 `failed + unknown`，不会自动重发。

PostgreSQL Migration 会将历史超限 `enabled` 接口安全停用，保留原 timeout、response_limit 和版本技术事实；新增条件 CHECK 阻止超限版本重新启用。当前本地数据库审计未发现超限记录。

前端接口定义表单同步为最大 120 秒和 65,536 KiB（64 MiB），新建、编辑草稿与创建版本复用同一表单校验；后端仍是最终安全边界。

### 15.11 INT-003B-7C 验证与结论边界

本次实际验证结果：

1. `cd backend && go test ./... -count=1`：通过。
2. `go test -race ./internal/integration/... ./service ./repository/impl ./initialize -count=1`：通过。
3. PostgreSQL 16 专项：统一上限 Migration、CHECK、重复执行、超限启用版本停用、`SKIP LOCKED` 领取、租约未到期不恢复及真正过期后恢复均通过。专项测试期间识别出 Go 侧时间解析与 PostgreSQL 会话时区可能不一致的问题，已将完成与恢复的租约时点判断收口为数据库条件更新，避免应用时区参与租约真值判断。
4. `cd frontend && yarn test`：30 个测试文件、112 项测试全部通过。
5. `yarn lint`、`yarn typecheck`、`yarn build`：通过；构建仅保留仓库既有的大体积 Chunk 提示。

本项补充记录统一契约、迁移、租约和测试证据，但不把本报告正式结论提前改为通过。INT-003B-7A、7B、7C 识别的代码阻塞项现均已完成整改；是否通过与冻结仍由 INT-003B-7R 基于完整链路重新验收决定。

## 16. INT-003B-7R 重新验收

### 16.1 复验范围与基线

本次以 `d6a28b5a4b98521c44097e103bc54aa2754c64f1` 为验收基线，逐项审计 INT-003B-7A、7B、7C 的真实代码、Router、Migration、Seed、前端页面和测试。环境使用 Docker PostgreSQL 16、Redis、后端与前端，并使用本地 TLS Server 构造不访问公网的真实 Transport 场景。

本节是当前正式结论依据。首次验收和整改章节继续保留，但不得覆盖本节实际结果。

### 16.2 原五项阻塞复核

| 原阻塞项 | 复验结果 | 证据 |
| --- | --- | --- |
| 状态机唯一入口 | 通过 | 管理端 `start`、`complete`、`fail` Router、Controller、前端 API、按钮和 Casbin 记录均已清理；实际 HTTP 请求均返回 404；Worker 领取与 Engine 完成原语继续通过测试 |
| 受控输入快照 | **不通过** | 结构化契约、服务端 Hash 和重建代码存在，但合法快照经 PostgreSQL JSONB 往返后被 Worker 拒绝，真实请求未发出 |
| 配置与 Transport 上限 | 通过 | timeout 1 至 120 秒、默认 30 秒；response_limit 1 KiB 至 64 MiB、默认 10 MiB；配置、Execution、Engine 与 Transport 复用统一限制并通过边界测试 |
| Worker 租约安全余量 | 通过 | 120 秒请求上限 + 30 秒完成余量 + 15 秒领取余量 = 165 秒最小安全租约；默认 180 秒、最大 600 秒；PostgreSQL 数据库时间条件测试通过 |
| Execution 与 Attempt 权限边界 | 通过 | Execution 详情不内嵌 Attempt 技术内容；Attempt 通过独立 Log API 加载，并分别执行功能权限和 Data Permission query/detail；前端无权限时不发请求 |

### 16.3 真实端到端调用结果

本次临时验收用例实际建立：

1. 独立 PostgreSQL Schema、ExternalSystem、Bearer Credential 和带 Path、Query、普通 Header、JSON Body 契约的 InterfaceDefinition。
2. 通过 Application Service 提交 Execution，并在提交请求 Context 结束后启动真实常驻 Runner。
3. Runner 自动领取 `created` Execution，在同一领取事务创建 running Attempt。
4. 使用本地 TLS Server 和受控 Endpoint Policy 准备真实 Transport，不访问公网。

实际结果：Runner 成功领取，但 Engine 在 Credential 解析和 HTTP 调用前将 Execution 收敛为 `failed`，Attempt 错误分类为 `configuration`，reason_code 为 `execution_input_too_large`，安全摘要为“集成执行输入快照不可用”；本地 TLS Server 未收到请求。

根因是 `integration_execution.input_snapshot` 使用 PostgreSQL JSONB。创建时保存的是应用规范化 JSON 字节的长度与 SHA-256；JSONB 读取时会改变键顺序或空白等原始字节表达。`LoadExecutionInputSnapshot` 当前先要求读取字节长度等于 `input_snapshot_size`，随后还要求重新规范化字节与数据库读取原字节完全一致，合法语义因此被当作损坏快照拒绝。

该问题同时影响成功链路、Credential 最后注入、真实 Path/Query/Header/Body 到达远端以及同输入幂等的端到端证明。它属于 ExecutionInputSnapshot 持久化与完整性校验核心语义，本次没有通过放宽校验或临时改存储来掩盖。

### 16.4 Worker、租约与异常链复核

除输入快照阻断真实 HTTP 外，以下能力通过现有自动化、race 和 PostgreSQL 专项：

- 默认 `enabled=false`，显式启用、重复 Start、可取消轮询、错误退避、并发上限、panic 恢复和优雅关闭。
- `retry_waiting` 不被一期 Runner 自动领取。
- SKIP LOCKED 并发领取、Attempt 唯一约束、lease_owner 与 revision 条件完成。
- 最大合法请求预算不会在租约到期前被恢复；真正过期后收敛为 `failed + unknown`，不自动重发。
- 租约与恢复关键条件使用数据库时间，不依赖应用时区。

Credential 失效、429、5xx、超时、响应超限、SSRF 和 Hash 篡改的组件与 Engine 专项继续通过；但由于成功端到端链被 JSONB 快照阻断，不能把这些专项等同于完整 Runtime 成功验收。

### 16.5 Transport 与 Credential 回归

Transport 的 HTTPS、SSRF、DNS Rebinding、重定向拒绝、Header 黑名单、超时、Content-Length 和流读取上限、Content-Type、429/5xx、响应 Hash 与并发 race 均通过。Transport 没有放宽 120 秒和 64 MiB 的硬上限，也没有静默截断。

Credential Provider 的 Basic、API Key、Bearer、OAuth 不支持、disabled/revoked/expired、密文损坏、主密钥缺失、轮换和不回退历史秘密均通过。Controller 仍无秘密读取入口。本次补充 `TransportAuthentication` 的 String、GoString 和 JSON 防泄露边界：格式化只返回脱敏标识，JSON 序列化稳定拒绝。

### 16.6 权限与脱敏复核

- Execution 与 IntegrationLog 分别使用独立 Resource 的 query/detail Data Permission；Resolver 失败不会放行。
- Execution 详情只返回 `current_attempt` 和 Execution 安全摘要，不直接返回 HTTP 状态、Worker、Credential、错误正文或请求/响应 Hash 等 Attempt 技术内容。
- 前端只有同时具备 Log query 时才请求 Attempt 列表，具备 Log detail 时才显示详情入口。
- 管理页面和 API 不返回完整 Path、Query、Header、JSON Body、Authorization、Cookie、Token、API Key、密文、nonce 或安全存储引用。
- 实际无集成权限账号登录后不渲染“集成中心”，直接访问执行记录路由进入 404。

### 16.7 浏览器人工验收

使用本地浏览器真实登录管理员账号，实际检查外部系统、接口定义、集成凭证、执行记录、Execution 详情、Attempt 独立加载、调用日志和 Worker 当前实例状态。结果如下：

1. 集成中心菜单顺序正确：外部系统、接口定义、集成凭证、执行记录、调用日志。
2. Worker 未启用显示中性状态，未被误报为故障。
3. 使用临时安全 Fixture 验证独立 Execution 详情路由、Attempt 列表、调用日志跳转和详情弹框；检查完成后数据库记录已全部删除。
4. 页面无 start、complete、fail、重试、在线调试或完整 Payload 入口。
5. 深色模式下 Execution、Log 和 Worker 状态区域正常可读。
6. 本次修复前端遗漏：后端已有 `input_summary`，前端类型和详情页未展示。修复后页面只展示快照版本、大小、Path/Query/Header 数量、是否包含 JSON Body 和输入 Hash，不展示输入原值。
7. 使用无集成权限账号验证：侧栏不显示集成中心，直接输入执行记录地址进入 404。

### 16.8 自动化与 PostgreSQL 结果

```text
SWEET_TEST_POSTGRES_DSN=<本地 Docker PostgreSQL 16>
SWEET_REQUIRE_POSTGRES_TESTS=true
cd backend && go test ./... -count=1
结果：通过，标准测试套件全部通过，PostgreSQL 门控未跳过。

go test -race ./internal/integration/... ./repository/impl ./service ./initialize -count=1
结果：通过。

go test -race ./controller -run Integration -count=1
结果：通过。

PostgreSQL Migration/CHECK/重复执行专项
结果：通过。

PostgreSQL SKIP LOCKED、唯一约束、行锁领取和租约专项
结果：通过。

真实 PostgreSQL + 常驻 Runner + 本地 TLS Server 临时验收用例
结果：失败；execution_input_too_large，远端未收到请求。
```

前端实际结果：

```text
yarn test
结果：30 个测试文件、113 项测试通过。

yarn lint
yarn typecheck
yarn build
结果：全部通过；仅保留仓库既有的大体积 Chunk 提示。
```

标准测试套件通过并不能覆盖本次发现的 JSONB 原始字节往返差异；因此不能以 `go test ./...` 通过替代真实端到端失败结论。

### 16.9 本次允许范围内修复

1. 为 `TransportAuthentication` 增加 String、GoString 与 JSON 防泄露实现和测试。
2. 前端补齐 Execution 输入快照摘要类型、详情展示和组件测试。

没有修改 JSONB 快照存储、Hash、Worker 重建或状态机核心语义。

### 16.10 复验结论与冻结判断

**最终验收结论：不通过。**

状态机、权限、统一上限和租约四项原阻塞已经关闭，但真实 PostgreSQL 环境中的输入快照无法被 Worker 成功重建，导致一期 Runtime 不能完成最基本的常驻 Worker 成功调用。这是代码与执行真实性阻塞项，不属于环境或后续能力限制。

**冻结结论：不冻结。** 本次不生成 `IntegrationRuntimeFreezeReview.md`，也不允许进入 INT-004 Retry。应先通过独立整改 Task 冻结 JSONB 语义校验方案，例如统一以重新规范化后的语义字节计算大小和 Hash，或选择可保持规范化字节的存储类型；整改必须补充真实 PostgreSQL 往返和常驻 Runner 成功调用回归，再重新执行 INT-003B-7R。

## 17. INT-003B-7D JSONB 输入快照语义校验整改

### 17.1 根因与设计修订

INT-003B-7R 的失败根因已确认：创建端以应用规范化 JSON 字节计算 `input_snapshot_size` 和 `input_hash`，而 Worker 读取 PostgreSQL JSONB 后，错误地把数据库返回文本长度直接与语义大小比较，并要求返回字节与创建字节逐字相等。JSONB 只保证 JSON 语义，会调整对象键顺序、空白和等价转义表达，因此原校验把合法往返误判为损坏。

正式设计已修订并冻结：

1. `input_snapshot_size` 是版本 1 快照重新规范化后的语义字节长度，不是 JSONB 驱动返回字节长度。
2. 创建和 Worker 加载复用同一个规范化入口；Header 名称小写、Query 多值稳定排序、JSON 对象稳定序列化、数组顺序保留。
3. Hash 信封继续固定包含 InterfaceDefinition 明确版本和规范化快照，不包含数据库 ID、request_id、trace_id、Worker ID 或 Credential。
4. Worker 严格解码 JSONB、检查版本、复核接口契约、重新规范化，再比较语义大小与 Hash；只有全部通过才构造 TransportRequest。
5. 新增独立的 512 KiB JSONB 读取防御上限 `max_snapshot_storage_bytes`；384 KiB 仍是规范化语义快照上限，两者不混用。

### 17.2 实现与错误语义

原创建端和 Worker 端的相似流程已收口到单一规范化实现。Worker 不再比较 JSONB 原始字节，也不再用原始字节长度判断 `input_snapshot_size`。

本次稳定区分：

- `execution_input_storage_too_large`：数据库返回 JSONB 超过独立存储防御上限。
- `execution_input_semantic_too_large`：重新规范化后的输入超过语义上限或结构复杂度上限。
- `execution_input_size_mismatch`：重新计算的语义大小与 Execution 摘要不一致。
- `execution_input_hash_mismatch`：语义大小一致但接口版本或输入语义发生变化。
- `execution_input_invalid`：JSONB 不能严格解码或结构不合法。
- `execution_input_contract_mismatch`：快照不再符合冻结接口契约。

### 17.3 数据库与 Migration 审计

`integration_execution.input_snapshot` 继续使用 PostgreSQL JSONB，不改为 TEXT 或 BYTEA。现有 `input_snapshot_size` CHECK 只验证 0 至 393,216 的语义摘要范围和版本关系，没有把 `octet_length(input_snapshot::text)` 等同于语义大小，因此无需变更数据库结构。

现有 Execution 创建逻辑原本就保存应用规范化后的大小和 Hash，合法开发记录不需要迁移重算；修复后会在 Worker 加载时重新规范化并验证。历史无快照记录继续由既有幂等 Migration 安全收敛为 failed，无法解码、版本不支持、摘要不一致或契约不匹配的记录均不会发送 HTTP，也不伪造空快照或修改已完成历史事实。

PostgreSQL Migration 专项新增验证：JSONB 返回文本长度与 `input_snapshot_size` 不同的合法记录可以通过 CHECK；Migration 连续执行仍通过。

### 17.4 PostgreSQL 16 语义往返与篡改测试

使用 `SWEET_TEST_POSTGRES_DSN` 和 `SWEET_REQUIRE_POSTGRES_TESTS=true` 实际执行 PostgreSQL 16 专项，未跳过：

1. 创建包含 Path、Query、普通 Header 和 JSON Body 的版本 1 快照，输入包含对象键顺序、空白、Unicode 转义和 Query 多值顺序差异。
2. 保存为 JSONB 后读取，实际返回字节内容和长度均不同于创建时规范化字节。
3. Worker 重新规范化后得到相同 semantic_size 和 input_hash，TransportRequest 重建成功。
4. 分别修改 Path、Query、Header、Body，均稳定触发 Hash 完整性失败。
5. 修改大小摘要触发 `execution_input_size_mismatch`；修改快照版本稳定拒绝；超过存储防御上限稳定触发 `execution_input_storage_too_large`。
6. 同语义 JSON 键顺序、Query 多值顺序和 Header 大小写继续得到相同 Hash；JSON 数组顺序变化继续得到不同 Hash。

### 17.5 常驻 Runner + TLS 端到端结果

本次把上一轮失败链路固化为自动化 PostgreSQL 门控测试并实际通过：

1. 在独立 PostgreSQL Schema 创建 ExternalSystem、Bearer Credential 和带 Path、Query、Header、JSON Body 契约的 InterfaceDefinition。
2. 通过 Application Service 提交 Execution，随后结束原提交 Context。
3. 启动真实 `IntegrationWorkerRunner`，没有直接调用 `RunOnce`。
4. Runner 自动领取 created Execution，并在领取事务内创建 running Attempt。
5. Engine 从 JSONB 读取快照，重新执行契约校验、规范化、semantic_size 与 Hash 校验。
6. 校验通过后 Credential Provider 才解析并注入 Bearer 认证。
7. 本地 TLS Server 实际收到正确的 Path、排序后的 Query、普通 Header、JSON Body 和 Authorization。
8. Attempt 与 Execution 均收敛为 succeeded，凭证摘要正确记录，完整输入和秘密未进入 DTO、Audit 或日志。

### 17.6 自动化结果与结论边界

```text
SWEET_TEST_POSTGRES_DSN=<本地 Docker PostgreSQL 16>
SWEET_REQUIRE_POSTGRES_TESTS=true

cd backend && go test ./... -count=1
结果：通过；PostgreSQL 门控测试实际执行，未跳过。

go test -race ./internal/integration/... ./service ./repository/impl ./initialize -count=1
结果：通过。

PostgreSQL JSONB 往返、Migration、CHECK、唯一约束、行锁和 Runner + TLS 专项
结果：全部通过。
```

本 Task 没有修改前端 DTO 或页面，因此未重复执行 Yarn。

INT-003B-7D 已关闭当前已知的 JSONB 语义校验代码阻塞，但按 Task 约束，本节不把第 16.10 节的总体验收结论改为通过，也不生成冻结评审。当前没有已知剩余代码阻塞项；仍需执行精简的 INT-003B-7R2 最终复验，复核后再决定是否通过和冻结。
