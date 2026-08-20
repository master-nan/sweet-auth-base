# PFCR-S1 Backend Structural Simplification

## 审计基线

- 仓库：`sweet-auth-base`
- 实施基线：`82445207be0471c8268d300fcdd9cd9085d25b9c`
- 分支：`main`
- 实施前工作区：干净，`main` 与 `origin/main` 同步
- 审计方法：以当前 Git 对象、生产调用、Router、Wire、Migration、测试和前端 API 为准；历史 PFCR 结论只作为检查线索

实施前已执行并记录 `git status`、`git diff`、`git log -n 10 --oneline`。没有覆盖或回滚用户修改。

## 审计范围

本轮读取了 `backend/controller`、`backend/service`、`backend/repository`、
`backend/repository/impl`、`backend/internal/cache`、`backend/internal/metadata`、
`backend/internal/datapermission`、`backend/internal/integration`、`backend/internal/queryscheme`、
`backend/model`、`backend/dto`、`backend/enum`、`backend/migrate`、`backend/initialize`、
`backend/api`，并联动检查前端 API、页面调用、Router/Wire 和测试契约。

## 实施前结构概况

统计口径为工作树内全部 `backend/**/*.go`；production 排除 `*_test.go`。

| 指标 | Before |
| --- | ---: |
| Backend Go 文件 | 593 |
| Backend production Go 文件 | 384 |
| Backend test Go 文件 | 209 |
| Service Go 文件（含测试） | 129 |
| Service production Go 文件 | 59 |
| Repository Go 文件（含实现与测试） | 92 |
| Cache Go 文件 | 26 |
| Migration Go 文件 | 35 |
| Backend LOC | 137746 |
| Production LOC | 87955 |
| Test LOC | 49791 |
| Service LOC | 50747 |
| Service production LOC | 28791 |
| `sys_table_service.go` | 2495 |
| `migrate/main.go` | 3647 |

大文件不是删除依据。审计确认的结构热点是 SysTable 混入低代码发布、Migration main 混入
Integration 展示默认值、跨六个领域的 system projection、历史 Generalization 安全入口、无消费者缓存
wrapper、运行时类型断言和一条 Router 暴露的 no-op API。

## 实施前对象裁决

### Response Projection

| 对象 | 裁决 | 当前职责与调用方 | 判断、替代方案与风险 | 验证 |
| --- | --- | --- | --- | --- |
| `system_response_projection.go` | MERGE | Dict、SMS、AccessLog、Application、User、Configure 的 DTO 投影；对应 Controller 调用 Response 方法 | 文件横跨六个领域。转换和 Response API 回到各自现有 Service；共享 Basic 投影进入已有 `dto/response/basic_res.go`。没有改名成另一种 Mapper，也没有把转换推给 Controller | Service/Controller、DTO 投影测试、Swagger |
| `metadata_response_projection.go` | KEEP | Metadata 管理态和 Runtime 白名单 DTO | 运行态/管理态隔离是安全边界，且全部方法有生产调用。仅修复三个 `context.Background()` 为请求 Context；不消灭该边界 | Metadata、Generalization、Context 回归 |
| `permission_response_projection.go` | KEEP | Menu/Role/Permission 白名单和树投影 | 递归与授权投影有稳定安全语义，不是无意义字段复制 | Menu/Role/Casbin 测试 |
| `report_response_projection.go` | KEEP | Report 管理态白名单 | Report 是保护区；53 行投影不会成为独立架构负担 | Report 回归 |
| `query_scheme_projection.go` | KEEP | Scheme Summary/Detail/Runtime 投影 | 包含 visibility、validation issue、role 范围等领域规则 | Query Scheme 回归 |

审计初稿曾计划合并 Metadata/Permission projection。完整调用图证明其白名单边界真实存在，因此实施前改判为 KEEP，未为满足旧结论制造修改。

### SysTable 与低代码发布

| 对象 | 裁决 | 当前职责与调用方 | 判断、修改后结构与风险 | 验证 |
| --- | --- | --- | --- | --- |
| SysTable Table/Field/Index/Relation/DDL/View | KEEP | Table Controller、Schema 管理、Metadata cache invalidation | 真实 Schema 生命周期保持一个 Service；按每个对象拆 Service 会增加跳转 | SysTable、DDL、Relation、PostgreSQL |
| Publish/Unpublish 及 13 个发布 helper | SPLIT | Database 页面发布入口；菜单、按钮模板、角色授权编排 | 提取唯一 `LowCodePublicationService`。不是 Table/Field/Relation 多拆分；Controller URL、Method、Request、Response、权限和事务不变 | publication helper、Wire、Controller、Migration |
| `MetadataRuntimeService` | KEEP | Runtime Metadata 和安全字段边界 | 已有独立运行时语义，不创建第三套 Runtime | Runtime/Query Scheme |

拆分后 SysTable 构造依赖从 14 个降为 7 个；原先仅供发布使用的六个 Menu/Role Repository 和一个从未使用的 Server Config 不再进入 Schema Service。

### SysDict 与缓存

| 对象 | 裁决 | 当前职责与调用方 | 判断、修改后结构与风险 | 验证 |
| --- | --- | --- | --- | --- |
| Dict/DictItem mutation | SIMPLIFY | Controller CRUD，Dict 聚合按 id/code 缓存 | mutation 前捕获 identity，DB 成功后统一失效 id/code；不再 delete 后查询、覆盖主错误或使 item 删除失效 key 0 | 新增 alias 失效和 mutation error 测试 |
| 五个在用 system model cache | MERGE | Configure、Dict、Table、TableField、User | 都只是 `BasicCache[T]`，合并到 `system_model_cache.go`，保留原类型和 constructor | Wire、Cache、Service |
| 六个无消费者 system cache wrapper | DELETE | Wire provider 列表但生成图未使用 | 删除运行时类型/constructor；仅保留 Migration 清理旧 key 所需前缀，避免历史 Redis key 泄漏 | Wire 生成、Migration prefix 测试 |
| DingTalk 两个 cache 文件 | MERGE | AccessToken 与 UserID，后者有 7 天 TTL | 同领域合并，保留独立类型与 TTL，不做 all-cache God file | DingTalk/Wire |
| Generalization/SmsLog/BlackUser cache | DELETE | 当前生产零消费者；BlackUser 只被未读取的 App 字段强行构造 | 删除死 provider/type；TokenBlack、LoginAttempt、SendCode 等真实安全缓存保持 | Wire、Auth、Migration prefix |
| SmsTemplate/LoginAttempt/SendCode/TokenBlack/Application | KEEP | 有真实消费者、专属 TTL 或安全状态 | 不因文件小删除 | 对应领域测试 |

字典父删除是否级联 item、数字 dict code 与 ID 共用 key 空间属于业务语义/缓存协议问题。本任务不改变数据库业务语义，记录为后续正确性候选。

### Generalization

| 对象 | 裁决 | 当前职责与调用方 | 判断、修改后结构与风险 | 验证 |
| --- | --- | --- | --- | --- |
| 完整 Runtime + DataPermission constructor | KEEP | Wire 唯一生产构造器 | 正式安全组合边界 | Wire、权限验收 |
| 两个历史 constructor | SIMPLIFY | 仅测试或被正式 constructor 间接调用 | 合并为一个 private constructor，测试不再决定生产 public surface | Generalization 测试 |
| `Query/GetById/Update/Delete` 无权限入口 | DELETE | 无生产调用，仅同包旧测试 | 删除公开绕权限入口；正式 `*WithDataPermission` 和 resolved 入口保持 | Data Permission、Report、Controller |
| `QueryWithResolvedDataPermission` | KEEP | Report 避免重复解析权限 | 有明确安全和性能语义，不是兼容 wrapper | Report 回归 |
| 受保护表 predicate 双入口 | MERGE | 同文件转发 | 收敛为一个 private 规则函数 | protected-table 测试 |

### Repository 与其他 public surface

| 对象 | 裁决 | 当前职责与调用方 | 判断、修改后结构与风险 | 验证 |
| --- | --- | --- | --- | --- |
| LoginLog/SmsLog context writer | MERGE | Log/SMS Service 通过运行时类型断言查找写能力 | 写能力进入正式 Repository interface，删除运行时断言；实现不变 | Log/SMS/Async 测试 |
| `CasbinRuleService` | DELETE | 三个 Repository 直转发方法；未被 Wire 生成图构造 | 删除死 Facade；Menu/Role/Report 继续直接依赖窄 Repository | Casbin/Wire/Report |
| `CasbinRuleRepository` generic CRUD 和 `RemoveFilteredPolicy` | SIMPLIFY | 无调用 | 去掉未使用能力，保留 policy consistency、事务投影和 reload | Casbin consistency |
| Dict/SMS/AccessLog/Application Model 与 Response 双公开 API | SIMPLIFY | Controller 只使用 Response；Model 版本只在同包使用 | Model 读取降为 private，Response JSON 契约不变；跨 Service 真正复用的 Configure/User 保留 | Controller、Service、Swagger |
| `BasicRepository` 与领域 interface/impl 文件 | KEEP | GORM 底座、DI、mock 和领域查询 | 不引入 V2，不把所有 Repository 合到一个文件 | Repository 全量 |
| Response fluent dead setters、`IsPageButton`、一次性 Application key helper | DELETE/SIMPLIFY | 无生产调用或仅调用一次 | 删除无行为 API，单次 helper 私有化 | DTO/Service 测试 |

### No-op API 与 Migration

| 对象 | 裁决 | 当前职责与调用方 | 判断、修改后结构与风险 | 验证 |
| --- | --- | --- | --- | --- |
| `POST /admin/menu/refresh-cache` | DELETE | Router、Controller、恒 `return nil` Service、未调用前端 wrapper、Seed/Casbin/Swagger | 删除完整链；Seed 幂等删除历史按钮、角色关联和 Casbin policy | Migration 新增专项、Swagger、Frontend |
| 全局 `refresh_cache` event action | KEEP | 业务按钮可以表达刷新外部状态/缓存；已有测试明确保护 | 不是上述 no-op endpoint 的专属枚举，不删除后端/前端枚举和字典 | Migration business refresh 测试 |
| Integration metadata defaults | SIMPLIFY | `systemColumnToTableField` 调用的 8 个纯 Seed 函数 | 从 `migrate/main.go` 移到已有 `integration_seed.go`，不新增文件、不改执行顺序/数据 | Migration/Integration |
| 其他历史 Migration、Ledger | DEFER | 已部署数据库升级路径 | 不依据当前调用删除；Ledger/checksum 属 PFCR-S3 | 本轮不处理 |

### Utils 与小 Helper

| 对象 | 裁决 | 判断 |
| --- | --- | --- |
| Session/reflection/escaping 等 10 个 definition-only helper | DELETE | Router、生产、测试均无调用；不是语言/GORM 反射钩子 |
| `IntFromAny`、`FlattenMenus`、两个旧 read-action helper 及三个内部数值转换 | DELETE | 无生产调用，旧测试只测试 helper 自身；正式 capability 路径已替代 |
| `HasTableField`、`MenuAllowsTableCode`、Validator、sanitize、指针 helper | KEEP | 有真实调用或安全规则 |
| 规则型/算法型/安全型 private helper | KEEP | 不执行“为了数量全部 inline” |

## 禁止处理清单

本轮明确不重构 Auth、Integration Engine/Worker/Retry/Sync/Transport、File 生命周期、
Data Permission Resolver/Registry/Preflight、Metadata Runtime/value contract、Query Scheme、HR adapter/parser、
BasicRepository、Router/Wire 文件结构和 Report 产品结构。不处理前端视觉、Query Center 页面、
DynamicFormDialog、Migration Ledger、TLS、Docker signal、CI workflow、Editable Grid、TMS 或新增业务能力。

## 实施顺序与回归计划

1. 建立本报告并完成真实调用图。
2. Generalization public surface、SysDict mutation、Cache wrapper。
3. SysTable publication 边界和 system projection 归属。
4. Repository 契约、dead helper、no-op endpoint、Migration main。
5. 每批运行相关 Go 测试；最终运行 `go test ./... -count=1`。
6. 按 Makefile 运行 `release-check`：PostgreSQL 强制全量、Race、Frontend test/ci、docs-check。

预计新增两个生产文件（一个 Service、一个合并 Cache），删除无消费者或被吸收的文件；不以预设删除数量作为验收门槛。

## 实际修改结果

### KEEP

- Metadata、Permission、Report、Query Scheme projection 保持。
- SysTable Schema 生命周期、Metadata Runtime、BasicRepository、Data Permission、Integration、HR、File、Auth 保持。
- `refresh_cache` 业务动作枚举/字典保持。
- 真实安全/TTL cache 保持。

### MERGE

- 11 个 system cache 文件收敛到 `backend/internal/cache/system_model_cache.go`；只保留 5 个在用 wrapper。
- DingTalk UserID cache 合入 `dingtalk_cache.go`。
- system projection 的六个领域块合回原 Service，共享 Basic 投影进入现有 DTO 文件。
- Login/SMS context writer 能力进入 Repository interface。
- Generalization 三个 constructor 收敛为一个正式 public + 一个 private 构造边界。

### SIMPLIFY

- SysTable 构造依赖 14 → 7；移除未使用 Server Config。
- Dict mutation 统一 `invalidateDictCache`，删除后不再查询已删除 row。
- 13 个仅供本包使用的 Model API 降为 private，Controller 继续使用原 Response API。
- Metadata 三个 Response 方法传播请求 Context。
- Casbin Repository 去掉未使用 generic CRUD/filtered remove。
- Integration Seed 从 Migration main 移回已有领域文件。

### SPLIT

- 新增 `backend/service/low_code_publication_service.go`，承接低代码菜单发布与授权编排。
- Table Controller 直接注入 Schema Service 与 Publication Service；HTTP 契约不变。

### DELETE

- 完整删除 `/admin/menu/refresh-cache` Router/Controller/Service/frontend wrapper/Swagger/Seed permission 链。
- 删除 `CasbinRuleService`、`system_response_projection.go`。
- 删除 9 个无消费者 cache 类型/provider、相关 App 字段；移除 SmsTemplate 未调用批量 setter。
- 删除 14 个无真实调用的 exported helper、4 个 private helper 及仅验证旧 helper 的自测。
- 删除 3 个 Response dead setter 和 1 个 Model dead method。
- Generalization 删除 4 个无权限 public operation、2 个历史 public constructor 和重复 predicate。

### DEFER

- 字典父删除级联、缓存 key namespace 调整：会改变业务/缓存协议，不属于结构等价整改。
- Data Permission 的测试/扩展 public API：属于明确保护区。
- ReportService、OrgService、DynamicFormDialog 等大文件：真实领域复杂度或后续专项。
- Migration Ledger/checksum、CI/Operations、文档 Final Freeze：PFCR-S3 / DOC-FINAL。

## 调用链变化

| 场景 | Before | After |
| --- | --- | --- |
| 普通系统 DTO | Controller → cross-domain system projection file → Service → Repository | Controller → owning Service DTO method → Repository |
| Metadata Schema | TableController → SysTableService（含发布依赖） | TableController → SysTableService |
| Low-code 发布 | TableController → SysTableService → Menu/Role repositories | TableController → LowCodePublicationService → Menu/Role repositories |
| Generalization 写读 | 存在 public 无权限和有权限平行入口 | 生产 public surface 只保留权限完整入口 |
| Login/SMS 日志写 | Service → runtime type assertion → Repository impl | Service → Repository contract |
| Menu cache refresh | HTTP → Controller → no-op Service | 契约和历史权限数据均删除 |

## 文件与 LOC 变化

| 指标 | Before | After | Delta |
| --- | ---: | ---: | ---: |
| Backend Go 文件 | 593 | 578 | -15 |
| Backend production Go 文件 | 384 | 369 | -15 |
| Backend test Go 文件 | 209 | 209 | 0 |
| Service Go 文件（含测试） | 129 | 129 | 0 |
| Service production Go 文件 | 59 | 58 | -1 |
| Repository Go 文件 | 92 | 92 | 0 |
| Cache Go 文件 | 26 | 12 | -14 |
| Migration Go 文件 | 35 | 35 | 0 |
| Backend LOC | 137746 | 136993 | -753 |
| Production LOC | 87955 | 87177 | -778 |
| Test LOC | 49791 | 49816 | +25 |
| Service LOC | 50747 | 50605 | -142 |
| Service production LOC | 28791 | 28581 | -210 |
| `sys_table_service.go` | 2495 | 2055 | -440 |
| `migrate/main.go` | 3647 | 3222 | -425 |

新增生产文件：

- `backend/internal/cache/system_model_cache.go`
- `backend/service/low_code_publication_service.go`

新增测试文件：

- `backend/service/sys_dict_service_test.go`

删除 17 个生产 Go 文件和 1 个被替代的测试文件；完整清单由本 Task 的 Git diff 保留。

## 兼容性说明

- 唯一 HTTP 行为变化是删除确认无调用且恒成功的 no-op `/admin/menu/refresh-cache`。
- 其他 URL、Method、Request JSON、Response JSON、状态码、错误码、权限、数据权限、Migration 数据结果和前端页面行为保持。
- `refresh_cache` 业务动作继续支持。
- Swagger 已重新生成，不再包含 dead endpoint。

## 测试结果

| 门禁 | 结果 |
| --- | --- |
| 分批 Service/Controller/Initialize/Migrate/Repository/Cache | PASS |
| `cd backend && go test ./... -count=1` | PASS |
| PostgreSQL 16 强制全量 | PASS |
| `go test -race ./... -count=1` | PASS |
| Frontend test/lint/typecheck/build | PASS |
| `make docs-check` | PASS（64 个 Markdown 文件） |
| `make release-check` | PASS |
| `git diff --check` | PASS |

完整门禁使用本地 PostgreSQL 16 DSN 执行。前端构建保留既有大 chunk warning
（PPTX renderer、HEIC、rich text 等），本轮没有前端结构与依赖拆分权限，因此不改变其行为。

## 未处理项

- PFCR-S2：Frontend/Query Center/DynamicFormDialog 的结构与 UX 收敛，本轮未触碰。
- PFCR-S3：Migration Ledger、CI/Release、TLS、shutdown 和 Operations 门禁。
- DOC-FINAL：吸收本 construction 报告中的稳定结论并删除阶段材料。
- Report 产品结构仍按 `REPORT_DEFERRED` 保护；仅执行兼容回归。
