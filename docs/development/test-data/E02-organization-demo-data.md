# E02 Organization Foundation 开发验收测试数据

## 1. 文档目的

本文档说明 Organization Foundation 开发验收数据的范围、装载方式和验证场景。

配套脚本：

```text
scripts/E02-organization-demo-data.sql
```

该脚本仅用于开发环境，不属于 Migration、正式 Seed 或生产初始化数据。

## 2. 安全边界

必须遵守：

1. 只能在已完成 Organization Migration 的开发数据库执行。
2. 执行时必须显式传入 `environment=development`。
3. 脚本没有注册到 `migrationSteps()` 或 `platformSeedSteps()`。
4. 脚本不修改 `sys_table`、`sys_table_field`、字典、菜单、按钮或 Casbin。
5. 脚本不创建数据库表，不执行 DDL。
6. 脚本不创建登录账号、角色或权限。
7. 脚本只把现有开发账号 `admin` 显式绑定到测试员工“张伟”。
8. 如果 `admin` 已绑定其他非测试员工，脚本会整体失败并回滚，不会抢占绑定。
9. 所有写入位于单个事务内；任一步失败时不保留部分数据。
10. 不得将该文件加入正式 Seed 注册链路。

未传开发环境确认时，脚本会返回非零状态：

```bash
docker compose exec -T postgres \
  psql -X -U sweet_admin -d sweet_admin \
  < scripts/E02-organization-demo-data.sql
```

## 3. 执行方式

前置条件：

1. PostgreSQL 开发容器已启动。
2. 当前代码对应的 Migration 已在该开发库执行。
3. `org_*` 表已经存在。
4. 正式基础 Seed 已创建 `admin` 账号。

执行：

```bash
docker compose exec -T postgres \
  psql -X -U sweet_admin -d sweet_admin \
  -v environment=development \
  < scripts/E02-organization-demo-data.sql
```

验证幂等时连续执行两次相同命令。两次都应输出：

```text
Organization demo data ready:
legal_entities=7, structures=2, units=9, nodes=13,
positions=8, employees=5, assignments=8
```

## 4. 幂等策略

统一测试源：

```text
source_system_code = sweet_dev_org_acceptance_v1
```

稳定定位规则：

| 对象 | 稳定业务键 |
| --- | --- |
| 法人、组织、岗位、人员、任职、架构节点 | `source_system_code + source_id` |
| 管理架构 | `code`，同时保持稳定 `source_system_code + source_id` |
| 同步批次 | `batch_no` |
| 同步记录 | `batch_id + object_type + source_id + action` |
| 账号绑定 | `org_employee.user_id`，账号按稳定 `sys_user.user_name=admin` 定位 |

重复执行时：

1. 不创建重复法人、组织、节点、岗位、人员、任职或同步记录。
2. 不依赖固定自增 ID。
3. 源镜像字段按测试定义更新。
4. `local_note`、`local_tags` 等平台扩展字段只在首次插入时写入，重复执行不覆盖。
5. `user_id` 是本测试明确负责的扩展字段，只对“张伟”执行显式绑定。
6. 相对日期会随执行日刷新，以持续保证“当前、历史、未来”三种任职均可验证。

## 5. 数据规模

| 对象 | 数量 | 主要覆盖 |
| --- | ---: | --- |
| 法人主体 | 7 | 两个根、集团、法人公司、分公司、核算主体 |
| 管理架构 | 2 | 行政管理架构、区域经营架构 |
| 管理组织 | 9 | 总部、事业部、区域、中心、部门、项目组 |
| 架构节点 | 13 | 多根、三级层级、跨架构复用组织 |
| 岗位 | 8 | 管理、专业、运营、有效与历史岗位 |
| 人员 | 5 | 在职、离职历史、未来入职、账号绑定 |
| 任职 | 8 | 一人多岗、当前、历史、未来、跨法人 |
| 同步批次 | 2 | 成功批次、失败批次 |
| 同步记录 | 3 | 成功、无变化、依赖错误 |

脚本末尾包含数据库自检；数量或关键关系不符合预期时事务会回滚。

## 6. 法人测试数据

```text
验收集团（DEV-LE-GROUP）
├─ 验收科技有限公司（DEV-LE-CN）
│  ├─ 验收科技上海分公司（DEV-LE-SH-BRANCH）
│  └─ 验收科技内部核算中心（DEV-LE-ACCOUNTING）
└─ 验收物流有限公司（DEV-LE-LOGISTICS）

验收海外控股（DEV-LE-OVERSEAS）
└─ 验收新加坡公司（DEV-LE-SG）
```

验证点：

1. 法人层级只使用 `org_legal_entity.parent_id`。
2. 法人树存在两个根节点。
3. 包含 `group`、`legal_company`、`branch`、`accounting_unit`。
4. 法人选择值始终为内部 `legal_entity_id`。

## 7. 管理架构测试数据

### 7.1 行政管理架构

```text
集团总部
├─ 华东事业部
│  └─ 上海销售部
├─ 华南事业部
│  ├─ 深圳运营部
│  └─ 重点项目组
└─ 共享服务中心
```

### 7.2 区域经营架构

```text
华东区域
├─ 上海销售部
└─ 共享服务中心

华南区域
├─ 深圳运营部
└─ 重点项目组
```

验证点：

1. 两棵树使用不同 `org_structure.id`。
2. 父子关系使用 `org_structure_node.parent_node_id`。
3. “共享服务中心”在两套架构中拥有不同 `structure_node_id`。
4. 两个节点引用同一个内部 `org_unit_id`。
5. 业务选择值必须为 `org_unit_id`，不能保存 `structure_node_id`。
6. 行政架构包含三级组织层级。

稳定架构编码：

| 编码 | 名称 |
| --- | --- |
| `DEV-STRUCT-ADMIN` | 行政管理架构 |
| `DEV-STRUCT-REGION` | 区域经营架构 |

## 8. 岗位与人员测试数据

### 8.1 岗位

| 稳定 source_id | 岗位名称 | 组织 | 状态 |
| --- | --- | --- | --- |
| `DEV-POS-HQ-DIRECTOR` | 总部负责人 | 集团总部 | 当前有效 |
| `DEV-POS-EAST-MANAGER` | 华东事业部负责人 | 华东事业部 | 当前有效 |
| `DEV-POS-SALES-MANAGER` | 销售经理 | 上海销售部 | 当前有效 |
| `DEV-POS-SALES-SPECIALIST` | 销售专员 | 上海销售部 | 当前有效 |
| `DEV-POS-SHARED-FINANCE` | 共享财务专员 | 共享服务中心 | 当前有效 |
| `DEV-POS-OPS` | 运营专员 | 深圳运营部 | 当前有效 |
| `DEV-POS-PROJECT` | 项目协调员 | 重点项目组 | 当前有效 |
| `DEV-POS-LEGACY-SALES` | 历史销售岗位 | 上海销售部 | 已停用、历史有效期 |

### 8.2 人员

| 员工号 | 姓名 | 状态 | 账号绑定 | 用途 |
| --- | --- | --- | --- | --- |
| `DEV-E0001` | 张伟 | 在职 | `admin` | 一人多岗、历史、未来、账号绑定 |
| `DEV-E0002` | 李娜 | 在职 | 无 | 普通当前任职 |
| `DEV-E0003` | 王强 | 在职 | 无 | 物流法人和华南组织 |
| `DEV-E0098` | 陈历史 | 离职 | 无 | 历史人员回显 |
| `DEV-E0099` | 赵未来 | 试用 | 无 | 未来人员和未来任职 |

所有手机号和邮箱均为测试值；API 仍应按现有 DTO 规则脱敏。

## 9. 任职测试数据

张伟拥有四条时间范围不同的任职：

| 任职 | 时间范围 | 组织 | 岗位 | 预期分类 |
| --- | --- | --- | --- | --- |
| `DEV-ASG-ZHANG-HISTORY` | 一年前结束 | 华东事业部 | 历史销售岗位 | history |
| `DEV-ASG-ZHANG-CURRENT-PRIMARY` | 一年前开始、无结束日 | 上海销售部 | 销售经理 | current |
| `DEV-ASG-ZHANG-CURRENT-PART` | 90 天前开始、无结束日 | 共享服务中心 | 共享财务专员 | current |
| `DEV-ASG-ZHANG-FUTURE` | 30 天后开始 | 重点项目组 | 项目协调员 | future |

验证点：

1. 当前任职返回 2 条，不能自动只取第一条。
2. 历史任职返回 1 条。
3. 未来任职返回 1 条。
4. 时间轴返回 4 条并按现有服务规则排序。
5. 当前组织摘要包含上海销售部和共享服务中心。
6. 当前岗位摘要包含销售经理和共享财务专员。
7. 未来任职跨到“验收物流有限公司”，用于验证跨法人任职数据结构。
8. 测试数据不定义新的主任职业务规则。

## 10. 内部 ID 查询

所有 API 和选择器都使用数据库内部 ID。测试前可通过稳定业务键查询：

```bash
docker compose exec -T postgres \
  psql -X -U sweet_admin -d sweet_admin -At -F $'\t' -c "
SELECT 'legal_entity', source_id, id
FROM org_legal_entity
WHERE source_system_code = 'sweet_dev_org_acceptance_v1'
UNION ALL
SELECT 'structure', source_id, id
FROM org_structure
WHERE source_system_code = 'sweet_dev_org_acceptance_v1'
UNION ALL
SELECT 'org_unit', source_id, id
FROM org_unit
WHERE source_system_code = 'sweet_dev_org_acceptance_v1'
UNION ALL
SELECT 'employee', source_id, id
FROM org_employee
WHERE source_system_code = 'sweet_dev_org_acceptance_v1'
UNION ALL
SELECT 'position', source_id, id
FROM org_position
WHERE source_system_code = 'sweet_dev_org_acceptance_v1'
ORDER BY 1, 2;"
```

账号绑定检查：

```sql
SELECT e.employee_no, e.name, u.id AS user_id, u.user_name
FROM org_employee e
LEFT JOIN sys_user u ON u.id = e.user_id
WHERE e.source_system_code = 'sweet_dev_org_acceptance_v1'
ORDER BY e.employee_no;
```

预期只有张伟绑定 `admin`。

## 11. API 验证准备

以下示例假定：

```bash
export BASE_URL=http://localhost:8008/sweet_admin
export TOKEN=<通过正常登录获得的开发令牌>
```

通用空查询结构：

```json
{
  "page": 1,
  "num": 20,
  "expressions": [
    {
      "logic": 1,
      "rules": [
        {
          "field": "",
          "value": null
        }
      ],
      "nested": []
    }
  ],
  "quick_query": {
    "keyword": ""
  }
}
```

API 验证应使用正常登录和既有菜单、按钮、Casbin 权限，不得绕过功能权限。

## 12. 法人查询验证

列表：

```bash
curl -sS "$BASE_URL/admin/org/legal-entity/query" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "page": 1,
    "num": 20,
    "expressions": [],
    "quick_query": {"keyword": "DEV-LE"}
  }'
```

树：

```bash
curl -sS "$BASE_URL/admin/org/legal-entity/tree" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"only_effective": true}'
```

验收：

1. 列表总数为 7。
2. 树返回两个根。
3. 上海分公司和内部核算中心均位于验收科技有限公司下。
4. Response 不包含 `source_id`、`source_version`。

## 13. 管理组织树验证

先查询 `DEV-STRUCT-ADMIN` 和 `DEV-STRUCT-REGION` 的内部 ID，然后分别调用：

```bash
curl -sS "$BASE_URL/admin/org/unit/tree" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "structure_id": <structure_id>,
    "only_effective": true
  }'
```

验收：

1. 行政架构根节点为集团总部。
2. 区域架构有华东区域和华南区域两个根。
3. 两棵树中的共享服务中心具有不同 `structure_node_id`。
4. 两个共享服务中心节点具有相同 `org_unit_id`。
5. 返回节点没有因相同 `org_unit_id` 被错误去重。

可用 SQL 直接核对：

```sql
SELECT
    s.code AS structure_code,
    n.id AS structure_node_id,
    n.org_unit_id,
    u.code AS org_unit_code
FROM org_structure_node n
JOIN org_structure s ON s.id = n.structure_id
JOIN org_unit u ON u.id = n.org_unit_id
WHERE u.source_system_code = 'sweet_dev_org_acceptance_v1'
  AND u.source_id = 'DEV-OU-SHARED'
ORDER BY s.code;
```

## 14. 人员查询验证

```bash
curl -sS "$BASE_URL/admin/org/employee/query" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "page": 1,
    "num": 20,
    "expressions": [],
    "quick_query": {"keyword": "DEV-E0001"}
  }'
```

验收：

1. 返回张伟且只返回一次。
2. `bound_status=bound` 能找到张伟。
3. 手机号和邮箱按 DTO 规则脱敏。
4. 不返回密码、角色、权限、`source_id` 或同步内部字段。
5. 按上海销售部或共享服务中心筛选均能命中张伟。
6. 按销售经理或共享财务专员筛选均能命中张伟。

## 15. 任职查询验证

先通过稳定员工号查询张伟的内部 `employee_id`，然后依次调用：

```bash
curl -sS "$BASE_URL/admin/org/assignment/query" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "page": 1,
    "num": 20,
    "employee_id": <employee_id>,
    "time_scope": "current"
  }'
```

将 `time_scope` 分别替换为：

- `current`：预期 2 条。
- `history`：预期 1 条。
- `future`：预期 1 条。
- `timeline`：预期 4 条。

当前任职摘要：

```bash
curl -sS \
  "$BASE_URL/admin/org/employee/<employee_id>/assignments/summary" \
  -H "Authorization: Bearer $TOKEN"
```

摘要应同时包含两项当前组织和两项当前岗位，不能自动取第一条任职。

## 16. Selector 验证

四类 Options API：

| Selector | API | keyword |
| --- | --- | --- |
| LegalEntitySelect | `POST /admin/org/legal-entity/options` | `DEV-LE` |
| OrgUnitSelect | `POST /admin/org/unit/options` | `DEV-OU` |
| EmployeeSelect | `POST /admin/org/employee/options` | `DEV-E` |
| PositionSelect | `POST /admin/org/position/options` | `DEV-POS` |

统一请求示例：

```json
{
  "page": 1,
  "num": 50,
  "keyword": "DEV-E",
  "only_effective": true
}
```

验收：

1. `value` 为数字内部 ID。
2. `label` 为编码和名称。
3. 不返回名称、`source_id` 或 `user_id` 作为 value。
4. 张伟的 Employee option value 是 `employee_id`，不是绑定的 `admin user_id`。
5. Options 有分页或数量限制。

历史回显示例：

1. 查出陈历史的内部 `employee_id`。
2. 调用 Employee Options 时传入：

```json
{
  "page": 1,
  "num": 50,
  "only_effective": true,
  "include_history": true,
  "selected_ids": [<history_employee_id>]
}
```

3. 陈历史应返回且 `disabled=true`，可以回显但不能作为新选择项。
4. `DEV-POS-LEGACY-SALES` 用相同方式验证历史岗位回显。

## 17. DynamicFormDialog 验证

本脚本不修改低代码元数据，也不创建测试页面。请使用现有开发专用低代码表字段验证。

字段 metadata 通过既有 selector resolver 表达，例如：

```json
{
  "selector": {
    "selector_type": "employee",
    "include_history": true
  }
}
```

该 JSON 放入开发字段现有的 `linkage_config`；其他三类分别使用：

- `legal_entity`
- `org_unit`
- `position`

验收步骤：

1. 打开使用 `DynamicFormDialog` 的开发表单。
2. Employee 字段搜索 `DEV-E0001` 或“张伟”。
3. 选择张伟并提交。
4. 检查提交请求，字段值必须是数字 `employee_id`。
5. 请求中不得保存 label、姓名、`source_id` 或 `user_id`。
6. 编辑已保存的陈历史 ID 时应回显 disabled 历史选项。
7. 普通文本、数字、日期、字典和关联字段行为保持不变。

## 18. AdvancedQuery 验证

使用与 DynamicFormDialog 相同的 selector metadata，不创建第二套配置。

验收步骤：

1. 打开带 `employee` selector metadata 字段的 AdvancedQuery。
2. 操作符只应出现 `=` 和 `in`。
3. `=` 选择张伟后，查询表达式 value 为单个数字 `employee_id`。
4. `in` 同时选择张伟和李娜后，value 为数字 ID 数组。
5. 请求中不得保存 label、姓名或 `user_id`。
6. 用 `org_unit`、`legal_entity`、`position` 重复验证。
7. 普通字段仍使用原文本、数字、日期、字典或关联查询控件。
8. 本场景不验证组织下级展开、descendant 或数据权限。

查询请求中 selector 规则的预期形态：

```json
{
  "field": "employee_id",
  "expression_type": 9,
  "value": [<zhang_employee_id>, <li_employee_id>],
  "type": 1
}
```

其中 `expression_type=9` 是当前平台的 `in`，`type=1` 是 BIGINT；验收时应以项目枚举为准，
不得在业务页面另造枚举。

## 19. 同步结果附加数据

脚本额外提供：

| batch_no | 状态 | 用途 |
| --- | --- | --- |
| `DEV-ORG-BATCH-SUCCESS` | success | 成功与 no_change 业务结果 |
| `DEV-ORG-BATCH-FAILED` | failed | 模拟 `org_position_missing` 依赖错误 |

这些记录只表示组织业务同步结果：

1. `execution_id` 为空。
2. 不包含 HTTP request/response payload。
3. 不模拟 IntegrationExecution。
4. 不重复建设通用接口日志或重试机制。

## 20. 脚本验证记录

验证环境：

- PostgreSQL 16。
- 使用当前 `model.Org*` 定义在临时独立 schema 建表。
- 临时 schema 只用于脚本验证，完成后已删除。

验证结果：

1. 未传 `environment=development` 时脚本返回非零状态且无数据写入。
2. 第一次执行成功。
3. 第二次连续执行成功。
4. 第二次执行后对象数量未增长。
5. 共享服务中心仍对应两个架构节点和一个组织 ID。
6. 张伟保持 2 条当前任职、1 条历史任职、1 条未来任职。
7. 张伟与 `admin` 的一对一绑定成立。
8. 当前长期运行的开发容器尚未包含 `org_*` 表，因此没有向该旧运行库装载数据；
   应在当前 Migration 完成后按本文命令执行。

## 21. 本次未做

1. 未修改 Migration。
2. 未修改正式 Seed。
3. 未修改生产初始化数据。
4. 未修改 Organization Model、Repository、Service、Controller 或 Router。
5. 未修改前端组件或页面。
6. 未创建新菜单、按钮、角色或 Casbin policy。
7. 未实现真实组织源同步。
8. 未实现数据权限。
