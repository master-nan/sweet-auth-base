# Sweet Platform 建设期资料清单

> Audience: 文档治理、架构和项目维护人员
> Lifecycle: construction
> Final Action: DELETE_AFTER_STABLE

## 1. RC-001 清理结论

Platform Enablement 完成后，RC-001 对建设期资料执行了第一次真实删除：

- 删除 23 份已被长期手册吸收或已失去独立价值的 tracked construction 文档；
- 删除本地 `docs/development/` 中 103 份 Task、评审、草稿和演示资料；
- 保留 34 份 tracked construction 文档；
- 保留 7 份 ignored HR 脱敏分析和 69 份 ignored 原始 HR 资料；
- Report 仅删除 6 份阶段验收/实施 Evidence，保留当前运行设计和未来重设计参考。

长期规则现在只以 `docs/user-guide/`、`docs/engineering/`、`docs/operations/` 和 `docs/DocumentationStandard.md` 为准。Git 历史承担已删除施工过程的追溯职责。

## 2. 长期文档

| 文件 | 用途 | 生命周期 | 最终动作 |
| --- | --- | --- | --- |
| `user-guide/PlatformAdministrationGuide.md` | 平台管理员总手册 | long-term | KEEP |
| `engineering/PlatformEngineeringGuide.md` | 工程架构与目录边界 | long-term | KEEP |
| `engineering/ExtensionDevelopmentGuide.md` | 扩展开发实施步骤 | long-term | KEEP |
| `operations/PlatformOperationsGuide.md` | 部署、运行、发布和排错唯一真值 | long-term | KEEP |

其他专题 User Guide 继续由 `docs/user-guide/README.md` 管理。

## 3. HR 与 Integration Sync 保护区

以下资料继续承担 HR Production Enablement 证据，不得在 DOC-FINAL 前机械删除：

| 当前范围 | 数量 | 保留原因 | 最终动作 |
| --- | ---: | --- | --- |
| `_construction/design/IntegrationRuntimeDesign.md` | 1 | HR Sync 前置 Runtime 契约 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/design/IntegrationRetryDesign.md` | 1 | HR Sync 前置 Retry 契约 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/design/IntegrationSyncDesign.md` | 1 | lower-bound-only 与 Checkpoint 依据 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/design/OrganizationHRSyncDesign.md` | 1 | HR Adapter 与生产 Gate 主设计 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/reviews/Integration*AcceptanceReport.md` / `*FreezeReview.md` | 6 | Runtime、Retry、Sync 冻结证据链 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/reviews/IntegrationSyncSourceContractChangeReview.md` | 1 | 单下界源契约变更证据 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/reviews/OrganizationHRAssignmentContractReview.md` | 1 | 任职/离职源契约证据 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/reviews/OrganizationHRSyncAcceptanceReport.md` / `FreezeReview.md` | 2 | HR Adapter 能力与 Gate 证据 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `_construction/analysis/organization-source/` | 7 ignored | 脱敏生产准入分析 | DELETE_AFTER_HR_PRODUCTION_GATE |
| `development/analysis/organization-source/raw/` | 69 ignored | 原始 OpenAPI、Header、元数据和响应 | 受控销毁；永不进入 Git |

真实 HR Production Gate 关闭后，先将仍需长期保留的操作结论吸收到 Engineering/Operations，再删除本表所列施工证据。

## 4. Report 保护区

Report 当前运行能力和未来重设计仍需要以下资料：

| 当前范围 | 数量 | 分类 | 最终动作 |
| --- | ---: | --- | --- |
| `_construction/report-designer/` | 9 | 当前 V1 运行设计、API、安全、导出和 UI 参考 | MERGE_AFTER_REPORT_REDESIGN |
| `_construction/report-v2/` | 8 | 产品定位、信息架构、领域模型、原型、API、路线、兼容和命名参考 | MERGE_AFTER_REPORT_REDESIGN |

RC-001 已删除 report-designer 下 6 份阶段 Checklist、Plan 和 Implementation Evidence。`report-v2` 当前全部归类为未来专题参考；隐藏 prototype 有真实路由且属于保护范围，不按死代码删除。

## 5. 已删除 construction 分类

| 分类 | 数量 | 删除依据 |
| --- | ---: | --- |
| Auth/Data Permission/Integration Foundation 普通设计 | 5 | 当前事实已进入 Engineering/Extension Guide |
| Data Permission、Integration Configuration、Platform Audit/Review | 7 | 验收过程已完成，长期结论已吸收 |
| Error/Transaction/Test/Naming Standard | 4 | 已进入 Engineering/Extension/Documentation Standard |
| Platform Capability Backlog | 1 | 状态与路线已过期，后续专项独立规划 |
| Report 阶段 Evidence | 6 | 代码和保留设计已能表达当前/未来事实 |
| **合计** | **23** | Git 历史保留追溯 |

## 6. development 与 analysis

`docs/development/` 继续整体 Git 忽略。RC-001 已删除所有非 HR Task、Bugfix、Design、Review、Sprint 和 test-data，仅保留：

```text
docs/development/analysis/organization-source/raw/
```

`docs/_construction/analysis/organization-source/` 继续精确 Git 忽略，保留脱敏分析。两处资料均不是产品文档，不得被 README 当作普通用户入口。

## 7. DOC-FINAL 条件

DOC-FINAL 前仍需满足：

1. HR Production Enablement 完成，生产 Gate 结论已吸收。
2. Report Platform 重设计完成，当前设计与 V2 参考完成取舍。
3. Frontend Consistency、Query Center 和 Final Code Review 不再引用 construction。
4. `make docs-check`、全仓路径扫描和敏感资料复核通过。
