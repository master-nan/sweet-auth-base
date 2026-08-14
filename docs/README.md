# Sweet Platform 文档导航

本目录只服务四类长期需求：用户和管理员使用、工程架构与扩展、部署运行与排错，以及文档导航。当前仍处于治理过渡期，施工设计、验收、冻结和审计证据集中在 `_construction/`，待核心结论吸收后由 DOC-FINAL 清理。

## 目录

| 目录 | 读者 | 用途 | 生命周期 |
| --- | --- | --- | --- |
| [`user-guide/`](user-guide/README.md) | 用户、业务管理员、平台管理员 | 页面操作、配置方法和产品使用说明 | 长期 |
| [`engineering/`](engineering/README.md) | 开发、架构、测试人员 | 架构边界、开发规范和扩展方式 | 长期；现有内容仍需后续重写 |
| [`operations/`](operations/README.md) | 部署、运维、开发人员 | 环境、配置、部署、运行和排错 | 长期 |
| [`_construction/`](_construction/README.md) | 建设与评审人员 | Design、Standard、Acceptance、Freeze、Audit 和生产 Gate 证据 | 临时；不是产品文档 |
| `development/` | 本地开发人员 | Task 记录、原始分析和敏感源资料 | Git 忽略；稳定后整体删除 |

新文档落位规则见 [DocumentationStandard.md](DocumentationStandard.md)，当前资产及最终动作见 [DocumentationInventory.md](_construction/DocumentationInventory.md)。不得默认把新文档写入 `docs/` 根目录。

## 当前模块导航

### 使用与配置

- [平台管理员使用手册](user-guide/PlatformAdministrationGuide.md)
- [低代码配置手册](user-guide/LowCodeManual.md)
- [字段类型指南](user-guide/FieldTypeGuide.md)
- [字段联动配置](user-guide/LinkageConfig.md)
- [数据权限管理员手册](user-guide/DataPermissionUserGuide.md)
- [组织管理使用说明](user-guide/OrganizationManagementUserGuide.md)

### 工程与运行

- [Sweet Platform 工程手册 V1](engineering/sweet-platform-engineering-handbook-v1.md)
- [运行手册](operations/Runbook.md)

### 建设期参考资料

Data Permission、Integration Runtime/Retry/Sync、Organization HR 和 Report 的阶段设计、验收与冻结资料位于 `_construction/`。其中 Organization HR 的真实生产源 Gate 尚未关闭，相关资料不得提前删除，也不得据此启用生产 Consumer。

## 清理原则

- Acceptance、Freeze、Audit、阶段 Review 和实现 Evidence 在核心结论吸收后删除。
- 施工 Design 逐份判断为合并、重写或在活跃实施结束后删除，不机械清理。
- Standard 的长期规则分别吸收进 Engineering Development Guide、Backend Architecture 或 Testing Guide，原文件在 DOC-FINAL 删除。
- `_construction/analysis/organization-source/` 是本地 production-enablement evidence，不是最终产品文档；原始 Swagger 和响应继续留在 ignored `development/raw`。
