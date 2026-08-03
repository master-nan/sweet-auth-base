# 数据权限验收指南

## 1. 文档目的

本文档用于在开发环境准备并验收 Sweet Platform 新数据权限链路：

```text
用户 -> 角色 -> 授权 -> 策略 -> 归属定义 -> 组织 Provider
     -> Resolver -> Metadata Adapter -> 低代码查询结果
```

验收数据不属于正式初始化数据，不得在生产环境执行，也不得复制到生产
Seed 或 Migration。

## 2. 当前验收状态

验收数据、组织层级、账号绑定、低代码实体、数据权限配置和安全回归测试
已经具备。当前完整业务结果验收存在一个明确阻塞：配置层允许
`self_and_descendants`，但 Resolver 运行时目前只接受 `exact`，Dimension
Provider 也只返回员工当前有效任职中的直接组织。

因此当前版本会对“本组织及下级组织”策略返回稳定配置冲突并拒绝访问，
不会错误放大为全部数据。待后续单独实现组织下级范围展开后，可直接复用
本指南和同一份数据脚本，无需改造验收数据。

禁止通过给用户增加下级组织任职、把关系改成 `exact`，或者把上海运输部
直接写进允许值等方式伪造验收通过。

## 3. 验收数据

### 3.1 组织与人员

| 对象 | 验收值 | 说明 |
| --- | --- | --- |
| 管理架构 | 数据权限验收管理架构 | 稳定编码 `DP-ACCEPTANCE-MGMT` |
| 管理组织 | 华东物流中心 | 华东根组织 |
| 管理组织 | 上海运输部 | 华东物流中心的下级组织 |
| 管理组织 | 华南物流中心 | 华南根组织 |
| 用户 A | `dp_acceptance_east` | 绑定验收员工 A，当前任职华东物流中心 |
| 用户 B | `dp_acceptance_south` | 绑定验收员工 B，当前任职华南物流中心 |
| 无授权用户 | `dp_acceptance_ungranted` | 有有效角色和华东任职，但没有 Data Grant |
| 授权角色 | 物流经理 | 用户 A、用户 B 使用 |

脚本默认将三个账号密码设置为 `DpDemo@2026`。执行时可使用参数覆盖。

### 3.2 数据权限配置

| 配置 | 验收值 |
| --- | --- |
| Resource | `transport_order` |
| Operation | `query`、`detail` |
| Ownership | `owner_org` |
| Dimension | `management_org` |
| Binding | `metadata_field` |
| Policy | 本组织及下级组织 |
| Scope source | `effective_org_units` |
| Relation | `self_and_descendants` |
| Operator | `in` |
| Grant | 物流经理 -> `transport_order` -> 本组织及下级组织 |

### 3.3 低代码测试实体

低代码实体编码为 `demo_transport_order`，包含运单号、所属管理组织和金额。

| 运单 | 所属管理组织 | 预期用户 |
| --- | --- | --- |
| ORD001 | 华东物流中心 | 用户 A |
| ORD002 | 上海运输部 | 用户 A（下级组织） |
| ORD003 | 华南物流中心 | 用户 B |

## 4. 准备数据

先确认后端 Migration 和基础 Seed 已执行，再从项目根目录运行：

```bash
psql "$DATABASE_URL" \
  -v environment=development \
  -v app_salt='当前开发环境配置中的应用盐值' \
  -v demo_password='DpDemo@2026' \
  -f scripts/DataPermissionDemoData.sql
```

安全要求：

1. 必须显式传入 `environment=development`，否则脚本拒绝执行。
2. `app_salt` 只作为本次数据库会话参数使用，不会写入验收表。
3. 脚本在一个事务内执行，任一校验失败都会整体回滚。
4. 重复执行会按稳定编码更新原验收对象，不重复创建。
5. 脚本不会注册到 Migration 或生产 Seed。

脚本结束前会自动检查组织父子关系、账号绑定、角色授权、Ownership 与
PolicyRule 一致性，以及华东、华南两组业务数据的静态预期集合。

## 5. 完整链路验收

### 5.1 目标结果

在组织下级范围运行时能力补齐后，分别以三个账号执行同一张低代码表的
列表、分页总数和详情查询。

| 登录账号 | rows | total | detail |
| --- | --- | --- | --- |
| `dp_acceptance_east` | ORD001、ORD002 | 2 | 可查看 ORD001、ORD002；不可查看 ORD003 |
| `dp_acceptance_south` | ORD003 | 1 | 可查看 ORD003；不可查看 ORD001、ORD002 |
| `dp_acceptance_ungranted` | 空 | 0 | 所有运单均不可查看 |

列表 rows 和 total 必须使用同一权限结果。无权详情不得泄露记录是否存在。

### 5.2 当前版本预期

当前版本执行用户 A 或用户 B 的 `query`、`detail` 时，Resolver 应返回数据
权限配置冲突，查询不得退回未过滤全量结果。自动化测试
`TestDataPermissionDemoAcceptanceDescendantPolicyFailsClosed` 固定了这一安全
行为。

这表示安全验收通过，但“用户 A 可见 ORD001、ORD002”的业务验收尚未通过。

## 6. 安全验收

执行后端测试：

```bash
cd backend
go test ./...
```

重点确认：

1. 无 Grant 时 Resolver 返回 `none`，rows 为空、total 为 0。
2. `self_and_descendants` 尚未支持时返回稳定错误，不得返回 `all`。
3. Resolver 依赖失败时不得执行原始全量查询。
4. Metadata Adapter 字段缺失、停用、类型漂移或绑定不匹配时整体失败。
5. 详情查询必须在数据库查询中同时应用业务 ID 和权限条件。
6. rows、total 和 detail 不得使用不同的权限解释。

## 7. 后续准入条件

完整业务验收前必须通过独立任务补齐以下能力：

1. Resolver 解释 `self_and_descendants` 和显式 `structure_code`。
2. 通过 Organization Permission Provider 获取指定架构、指定日期的下级组织。
3. 保持 Organization Provider 为唯一组织事实来源，Data Permission 不直接访问组织表。
4. 下级展开失败、循环、孤儿、超限或架构不存在时安全收敛，绝不返回全部数据。
5. 补齐后重新执行本脚本，并验证本指南第 5.1 节全部结果。
