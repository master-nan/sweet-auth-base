# Sweet Platform Organization HR V1 正式验收报告

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| Task | INT-006G |
| 验收日期 | 2026-08-13 |
| 验收范围 | Organization HR Adapter V1 能力、唯一运行链、七类 Consumer、任职安全解析边界、页面、权限、脱敏与生产 Gate |
| 验收基线 | `4787aaa3448a74c13efb5ac3584cfd9b26496bd0`（实现组织离职同步安全子集） |
| INT-006B | `53158a9`（建立组织HR同步契约与完整性基础） |
| INT-006C | `9bbd60e`（实现组织法人和组织结构同步） |
| INT-006D | `e04e1e8`（实现组织岗位同步） |
| INT-006E | `f9355a2`（实现组织员工实体同步） |
| INT-006F-A | `1b45920`（完成组织任职与离职源契约评审） |
| INT-006F-B | `4787aaa`（实现组织离职同步安全子集） |
| Integration Sync Freeze | `d4f8fa9`（完成集成同步任务验收与冻结） |
| 验收环境 | macOS arm64、Go、Node.js 22、Yarn 1.22、PostgreSQL 16.14、Redis 6.2、本地 TLS Server、Docker Compose、真实浏览器 |
| 正式设计 | [OrganizationHRSyncDesign.md](../design/OrganizationHRSyncDesign.md) |
| 源契约评审 | [OrganizationHRAssignmentContractReview.md](OrganizationHRAssignmentContractReview.md) |
| Sync 扩展评审 | [IntegrationSyncSourceContractChangeReview.md](IntegrationSyncSourceContractChangeReview.md) |

本报告以当前仓库真实代码、Migration、Registry、Service、Runner、页面、PostgreSQL 16 和自动化结果为证据，不以历史 Task 回复代替审计。

## 2. 两层正式结论

### 2.1 Platform / Adapter Capability Freeze

**结论：通过冻结。**

法人、管理公司、管理部门、法人部门、岗位、员工和离职安全子集已形成同一 registered Consumer 架构。Source DTO、Canonical Input、Organization Domain、业务 Batch/Record、Integration 业务成功边界和 Checkpoint 之间没有旁路。任职能力准确停留在 Parser、已确认日期规范化和 Crosswalk 端口，生产路径没有猜测主任职或兼职落库。

本轮未发现重复业务对象、错误 Checkpoint 推进、业务失败自动 Retry、Organization 自行 HTTP/调度/重试、Source-specific 字段侵入 Domain、Payload 持久化或敏感源值泄露等冻结阻塞项。

### 2.2 Current HR Source Production Enablement

**结论：不允许。**

七类生产 Consumer 均已静态登记但保持 `disabled`。BIP ID 生命周期、`changeTime` 权威性/时区/精度/同秒完整性等共同 Gate 尚无源负责人权威证据；对象特有的编码、员工编号、人员响应量和离职视图等 Gate 也未全部关闭。显式 Test SourceContract 只能用于自动化与验收，不能替代生产源契约。

## 3. 唯一运行链

```text
IntegrationSyncRunner
  -> IntegrationSyncBatch
  -> IntegrationSyncCoordinator
  -> IntegrationExecutionService
  -> IntegrationExecution
  -> IntegrationWorkerRunner
  -> CredentialProvider
  -> TransportClient
  -> Organization SyncResultConsumer
  -> Organization Domain Service
  -> OrgSyncBatch / OrgSyncRecord
  -> Execution 收敛
  -> IntegrationSyncCoordinator 推进 Checkpoint
```

代码审计确认 `internal/organization/hrsync` 不创建 HTTP Client、不读取 Credential、不启动 Scheduler、不实现 Retry、不修改 `next_run_at`、不直接推进 IntegrationExecution 或 Checkpoint。技术失败仍由 Integration Runtime/Retry 处理；业务失败固定为 confirmed `business_processing_failed`，不得进入 Retry。

## 4. Consumer 验收

| Consumer | 领域写入 | 关键结果 | 生产状态 |
| --- | --- | --- | --- |
| `org.hr.legal_entity` v1 | `org_legal_entity` | Stable SourceKey upsert；inactive 仅停用；不创建管理组织 | disabled |
| `org.hr.management_company` v1 | `org_unit` + management node | business_unit 根；两阶段关系；不写法人表 | disabled |
| `org.hr.management_department` v1 | `org_unit` + management node | 子先父后、历史父、deferred、self/cycle 受控 | disabled |
| `org.hr.legal_department` v1 | `org_unit` + legal node | legal 树独立；不按名称与 management 合并 | disabled |
| `org.hr.position` v1 | `org_position` | 组织 SourceKey 引用；同名可并存；code 冲突失败；不创建 Role | disabled |
| `org.hr.employee` v1 | `org_employee` | 可空联系方式；stale write；不创建账号/任职；普通 inactive 不直接 resigned | disabled |
| `org.hr.resigned_employee` v1 | `org_employee` + 已有 Assignment | 严格 `lzdate`；resigned；原子关闭当前任职；不自动再入职 | disabled |

Registry 测试逐个验证七个 code/version 的 disabled 元数据不能被生产 Resolve。代码没有环境变量或角色名绕过 Gate；测试通过显式 SourceContract 使用相同实现。

## 5. 初始化顺序与完整 E2E

新增真实 PostgreSQL 16 + IntegrationSyncRunner + IntegrationWorkerRunner + TLS Mock + 七类 Test SourceContract 的初始化顺序验收：

```text
法人
  -> 管理公司
  -> 管理部门
  -> 法人部门
  -> 岗位
  -> 员工
  -> 离职
```

测试没有预造最终领域对象；后续对象引用前序 Consumer 的真实产物。首次法人请求返回 503，由 Integration Retry 在同一 Execution 追加 Attempt 后成功。七个 Batch 均由 Runner/Coordinator 自动收敛，最终得到正确的 legal entity、unit、management/legal structure node、position、employee，以及 resigned 员工和关闭的既有 Assignment。共 7 个 Execution、8 个 Attempt；没有第二条 Retry 链，也没有持久化 Response Body。

## 6. Organization 结构

- `management` 与 `legal` 使用固定受控结构，节点不跨树。
- 部门先 Phase 1 upsert 主体，再 Phase 2 解析关系，不依赖响应顺序。
- 本 Slice 后到父节点和历史父节点均可解析；缺父写 deferred 并使整片失败。
- self-parent、A-B-A 和多节点循环稳定失败，不递归失控、不自动断链。
- inactive 父保留历史关系；inactive 对象只停用，不物理删除或设置 `source_deleted=true`。
- 名称不参与身份合并，`structure_node_id` 不作为业务组织归属 ID。

## 7. 岗位

岗位身份只使用 `(source_system_code, position, raw_source_id)`。部门引用只按已同步 org_unit SourceKey 解析；缺组织 deferred，补齐后由 Lookback 重放收敛。不同 Source ID、不同 code、同名岗位可以跨组织并存；业务 code 冲突稳定失败且不加随机后缀。岗位类型固定受控 `professional`，等级只是可空显示属性，`is_manager_position=false`，不会根据名称猜测或写 Role/Casbin。inactive 只映射 disabled。

## 8. 员工

员工身份只使用 `(source_system_code, employee, raw_source_id)`。`employee_no` 是受 Gate 的业务属性，不是身份回退；缺失或冲突失败。手机号和邮箱均可为空。普通人员 active 映射 active，其他已确认普通状态保守映射 suspended，不直接设置 resigned。

源版本只来自受控 SourceContract 解析的源时间：旧版本 noop，同版本同事实 noop，同版本受保护事实冲突失败。不使用数据库时间或到达顺序。员工 Consumer 不创建 SysUser、不修改已有 `user_id`、不创建 Assignment、不读取顶层公司/部门/岗位为主任职，也不消费 `sendpost` 形成任职。

## 9. 离职安全子集

`lzdate` 严格按 `YYYY-MM-DD` LocalDate 解析，不能空、不能用 `changeTime` 或数据库当前时间替代。合法事实保留 Employee 和 `user_id`，写 `employment_status=resigned`、`valid_to=lzdate`，不设置 source_deleted。

同一离职记录事务内锁 Employee，校验并关闭该员工真实存在、enabled、当前有效且 `valid_from <= lzdate` 的 Assignment。周期冲突时 Employee、Assignment 和 Record 一起回滚。员工不存在写 deferred；旧离职 noop；离职后更晚普通 active 返回 `org_sync_employment_state_conflict`，不自动再入职。Consumer 不停用 SysUser、不清理角色或账号绑定。

## 10. 任职未支持边界

生产路径不存在以下行为：

- 人员顶层公司/部门/岗位转 Assignment；
- `sendpost` 或 `sendPostList` 转 Assignment；
- 自动 `is_primary=true` 或 part_time 持久化；
- NCID 直接作为 BIP ID、按名称或 code Crosswalk；
- 魔法结束日期；
- 离职事件创建一条猜测 Assignment。

`AssignmentSourceParser` 只输出 Source Adapter candidate，限制 256 KiB、最多 100 项、关系 ID 非空且唯一、每项 object、岗位可空、未知字段忽略。`NormalizeAssignmentPeriod` 只接受已确认的 current 空 end 和合法 `end >= start` 组合。默认 Crosswalk Resolver 稳定不可用。上述对象不能直接进入领域持久化 API。

## 11. Source Adapter 可插拔边界

当前 HR 特有的 `psnidzjkid_ignore`、`postidzjkid_ignore`、`jhcode`、`sendpost`、`userType`、`ifreentry`、NCID 和源日期格式只存在于 `internal/organization/hrsync` DTO/Parser/Normalizer。Organization Domain Service 接收 source-independent Canonical Input，不依赖这些源字段名。没有动态 Adapter、脚本、反射或 SQL 扩展点。

## 12. lower_bound_only、Window 与 Checkpoint

V1 `bounded_window` 行为未改变；V2 `lower_bound_only` 只绑定真实 start 参数。Execution 仍冻结 `[logical_start, logical_end)`，Consumer 过滤 future-of-logical-window 记录，不生成当前 Slice 成功 Record，Checkpoint 只在整片业务成功后推进到 logical_end。HTTP 不伪造 end 参数。

该契约不声称响应有界，不减少已经下载的 Body，也不替代生产响应量验证。Transport 64 MiB 上限未修改；Organization、Position、Employee、Resigned Consumer 仍有更低各自上限。

## 13. Retry、Lookback、幂等与部分失败

真实 E2E 的首次 HTTP 503 只由 Integration Retry 处理：同一 Execution 的 Attempt 1 失败、Attempt 2 成功后才调用 Consumer 并推进 Checkpoint。业务引用缺失、源冲突、周期冲突和持久化失败均只有一个 Attempt，不进入 retry_waiting。

Organization、Position、Employee、Resigned Employee 均覆盖 Lookback/重复 Consume；SourceKey、源版本、业务 Batch 唯一关系和 `(batch, object_type, source_id)` 约束保证不重复领域对象、节点、关闭动作或 Record。

专项 PostgreSQL E2E 在同一 Slice 放入一个正确根节点和一个缺父子节点：正确主体先提交，Consumer 整体失败、Execution/SyncBatch failed、Checkpoint 不推进；补齐父节点后重放，已成功根节点 noop，失败对象成功，最终 Checkpoint 推进且根节点仍只有一个。

## 14. 事务与资源边界

| Consumer 类别 | 最大业务 Chunk |
| --- | ---: |
| 法人/公司/部门 | 500 |
| Position | 500 |
| Employee | 200 |
| Resigned Employee | 500 |

解码、窗口过滤和 Normalizer 在事务外；领域 upsert 与 OrgSyncRecord 使用短 Chunk 事务。离职单记录的 Employee、Assignment 关闭和 Record 原子提交。不存在整响应一个事务，也不存在正常路径每条记录一个事务。

Employee 上限为 16 MiB；`sendpost` 上限 256 KiB/100 项。其他 Consumer 也执行独立元数据上限和 Runtime 响应限制。没有提升 Transport 64 MiB、Response Artifact、磁盘临时 Payload 或数据库完整 Body。

## 15. DTO、日志、Audit 与业务记录脱敏

本轮加强了 Organization 同步查询 DTO：列表、详情和错误接口只返回 `source_summary`、`dependency_summary` 与稳定 Reason Code。遗留数据库中的原始 source/error/dependency 文本不满足安全格式时 fail-closed 为空，不再由管理员详情旁路返回。`OrgSyncRecord.source_id` 的正式 HR 写入是不可逆摘要。

DTO、日志、Audit 和业务 Record 不返回或记录完整 Response、Source DTO、ExecutionInputSnapshot、Credential、Authorization、Cookie、Token、数据库错误原文、原始 `sendpost`、原始 Source ID 或人员联系方式。Scheduler/Consumer 不伪造管理员 AuditSubject。

## 16. 页面与权限验收

真实浏览器以当前前端和后端代码登录管理员账号，确认：

- 集成中心含外部系统、接口定义、集成凭证、重试策略、同步任务、同步批次、执行记录、调用日志；
- Organization 含组织架构、人员与任职、岗位、同步批次、同步异常；没有第三套 HR 同步中心；
- 技术 Batch/Execution/Log 与业务 Batch/Record 分开；业务异常显示对象、Action、状态、稳定 Reason Code、安全 Source 摘要和目标事实，不显示 Payload；
- 深色模式实际切换为 Quasar `body--dark`，页面可读；动态按钮来自权限；
- 业务 Batch 详情在只有 batch detail 时不会隐式请求 record query；Execution/Sync 页和 Organization 列表在缺少各自 query 权限时不预加载数据。

无权限账号 `dp_acceptance_ungranted` 实际登录后为 0 菜单、0 按钮、0 接口权限。Organization 业务同步、Integration SyncTask/SyncBatch/Execution 五个直达路由均稳定进入 404。数据库访问日志仅出现 `user/me` 和 `menu/my`，没有相关受保护数据请求。Integration 权限不会自动授予 Organization 业务结果权限；后端 Casbin 仍是最终边界。

## 17. PostgreSQL 16 与自动化结果

实际设置 `SWEET_TEST_POSTGRES_DSN` 和 `SWEET_REQUIRE_POSTGRES_TESTS=true`，使用 PostgreSQL 16.14，未跳过门控。

```text
SWEET_REQUIRE_POSTGRES_TESTS=true go test ./... -count=1
结果：通过；包含 Migration、约束、七类初始化顺序、TLS、Retry、Checkpoint、部分失败重放与幂等。

go test -race ./internal/organization/hrsync ./internal/integration ./repository/impl ./service ./initialize -count=1
结果：通过。

go test -race ./controller -run 'TestOrgControllerSync|TestIntegrationSyncController|TestIntegrationExecutionController|TestIntegrationConfigurationControllers' -count=1
结果：通过。

yarn test
结果：通过，36 个测试文件、136 个测试。

yarn lint / yarn typecheck / yarn build
结果：全部通过；仅有既有大 chunk 提示。
```

扩大的历史 Controller race 表达式仍可命中既有 `TestOrgControllerEmployeeUserBindingUsesPermissionsAndSafeResponse` 与 Gin/异步访问日志的测试夹具竞态；该测试单独执行通过，本次 Organization HR Sync/Integration 专项 Controller race 通过。该历史测试基础问题未被掩盖，也不属于 HR Consumer 运行链。

## 18. 本次发现并修复的问题

1. 业务同步 Record/Error DTO 会返回数据库中的原始 `source_code`、`dependency_key` 和自由错误文本：改为安全摘要与稳定 Reason Code，并对遗留不安全值 fail-closed。
2. Organization 业务 Batch 详情把 Batch detail 和 Record query 权限耦合，导致最小权限下详情不可用或越权预取：拆分加载边界。
3. 通用详情、Integration Execution 详情和多个列表页在缺少按钮权限时仍可能预加载 API：补充前端 query/detail 守卫和自动化。
4. 缺少七类 Consumer 按真实初始化顺序、同一 Execution 技术 Retry 的统一 PostgreSQL E2E：新增完整链测试。
5. 缺少“同 Slice N-1 成功、1 条 deferred、修复后重放”的连续 Checkpoint 证据：扩展 PostgreSQL E2E。

## 19. Production Gate 矩阵

| Gate | 状态 | 阻塞 Consumer | 生产影响 | 所需外部确认 | 能力冻结影响 |
| --- | --- | --- | --- | --- | --- |
| BIP ID 永久稳定、不可复用 | unconfirmed | 全部七类 | 无法承诺长期 upsert 身份 | 源负责人书面 ID 生命周期契约 | 否；SourceKey/冲突停止已冻结 |
| changeTime 权威性、时区、精度、同秒完整性 | unconfirmed | 全部七类 | Checkpoint/陈旧写可能漏数或误序 | 字段、时区、精度、包含边界和同秒保证 | 否；测试契约明确隔离 |
| 法人/组织/岗位编码命名空间 | unconfirmed | legal/company/department/position | 目标唯一 code 可能冲突 | 各对象 code 唯一范围和生命周期 | 否；冲突稳定失败 |
| 法人部门根边界 | unconfirmed | legal_department | 根/父关系可能误判 | 法人视图根节点和公司父引用契约 | 否；当前 unresolved 失败 |
| employee_no 生命周期 | unconfirmed | employee | 编号缺失/复用不能安全覆盖 | `jhcode` 必填、唯一、复用和再入职规则 | 否；缺失/冲突失败 |
| company partition 批准清单 | unconfirmed | employee | 不能配置生产静态分区 | 非敏感稳定 ID 清单和变更流程 | 否；无动态 Fan-out |
| 人员响应量 | unconfirmed | employee | 单下界可能超过 16/64 MiB | 生产窗口量测、限流及分区容量 | 否；超限安全失败 |
| resigned 权威 `userType` | unconfirmed | resigned_employee | 无法选定生产离职视图 | 源负责人确认 type 0/1 权威语义 | 否；生产 disabled |
| 主任职语义 | unconfirmed | 未实现 Assignment | 不能创建主职 | 主职标识、唯一性和历史契约 | 否；明确不支持 |
| Assignment stable ID | unconfirmed | 未实现 Assignment | 不能幂等保存任职段 | 关系 ID 生命周期或稳定雇佣段键 | 否；Parser 不持久化 |
| sendpost 权威性 | unconfirmed | 未实现 Assignment | 不能生产消费兼职 | 与 sendPostList/sendposten 的权威关系 | 否；只解析测试 |
| NCID -> BIP Crosswalk | unconfirmed | 未实现 Assignment | 兼职组织引用不可解析 | 受控 Crosswalk 数据与维护责任 | 否；默认 Resolver 不可用 |
| 再入职/雇佣段 | unconfirmed | employee/resigned/Assignment | resigned 后 active 不能自动恢复 | 事件优先级、再入职标识和雇佣段 ID | 否；冲突停止 |
| 物理删除表达 | unconfirmed | 全部领域对象 | V1 不能标记源删除 | Tombstone/全量对账契约 | 否；只停用不删除 |

## 20. V1 明确不支持

V1 不支持主任职同步、生产兼职同步、自动再入职、物理删除、动态公司 Fan-out、全量对账、视图 99、OA 专用人员源、Response Artifact 或 HR 业务脚本。这些限制没有被实现伪装，因此不构成 Capability Freeze 失败；在后续权威契约和独立设计完成前不得通过临时代码打开。

## 21. 最终判断

**Organization HR Adapter V1 Capability：通过冻结。**

**Current HR Source Production Enablement：不允许。**

下一阶段应先完成源负责人 Gate 确认与生产容量试验，而不是继续扩展 Adapter。任何 Consumer 生产启用都必须逐项关闭其关联 Gate、形成受控 SourceContract 并重新执行 PostgreSQL/TLS 小流量准入验证。
