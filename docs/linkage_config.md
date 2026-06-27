# 字段联动配置（linkage_config）

`linkage_config` 是 `sys_table_field` 表上的一个 JSON 字段，用于将表单字段与其他数据表关联，使其自动渲染为**下拉选择框**或**级联选择器**，并从关联表动态加载选项数据。

---

## 基本结构

```json
{
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableId": 101724747440640,
    "labelKey": "title",
    "valueKey": "id",
    "filterMapping": {}
  }
}
```

---

## 字段说明

### 通用字段

| 字段            | 类型    | 必填 | 默认值                        | 说明                                                           |
| --------------- | ------- | :--: | ----------------------------- | -------------------------------------------------------------- |
| `enabled`       | boolean |  ✅  | —                             | 是否启用联动，为 `false` 时整个配置不生效                      |
| `mode`          | string  |  ✅  | —                             | 联动模式：`"relation"` 普通下拉选择，`"cascader"` 级联树形选择 |
| `tableId`       | number  |  ⚠️  | —                             | 关联表的 ID（与 `tableCode` 二选一）                           |
| `tableCode`     | string  |  ⚠️  | —                             | 关联表的编码（与 `tableId` 二选一）                            |
| `labelKey`      | string  |  ❌  | `"label"`                     | 关联表中作为**显示文本**的字段名，自动回退 `name` → `title`    |
| `valueKey`      | string  |  ❌  | `"value"`                     | 关联表中作为**存储值**的字段名，自动回退 `id`                  |
| `pageSize`      | number  |  ❌  | relation: 50, cascader: 1000  | 查询关联表时的分页大小；relation 支持远程搜索和滚动加载        |
| `filterMapping` | object  |  ❌  | `{}`                          | 级联过滤映射，详见下文                                         |

### cascader 模式额外字段

| 字段          | 类型    | 默认值        | 说明                                                           |
| ------------- | ------- | ------------- | -------------------------------------------------------------- |
| `parentKey`   | string  | `"parent_id"` | 父节点字段名，用于构建树                                       |
| `childrenKey` | string  | `"children"`  | 子节点数组的属性名                                             |
| `rootValue`   | number  | `0`           | 根节点的 parentKey 值（tree 根从此值开始构建）                 |
| `selectable`  | string  | `"any"`       | 可选模式：`"any"` 任意级可选，`"leaf"` 只能选末级节点          |
| `showPath`    | boolean | `true`        | 是否在输入框中显示完整路径（如 `父 / 子`），`false` 只显示末级 |

---

## 工作原理

### 1. 输入类型自动推断

只要字段的 `linkage_config` 中 `enabled` 为 `true`：

- `mode = "relation"` → 自动渲染为 `<q-select>` 下拉选择框
- `mode = "cascader"` → 自动渲染为 `<cascader-select>` 级联选择器

> 即使字段的 `input_type` 设置为 `NUMBER` 或其他类型，联动配置也会覆盖渲染类型。

### 2. 数据加载

对话框打开时自动执行：

1. 解析 `linkage_config` JSON
2. 根据 `tableId` 或 `tableCode` 调用后端接口查询关联表数据
3. 用 `labelKey` / `valueKey` 将返回数据映射为 `{ label, value }` 格式
4. cascader 模式额外按 `parentKey` 构建树形结构
5. 将选项挂载到字段的 `options` 属性上

**API 端点**：

- 按 ID：`POST /admin/generalization/query/{tableId}`
- 按编码：`POST /admin/generalization/query/code/{tableCode}`

relation 模式会按需加载：

- 打开下拉时加载第一页
- 输入关键字时通过 `quick_query.keyword` 远程筛选
- 滚动到底时继续加载下一页
- 编辑已有记录时补查当前已选值，保证 ID 能回显成名称

### 3. 表单值变化时自动刷新

当表单数据发生变化时，会重新触发联动选项加载，确保依赖字段（通过 `filterMapping`）能根据父字段的值动态更新。

---

## 配置示例

### 示例 1：普通关联（下拉选择）

场景：字段 `menu_id` 关联到菜单表，显示菜单标题，存储菜单 ID。

```json
{
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableId": 101724747440640,
    "labelKey": "title",
    "valueKey": "id",
    "filterMapping": {}
  }
}
```

**效果**：表单中 `menu_id` 渲染为下拉框，选项来自菜单表，显示 `title`，存储 `id`。

---

### 示例 2：关联下拉联动过滤

场景：`vehicle_id` 字段需要根据已选的 `carrier_id` 过滤，只显示当前承运商下的车辆。

```json
{
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableCode": "base_vehicle",
    "labelKey": "plate_no",
    "valueKey": "id",
    "pageSize": 50,
    "filterMapping": {
      "carrier_id": "carrier_id"
    }
  }
}
```

**`filterMapping` 规则**：`{ "关联表字段": "当前表单字段" }`

- 键 `"carrier_id"` → 关联表（车辆表）中的过滤字段
- 值 `"carrier_id"` → 当前表单中的字段名，取其当前值

当用户选择了 `carrier_id = 100` 后，查询车辆表时会自动附加 `carrier_id = 100` 的过滤条件。

如果两个表字段同名，可以这样写：

```json
{
  "filterMapping": {
    "carrier_id": "carrier_id"
  }
}
```

---

### 示例 3：级联树形选择

场景：`parent_id` 字段选择上级菜单，数据为树形结构。

```json
{
  "linkage": {
    "enabled": true,
    "mode": "cascader",
    "tableId": 300,
    "labelKey": "title",
    "valueKey": "id",
    "parentKey": "parent_id",
    "rootValue": 0,
    "pageSize": 1000
  }
}
```

**效果**：查出所有菜单的扁平数据后，按 `parent_id` 字段组装成树形结构，渲染为级联选择器。`parent_id = 0` 的记录为根节点。

---

### 示例 4：使用 tableCode 替代 tableId

当不确定目标表 ID 或 ID 可能变化时，可使用 `tableCode`：

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

> `tableId` 和 `tableCode` 二选一。同时存在时优先使用 `tableId`。

---

## 注意事项

1. **存储格式**：`linkage_config` 存储为 JSON 字符串，确保 JSON 格式正确
2. **字段类型**：关联字段通常为 `BIGINT` 类型（存储关联表的 ID）
3. **选项回退**：`labelKey` 找不到时会依次尝试 `label` → `name` → `title` → `code`；`valueKey` 找不到时回退到 `id`
4. **分页限制**：relation 默认按页加载，建议 `pageSize` 设为 50；cascader 通常一次加载树形数据，建议控制在 1000 条以内
5. **联动刷新**：使用 `filterMapping` 时，父字段值变化会自动触发子字段选项重新加载
6. **更多示例**：按钮参数 Schema、静态选项、字典选项、关联表下拉的完整写法见 [LOW_CODE_MANUAL.md](LOW_CODE_MANUAL.md)
