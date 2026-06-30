# 字段类型与输入类型配套指南

> 本文档说明 `sys_table_field` 中 `field_type`（字段类型）和 `input_type`（输入类型）的合法搭配关系，
> 以及各输入类型的适用场景和存储格式。

---

## 一、字段类型一览（field_type）

| 枚举值 | 名称              | 数据库类型   | 说明                            |
| ------ | ----------------- | ------------ | ------------------------------- |
| 1      | 大数字 BIGINT     | bigint       | ID、雪花ID、大数值              |
| 2      | 浮点 FLOAT        | float/double | 小数、金额                      |
| 3      | 字符串 VARCHAR    | varchar(N)   | 定长字符串，需指定 field_length |
| 4      | 文本 TEXT         | text         | 大段文本（备注、描述等）        |
| 5      | 布尔 BOOLEAN      | tinyint(1)   | true/false                      |
| 6      | 日期 DATE         | date         | 年-月-日                        |
| 7      | 日期时间 DATETIME | datetime     | 年-月-日 时:分:秒               |
| 8      | 时间 TIME         | time         | 时:分:秒                        |
| 9      | 微型整数 TINYINT  | tinyint      | 小范围枚举值（0~255）           |
| 10     | JSON              | json         | JSON 对象/数组                  |
| 11     | 数字 INT          | int          | 普通整数                        |

---

## 二、输入类型一览（input_type）

| 枚举值 | 名称                        | 表单组件              | 说明                                |
| ------ | --------------------------- | --------------------- | ----------------------------------- |
| 1      | 输入框 INPUT                | q-input               | 普通文本输入                        |
| 2      | 数字输入 INPUT_NUMBER       | q-input type=number   | 数值输入                            |
| 3      | 多行文本 TEXTAREA           | q-input type=textarea | 大段文本                            |
| 4      | 下拉选择 SELECT             | q-select              | 字典或关联表选项                    |
| 5      | 日期选择 DATE_PICKER        | q-date 弹出选择       | 年-月-日                            |
| 6      | 日期时间 DATETIME_PICKER    | q-date + q-time       | 年-月-日 时:分:秒                   |
| 7      | 时间选择 TIME_PICKER        | q-time 弹出选择       | 时:分:秒                            |
| 8      | 年份选择 YEAR_PICKER        | q-date 年视图         | 仅选年份，存为 "2026"               |
| 9      | 年月选择 YEAR_MONTH_PICKER  | q-date 月视图         | 选年月，存为 "2026/02"              |
| 10     | 文件选择 FILE_PICKER        | 文件上传组件          | 上传、预览、下载，保存文件 ID 数组  |
| 11     | 布尔开关 BOOLEAN            | q-toggle              | 开关切换                            |
| 12     | JSON编辑器 JSON_EDITOR      | JsonEditor            | JSON 文本编辑 + 校验                |
| 13     | 数组输入 ARRAY_INPUT        | ArrayInput            | 标签式数组编辑，存为 JSON 数组      |
| 14     | 键值对编辑 KEY_VALUE_EDITOR | KeyValueEditor        | 双列 key-value 编辑，存为 JSON 对象 |
| 15     | 级联选择 CASCADER           | CascaderSelect        | 树形级联选择，需配 linkage_config   |

---

## 三、配套关系矩阵

✅ = 推荐搭配 | ⚠️ = 可用但不推荐 | ❌ = 不兼容

| 输入类型 ↓ \ 字段类型 →         | BIGINT | FLOAT | VARCHAR | TEXT | BOOLEAN | DATE | DATETIME | TIME | TINYINT | JSON | INT |
| ------------------------------- | :----: | :---: | :-----: | :--: | :-----: | :--: | :------: | :--: | :-----: | :--: | :-: |
| **输入框** INPUT                |   ⚠️   |  ⚠️   |   ✅    |  ⚠️  |   ❌    |  ❌  |    ❌    |  ❌  |   ⚠️    |  ❌  | ⚠️  |
| **数字输入** INPUT_NUMBER       |   ✅   |  ✅   |   ❌    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ✅    |  ❌  | ✅  |
| **多行文本** TEXTAREA           |   ❌   |  ❌   |   ⚠️    |  ✅  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ❌  | ❌  |
| **下拉选择** SELECT             |   ✅   |  ❌   |   ✅    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ✅    |  ❌  | ✅  |
| **日期选择** DATE_PICKER        |   ❌   |  ❌   |   ⚠️    |  ❌  |   ❌    |  ✅  |    ❌    |  ❌  |   ❌    |  ❌  | ❌  |
| **日期时间** DATETIME_PICKER    |   ❌   |  ❌   |   ⚠️    |  ❌  |   ❌    |  ❌  |    ✅    |  ❌  |   ❌    |  ❌  | ❌  |
| **时间选择** TIME_PICKER        |   ❌   |  ❌   |   ⚠️    |  ❌  |   ❌    |  ❌  |    ❌    |  ✅  |   ❌    |  ❌  | ❌  |
| **年份选择** YEAR_PICKER        |   ❌   |  ❌   |   ✅    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ❌  | ✅  |
| **年月选择** YEAR_MONTH_PICKER  |   ❌   |  ❌   |   ✅    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ❌  | ❌  |
| **文件选择** FILE_PICKER        |   ❌   |  ❌   |   ✅    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ✅  | ❌  |
| **布尔开关** BOOLEAN            |   ❌   |  ❌   |   ❌    |  ❌  |   ✅    |  ❌  |    ❌    |  ❌  |   ❌    |  ❌  | ❌  |
| **JSON编辑器** JSON_EDITOR      |   ❌   |  ❌   |   ❌    |  ⚠️  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ✅  | ❌  |
| **数组输入** ARRAY_INPUT        |   ❌   |  ❌   |   ❌    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ✅  | ❌  |
| **键值对编辑** KEY_VALUE_EDITOR |   ❌   |  ❌   |   ❌    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ✅  | ❌  |
| **级联选择** CASCADER           |   ✅   |  ❌   |   ✅    |  ❌  |   ❌    |  ❌  |    ❌    |  ❌  |   ❌    |  ❌  | ✅  |

---

## 四、常用配置示例

### 4.1 普通文本字段

```
field_type: VARCHAR (3)
input_type: INPUT (1)
field_length: 128
```

### 4.2 枚举字段（用字典）

```
field_type: TINYINT (9) 或 INT (11)
input_type: SELECT (4)
dict_code: "status"         ← 关联字典编码
```

### 4.3 布尔状态字段

```
field_type: BOOLEAN (5)
input_type: BOOLEAN (11)
dict_code: "status"         ← 可选，列表显示时用字典标签代替"是/否"
```

### 4.4 日期时间字段

```
field_type: DATETIME (7)
input_type: DATETIME_PICKER (6)
```

### 4.5 关联表下拉

```
field_type: BIGINT (1) 或 INT (11)
input_type: SELECT (4)
linkage_config: {
  "linkage": {
    "enabled": true,
    "mode": "relation",
    "tableCode": "sys_role",
    "labelKey": "name",
    "valueKey": "id",
    "pageSize": 50
  }
}
```

数据量较大的关联表会按页远程加载，支持输入关键字筛选和滚动加载。按钮参数 Schema 里的简写方式见 [low-code-manual.md](low-code-manual.md)。

### 4.6 级联选择（树形）

```
field_type: BIGINT (1)
input_type: CASCADER (15)
linkage_config: {
  "linkage": {
    "enabled": true,
    "mode": "cascader",
    "tableCode": "sys_menu",
    "labelKey": "menu_name",
    "valueKey": "id",
    "parentKey": "pid",
    "selectable": "any",
    "showPath": true
  }
}
```

### 4.7 JSON 配置字段

```
field_type: JSON (10)
input_type: JSON_EDITOR (12)    ← 复杂对象
         或 ARRAY_INPUT (13)     ← 简单字符串数组
         或 KEY_VALUE_EDITOR (14) ← 简单键值对
```

### 4.8 年份/年月字段

```
field_type: VARCHAR (3)
input_type: YEAR_PICKER (8)      ← 存储 "2026"
field_length: 4

field_type: VARCHAR (3)
input_type: YEAR_MONTH_PICKER (9) ← 存储 "2026/02"
field_length: 7
```

### 4.9 文件上传字段

```
field_type: VARCHAR (3)          ← 存文件URL
input_type: FILE_PICKER (10)
field_length: 512

或者

field_type: JSON (10)            ← 存多文件信息数组
input_type: FILE_PICKER (10)
```

---

## 五、自动推断规则

当 `input_type` 未设置（为0或null）时，系统会根据 `field_type` 自动推断：

| 字段类型                       | 自动推断为   |
| ------------------------------ | ------------ |
| BIGINT / FLOAT / TINYINT / INT | 数字输入     |
| TEXT                           | 多行文本     |
| JSON                           | JSON编辑器   |
| BOOLEAN                        | 布尔开关     |
| DATE                           | 日期选择     |
| DATETIME                       | 日期时间选择 |
| 其他                           | 输入框       |

---

## 六、特殊说明

1. **dict_code 优先级**：字段配了 `dict_code` 时，列表展示优先用字典标签，匹配不到再 fallback
2. **linkage_config**：`input_type=SELECT` 或 `CASCADER` 时，通过 `linkage_config` JSON 配置关联表，字段类型应为存储关联 ID 的类型（通常 BIGINT 或 INT）
3. **JSON 系列**：`JSON_EDITOR`、`ARRAY_INPUT`、`KEY_VALUE_EDITOR` 三者的字段类型都应该是 **JSON**，不要用 TEXT
4. **字典选择 vs 关联选择**：
   - 固定选项（如状态: 启用/停用）→ 用字典 `dict_code`
   - 动态选项（如角色列表）→ 用 `linkage_config` 关联表
5. **低代码配置手册**：字段联动、按钮参数表单、关系下拉、字典和静态选项的完整写法见 [low-code-manual.md](low-code-manual.md)
