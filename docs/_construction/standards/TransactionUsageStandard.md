# Sweet Platform 事务使用规范

## 文档信息

| 项目 | 内容 |
| --- | --- |
| 文档目的 | 统一 Sweet Platform 后端事务职责、入口、异常处理和渐进迁移规则 |
| 适用范围 | `backend/` 中的 Controller、Service、Repository、Internal、Migration 与测试代码 |
| 审计日期 | 2026-08-04 |
| 审计基线 | `8dc5602d935a05302fdb545bcc22d750a0a1936e` |
| 当前推荐入口 | `service.RunInTransaction` |

本规范既记录当前仓库的事务使用现状，也作为新增代码和后续模块整改的准入规则。它不要求一次性替换所有历史事务入口。

## 1. 当前事务审计

### 1.1 审计口径

本次审计覆盖 `backend/service`、`backend/repository`、`backend/repository/impl`、`backend/internal`、`backend/migrate` 及测试代码，按调用语句统计事务入口。事务辅助函数自身的 GORM 调用单独标注，避免和业务调用混为一谈。

### 1.2 调用统计

| 事务入口 | 生产 Service | 生产 Repository | Migration | 测试 | 合计 |
| --- | ---: | ---: | ---: | ---: | ---: |
| `RunInTransaction` | 26 | 0 | 0 | 8 | 34 |
| `ExecuteTx` | 26 | 1 | 0 | 0 | 27 |
| 直接调用 GORM `Transaction` | 3 | 3 | 8 | 4 | 18 |

直接调用 GORM `Transaction` 的 18 处中，有 2 处是事务辅助函数自身的实现：

- `service.RunInTransaction` 内部调用 GORM `Transaction`。
- `BasicRepository.ExecuteTx` 内部调用 GORM `Transaction`。

排除这两处实现后，独立的业务、Migration 和测试调用共 16 处。

### 1.3 模块分类

| 分类 | 模块与数量 | 当前结论 |
| --- | --- | --- |
| 新模块 | Data Permission 24 处、Organization 账号绑定 2 处使用 `RunInTransaction` | 符合当前 Service 事务基线，保留 |
| 存量 Service | SysTable 20 处、SysMenu 3 处、SysRole 1 处、Report 2 处使用 `ExecuteTx` | 按模块渐进迁移，不做机械替换 |
| 存量 Repository | File Repository 1 处使用 `ExecuteTx` | Repository 自开业务事务，列入后续迁移 |
| 直接业务调用 | Report 1 处、SysUser 1 处直接调用 GORM `Transaction` | 后续随模块整改迁移到 Service 入口 |
| Repository 原子操作 | Generalization Repository 2 处直接调用 GORM `Transaction` | 当前用于批量写入原子性，后续结合 Service 边界单独评审 |
| Migration / Seed | 8 处直接调用 GORM `Transaction` | 合理例外，继续由 Migration 自主管理 |
| 测试 | `RunInTransaction` 8 处，直接 GORM `Transaction` 4 处 | 用于事务语义和 Migration 验证，保留 |

### 1.4 Controller 边界

当前 Controller 未发现创建事务、获取数据库连接或调用事务辅助函数的生产代码。现有静态测试也会阻止 Controller 引入 `DBWithContext`、`ExecuteTx`、`RunInTransaction` 或 GORM 依赖。

### 1.5 嵌套事务

- 生产代码未发现显式嵌套事务调用。
- 测试代码存在 2 处有意嵌套 `RunInTransaction`，用于验证保存点和错误传播。
- `RunInTransaction` 传入已有事务句柄时，由 GORM 使用保存点表达嵌套语义。
- `BasicRepository.ExecuteTx` 总是从 Repository 持有的根数据库句柄开始事务，不能可靠加入上层已有事务，这是存量接口的重要迁移原因。

静态审计只能确认显式调用。跨 Service 的间接嵌套仍需在代码评审和集成测试中防范。

### 1.6 事务内外部调用

本次未发现事务回调中直接执行 HTTP、消息发送、文件存储、短信、钉钉或其他第三方网络调用。

以下行为不属于外部调用：

- 使用事务句柄执行数据库查询、写入和审计记录。
- PostgreSQL DDL 或 Migration 操作。
- 进程内稳定 ID 生成和注册项校验。

存量模块存在数据库提交后的跨组件操作：

- Report、SysMenu、SysRole 在数据库事务完成后同步 Casbin 策略。
- SysUser 在数据库事务完成后刷新用户缓存。

这种安排避免外部组件长期占用数据库事务，但存在“数据库已提交、后续同步失败”的短暂一致性窗口。该问题应通过模块级补偿、重试或事件机制治理，不得把网络或缓存调用简单搬进数据库事务。

## 2. 事务职责

### 2.1 Controller

Controller 只负责请求解析、参数校验、调用 Service 和响应适配。

Controller 禁止：

- 创建、提交或回滚数据库事务。
- 获取 `*gorm.DB` 或 Repository 数据库句柄。
- 根据 HTTP 请求分支决定部分提交。
- 把事务句柄放入响应、Session 或异步任务。

### 2.2 Service

Service 定义业务事务边界，决定哪些数据库变更必须整体成功或整体失败。

Service 必须：

- 在执行写操作前完成可在事务外安全完成的请求格式和静态规则校验。
- 使用统一事务入口启动业务事务。
- 把同一个 `tx` 显式传给参与该事务的 Repository 方法。
- 将并发保护、引用检查和写入放在同一事务中。
- 在事务失败时返回错误，不继续执行依赖提交结果的后续动作。

### 2.3 Repository

Repository 负责数据访问，不负责定义跨聚合业务事务。

新增 Repository 必须：

- 接收 Service 传入的数据库或事务句柄。
- 在传入事务中完成单次查询或写入。
- 原样传播数据库错误，由 Service 转换为稳定业务错误。

Repository 禁止自行开启业务事务。确需为单个持久化原语保证内部原子性时，必须说明原因、限定范围并通过架构评审，不能形成隐藏的第二层业务事务。

### 2.4 Migration 与测试

Migration、Seed 和数据库专项测试没有 Service 层，可以直接使用 GORM `Transaction`。它们必须保持幂等，并让错误直接中止事务。

单元测试可以直接使用 GORM 事务构造隔离场景，但生产 Service 测试应优先通过 `RunInTransaction` 验证真实基线。

## 3. 统一事务入口

### 3.1 新代码规则

新增业务代码默认使用 `service.RunInTransaction`。不得新增以下调用：

- Service 直接调用 `db.Transaction`。
- Service 新增 `ExecuteTx` 调用。
- Repository 自行创建跨多项业务操作的事务。
- Controller 创建任何事务。

当前 Repository Context 仍处于渐进治理阶段。取得数据库句柄的兼容方式可以暂时沿用现有接口，但事务所有权必须属于 Service；后续标准 Context 改造不得改变这一职责。

### 3.2 回调约束

事务回调必须：

1. 只使用回调收到的 `tx` 执行本事务内数据库操作。
2. 任一失败立即返回错误。
3. 不吞掉唯一约束、外键、锁或审计写入错误。
4. 不在回调内启动 goroutine。
5. 不把 `tx` 保存到结构体、全局变量、缓存或异步任务。
6. 不在事务结束后继续使用 `tx`。

禁止在事务回调中重新使用 Repository 的根数据库句柄，否则同一业务操作可能跨两个独立事务。

## 4. 错误、panic 与提交规则

### 4.1 正常提交

回调返回 `nil` 后由 GORM 提交事务。提交成功后 Service 才能返回成功或执行明确的提交后动作。

### 4.2 错误回滚

回调返回错误时必须回滚，并将错误向上传播。需要补充语义时使用 `%w` 保留错误链，确保 `errors.Is` 和统一错误映射仍然有效。

不得：

- 记录错误后返回 `nil`。
- 把部分失败改成成功。
- 在回滚后继续使用事务中生成但未落库的数据作为成功结果。

### 4.3 panic 处理

`RunInTransaction` 采用 GORM 的事务行为：panic 触发回滚，并继续向上抛出。panic 不属于正常业务分支，不应在 Repository 中静默恢复成成功。

HTTP 场景由统一恢复中间件处理未捕获 panic；后台任务由任务执行框架记录并隔离。若某模块需要把特定 panic 转换为稳定错误，必须在更高层明确评审，不能在通用 Repository 中隐藏未知故障。

存量 `BasicRepository.ExecuteTx` 会捕获 panic 并转换为错误，其行为与新基线不同。迁移时必须补充对应回归测试，不能只替换函数名。

### 4.4 回滚和提交失败

- 回滚或提交失败均不得返回成功。
- GORM 返回的事务错误必须由 Service 传播或转换为稳定业务错误。
- 客户端响应不得暴露数据库驱动文本、索引名或 SQL。
- 日志应包含 request/trace 标识和稳定业务对象信息，不记录敏感参数或完整 SQL。

## 5. 嵌套事务规范

默认禁止通过嵌套事务组织普通业务流程。优先把事务所有权放在最外层 Service，并将同一个 `tx` 传入内部协作者。

只有确需局部保存点时才允许嵌套，且必须：

1. 把外层 `tx` 传给内层 `RunInTransaction`。
2. 明确内层失败是仅回滚保存点，还是向外传播并回滚整个事务。
3. 增加保存点成功、局部回滚和错误传播测试。
4. 不在嵌套层使用根数据库句柄。

禁止通过调用另一个会自行开启事务的 Service 或 Repository 来制造隐式嵌套。

## 6. 事务边界与外部调用

### 6.1 应放在事务内

- 需要保持一致的数据库读取、锁定、引用检查和写入。
- 与业务变更必须同成同败的数据库审计记录。
- 依赖数据库约束判定结果的更新或删除。

### 6.2 应放在事务外

- HTTP 和第三方 API。
- 消息发布。
- 文件上传、下载和对象存储操作。
- 短信、邮件、钉钉等通知。
- Redis 或本地缓存刷新。
- 大量计算、报表生成和长时间等待。

在进入事务前调用外部能力时，不得假设后续数据库一定提交成功。在提交后调用外部能力时，必须考虑失败补偿、幂等重试或事件投递。需要数据库与消息最终一致时，应单独设计 Outbox 或可靠事件机制，不得用长事务包住网络调用。

### 6.3 锁和耗时

事务应尽可能短。加锁读取应靠近写入，完成必要写入后立即结束事务。分页查询、文件处理、远程校验和前端等待不应占用数据库事务。

## 7. 提交后动作

缓存、Casbin、消息和第三方状态更新属于提交后动作。Service 必须明确其失败策略：

- 可以安全重试的动作应使用稳定业务键保证幂等。
- 必须与数据库最终一致的动作应具备补偿或重建能力。
- 不允许因提交后动作失败而声称数据库事务已回滚。
- 不允许为避免一致性设计而把外部调用移入数据库事务。

当前 Report、SysMenu、SysRole 和 SysUser 的相关流程按存量行为保留，后续应按模块补充失败补偿与重建测试。

## 8. 测试要求

新增或迁移事务代码至少覆盖：

1. 全部成功时提交。
2. 中间步骤失败时全部回滚。
3. 原始错误或稳定业务错误能够传播。
4. panic 时不提交任何写入。
5. 提交失败不返回成功。
6. 约束冲突时不产生部分数据。
7. 审计写入失败时业务写入回滚。
8. 存在并发竞争时仍满足唯一性和状态约束。
9. 如使用嵌套保存点，覆盖局部回滚和外层回滚。
10. 提交后动作失败时具备明确、可验证的处理结果。

涉及共享状态或并发流程时，相关包应执行 `go test -race`。Migration 事务应在 PostgreSQL 专项测试中验证，不能只依赖 SQLite。

## 9. 渐进迁移策略

事务治理采用新增代码准入和模块级整改，不执行全仓文本替换。

### 9.1 新增代码

- 一律以 Service 定义事务边界。
- 一律使用 `RunInTransaction`。
- Repository 接收外部事务句柄。
- 外部调用与长耗时操作不进入数据库事务。

### 9.2 存量模块

建议按以下顺序整改：

1. SysUser 和 Report 中直接调用 GORM `Transaction` 的 Service。
2. File Repository 自开事务。
3. SysMenu、SysRole 和 Report 的 `ExecuteTx` 调用及 Casbin 提交后一致性。
4. SysTable 大量 `ExecuteTx` 调用，结合 DDL、缓存和菜单发布整体评审。
5. Generalization Repository 的批量原子操作，结合受控写入边界单独评审。

每个模块迁移必须先补足回滚、panic、约束冲突和提交后动作测试，再替换事务入口。

### 9.3 不采用的迁移方式

- 不删除 `ExecuteTx` 后强制全仓编译修复。
- 不把 Repository 事务简单搬到 Controller。
- 不同时重构领域逻辑、Context、缓存和事务。
- 不修改已冻结的 Data Permission Resolver、Adapter 或 Organization Provider 边界。

## 10. 第一阶段整改结论

本次未修改生产代码，原因如下：

- 新 Data Permission 与 Organization 事务已经使用 `RunInTransaction`，未发现明确错误。
- 未发现生产代码显式嵌套事务。
- 未发现事务回调直接执行外部网络、消息或文件调用。
- 其余差异集中于存量模块，直接替换会同时影响 panic 语义、Gin Context、DDL、Casbin 或缓存一致性，不符合轻量整改边界。

因此，第一阶段以正式规范和审计基线收口。后续新增代码必须遵守本规范，存量代码按模块逐步整改。

## 11. 代码评审检查表

- 事务是否由 Service 创建？
- 是否使用 `RunInTransaction`？
- Controller 是否完全不接触数据库事务？
- Repository 是否只使用传入的 `tx`？
- 任一错误是否会立即返回并触发回滚？
- 是否存在吞错、部分成功或失败后继续写入？
- 是否把 HTTP、消息、文件、第三方或缓存调用放进事务？
- 是否存在隐式或不必要的嵌套事务？
- 是否覆盖回滚、panic、约束冲突和提交失败？
- 提交后动作是否具备幂等、重试或补偿策略？
