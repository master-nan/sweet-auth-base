# PFCR-S2T 平台测试架构与测试资产最终治理

> 状态：PASS
> 基线：`33680c8679eb0234c6a6ffb29c1822e7fe15683c`
> 审计日期：2026-08-21
> 性质：TEST ASSET GOVERNANCE / CLEANUP

## 1. 当前工作区与审计口径

实施前 `git status --short` 和 `git diff --stat` 均为空。本文只治理测试资产、测试夹具、测试门禁和长期测试规则，不改变生产 API、数据库结构、权限、查询或页面行为。

统计口径：

- Backend Test 数量按顶层 `func TestXxx` 统计，子测试不重复计数。
- Frontend 用例数量采用 Vitest JSON 报告中的 `numTotalTests`。
- Source-string 测试指直接读取生产源码后依赖字符串、正则或文件结构得出结论的测试。
- PostgreSQL 专项按文件名、`PostgreSQLDSN`/强制门禁和真实驱动综合识别；SQLite 只用于非 PostgreSQL 专属语义。

## 2. Before 统计

| 资产 | Before |
| --- | ---: |
| Backend `*_test.go` | 210 |
| Backend 顶层 Test | 925 |
| Backend Benchmark | 0 |
| Backend TestMain | 6 |
| Backend 测试 LOC | 50,066 |
| 文件名含 postgres 的专项测试 | 9 |
| 显式 PostgreSQL Gate 入口文件 | 2 |
| SQLite 测试文件 | 64 |
| E2E 标识文件 | 2 |
| Frontend `*.spec.ts` / `*.test.ts` | 72 |
| Frontend Vitest 用例 | 243 |
| Frontend 测试 LOC | 8,346 |
| Frontend Source-string 文件 | 8 |
| Script Test 文件 | 4 |
| Script Test 用例 | 44 |
| Script Test LOC | 924 |
| 显式 Backend `time.Sleep` | 7 |
| 显式 Skip（含 PostgreSQL Gate helper） | 1 |

实施前基线验证：`go test ./... -count=1` 通过；`yarn test` 通过 72 files / 243 tests。

## 3. 分类标准

| 分类 | 决策标准 |
| --- | --- |
| KEEP | 直接保护业务行为、安全边界、权限、事务、状态机、并发、数据库约束、Migration 或稳定跨端契约 |
| MERGE | 同一行为域被拆成多个微型测试文件，合并后不损失定位能力 |
| REWRITE | 保护目标长期有效，但当前测试依赖源码字符串、实现细节或重复脆弱夹具 |
| DELETE | 仅服务历史 Task/Freeze、只验证文件结构/字符串、已被更高层行为完整覆盖，或测试已删除兼容能力 |

## 4. 全量文件分类

以下采用“完整集合减例外”的无遗漏分类：基线 210 个 Backend 测试文件全部为 **KEEP**，但下表标记为 REWRITE/MERGE/DELETE 的文件除外；基线 72 个 Frontend 测试文件全部为 **KEEP**，但下表例外除外；4 个 Script Test 全部为 **KEEP**。因此基线中的每个测试文件均有且只有一个分类。

### 4.1 Backend

**KEEP（200 个文件）**

按长期保护域归组：

- Auth / Security / Middleware：密码、Token、Refresh race、Logout、Lockout、HMAC、Casbin、CORS、Recovery、安全审计。
- Integration：Execution/Attempt 状态机、Retry、Lease、Claim、Recover、Transport、Credential、Sync Window、Checkpoint、Consumer、PostgreSQL E2E。
- Data Permission：Resolver、Ownership、Dimension、Grant、Policy、Rule、Subject、DataScope、Preflight、deny-by-default。
- Metadata / Generalization：Decimal、SmallInt、1..11 Contract、Storage/Logical/Display、Relation、DDL、Runtime Metadata、Query Capability。
- Query Scheme：PERSONAL/PUBLIC/ROLE/PAGE_DEFAULT、Default、Revision、Binding、Metadata Validation、IDOR、Data Permission AND。
- Organization HR：SourceKey、Adapter、Normalizer、Structure、Cycle、Parent Late Arrival、Position、Employee、Resigned、Checkpoint、Gate。
- File：Upload、Chunk、Merge、Quick Upload、Access Purpose、Delete/Compensation、Ownership、Concurrency。
- Migration / PostgreSQL：真实升级路径、幂等、CHECK/FK/partial unique/JSONB/DDL。
- Repository：通用底座行为及领域 unique/lock/batch/scope/revision/special query，不保留仅重复 GORM CRUD 的独立文件。
- Controller / DTO：HTTP binding、权限、稳定错误映射和高风险字段白名单。

**REWRITE（9 个文件）**

| 文件 | 当前问题 | 最终方向 |
| --- | --- | --- |
| `backend/internal/test/harness_test.go` | 仅由自身使用的 helper 自证测试没有产品防回归价值 | 删除 `WithRollback`、`NewHTTPServer`、`AssertIdempotent` 及其自证测试，保留仍有生产测试调用的 helper |
| `backend/internal/test/runtime_test.go` | `ConfigureGinTestMode` 用例只重复验证 Gin setter | 删除第三方 setter 断言，保留 PostgreSQL Gate 解析行为 |
| `backend/migrate/permission_projection_test.go` | 混有“旧函数名不再存在”的阶段性源码字符串断言 | 删除该实现细节断言，保留权限投影、Route Coverage 和历史 Migration 行为 |
| `backend/service/context_boundary_test.go` | 依赖源码文本匹配判断 Service 不导入 Gin | 改为 Go AST/import guard，避免注释和格式造成误判 |
| `backend/service/metadata_boundary_test.go` | 依赖文件名和源码字符串判断 Metadata 边界 | 改为 Go AST/import guard，直接检查禁止依赖 |
| 4 个 Integration/HR polling 测试文件 | 各自用 `time.Sleep` 实现相同轮询 | 复用有界 `testutil.Eventually`，保留状态机和 HR 行为断言 |

**MERGE：无。** Backend 当前按包/领域定位价值高，未发现通过合并文件能真实降低维护成本的候选。

**DELETE（1 个文件）**

| 文件 | 原因 |
| --- | --- |
| `backend/enum/enum_test.go` | 仅重复断言 FieldType 数值；更强的 `internal/metadata/value_contract_test.go` 已同时保护 1..11 唯一连续契约和 Metadata 行为 |

`backend/model/test_main_test.go` 最终判定 **KEEP**：`backend/internal/test` 依赖 `model`，反向复用会形成 Go import cycle；包级一行 Gin Mode 初始化比引入新共享层更清晰。

### 4.2 Frontend

**DELETE（6 个文件）**

| 文件 | 原因 |
| --- | --- |
| `frontend/src/pages/frontend-consistency.spec.ts` | FE 阶段 Freeze 源码字符串清单，不验证运行行为 |
| `frontend/src/pages/organization/employee/Index.spec.ts` | 只查找初始化、API 名和路由字符串 |
| `frontend/src/pages/organization/position/Index.spec.ts` | 只查找初始化、API 名和路由字符串 |
| `frontend/src/pages/organization/sync-batch/Index.spec.ts` | 只查找初始化、详情和错误按钮字符串 |
| `frontend/src/pages/organization/sync-error/Index.spec.ts` | 只切片生产源码并断言表达式文字 |
| `frontend/src/components/Display/StatusChip.spec.ts` | 只断言 Quasar `QChip` props 透传，未保护状态映射或交互，且已被使用方行为覆盖 |

**MERGE（3 个输入文件合并为 2 个目标文件）**

| 原文件 | 最终文件 | 原因 |
| --- | --- | --- |
| `frontend/src/pages/organization/organization-detail-mode.spec.ts` | `frontend/src/pages/organization/organization-detail.spec.ts` | 同属 Organization Detail 导航契约 |
| `frontend/src/pages/organization/organization-detail-route.spec.ts` | `frontend/src/pages/organization/organization-detail.spec.ts` | 与 Detail Mode 总是共同维护，合并后仍可独立定位用例 |
| `frontend/src/utils/query-state-decimal.spec.ts` | `frontend/src/utils/query-scheme-state.spec.ts` | Decimal/SmallInt payload 规范化属于同一个 Query Scheme snapshot/normalize 契约 |

**REWRITE（12 个文件）**

| 文件 | 当前问题 | 最终方向 |
| --- | --- | --- |
| `frontend/src/pages/query-scheme/EligiblePageMatrix.spec.ts` | 每页重复断言组件/import 字符串 | 收敛为唯一一个精炼 Architecture Guard，只守 17 个 Scope 页面统一入口 |
| `frontend/src/pages/develop/dictionary/Index.spec.ts` | 读取 `.vue` 源码判断 EXEMPT 和主从流程 | 改为挂载页面，验证主从查询、372px 工作区和无 Query Scheme 控件 |
| 8 个 `frontend/src/pages/integration/*/Index.spec.ts` Query Scheme mock | 重复 20 余行相同 mock shape | 保留文件与行为用例，改用一个领域测试夹具，不合并页面测试 |
| `frontend/src/components/Table/StandardTableToolbar.spec.ts` | 仍用旧“保存方案常驻”文案描述顺序 | 对齐 S2UX 后 Selector 内保存入口的真实布局语义 |
| `frontend/src/components/QueryScheme/QuerySchemeSelector.spec.ts` | 重复 mount setup，且缺少 Dirty 切换、DEGRADED/INVALID 的交互保护 | 收敛 mount helper，新增真实点击/确认/状态反馈行为 |

**KEEP（51 个文件）**

- Query Center：Default、Dirty、Apply、Save/Save As、DEGRADED/INVALID、Revision、Visibility、Binding、Quick + Advanced AND。
- Metadata / Dynamic Form：Decimal-safe、Field Contract、Relation/Dictionary、Validation、Linkage、Input Control。
- Permission / Page：按钮能力、无权限不预加载、Route Context、Retry Advanced Fields、Execution 高度与 Runtime 状态条。
- Component / Utils / API：只保留点击、emit、disabled、request DTO、错误转换和稳定数据转换。
- Organization / Integration：保留真实 mount 与领域行为；不再逐页重复测 Query Scheme 模板存在性。

### 4.3 Scripts

| 文件 | 分类 | 长期价值 |
| --- | --- | --- |
| `scripts/preflight-local.test.mjs` | KEEP | Docker/磁盘解析和服务健康判断 |
| `scripts/preflight-external.test.mjs` | KEEP | 生产配置、Secret 文件安全、破坏性操作确认 |
| `scripts/db-backup-external.test.mjs` | KEEP | Backup/Restore Manifest、Hash、权限和 Secret 脱敏 |
| `scripts/smoke-readonly.test.mjs` | KEEP | Read-only Smoke 凭据、URL、Captcha 受控解析 |

现状缺口：44 个 Node 原生测试尚未进入 `release-check`。本轮将增加明确 `scripts-test` 门禁并纳入本地完整发布检查；GitHub Actions 的最终 CI 接入留给 PFCR-S3。

## 5. 重复 Fixture Top 20

| 排名 | 模式 | 当前证据 | 处理 |
| ---: | --- | --- | --- |
| 1 | SQLite DB 初始化 | 64 个 Backend 文件 | KEEP：已统一到 `backend/internal/test.OpenSQLite*` |
| 2 | Integration 页面 Query Scheme mock | 8 个 Frontend 页面 | REWRITE：抽一个窄职责 test fixture |
| 3 | Gin Test Mode TestMain | 6 个包 | KEEP：包级初始化必要；统一调用既有 helper |
| 4 | PostgreSQL schema/cleanup | 多个 migrate/integration/service 文件 | DEFER：跨包差异较大，PFCR-S3 评审共享 schema helper |
| 5 | Data Permission Resource/Policy/Grant | config service/controller 多文件 | KEEP：同包已有局部 helper，领域阶段不同 |
| 6 | Integration Execution fixture | engine/worker/coordinator/repository | KEEP：状态机层级不同 |
| 7 | Organization HR canonical fixture | sync/acceptance/migrate | KEEP：Source、Domain、Migration 事实不同 |
| 8 | Query Scheme actor/scope fixture | service/controller/internal | KEEP：分别验证 Service、HTTP、Schema 边界 |
| 9 | Runtime Metadata field fixture | metadata/generalization/query | KEEP：各边界期望不同 |
| 10 | HTTP recorder/router setup | controller/middleware | KEEP：已可复用 `PerformRequest`，复杂 Gin binding 保持局部 |
| 11 | Casbin role/menu/button fixture | initialize/service/migrate | KEEP：Seed、Projection、Runtime 权限语义不同 |
| 12 | File owner/chunk fixture | service/repository/storage | KEEP：生命周期层级不同 |
| 13 | External System/Interface fixture | service/integration | KEEP：配置与 Runtime 分开 |
| 14 | Retry Policy fixture | service/engine/postgres | KEEP：版本、决策和持久化职责不同 |
| 15 | Frontend Slot/QTable stubs | Integration 页面测试 | KEEP：局部模板契约不同，不造 UniversalMount |
| 16 | Pinia 初始化 | Frontend 页面测试 | KEEP：一行 setup，无抽象价值 |
| 17 | Role/Menu permission button arrays | Frontend 页面测试 | KEEP：页面领域动作不同 |
| 18 | Query pagination defaults | Composable/Page tests | KEEP：Composable 和页面组合分别验证 |
| 19 | Audit subject/context | audit/middleware/model | KEEP：输入、传播、持久化三层 |
| 20 | Secure config fixture | initialize/script tests | KEEP：Go 启动配置和 Node 外部部署配置不是同一协议 |

## 6. Sleep、Timer 与 Mock 审计

- 7 个 `time.Sleep` 中，6 个位于重复的 eventual/polling helper；改为一个 `backend/internal/test` 的有界 ticker helper。
- TLS Server 的首响应延迟用于真实 Retry/Timeout 行为模拟，保留 1 处并记录理由。
- `time.After` 均用于 channel 超时或并发屏障，不属于“等一下就会过”的脆弱 Sleep。
- Frontend 页面 mock 必须最终断言请求、权限、emit、状态或业务组合；只返回 mock 再断言 mock 自身的测试不保留。

## 7. 实施顺序

1. 删除/改写 Frontend source-string 测试，合并 Organization Detail 测试。
2. 收敛 Integration 页面 Query Scheme 测试夹具。
3. 收敛 Backend eventual helper和阶段性源码断言；保留有 import cycle 约束的 `model` TestMain。
4. 将 Node operational tests 纳入 `release-check`。
5. 更新长期工程指南的测试价值、数据库、Race、Frontend、Browser 和 Contract Guard 规则。
6. 执行全量 Go、Race、PostgreSQL 16、Frontend 四项、Scripts、release-check、docs-check。

## 8. 禁止处理清单

- 不改生产业务逻辑、API、Query、权限、数据权限或数据库模型。
- 不删除 Integration 状态机、HR Production Gate、Query Center、Metadata、Data Permission、Auth、File 核心测试。
- 不用 `skip`、注释或 archive 代替删除。
- 不建立 UniversalTestFramework、MegaFixtureFactory 或跨领域万能 Mount。
- 不在本轮实施 PFCR-S3 的 CI、TLS、Migration Ledger、Docker Signal 工作。

## 9. 回归测试计划

- Backend：`go test ./... -count=1`、`go test -race ./... -count=1`。
- PostgreSQL：`SWEET_REQUIRE_POSTGRES_TESTS=true` + PostgreSQL 16 DSN 全量。
- Frontend：`yarn test`、`yarn lint`、`yarn typecheck`、`yarn build`。
- Scripts：`node --test scripts/*.test.mjs`。
- Aggregate：`make release-check`、`make docs-check`。

## 10. 实际修改结果

### 10.1 KEEP

- 保留 Auth、Data Permission、Integration 状态机、Organization HR Production Gate、File 生命周期、Query Center、Metadata、Migration 和 PostgreSQL 专属测试。
- 保留 4 个 Node operational script test，并将其从“可手工运行”提升为 `release-check` 的明确门禁。
- 保留 `backend/migrate/permission_projection_test.go` 的真实 Router/Permission 投影覆盖；它是剩余唯一 Backend source-reading guard，保护动态注册形成的权限闭包。
- 保留 12 处 `time.After`：均为 channel 超时、并发屏障或异步失败上限，不是等待测试碰运气。
- 保留 1 处 TLS 首响应 `time.Sleep`：用于真实模拟第一次响应延迟并验证 Retry/Timeout，不是轮询同步手段。

### 10.2 MERGE

- `organization-detail-mode.spec.ts` 与 `organization-detail-route.spec.ts` 合并为 `organization-detail.spec.ts`，两个导航契约仍分别成用例。
- `query-state-decimal.spec.ts` 合入 `query-scheme-state.spec.ts`，Decimal/SmallInt 与 Query Scheme normalize 在同一契约文件维护。

### 10.3 REWRITE

- `EligiblePageMatrix.spec.ts` 从逐页组件/import 字符串清单缩成一个 Architecture Guard，只检查 17 个 Eligible 页面统一使用一次 `useQuerySchemePage`。
- Dictionary 测试从读取 `.vue` 源码改为 shallow mount，真实验证 Master-Detail 查询隔离、372px 主面板和 Query Center EXEMPT。
- QuerySchemeSelector 测试增加 Dirty 切换确认、DEGRADED/INVALID 阻断和 64 字符方案名行为，不再只看静态文字。
- 8 个 Integration 页面改用 `query-scheme-page-stub.ts`，页面仍只测试自己的 Route Context、Advanced Fields、Runtime strip 和业务动作。
- Backend Context/Metadata Architecture Guard 使用 Go AST/import 分析，避免源码格式和注释导致误判。
- 6 处 open-coded polling 改用一个有界 `testutil.Eventually`；唯一真实延迟场景保留。

### 10.4 DELETE

按分类直接删除 7 个低价值文件：

1. `backend/enum/enum_test.go`
2. `frontend/src/components/Display/StatusChip.spec.ts`
3. `frontend/src/pages/frontend-consistency.spec.ts`
4. `frontend/src/pages/organization/employee/Index.spec.ts`
5. `frontend/src/pages/organization/position/Index.spec.ts`
6. `frontend/src/pages/organization/sync-batch/Index.spec.ts`
7. `frontend/src/pages/organization/sync-error/Index.spec.ts`

另有 3 个 MERGE 输入文件被物理删除：

1. `frontend/src/pages/organization/organization-detail-mode.spec.ts`
2. `frontend/src/pages/organization/organization-detail-route.spec.ts`
3. `frontend/src/utils/query-state-decimal.spec.ts`

删除 3 个没有调用方的 Backend test helper：`WithRollback`、`NewHTTPServer`、`AssertIdempotent`。删除 2 个只证明 helper/Gin setter 自身工作的用例，以及 Migration 中一个阶段函数名不存在断言。

### 10.5 Fixture、Sleep 与 Mock

- 新增 `backend/internal/test.Eventually`，只统一至少 6 处相同的有界轮询，不发展为通用异步框架。
- 新增 `frontend/src/test/query-scheme-page-stub.ts`，只承载 8 个 Integration 页面共同的 Query Scheme composable shape。
- 没有抽取跨领域用户、角色、菜单、组织、Execution 或 Metadata 超级 Fixture；这些 fixture 在不同边界表达不同事实。
- Controller 测试只有在继续验证 binding、permission、DTO 或错误映射时保留；没有留下“mock 返回什么就断言什么”的新增测试。

### 10.6 Contract Guard

最终保留 3 个 source-reading guard：

1. `backend/migrate/permission_projection_test.go`：动态 Router 与权限投影闭包。
2. `frontend/src/pages/query-scheme/EligiblePageMatrix.spec.ts`：17 个 Query Scope 页统一入口。
3. `frontend/src/utils/field-metadata.spec.ts`：Backend/Frontend FieldType 跨端一致性。

除此之外，Context/Metadata 边界改为 AST guard，页面源码字符串清单全部删除。最终 Skip 仅 1 处，位于 PostgreSQL DSN 中央门禁；没有测试文件自行 `skip`。

### 10.7 门禁与长期文档

- Makefile 新增 `scripts-test`，`release-check` 现在包含 44 个 Node operational tests。
- PostgreSQL 和 Race recipe 使用静默命令前缀，避免把测试 DSN 回显到日志。
- `PlatformEngineeringGuide`、`ExtensionDevelopmentGuide` 只记录长期测试策略：行为价值、领域 fixture、PostgreSQL、Race、Frontend、Browser 和少量 Contract Guard。
- GitHub Actions 是否执行完整 `release-check` 属于 PFCR-S3；本轮不扩大 CI 范围。

## 11. After 统计

| 资产 | Before | After | 变化 |
| --- | ---: | ---: | ---: |
| Backend `*_test.go` | 210 | 209 | -1 |
| Backend 顶层 Test（含 6 个 TestMain） | 925 | 920 | -5 |
| Backend 测试 LOC | 50,066 | 49,969 | -97 |
| Frontend 测试文件 | 72 | 64 | -8 |
| Frontend Vitest 用例 | 243 | 231 | -12 |
| Frontend 测试 LOC | 8,346 | 8,109 | -237 |
| Frontend Source-string 文件 | 8 | 2 | -6 |
| Script Test 文件 / 用例 | 4 / 44 | 4 / 44 | 0 |
| Backend `time.Sleep` | 7 | 1 | -6 |
| 显式 Skip | 1 | 1 | 0 |
| 全部测试文件（Backend + Frontend + Scripts） | 286 | 277 | -9 |

物理删除 10 个测试文件，新增 1 个合并后的测试文件，净减少 9 个测试文件。新增 2 个窄职责 test helper，删除 3 个无调用 test helper。

## 12. 验证结果

| 验证 | 结果 |
| --- | --- |
| `go test ./... -count=1` | PASS |
| `go test -race ./... -count=1` | PASS |
| PostgreSQL 16 强制全量 | PASS，`postgres:16-alpine`，`SWEET_REQUIRE_POSTGRES_TESTS=true` |
| `yarn test` | PASS，64 files / 231 tests |
| `yarn lint` | PASS，0 warning |
| `yarn typecheck` | PASS |
| `yarn build` | PASS；保留既有 >900 KiB chunk warning，本轮不改产品依赖 |
| `make scripts-test` | PASS，44 tests |
| `make docs-check` | PASS，68 Markdown files |
| `make release-check` | PASS；包含 docs、scripts、PostgreSQL 16、Race、Frontend test/lint/typecheck/build |

## 13. 最终结论

当前未发现仍可在不损失真实防回归价值的明显 DELETE 候选。剩余测试规模主要来自 Integration 状态机、权限/数据权限、Metadata、Migration、Organization HR Gate 和文件生命周期的真实复杂度。Controller mock 与跨包 PostgreSQL bootstrap 仍可在 PFCR-S3 结合 CI 时间和失败定位数据继续评估，但不构成本轮继续删测试的依据。
