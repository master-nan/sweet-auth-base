# E02-DB-Comment-001 Organization Database Comment Baseline

## 1. 任务目标

为 Organization Foundation 已有九张 PostgreSQL 表补充表注释和字段注释，形成组织主数据镜像数据库文档基线。

本任务只增加数据库注释，不改变表结构、字段、索引、约束、Seed、API、页面或数据权限。

## 2. 开始前工作区状态

任务开始前工作区已经存在用户此前未提交的 Report、Frontend、design 和相关基础文件改动，包括：

- `.gitignore`
- `backend/controller/report_controller.go`
- `backend/dto/request/report_req.go`
- `backend/dto/response/report_res.go`
- `backend/enum/enum.go`
- `backend/initialize/router.go`
- `backend/initialize/wire_gen.go`
- `backend/migrate/main.go`
- `backend/migrate/main_test.go`
- `backend/model/sys.go`
- `backend/repository/impl/sys_role_menu_button_impl.go`
- `backend/service/report_service.go`
- `docs/report-designer/`
- `frontend/`
- `design/`

本任务不覆盖、不回滚、不暂存上述历史改动。`backend/migrate/main_test.go` 仅精确提交本任务新增的迁移步骤断言。

## 3. 实际模型审计

注释定义以 `backend/model/org.go` 当前已提交模型为唯一依据，覆盖：

1. `org_legal_entity`
2. `org_unit`
3. `org_structure`
4. `org_structure_node`
5. `org_position`
6. `org_employee`
7. `org_assignment`
8. `org_sync_batch`
9. `org_sync_record`

共覆盖 9 张表、248 个实际持久化字段。公共 `model.Basic` 字段使用统一注释，组织领域字段逐表定义。

## 4. 注释边界

表注释明确：

- Organization 是唯一外部权威源的组织主数据镜像，不是 HR 或本地组织源。
- `org_legal_entity` 表达法人或核算主体。
- `org_unit` 表达管理组织单元。
- `org_structure_node` 只表达特定管理架构中的节点关系，不是组织实体。
- `org_employee.user_id` 只表示当前 Sweet Platform 应用实例账号绑定，不是集团统一身份或外部账号映射。
- `org_assignment` 表达人员、法人、管理组织和岗位之间的任职关系，不承担调岗、离职或人事流程。
- `org_sync_batch` 和 `org_sync_record` 只记录组织同步业务结果，不等同于 HTTP 技术执行日志。

字段注释覆盖内部ID、源系统标识、业务编码、有效期、状态、账号绑定、组织关系、树路径、同步状态和错误摘要等全部实际字段。

## 5. Migration 方式

新增独立迁移函数 `applyOrganizationDatabaseComments`，通过现有 `migrationSteps()` 注册为：

```text
organization_database_comments
```

执行规则：

1. 在核心表结构迁移完成后执行。
2. 仅 PostgreSQL 执行 `COMMENT ON TABLE` 和 `COMMENT ON COLUMN`。
3. 所有注释在同一数据库事务中写入。
4. 重复执行设置相同注释，保持幂等。
5. 非 PostgreSQL 测试数据库安全跳过，不执行不兼容 SQL。
6. 运行时根据 GORM Model 的实际持久化字段校验注释覆盖；新增字段但未补注释时迁移明确失败。
7. 表名和字段名使用受控标识符引用，注释文本进行 PostgreSQL 字面量转义。

本任务未修改历史 Migration。

## 6. 测试范围

自动化测试覆盖：

1. 九张组织表全部登记。
2. 每个 GORM 持久化字段均有非空注释。
3. 注释定义不得引用不存在的字段。
4. 非 PostgreSQL 环境重复执行不改变表结构。
5. PostgreSQL 隔离 schema 中重复执行两次成功。
6. PostgreSQL 表注释和全部字段注释可从系统目录读取。
7. PostgreSQL 执行前后的字段、索引和约束快照一致。
8. `migrationSteps()` 注册顺序稳定。

## 7. 测试结果

通过：

```text
cd backend
go test ./...
```

通过 PostgreSQL 专项验证：

```text
SWEET_TEST_POSTGRES_DSN='host=127.0.0.1 port=15432 user=sweet_admin password=*** dbname=sweet_admin sslmode=disable TimeZone=Asia/Shanghai' \
go test ./migrate -run TestOrganizationDatabaseCommentsPersistOnPostgreSQLWithoutSchemaChanges -count=1 -v
```

PostgreSQL 专项测试使用临时隔离 schema，测试结束后自动删除，没有修改开发库 `public` schema 的组织数据或结构。

## 8. 实际修改文件

- `backend/migrate/organization_comments.go`
- `backend/migrate/organization_comments_test.go`
- `backend/migrate/registry.go`
- `backend/migrate/main_test.go`
- `docs/development/task/E02-DB-Comment-001.md`

## 9. 影响范围与未实现范围

影响范围仅为执行正式 PostgreSQL Migration 后的数据库对象说明文本。

本任务未：

- 修改组织 Model。
- 修改表结构、索引或约束。
- 修改 Seed。
- 修改 Organization Service、Controller、API 或页面。
- 修改 Report、Frontend、低代码或数据权限。
- 向开发库写入组织业务数据。

## 10. 遗留问题

1. PostgreSQL 注释不会在 SQLite 中持久化；SQLite 只验证安全跳过和结构不变。
2. 后续组织模型新增持久化字段时，必须同步补充数据库字段注释，否则注释迁移会按设计失败。
3. 本任务不处理其他平台表的数据库注释，后续应按独立模块任务推进。
