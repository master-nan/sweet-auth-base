# Sweet Platform Organization HR 任职与离职源契约评审

## 1. 文档状态

| 项目 | 内容 |
| --- | --- |
| Task | INT-006F-A |
| 日期 | 2026-08-13 |
| 状态 | 评审完成；关键任职 Gate 未关闭，INT-006F-B 仅允许安全子集 |
| 性质 | 源契约审计、证据确认、实施门控，不包含代码或数据变更 |
| 正式设计 | `docs/_construction/design/OrganizationHRSyncDesign.md` |

本评审只回答现有资料能够证明什么。样本中的非空、唯一和一致只属于样本事实，不等于永久稳定、不可复用或生产契约保证。没有证据的规则继续保持未知，不以默认值、数组顺序、名称匹配或经验推断补齐。

## 2. 证据范围与方法

本轮重新读取并交叉核对：

- `docs/_construction/analysis/organization-source/OrganizationSourceApiInventory.md`；
- `docs/_construction/analysis/organization-source/OrganizationSourceFieldDictionary.csv`；
- `docs/_construction/analysis/organization-source/OrganizationSanitizedSamples.json`；
- `docs/_construction/analysis/organization-source/OrganizationSourceDataAnalysis.md`；
- `docs/_construction/analysis/organization-source/OrganizationSourceMappingDraft.md`；
- `docs/_construction/analysis/organization-source/OrganizationSourceDataQualityReport.md`；
- `docs/_construction/analysis/organization-source/OrganizationSourceOpenQuestions.md`；
- Git 忽略目录中的原始 OpenAPI；
- 已保存的按公司人员响应、管理/法人离职响应。

本轮没有重新采集人员数据，没有调用 POST、人员敏感搜索或照片接口。统计仅输出数量、字段名和匹配关系；正式文档不包含真实姓名、联系方式、证件号、邮箱或真实源 ID。

证据状态只使用：

| 状态 | 含义 |
| --- | --- |
| `confirmed` | OpenAPI 有明确语义，且保存样本没有反证；仍不自动等同永久生命周期承诺 |
| `partially confirmed` | 只确认字段形态、有限样本事实或部分语义，仍不足以启用完整生产规则 |
| `unconfirmed` | OpenAPI 与样本不足以作出所需业务判断 |

## 3. 主任职语义

### 3.1 顶层字段审计

人员对象顶层存在管理公司、法人公司、管理部门、法人部门、岗位、`entrydate`、`begindate`、`lzdate`、`type` 和 `isenable`。OpenAPI 对这些字段的描述分别是公司/部门/岗位、入职日期、开始日期、离职日期、结束标志和启停状态。

OpenAPI 没有出现 `primary assignment`、主任职、主岗位、任职关系、雇佣段或人员岗位关系等契约描述，也没有说明顶层组织/岗位：

- 必然代表主任职；
- 在同一时点唯一；
- 永远非空；
- 是当前值而不是历史投影；
- 调岗后是否保留上一段；
- 是否包含未来任职。

101 个去重人员样本的顶层公司、部门和岗位均非空，只能证明本样本没有顶层缺失。顶层是单值且另有名为“兼职”的容器，属于结构暗示，不是主任职契约。

### 3.2 结论

`P0 主任职语义` 为 `unconfirmed`，不关闭。

安全降级保持不变：顶层组织、法人和岗位只用于受控诊断，不持久化 `org_assignment`，不设置 `is_primary=true`，不写员工主法人，不据此关闭或迁移任职。

## 4. 主任职 Stable Assignment ID

OpenAPI 中没有发现独立的 assignment ID、employment relation ID、post relation ID、主岗位关系 ID、雇佣段 ID 或人员岗位关系 ID。顶层 `begindate` 仅描述为“开始日期”，没有生命周期或唯一性保证。

OpenAPI 提供 `ifreentry`，描述为“二次入厂”；样本观察到多个不透明值。但文档没有枚举解释，也没有把它关联到任职或雇佣段 ID。它只能作为源 Adapter 的诊断候选，不能生成任职身份。

以下复合键都不成立：

- 员工 + 当前部门 + 当前岗位：调岗会改变组合；
- 员工 + `begindate`：字段是否属于任职段未知；
- 员工 + 法人/组织：跨组织和再入职会产生歧义；
- 员工 + 数组位置或“第一条”：没有稳定性。

`P0 主任职 stable assignment identity` 为 `unconfirmed`，不关闭。源方必须提供稳定关系 ID，或书面保证一组字段在一个雇佣段内不可变并明确调岗、历史和再入职规则。

## 5. `sendpost`、`sendposten` 与 `sendPostList`

### 5.1 OpenAPI 契约

- `sendpost`：字符串，描述为“兼职信息中文”，没有声明字符串内部 JSON Schema；
- `sendposten`：字符串，描述为“兼职信息英文”，没有声明字符串内部 JSON Schema；
- `sendPostList`：结构化数组，元素包含 `id`、时间、启停、兼职类型、公司、法人公司、部门、法人部门、岗位和岗位等级。

结构化 `sendPostList.id` 描述为 BIP ID；其公司、部门和岗位子对象使用 `newCenturyId`，OpenAPI 明确描述为 NCID。

### 5.2 保存样本事实

101 个去重人员中：

- `sendPostList` 全部为空；
- 12 人的 `sendpost` 可解析出共 25 条兼职；
- `sendposten` 也可解析出 25 条；
- 中英文容器的 25 个关系 ID、时间、状态和空岗位字段逐项一致；
- 中英文容器的公司 ID 和部门 ID 在 25 条中全部不同，因此不能把 `sendposten` 简单称为字段值完全相同的镜像；
- 没有观察到 `sendPostList` 与字符串容器同时非空，无法验证双源对齐和冲突优先级。

### 5.3 结论

当前只能确认字符串容器在保存样本中承载了兼职事实，不能确认 `sendpost` 永久权威，也不能确认未来版本不会切换到 `sendPostList`。

`P0 sendpost 权威性` 为 `unconfirmed`，不关闭。源负责人仍需确认：生产权威字段、字符串内部 Schema、两类字符串的语义差异、结构化数组启用计划，以及多源同时非空时的优先级和冲突规则。

## 6. `sendpost[].ID`

25 条样本中，`sendpost[].ID`：

- 全部非空；
- 样本内全局唯一；
- 没有跨人员重复；
- 与同批已采集的人员、公司、部门和岗位 BIP/NCID 集合均无重合；
- 与 `sendposten[].id` 一一相同。

这些事实强烈支持它是关系 ID，而不是人员、组织或岗位 ID。结构化 `sendPostList.id` 又被 OpenAPI 描述为 BIP ID，进一步提供了形态支持。

但 OpenAPI 没有明确声明字符串 `sendpost[].ID` 与结构化 `sendPostList.id` 是同一字段，也没有保证它跨窗口、跨人员、调岗、停用、删除或再入职时不可变且不可复用。

结论：`sendpost assignment ID` 为 `partially confirmed`。它可作为后续 Adapter 测试中的唯一候选，不足以启用生产兼职 upsert；仍需 HR 负责人书面确认关系 ID 语义和生命周期。

## 7. 兼职公司、部门与岗位 ID 空间

### 7.1 OpenAPI 证据

结构化 `sendPostList` 的公司、法人公司、部门、法人部门和岗位引用均通过 `newCenturyId` 表达，OpenAPI 分别描述为公司 NCID、部门 NCID和岗位 NCID。因此结构化契约指向 NCID 空间，不是 Organization 当前 SourceKey 使用的 BIP ID 空间。

### 7.2 样本 Cross-check

对 25 条字符串兼职做现有增量样本交叉核对：

| 引用 | 样本事实 | 结论 |
| --- | --- | --- |
| 中文公司 ID | 25 条非空；11 条命中已采集法人公司 NCID，0 条命中公司 BIP ID | 支持 NCID，但样本不全 |
| 中文部门 ID | 25 条非空；在当前有限部门增量中未命中 NCID 或 BIP ID | 无法判定失效；不能完成引用解析 |
| 中文岗位 ID | 25 条全部为空 | 合法空岗位场景，无法验证岗位 ID 空间 |
| 英文公司/部门 ID | 与中文值在 25 条中全部不同，当前增量集合也未形成可靠命中 | 语义未知，不能与中文值互换 |

结论：`兼职组织 ID 体系` 为 `partially confirmed`，不关闭。结构化契约明确指向 NCID，但当前 Organization 领域身份使用 BIP SourceKey。生产兼职 Consumer 必须先获得受控 NCID -> BIP Crosswalk，或让源端提供 BIP ID；不得绕过 SourceKey、按名称解析或直接把 NCID 当 BIP ID。

## 8. 兼职岗位可空

OpenAPI 将结构化 `sendPostList.post` 标为 nullable。保存样本中 25/25 条兼职的岗位和岗位等级引用均为空，而公司、部门和关系 ID 都存在。

结论：`兼职 position_id 可空` 为 `confirmed`，该 Gate 关闭。后续 Domain 输入必须允许兼职 `position_id=NULL`，不得补造岗位，也不得因空岗位拒绝一条其他关键事实完整的兼职。

## 9. 任职时间与开放日期

### 9.1 原始资料复核结果

对 25 条保存的 `sendpost` 原始记录重新统计：

- 13 条为 `在岗=Y` 且 `结束兼职=N`；其结束时间全部为空；
- 12 条为 `在岗=N` 且 `结束兼职=Y`；其中 11 条有非空结束时间且 `end >= start`，1 条结束时间为空；
- 14 条结束时间为空；
- 非空时间中没有 `end < start`；
- 没有观察到未来开始时间。

OpenAPI 的结构化 `sendPostList.endDate` 明确 nullable。保存的脱敏样本也保留了空结束时间形态。

这与早期分析文档中“14 条日期反转、其中 13 条同一疑似占位值”的派生结论不一致。只读原始响应和正式脱敏样本均不支持该反转结论。本评审以原始响应复核为准，并在 `OrganizationHRSyncDesign.md` 中更正；不修改历史分析材料的原始记录。

### 9.2 结论

可以确认：空/nullable 结束时间是源结构允许的真实形态；当前样本没有需要识别的魔法结束日期。

不能确认：历史标记但结束为空的业务含义、是否存在未采集到的占位日期、状态与日期的完整状态机。

`P0 开放任职日期规则` 为 `partially confirmed`，不完全关闭。安全规则为：

- 只有显式 null/空结束时间且当前状态标记一致时，Adapter 才可规范为开放结束；
- 历史标记但结束为空，返回 `org_sync_assignment_status_conflict` 或 `org_sync_assignment_period_invalid`；
- 未经源方确认，不识别任何魔法日期；
- 非空 `end < start` 始终失败，不做占位猜测。

## 10. 离职接口

### 10.1 OpenAPI 证据

接口摘要明确为“同步离职人员”。响应对象提供：

- `psnidzjkid_ignore`：描述为 BIP ID；
- `lzdate`：明确描述为离职日期；
- `changeTime`：描述为更改时间；
- `isenable`：0/1/2 启停描述；
- 顶层组织与岗位投影；
- 结构化 `sendPostList`。

离职响应不包含 `entrydate`、`begindate`、`ifreentry`、`sendpost` 或 `sendposten`，也没有雇佣段或离职事件 ID。

### 10.2 保存样本事实

管理和法人两个请求各返回 210 条：

- 每侧均为 210 个唯一、非空人员 BIP ID；
- `lzdate` 和 `changeTime` 全部非空；
- `isenable` 全部为 `2`；
- `sendPostList` 全部为空；
- 没有未来离职日期；
- 45 条离职日期早于本次查询下界，而其 `changeTime` 均在查询下界之后，说明接口能返回“历史离职事实的后续变更”，不能用 `lzdate` 代替增量时间；
- 每侧同一人员只出现一次，当前样本没有多段离职历史；
- 两个视图的人员 ID 集合、离职日期、状态和更新时间一致。

没有观察到未来离职，也没有观察到“人员已经重新 active，但离职接口仍返回旧离职段”的可判定样本。接口名称可以确认这些记录是离职人员事实；普通人员接口也出现 `isenable=2`，因此不能反向推出所有状态 2 人员都是离职人员。

结论：`lzdate` 的离职日期字段语义为 `confirmed`，该字段语义 Gate 关闭。员工 BIP ID 在样本内可关联普通人员与离职人员，但永久稳定/不可复用 Gate 仍打开。`changeTime` 作为路径下界权威字段、时区、精度和同秒完整性没有边界测试或正式说明，保持 `partially confirmed`。

## 11. 离职 `type=0/1`

普通人员接口的 OpenAPI 摘要明确 `userType=0` 为管理、`1` 为法人；离职接口仅声明 `userType` 整数参数，没有给出 0/1 说明。

保存响应表明两侧并非完全相同视图：210 人的离职事实一致，但管理公司/部门相关投影字段在全部 210 条上存在差异。证据支持“同一离职人员集合的不同组织投影”，但不能证明这在所有时间窗口永久成立，也不能指定某一侧为离职事件的唯一权威源。

结论：`离职 type` 为 `partially confirmed`，不关闭。生产只允许在源负责人明确指定权威 `userType` 后配置一个离职 Task；禁止同时运行两个视图造成重复业务处理，也禁止仅因当前集合相同便默认选择 0。

## 12. 普通人员与离职事实优先级

现有证据可以支持以下有限规则：

1. 来自明确“离职人员”接口且 `lzdate` 合法的记录，是离职事实，不等同普通人员接口中的一般停用；
2. 普通人员 `isenable=2` 本身仍只能映射 `suspended`，不能单独推断 resigned；
3. 若权威源时间已经确认且离职事实严格晚于普通 active 事实，可将员工收敛为 resigned；
4. 若普通 active 事实晚于离职事实，不得自动解释为再入职；应返回 `org_sync_employment_state_conflict`，等待雇佣段规则；
5. 时间相同但状态冲突、时间不可比较或来源不明确时，不按到达顺序覆盖。

保存的有效普通人员样本与离职样本只有 2 人重叠，两人的状态、离职日期和 `changeTime` 一致。该重叠量不能验证状态冲突、延迟更新或再入职顺序。

结论：`普通人员/离职优先级` 为 `partially confirmed`。明确离职事件可以作为 resigned 事实，但跨接口排序和重新激活 Gate 不关闭。

## 13. 再入职

普通人员 OpenAPI 存在 `ifreentry`，描述为“二次入厂”；样本观察到多个不透明值。OpenAPI 未给出枚举含义，离职响应没有该字段，也没有 hire segment、employment history ID 或独立再入职事件。

因此当前模型虽能保留一个员工实体，却无法仅凭现有资料可靠区分首次入职、离职和一个或多个再次入职雇佣段。不能使用“同一人员 ID 后来重新 active”自动推断再入职，也不能用 `entrydate`/`begindate` 生成段 ID。

结论：`再入职` 为 `unconfirmed`，不关闭。`ifreentry` 仅作为 Adapter 诊断字段候选，不进入 Organization Domain 通用规则。

## 14. Gate 矩阵

| Gate | 当前状态 | 证据 | 是否关闭 | 安全降级 |
| --- | --- | --- | --- | --- |
| 主任职语义 | `unconfirmed` | 顶层单值；OpenAPI 无主职/唯一说明 | 否 | 只作诊断，不写 assignment |
| 主任职 stable ID | `unconfirmed` | 无关系 ID/雇佣段 ID；`begindate` 无生命周期保证 | 否 | 不派生复合键，不写主职 |
| `sendpost` 权威性 | `unconfirmed` | 字符串有 25 条，结构化数组全空；无优先级契约 | 否 | 生产不消费兼职 |
| `sendpost` assignment ID | `partially confirmed` | 25/25 非空唯一且不与其他 ID 空间重合；无生命周期承诺 | 否 | 仅作测试候选，需书面确认 |
| 兼职组织 ID 体系 | `partially confirmed` | 结构化契约为 NCID；公司 11/25 命中法人 NCID；部门未形成 Crosswalk | 否 | 要求 Crosswalk 或源端 BIP ID |
| 兼职岗位可空 | `confirmed` | OpenAPI nullable；25/25 实测为空 | 是 | `position_id=NULL`，不造岗位 |
| 开放任职日期 | `partially confirmed` | 14 条空 end；OpenAPI end nullable；无反转魔法日期证据 | 否 | 只接受显式空且状态一致，不认魔法日期 |
| 离职日期 | `confirmed` | 离职接口和 `lzdate` 描述明确；两侧共 420 个视图行均非空 | 是 | 严格解析，非法值失败 |
| 离职更新时间 | `partially confirmed` | `changeTime` 全部非空且样本符合查询下界；无时区/边界/同秒保证 | 否 | 生产 Checkpoint 继续 Gate |
| 离职 type | `partially confirmed` | 人员集合和离职事实相同，组织投影不同；接口未描述枚举 | 否 | 源方指定一个权威视图 |
| 普通人员/离职优先级 | `partially confirmed` | 明确离职接口可证明 resigned；冲突与再入职样本不足 | 否 | 离职可收敛，后续 active 记冲突 |
| 再入职 | `unconfirmed` | 有 `ifreentry` 标签，无枚举、事件或雇佣段 ID | 否 | 不自动重新 active |
| BIP ID 永久性 | `partially confirmed` | OpenAPI 标注 BIP ID；样本非空唯一 | 否 | SourceKey 隔离、冲突停止、不合并 |
| `changeTime` 全局契约 | `partially confirmed` | 公司含下界实测；任职/人员/离职无完整契约 | 否 | 不启用生产时间 Checkpoint |

本轮关闭的只是两个局部语义：兼职岗位可空、`lzdate` 是离职日期。它们不关闭 BIP ID、权威时间、来源视图或完整任职生命周期 Gate，因此不构成生产 HR 任职 Task 的启用授权。

## 15. INT-006F-B 准入范围

### 15.1 允许立即实现的安全子集

1. `HrLeaveUserView` 最小 Source DTO、离职日期解析和严格字段校验；
2. 基于 Employee SourceKey 的离职幂等处理框架；
3. 在显式测试 SourceContract 下，将合法离职事实收敛为 `org_employee.employment_status=resigned` 和受控 `valid_to`；
4. 保留 Employee、`user_id`、SysUser 和历史事实，不删除、不自动停用账号；
5. 对较早普通 active + 较晚离职的确定性测试；
6. 对较晚 active、同时间冲突和不可比较时间返回 `org_sync_employment_state_conflict`；
7. 仅关闭已明确属于 HR 来源且当前有效的 assignment；若没有可识别的 HR assignment，则保持 no-op，不用顶层组织/岗位补造；
8. `sendpost`/`sendposten` 的受限 Parser、关系 ID 一致性检查、空岗位和空结束时间测试骨架，但不做生产持久化；
9. 生产 Consumer 可按现有 Registry 模式静态登记为 `disabled`，不得被 Task 选择或 Resolve。

### 15.2 仍禁止实现

- 顶层主任职创建、`is_primary=true` 或主职复合键；
- 以 `sendpost`、`sendposten` 或 `sendPostList` 任一方作为永久权威源；
- 未经书面确认将 `sendpost[].ID` 当永久 assignment Source ID；
- 直接以 NCID 解析 Organization BIP SourceKey，或按名称关联；
- 识别任何“魔法结束日期”；
- 自动再入职或将后到 active 直接覆盖 resigned；
- 未确认 `userType` 时选择离职 0 或 1；
- 在 changeTime Gate 未关闭时启用生产 Checkpoint；
- 因离职删除员工、任职历史或 SysUser。

结论：允许进入 INT-006F-B，但只能实现上述安全子集。任职生产落库仍被主任职、`sendpost` 权威性、关系 ID 生命周期和 NCID Crosswalk 阻塞；离职生产启用仍被权威 `userType`、BIP ID 生命周期和时间契约阻塞。

## 16. Source Adapter 与 Organization Domain 边界

### 16.1 当前 HR Source Adapter 特有

- `sendpost`/`sendposten` 字符串 JSON 解析和双源比较；
- `sendPostList` 兼容与权威源选择；
- 中文/英文字段名映射；
- NCID -> BIP Crosswalk；
- `兼职架构`、兼职类型和状态枚举；
- OpenAPI 的 `userType` 视图参数；
- `ifreentry` 枚举解释；
- 源时间格式、时区和 changeTime 下界语义；
- 任何经源方确认的特殊日期规则。

### 16.2 Organization Domain 通用

- assignment 必须有稳定、源隔离的身份；
- `position_id` 可空；
- 有效期必须规范为合法区间；
- 同一员工当前主任职唯一，但只有上游明确主职时才设置；
- resigned 不删除员工、账号绑定或历史任职；
- 合法离职事实可以关闭受控当前任职；
- stale write、同版本冲突和业务幂等；
- 业务失败不进入 Integration Retry，Checkpoint 只在整片成功后推进。

BIP 的字符串结构、ID 命名和视图参数不得进入 Organization Domain Service。

## 17. 最终结论

INT-006F-A 源契约评审完成。现有资料不足以关闭主任职、主任职稳定身份、`sendpost` 权威性、兼职关系 ID 生命周期、NCID Crosswalk、离职权威视图、跨接口排序和再入职 Gate。

本轮确认了兼职岗位可空和 `lzdate` 的离职日期语义，并纠正了早期“反转结束日期”结论：当前保存语料是显式空结束时间，不存在可据此冻结的魔法日期。INT-006F-B 可以继续，但仅限离职安全子集、冲突保护和任职 Parser/Test Harness；不得宣布完整任职与离职契约已经关闭。
