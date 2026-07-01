# 字段联动配置

字段联动用于把表单字段配置成关联下拉或级联选择，例如项目、客户、菜单、组织、区域这类来自其他表的数据。

日常配置优先使用字段管理界面：选择关联表、显示字段、值字段、分页大小和联动筛选即可。`sys_table_field.linkage_config` 是后端保存这些配置的 JSON 字段，主要用于批量初始化、排障、数据修复和高级场景参考，不建议日常手写。

## 1. 推荐配置方式

在字段管理里配置关联字段时，按下面顺序确认：

1. 字段本身保存什么值，通常保存关联表的 `id`。
2. 数据来源是普通关联下拉还是树形级联。
3. 关联表优先使用 `tableCode`，避免环境间表 ID 变化。
4. `labelKey` 选择用户可读字段，例如 `name`、`title`、`project_name`。
5. `valueKey` 通常选择 `id`。
6. 如果需要父子联动，配置筛选映射：左边是关联表字段，右边是当前表单字段。

按钮参数表单里的关联下拉优先使用 `relation` 或 `cascader` 简写，写法见 [low-code-manual.md](low-code-manual.md)。前端会把简写转换成同一套联动配置。

## 2. JSON 存储格式参考

普通关联下拉的存储结构如下：

```json
{
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableCode": "demo_project",
    "labelKey": "title",
    "valueKey": "id",
    "filterMapping": {}
  }
}
```

字段管理界面生成的 JSON 可能包含更多字段；排障时只需要确认 `enabled`、`mode`、`tableCode`、`labelKey`、`valueKey` 和 `filterMapping` 是否正确。

## 3. 字段说明

### 3.1 通用字段

| 字段            | 类型    | 必填 | 默认值                        | 说明                                                           |
| --------------- | ------- | :--: | ----------------------------- | -------------------------------------------------------------- |
| `enabled`       | boolean | 是 | - | 是否启用联动，为 `false` 时整个配置不生效 |
| `mode`          | string  | 是 | - | 联动模式：`relation` 普通下拉选择，`cascader` 级联树形选择 |
| `tableCode`     | string  | 是 | - | 关联表编码；可视化配置默认保存这个字段 |
| `labelKey`      | string  | 否 | `label` | 关联表中作为显示文本的字段名，自动回退 `name`、`title`、`code` |
| `valueKey`      | string  | 否 | `value` | 关联表中作为存储值的字段名，自动回退 `id` |
| `pageSize`      | number  | 否 | 200 | 查询关联表时的分页大小；relation 运行时会限制在 20-200，cascader 建议不要超过 1000 |
| `filterMapping` | object  | 否 | `{}` | 联动过滤映射，格式为 `{ "关联表字段": "当前表单字段" }` |

### 3.2 cascader 模式额外字段

| 字段          | 类型    | 默认值        | 说明                                                           |
| ------------- | ------- | ------------- | -------------------------------------------------------------- |
| `parentKey`   | string  | `"parent_id"` | 父节点字段名，用于构建树                                       |
| `childrenKey` | string  | `"children"`  | 子节点数组的属性名                                             |
| `rootValue`   | number  | `0`           | 根节点的 parentKey 值（tree 根从此值开始构建）                 |
| `selectable`  | string  | `"any"`       | 可选模式：`"any"` 任意级可选，`"leaf"` 只能选末级节点          |
| `showPath`    | boolean | `true`        | 是否在输入框中显示完整路径（如 `父 / 子`），`false` 只显示末级 |

## 4. 工作原理

### 4.1 输入类型自动推断

只要字段的 `linkage_config` 中 `enabled` 为 `true`：

- `mode = "relation"` → 自动渲染为 `<q-select>` 下拉选择框
- `mode = "cascader"` → 自动渲染为 `<cascader-select>` 级联选择器

> 即使字段的 `input_type` 设置为 `NUMBER` 或其他类型，联动配置也会覆盖渲染类型。

### 4.2 数据加载

对话框打开时自动执行：

1. 解析 `linkage_config` JSON
2. 根据 `tableCode` 调用后端接口查询关联表数据
3. 用 `labelKey` / `valueKey` 将返回数据映射为 `{ label, value }` 格式
4. cascader 模式额外按 `parentKey` 构建树形结构
5. 将选项挂载到字段的 `options` 属性上

**API 端点**：

- 按编码：`POST /admin/generalization/query/code/{tableCode}`

新配置统一使用 `tableCode`。前端下拉、级联、列表回显和参数 Schema 运行时都依赖 `tableCode`，不要再手写环境相关的表 ID。

relation 模式会按需加载：

- 打开下拉时加载第一页
- 输入关键字时通过 `quick_query.keyword` 远程筛选
- 滚动到底时继续加载下一页
- 编辑已有记录时补查当前已选值，保证 ID 能回显成名称

### 4.3 表单值变化时自动刷新

当表单数据发生变化时，会重新触发联动选项加载，确保依赖字段（通过 `filterMapping`）能根据父字段的值动态更新。

## 5. JSON 示例

下面示例用于理解最终保存结构或批量初始化。界面能配置的场景，优先通过界面维护。

### 5.1 普通关联下拉

场景：字段 `menu_id` 关联到菜单表，显示菜单标题，存储菜单 ID。

```json
{
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableCode": "sys_menu",
    "labelKey": "title",
    "valueKey": "id",
    "filterMapping": {}
  }
}
```

**效果**：表单中 `menu_id` 渲染为下拉框，选项来自菜单表，显示 `title`，存储 `id`。

### 5.2 关联下拉联动过滤

场景：`project_id` 字段需要根据已选的 `scope_id` 过滤，只显示当前业务范围下的项目。

```json
{
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableCode": "demo_project",
    "labelKey": "project_name",
    "valueKey": "id",
    "pageSize": 50,
    "filterMapping": {
      "scope_id": "scope_id"
    }
  }
}
```

**`filterMapping` 规则**：`{ "关联表字段": "当前表单字段" }`

- 键 `"scope_id"` → 关联表（项目表）中的过滤字段
- 值 `"scope_id"` → 当前表单中的字段名，取其当前值

当用户选择了 `scope_id = 100` 后，查询项目表时会自动附加 `scope_id = 100` 的过滤条件。

如果两个表字段同名，左右两边也都写同一个字段名：

```json
{
  "filterMapping": {
    "scope_id": "scope_id"
  }
}
```

### 5.3 级联树形选择

场景：`parent_id` 字段选择上级菜单，数据为树形结构。

```json
{
  "linkage": {
    "enabled": true,
    "mode": "cascader",
    "tableCode": "sys_menu",
    "labelKey": "title",
    "valueKey": "id",
    "parentKey": "parent_id",
    "rootValue": 0,
    "pageSize": 1000
  }
}
```

**效果**：查出所有菜单的扁平数据后，按 `parent_id` 字段组装成树形结构，渲染为级联选择器。`parent_id = 0` 的记录为根节点。

## 注意事项

1. **配置入口**：常规维护优先使用字段管理界面，JSON 只作为高级参考和排障入口
2. **字段类型**：关联字段通常为 `BIGINT` 类型（存储关联表的 ID）
3. **选项回退**：`labelKey` 找不到时会依次尝试 `label` → `name` → `title` → `code`；`valueKey` 找不到时回退到 `id`
4. **分页限制**：字段管理编辑器默认 `pageSize=200`；relation 按页加载并在运行时限制为 20-200，cascader 通常一次加载树形数据，建议控制在 1000 条以内
5. **联动刷新**：使用 `filterMapping` 时，父字段值变化会自动触发子字段选项重新加载
6. **更多示例**：按钮参数 Schema、静态选项、字典选项、关联表下拉的完整写法见 [low-code-manual.md](low-code-manual.md)
