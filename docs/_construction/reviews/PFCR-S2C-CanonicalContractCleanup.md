# PFCR-S2C Canonical Contract Cleanup

## 1. Current Contract

- 审计基线：`9c2f6a814dbc7e550bca981eba1e4327394ed5d5`
- 审计开始时工作区：干净
- 当前 Backend / Frontend `SysTableFieldType` 一致，但仍使用历史过渡编号：
  `BigInt=1, Varchar=3, Text=4, Boolean=5, Date=6, Datetime=7, Time=8, Json=10, Int=11, SmallInt=12, Decimal=13`。
- 当前 PostgreSQL CHECK 允许上述 11 个编号，并拒绝 `2/9`。
- 当前 `sys_table_field_type` Seed 有 11 项，但 SmallInt / Decimal 的值仍为 `12/13`；已有 Seed 逻辑不会更新同 `item_code` 的历史值。
- Frontend `SysTableFieldInputType` 的年月选择符号误拼为 `YREA_MONTH_PICKER`，其数值仍是稳定的 `9`。
- Organization selector 的 `_select` 形式只存在于 Frontend compatibility map 和测试；当前 Backend Seed、Runtime DTO、菜单与 Metadata Seed 没有真实使用证据。

## 2. Historical Contract

存在两类可升级数据库：

1. 尚未运行 PFCR-002C：`field_type=2` 表示历史 Float，`field_type=9` 表示历史 TinyInt。
2. 已运行 PFCR-002C：`field_type=13` 表示 Decimal，`field_type=12` 表示 SmallInt。

历史迁移理解仅保留在 `backend/migrate`。生产 Runtime 不再识别 12/13、Float、TinyInt、旧 selector alias 或源码拼写 alias。

## 3. Canonical Contract

| 编号 | Backend | Frontend | PostgreSQL |
| --- | --- | --- | --- |
| 1 | `BigIntFieldType` | `BIGINT` | `bigint` |
| 2 | `DecimalFieldType` | `DECIMAL` | `numeric(p,s)` |
| 3 | `VarcharFieldType` | `VARCHAR` | `varchar` |
| 4 | `TextFieldType` | `TEXT` | `text` |
| 5 | `BooleanFieldType` | `BOOLEAN` | `boolean` |
| 6 | `DateFieldType` | `DATE` | `date` |
| 7 | `DatetimeFieldType` | `DATETIME` | `timestamp` |
| 8 | `TimeFieldType` | `TIME` | `time` |
| 9 | `SmallIntFieldType` | `SMALLINT` | `smallint` |
| 10 | `JsonFieldType` | `JSON` | `jsonb` |
| 11 | `IntFieldType` | `INT` | `integer` |

Runtime、DTO、Swagger、Frontend UI 和字典只公开以上 11 项。业务代码只使用 enum symbol，不直接使用 FieldType magic number。

## 4. Numeric Enum Audit Matrix

| Enum | 当前状态 | 结论 | 依据 |
| --- | --- | --- | --- |
| `SysTableFieldType` | 1,3,4,5,6,7,8,10,11,12,13 | `RENUMBER` | 2/9 是已删除类型空洞，12/13 是过渡 replacement |
| `DataPermissionsEnum` | 1..3 | `REMOVE_LEGACY` | 无生产、测试、Migration、DTO、Model 或 Frontend 消费者；正式数据权限已使用领域字符串合同 |
| `SysTableType` | 1..2 | `KEEP` | DB、Metadata、Report 和 Frontend 的稳定协议 |
| `SysTableFieldInputType` | 1..16 | `KEEP` | 编号连续且被 Metadata / Form 使用；只改 Frontend 符号拼写 |
| `ExpressionType` | 1..14 | `KEEP` | 已进入 Query、AdvancedQuery 和 Query Scheme 持久化协议 |
| `ExpressionLogic` | 1..2 | `KEEP` | 已进入嵌套查询和 Query Scheme 协议 |
| `SysTableRelationType` | 1..4 | `KEEP` | Relation Metadata 稳定协议 |
| `SysMenuButtonPosition` | 1..7 | `KEEP` | MenuButton、Form、Detail 和权限稳定协议 |
| `SmsStatus` | 1..3 | `KEEP` | 数据库存储稳定；HTTP 输出使用字符串 |
| HR `SourceEnableStatus` | 0..2 | `KEEP` | Source Adapter 内部受控状态，不是平台公共枚举 |
| Frontend `removeType` | 0..2 | `KEEP` | TagView store 内部控制流，无历史 replacement |

除 `SysTableFieldType` 外，不为追求数字连续重排稳定协议。

## 5. String Alias Audit Matrix

| 关键词/对象 | 分类 | 结论 |
| --- | --- | --- |
| `legal_entity_select/org_unit_select/employee_select/position_select` | Runtime 历史兼容 | Migration 规范化 Metadata 后删除 Frontend alias |
| `YREA_MONTH_PICKER` | 源码拼写错误 | 改为 `YEAR_MONTH_PICKER`，数值保持 9，不保留 alias |
| Integration `SyncExecutionInputPlanVersion` | 无消费者的 V1 源码 alias | 删除；正式调用继续显式使用 V1/V2 |
| Low-code 发布时 `system_*` 按钮清理 | Runtime 历史清理 | 移入幂等 Migration，删除 Publication Service / Repository 兼容入口 |
| 旧无 JTI/SID Token 与旧 Redis raw-token key | Runtime 历史兼容 | Breaking baseline 后拒绝旧 Token，删除双 key 读取和秒级 revoke timestamp 解释 |
| Generalization 菜单按钮全量 API fallback | 权限 UI fail-open fallback | 删除；用户菜单读取失败时不再回退管理侧全量按钮 |
| Report `legacy_sheet` 与 Report compatibility | Report 延期桥接 | `REPORT_DEFERRED`，本 Task 不改产品语义 |
| `fallback`、SQL alias、operator incompatible | 正常业务语义 | 保留，不属于历史 contract alias |

## 6. Migration Strategy

`metadata_value_contract` 保持唯一幂等升级边界：

1. PostgreSQL 先删除旧 FieldType / Decimal / SmallInt CHECK，避免旧 CHECK 阻止 `13 -> 2`、`12 -> 9`。
2. 对 `field_type IN (2,13)` 规范 Decimal precision/scale；保留合法 `scale=0`，只在缺失时读取历史 length 字段或使用默认值。
3. 执行 `13 -> 2`、`12 -> 9`；旧库中的 2/9 原位成为 Canonical Decimal/SmallInt。
4. 更新 Decimal / SmallInt 字典项为 2/9，删除 Float / TinyInt 项。
5. 规范化可解析 Metadata 中的 Organization selector alias。
6. 在菜单 page binding 完成后，一次性删除 Low-code 菜单的历史 `system_*` 按钮及角色按钮关系。
7. 重建 PostgreSQL CHECK，仅允许 1..11，并让 Decimal / SmallInt 约束绑定 2/9。
8. 重复执行不得改变已规范数据。

Migration 测试覆盖未运行 PFCR-002C、已运行 PFCR-002C、混合状态、软删除行、重复执行、CHECK 拒绝 12/13、Decimal precision/scale 与 SmallInt 范围。

## 7. Removed Compatibility

已删除：

- Runtime `SmallInt=12` / `Decimal=13`；
- `DataPermissionsEnum` 孤立声明；
- Frontend `YREA_MONTH_PICKER` 拼写；
- Frontend Organization selector `_select` alias；
- Integration 无消费者的 V1 plan version alias；
- Low-code Publication Service 的历史按钮清理分支；
- 旧无 JTI/SID Token、raw-token Redis key 和秒级 revoke timestamp 解释；
- Generalization 权限按钮的管理 API fallback；
- Swagger / UI 中 12/13 的合法类型声明；
- Migration 之外对历史 FieldType 编号的解释。

## 8. Remaining Legitimate Compatibility

- Migration 可以识别历史 2/9/12/13 和 selector alias，这是数据库升级职责。
- Report 的 `legacy_sheet`、旧 Sheet 运行桥接继续 `REPORT_DEFERRED`。
- 正常 fallback、SQL alias、兼容性校验均为当前业务语义，不属于历史枚举兼容。

## 9. QueryModel Conclusion

生产调用方共三处直接转换：

- `GeneralizationService.ResolveRuntimeTable`；
- `ReportService.GetDataSources`；
- `ReportService.ResolveRuntimeTable`。

Generalization Repository、Query Builder、Data Permission 和 Controller 当前共同消费 `model.SysTable`。仅移除 Generalization 的桥接会扩大为动态查询引擎契约重写，并让 Report 仍保留同一转换，因此本 Task 保留唯一 `TableMetadata.QueryModel()` adapter。禁止新增调用方；后续应在 Report 产品边界稳定后，由动态查询引擎直接消费 Runtime Metadata，再一次性删除 adapter。该约束同步进入 `PlatformEngineeringGuide` 的延期边界。

## 10. Tests

专项：

- Backend canonical FieldType 1..11 唯一连续；
- Frontend FieldType 与 canonical 编号一致；
- Frontend/Backend enum contract guard；
- old 2/9、previous 12/13、混合数据库和幂等 Migration；
- PostgreSQL CHECK 接受 2/9、拒绝 12/13；
- Decimal precision/scale、SmallInt 范围和 Generalization round-trip；
- Runtime Metadata、Dynamic Form、Advanced Query、Query Scheme；
- Organization selector canonical 解析。

最终门禁：`go test ./... -count=1`、全量 Race、PostgreSQL 16 强制门禁、Frontend test/lint/typecheck/build、`make release-check`、`make docs-check`。

## 实际修改结果

### Canonical Contract

- Backend、Frontend、Swagger 与 `sys_table_field_type` Seed 已统一为连续的 `1..11`。
- Decimal 固定为 `2`，SmallInt 固定为 `9`；生产 Runtime 不再声明或接受 `12/13`。
- PostgreSQL Migration 支持旧库 `2/9` 和上一过渡基线 `12/13`，重复执行幂等；最终 CHECK 拒绝 `12/13`。
- Decimal precision/scale 迁移保留合法 `scale=0`，Generalization 读写继续以字符串保持 Numeric 精度。
- FieldType 字典最终只有 11 个 Canonical item；旧 Float/TinyInt 和过渡值均由 Migration 清理或更新。

### Runtime Compatibility Cleanup

- Frontend `YREA_MONTH_PICKER` 已直接改名为 `YEAR_MONTH_PICKER`，数值不变且没有旧 alias。
- Organization selector 的四个 `_select` alias 已从 Runtime 删除；Migration 规范化 Data Permission selector 和嵌套 linkage JSON。
- 无消费者的 `DataPermissionsEnum` 与 Integration V1 源码 alias 已删除，稳定 Numeric Enum 均保持原编号。
- Low-code `system_*` 历史按钮清理由 Publication Runtime 移到幂等 Migration，删除对应 Repository public method。
- 历史无 JTI/SID Token、raw-token Redis key 和旧用户撤销时间格式不再由 Runtime 兼容；当前签发 Token 始终使用 JTI/SID 与 Session 真值。
- Generalization 用户按钮读取失败时保持 fail-closed，不再回退管理侧全量按钮 API。

### Retained Boundaries

- `TableMetadata.QueryModel()` 仍是唯一动态查询技术桥，生产直接调用保持三处：Generalization 一处、Report 两处。Report 继续 `REPORT_DEFERRED`，未新增 adapter。
- Integration input plan 的 V1/V2 是已持久化 Schema Version，不是别名，继续保留。
- DB preflight 和 Migration 对历史数据库状态的识别继续保留；普通业务 fallback、SQL alias 与 compatibility validation 也不属于历史 Runtime contract。

### Test Results

- `go test ./... -count=1`：通过。
- `SWEET_REQUIRE_POSTGRES_TESTS=true ... go test ./... -count=1`：PostgreSQL 16 强制全量通过。
- `go test -race ./... -count=1`：通过。
- `yarn test`：72 个测试文件、232 个测试通过（含 Backend/Frontend FieldType 跨端一致性 Guard）。
- `yarn lint`、`yarn typecheck`、`yarn build`：通过；build 仅保留既有大 chunk warning。
- `make release-check`：通过，包含 docs、PostgreSQL 16 全量、全量 race、Frontend test/lint/typecheck/build。
- `make docs-check`：通过，检查 66 个 Markdown 文件。

### Behavior Impact

本 Task 不新增业务功能、不改变 API URL、Request/Response JSON、权限模型或 Query 协议。Breaking 影响限定为已批准的 Canonical Cleanup：旧 FieldType `12/13` 写入被拒绝、旧 selector/source alias 不再被 Runtime 接受、旧无 JTI/SID Token 失效。数据库历史状态通过 Migration 一次性升级。

### Remaining Debt

- 历史 `field_type=2` 对应的物理 `real/double precision` 列不会被静默改列；DB preflight 会报告 Metadata/physical mismatch，需要显式 DDL Migration，避免在本 Task 隐式改业务数据类型。
- `QueryModel()` 技术桥与 Report `legacy_sheet` 是剩余明确兼容边界，已进入工程指南 backlog / Report Deferred，禁止扩散新调用方。
