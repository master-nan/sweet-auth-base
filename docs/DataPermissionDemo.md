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

准备一个运营角色：

- `tms_operator`：TMS 运营，拥有 `tms_waybill`、`tms_vehicle` 的菜单和按钮权限，数据权限策略为“当前用户归属”。

准备两个测试用户：

- `tms_east_operator_user`：归属值为华东运输公司 ID。
- `tms_south_operator_user`：归属值为华南运输公司 ID。

两个用户都分配到 `tms_operator`。角色本身不写死公司范围，运行时根据 `sys_user_dimension_value` 读取当前登录用户自己的公司归属。

如果业务项目已经在 `sys_user` 扩展了 `company_id` 字段，也可以把角色数据权限策略改为“当前用户字段”，范围值选择 `company_id`。这样不再需要维护 `sys_user_dimension_value`，运行时会直接读取登录用户自己的 `company_id`。

## 验证点

登录华东运营账号后：

- 查询 `tms_waybill`：只能看到华东公司的运单。
- 详情华南公司运单：应返回无权限或查不到。
- 新增运单时 `company_id = 1`：成功。
- 新增运单时 `company_id = 2`：拒绝。
- 更新华东公司运单并保持 `company_id = 1`：成功。
- 把华东公司运单改成 `company_id = 2`：拒绝。
- 删除华南公司运单：拒绝。
- 数据权限页右侧“权限排查”选择运单菜单、动作“查询”，应看到解析后的范围条件包含 `company_id` 和当前用户归属公司 ID。

关联下拉候选项是否被同一数据范围限制，取决于候选目标表对应菜单是否能解析数据权限。比如运单表里的 `company_id` 下拉如果要按公司权限过滤，需要 `tms_company` 目标菜单也具备可解析的菜单权限和数据权限上下文。

自动化回归：

```bash
source ~/.nvm/nvm.sh && nvm use 22
node scripts/smoke-tms-data-permission.mjs
```

项目里的 `scripts/smoke-lowcode.mjs` 仍覆盖通用低代码 CRUD、关联候选项和数据权限越权场景；`scripts/smoke-tms-data-permission.mjs` 覆盖 TMS 公司、运单、车辆和“当前用户归属”策略。“当前用户字段”策略由后端单元测试覆盖。
