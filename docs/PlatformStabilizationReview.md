# Sweet Platform 平台治理阶段冻结评审

## 文档信息

| 项目 | 内容 |
| --- | --- |
| 文档目的 | 复核第一轮平台治理成果，冻结平台基础规范与核心架构边界，判断是否允许进入 Integration Foundation |
| 评审范围 | ST-001 至 ST-006-B 的正式文档、规范、代码治理与清理结果 |
| 评审日期 | 2026-08-04 |
| 评审基线 | `b014784c60750ba171a1de1b358e31c0ccec6608` |
| 适用范围 | Sweet Platform 平台底座及后续新增模块 |
| 评审结论 | **通过冻结** |

本评审以当前仓库中的实际提交和正式文档为依据。详细代码健康问题及原始整改优先级见 [PlatformBackendCodeAuditReport.md](PlatformBackendCodeAuditReport.md)。

## 1. 治理背景

Sweet Platform 在组织管理、数据权限、低代码、报表和系统管理等能力持续建设后，平台代码规模与模块协作关系明显增长。第一轮平台治理的目的不是压缩代码量或重写已有领域，而是在进入新的基础设施阶段前，建立可持续扩展所需的安全边界和工程基线。

本轮治理重点解决以下问题：

1. 历史代码持续增长后，文件命名、注释、错误处理、事务和测试方式不完全一致。
2. 异步任务复用请求级 Gin Context、文件签名用途未隔离等安全风险需要优先收口。
3. Model 直接作为响应、底层依赖 HTTP 框架等边界问题增加了接口变化和非 HTTP 场景复用成本。
4. 重复测试初始化、零散日志输出、失效文档引用和空实现文件增加了维护与审查成本。
5. Organization、Data Permission 和 Generalization 已形成明确领域边界，需要防止后续治理以“减少代码量”为由破坏这些边界。

治理采用“先审计、再规范、最后小步整改”的方式。存量大文件和历史入口不进行一次性重写；新增代码从冻结日起执行统一准入规则。

## 2. 已完成治理

### 2.1 文档治理

- 正式文档统一使用 `PascalCase.md` 命名，`docs/` 根目录与 `docs/development/` 的用途边界已经明确，见 [DocumentationNamingStandard.md](DocumentationNamingStandard.md)。
- README、Makefile、正式 Markdown 和相关文本配置中的失效文档名称、路径及大小写引用已经清理。
- 正式设计、验收、操作手册与开发过程材料不再混放，正式交付资产的识别边界已经形成。

### 2.2 代码与工程规范

- 后端解释性注释已按中文规范完成第一阶段治理，API、HTTP、JSON、SQL、GORM、Resolver、Adapter、Provider 等技术名词保持原义。
- 已建立正式的 [ErrorHandlingStandard.md](ErrorHandlingStandard.md)，冻结 Repository、Service、Controller/Middleware 三层错误边界，并完成一批明确的错误信息泄露整改。
- 已建立正式的 [TransactionUsageStandard.md](TransactionUsageStandard.md)，明确 Service 定义业务事务边界、Repository 执行数据访问、Controller 不创建事务。
- 已建立正式的 [TestInfrastructureStandard.md](TestInfrastructureStandard.md)，统一测试分类、数据库 Helper、Gin 包级初始化、隔离要求及 CI 可见性要求。
- 生产代码中的 `fmt.Print*` 已清理，结构体映射错误改用带稳定字段的结构化日志。

### 2.3 安全治理

- 异步登录日志、短信日志和状态更新不再捕获或传递请求级 `*gin.Context`，异步任务只接收必要业务元数据和标准 `context.Context`。
- 文件预览与下载签名已绑定明确用途；缺少用途、用途篡改及跨用途访问均按安全失败处理。
- 统一认证目标架构已经冻结，密码、短信和未来 SSO 被定义为 Credential Provider，账号状态、安全策略、Token 与审计由统一编排承担，见 [AuthenticationArchitectureDesign.md](AuthenticationArchitectureDesign.md)。存量认证入口按该设计渐进迁移，不在本轮进行一次性重写。

### 2.4 架构边界治理

- 已建立不依赖 Gin 的 `AuditSubject`，支持通过标准 `context.Context` 注入和读取用户审计主体；HTTP Middleware 已提供标准 Context 注入能力。
- File、Report、SysTable/SysTableField、SysMenu、SysRole 等高风险接口已建立 Response DTO 白名单边界，列表和详情响应不再依赖 Model 字段自然扩张。
- Model 和 Repository 的存量 Gin 依赖仍保留兼容入口，但新增代码已有标准 Context 的优先路径。

### 2.5 低风险清理

- 修正 `sys_role_cache.go`、`snowflake.go` 文件名，保留完整 Git 历史。
- 删除不承担声明、接口实现或 Wire 绑定的空实现文件。
- 清理审计报告中的失效文件引用。
- 对疑似死代码完成路由、Swagger、Service、Wire、测试、文档和前端调用审计；存在外部契约或领域接口边界的对象继续保留，没有仅凭文本搜索结果删除代码。

## 3. 当前冻结边界

以下能力是平台稳定底座的一部分，后续不得为了减少文件数、缩短调用链或统一形式而合并、绕过或降级：

### 3.1 Organization Provider

Organization Provider 是组织事实与其他平台领域之间的边界。数据权限及后续 Integration 不得直接访问 `org_*` Repository 获取组织范围，也不得在调用方复制组织关系算法。

### 3.2 Data Permission Resolver 与 DataScopeResult

Resolver 负责组合授权语义，DataScopeResult 负责表达 `not_applicable`、`all`、`none`、`filtered` 四态结果。后续模块不得在 Controller 或业务 Repository 中重新读取 Grant、Policy 并实现另一套权限算法，也不得将异常或空范围降级为全量访问。数据权限的详细冻结结论见 [DataPermissionFreezeReview.md](DataPermissionFreezeReview.md)。

### 3.3 MetadataFieldAdapter 与 RegisteredFieldAdapter

Adapter 是结构化权限结果与具体查询执行之间的安全边界。客户端不得提交 SQL、表名、字段表达式或权限过滤条件；业务模块必须通过受控 Metadata 或服务端注册能力接入。

### 3.4 Generalization

Generalization 统一承载低代码列表、total、详情、更新、删除、批量删除和导出的查询链。后续不得拆出绕过 Resolver/Adapter 的旁路，也不得让不同入口使用不一致的数据权限结果。

### 3.5 保护原则

- 不以“通用化”为由创建跨领域万能 Service 或万能 Registry。
- 不以“减少代码量”为由删除 Provider、Resolver、Adapter、DTO 或 Repository 端口。
- 不让 Controller 承担业务编排、事务控制或权限计算。
- 核心边界调整必须经过独立设计、兼容性评估和冻结评审。

## 4. 延期治理事项

下列事项属于存量代码的渐进优化，不是当前平台功能缺陷，也不阻塞 Integration Foundation：

| 延期事项 | 当前状态 | 后续处理原则 |
| --- | --- | --- |
| Model/Repository 去 Gin | 已建立标准 AuditSubject 和 Context 入口，存量兼容仍在 | 按模块迁移，不一次修改全部 Model 与 Repository |
| FileController 拆分 | 签名用途隔离已完成，Controller 仍偏大 | 先补行为测试，再按上传、下载、签名职责渐进拆分 |
| ReportService 拆分 | 当前报表领域边界有效，但 Service 规模较大 | 遵守报表第一阶段架构约束，按发布、执行、导出职责小步演进 |
| SysTableService 拆分 | 元数据、Schema 与发布流程集中 | 保持行为和事务语义，优先抽取纯辅助逻辑 |
| 更多 DTO 迁移 | 高风险模块已完成第一阶段白名单治理 | 按接口风险和前端契约逐模块推进 |
| 事务存量迁移 | 新规范已冻结，历史入口仍并存 | 新代码遵守 Service 事务入口，旧代码随模块整改 |
| 错误体系存量迁移 | 高风险直接错误文本已完成轻量整改 | 按模块补稳定领域错误，不进行全仓机械替换 |
| 测试夹具迁移 | 已提供统一 Helper 和 Gin 包级初始化 | 逐步替换重复 SQLite、AutoMigrate 和全局状态初始化 |
| 统一认证存量迁移 | 目标架构已冻结，现有入口尚未全部统一编排 | 按认证设计分阶段迁移，禁止新增安全策略分叉 |
| 疑似死代码确认 | 已完成证据审计，部分方法仍有 Swagger 或接口边界 | 先决定外部契约，再单独清理，不在业务任务中顺手删除 |

延期事项不得成为新代码继续复制历史模式的理由。新增模块必须直接遵守冻结规范。

## 5. 新增代码准入规则

自本评审通过之日起，新增代码必须满足以下要求：

1. **标准 Context**：Service、Repository、异步任务和领域能力使用 `context.Context`；不得把 `*gin.Context` 传入异步任务或新增底层领域依赖。
2. **事务边界**：Service 定义业务事务，Repository 不自行开启业务事务，Controller 不创建事务；HTTP、消息、文件和第三方调用不得长期占用数据库事务。
3. **Response DTO 白名单**：Controller 只返回受控 Response DTO；列表和详情分别定义字段，不直接暴露完整 Model。
4. **稳定错误**：Repository 传播技术错误，Service 转换领域错误，Controller/Middleware 映射安全响应；外部响应不得包含 SQL、表名、字段名、堆栈或底层依赖信息。
5. **统一测试 Helper**：数据库、Gin、缓存、Registry 和全局状态测试优先复用 `backend/internal/test` 及包级初始化能力；PostgreSQL 专属约束不得仅由 SQLite 代替。
6. **结构化日志**：使用平台日志组件和稳定字段，至少保留必要的 `request_id`、`trace_id`、主体及业务对象摘要；禁止新增 `fmt.Print*` 处理生产错误。
7. **安全失败**：权限、认证、签名、Provider 或 Adapter 异常不得默认放行，不得通过旧链路或弱校验静默回退。
8. **领域扩展**：Integration 等新模块通过既有 Provider、Resolver、Adapter、DTO 和 Service 边界扩展，不复制平台级算法。

代码评审应把上述规则作为准入条件，而不是仅作为后续整理建议。

## 6. Integration 准入判断

**允许进入 Integration Foundation。**

准入理由如下：

1. Organization 与 Data Permission 已完成架构冻结，Integration 可以依赖稳定的组织事实、主体上下文和权限执行边界。
2. 异步任务 Context 和文件签名用途等 P0 安全问题已完成针对性治理，新增集成任务有明确的安全实现基线。
3. 错误、事务、测试、DTO、日志和标准 Context 已形成正式规范，Integration 无需复制历史实现方式。
4. 当前延期事项集中在存量代码维护成本，不影响新模块按冻结边界独立建设。
5. 正式文档目录、命名和引用规则稳定，后续 Integration 设计与操作文档有明确交付位置。

Integration Foundation 必须遵守以下约束：

- 不绕过现有功能权限、数据权限及 Casbin 按钮/API 权限体系。
- 不直接访问组织表或复制组织关系算法，组织事实通过 Organization Provider 获取。
- 不改变 Data Permission 核心模型、Resolver 四态语义和 Adapter 安全边界。
- 不在 Controller 中实现集成业务编排、事务或重试策略。
- 不通过原始错误文本、SQL、表名或字段表达式建立外部契约。
- 不把外部系统调用放入长事务；重试、幂等和审计必须在 Integration 自身边界内设计。

如果后续 Integration 需求与冻结边界冲突，应先发起独立平台设计评审，不得在业务实现中隐式修改平台核心能力。

## 7. 最终结论

本次评审结论为：**通过冻结**。

判断理由：

- 第一轮平台治理已覆盖文档、代码规范、安全、架构边界、测试基础设施和低风险清理。
- 已识别的高风险问题获得了代码治理或正式目标架构约束，不存在要求通过删除核心抽象解决的问题。
- Organization Provider、Data Permission、Adapter 和 Generalization 的职责边界清晰，可作为后续模块依赖的稳定底座。
- 延期项目均有明确迁移原则，主要影响存量维护成本，不阻塞新基础设施按新规范建设。
- Integration Foundation 已具备统一的工程准入规则和安全约束。

冻结不表示平台代码从此不再演进，而是表示后续演进必须保持当前领域边界和安全语义。新增能力可以扩展端口与实现，但不得绕过、复制或削弱已冻结核心。

## 8. 后续路线

下一阶段进入 **Integration Foundation**。

进入具体设计前，应以本评审冻结的标准 Context、事务、错误、DTO、测试、日志、安全失败和领域边界作为前置约束。Integration Foundation 的具体领域模型、运行机制和实施拆分由后续独立任务评审，本文件不展开其方案设计。
