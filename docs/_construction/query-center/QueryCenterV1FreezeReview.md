# Sweet Platform Query Center V1 Freeze Review

> Lifecycle: construction
>
> Final Action: DELETE_AFTER_STABLE
>
> Freeze Baseline: `55e4f9fa9ba865084d24e888d37321511946b717`

## 1. Freeze 结论

**Query Center V1 Freeze = PASS**。

后端安全边界、18 个 Eligible 页面接入、Data Permission AND 语义、权限隔离、UX、PostgreSQL 16、Race、浏览器和 Console 门禁均通过。P0=0，P1=0；允许进入 Platform Final Code Review，不新增 QC-002D。

## 2. 冻结能力

1. `sys_menu.query_scope_code` 是唯一 Query Scope Identity；Registry 只提供运行配置，Frontend 只消费 Runtime Scope Config。
2. PERSONAL、PUBLIC、ROLE、PAGE_DEFAULT 四类方案及其可见性、默认、复制、启停、删除与 revision 语义冻结。
3. 默认优先级冻结为 PERSONAL > PAGE_DEFAULT > 页面原始 Query；PUBLIC/ROLE 不自动默认。
4. Query Payload 继续使用现有 Query 协议；Schema 深度上限 3，V1 UI 编辑深度上限 2。
5. Dynamic Binding 固定为后端七类白名单；Binding 不是 DSL。
6. Scheme 条件只能作为业务 Query 的额外过滤，并与 Data Permission 做 AND。
7. Business Page 通过统一 Query Center 接入 Pattern 使用 Selector、Save、Advanced Query、Dirty 和 Default；Refresh 不重新初始化方案。
8. Query Scheme Manager 继续使用 Hidden Route；共享管理只由 `query_scheme_shared_manage` 控制。
9. DEGRADED/INVALID 不自动执行、不静默删除条件、不暴露技术错误。
10. Query Scheme 不保存 page、pageSize、列偏好、Data Scope、SQL 或 Metadata snapshot。

## 3. 页面冻结范围

后端 Registry 的 18 个固定 Scope 全部 ENABLE；PARTIAL=0，EXEMPT=0。Organization Tree、Database 工作台、Dashboard、Generalization 动态页面和 Report 不因固定 Registry 被机械接入。

Report 保持 `REPORT_DEFERRED`。本 Task 未修改 Report 产品代码，仅执行公共构建与测试兼容回归。

## 4. 质量门禁

- Frontend：69 个测试文件、269 项测试、lint、typecheck、build 通过。
- Backend：全量测试通过；Query Scheme 相关 Service/Repository/Controller/Internal/Initialize Race 通过。
- PostgreSQL 16：Query Scheme 专项和全仓强制门禁通过。
- Browser：Admin、普通/只读、无页面权限和 Shared Manager 通过；18 页面遍历、双标签 revision、亮/暗主题、1366/宽屏和 Console 通过。
- Docs：`make docs-check` 通过。

## 5. P2/P3 与后续入口

P2 仅保留全局自定义主题在深色模式下的 primary 对比度治理；P3 保留既有大 chunk 优化和未来 relation display resolver。它们不改变 Query Center 的能力、安全或数据语义。

后续进入 Platform Final Code Review。稳定规则在 DOC-FINAL 吸收进长期 Engineering/User Guide 后，本目录的 Design、Gap、Acceptance 和 Freeze 文档按 `DELETE_AFTER_STABLE` 删除。

## 6. V1 明确不支持

跨表查询、JOIN 设计器、SQL 编辑、Report 接入、收藏、分享流程、审批、文件夹、订阅、导入导出、Query 历史版本、Redis Scheme Cache、第三层 UI 编辑、字段说明、条件树和列偏好保存均不属于 V1。
