# 通用数据权限 Demo

这个 Demo 用中性的“示例中心”说明数据权限怎么测，不依赖公司、组织或行业实体。

## Demo 表

建议用低代码创建三张业务表：

| 表 | 作用 | 关键字段 |
| --- | --- | --- |
| `demo_scope` | 业务范围来源 | `scope_code`、`scope_name`、`parent_id` |
| `demo_project` | 示例项目 | `project_code`、`project_name`、`scope_id` |
| `demo_ticket` | 示例事项 | `ticket_no`、`title`、`project_id`、`scope_id`、`status` |

`scope_id` 是这个 Demo 的数据权限字段。后续项目也可以换成 `tenant_id`、`project_id`、`owner_id`、`role_id` 等任意业务字段。

## 菜单

发布低代码页面：

- `demo_scope`：范围示例
- `demo_project`：项目示例
- `demo_ticket`：事项示例

确认角色已经勾选对应菜单和按钮：查询、详情、新增、编辑、删除、刷新。

## 数据权限配置

创建维度：

```json
{
  "code": "demo_scope",
  "name": "业务范围",
  "value_type": "number",
  "source_type": "table",
  "source_code": "demo_scope",
  "label_field": "scope_name",
  "value_field": "id",
  "parent_field": "parent_id",
  "state": true
}
```

给 `demo_project` 菜单绑定：

```json
{
  "dimension_code": "demo_scope",
  "field_code": "scope_id",
  "match_type": "in",
  "required": true,
  "actions": ["query", "detail", "create", "update", "delete", "export", "batch_delete"]
}
```

给 `demo_ticket` 菜单绑定同一条规则，字段仍是 `scope_id`。

## 角色授权

准备两个范围：

- Scope A：`scope_id = 1`
- Scope B：`scope_id = 2`

准备两个角色：

- `demo_all_scope`：策略为“全部”。
- `demo_scope_a`：策略为“指定范围”，范围值为 `["1"]`。

把测试用户分配到 `demo_scope_a`，并给它 `demo_project`、`demo_ticket` 的菜单和按钮权限。

## 验证点

登录测试用户后：

- 查询 `demo_ticket`：只能看到 Scope A 的事项。
- 详情 Scope B 事项：应返回无权限或查不到。
- 新增事项时 `scope_id = 1`：成功。
- 新增事项时 `scope_id = 2`：拒绝。
- 更新 Scope A 事项并保持 `scope_id = 1`：成功。
- 把 Scope A 事项改成 `scope_id = 2`：拒绝。
- 删除 Scope B 事项：拒绝。
- `demo_ticket.project_id` 如果关联 `demo_project`，下拉候选项也应只出现 Scope A 的项目。

项目里的 `scripts/smoke-lowcode.mjs` 已经用 `scope_id` 覆盖了低代码 CRUD、关联候选项和数据权限越权场景，可以作为自动化回归参考。
