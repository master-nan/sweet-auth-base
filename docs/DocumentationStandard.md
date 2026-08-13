# Sweet Platform 文档治理规则

## 1. 写文档前的四个问题

新增文档前必须先明确：

1. 主要读者是谁；
2. 文档解决什么长期问题；
3. 生命周期是长期、活跃施工、生产准入证据，还是本地临时资料；
4. 已有文档能否承载，是否真的需要新文件。

没有明确读者和生命周期的文档不得创建。不得默认写入 `docs/` 根目录。

## 2. 目录规则

| 内容 | 目录 | 生命周期 |
| --- | --- | --- |
| 用户、业务管理员、平台管理员操作 | `docs/user-guide/` | 长期产品资料 |
| 架构、开发、测试、扩展契约 | `docs/engineering/` | 长期工程资料 |
| 部署、环境、配置、运行、排错 | `docs/operations/` | 长期运维资料 |
| Task、Audit、Acceptance、Freeze、临时 Design、实现 Evidence | `docs/_construction/` | 建设期临时资料 |
| 本地 Task 草稿、原始分析、敏感 Swagger/响应 | `docs/development/` | Git 忽略，稳定后删除 |

`docs/` 根目录只保留导航和极少数全局规则。不得新建永久的 `acceptance/`、`freeze/` 或其他会把施工证据伪装成产品文档的目录。

## 3. 建设期文档元数据

新增 `_construction` 文档应在开头说明：

- `Audience`：预期读者；
- `Lifecycle`：`construction`、`active-implementation` 或 `production-enablement-evidence`；
- `Final Action`：`KEEP`、`MERGE`、`REWRITE`、`DELETE_AFTER_STABLE`、`DELETE_AFTER_PRODUCTION_GATE`、`IGNORED_RAW`；
- `Removal Gate`：删除或转正条件。

旧文档由 [DocumentationInventory.md](_construction/DocumentationInventory.md) 统一补充生命周期，不要求本轮逐文件改写正文。

## 4. 内容边界

- User Guide 只写已存在、可操作的产品能力，不用阶段 Task 或测试数据冒充用户功能。
- Engineering 文档写稳定架构事实和扩展约束，不长期保留实施流水账。
- Operations 文档写可执行的环境、部署、配置和故障处理，不复制架构正文。
- Construction 文档可以保留决策证据，但必须明确不是最终产品文档。
- 原始响应、真实人员数据、内网地址、Token、Cookie、Authorization 和 Credential 秘密永不进入 Git。

## 5. 引用与安全

迁移或重命名文档时必须同步修复全仓 Markdown、README、Makefile、脚本、CI、代码注释和本地 development 记录中的路径。提交前运行：

```bash
python3 scripts/check_docs.py
```

同时执行受跟踪文档敏感信息扫描，确认没有真实姓名、手机号、身份证、真实邮箱、内网 Host、密钥或原始业务响应。示例值必须明显不可用且不对应真实主体。

## 6. 最终清理

DOC-FINAL 才执行建设资料删除。删除前必须证明：

1. 仍有效的架构和安全结论已吸收进长期文档；
2. 所有 Production Gate 已关闭或迁入正式准入文档；
3. 没有代码、脚本、README 或开发流程继续引用旧文件；
4. Git 历史足以承担已结束阶段的追溯需求。
