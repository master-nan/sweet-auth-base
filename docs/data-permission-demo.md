# 通用数据权限 Demo

这个 Demo 用 TMS 运单场景说明数据权限怎么测。这里的“公司”只是业务示例维度，不是底座硬编码；正式项目也可以换成网点、项目、租户、负责人、课程等任意维度。

## Demo 表

建议用低代码创建三张业务表：

| 表 | 作用 | 关键字段 |
| --- | --- | --- |
| `tms_company` | 承运公司/运营公司 | `company_name`、`parent_id` |
| `tms_waybill` | 运单 | `waybill_no`、`company_id`、`customer_name`、`status` |
| `tms_vehicle` | 车辆 | `plate_no`、`company_id`、`driver_name` |

`company_id` 是这个 Demo 的数据权限字段。它表示当前业务数据属于哪家公司。

## 菜单

发布低代码页面：

- `tms_company`：公司管理
- `tms_waybill`：运单管理
- `tms_vehicle`：车辆管理

确认角色已经勾选对应菜单和按钮：查询、详情、新增、编辑、删除、刷新。

## 数据权限配置

创建维度：

```json
{
  "code": "tms_company",
  "name": "所属公司",
  "value_type": "number",
  "source_type": "table",
  "source_code": "tms_company",
  "label_field": "company_name",
  "value_field": "id",
  "parent_field": "parent_id",
  "state": true
}
```

给 `tms_waybill` 菜单绑定：

```json
{
  "dimension_code": "tms_company",
  "field_code": "company_id",
  "match_type": "in",
  "required": true,
  "actions": ["query", "detail", "create", "update", "delete", "export", "batch_delete"]
}
```

给 `tms_vehicle` 菜单绑定同一条规则，字段仍是 `company_id`。

## 角色授权

准备两家公司：

- 华东运输公司：`company_id = 1`
- 华南运输公司：`company_id = 2`

准备三个角色：

- `tms_admin`：TMS 管理员，策略为“全部”。
- `tms_east_operator`：华东运营，策略为“指定值”，范围值为 `["1"]`。
- `tms_south_operator`：华南运营，策略为“指定值”，范围值为 `["2"]`。

把测试用户分配到 `tms_east_operator`，并给它 `tms_waybill`、`tms_vehicle` 的菜单和按钮权限。

## 验证点

登录华东运营账号后：

- 查询 `tms_waybill`：只能看到华东公司的运单。
- 详情华南公司运单：应返回无权限或查不到。
- 新增运单时 `company_id = 1`：成功。
- 新增运单时 `company_id = 2`：拒绝。
- 更新华东公司运单并保持 `company_id = 1`：成功。
- 把华东公司运单改成 `company_id = 2`：拒绝。
- 删除华南公司运单：拒绝。
- `tms_waybill.company_id` 或关联车辆下拉候选项也应只出现当前账号有权限的数据。
- 数据权限页右侧“权限排查”选择运单菜单、动作“查询”，应看到最终范围条件包含 `company_id IN ["1"]`。

项目里的 `scripts/smoke-lowcode.mjs` 已经用 `scope_id` 覆盖了低代码 CRUD、关联候选项和数据权限越权场景，可以作为自动化回归参考。后续可以再补一份 TMS seed/smoke，让手测账号和页面数据更贴近这个 Demo。
