# 低代码配置手册

这份手册说明低代码页面、字段联动、按钮参数表单的常用配置方式。目标是：新增表、发布页面、配置按钮参数时，不需要翻代码也能知道该怎么填。

## 0. 先看哪个章节

| 你要做什么 | 建议阅读 |
| --- | --- |
| 从一张业务表做成低代码页面 | 1.1、16、17、9、20 |
| 配置字段输入组件、列表、表单、详情 | 7、17、[字段类型指南](field-type-guide.md) |
| 配置下拉、字典、关联表、级联 | 2、3、4、5、8、[联动配置指南](linkage-config.md) |
| 配置行按钮、详情按钮、自定义业务按钮 | 11、12、13、14、18、19 |
| 配置按钮弹出的参数表单 | 6、7、12、18 |
| 排查无权限、下拉为空、字段显示不对 | 9、20 |

## 1. 配置写在哪

| 场景 | 配置位置 | 作用 |
| --- | --- | --- |
| 普通低代码页面字段 | `sys_table_field` | 决定列表、查询、新增、编辑、详情如何渲染 |
| 字段关联其他表 | `sys_table_field.linkage_config` | 例如用户表选择角色、事项选择客户 |
| 字段使用固定字典 | `sys_table_field.dict_code` | 例如状态、类型、等级 |
| 按钮点击后弹参数表单 | `sys_menu_button.params_schema` | 例如标记完成、审核、修改业务范围这类自定义动作 |
| 按钮本身做什么 | `sys_menu_button.event_action` | 前端按动作执行内置逻辑，例如 `detail`、`update`、`delete`、`custom` |

普通新增/编辑页面的字段联动写在字段元数据里；按钮点击后弹出的参数表单写在按钮的 `params_schema` 里。

## 1.1 从零配置一个低代码页面

一个低代码页面上线前，建议按这个顺序做：

1. 在数据库里建好业务表，字段命名尽量稳定，主键用 `id`。
2. 进入“数据管理”，选择目标表，执行“初始化元数据”或“同步字段”。
3. 在字段管理里调整字段：字段名称、字段类型、输入类型、是否列表显示、是否新增/编辑、是否查询、是否排序、字典、联动配置、表单跨度。
4. 发布低代码菜单。发布不是生成前端代码，而是在菜单表里创建或恢复一个菜单，让它能出现在左侧菜单和页签里。
5. 在菜单管理里检查生成的按钮：查询、元数据、新增、编辑、删除、详情、刷新等按钮是否符合当前页面需要。
6. 在角色管理里给角色授权：先勾选菜单权限，再勾选该菜单下需要的按钮权限。
7. 打开页面实测列表、搜索、高级查询、新增、编辑、详情、删除、文件字段、富文本字段。
8. 如果字段关联了其他表，还要确认当前角色有目标表的查询权限，否则下拉可能为空或 403。

不要直接改前端路由来加低代码页面。低代码页面应该通过数据管理发布，前端只保留通用页面入口。

发布时只能选择“目录菜单”作为父级，例如系统管理、开发管理这类菜单分组；已经绑定组件的固定页面、已经发布的低代码页面、普通业务菜单都不应该作为发布父级。这样可以避免低代码页面挂到某个功能页下面，导致面包屑、左侧选中和权限判断混乱。

## 2. 字段关联表下拉

适用场景：字段存的是另一张表的 ID，但页面上要显示可读文本。例如 `project_id` 存项目 ID，表单里显示项目名称。

优先在字段管理界面里配置关联表、显示字段、值字段、分页和联动筛选；只有批量初始化、排障或高级配置时才直接编辑 `linkage_config` JSON。完整字段说明见 [联动配置指南](linkage-config.md)。

### 2.1 写在字段 `linkage_config`

```json
{
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableCode": "demo_project",
    "labelKey": "project_name",
    "valueKey": "id",
    "pageSize": 50
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `enabled` | 是否启用联动 |
| `mode` | `relation` 表示普通关联下拉 |
| `tableCode` | 关联表编码，优先用表编码，不建议写死表 ID |
| `labelKey` | 下拉显示字段，例如项目名称 `project_name` |
| `valueKey` | 实际保存字段，一般是 `id` |
| `pageSize` | 每页加载多少条，建议 50，最大不要超过 200 |

### 2.2 写在按钮 `params_schema`

按钮参数 Schema 支持简写，不需要手写完整 `linkage_config`：

```json
{
  "fields": [
    {
      "field_code": "project_id",
      "field_name": "项目",
      "input_type": "select",
      "required": true,
      "relation": {
        "table_code": "demo_project",
        "label_field": "project_name",
        "value_field": "id",
        "page_size": 50
      }
    }
  ]
}
```

前端会自动转换为 `linkage_config` 并渲染为远程下拉。

## 3. 关联下拉支持什么能力

关联下拉不是一次性加载全部数据，而是：

| 能力 | 说明 |
| --- | --- |
| 打开加载 | 打开下拉时加载第一页 |
| 输入筛选 | 在下拉中输入关键字，通过后端 `quick_query.keyword` 查询 |
| 滚动加载 | 滚动到底后自动加载下一页 |
| 编辑回显 | 编辑已有数据时，如果保存的是 ID，会补查当前值并显示成文本 |
| 联动刷新 | 父字段变化后，子字段选项会按 `filterMapping` 重新加载 |

## 4. 联动筛选怎么填

联动筛选写在 `filterMapping` 或 `filter_mapping`。

当前实现的方向是：

```json
{
  "关联表字段": "当前表单字段"
}
```

也就是说：左边是要查的目标表字段，右边是当前表单里取值的字段。

### 4.1 项目按业务范围筛选

当前表单有 `scope_id`，项目表也有 `scope_id`：

```json
{
  "field_code": "project_id",
  "field_name": "项目",
  "input_type": "select",
  "required": true,
  "relation": {
    "table_code": "demo_project",
    "label_field": "project_name",
    "value_field": "id",
    "page_size": 50,
    "filter_mapping": {
      "scope_id": "scope_id"
    }
  }
}
```

当表单里 `scope_id = 1001` 时，项目下拉查询会附加：

```json
{
  "filters": {
    "scope_id": 1001
  }
}
```

### 4.2 字段名不一样的情况

当前表单字段叫 `selected_scope`，项目表字段叫 `scope_id`：

```json
"filter_mapping": {
  "scope_id": "selected_scope"
}
```

不要反过来写。

## 5. 静态选项、字典、关联表怎么选

| 数据来源 | 配置方式 | 适合场景 |
| --- | --- | --- |
| 固定少量选项 | `options` | 只服务当前按钮参数，例如普通/加急 |
| 系统字典 | `dict_code` | 多个页面共用枚举，例如状态、类型 |
| 关联表 | `relation` / `linkage_config` | 数据来自业务表，例如项目、客户、员工 |
| 树形数据 | `cascader` / `mode=cascader` | 菜单、组织、区域、分类 |

### 5.1 静态选项

```json
{
  "field_code": "priority",
  "field_name": "优先级",
  "input_type": "select",
  "required": true,
  "default_value": "normal",
  "options": [
    { "label": "普通", "value": "normal" },
    { "label": "加急", "value": "urgent" }
  ]
}
```

### 5.2 字典选项

```json
{
  "field_code": "status",
  "field_name": "状态",
  "input_type": "select",
  "required": true,
  "dict_code": "order_status"
}
```

### 5.3 关联表选项

```json
{
  "field_code": "customer_id",
  "field_name": "客户",
  "input_type": "select",
  "relation": {
    "table_code": "base_customer",
    "label_field": "customer_name",
    "value_field": "id",
    "page_size": 50
  }
}
```

## 6. 按钮参数 Schema 完整例子

例如“标记完成”按钮需要填写处理意见、优先级、是否通知、项目：

```json
{
  "fields": [
    {
      "field_code": "remark",
      "field_name": "处理意见",
      "input_type": "textarea",
      "required": true,
      "placeholder": "请输入本次处理意见",
      "form_span": 12
    },
    {
      "field_code": "priority",
      "field_name": "优先级",
      "input_type": "select",
      "required": true,
      "default_value": "normal",
      "options": [
        { "label": "普通", "value": "normal" },
        { "label": "加急", "value": "urgent" }
      ]
    },
    {
      "field_code": "notify",
      "field_name": "是否通知",
      "input_type": "boolean",
      "default_value": true
    },
    {
      "field_code": "project_id",
      "field_name": "项目",
      "input_type": "select",
      "required": true,
      "relation": {
        "table_code": "demo_project",
        "label_field": "project_name",
        "value_field": "id",
        "page_size": 50,
        "filter_mapping": {
          "scope_id": "scope_id"
        }
      }
    }
  ]
}
```

## 7. 常用字段属性

| 属性 | 说明 |
| --- | --- |
| `field_code` | 参数编码，提交给后端的 key |
| `field_name` | 表单显示名称 |
| `input_type` | 输入组件类型 |
| `required` | 是否必填 |
| `default_value` | 默认值 |
| `placeholder` | 占位提示 |
| `options` | 静态选项 |
| `dict_code` | 字典编码 |
| `relation` | 关联表下拉简写 |
| `cascader` | 级联选择简写 |
| `form_span` | 表单宽度；字段元数据建议填 `1` 半行、`2` 整行；参数 Schema 里 `12` 也会按整行兼容 |
| `detail_span` | 详情页宽度；字段元数据建议填 `1` 到 `4`，`4` 为整行；参数 Schema 里 `12` 也会按整行兼容 |

## 8. 级联选择例子

```json
{
  "field_code": "area_id",
  "field_name": "区域",
  "input_type": "cascader",
  "required": true,
  "cascader": {
    "table_code": "base_area",
    "label_field": "area_name",
    "value_field": "id",
    "parent_field": "parent_id",
    "root_value": 0,
    "selectable": "leaf",
    "show_path": true,
    "page_size": 1000
  }
}
```

## 9. 权限前提

关联表下拉本质会调用低代码查询接口：

```text
POST /admin/generalization/query/code/{tableCode}
```

所以目标表需要满足至少一个条件：

1. 目标表已经发布成低代码菜单，并且当前角色有该菜单的查询权限。
2. 当前按钮或页面通过权限规则允许读取该目标表。

如果下拉为空或 403，先检查目标表是否发布、当前角色是否有查询按钮或对应接口权限。

## 10. 推荐配置习惯

1. 关联业务表优先写 `table_code`，不要写表 ID。
2. 大表下拉 `page_size` 用 50，依赖输入筛选和滚动加载。
3. 父子联动要明确 `filter_mapping` 方向：`关联表字段 -> 当前表单字段`。
4. 多页面共用枚举用字典，不要在每个按钮里重复写 `options`。
5. 单个按钮独有的小选项用 `options`，不用新建字典。
6. 富文本、附件这类大内容字段建议 `form_span` 和 `detail_span` 设置为 12。

## 11. 按钮事件动作

按钮真正执行什么，不看按钮编码，而是看 `event_action`。按钮编码只负责权限和唯一标识，不能用编码去猜动作。

| 动作 | 适合位置 | 说明 |
| --- | --- | --- |
| `query` | 表格顶部 | 查询列表，通常作为页面基础权限 |
| `metadata` | 表格顶部 | 查询页面元数据，通常作为页面基础权限 |
| `detail` | 行按钮 | 打开当前行详情页 |
| `create` | 表格顶部 | 新增记录 |
| `update` | 行按钮 | 编辑当前行 |
| `delete` | 行按钮 | 删除当前行 |
| `refresh` | 表格顶部、详情顶部 | 刷新当前页面 |
| `batch_delete` | 表格顶部 | 删除勾选的多行 |
| `copy` | 行按钮 | 复制当前行并打开新增 |
| `export` | 表格顶部 | 导出当前列表或选中数据 |
| `navigate` | 任意位置 | 跳转到 `api_path` 配置的前端路由 |
| `custom` | 任意位置 | 调用 `api_path` 配置的后端接口 |

业务按钮建议使用 `custom`，例如标记完成、审核、驳回、修改业务范围。自定义按钮一般需要同时配置：

| 字段 | 是否需要 | 说明 |
| --- | --- | --- |
| `event_action` | 必填 | 固定填 `custom` |
| `api_path` | 必填 | 后端业务接口，例如 `/admin/demo/ticket/mark-done` |
| `http_method` | 必填 | `POST`、`PUT`、`DELETE`、`GET` |
| `params_schema` | 按需 | 点击按钮后需要补充参数时填写 |
| `confirm_text` | 按需 | 执行前二次确认 |
| `disable_when` | 按需 | 当前行不满足条件时禁用按钮 |
| `before_hooks` | 按需 | 执行前置检查 |
| `after_hooks` | 按需 | 执行成功后的前端动作 |

## 12. 按钮参数提交格式

按钮带 `params_schema` 时，点击按钮会先弹参数表单。提交后前端会把参数放到 `params` 里传给后端。

`custom` 按钮默认请求体格式：

```json
{
  "table_code": "demo_file_page",
  "row": {
    "id": 1001,
    "title": "测试记录"
  },
  "selection": [],
  "params": {
    "remark": "处理意见",
    "priority": "normal",
    "notify": true,
    "project_id": 2001
  }
}
```

后端接口不要从按钮编码推断业务，应该按接口自身的业务参数处理。

## 13. 禁用条件 `disable_when`

`disable_when` 用来控制按钮是否可点。它只负责前端显示禁用，不代替后端权限校验。

可读取的上下文：

| 前缀 | 说明 |
| --- | --- |
| `row.xxx` | 当前行字段，行按钮和详情按钮常用 |
| `selectionCount` | 当前勾选数量 |
| `selection.xxx` | 勾选数据 |
| `query.xxx` | 当前查询条件 |
| `params.xxx` | 参数弹框里的值，主要用于后续扩展 |

支持操作符：

| 操作符 | 说明 |
| --- | --- |
| `eq` | 等于 |
| `ne` | 不等于 |
| `gt` / `gte` | 大于 / 大于等于 |
| `lt` / `lte` | 小于 / 小于等于 |
| `in` / `not_in` | 在集合内 / 不在集合内 |
| `includes` / `not_includes` | 包含 / 不包含 |
| `empty` / `not_empty` | 空 / 非空 |
| `truthy` / `falsy` | 真值 / 假值 |

当前行状态为已关闭时禁用：

```json
{
  "field": "row.status",
  "op": "eq",
  "value": "closed"
}
```

没有选中数据时禁用批量按钮：

```json
{
  "field": "selectionCount",
  "op": "eq",
  "value": 0
}
```

多个条件全部满足才禁用：

```json
{
  "all": [
    {
      "field": "row.status",
      "op": "eq",
      "value": "closed"
    },
    {
      "field": "row.locked",
      "op": "truthy"
    }
  ]
}
```

多个条件任意满足就禁用：

```json
{
  "any": [
    {
      "field": "selectionCount",
      "op": "eq",
      "value": 0
    },
    {
      "field": "row.state",
      "op": "falsy"
    }
  ]
}
```

## 14. 前后钩子

钩子不是数据库里写脚本执行，而是从前端已注册的安全函数里选择。这样不会把任意代码执行权限暴露给数据库配置。

`before_hooks` 在按钮动作执行前运行。任意一个钩子返回失败，按钮动作就中止。

```json
["requireRow"]
```

当前内置前置钩子：

| 钩子 | 说明 |
| --- | --- |
| `requireRow` | 必须有当前行，适合行按钮和详情按钮 |
| `requireSelection` | 必须勾选至少一行，适合批量按钮 |

`after_hooks` 在按钮动作执行成功后运行。

```json
["refresh", "clearSelection"]
```

当前内置后置钩子：

| 钩子 | 说明 |
| --- | --- |
| `refresh` | 刷新当前数据 |
| `clearSelection` | 清空表格勾选 |
| `closeDialog` | 关闭当前参数弹框 |

如果后续有特殊业务场景，例如“标记完成后刷新看板”、“审核通过后跳转详情”，建议新增受控钩子，例如 `refreshBoard`、`openDetail`，不要直接在数据库里写 JS。

## 15. 标记完成按钮完整例子

场景：先选业务范围，再按业务范围筛选项目，填写处理意见后提交标记完成。

业务范围字段先从 `demo_scope` 表选择：

```json
{
  "field_code": "scope_id",
  "field_name": "业务范围",
  "input_type": "select",
  "required": true,
  "relation": {
    "table_code": "demo_scope",
    "label_field": "scope_name",
    "value_field": "id",
    "page_size": 50
  }
}
```

项目字段依赖 `scope_id`：

```json
{
  "field_code": "project_id",
  "field_name": "项目",
  "input_type": "select",
  "required": true,
  "relation": {
    "table_code": "demo_project",
    "label_field": "project_name",
    "value_field": "id",
    "page_size": 50,
    "filter_mapping": {
      "scope_id": "scope_id"
    }
  }
}
```

按钮配置建议：

| 字段 | 值 |
| --- | --- |
| 按钮名称 | 标记完成 |
| 按钮编码 | `demo_ticket_mark_done` |
| 按钮位置 | 行按钮或详情顶部 |
| 事件动作 | `custom` |
| 接口路径 | `/admin/demo/ticket/mark-done` |
| 请求方法 | `POST` |
| 前置钩子 | `["requireRow"]` |
| 后置钩子 | `["refresh"]` |

`params_schema`：

```json
{
  "fields": [
    {
      "field_code": "scope_id",
      "field_name": "业务范围",
      "input_type": "select",
      "required": true,
      "relation": {
        "table_code": "demo_scope",
        "label_field": "scope_name",
        "value_field": "id",
        "page_size": 50
      }
    },
    {
      "field_code": "project_id",
      "field_name": "项目",
      "input_type": "select",
      "required": true,
      "relation": {
        "table_code": "demo_project",
        "label_field": "project_name",
        "value_field": "id",
        "page_size": 50,
        "filter_mapping": {
          "scope_id": "scope_id"
        }
      }
    },
    {
      "field_code": "remark",
      "field_name": "处理意见",
      "input_type": "textarea",
      "required": true,
      "placeholder": "请输入本次处理意见",
      "form_span": 12
    },
    {
      "field_code": "notify",
      "field_name": "是否通知",
      "input_type": "boolean",
      "default_value": true
    }
  ]
}
```

禁用条件示例：已完成或已取消的事项不能标记完成。

```json
{
  "field": "row.status",
  "op": "in",
  "value": ["finished", "cancelled"]
}
```

## 16. 发布、取消发布和菜单关系

低代码发布的本质是维护菜单数据，不是生成新的 Vue 文件。

| 操作 | 实际含义 |
| --- | --- |
| 发布 | 给当前表创建或恢复一个菜单，并生成默认按钮 |
| 取消发布 | 禁用对应菜单，表结构和字段元数据仍保留 |
| 删除菜单 | 菜单不可访问；如果是低代码表，数据管理里的发布状态应以后端菜单状态为准 |
| 刷新菜单缓存 | 菜单、按钮或权限变更后，让前端重新拿到最新菜单 |

发布时父级菜单应该选择真实业务归属。例如文件演示属于开发管理，就发布到开发管理下；如果以后是 业务示例 事项，就发布到示例中心或事项管理下。

### 16.1 哪些页面不应该发布成低代码

下面这些属于前端固定功能页，不应该再通过低代码发布一份：

| 类型 | 例子 | 原因 |
| --- | --- | --- |
| 系统固定页面 | 菜单管理、角色管理、用户管理、配置管理 | 页面有定制交互，不是单表 CRUD |
| 开发工具页 | 数据管理、字典管理 | 本身负责维护低代码元数据 |
| 复杂工作台 | 仪表盘、权限分配弹框、表结构工作台 | 需要定制布局或组合多个接口 |

低代码更适合业务数据表，例如客户档案、项目档案、事项、费用规则、公告、附件演示等。

### 16.2 发布边界怎么判断

发布边界以后端菜单数据为准，不以前端 `router.ts` 写了哪些路由为准。

菜单表用显式字段描述页面边界，不再通过 `option`、菜单编码或组件路径猜测绑定关系：

| 字段 | 作用 |
| --- | --- |
| `page_type` | 页面类型：`directory` 表示目录，`fixed` 表示前端固定功能页，`low_code` 表示低代码发布页 |
| `table_code` | 当前菜单绑定的表编码；固定功能页和低代码页都应显式填写 |
| `option` | 扩展配置，不再作为表绑定依据 |

| 菜单类型 | 判断方式 | 能否从数据管理发布 |
| --- | --- | --- |
| 固定功能页 | `page_type = fixed` 且 `table_code` 等于当前表编码 | 不能发布 |
| 低代码页面 | `page_type = low_code` 且 `table_code` 等于当前表编码 | 已发布或可恢复 |
| 父级目录 | `page_type = directory`，只负责挂载子菜单 | 可作为发布目录 |
| 新业务表 | 没有固定功能菜单绑定同一个 `table_code` | 可以发布 |

这样做的好处是：菜单管理、角色管理、用户管理这些固定页面不会被重复发布成低代码 CRUD；新建业务表只要没有被固定页面绑定，就可以选择父级目录发布。

如果数据管理显示“固定页面”，说明该表已经由系统定制页面承载；要调整菜单入口，应去菜单管理改原菜单，不要再点发布生成第二个入口。

## 17. 字段配置清单

字段元数据决定列表、表单、详情和查询怎么渲染。常用字段如下：

| 字段 | 作用 | 建议 |
| --- | --- | --- |
| `field_name` | 页面显示名称 | 用中文业务名，例如“项目”“处理意见” |
| `field_code` | 字段编码 | 对应数据库字段名 |
| `field_type` | 数据类型 | 用字段类型字典，例如字符串、文本、布尔、大数字 |
| `input_type` | 输入控件 | 用输入类型字典，例如输入框、下拉选择、文件选择、富文本 |
| `dict_code` | 字典编码 | 状态、类型、等级等复用枚举放字典 |
| `linkage_config` | 关联和级联配置 | 下拉来自业务表时填写 |
| `form_span` | 新增/编辑表单宽度 | `0` 自动，`1` 半行，`2` 整行 |
| `detail_span` | 详情页宽度 | `0` 自动，`1` 四分之一，`2` 半行，`4` 整行 |
| `is_list_show` | 是否列表显示 | 大字段、内部字段通常不显示 |
| `is_insert_show` | 是否新增显示 | 系统生成字段一般关闭 |
| `is_update_show` | 是否编辑显示 | 主键、创建人、创建时间一般关闭 |
| `is_quick_search` | 是否快捷搜索 | 名称、编码、手机号等常用关键词 |
| `is_advanced_search` | 是否高级查询 | 状态、时间、类型、归属等结构化条件 |
| `is_sort` | 是否允许排序 | 时间、金额、序号、状态等 |

### 17.1 输入类型怎么选

| 业务字段 | 推荐输入类型 | 备注 |
| --- | --- | --- |
| 名称、编码、标题 | 输入框 | `field_type` 通常是字符串 |
| 备注、说明 | 多行文本 | 表单可设整行 |
| 金额、数量、排序 | 数字输入 | 注意小数位和默认值 |
| 状态、类型、等级 | 下拉选择 | 优先配字典 |
| 是否启用、是否通知 | 布尔开关 | 列表和详情显示为是/否 |
| 日期、时间 | 日期/日期时间/时间选择 | 和数据库类型保持一致 |
| 附件 | 文件选择 | 可保存单个或多个文件 ID |
| 正文、富文本说明 | 富文本编辑器 | 建议 `form_span=2`、`detail_span=4` |
| JSON 扩展配置 | JSON 编辑器 | 只给开发或管理员使用 |
| 地区、组织、分类树 | 级联选择 | 配 `cascader` |

### 17.2 文件和富文本字段怎么配

文件和富文本都不要当普通文本字段处理，否则新增、编辑、详情都会退化成输入框或直接显示原始值。

| 场景 | 字段配置 | 页面表现 |
| --- | --- | --- |
| 单个或多个附件 | `input_type = file` 或 `file_picker` | 新增/编辑显示上传控件，保存文件 ID 数组 |
| 富文本正文 | `input_type = rich_text` | 新增/编辑显示富文本编辑器，详情按 HTML 渲染 |
| 大段说明 | `input_type = textarea` | 表单和详情建议整行展示 |

文件字段保存的是文件 ID 数组，例如 `[123, 456]`。只上传一个文件时也按数组保存，方便以后同一个字段支持多个附件，不需要改表结构。

上传超过分片阈值的文件时，前端会走分片上传：先初始化上传任务，再并发上传分片，最后合并文件。上传进度应该显示在字段下方，不能让用户只看到全局转圈。分片上传不是为了减少总流量，而是为了支持大文件、失败重试和断点续传。

详情页展示文件字段时，应该显示文件名和操作按钮，而不是直接显示 ID。能在线预览的类型走预览，不能预览的类型走下载。PDF、图片、文本、CSV 属于浏览器比较稳定的预览类型；Office 文件是否能预览取决于前端 viewer 和后端文件访问地址，不能保证所有浏览器都原生支持。

富文本里上传图片或附件时，应保存稳定的文件标识，例如 `data-file-uuid`。如果历史内容里只有临时地址或错误的 `data-file-uuid`，详情页无法可靠还原预览地址，需要重新编辑保存一次。

## 18. 参数 Schema 支持的写法

`params_schema` 可以用两种格式：`fields` 数组格式和 JSON Schema 简写格式。推荐优先用 `fields`，更直观。

### 18.1 fields 数组格式

```json
{
  "fields": [
    {
      "field_code": "remark",
      "field_name": "处理意见",
      "input_type": "textarea",
      "required": true,
      "form_span": 12
    }
  ]
}
```

### 18.2 JSON Schema 简写格式

```json
{
  "type": "object",
  "required": ["remark", "priority"],
  "properties": {
    "remark": {
      "title": "处理意见",
      "type": "string",
      "x-input-type": "textarea",
      "x-form-span": 12
    },
    "priority": {
      "title": "优先级",
      "type": "string",
      "enum": ["normal", "urgent"],
      "enumNames": ["普通", "加急"]
    }
  }
}
```

### 18.3 input_type 可用值

| 值 | 渲染 |
| --- | --- |
| `input` / `text` | 输入框 |
| `number` / `input_number` | 数字输入 |
| `textarea` | 多行文本 |
| `select` | 下拉选择 |
| `date` | 日期 |
| `datetime` | 日期时间 |
| `time` | 时间 |
| `year` | 年份 |
| `year_month` | 年月 |
| `file` / `file_picker` | 文件选择 |
| `boolean` / `bool` | 布尔选择 |
| `json` / `json_editor` | JSON 编辑器 |
| `array` / `array_input` | 数组输入 |
| `key_value` / `key_value_editor` | 键值对编辑 |
| `cascader` | 级联选择 |
| `rich_text` / `richtext` | 富文本编辑器 |

### 18.4 数据来源写法

静态选项：

```json
{
  "field_code": "priority",
  "field_name": "优先级",
  "input_type": "select",
  "options": [
    { "label": "普通", "value": "normal" },
    { "label": "加急", "value": "urgent" }
  ]
}
```

字典：

```json
{
  "field_code": "status",
  "field_name": "状态",
  "input_type": "select",
  "dict_code": "order_status"
}
```

也可以用 `data_source`：

```json
{
  "field_code": "status",
  "field_name": "状态",
  "input_type": "select",
  "data_source": "dict:order_status"
}
```

关联表：

```json
{
  "field_code": "project_id",
  "field_name": "项目",
  "input_type": "select",
  "relation": {
    "table_code": "demo_project",
    "label_field": "project_name",
    "value_field": "id",
    "page_size": 50
  }
}
```

级联：

```json
{
  "field_code": "area_id",
  "field_name": "区域",
  "input_type": "cascader",
  "cascader": {
    "table_code": "base_area",
    "label_field": "area_name",
    "value_field": "id",
    "parent_field": "parent_id",
    "root_value": 0,
    "selectable": "leaf",
    "show_path": true
  }
}
```

## 19. 按钮配置字段说明

| 字段 | 说明 |
| --- | --- |
| `name` | 按钮名称，页面上显示的文字 |
| `code` | 按钮唯一编码，用于权限标识；不要依赖编码推断动作 |
| `position` | 按钮位置：行按钮、表格顶部、详情顶部等 |
| `event_action` | 按钮动作，必须从动作字典里选 |
| `icon` | 图标名称，建议使用 Quasar/Material Icons 名称 |
| `color` | 按钮颜色，例如 `primary`、`negative`、`warning` |
| `display_mode` | 显示方式：自动、仅图标、仅文字、图标文字 |
| `sequence` | 排序，数字越小越靠前 |
| `api_path` | 自定义按钮或接口权限对应的后端路径 |
| `http_method` | 请求方法，例如 `GET`、`POST`、`PUT`、`DELETE` |
| `params_schema` | 点击按钮时弹出的参数表单 |
| `confirm_text` | 点击后是否二次确认 |
| `disable_when` | 前端禁用条件 |
| `before_hooks` | 前置钩子 JSON 数组 |
| `after_hooks` | 后置钩子 JSON 数组 |

### 19.1 详情按钮怎么配

行详情按钮只需要配置按钮动作，不需要自己拼详情 URL。

| 字段 | 值 |
| --- | --- |
| 按钮名称 | 详情 |
| 按钮编码 | 建议 `{menu_code}_detail`，只要唯一即可 |
| 按钮位置 | 行按钮 |
| 事件动作 | `detail` |
| 图标 | `visibility` |

前端点击时会拿当前行 `id` 打开通用详情页，并根据当前菜单和表编码做权限校验。

### 19.2 自定义业务按钮怎么配

例如“标记完成”“审核”“修改业务范围”：

| 字段 | 值 |
| --- | --- |
| 事件动作 | `custom` |
| 接口路径 | 后端真实业务接口 |
| 请求方法 | 和后端接口一致 |
| 参数 Schema | 需要用户补充参数时填写 |
| 前置钩子 | 常用 `["requireRow"]` |
| 后置钩子 | 常用 `["refresh"]` |

自定义按钮只是把“当前行、勾选行、参数表单值”提交给后端，真正业务仍然应该由后端接口完成。

### 19.3 参数 Schema、禁用条件和钩子的边界

| 能力 | 能做什么 | 不应该做什么 |
| --- | --- | --- |
| `params_schema` | 配置弹框表单，让用户补充参数 | 不能代替后端业务校验 |
| `disable_when` | 根据当前行或选择状态禁用按钮 | 不能代替权限校验 |
| `before_hooks` | 做受控的前置检查 | 不能在数据库里写任意 JS |
| `after_hooks` | 成功后刷新、清空选择、关闭弹框 | 不能承载复杂业务流程 |

复杂业务流程建议写后端接口或工作流，再由 `custom` 按钮触发。

## 20. 权限和接口依赖怎么理解

页面能看到按钮，只代表角色有这个按钮权限；接口是否能调用，还要看该接口是否被授权。

当前模型里用 `is_button` 区分两类配置：

| `is_button` | 含义 | 是否展示在页面 |
| --- | --- | --- |
| `true` | 页面按钮，例如新增、编辑、删除、详情、自定义业务动作 | 展示 |
| `false` | 接口权限，例如列表查询、元数据、详情接口、某些后台依赖接口 | 不展示 |

`is_hidden` 只控制页面按钮是否隐藏；接口权限是否展示只由 `is_button=false` 决定。不要把“接口权限”理解成“隐藏按钮”，它的作用是给后端接口授权，不负责控制页面显示隐藏。

低代码推荐规则：

1. 页面列表需要菜单权限和查询按钮权限。
2. 新增、编辑、删除、详情需要对应按钮权限。
3. 关联下拉会查目标表，目标表也需要可查询权限。
4. 自定义按钮调用业务接口时，接口路径和请求方法要配置清楚，并给角色授权。
5. 不要靠按钮编码推断权限动作，动作看 `event_action`。

如果一个按钮没有后端接口，例如纯前端打开详情页，可以不填 `api_path`。如果按钮会调用后端接口，必须填 `api_path` 和 `http_method`。

### 20.1 数据权限当前边界

当前已经落地的是菜单权限、按钮权限、接口权限，以及通用数据权限基础模型。也就是说：

1. 能不能进入页面，看菜单权限。
2. 能不能看到按钮，看按钮权限。
3. 能不能调用接口，看接口权限。
4. 低代码列表、详情、新增、编辑、删除会按当前菜单绑定的维度、字段和角色/用户范围做行级检查。
5. 树形范围展开和更完整的审计说明仍按统一数据权限模型继续演进。

不要在某个控制器里临时写一套固定业务字段、`table_code`、`field_code` 的判断来当最终数据权限。详情、列表、导出、批量操作、文件预览和下载都应该复用同一套数据权限解析结果。当前模型见 [通用数据权限设计与实现说明](data-permission-design.md)。

### 20.2 通用接口为什么要二次权限判断

低代码列表、详情、新增、编辑、删除共用一组通用接口，例如：

```text
POST /admin/generalization/query/code/{tableCode}
GET  /admin/generalization/detail/code/{tableCode}/{id}
```

这类接口不能只靠 URL 判断权限，因为同一个 URL 模板可以访问不同表。后端处理顺序是：

1. 中间件只允许已登录用户进入通用接口。
2. 控制器根据 `tableCode` 找到发布菜单。
3. 再按当前用户角色、菜单绑定表、按钮 `event_action` 判断是否有权限。
4. 自定义接口还会按按钮配置的 `api_path` 和 `http_method` 授权。

前端 URL 不需要带 `menu_id`。如果请求体里出现 `menu_id`，它只作为审计和后续数据权限上下文，不作为页面路由参数。

## 21. 常见问题排查

| 问题 | 优先检查 |
| --- | --- |
| 列表 403 | 菜单是否发布、角色是否勾选菜单、是否有查询按钮权限 |
| 点击详情 403 | 行按钮是否授权、当前表是否有详情按钮、详情接口权限是否随页面授权 |
| 点击编辑 403 | 编辑按钮是否授权；编辑取数应使用当前表和当前行 ID |
| 关联下拉为空 | 目标表是否发布、目标表查询权限、`table_code` 是否正确、`filter_mapping` 是否过滤过严 |
| 参数 Schema 全变成输入框 | JSON 是否合法，`input_type` 是否写对 |
| 字典不显示 | `dict_code` 是否存在，字典项是否启用 |
| 文件字段变文本框 | 字段输入类型必须是文件选择，参数 Schema 用 `file` 或 `file_picker` |
| 富文本详情显示 HTML | 字段输入类型必须是富文本，详情会按富文本渲染 |
| 必填没生效 | `required` 必须是布尔值 `true`，不要写字符串 `"true"` |
| 按钮总是灰掉 | 检查 `is_disabled` 和 `disable_when` |
| 后置不刷新 | `after_hooks` 是否是 JSON 数组，例如 `["refresh"]` |

## 22. 推荐复制模板

### 22.1 最小详情按钮

```json
{
  "name": "详情",
  "code": "demo_file_page_detail",
  "position": "行按钮",
  "event_action": "detail",
  "icon": "visibility",
  "color": "primary",
  "display_mode": "auto",
  "sequence": 10
}
```

### 22.2 带参数的业务按钮

```json
{
  "name": "处理",
  "code": "demo_file_page_process",
  "position": "行按钮",
  "event_action": "custom",
  "icon": "tune",
  "color": "primary",
  "api_path": "/admin/demo/file/process",
  "http_method": "POST",
  "before_hooks": "[\"requireRow\"]",
  "after_hooks": "[\"refresh\"]",
  "params_schema": {
    "fields": [
      {
        "field_code": "remark",
        "field_name": "处理意见",
        "input_type": "textarea",
        "required": true,
        "form_span": 12
      },
      {
        "field_code": "priority",
        "field_name": "优先级",
        "input_type": "select",
        "required": true,
        "default_value": "normal",
        "options": [
          { "label": "普通", "value": "normal" },
          { "label": "加急", "value": "urgent" }
        ]
      }
    ]
  }
}
```

实际在数据库字段里保存时，`params_schema`、`before_hooks`、`after_hooks` 是字符串，需要填合法 JSON 文本。

### 22.3 业务范围和项目联动

```json
{
  "fields": [
    {
      "field_code": "scope_id",
      "field_name": "业务范围",
      "input_type": "select",
      "required": true,
      "relation": {
        "table_code": "demo_scope",
        "label_field": "scope_name",
        "value_field": "id",
        "page_size": 50
      }
    },
    {
      "field_code": "project_id",
      "field_name": "项目",
      "input_type": "select",
      "required": true,
      "relation": {
        "table_code": "demo_project",
        "label_field": "project_name",
        "value_field": "id",
        "page_size": 50,
        "filter_mapping": {
          "scope_id": "scope_id"
        }
      }
    }
  ]
}
```

用户先在参数弹框里选择业务范围，项目下拉会带着业务范围 ID 查询项目表。

## 23. 配置入口和操作验收

这一节按实际页面操作顺序写，适合边配边对照。

### 23.1 从表到页面的配置入口

| 要做的事 | 去哪里配 | 操作要点 |
| --- | --- | --- |
| 初始化字段元数据 | 开发管理 / 数据管理 | 选择表后执行初始化元数据或同步字段 |
| 调整字段显示和输入控件 | 开发管理 / 数据管理 / 字段管理 | 修改字段名称、输入类型、字典、联动、表单宽度、详情宽度 |
| 发布成左侧菜单 | 开发管理 / 数据管理 / 发布 | 选择父级菜单，发布后会创建或恢复菜单 |
| 配置页面按钮 | 系统管理 / 菜单管理 | 选中菜单后，在按钮管理里新增或编辑按钮 |
| 分配菜单和按钮权限 | 系统管理 / 角色管理 / 分配权限 | 勾选菜单，再勾选右侧按钮权限 |
| 让菜单立即生效 | 刷新菜单缓存或重新登录 | 菜单、按钮、权限变更后都建议刷新 |

低代码页面不需要手写前端路由。只要表已经发布成菜单，前端通用页面会根据菜单绑定的表编码渲染列表、新增、编辑、详情和查询。

### 23.2 菜单按钮怎么配

按钮分两类：

| 类型 | 典型动作 | 说明 |
| --- | --- | --- |
| 页面能力按钮 | `query`、`create`、`update`、`delete`、`detail`、`refresh` | 由通用页面识别并执行，例如打开新增弹框、打开详情页 |
| 业务动作按钮 | `custom` | 调后端业务接口，例如标记完成、审核、修改业务范围、调整地点 |

配置按钮时优先看这几个字段：

| 字段 | 怎么填 |
| --- | --- |
| 按钮名称 | 页面显示的中文，例如“详情”“审核” |
| 按钮编码 | 保证唯一即可，不要依赖编码推断动作 |
| 按钮位置 | 表格顶部、行按钮、详情顶部、详情底部等 |
| 事件动作 | 必须从动作字典选择，例如 `detail` 或 `custom` |
| 接口路径 | 只有需要调后端接口的按钮才填 |
| 请求方法 | 和后端接口一致 |
| 参数 Schema | 点击按钮需要补充参数时填 |

例如详情按钮只需要配置 `event_action=detail`，不需要填写详情 URL。前端会自动取当前行 ID 打开通用详情。

### 23.3 接口权限怎么配

菜单权限、按钮权限和接口权限不是一回事：

| 权限 | 控制什么 |
| --- | --- |
| 菜单权限 | 左侧菜单和页签是否能进入 |
| 按钮权限 | 页面上是否显示某个按钮 |
| 接口权限 | 点击按钮后对应后端接口是否能访问 |

普通低代码 CRUD 的接口权限由页面能力和按钮权限共同决定。自定义业务接口必须在按钮里填写 `api_path` 和 `http_method`，然后在角色分配里把这个按钮授权给角色。

常见规则：

1. 能看到菜单但列表 403，检查查询按钮权限。
2. 能看到详情按钮但点详情 403，检查详情按钮权限和当前菜单绑定表是否正确。
3. 关联下拉 403，检查目标表的查询权限。
4. 自定义按钮 403，检查按钮里的 `api_path`、`http_method` 和角色授权。

### 23.4 新建数据怎么验收

发布和授权完成后，用下面顺序验收：

1. 打开低代码菜单，列表能正常加载。
2. 点击新增，弹框字段应按字段元数据渲染。
3. 字典字段应该显示下拉选项。
4. 关联字段应该能搜索和选择目标表数据。
5. 文件字段应该显示上传控件，不应该是普通文本框。
6. 富文本字段应该显示富文本编辑器，详情页应渲染内容而不是直接显示 HTML。
7. 保存成功后列表刷新，能看到新记录。
8. 点击编辑，应该按当前行 ID 读取详情后回填表单。
9. 点击详情，应该打开当前行详情页。
10. 删除或业务按钮操作后，列表应按后置钩子刷新。

如果新增按钮不显示，先检查角色是否有 `create` 按钮权限。如果保存时报 403，检查新增按钮和新增接口权限。如果保存成功但列表看不到，检查查询条件、列表刷新和字段是否在列表显示。

### 23.5 发布、取消发布、再次发布怎么验收

| 操作 | 正常结果 |
| --- | --- |
| 发布 | 左侧菜单出现对应页面，角色授权后可进入 |
| 取消发布 | 菜单禁用，数据表和字段元数据不删除 |
| 再次发布 | 优先恢复原菜单，不应重复创建同编码菜单 |
| 删除菜单 | 菜单不可访问，重新发布时应按表重新建立菜单关系 |

发布状态以菜单状态为准。不要为了修复发布状态直接改前端路由，应该从数据管理或菜单管理里处理菜单数据。
