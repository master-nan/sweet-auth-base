# Sweet Platform 文档资产清单

> Audience: 文档治理、架构和项目维护人员
> Lifecycle: construction
> Final Action: DELETE_AFTER_STABLE
> Removal Gate: DOC-FINAL 完成长期文档吸收、引用复核和生产 Gate 迁移后删除

## 1. 盘点口径

DOC-001A 治理前共有 67 份非 `development` 文档资产：35 份 Git 已跟踪根目录 Markdown，以及 32 份因旧 `docs/*/` 规则而被忽略的工程、报表和脱敏分析资产。另有 `docs/development/` 下 162 份有效本地资产（不计 3 个 `.DS_Store`），保持 Git 忽略并按目录治理。

本表的目标动作只允许：`KEEP`、`MERGE`、`REWRITE`、`DELETE_AFTER_STABLE`、`DELETE_AFTER_PRODUCTION_GATE`、`IGNORED_RAW`。`MERGE` 和 `REWRITE` 均表示原文件在结论吸收后删除，不代表当前内容已经成为长期事实。

## 2. 长期入口与使用资料

| 当前文件 | 当前用途 | 目标长期类型 | 生命周期 | 最终动作 |
| --- | --- | --- | --- | --- |
| `user-guide/DataPermissionUserGuide.md` | 数据权限管理员操作 | User Guide | long-term | KEEP |
| `user-guide/FieldTypeGuide.md` | 字段类型配置说明 | User Guide | long-term | KEEP |
| `user-guide/LinkageConfig.md` | 字段联动配置说明 | User Guide | long-term | KEEP |
| `user-guide/LowCodeManual.md` | 低代码平台使用手册 | User Guide | long-term | KEEP |
| `user-guide/OrganizationManagementUserGuide.md` | 组织管理操作说明 | User Guide | long-term | KEEP |
| `operations/Runbook.md` | 本地运行、构建和排错 | Operations Guide | active-maintenance | REWRITE |
| `engineering/sweet-platform-engineering-handbook-v1.md` | 当前工程总览 | Engineering Guide | active-maintenance | REWRITE |

## 3. Design 与 Architecture

| 当前文件 | 当前用途 | 目标长期类型 | 生命周期 | 最终动作 |
| --- | --- | --- | --- | --- |
| `_construction/design/AuthenticationArchitectureDesign.md` | 认证架构设计 | Backend Architecture | construction | REWRITE |
| `_construction/design/DataPermissionDesign.md` | 数据权限详细设计 | Data Permission Engineering Guide | construction | MERGE |
| `_construction/design/DataPermissionOwnershipDesign.md` | Ownership 与权限边界 | Data Permission Engineering Guide | construction | MERGE |
| `_construction/design/IntegrationFoundationDesign.md` | Integration 基础架构 | Integration Engineering Guide | construction | MERGE |
| `_construction/design/IntegrationConfigurationDesign.md` | Integration 配置模型 | Integration Engineering Guide | construction | MERGE |
| `_construction/design/IntegrationRuntimeDesign.md` | Runtime 执行链设计 | Integration Engineering Guide | construction | MERGE |
| `_construction/design/IntegrationRetryDesign.md` | Retry 设计 | Integration Engineering Guide | construction | MERGE |
| `_construction/design/IntegrationSyncDesign.md` | Sync 设计 | Integration Engineering Guide | construction | MERGE |
| `_construction/design/OrganizationHRSyncDesign.md` | HR Adapter 设计与生产 Gate | Organization Integration Guide | production-enablement-evidence | REWRITE |

## 4. Acceptance、Freeze、Audit 与 Review

| 当前文件 | 当前用途 | 目标长期类型 | 生命周期 | 最终动作 |
| --- | --- | --- | --- | --- |
| `_construction/reviews/DataPermissionAcceptanceGuide.md` | 数据权限阶段验收步骤 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/DataPermissionAcceptanceReport.md` | 数据权限验收结果 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/DataPermissionFreezeReview.md` | 数据权限冻结依据 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationConfigurationAcceptanceReport.md` | Integration 配置验收 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationRuntimeAcceptanceReport.md` | Runtime 验收结果 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationRuntimeFreezeReview.md` | Runtime 冻结依据 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationRetryAcceptanceReport.md` | Retry 验收结果 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationRetryFreezeReview.md` | Retry 冻结依据 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationSyncAcceptanceReport.md` | Sync 验收结果 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationSyncFreezeReview.md` | Sync 冻结依据 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/PlatformBackendCodeAuditReport.md` | 阶段后端代码审计 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/PlatformStabilizationReview.md` | 平台稳定化评审 | Construction Evidence | construction | DELETE_AFTER_STABLE |
| `_construction/reviews/IntegrationSyncSourceContractChangeReview.md` | 单下界源契约变更评审 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/reviews/OrganizationHRAssignmentContractReview.md` | 任职/离职源契约评审 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/reviews/OrganizationHRSyncAcceptanceReport.md` | HR Adapter 能力验收 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/reviews/OrganizationHRSyncFreezeReview.md` | HR Adapter 能力冻结与 Gate | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |

## 5. Standard

| 当前文件 | 当前用途 | 目标长期类型 | 生命周期 | 最终动作 |
| --- | --- | --- | --- | --- |
| `_construction/standards/DocumentationNamingStandard.md` | 旧文档命名约定 | Documentation Governance | superseded-construction | DELETE_AFTER_STABLE |
| `_construction/standards/ErrorHandlingStandard.md` | 错误处理规范 | Backend Architecture | construction | MERGE |
| `_construction/standards/TransactionUsageStandard.md` | 事务使用规范 | Engineering Development Guide | construction | MERGE |
| `_construction/standards/TestInfrastructureStandard.md` | 测试基础设施规范 | Testing Guide | construction | MERGE |

## 6. Report 活跃实施资料

| 当前文件 | 当前用途 | 目标长期类型 | 生命周期 | 最终动作 |
| --- | --- | --- | --- | --- |
| `_construction/report-designer/01-v1-scope.md` | 报表 V1 范围 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/02-v1-architecture-decisions.md` | 报表架构决策 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/03-v1-version-runtime-design.md` | 发布版本与运行态设计 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/04-v1-security-design.md` | 报表安全设计 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/05-v1-api-design.md` | 报表 API 设计 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/06-v1-frontend-adjustment.md` | 前端增量调整 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/07-v1-acceptance-checklist.md` | 阶段验收清单 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-designer/09-v1a-implementation-evidence.md` | V1A 实施证据 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-designer/11-v1b-export-plan.md` | 导出实施计划 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/12-v1b-implementation-evidence.md` | V1B 实施证据 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-designer/13-v1d-frontend-plan.md` | 前端实施计划 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-designer/14-v1d-ui-spec.md` | 报表 UI 规格 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/15-v1d-frontend-implementation-evidence.md` | 前端实施证据 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-designer/17-v1e-ui-redesign-spec.md` | UI 重设计规格 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-designer/18-v1e1-runtime-refactor-evidence.md` | Runtime 重构证据 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-v2/01-product-positioning.md` | 报表产品定位 | Report Product/Engineering Guide | active-implementation | REWRITE |
| `_construction/report-v2/02-information-architecture.md` | 报表信息架构 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-v2/03-domain-model.md` | 报表领域模型 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-v2/04-ui-wireframe-spec.md` | 报表线框规格 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-v2/05-api-design.md` | 报表 API 设计 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-v2/06-implementation-roadmap.md` | 报表实施路线图 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |
| `_construction/report-v2/07-v1-compatibility-and-migration.md` | V1 兼容与迁移 | Report Engineering Guide | active-implementation | MERGE |
| `_construction/report-v2/08-naming-and-promotion-strategy.md` | 命名与推广策略 | Construction Evidence | active-implementation | DELETE_AFTER_STABLE |

## 7. Planning 与 production-enablement evidence

| 当前文件 | 当前用途 | 目标长期类型 | 生命周期 | 最终动作 |
| --- | --- | --- | --- | --- |
| `_construction/planning/platform-capability-backlog-v1.md` | 平台能力阶段 Backlog | Construction Planning | active-implementation | DELETE_AFTER_STABLE |
| `_construction/analysis/organization-source/OrganizationSourceApiInventory.md` | HR 源接口清单 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/analysis/organization-source/OrganizationSourceFieldDictionary.csv` | 脱敏字段字典 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/analysis/organization-source/OrganizationSanitizedSamples.json` | 脱敏样本 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/analysis/organization-source/OrganizationSourceDataAnalysis.md` | HR 源数据分析 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/analysis/organization-source/OrganizationSourceMappingDraft.md` | HR 映射草案 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/analysis/organization-source/OrganizationSourceDataQualityReport.md` | HR 数据质量报告 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |
| `_construction/analysis/organization-source/OrganizationSourceOpenQuestions.md` | HR 生产准入待确认项 | Production Gate Evidence | production-enablement-evidence | DELETE_AFTER_PRODUCTION_GATE |

上述 7 份脱敏分析继续由 `.gitignore` 精确排除。本轮仅迁移本地位置并登记，不把未经独立隐私复核的源分析资料纳入 Git。

## 8. development 本地资产

| 当前范围 | 数量 | 当前用途 | 生命周期 | 最终动作 |
| --- | ---: | --- | --- | --- |
| `development/task/`、`bugfix/`、`design/`、`review/`、`sprint-*`、`test-data/` | 93 | Task、评审、演示数据和阶段草稿 | local-construction | DELETE_AFTER_STABLE |
| `development/analysis/organization-source/raw/` | 69 | 原始 OpenAPI、Header、元数据和响应 | local-sensitive-raw | IGNORED_RAW |

`development` 资产按目录而非逐文件转正：它们从未属于正式产品文档，继续全部 Git 忽略。DOC-FINAL 在确认没有活跃引用后删除非原始过程资料；原始敏感资料按安全与生产准入流程单独销毁，不进入 Git 历史。

## 9. 最终清理摘要

- 长期保留或重写：5 份 User Guide、1 份 Operations Runbook、1 份 Engineering Handbook。
- 合并/重写后删除原件：9 份架构设计、3 份 Standard、14 份 Report 核心设计。
- 稳定后删除：12 份非 HR Review、1 份旧命名规范、9 份 Report 过程资料、1 份规划 Backlog，以及 93 份本地过程资料。
- Production Gate 后删除：4 份 HR/Source Contract Review 与 7 份脱敏分析资料。
- 始终不进入 Git：69 份原始 HR 资料。

DOC-001A 不执行上述最终删除；所有动作留给对应 Engineering Documentation 重写、HR Production Enablement 和 DOC-FINAL。
