# Sweet Platform Organization HR 同步详细设计

## 1. 文档状态

| 项目 | 内容 |
| --- | --- |
| Task | INT-006A |
| 状态 | 详细设计完成；INT-006B 已提供单下界平台兼容契约，其余第 22 章 P0 继续门控生产 Consumer |
| 日期 | 2026-08-12 |
| 范围 | Organization HR 源映射与服务端 `SyncResultConsumer` 设计 |
| Runtime 基线 | `IntegrationRuntimeFreezeReview.md` |
| Retry 基线 | `IntegrationRetryFreezeReview.md` |
| Sync 基线 | `IntegrationSyncFreezeReview.md` |

本文依据当前仓库中的 Organization 模型、已冻结的 Integration Runtime/Retry/Sync 设计、本地 OpenAPI，以及 `docs/analysis/organization-source/` 下的脱敏分析证据编写。本文不授权建立第二条 HTTP、Retry、Scheduler、Checkpoint 或 Execution 链路。

真实源证据来自有限增量窗口。样本中非空、唯一只能证明样本事实，不能替代源系统关于永久稳定性和不可复用性的正式承诺。

## 2. 范围与硬边界

Organization HR Adapter 只能：

1. 注册服务端 `SyncResultConsumer`；
2. 解码 Integration Runtime 交付的响应 Body；
3. 校验、规范化源记录；
4. 解析引用，并在短业务事务中 upsert Organization 领域对象；
5. 写入 `OrgSyncBatch`、`OrgSyncRecord` 业务事实；
6. 返回安全的 `SyncConsumptionResult`。

它不得自行执行 HTTP、解析 Credential、调度任务、计算 Retry、修改 `next_run_at`、创建或推进 `IntegrationExecution`、覆盖 Attempt，或持久化完整 Response Artifact。唯一运行链保持为：

```text
IntegrationSyncRunner
  -> IntegrationSyncBatch
  -> IntegrationExecutionService
  -> IntegrationWorkerRunner
  -> CredentialProvider
  -> TransportClient
  -> Organization SyncResultConsumer
  -> IntegrationExecution 收敛
  -> IntegrationSyncCoordinator 推进 Checkpoint
```

Consumer 业务失败属于 `confirmed`，不进入 Integration Retry。后续定时 SyncBatch 可以从未推进的 Checkpoint 再次获得相同源记录；这是正常 Sync 链路上的业务重放，不是 Retry。

## 3. 真实证据与源契约结论

真实源提供公司、部门、岗位、人员、离职人员和内嵌任职数据。核心列表接口均为 JSON GET，没有分页、Cursor、总数、快照令牌或已声明限流。时间路径参数格式为 `yyyy-MM-dd HH:mm:ss`，没有时区偏移。

本次逐项复核的正式脱敏分析材料为：

- `docs/analysis/organization-source/OrganizationSourceApiInventory.md`；
- `docs/analysis/organization-source/OrganizationSourceFieldDictionary.csv`；
- `docs/analysis/organization-source/OrganizationSanitizedSamples.json`；
- `docs/analysis/organization-source/OrganizationSourceDataAnalysis.md`；
- `docs/analysis/organization-source/OrganizationSourceMappingDraft.md`；
- `docs/analysis/organization-source/OrganizationSourceDataQualityReport.md`；
- `docs/analysis/organization-source/OrganizationSourceOpenQuestions.md`。

同时只读核对了 Git 忽略目录中的原始 OpenAPI。没有重新采集真实人员数据，也没有调用写接口。现有脱敏样本覆盖 2 条管理公司、27 条法人公司、99 条管理部门、134 条法人部门、91 条岗位、101 个去重人员、210 个去重离职人员及 25 条内嵌兼职；该规模只用于验证结构和异常形态。

本设计采用的实测事实：

- 公司查询的时间下界在样本环境中为包含式；
- 返回结果未按 `changeTime` 单调排序；
- 部门父节点可能晚于子节点返回；
- 人员全局九天增量超过本地采集的 2 MiB 防御上限；
- 按公司查询的人员响应显著更小；
- `sendPostList` 为空，而 JSON 字符串 `sendpost` 中存在任职；
- 13 条当前任职出现相同的反转结束日期形态，疑似开放任职占位值，但尚未确认；
- 源端提供启停状态，但没有可靠物理删除事件；
- 管理、法人视图彼此独立，视图 `99` 的业务语义未确认。

### 3.1 已解决的平台兼容项：源接口只有单侧时间下界

真实 Swagger 只提供一个下界 `{time}` 参数，没有结束时间参数。已冻结的 timestamp `SyncExecutionInputPlan` 要求 `window_start_binding` 和 `window_end_binding` 指向两个不同目标。

INT-006B 通过 `SyncExecutionInputPlan version=2 + window_mode=lower_bound_only` 解决平台表达缺口：HTTP 只绑定真实下界参数，IntegrationExecution 仍冻结逻辑起止窗口，Organization Consumer 按权威 source change timestamp 过滤半开区间。V1 和 V2 `bounded_window` 仍要求真实起止双绑定。

禁止把 `window_end` 绑定到无关字段、虚构 Query 或重复绑定同一参数。`logical_window_end` 只提供给 Consumer 和 Checkpoint，不是 HTTP 参数。该能力只关闭“平台无法表达单下界接口”的兼容问题，不关闭 P0-7 的 changeTime 权威性、时区、精度、同秒完整性，也不关闭 P0-8 的人员大响应问题。

Consumer 过滤不能减少已经下载的响应。`lower_bound_only` 不限制源响应上界，不等于真正时间切片，也不允许放宽 Transport 64 MiB、持久化 Response Artifact 或落磁盘临时 Payload。初始化和 Catch-up 必须先通过实际响应量门控；超限 Task 安全失败。

## 4. Organization 九表适配审计

| 表 | V1 适配性 | 直接或转换映射 | 明确不保存 | 后续要求 |
| --- | --- | --- | --- | --- |
| `org_legal_entity` | 可承载 | 法人身份、编码、名称、简称、父法人、信用代码、状态、源更新时间 | 负责人证件、财务/人资负责人、产业标签 | 无新增字段 |
| `org_unit` | 可承载 | 管理公司、管理/法人部门身份、编码、名称、类型、法人引用、状态 | 源父子关系 | 无新增字段；需确认编码命名空间 |
| `org_structure` | 表结构可承载 | 管理、法人两套受控结构 | 视图 `99` | Organization Service 受控增加 `legal` 枚举 |
| `org_structure_node` | 可承载 | 结构位置、父节点、路径、层级、排序、状态 | 业务归属外键 | 无新增字段 |
| `org_position` | 可承载 | 稳定岗位、组织、编码/名称、状态、可选等级 | Role/Casbin 映射、猜测的管理岗标记 | 无新增字段 |
| `org_employee` | 可承载 | 稳定人员、员工编号、姓名、可空联系方式、雇佣状态 | 证件、地址、银行、照片、教育、密码、自动账号绑定 | 无新增字段；员工编号契约为 P0 |
| `org_assignment` | 可承载已确认的兼职 | 人员、法人、部门、可空岗位、类型、有效期、状态 | 猜测的主任职 | 无新增字段；主任职和日期规则为 P0 |
| `org_sync_batch` | 语义可承载，完整性需加强 | 每个 Sync Execution 一个业务批次 | HTTP、Retry、Payload 技术事实 | `execution_id` 增加 FK 和非空部分唯一约束 |
| `org_sync_record` | 语义可承载，完整性需加强 | 单个源对象的动作、结果、原因、依赖 | Payload、Attempt 技术事实 | 增加 `(batch_id, object_type, source_id)` 唯一约束和受控状态/动作 CHECK |

结论：六张 Organization 领域表可表达当前观测到的 V1 数据，不需要新增业务字段或新表。两张同步追踪表需在 Consumer 实施前补充关系和幂等约束。不建议新增直接的 `integration_sync_batch_id`：`OrgSyncBatch.execution_id -> IntegrationExecution.sync_batch_id` 已是无重复真值的关联路径。

## 5. 稳定源身份

所有 upsert 身份都受逻辑源系统约束。内部身份元组为：

```text
(source_system_code, object_kind, raw_source_id)
```

除 `org_unit` 同时容纳多类源对象外，目标表本身已提供 `object_kind` 隔离。原始源 ID 只写入领域 Source 字段并按机密标识保护；日志只能记录带密钥摘要或 SHA-256 短摘要，不记录原值。

| 对象 | V1 源 ID 候选 | 当前证据 | 持久化键规则 | 置信度 |
| --- | --- | --- | --- | --- |
| 法人 | 公司 `zjkid_ignore` | Swagger 标注 BIP ID；样本非空、唯一 | `(source_system_code, raw ID)` | 样本 high，永久性未证实 |
| 管理公司主体 | 公司 `zjkid_ignore` | 样本非空、唯一 | `management_company:<raw ID>` 写入 `org_unit.source_id` | medium |
| 管理部门主体 | 部门 `zjkid_ignore` | 样本非空、唯一 | `management_department:<raw ID>` | 样本 high |
| 法人部门主体 | 部门 `zjkid_ignore` | 样本非空、唯一 | `legal_department:<raw ID>` | 样本 high |
| 结构节点 | 结构 + 主体身份 | 源端无独立节点 ID | `<structure-code>:<unit-source-id>` | 受控派生 |
| 岗位 | `postidzjkid_ignore` | Swagger BIP ID；样本非空、唯一 | `(source_system_code, raw ID)` | 样本 high |
| 员工 | `psnidzjkid_ignore` | Swagger BIP ID；样本非空、唯一 | `(source_system_code, raw ID)` | 样本 high |
| 兼职任职 | `sendpost[].ID` | 样本非空、唯一 | `(source_system_code, raw ID)` | 样本 high |
| 顶层主任职候选 | 尚无可靠 ID | 无 assignment ID、无主职标记 | P0 关闭前不持久化 | 阻塞 |

禁止在 Source ID、NCID、名称、联系方式、编码之间相互回退。若后续证明管理和法人视图中的相同原始组织 ID 表示同一主体，必须通过显式 Crosswalk 和迁移合并；V1 宁可暂时隔离，也不做错误合并。

BIP ID 的不可变、不可复用保证仍是设计假设和生产门控。相同身份出现受保护事实冲突时，Consumer 写 `org_sync_source_id_conflict`，整片失败，不覆盖已有对象。

## 6. 管理与法人结构映射

### 6.1 两套受控结构

Adapter 确保存在两条服务端受控 `org_structure`：

| 源视图 | code | name | structure_type |
| --- | --- | --- | --- |
| 管理 `0` | `hr_management` | 管理架构 | `management` |
| 法人 `1` | `hr_legal` | 法人架构 | `legal` |

V1 不同步视图 `99`。当前 Organization 请求校验只允许 `management`，实施时需受控加入 `legal`，但不修改表结构，也不允许任意结构类型。

`org_unit` 是稳定组织主体，`org_structure_node` 只是主体在某套树中的位置。业务表和权限归属必须保存 `org_unit_id`，不得保存 `structure_node_id`。

### 6.2 公司映射

- 法人公司视图 `1` 创建或更新 `org_legal_entity`；
- 法人层级写入 `org_legal_entity.parent_id`，使用第二阶段解析；
- 不为视觉根节点而把法人公司重复复制为 `org_unit`；
- 管理公司视图 `0` 创建 `unit_type=business_unit` 的 `org_unit` 和管理结构节点；
- 法人部门通过 `primary_legal_entity_id` 关联所属法人；
- 若确认法人部门父引用指向所属法人公司而非另一部门，则该部门作为法人结构根节点，`parent_node_id=NULL`。

`code` 只能来自经确认的稳定业务编码。缺失或冲突时不得用名称替代。编码命名空间确认前，冲突以 `org_sync_business_conflict` 失败；未来如需前缀，必须成为正式冻结映射，不能是导入时的临时拼接。

### 6.3 部门映射

- 管理视图 `0`：`org_unit(unit_type=department)` + `hr_management` 节点；
- 法人视图 `1`：`org_unit(unit_type=department, primary_legal_entity_id=...)` + `hr_legal` 节点；
- 源父字段不写 `org_unit`，只用于解析节点父子关系；
- 源层级与派生层级交叉校验，完整父链解析后以树结构事实为准；
- 停用父节点仍可作为有效历史/引用父节点，不因此把子节点判孤儿。

## 7. 父节点解析算法

组织 Consumer 采用不依赖响应顺序的两阶段流程：

1. 校验身份，在受控 Chunk 中 upsert 法人或组织主体；
2. 在内存构建源父图，解析本地历史父节点、检测非法边，再按拓扑顺序 upsert 节点。

| 场景 | 处理 | Slice 结果 |
| --- | --- | --- |
| 符合已确认根规则且父为空 | 创建根节点 | success |
| 父节点在当前 Body 中后到 | 主体阶段后解析 | success |
| 父节点已在历史批次落库 | 按稳定源键解析 | success |
| 父引用为所属法人边界 | 法人部门作为根，保留 `primary_legal_entity_id` | 契约确认后 success |
| Body 和本地均无父节点 | 记录 deferred 依赖，不造假父节点 | failure |
| self-parent | `org_sync_parent_self_reference` | failure |
| 循环 | `org_sync_parent_cycle` | failure |
| 父节点跨越错误结构/类型 | `org_sync_parent_invalid` | failure |

未解析父节点不能静默转成根。Checkpoint 不推进时，后续 Batch 会在上游依赖到达后重放该窗口，因此所有 upsert 必须幂等。

## 8. 岗位映射

`postidzjkid_ignore` 是岗位身份，`postCode` 是业务编码，`postname` 是显示名称。`deptidzjkid_ignore` 解析 `org_unit_id`；依赖组织不存在时属于可等待依赖失败。

真实样本中不同组织存在同名岗位，名称绝不作为唯一键。岗位不映射为 `SysRole` 或 Casbin 角色。源岗位序列字典确认前，`position_type` 使用平台受控值 `professional`；`posLevel` 可作为长度受限的非敏感标签写入 `job_level`，不新增等级表。没有明确源标记时 `is_manager_position=false`。

`isenable=true/false` 映射启用/停用。停用岗位仍可被历史任职引用，不物理删除。

## 9. 员工映射

`psnidzjkid_ignore` 标识 `org_employee`。`jhcode` 是 `employee_no` 的暂定候选，但生产启用前必须确认唯一性和生命周期。员工编号缺失或冲突时该记录失败；严禁用姓名、手机、邮箱、账号名、NCID 或数组顺序替代。

仅映射姓名、可空手机/邮箱、受控雇佣状态、已确认的法人引用、已确认有效期和源更新时间。邮箱、手机、岗位均允许为空。

以下源字段明确不进入 Organization V1：身份证、住址、生日、银行、头像、密码、教育、政治/民族信息、身体信息及其他无关 HR 档案字段。

员工实体不等于账号。除非独立且审计完整的账号绑定流程明确关联，否则 `user_id` 保持原值或空。Consumer 不按姓名、手机、邮箱、`jhcode`、`esncode` 或 `psnid` 猜测 `SysUser`。

主任职语义确认前，人员顶层公司/部门/岗位只可做诊断校验，不写 `primary_legal_entity_id`，也不创建主职任职。

## 10. 任职设计

### 10.1 主任职候选

人员顶层的管理公司、法人公司、管理部门、法人部门和岗位看起来是当前主职候选，但 Swagger 没有说明它唯一或为主任职，也没有独立 assignment ID。

V1 不得：

- 仅凭顶层位置设置 `is_primary=true`；
- 取第一条任职为主职；
- 未经书面生命周期规则，用可变部门/岗位字段派生身份；
- 调岗时覆盖上一段任职历史。

主任职实施受“主职语义”和“稳定身份”两个 P0 门控。优先要求源端提供稳定 assignment ID。只有源负责人书面确认复合键字段在一个雇佣段内不可变，并定义调岗/再入职规则后，才允许使用复合键。

安全降级：只同步员工实体，不创建猜测的 `org_assignment`。若业务要求员工任务必须产出任职，则在缺口关闭前不得启用生产任务。

### 10.2 兼职与 `sendpost`

样本运行事实是 JSON 字符串 `sendpost`；`sendPostList` 为空，`sendposten` 是英文键镜像。P0 权威性确认后按以下规则处理：

- `sendpost[].ID` -> assignment Source ID；
- 父人员 BIP ID -> `employee_id`；
- 经确认的公司 ID -> `legal_entity_id`；
- 经确认的部门 ID -> `org_unit_id`；
- 岗位 ID -> 可空 `position_id`，空岗位是合法场景；
- 容器语义 -> `assignment_type=part_time`；
- `is_primary=false`；
- 不透明兼职子类型仅校验/诊断，不直接成为平台任意枚举；
- 公司/部门 ID 体系无法解析时记录依赖失败，禁止按名称关联。

若未来 `sendPostList` 和 `sendpost` 同时非空，Consumer 必须按稳定 assignment ID 比较规范化事实。语义完全一致只处理一次；冲突写 `org_sync_assignment_source_conflict` 并使 Slice 失败，禁止简单拼接两个数组。

### 10.3 任职有效期规范化

`NormalizeAssignmentPeriod` 是 Organization 领域唯一入口，输入源时区和经源负责人确认的开放日期规则，输出 `future`、`current`、`historical` 或错误。

| 源形态 | 规范化结果 |
| --- | --- |
| 开始有效、结束为空、当前标记 | `valid_to=NULL`，current |
| 起止有效且 end >= start | 按区间和状态得到 future/current/historical |
| 当前标记且结束值命中正式确认的占位规则 | `valid_to=NULL`，current |
| 历史标记但结束为空 | invalid |
| 反转日期且不属于已确认当前占位 | `org_sync_assignment_period_invalid` |
| 在岗/结束标记冲突 | `org_sync_assignment_status_conflict` |

P0-9 关闭前不内置魔法日期。未来任职可以用未来 `valid_from` 保存，但不能视为当前。结束任职改为 disabled 并保留，`source_deleted` 仍为 false。

## 11. 离职与状态规则

离职接口补充人员生命周期事实，不删除人员：

1. 按稳定人员 Source ID 解析员工；
2. 严格校验日期后设置 `employment_status=resigned`、`valid_to=lzdate`；
3. 在离职生效日关闭该员工所有当前 enabled 任职，不删除记录；
4. 保留员工、历史任职、岗位、组织、法人和 `user_id` 绑定；
5. 不停用或删除 `SysUser`，账号生命周期属于独立受控流程。

普通人员 `isenable=1` 映射 active；没有已确认离职事件时，`0/2` 保守映射 suspended，而不是 resigned。带有效日期的离职事件优先于更早的普通人员事件。只有源端确认再入职顺序和雇佣段规则后，更晚的再入职事实才可重新激活；否则写 `org_sync_employment_state_conflict`。

陈旧写保护使用 P0-7 最终确认的权威源更新时间，不能用到达顺序决定最终状态。

## 12. 停用与删除

源端 inactive/disabled 只映射领域停用状态，绝不设置 `source_deleted=true`。长期没有出现在增量响应中也不是删除证据。

只有未来具备以下任一正式契约时，`source_deleted` 才可设置 true：

- 含稳定身份和事件时间的明确 Tombstone；
- 另行设计、完整且一致的全量对账证明对象确实不存在。

V1 均不实现。物理删除能力不完整但保持安全：可能保留源端已不存在的记录，但不伪造删除或破坏历史。

## 13. SyncTask 与 Consumer 拆分

建议使用一个逻辑 ExternalSystem：`hr_source`。URL、Host、Credential 和内网信息只存在于 Integration 配置与 Credential Provider，不进入本文。

公司和部门是不同接口，而一个 SyncTask 只固定一个 InterfaceDefinition 和 Consumer 版本，因此 V1 使用七个 Task。下表全部使用逻辑 ExternalSystem `hr_source`，Cron 为源时区确认、P0 关闭和负载测试后的试运行建议：

| Task code | Interface code / 固定输入 | Consumer code/version | Checkpoint | Lookback / Slice | 试运行 Cron | 主要产出 |
| --- | --- | --- | --- | --- | --- | --- |
| `hr_legal_entity_sync` | `hr_company_incremental` / type 1 | `org.hr.legal_entity` v1 | timestamp，P0 gated | 10 min / 24 h | `5 * * * *` | 法人 |
| `hr_management_company_sync` | `hr_company_incremental` / type 0 | `org.hr.management_company` v1 | timestamp，P0 gated | 10 min / 24 h | `10 * * * *` | 管理根主体/节点 |
| `hr_management_department_sync` | `hr_department_incremental` / type 0 | `org.hr.management_department` v1 | timestamp，P0 gated | 10 min / 6 h | `15 * * * *` | 管理部门/节点 |
| `hr_legal_department_sync` | `hr_department_incremental` / type 1 | `org.hr.legal_department` v1 | timestamp，P0 gated | 10 min / 6 h | `25 * * * *` | 法人部门/节点 |
| `hr_position_sync` | `hr_position_incremental` | `org.hr.position` v1 | timestamp，P0 gated | 10 min / 12 h | `35 * * * *` | 岗位 |
| `hr_employee_sync` | `hr_employee_incremental` / 初期 user type 0；必要时批准公司分片 | `org.hr.employee` v1 | timestamp，P0 gated | 15 min / 1 h | `40 * * * *` | 员工及内嵌任职 |
| `hr_resigned_employee_sync` | `hr_resigned_employee_incremental` / 一个已确认权威 user type | `org.hr.resigned_employee` v1 | timestamp，P0 gated | 15 min / 6 h | `55 * * * *` | 离职与任职关闭 |

Cron 使用确认后的业务 IANA 时区计算，调度事实和窗口继续以数据库 UTC 持久化。若一个任务的实测最长处理时间接近下一任务的偏移，必须扩大错峰间隔，不得把表中分钟值当作跨环境固定 SLA。

实测离职 type 0/1 返回相同 210 个稳定人员 ID，不应两边重复执行；需源负责人确认后选一个权威视图。视图 `99`、OA 专用人员接口、带敏感查询值的人员精确搜索均排除。

SyncTask 不配置自己的 RetryPolicy；Retry 继续由 InterfaceDefinition 和 Execution 快照决定。

## 14. 首次初始化与持续增量

### 14.1 首次基线

首次基线按运营顺序串行完成：

1. 法人；
2. 管理公司；
3. 管理部门；
4. 法人部门；
5. 岗位；
6. 员工和任职；
7. 离职人员。

`initial_checkpoint_at` 必须是源负责人确认的最早可靠历史边界。禁止使用 Unix Epoch、1900 年或猜测的公司成立时间。依赖 Task 完成并完成业务核对后，才启用下一层 Task。

### 14.2 持续增量

Integration Sync V1 不提供 DAG。Cron 按同一依赖顺序错峰。试运行建议每小时执行一次并设置不同分钟偏移；确切表达式需结合真实耗时和源 SLA 配置。单活动 Batch 与 `coalesce_one` 防止停机后产生历史调度风暴。

依赖竞态通过业务失败记录和未推进 Checkpoint 处理，不在 Organization 内建立隐藏 Scheduler。上游 Task 补齐依赖后，下次定时 Batch 从旧 Checkpoint 重放。

## 15. Checkpoint、Lookback 与响应上限

计划采用 `changeTime` 作为源 Checkpoint 字段，因为文档将其描述为变更时间，且公司接口边界测试与它一致。`ts`、`synctime` 在语义确认前只作诊断。源时间按最终确认时区解析后统一转 UTC。

逻辑处理区间为半开 `[logical_start, logical_end)`。首个请求起点可减去 Lookback；重复记录通过稳定源身份和陈旧写保护收敛。源端下界包含式意味着边界重复是正常且安全的。

以下为试运行建议，不是源 SLA：

| 对象 | 试运行 Lookback | 逻辑 Slice | 超过该积压需人工评估 | 建议 Consumer 响应上限 |
| --- | ---: | ---: | ---: | ---: |
| 公司 | 10 分钟 | 24 小时 | 7 天 | 4 MiB |
| 部门 | 10 分钟 | 6 小时 | 2 天 | 8 MiB |
| 岗位 | 10 分钟 | 12 小时 | 2 天 | 8 MiB |
| 人员 | 15 分钟 | 1 小时 | 6 小时 | 16 MiB，优先按公司受控分片 |
| 离职人员 | 15 分钟 | 6 小时 | 1 天 | 8 MiB |

上述 Lookback 只是高于实测秒级精度的保守试运行缓冲，并不声称覆盖未公开的数据复制延迟。生产冻结前，P0-7 必须用源 SLA 支撑的值替换。

关键限制：当前接口没有上界，因此这些逻辑 Slice 不能减少响应大小。人员数据唯一有实证的缩小方式是按公司查询。只有源负责人提供完整、稳定的公司分片清单和 ID 命名空间后，才可配置明确的公司分片试运行；Organization 和 Integration 都不增加动态 Fan-out。

单次响应超过 InterfaceDefinition 或 Consumer 上限时，Task 失败。不得提高冻结的 64 MiB Transport 上限、持久化响应或落磁盘临时文件。若受控分区或有界源查询仍不能将响应压到限制内，该接口不兼容 V1，必须由源端改造。

## 16. SyncExecutionInputPlan

静态输入只能包含服务端受控的视图类型和经批准的公司分片 ID。窗口值必须由 Sync Binding 产生。Host、URL、Authorization、Credential、自由 Header、SQL、模板、脚本和表达式继续禁止。

Consumer 可复核 Request 中的窗口事实与对象契约，但不信任或重建 HTTP 参数。只有源契约能正确表达有界窗口后，Consumer 才按权威源更新时间处理位于逻辑窗口内的记录；窗口外记录不得用于推进 Checkpoint。

## 17. Consumer 处理管道

每个 Consumer 都是特定对象范围的强类型 Adapter，不构建动态 ETL：

```text
校验请求元数据
  -> 流式解码 JSON Envelope 和受控 DTO 字段
  -> 校验稳定 ID 与必需语义
  -> 规范化 Source DTO
  -> 解析引用
  -> 分 Chunk upsert 领域对象
  -> upsert OrgSyncRecord
  -> 收敛 OrgSyncBatch 摘要
  -> 返回 SyncConsumptionResult
```

未知源字段忽略且不持久化。Envelope 类型、`success`、`data`、选定字段类型、记录数、内嵌任职数和字符串长度均设边界。`sendpost` 使用独立 JSON Parser 和上限，绝不作为代码执行。

建议 V1 处理边界：

- 公司/组织/岗位：每个事务最多 500 条源记录；
- 员工：每个事务最多 200 人；
- 每人内嵌任职最多 100 条，编码后的任职字符串最多 256 KiB；
- Consumer 只获取一次 `Body()` 副本并流式解码，不保留第二份完整 DTO 数组。

这些是服务端常量，负载测试只能调低。提高 Chunk 或 Consumer 响应上限前必须证明内存和租约预算仍成立。

## 18. 业务幂等与事务边界

领域 upsert 使用稳定源键，并比较权威源更新时间/版本事实。Lookback 重复应收敛为 no-op 或确定性更新。名称、联系方式、当前部门和返回顺序都不能识别对象。

一次 Execution 的业务处理流程：

1. 按 `execution_no` 解析 `IntegrationExecution`，复核 SyncBatch、Task、Slice、Consumer 事实；
2. 使用唯一 `execution_id` find-or-create 一个 `OrgSyncBatch`；
3. 在事务外解码和规范化；
4. 每个受控 Chunk 使用独立短业务事务持久化；
5. Chunk 内原子 upsert 领域对象和对应 `OrgSyncRecord`；
6. 全部 Chunk 后，从已持久化 Record 聚合结果，并在短事务中收敛 `OrgSyncBatch`；
7. 返回 `business_reference=OrgSyncBatch.batch_no`。

Integration 不开启 Consumer 的业务事务，Consumer 也不在 HTTP 期间持有事务。Chunk 后崩溃可安全恢复：规范化记录可重复 upsert，唯一业务记录防止重复。

若 Consumer 已提交全部业务数据，而 Integration 完成事务随后失败，Runtime 可能收敛 `failed + unknown`，且不得自动重发。未来正常 SyncBatch 通过 Lookback 再次获得相同源记录时，Organization upsert 必须继续幂等。

## 19. 部分业务失败与 Checkpoint

V1 不允许“99 条成功、1 条静默跳过”后推进 Checkpoint。

Consumer 可以继续处理互不依赖的记录并提交有效 Chunk，但只有每条影响 Checkpoint 的源记录都属于以下结果时，最终才返回 success：

- created；
- updated；
- 按正式规则 disabled/resigned/closed；
- 精确幂等 no-op。

任一稳定 ID 缺失、依赖未解析、循环、必需枚举非法、有效期非法、业务冲突或持久化失败，均写 `OrgSyncRecord` failure/deferred，并使 Consumer 整体失败。Integration Execution 按业务失败收敛，SyncBatch failed，Checkpoint 不推进。已提交的正确记录在旧 Checkpoint 下重复到达时无副作用。

该规则有意选择“可见的 Checkpoint 阻塞”，而不是永久丢数。永久非法记录必须由源端修正，或后续单独设计有审计的接受/隔离流程；V1 没有“接受并永久跳过”。

只有不影响身份、层级、状态、有效期和必需引用的非建模可选字段，才允许忽略且不使 Slice 失败；它们记为成功 no-op 诊断，不记业务失败。

## 20. OrgSyncBatch 与 OrgSyncRecord

一个 HR Slice Execution 对应一个 `OrgSyncBatch`。一个 Integration SyncBatch 可以经其多个 Slice Execution 关联多个 Organization 业务批次。

`OrgSyncBatch` 只保存业务范围和聚合结果：

- 平台生成的 `batch_no`；
- 唯一 `execution_id`；
- 由受控运行模式产生的 `sync_type`，不能因时间很早就声称 full；
- 受控 `object_scope`；
- 开始/完成时间、计数、状态和脱敏错误摘要。

`OrgSyncRecord` 保存单个源对象结果：

- `batch_id`、受控 `object_type`、Source ID/Code、可空目标 ID；
- Action：`create|update|disable|close|noop|error|deferred`；
- 状态、稳定 Reason Code、脱敏依赖类型/键；
- 不保存 Response Body、Source DTO、员工姓名/联系方式、Attempt、Retry、Credential 或 Header。

不重复保存 `integration_sync_batch_id`，技术关联统一为：

```text
OrgSyncBatch.execution_id
  -> IntegrationExecution.sync_batch_id
  -> IntegrationSyncBatch
```

`OrgSyncRecord.retry_count`、`last_retry_at` 在 V1 保持 0/null，它们不是 Integration Retry 控制项。

## 21. 稳定业务 Reason Code

| Code | 分类 | 含义 |
| --- | --- | --- |
| `org_sync_envelope_invalid` | 契约 | 响应 Envelope/类型非法 |
| `org_sync_source_id_missing` | 永久数据 | 必需稳定 ID 缺失 |
| `org_sync_source_id_conflict` | 冲突 | 相同源身份与受保护事实冲突 |
| `org_sync_parent_unresolved` | 可等待依赖 | 父节点尚未到达 |
| `org_sync_parent_self_reference` | 永久数据 | 自引用 |
| `org_sync_parent_cycle` | 永久数据 | 组织循环 |
| `org_sync_parent_invalid` | 永久数据 | 父节点跨越错误结构/类型 |
| `org_sync_reference_missing` | 可等待依赖 | 法人、组织、岗位或员工引用缺失 |
| `org_sync_assignment_invalid` | 永久数据 | 任职结构非法 |
| `org_sync_assignment_period_invalid` | 永久数据 | 任职有效期无法规范化 |
| `org_sync_assignment_status_conflict` | 永久数据 | 任职状态标记冲突 |
| `org_sync_primary_assignment_ambiguous` | 契约 | 主任职语义/身份未确认 |
| `org_sync_assignment_source_conflict` | 契约 | 结构化/字符串任职源冲突 |
| `org_sync_enum_unknown` | 契约/数据 | 必需枚举未映射 |
| `org_sync_employment_state_conflict` | 业务冲突 | 陈旧、再入职或离职事实冲突 |
| `org_sync_business_conflict` | 业务冲突 | 受保护的本地/源唯一事实冲突 |
| `org_sync_persistence_failed` | 系统 | 短业务事务失败 |

返回 Integration 的错误消息只包含通用描述，不包含源值。业务 Record 仅保存稳定 Code 和脱敏依赖摘要。上述失败均不能被 Integration Retry 自动重试。

## 22. 待确认问题与门控

### 22.1 P0：相关 Consumer 开发或启用前必须关闭

| P0 | 当前证据 | 阻塞原因 | 安全降级 |
| --- | --- | --- | --- |
| 单下界源接口与双边界 Sync Plan 冲突 | Swagger 只有 `{time}`，冻结 Plan 要求 start/end | 无法正确配置 timestamp Task 或声称响应切片 | 不启用 timestamp Task；取得源端上界或批准 Sync 变更 |
| BIP ID 永久且不复用 | 文档称 BIP ID，样本非空/唯一 | 所有幂等 upsert 依赖身份 | 源/对象复合隔离、冲突停止、不合并不删除；仅试运行 |
| 权威更新时间、时区、精度、同秒完整性 | 公司 `changeTime` 下界包含；无时区 | 错误 Checkpoint 会漏数 | 确认前不启用生产 Checkpoint；明确转换后才存 UTC |
| 法人/组织/岗位业务编码及命名空间 | 源提供多套 code，但无全量唯一性保证 | 目标模型要求受控唯一 code，不能用名称或敏感 ID 代替 | 冲突即失败；不临时拼接掩盖，待正式冻结命名空间 |
| 法人部门根节点与公司父引用语义 | 父引用可能不在同一增量部门样本 | 错判会制造孤儿或错误根 | 未确认的父引用按 unresolved 失败，不静默转根 |
| 员工编号身份 | `jhcode` 仅是候选 | 目标字段必填且唯一 | 缺失/冲突失败，不使用联系方式/姓名回退 |
| 顶层任职是否唯一主任职 | 顶层单值，Swagger 未声明 | 决定 `is_primary` 和权限语义 | 仅同步员工，不创建主职 |
| 主任职稳定 ID 与生命周期 | 顶层无 assignment ID | 无法幂等保存调岗/历史 | 要求源 ID 或正式保证的复合键 |
| `sendpost` 权威性和双源优先级 | `sendPostList=[]`，`sendpost` 有值 | 可能漏掉或重复全部兼职 | 未批准源非空时失败并保持 Checkpoint |
| 兼职 ID 体系和结构枚举 | 部分命中 NCID，无字典 | 可能错误关联必需外键 | 不按名称解析，引用不明则失败 |
| 开放任职结束日期占位 | 重复反转日期疑似占位 | 错误规范化会关闭当前任职 | 不猜魔法日期，受影响任职失败 |
| 人员响应上限与首次初始化 | 全局九天超过采集上限，无分页/上界 | Transport/内存风险，基线可能不完整 | 受控按公司试运行；无法限流则要求源端改造 |

### 22.2 P1：生产上线前必须关闭

- 源物理删除/Tombstone 契约，或明确接受 V1 永不标记删除；
- 跨视图部门是否为同一组织主体；
- 人员状态、离职优先级、再入职和雇佣段规则；
- 兼职子类型和岗位等级字典；
- 历史、未来任职返回范围；
- 精确复制延迟、生产 Lookback、限流、超时和调用时段；
- 普通人员、离职人员的权威 user type；
- 首次基线最早可靠时间和源快照一致性预期。

### 22.3 P2：后续优化

- 视图 `99` 与混合结构；
- OA 专用人员接口差异；
- 在存在稳定编码分片后评审产业过滤；
- 组织负责人、岗位序列/等级和额外 HR 档案产品化；
- 受控全量对账与物理删除页面；
- 更丰富的业务修复和人工接受隔离；
- Organization 同步高级筛选与 Dashboard。

## 23. 页面与权限职责

V1 不新增第三套“HR 同步中心”。

| 管理需求 | 页面 |
| --- | --- |
| 配置 Interface/Credential/Retry/Schedule/Window，运行 Task | 集成中心 / 同步任务 |
| 查看技术 Batch、Checkpoint、Slice、Execution、Attempt、Retry | 集成中心 / 同步批次、执行记录、调用日志 |
| 查看业务映射计数和业务失败 | Organization / 业务同步批次与记录 |
| 处理 Organization 本地状态 | 现有 Organization 详情和权限流程 |

Organization 业务页面只展示安全的对象类型、动作、状态、Reason Code、脱敏源引用、目标链接和计数。不展示 Response Body、Input Snapshot、Credential、日志中的人员姓名/联系方式或原始 `sendpost`。Organization 现有功能权限和 Data Permission 继续生效；Integration 权限不授予 Organization 业务数据访问。

## 24. 安全与资料治理

- 原始 OpenAPI、响应继续保存在 Git 忽略的受限目录；
- 脱敏样本只作为设计证据，不得成为运行 Seed 或生产数据测试 Fixture；
- Response Body 只存在于 Consumer 调用栈，不记录、不持久化；
- Audit 只记录 Task/Consumer/Batch 标识与计数，不记录人员或源 Payload；
- 结构化日志只记录 Execution、Batch、对象类型、Source ID 摘要、Reason Code、数量和耗时；
- `ErrorSummary`/`ErrorMessage` 只保存长度受限的脱敏信息，不保存源端或数据库原始错误；
- Source ID 按机密标识处理，不作为默认列表字段；
- 证件、银行、地址、生日、照片、密码、Authorization、Cookie、Token 和 Credential 不进入 Organization 同步存储。

建议在独立隐私/安全复核后，把脱敏的接口清单、字段字典、映射、质量报告、问题清单和通过机器扫描的脱敏样本纳入受控 Git 资料，使后续评审可复核设计依据。本 Task 不修改 `.gitignore`，也不提交这些材料。原始响应、内网 Host、Token 和任何可重新识别的对照表必须继续不跟踪。

## 25. 推荐实施 Task

P0 源契约确认是正式 Gate，不属于隐藏开发工作。

### INT-006B：源契约门控与 Organization 同步完整性基础

- 关闭或正式记录 P0 结论；
- 经变更控制确认单下界时间接口的 Integration 兼容方案；
- 增加 `OrgSyncBatch.execution_id` FK/部分唯一和 `OrgSyncRecord` 幂等约束；
- 增加受控 `legal` 结构校验和 Organization HR Reason Code；
- 建立强类型 Source DTO、Normalizer、Source Key Helper 和 Consumer 测试框架；
- 暂不注册生产 Consumer。

### INT-006C：法人和双组织结构 Consumer

- 实现法人、管理公司、管理部门、法人部门处理器；
- 实现两阶段父解析、循环/自引用检测、状态映射和结构初始化；
- 仅在身份/时间门控关闭后注册四个 Consumer 版本；
- PostgreSQL 幂等和父节点晚到 E2E。

### INT-006D：岗位 Consumer

- 实现岗位、组织引用和状态映射；
- 验证同名不同组织和历史引用；
- 不创建 Role，不修改权限。

### INT-006E：员工实体 Consumer

- 实现员工身份、可空联系方式、源时间陈旧写保护和账号边界；
- 建立内存/响应防御和公司分片试运行；
- 不提前宣称任职完整。

### INT-006F：任职与离职 Consumer

- 受主任职、`sendpost`、ID 体系和有效期 P0 门控；
- 实现主任职/兼职、有效期规范化、离职排序、任职关闭和再入职冲突；
- 证明无猜测主职、无假引用、无账号删除。

### INT-006G：端到端验收与冻结

- 真实 PostgreSQL、Integration SyncRunner、WorkerRunner、TLS HR Mock、注册的 Organization Consumer、Retry 联动、Checkpoint 和浏览器验收；
- 覆盖基线/增量顺序、Lookback 幂等、依赖重放、部分失败、内存/租约、DTO/日志脱敏；
- 只有全部 P0 Gate 和安全不变量通过后，才生成 Organization HR 验收报告和 Freeze Review。

若源端无法提供有界窗口契约，INT-006B 必须先形成经评审的兼容决定，INT-006C 才能开始。Organization Consumer 不得以自建 Scheduler、Proxy、Artifact Store 或第二条 HTTP 链路补偿。

## 26. V1 最终冻结建议

1. 不新增 Organization 领域表，不保存完整 HR 响应。
2. 实施前补齐现有同步表的最小关系和幂等约束。
3. BIP ID 是按源和对象类型隔离的暂定稳定键，永久性是明确 Gate。
4. 管理和法人结构分开，视图 `99` 排除。
5. 法人公司进入法人表，管理公司是管理架构业务单元根，部门是 Unit/Node。
6. 岗位按源 ID 和组织识别，不按名称，不映射 Role。
7. 员工不等于账号；联系方式和岗位允许为空。
8. 没有源证据时不推断主任职。
9. `sendpost` 的权威性和 ID 体系确认前不进入正式同步。
10. 开放任职日期使用经确认的统一 Normalizer，不猜占位值。
11. 离职保留员工、账号绑定和任职历史。
12. 停用不等于删除，V1 的 `source_deleted` 保持 false。
13. 七个 SyncTask 隔离接口、Checkpoint、依赖和业务失败范围。
14. 初始化和增量按依赖顺序错峰，不在 Organization 内建 DAG/Scheduler。
15. 任一影响 Checkpoint 的记录失败都会使 Consumer 失败并保持 Checkpoint。
16. 一个 Sync Execution 对应一个 Organization 业务批次，技术事实继续留在 Integration。
17. 响应大小只能由源端有界查询或受控分区解决，不提高 Runtime 上限、不存 Payload。
18. 单下界平台表达缺口已由 V2 受控扩展解决；changeTime 权威性、时区、精度、同秒完整性和人员响应规模 P0 未关闭前，禁止生产启用。

## 27. INT-006B 实现基础与 Gate 状态

INT-006B 建立 `backend/internal/organization/hrsync` 源适配边界：

- 强类型 `SourceKey(source_system_code, object_kind, raw_source_id)`，对象类别白名单、长度限制，日志格式只输出 SHA-256 摘要；
- 按公司、部门、岗位、员工、离职员工和任职拆分的最小源 DTO，未知 JSON 字段忽略，证件、地址、银行等敏感字段不进入 DTO；
- source-independent 的 LegalEntity/OrgUnit/Position/Employee canonical input；
- 法人、组织、岗位、员工纯 Normalizer；
- Assignment Normalizer 只保留接口并返回 `org_sync_source_contract_unconfirmed`，不猜主任职、`sendpost` 权威性、内部 ID 或开放日期；
- 全部 Organization HR 稳定 Reason Code；
- `management`、`legal` 两个固定 structure type；
- test consumer harness 验证 Source DTO -> Normalizer -> OrgSyncBatch -> OrgSyncRecord -> SyncConsumptionResult，不注册生产 HR Consumer。

数据库完整性采用 `OrgSyncBatch.execution_id -> IntegrationExecution.id` RESTRICT FK 和非空部分唯一索引；每个 Integration Execution 最多对应一个 Organization 业务批次。`OrgSyncRecord` 对非空稳定源 ID 使用 `(batch_id, object_type, source_id)` 部分唯一约束；源 ID 缺失的错误记录允许空 ID，但必须使用 `action=error`。动作固定为 `create/update/disable/close/noop/error/deferred`，状态继续使用现有受控集合。旧开发动作编码按显式表迁移，`skip/no_change` 统一收敛为 `noop`；不增加 `integration_sync_batch_id`。

人员按公司分区仍是后续受控配置能力：partition ID 必须由服务端批准并作为非敏感 static input 固定在 Task 版本中；不得提供任意 ID 输入、动态读取 Organization fan-out 或 Organization Scheduler。该设计不能替代 P0-8 的生产响应量验证。

截至 INT-006B，以下 P0 继续打开：BIP ID 永久稳定/不可复用；changeTime 权威字段、时区、正式精度和同秒完整性；法人/组织/岗位编码命名空间；员工编号生命周期；主任职及其稳定 assignment ID；`sendpost` 权威性和兼职 ID 体系；开放任职日期规则；人员大响应生产方案；源物理删除表达。代码没有以布尔开关或环境变量伪造关闭这些 Gate，相关生产 Consumer 不注册。
