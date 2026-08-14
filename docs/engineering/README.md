# Engineering 文档

本目录面向开发、架构、测试和后续维护人员，承载基于稳定代码的架构、目录职责、开发约束和扩展指南。

## 当前文档

- [平台工程架构与目录手册](PlatformEngineeringGuide.md)：总体分层、目录职责、模块边界、关键文件、测试架构和新代码落位规则。

平台管理员如何操作系统请阅读[平台管理员使用手册](../user-guide/PlatformAdministrationGuide.md)，部署和排错请阅读[运行手册](../operations/Runbook.md)。

## 维护规则

- Engineering 文档只描述当前稳定代码和长期扩展约束，不记录 Task 过程。
- 新增目录、公共接口或跨模块依赖时，应同步核对并更新工程手册。
- Authentication、Data Permission、Integration、Organization HR 和 Report 的历史设计、验收与冻结资料仍位于 [`_construction/`](../_construction/README.md)，仅作建设期追溯。
- 具体开发扩展步骤将在 PE-003 中补充；不得用 construction 文档替代长期开发指南。
