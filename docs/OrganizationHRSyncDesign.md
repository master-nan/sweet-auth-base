# Sweet Platform Organization HR 同步详细设计

## 1. 文档状态

| 项目 | 内容 |
| --- | --- |
| Task | INT-006A |
| 状态 | 详细设计完成；INT-006G 已完成 Adapter V1 能力验收与冻结，真实 HR 源生产启用仍受 Gate 阻断 |
| 日期 | 2026-08-12 |
| 最近更新 | 2026-08-13（INT-006G） |
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
- 25 条兼职中 13 条当前任职结束时间为空；原始响应复核未发现非空结束时间反转，不存在可据此冻结的魔法日期；
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
| 兼职任职候选 | `sendpost[].ID` | 25 条样本非空、唯一且不与其他已采集 ID 空间重合；字符串字段生命周期未获承诺 | P0 关闭后才使用 `(source_system_code, raw ID)` | 样本 high，生产阻塞 |
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

样本运行事实是 JSON 字符串 `sendpost`；`sendPostList` 为空。`sendposten` 与中文容器的关系 ID、时间、状态和空岗位逐项一致，但 25 条公司/部门引用值全部不同，不能视为字段值完全相同的英文镜像。P0 权威性、双源优先级和引用 ID 体系确认后才按以下规则处理：

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

INT-006F-A 对保存原始响应重新统计：13 条当前任职均为空结束时间；12 条历史任职中 11 条为合法非空区间，1 条结束时间为空；不存在非空结束时间反转。早期“13 条相同反转占位值”的派生结论由本次原始资料复核更正。P0 关闭前仍不内置魔法日期；只允许显式 null/空且当前状态一致时规范为开放结束，历史状态但结束为空属于冲突。未来任职可以用未来 `valid_from` 保存，但不能视为当前。结束任职改为 disabled 并保留，`source_deleted` 仍为 false。

## 11. 离职与状态规则

离职接口补充人员生命周期事实，不删除人员：

1. 按稳定人员 Source ID 解析员工；
2. 严格校验日期后设置 `employment_status=resigned`、`valid_to=lzdate`；
3. 在离职生效日关闭该员工所有当前 enabled 任职，不删除记录；
4. 保留员工、历史任职、岗位、组织、法人和 `user_id` 绑定；
5. 不停用或删除 `SysUser`，账号生命周期属于独立受控流程。

普通人员 `isenable=1` 映射 active；没有已确认离职事件时，`0/2` 保守映射 suspended，而不是 resigned。OpenAPI 明确离职接口和 `lzdate` 字段语义，因此在权威源时间可比较时，带合法日期的离职事实可以优先于更早的普通 active 事实。更晚的普通 active 不能自动解释为再入职；源端确认再入职枚举、顺序和雇佣段规则前写 `org_sync_employment_state_conflict`。普通人员虽有描述为“二次入厂”的 `ifreentry`，但枚举和生命周期未说明，不能据此重新激活。

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
| `sendpost` 权威性和双源优先级 | `sendPostList=[]`，字符串有值；中英文公司/部门引用不一致 | 可能漏掉、重复或错误关联全部兼职 | 未批准源不落库并保持 Checkpoint |
| 兼职 ID 体系和结构枚举 | 关系 ID 仅样本唯一；结构化组织/岗位引用契约为 NCID，当前领域身份为 BIP | 缺少生命周期承诺和 NCID Crosswalk | 不按名称解析，不把 NCID 当 BIP，引用不明则失败 |
| 开放任职结束日期与状态组合 | OpenAPI end nullable；13 条当前 end 为空，另有 1 条历史 end 为空；无非空反转样本 | 历史空 end 与未知占位规则可能误判当前状态 | 只接受显式空且当前标记一致，不猜魔法日期 |
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

### INT-006F-A：任职与离职源契约评审

- 只读复核 OpenAPI、脱敏样本和保存响应；
- 形成 `OrganizationHRAssignmentContractReview.md` Gate 矩阵；
- 关闭兼职岗位可空和 `lzdate` 字段语义两个局部结论；
- 主任职、`sendpost` 权威性、关系 ID 生命周期、NCID Crosswalk、离职视图、跨接口排序和再入职继续门控。

### INT-006F-B：任职与离职安全子集

- 允许实现离职 Source DTO、合法 `lzdate`、Employee resigned 收敛、历史保护和冲突处理；
- 允许实现受限任职 Parser/Test Harness、空岗位和显式空结束时间校验，但不允许生产兼职持久化；
- 不实现主任职、自动再入职、魔法日期或 NCID 绕过；
- 生产 Consumer 在权威视图、BIP 生命周期和 changeTime Gate 关闭前保持 disabled。

### INT-006G：端到端验收与冻结

- 真实 PostgreSQL、Integration SyncRunner、WorkerRunner、TLS HR Mock、注册的 Organization Consumer、Retry 联动、Checkpoint 和浏览器验收；
- 覆盖基线/增量顺序、Lookback 幂等、依赖重放、部分失败、内存/租约、DTO/日志脱敏；
- 将 Adapter Capability Freeze 与真实 HR Source Production Enablement 分层判断：安全不变量通过即可冻结 Adapter 架构；全部相关源 Gate 关闭后才允许生产启用 Consumer。

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

## 28. INT-006C 法人和组织结构 Consumer 实现

INT-006C 按对象职责实现四个静态 Consumer：`org.hr.legal_entity`、`org.hr.management_company`、`org.hr.management_department`、`org.hr.legal_department`，版本均为 v1。法人公司只写 `org_legal_entity`；管理公司写 `org_unit(unit_type=business_unit)` 和 `hr_management` 节点；管理部门写 `org_unit(unit_type=department)` 和 `hr_management` 节点；法人部门写独立 `org_unit(unit_type=department)`、`primary_legal_entity_id` 和 `hr_legal` 节点。名称不参与身份识别，管理与法人视图不会因名称相同自动合并。

领域持久身份使用 `SourceKey.PersistenceID()`；独立领域表由表边界隔离对象类别，共享的 `org_unit` 则把 `object_kind` 纳入 `source_id` 空间。原始 Source ID 不写日志或业务追踪记录，`OrgSyncRecord.source_id` 保存不可逆短摘要，依赖键也只保存摘要。源业务编码仍是受控属性和冲突边界，不替代 SourceKey；与另一稳定身份发生唯一编码冲突时整片失败，不拼接随机后缀。

公司和部门采用两阶段处理。Phase 1 在每个最多 500 条的短事务中 upsert 全部主体，并为组织主体建立不可执行的结构节点占位；Phase 2 从当前 Slice 和历史数据共同解析父关系，计算路径和层级。父节点在 Body 中晚到不受返回顺序影响；历史已同步父节点可复用；停用但关系完整的父节点保留为历史父引用。父缺失、父仍在 `dependency_waiting` 或法人引用缺失时记录为 `deferred`/`dependency_waiting`；self-parent、两节点或多节点循环、跨结构引用与稳定身份冲突记录为 `error`/`failed`。两类异常都写稳定 Reason Code，使 Consumer 整体失败且 Checkpoint 不推进；不会自动转根、造假父节点或按名称查父级。

源 inactive 只映射领域对象和节点 `disabled`，`source_deleted` 始终保持 false，不物理删除对象或结构事实。Lookback 和同一 Execution 的重复 Consume 通过领域 SourceKey、业务 Batch 唯一关系、Record 唯一约束和陈旧事实检查收敛；future-of-logical-window 记录不写领域对象，也不写当前 Slice 成功 Record。

每个 Sync Execution 仍只对应一个 `OrgSyncBatch`。Consumer 开始时复核 Execution、IntegrationSyncBatch、Task、Slice 和 Consumer code/version；业务 Record 只保存对象类型、Source 摘要、目标 ID、Action、状态、Reason Code 和脱敏依赖摘要。任何关键记录失败都返回业务失败，Runtime 将其收敛为 confirmed `business_processing_failed`，不得进入 Integration Retry。

`hr_management` 和 `hr_legal` 由平台 Seed 幂等创建，结构类型固定为 `management`/`legal`。生产装配静态注册上述四个 code/version，但在当前源 P0 未关闭时元数据状态为 `disabled`，因此 Task 配置无法选择、Resolve 或启用它们。PostgreSQL/E2E 测试通过显式 `SourceContract(source_system_code, source timezone)` 启用同一实现；不存在按环境猜测主职或时区的临时开关。

INT-006C 未关闭任何既有源 Gate。BIP ID 永久稳定和不可复用、changeTime 权威性/时区/精度/同秒完整性、编码命名空间、法人部门根边界、员工编号、主任职、`sendpost`、兼职 ID、开放任职日期、人员大响应和物理删除表达继续保持 P0/P1 原结论。进入 INT-006D 只允许实现岗位已确认范围；生产 HR Task 仍不得启用。

## 29. INT-006D 岗位 Consumer 实现

INT-006D 增加静态 `org.hr.position` v1 Consumer，保持 `HRPositionSourceDTO -> NormalizePositionSource -> PositionSyncInput -> OrganizationHRSyncService` 边界。岗位身份只使用 `SourceKey(hr_source, position, postidzjkid_ignore)`；名称、`postCode`、部门名称、NCID 和数组位置都不参与身份回退。生产注册在岗位 BIP ID、changeTime 和编码命名空间 Gate 关闭前保持 `disabled`，不能被 SyncTask 选择或 Runtime Resolve；PostgreSQL/E2E 通过显式 SourceContract 启用同一实现。

岗位只写 `org_position`。`deptidzjkid_ignore` 必须解析已同步且业务同步状态为 `synced` 的管理部门 `org_unit`，不会按名称搜索、创建假组织、使用结构节点 ID 或跨到法人结构。依赖缺失时不创建岗位，写 `deferred/dependency_waiting + org_sync_reference_missing`，Consumer 整体失败且 Checkpoint 不推进；组织补齐后由下一轮 Lookback 依靠稳定键收敛。

`postCode` 是受控业务属性，不是同步身份。现有模型同时具有源内 `(source_system_code, code)` 唯一约束和启用岗位的组织内 code 约束；INT-006D 不降低这些约束。不同 Source ID 使用相同 `postCode` 时返回 `org_sync_business_conflict`，不覆盖、不拼随机后缀或临时组织前缀。不同 Source ID、不同 code、同名岗位可以在不同 `org_unit` 独立存在。

缺少权威岗位序列规则时，`position_type` 固定为平台受控 `professional`，`is_manager_position=false`；不按岗位名称或等级猜测管理岗。`posLevel` 仅作为最长 64 字符、允许为空的非敏感 `job_level` 显示属性，不创建等级、Role、Casbin 或权限。源 inactive 映射 `status=disabled`，保留历史引用并保持 `source_deleted=false`。

岗位每个 Chunk 最多 500 条，在短事务中完成稳定身份锁、管理部门引用解析、code 冲突检查、领域 upsert 和 OrgSyncRecord；解码、窗口过滤与 Normalize 在事务外。持久化 `source_version=SourceChangedAt.UTC().Format(RFC3339Nano)`，旧版本只产生 no-op，同一源时间但事实冲突返回 `org_sync_source_id_conflict`，不使用数据库当前时间伪造源版本。

每个 Position Slice Execution 仍只对应一个 OrgSyncBatch。Record 只保存 `object_type=position`、Source ID 摘要、目标岗位 ID、`create|update|disable|noop|error|deferred`、状态、Reason Code 和脱敏依赖摘要；不保存岗位 DTO、Body、Header、Credential、Attempt 或 Retry。future-of-logical-window 岗位不落库、不写当前 Slice 成功记录；任何关键记录失败都由 Runtime 收敛为 confirmed `business_processing_failed`，不会进入自动 Retry。

PostgreSQL 16 的 Runner/TLS E2E 覆盖：首次 503 后技术 Retry 成功、同名岗位跨组织独立创建、Lookback 重放不重复、后一 Slice 停用、future 过滤和 Checkpoint 推进；另一路覆盖缺组织导致业务失败、Checkpoint 保持、无额外 Retry，以及补齐组织后自动重放成功。INT-006D 不关闭 BIP ID 永久稳定/不可复用、changeTime 权威性/时区/精度/同秒完整性、岗位编码命名空间、员工编号、主任职、`sendpost`、兼职 ID、开放任职日期、人员大响应或物理删除表达 Gate。

## 30. INT-006E 员工实体 Consumer 实现

INT-006E 增加静态 `org.hr.employee` v1 Consumer，保持 `HREmployeeSourceDTO -> NormalizeEmployeeSource -> EmployeeSyncInput -> OrganizationHRSyncService` 边界。员工身份只使用 `SourceKey(hr_source, employee, psnidzjkid_ignore)`；`jhcode`、姓名、手机、邮箱、`esncode`、NCID、`psnid` 和数组位置都不参与身份回退。生产装配登记固定 code/version，但在 BIP ID、changeTime、员工编号和响应量 Gate 关闭前保持 `disabled`，不能被 SyncTask 选择或 Runtime Resolve。

`jhcode` 仅作为最长 128 字符且非空的 `employee_no` 候选。现有模型要求 `(source_system_code, employee_no)` 唯一；不同 SourceKey 使用同一编号时返回 `org_sync_business_conflict`，不查找后覆盖、不使用联系方式补值，也不拼公司、部门或随机后缀。员工编号生命周期仍未获得权威证据，因此该实现不能据此启用生产 Task。

V1 只保存姓名、可空手机、可空邮箱、雇佣状态和源时间。普通人员 `isenable=1` 映射 `active`，`0/2` 映射 `suspended`；不会映射 `resigned`，不会写员工或任职有效期。人员顶层公司、部门、岗位和 `sendpost` 不进入 canonical input，不创建 `org_assignment`、不设置 `primary_legal_entity_id`、不推断主任职。更新只使用 Organization 来源字段白名单，已有 `user_id`、本地备注和标签保持不变；Consumer 不创建、绑定或停用 `SysUser`。

陈旧写保护持久化 `source_version=SourceChangedAt.UTC().Format(RFC3339Nano)` 和 `source_updated_at`。旧版本返回 `noop`；同一源时间且受保护事实相同返回 `noop`；同一源时间但员工编号、姓名、联系方式或雇佣状态不同返回 `org_sync_source_id_conflict`。同一 Body 的重复 SourceKey 先按源时间收敛，时间相同但事实不同则整片失败，不依赖到达顺序、数据库时间或 Batch 顺序。

员工 Consumer 的业务响应上限为 16 MiB，低于 Runtime 64 MiB 绝对上限；InterfaceDefinition 仍可配置更低限制。源 Envelope 使用流式 Decoder 逐条转成精简 canonical input，不把完整源 DTO 数组持久化；领域写入每个短事务最多 200 人。`sendpost` 本阶段只做边界防御：原始字符串最多 256 KiB、JSON 数组最多 100 项且每项必须是对象；不会解析任职 ID、日期、部门或岗位，也不会改变员工成功语义。超限或结构非法以 `org_sync_envelope_invalid` 使 Consumer 失败，不写 Payload、Response Artifact 或临时文件，也不提高 Transport 上限。

每个 Employee Slice Execution 对应一个 `OrgSyncBatch`。`OrgSyncRecord` 只保存 `object_type=employee`、不可逆 Source 摘要、目标员工 ID、动作、状态和稳定 Reason Code；不保存姓名、手机、邮箱、`sendpost`、Body、Header、Credential、Attempt 或 Retry。future-of-logical-window 人员不落库、不写当前 Slice 成功 Record；Lookback 和重复 Consume 依靠 SourceKey、源版本、业务 Batch/Record 唯一约束收敛。任何关键员工失败都由 Runtime 收敛为 confirmed `business_processing_failed`，不得进入自动 Retry，Checkpoint 仅在整片业务成功后推进到 `logical_window_end`。

PostgreSQL 16 的 SyncRunner + WorkerRunner + TLS E2E 覆盖：首轮 HTTP 503 由 Integration Retry 后成功、固定测试公司分区 static input 注入、两片连续消费、空联系方式、重复人员幂等、后片更新为 suspended、future 过滤和 Checkpoint 推进；另一路覆盖同源同时间事实冲突导致业务失败、Checkpoint 保持且 Consumer 失败不产生额外 Retry。测试分区值只属于显式 Test SourceContract。生产公司分区 ID 完整清单和命名空间尚未确认，因此没有建立可供生产选择的任意值入口或动态 Fan-out；生产 Consumer 继续 disabled。

INT-006E 不关闭 BIP ID 永久稳定/不可复用、changeTime 权威性/时区/精度/同秒完整性、员工编号生命周期、公司分区清单、主任职及其稳定 assignment ID、`sendpost` 权威性、兼职 ID 体系、开放任职日期、人员大响应生产策略、物理删除表达或离职/再入职完整规则。进入 INT-006F 只能实现经权威材料确认的任职和离职范围，不能依据本次员工实体同步推断这些语义。

## 31. INT-006F-A 任职与离职源契约评审结论

正式证据与 Gate 矩阵见 `docs/OrganizationHRAssignmentContractReview.md`。本轮没有修改代码、数据库或生产 Consumer，也没有重新采集真实人员数据。

已确认的局部语义只有：

1. 兼职岗位可以为空，`org_assignment.position_id=NULL` 是合法场景；
2. 离职接口的 `lzdate` 明确表示离职日期；
3. 明确离职事实可在权威源时间可比较时使员工收敛为 resigned，但普通 `isenable=2` 仍不能单独等同离职。

仍未关闭：顶层主任职语义和稳定身份、`sendpost` 永久权威性、兼职关系 ID 生命周期、NCID -> BIP Crosswalk、历史空结束时间语义、离职权威 `userType`、changeTime 时区/精度/同秒完整性、跨接口冲突排序和再入职雇佣段。

INT-006F-B 只能实现离职安全子集和任职 Parser/Test Harness。生产任职落库继续禁止；离职生产启用也必须等待权威视图、BIP 生命周期与 changeTime Gate 关闭。当前 BIP 字段、字符串容器、`userType` 和 `ifreentry` 均属于 Source Adapter 规则，不得进入 Organization Domain 通用服务。

## 32. INT-006F-B 离职安全子集实现

INT-006F-B 增加静态 `org.hr.resigned_employee` v1 Consumer，保持 `HRResignedEmployeeSourceDTO -> NormalizeResignedEmployeeSource -> ResignationSyncInput -> OrganizationHRSyncService` 边界。源 DTO 只接收 `psnidzjkid_ignore`、`changeTime` 和 `lzdate`；姓名、员工编号、联系方式和其他人员字段不进入离职 canonical input。员工身份只使用 `SourceKey(hr_source, employee, psnidzjkid_ignore)`，员工不存在时写 `deferred + org_sync_reference_missing`，不会从离职事件创建残缺 Employee。

`lzdate` 按严格 `YYYY-MM-DD` LocalDate 解析，空值、非法日期、日期时间和明显越界年份均拒绝。现有 Organization 字段为 timestamp，因此持久化使用 UTC 午夜作为 LocalDate 的无时区载体；该值不是离职事件 instant，不参与跨接口排序。事件排序只使用显式 SourceContract 解析出的 `changeTime`：较旧离职事实为 noop；较新的明确离职事实把 Employee 收敛为 `employment_status=resigned` 并写 `valid_to=lzdate`；同一事件时间但离职事实冲突返回 `org_sync_employment_state_conflict`。离职后的较新普通 active 事件同样返回该冲突，不能自动解释为再入职；普通 suspended 事件不得反转已确认 resigned 状态。

每条离职记录在一个短事务中锁定 Employee，先读取并校验该员工全部未删除 Assignment，再更新 Employee、关闭当前 enabled Assignment 并写 Employee/Assignment 业务记录。合法关闭只修改既有 Assignment 的 `valid_to=lzdate`、`status=disabled`、`sync_status=synced`，不创建任职、不推断主任职、不删除历史；`lzdate < valid_from` 返回 `org_sync_assignment_period_invalid`，Employee 与 Assignment 均不提交。已有 `user_id`、本地备注和标签保持不变，Consumer 不创建、停用或解绑 SysUser。每个 Chunk 最多 500 条离职记录。

每个离职 Slice Execution 仍只对应一个 `OrgSyncBatch`。Employee 记录使用 `object_type=employee`；实际关闭的任职使用 `object_type=assignment` 和 `action=close`，Source ID 只保存不可逆摘要。同一 Execution、同一事件和 Lookback 重放依靠 Employee 源版本、Assignment 当前状态以及 `(batch, object_type, source_id)` 唯一约束收敛为 noop，不重复关闭或创建记录。任一员工缺失、日期非法、事件冲突、任职周期冲突或持久化失败都会使 Consumer 整体失败，Runtime 收敛为 confirmed `business_processing_failed`，不进入 Integration Retry，Checkpoint 不推进。future-of-logical-window 离职事实不落库、不写当前 Slice 成功记录。

任职侧只新增 Source Adapter/Test Harness。`AssignmentSourceParser` 解析 JSON 字符串数组，限制 256 KiB、最多 100 项、每项必须为对象；关系 ID 必须非空且 Slice 内唯一，组织引用按 NCID 候选保留，岗位引用允许为空，未知字段忽略且字符串从不执行。`NormalizeAssignmentPeriod` 只接受已确认组合：当前任职允许空结束时间；合法起止区间要求 `end >= start`；历史任职必须有结束时间；状态与日期冲突稳定拒绝；不识别任何魔法日期。输出 `AssignmentSourceCandidate` 不是可持久化的 `AssignmentSyncInput`，生产 Employee/Resigned Consumer 都不会据此创建 Assignment。

`OrganizationSourceCrosswalkResolver` 只作为当前 HR Source Adapter 的显式端口。V1 没有新增 Crosswalk 表，默认实现稳定返回不可用；NCID 不得作为 BIP ID 使用，也不得通过名称、业务 code 或临时拼接解析。生产兼职落库在 `sendpost` 权威性、关系 ID 生命周期和 NCID -> BIP Crosswalk Gate 关闭前继续禁止。

生产装配登记固定 `org.hr.resigned_employee` code/version，但和其他 HR Consumer 一样保持 `disabled`，不会出现在 SyncTask 可选列表，也不能被 Runtime Resolve。PostgreSQL 16 的 SyncRunner + WorkerRunner + TLS E2E 通过显式 Test SourceContract 启用同一实现，覆盖首次 HTTP 503 后由 Integration Retry、离职 Employee 收敛、Assignment 关闭、重复事实 noop、future 过滤与 Checkpoint 推进；独立场景还覆盖 Employee 缺失 deferred、任职周期冲突整条回滚、旧离职事件 noop，以及离职后较新 active 产生业务冲突。三类业务失败均保持 Checkpoint 且只产生一个 Attempt，未进入 Integration Retry。测试没有引入 userType 权威选择或环境变量 Gate。

INT-006F-B 不关闭以下 Gate：BIP ID 永久稳定/不可复用；changeTime 权威性、时区、精度、同秒完整性与跨接口排序；离职权威 `userType`；主任职语义和稳定 Assignment ID；`sendpost` 永久权威性；兼职关系 ID 生命周期；NCID -> BIP Crosswalk；历史空结束日期完整语义；自动再入职和雇佣段；人员大响应生产策略；物理删除表达。V1 仍不支持主任职创建、顶层组织/岗位转任职、生产兼职落库、自动再入职、魔法日期、Employee/Assignment 删除或 SysUser 生命周期变更。

## 33. INT-006G 验收与两层冻结结论

正式证据见 `docs/OrganizationHRSyncAcceptanceReport.md`，能力冻结见 `docs/OrganizationHRSyncFreezeReview.md`。INT-006G 以 INT-006F-B 实现提交为基线，完成代码、Migration、Registry、七类 Consumer、PostgreSQL 16、Integration SyncRunner + WorkerRunner + TLS、Retry/Checkpoint、页面、权限和脱敏验收。

真实初始化顺序 E2E 按法人、管理公司、管理部门、法人部门、岗位、员工、离职自动运行；首次 HTTP 503 只由 Integration Retry 在同一 Execution 追加 Attempt，业务成功后才推进 Checkpoint。另一个 E2E 在同一 Slice 验证 N-1 成功、1 条父依赖失败、Checkpoint 保持，以及补齐依赖后 Lookback 重放和幂等收敛。生产路径没有第二套 HTTP、Retry、Scheduler、Checkpoint 或 Execution 状态入口。

Organization 业务查询 DTO 已收敛为安全 `source_summary`、`dependency_summary` 和稳定 Reason Code；不安全的遗留原值 fail-closed。前端补充了 Sync/Execution/Organization 列表与详情的 query/detail 权限守卫。真实浏览器确认管理员技术/业务页面、深色模式和动态按钮正常；无权限账号为零菜单，Organization/Integration 相关直达路由为 404，且没有预加载受保护 API。

正式分层结论：

1. **Organization HR Adapter V1 Capability：通过冻结。** 已实现法人、双组织结构、岗位、员工、离职安全子集，以及任职 Parser/Test Harness 和 Crosswalk 边界；Source DTO -> Canonical Input -> Domain、SourceKey、registered Consumer、Business failure 不 Retry、Checkpoint、业务 Batch/Record 与 Payload 不持久化规则被冻结。
2. **Current HR Source Production Enablement：不允许。** 七类生产 Consumer 继续保持 disabled。BIP ID 生命周期、changeTime 完整契约、对象编码、员工编号、人员响应量、离职权威视图、主任职、任职 ID、sendpost、Crosswalk、再入职和物理删除等 Gate 没有被本次验收伪造关闭。

V1 明确不支持主任职同步、生产兼职同步、自动再入职、物理删除、动态公司 Fan-out、全量对账、视图 99、OA 专用人员源、Response Artifact 和 HR 业务脚本。下一步只允许源契约确认、容量试验和逐 Consumer 生产准入，不允许继续用默认规则扩展业务语义。
