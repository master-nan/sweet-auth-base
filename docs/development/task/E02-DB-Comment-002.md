# E02-DB-Comment-002 Organization Database Comment Correction

## 1. 调整原因

E02-DB-Comment-001 已建立独立 PostgreSQL COMMENT Migration，但首版注释混入了业务边界、权限、同步覆盖和生命周期规则，并由组织模块重新解释了 `model.Basic` 通用字段。

数据库 COMMENT 应作为简洁的数据字典，不承担架构设计文档职责。本次仅修正注释文本，不修改组织模型或迁移机制。

## 2. 调整原则

1. 表注释只保留对象名称和镜像属性。
2. 字段注释只保留字段名称或简单含义。
3. 删除权限、维护方式、生命周期和业务处理规则等长句。
4. `model.Basic` 通用字段恢复平台统一注释。
5. 组织模块字段使用稳定、简短的词典式注释。

平台通用字段统一为：

| 字段 | 注释 |
| --- | --- |
| `id` | 主键ID |
| `gmt_create` | 创建时间 |
| `create_user` | 创建人ID |
| `create_name` | 创建人 |
| `gmt_modify` | 修改时间 |
| `modify_user` | 修改人ID |
| `modify_name` | 修改人 |
| `gmt_delete` | 删除时间 |
| `delete_user` | 删除人ID |
| `delete_name` | 删除人 |

## 3. 注释覆盖范围

本次修正以下九张组织表的表注释及全部实际持久化字段注释：

1. `org_legal_entity`
2. `org_unit`
3. `org_structure`
4. `org_structure_node`
5. `org_position`
6. `org_employee`
7. `org_assignment`
8. `org_sync_batch`
9. `org_sync_record`

典型字段调整为：

- `source_system_code`：源系统编码
- `source_id`：源对象ID
- `employee_no`：员工编号
- `mobile`：手机号
- `email`：邮箱
- `user_id`：当前应用账号ID
- `parent_node_id`：父节点ID
- `sync_status`：同步状态
- `error_message`：错误信息

## 4. Migration 方式

继续复用 E02-DB-Comment-001 注册的 `organization_database_comments` Migration：

1. 不修改 `migrationSteps()` 注册位置和名称。
2. 不修改历史 Migration。
3. 继续使用 PostgreSQL `COMMENT ON TABLE` 和 `COMMENT ON COLUMN`。
4. 重复执行会将对象注释设置为相同值，保持幂等。
5. 继续在同一事务中更新全部组织数据库注释。
6. 不修改表、字段、索引或约束。

## 5. 回归保护

自动化测试增加以下约束：

1. 平台通用字段注释必须保持统一固定值。
2. 组织表不得覆盖平台通用字段注释。
3. 表和字段注释不得包含句号、分号、换行或业务规则词语。
4. 注释长度保持在数据字典短语范围内。
5. 九张组织表全部字段仍必须有注释。
6. PostgreSQL 中连续执行两次后，COMMENT 值一致且表结构快照不变化。

## 6. 修改范围

- `backend/migrate/organization_comments.go`
- `backend/migrate/organization_comments_test.go`
- `docs/development/task/E02-DB-Comment-002.md`

本次未修改组织 Model、Migration 注册、Seed、API、页面、权限、Report 或其他业务模块。

## 7. 测试结果

执行并通过：

```text
cd backend
go test ./...
```

PostgreSQL 专项验证使用独立临时 schema，连续执行两次 COMMENT Migration，并校验表注释、字段注释及结构快照。
