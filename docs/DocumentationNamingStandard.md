# 正式文档命名规范

## 一、适用范围

本规范适用于 `docs/` 根目录中的正式项目文档。正式文档属于项目交付资产，应提交到 Git，并随项目版本持续维护。

`docs/development/` 用于保存开发任务、设计讨论、评审记录和实施过程材料。这些内容属于开发过程记录，不作为正式交付资产，不应与正式文档混放。

## 二、文件命名

正式 Markdown 文档统一使用：

```text
PascalCase.md
```

命名要求：

- 使用能够准确表达文档用途的英文单词。
- 每个单词首字母大写。
- 单词之间不使用连字符、下划线或空格。
- 文件扩展名统一使用 `.md`。
- 同一主题只保留一份职责明确的正式文档，避免重复内容。

正确示例：

- `OrganizationManagementUserGuide.md`
- `DataPermissionDesign.md`
- `PlatformFoundationProgress.md`

禁止示例：

- `organization-management-user-guide.md`
- `data_permission_design.md`
- `Platform-foundation_Progress.md`

## 三、目录用途

| 目录 | 用途 | 是否作为正式交付资产 |
| --- | --- | --- |
| `docs/` | 用户手册、运维手册、正式设计说明和项目规范 | 是 |
| `docs/development/` | 开发任务、讨论、评审、阶段设计和临时记录 | 否 |

正式文档不得依赖开发过程文档才能理解。需要长期保留的结论，应整理后写入 `docs/` 根目录的正式文档。

## 四、重命名要求

调整正式文档名称时：

1. 使用 `git mv` 保留文件历史。
2. 同步更新正式文档中的相对链接。
3. 不通过复制文件保留旧名称，避免形成重复文档。
4. 确认新文件名符合 PascalCase，且链接目标真实存在。
