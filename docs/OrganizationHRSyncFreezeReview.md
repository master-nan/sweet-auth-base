# Sweet Platform Organization HR Adapter V1 冻结评审

## 1. 冻结背景

Integration Runtime、Retry 和 Sync V1 已完成正式冻结。INT-006A 至 INT-006F-B 在唯一 Integration 执行链上建立 Organization HR Source Adapter、业务映射、业务 Batch/Record 和离职安全子集。INT-006G 以 `4787aaa3448a74c13efb5ac3584cfd9b26496bd0` 为实现基线完成代码审计、PostgreSQL 16、七类初始化顺序、TLS、Retry、Checkpoint、race、前端和真实浏览器验收。

正式证据见 [OrganizationHRSyncAcceptanceReport.md](OrganizationHRSyncAcceptanceReport.md)。本评审冻结的是 **Adapter 架构和已验收安全能力**，不是当前真实 HR 源的生产契约。

## 2. 两层结论

| 层级 | 结论 |
| --- | --- |
| Organization HR Adapter V1 Capability | **通过冻结** |
| Current HR Source Production Enablement | **不允许** |

生产不允许的原因是外部源契约 Gate 尚未关闭，不是通过代码默认值可以补偿的实现缺口。七类生产 Consumer 必须继续保持 disabled；显式 Test SourceContract 不能用于生产。

## 3. 唯一扩展链

```text
HR Source DTO
  -> Source Normalizer
  -> Source-independent Canonical Input
  -> Organization Domain Service
  -> OrgSyncBatch / OrgSyncRecord
  -> SyncConsumptionResult
```

上游远程调用只允许：

```text
IntegrationSyncRunner
  -> IntegrationExecution
  -> IntegrationWorkerRunner
  -> CredentialProvider
  -> TransportClient
  -> registered Organization Consumer
  -> Integration 收敛与 Checkpoint
```

Organization 不得新增 HTTP Client、Credential 读取、Scheduler、Retry、Checkpoint 或 Execution 状态推进。

## 4. Source DTO 与 Canonical Input

当前 HR 字段名、Envelope、日期格式、`sendpost`、`userType`、`ifreentry` 和 NCID 只能存在于 `internal/organization/hrsync` Source Adapter。Domain Service 只依赖 SourceKey、Canonical Input 和领域语义，不依赖 `psnidzjkid_ignore`、`postidzjkid_ignore`、`jhcode` 等源字段。

后续替换源系统时应新增 Source DTO/Normalizer，复用相同 Canonical/Domain 边界；不得把 Adapter 变成动态 ETL、脚本、反射、SQL 或插件运行时。

## 5. SourceKey

稳定身份固定为：

```text
(source_system_code, object_kind, raw_source_id)
```

不得以名称、员工编号、业务 code、手机号、邮箱、NCID 或数组位置回退。原始 ID 不进入日志、DTO、Audit 或业务 Record 默认展示；只允许不可逆安全摘要。身份永久性未获源方确认时，冲突必须停止，不得覆盖或自动合并。

## 6. Registered Consumer 与生产 Gate

七类 code/version 固定静态注册：

- `org.hr.legal_entity` v1；
- `org.hr.management_company` v1；
- `org.hr.management_department` v1；
- `org.hr.legal_department` v1；
- `org.hr.position` v1；
- `org.hr.employee` v1；
- `org.hr.resigned_employee` v1。

Registry 不支持动态加载、脚本、反射方法、SQL 或客户端任意 Consumer。生产启用必须由受控 SourceContract 和外部 Gate 证据驱动；不能通过环境变量、默认时区、默认 ID 稳定性或管理员按钮绕过。

## 7. 双组织结构

management 和 legal 是固定独立结构。`org_unit` 是业务组织主体，`org_structure_node` 只是结构位置，业务归属不得保存 node ID。主体 upsert 与父关系解析分阶段进行，响应顺序不构成前置条件。

缺父可 deferred 但 Consumer 必须失败并保持 Checkpoint；self-parent、cycle、跨结构和身份冲突稳定失败。不得自动转根、创建假父、按名称找父或混合两棵树。停用不等于删除。

## 8. Position 规则

岗位只按 SourceKey 识别并引用 org_unit。岗位名称不是唯一键，同名跨组织可并存；业务 code 是受控属性，冲突失败而不是拼接后缀。缺少权威岗位序列时类型固定 `professional`，管理岗标识不猜测。岗位不映射 Role、Casbin 或权限。

## 9. Employee 与 Account 分离

员工实体不等于 SysUser。同步允许手机号/邮箱为空，不以联系方式或姓名识别，不自动创建、绑定、停用账号，不修改已有 `user_id`。普通人员只收敛已确认 active/suspended 语义，不直接映射 resigned，不根据顶层组织/岗位创建 Assignment。

陈旧写保护只使用已确认 SourceContract 的源版本；数据库时间、到达顺序和 Batch 顺序不得充当源版本。

## 10. Resignation 安全语义

离职日期使用严格 LocalDate。明确离职事实可以把已有 Employee 收敛为 resigned，并在同一短事务中关闭该员工真实存在且周期合法的当前 Assignment。保留 Employee、账号绑定和历史任职；不创建任职、不删除对象、不修改 SysUser。

员工缺失 deferred，周期冲突整体回滚，旧事件 noop，离职后较新 active 在再入职契约关闭前必须冲突停止。权威 `userType` 和跨接口排序 Gate 未关闭时生产 Consumer 保持 disabled。

## 11. Assignment 不猜测

V1 只冻结 Assignment Parser、已确认日期组合和 Crosswalk 边界，不冻结生产任职同步。人员顶层组织/岗位、`sendpost`、`sendPostList` 均不得创建 Assignment；不得自动 primary/part_time、不得 NCID 当 BIP、不得名称 Crosswalk、不得魔法日期。

Parser 输出不是持久化输入。默认 Crosswalk Resolver 不可用。未来任职实现必须在主任职语义、关系 ID 生命周期、源权威性和 Crosswalk Gate 关闭后另行设计和验收。

## 12. Business Failure 与 Retry

HTTP/Transport 失败继续由 RetryDecision 唯一决策。Organization Consumer 的引用缺失、源冲突、数据质量、周期冲突和持久化失败属于 confirmed `business_processing_failed`，不得进入 retry_waiting。

Lookback 重放是未推进 Checkpoint 后的新 Sync 运行，不是 Consumer 自行 Retry。Organization 不得调用 Transport、修改 `next_run_at` 或创建平行 Execution。

## 13. lower_bound_only 与 Checkpoint

V2 `lower_bound_only` 只解决源接口没有 HTTP end 参数的表达问题。Execution 仍冻结逻辑半开窗口；future 记录不消费；整片业务成功后 Checkpoint 才推进 logical_end。该模式不保证 Response 有界，也不允许放宽 Transport 64 MiB、保存 Artifact 或写临时 Payload。

任何影响 Checkpoint 的关键记录失败都必须使 Consumer 失败。部分成功对象可以提交，但 Checkpoint 不得越过失败对象；后续 Lookback 依靠 SourceKey 和源版本幂等收敛。

## 14. OrgSyncBatch 与 OrgSyncRecord

一个 Integration Execution 最多对应一个 OrgSyncBatch。OrgSyncBatch 记录业务对象范围和计数，不复制 Attempt、HTTP、Retry、Credential 或 Payload。OrgSyncRecord 以 `(batch_id, object_type, source_id)` 约束稳定业务事实，只允许受控 Action/状态、Reason Code、安全源摘要和目标 ID。

查询 DTO 必须 fail-closed：不安全的遗留 source/error/dependency 文本不得原样返回。技术页面权限不自动授予 Organization 业务结果权限。

## 15. 事务与资源

法人/组织和岗位 Chunk 上限 500，员工 200，离职 500。解码和 Normalize 在事务外，领域 upsert 与 Record 使用短事务。离职单记录的 Employee、Assignment 关闭和 Record 必须原子。

Employee Body 上限 16 MiB，`sendpost` 上限 256 KiB/100 项；所有 Consumer 继续受 Interface/Registry 与 Runtime 上限保护。禁止提升 Transport 64 MiB、持久化完整响应、Response Artifact 或磁盘临时 Payload。

## 16. 权限、页面与脱敏

不新增第三套 HR 同步中心。Integration 页面负责 Task、技术 Batch、Execution、Attempt；Organization 页面负责领域对象和业务 Batch/Record。二者使用独立菜单按钮、Casbin 和现有 Data Permission，不硬编码角色。

无 query/detail 权限时前端不得预取数据，直达路由稳定拒绝；后端始终是最终权限边界。DTO、日志、Audit、Record 禁止人员联系方式、证件、原始 Source ID、`sendpost`、Response Body、Credential、Authorization、Cookie、Token 和底层错误原文。

## 17. V1 明确不支持

当前冻结不包含：

- 主任职同步；
- 生产兼职同步；
- 自动再入职和雇佣段；
- 物理删除；
- 动态公司 Fan-out；
- 全量对账；
- 视图 99；
- OA 专用人员源；
- Response Artifact；
- HR 业务脚本。

上述能力不得通过直接写 Organization/Integration 数据、默认源语义或临时开关实现。

## 18. 冻结后的变更控制

后续代码变更不得：

1. 破坏 Source DTO -> Canonical Input -> Domain 边界；
2. 绕过 SourceKey 或以名称/联系方式回退；
3. 让业务失败进入 Integration Retry；
4. 自行推进 Checkpoint 或 Execution；
5. 把 Parser candidate 直接持久化为 Assignment；
6. 返回原始 Payload、Source ID 或错误正文；
7. 在外部 Gate 未关闭时启用生产 Consumer。

源负责人提交权威契约后，应更新 Gate 矩阵、形成 SourceContract 变更评审，并针对受影响 Consumer 重新运行 PostgreSQL 16 + TLS + Runner 小流量准入验收。

## 19. 最终冻结结论

**Organization HR Adapter V1 Capability：通过冻结。**

**Current HR Source Production Enablement：不允许。**

当前允许进入生产准入准备、源契约确认和容量试验，不允许直接启用七类真实 HR SyncTask。能力冻结不等于源契约背书，也不关闭任何仍为 unconfirmed 的 Gate。
