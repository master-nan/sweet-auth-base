# Platform V1 Final Runtime Freeze Review

> 状态：PASS
> 基线：`c42e1ae084b6137492f092b151c7fd48777e26fa`
> 审计日期：2026-08-21
> 性质：FINAL RUNTIME / RELEASE HARDENING

## 1. Current HEAD

实施前工作区干净，`git diff`为空，当前分支为`main`，HEAD与任务基线一致。审计重新读取了GitHub Actions、Makefile、容器入口、配置、Migration、数据库与Redis初始化、Integration Runner、文件分片、运维脚本、Frontend构建和既有长期文档；本报告随实现持续更新。

## 2. Runtime Topology

V1运行拓扑为Frontend静态服务、Go HTTP应用、PostgreSQL 16、Redis，以及Go进程内的Cron、Integration Worker、Sync Runner和Chunk Cleanup。应用由Wire组装Auth、Casbin、Metadata、Data Permission、Query Scheme、File、Report和Organization等既有边界。

原主进程先停Runner、后停HTTP，且没有显式关闭Redis和`sql.DB`。最终建立signal root Context和45秒统一关闭预算：停止HTTP接流量；取消Cron、Sync Runner、Worker和Chunk Cleanup；等待受控in-flight请求/任务；关闭Redis和SQL连接池；最后flush/close异步日志。Runner没有新增第二套生命周期框架。

## 3. Release Gate

本地与CI共用`make release-check`作为唯一完整清单：

- tracked secret scan、docs-check、Node运维脚本测试；
- 强制PostgreSQL单轮测试、包级串行三轮稳定性测试、包级串行全量Race；
- Frontend Vitest、lint、typecheck、production build。

`-p=1`只限制跨包并行，包内并发和Race instrumentation保持有效；原因是多个PostgreSQL测试包同时建立隔离连接池会在`max_connections=100`下产生环境性耗尽。旧Backend/Frontend workflow已收敛为单一`.github/workflows/release.yml`，PR与main直接调用同一Make目标，不维护第二份命令列表。

Node唯一版本真值为仓库根`.nvmrc`的`22.23.0`；Frontend engine限定`>=22.23.0 <23`，CI通过`node-version-file`读取根文件，删除了Frontend第二份`.nvmrc`。

## 4. Migration

`backend/internal/migration`提供15个稳定version/key/contract/checksum定义和53张受管表目录。`schema_migration`记录`version`、`key`、`checksum`、`applied_at`；同version的key/checksum变化、未知版本、乱序或缺失ledger均fail closed。Migration步骤与成功ledger insert处于同一事务，失败不会登记成功；Migration和Seed使用按database/schema派生的PostgreSQL session advisory lock串行化。

当前`version: 0.1`不是可靠的发布版本来源，因此ledger不写入容易漂移的application version；正式事实只使用稳定Migration version/key/checksum。

Fresh DB逐步迁移并登记。已有Canonical或部分升级但无ledger的数据库会被普通`migrate`拒绝，必须先备份和核验，再显式运行`migrate adopt`；Adopt重跑实际幂等步骤后逐步登记，不按“表存在”盲写完成事实。仓库内置的本地完整Compose由`make docker-up`显式执行受控Adopt，以便既有开发卷一次性接入Ledger；外部/生产Compose不自动Adopt。测试Fixture的AutoMigrate不进入ledger。V1基线仍包含现有`auto_migrate_core_schema`首步，后续生产Schema变化必须追加显式catalog步骤。

Release Drill第一次严格Preflight发现`access_log`缺少运行所需索引，说明仅有表结构并不足以安全上线。修复通过追加version 15 `access_log_operational_indexes`完成，没有改写已登记checksum。最终Fresh、14→15 Upgrade、Adopt、重复执行、checksum mismatch、失败不登记和并发启动均由PostgreSQL 16测试覆盖。

## 5. DB

应用、Migration和Preflight共用`backend/internal/database.PostgresDSN`，仅接受`disable`、`require`、`verify-ca`、`verify-full`四种受控TLS模式；CA、client cert和key采用结构化配置，不接受任意DSN片段。开发/测试显式`disable`，生产样例为`verify-full`，secure环境拒绝`disable`。连接池在统一生命周期中显式关闭。

## 6. Redis

Redis新增受控TLS配置：`enabled`、`server_name`、CA、client cert和key，最低TLS 1.2，不提供跳过证书校验开关。Production要求TLS与server name；Development可显式关闭。Redis客户端由统一生命周期关闭，Preflight使用同一options构造边界。

## 7. Process Lifecycle

Worker和Sync Runner继续使用既有可取消Context、Timer和ShutdownTimeout。HTTP listener在收到信号后立即停止接受连接，Runner不再claim新任务，in-flight在统一预算内收敛。异步日志原`Close`持文件锁再次进入写路径，缓冲日志存在时可能自锁，现已改成停止接收、等待消费goroutine、关闭文件的幂等流程。

`backend/main_test.go`以真实listener和阻塞HTTP请求验证停止接流量与资源关闭顺序；Runner专项继续验证cancel、stop claim和timeout。Release Drill对真实Go进程执行SIGINT，日志依次出现`shutdown signal received`、Sync Runner stopped、Worker stopped、`runtime shutdown completed`，进程退出码为0。

## 8. Container

Container entrypoint在Migration、Seed和严格Preflight完成后使用`syscall.Exec`将PID 1替换为`/app/sweet_admin`，不再由shell/Go父进程Start/Wait子进程。Compose `stop_grace_period`为45秒，与应用预算一致。专项测试覆盖Preflight失败不启动应用及最终exec参数。

## 9. File Lifecycle

合并成功清理`chunks/<upload_id>`；写入或合并失败清理当前不完整文件。新增Chunk TTL清理仅遍历受控`chunks`根目录，按目录和分片最新活动时间判断，跳过symlink，启动时执行并按配置周期重复。当前没有正式客户端cancel API，本轮不扩协议，放弃上传与进程崩溃残留由TTL兜底。

Local Chunk Staging仍为节点本地能力。多实例部署必须对同一upload session使用粘性路由；共享Staging为明确Deferred，不把当前实现描述为多节点共享。

## 10. Frontend Production Build

PPTX与HEIC继续按文件类型dynamic import。移除手工`vendor-file-preview`分组后，最终`index-B6tGy-Wp.js`入口为4,179 bytes；初始modulepreload仅包含rolldown runtime、Quasar和Vue。PPTX chunk `aiden0z-pptx-renderer.es-DR5uuHva.js`为1,444,611 bytes，HEIC chunk `heic2any-DBUyPcaT.js`为1,352,159 bytes，二者均不在modulepreload，普通登录和业务列表不会下载。

全局主题设置入口从右下角fixed浮层移入Header；浏览器测得按钮为Header内`position: relative`，不再覆盖表格、操作或分页。

## 11. Preflight

`db-preflight`复用Migration catalog，覆盖53张受管表、完整ledger/checksum、关键列/索引/约束、Canonical Metadata与物理结构一致性、Casbin、File、Dictionary、数据库footprint和Redis。Production/secure模式同时拒绝启用的默认Application secret。Preflight只读，不执行Migration；错误描述对象类别，不输出DSN、密码或credential。

Release Drill严格Preflight结果：PostgreSQL与Redis均OK，`schema_migration.applied=15`、`expected=15`；Metadata缺列、类型、nullable、length、relation/dict孤儿、权限孤儿和文件回填pending全部为0，warnings/problems均为空。

## 12. Backup / Restore

外部备份脚本固定使用PostgreSQL 16 client和结构化TLS配置。schema v2 manifest记录实际数据库身份、dump文件SHA-256和Migration Ledger摘要。Restore先校验manifest版本、文件权限、hash、目标环境与显式破坏性确认，再以`psql --single-transaction`恢复，随后核对ledger、运行严格db-preflight和readiness；全部成功才写owner-only evidence。

本地/OSS业务文件需按部署存储策略与数据库备份配套，数据库脚本不伪装成完整文件对象恢复平台。脚本Node测试覆盖hash/manifest/ledger mismatch、目标确认和preflight失败路径。

## 13. Security

Production配置拒绝弱Session/Conf/DB/Redis/OSS/SMS配置、非TLS连接、不安全CORS/Upload访问、无效Chunk TTL与启用的默认Application secret。生产Seed要求显式提供至少32字符随机`APP_BOOTSTRAP_APPLICATION_SECRET`。Tracked secret scan进入本地与CI门禁，只报告规则和文件位置，不回显命中值。

Report运行入口原缺少对象所属页面授权，且SQL执行没有强制deadline。最终按`permission_menu_id`、当前活动角色菜单/按钮和Casbin policy过滤list/detail/run/export；管理动作继续使用各自Capability，Data Permission继续叠加。设计预览、运行、导出分别使用10秒、30秒、2分钟Context预算，PostgreSQL事务内使用transaction-local `statement_timeout`，连接复用测试确认不泄漏。

真实只读浏览器验收又发现Report Manager按钮曾使用“页面定义能力”而非“用户已授权能力”；最终改用`hasGrantedCapability`。只读用户仅显示获授的运行/导出，写操作不渲染且后端返回403。新增Vue行为测试防止回归。

## 14. Browser Acceptance

使用Fresh Release Drill数据库、生产静态构建和真实登录执行：

- Admin：遍历Home、System、Develop、Organization、Integration、Query Scheme、Data Permission、Report Center/Manager/V2 Workbench等25个正式入口；1366x768无横向溢出，核心页面深色模式正常。
- Read-only：Application、Employee、External System、Data Permission、Query Scheme、Report Center/Manager可读；业务新建/修改按钮不显示，Report Manager仅显示运行/导出，写API返回403。
- No-permission：动态菜单为0，直接访问Application、Report Manager、Data Permission和Integration页面进入404；Query Scheme hidden route可进入但scope与方案均为空，Runtime Scope API返回403；Report list按对象授权返回空集，不泄露报表。

最终完整静态产物重新挂载后再次遍历，统计窗口内Console为0 Error、0 Warning，无Unhandled Promise或意外资源404。构建过程中原地替换bind mount曾产生旧hash动态模块404，该验收环境问题通过按真实发布方式重新挂载完整不可变产物消除，不计为产品运行结果。

Fresh Seed没有发布的Low-code/TMS页面，也没有TMS Metadata，不能伪造浏览器路由；合法空Query通过Generalization行为测试确认不会再产生400。JSON、Array、KeyValue、RichText的Create/Edit/Readonly/Required边界由DynamicFieldControl与ComplexValueEditors行为测试覆盖。当前没有独立File菜单，File能力由API/Service、Dynamic Form和Preview组件链路验收。

## 15. Release Drill

在隔离数据库`pfcr_s3_freeze`和隔离Redis DB 14完成：clean build；Fresh Migration 15步；Seed；严格Preflight；启动HTTP、Worker和Sync Runner；health/readiness；Admin/Read-only/No-permission smoke；SIGINT shutdown；restart；再次readiness和smoke；最终SIGINT shutdown。两次关闭均在预算内退出，退出码0。测试fixture、容器和临时构建物在提交前清理。

## 16. Deferred Capability

V1明确延期且不伪装为已支持：

- HR Production Enablement（真实Consumer继续disabled）；
- Report产品重设计（现有入口仅做安全与运行正确性修复）；
- Editable Grid与Master-detail增强；
- Shared File Staging；
- 移动端专项。

`TableMetadata.QueryModel()`仍有3个生产调用：Generalization 1个、Report 2个。当前只加边界注释并禁止新增调用；Generalization与Report Query Engine改造会扩大Final Runtime范围，进入Engineering backlog。Report继续`REPORT_DEFERRED`。

## 17. Final Blockers

已关闭原P1：CI真实门禁、PostgreSQL 16强制执行、Migration Ledger/Checksum/Baseline/并发、生产TLS、资源关闭、容器信号、Chunk TTL、默认Application Secret、Report对象级授权/超时和Report Manager前端权限投影。

已关闭原P2：Generalization空规则首请求、复杂Input只读/必填、Settings浮层遮挡、预览大依赖预加载。

最终分级：P0=0、P1=0、P2=2、P3=1。两个非阻塞P2分别为`TableMetadata.QueryModel()`过渡桥和Local Chunk Staging多实例限制；一个P3为尚未建设Prometheus/OpenTelemetry指标端点。三项均有明确边界、文档和Deferred/Backlog归属，不构成V1 Capability Freeze阻塞。

## 18. Freeze Conclusion

- Platform V1 Capability Freeze：**PASS**
- Production Release Readiness：**CONDITIONAL**

最终`make release-check`通过：tracked secret scan、69份Markdown docs-check、55项Node脚本测试、PostgreSQL 16强制单轮、三轮、全量Race、68个Frontend test文件/240项用例、lint、typecheck和production build全部通过。此前独立执行的包级串行`go test ./... -count=3`亦通过。

CONDITIONAL仅指真实生产发布仍必须由部署方提供受信任的PostgreSQL/Redis证书、Production DSN/Secret、文件存储备份目标，并在目标环境执行Preflight与Release Drill；代码仓库不能替代这些外部部署事实。仓库内P0/P1为0，允许进入DOC阶段。
