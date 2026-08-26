# Sweet Platform 部署运维指南

本文是 Sweet Platform 当前唯一的长期部署、运行和排错手册，面向运维、实施和发布人员。平台功能配置见[平台管理员使用手册](../user-guide/PlatformAdministrationGuide.md)，代码扩展见[平台扩展开发指南](../engineering/ExtensionDevelopmentGuide.md)。

## 1. 当前运行组成

标准运行环境包含：

```text
Browser
  -> Frontend static server
  -> Backend HTTP service
       -> PostgreSQL 16
       -> Redis
       -> Local storage or Aliyun OSS
       -> configured external systems
```

仓库提供两种 Compose 方式：

- `docker-compose.yml`：本地完整环境，包含 PostgreSQL 16、Redis 6.2.7、backend 和 frontend。
- `docker-compose.external.yml`：只启动 backend/frontend，连接已有 PostgreSQL 和 Redis。

后端使用 Go 1.23.2；前端 Node.js 的唯一版本入口是仓库根 `.nvmrc` 的 `22.23.0`，`package.json` 将 engine 限定为 `>=22.23.0 <23`。生产环境必须使用 PostgreSQL；SQLite 只用于允许的常规单元测试夹具。

## 2. 配置来源与优先级

后端根据 `APP_ENV` 读取 `backend/config-<env>.yaml`。仓库提供 `config-dev.yaml`、`config-docker.yaml`、`config-pro.yaml`；`prod`、`production` 等生产取值回落到生产配置。Viper 使用 `APP_` 环境变量覆盖配置，层级中的点转换为下划线，例如：

```text
dbs.primary.host -> APP_DBS_PRIMARY_HOST
integration.worker.worker_id -> APP_INTEGRATION_WORKER_WORKER_ID
upload.oss.access_key_secret -> APP_UPLOAD_OSS_ACCESS_KEY_SECRET
```

生产秘密只能通过部署平台秘密管理或受保护环境文件注入，不得提交 Git。`.env.external.example` 是字段模板，不是可直接用于生产的配置。

### 2.1 关键配置组

| 配置组 | 关键字段 | 作用 |
| --- | --- | --- |
| `dbs.primary` | host、port、name、user、password、prefix、TLS mode/CA/client cert/key | PostgreSQL |
| `redis` | host、port、db、password、pool、TLS enabled/server name/CA/client cert/key | Cache、认证状态等 |
| `session` | secret | Session 安全材料 |
| `security` | Casbin coverage、CORS origins/credentials | HTTP 安全策略 |
| `audit` | access_log_retention_days | AccessLog 保留策略 |
| `upload` | driver、dir、base_url、大小、扩展名、MIME、OSS | 文件存储 |
| `integration.worker` | enabled、worker_id、poll、claim、concurrency、lease、shutdown | Execution Worker |
| `integration.sync_runner` | enabled、runner_id、poll、batch、shutdown | Sync Runner |
| `aliyun.sms` | access key、sign、template、发送间隔 | 短信验证码 |
| `conf` | salt、enable | 平台配置加密和现有通用开关 |
| Bootstrap/startup | application secret、admin password、run migrations、run seeds | 首次数据和启动期写入边界 |

生产模式会检查 Session/配置 Salt、数据库和 Redis、CORS、上传安全以及 OSS/SMS 配置完整性。不要关闭 `APP_REQUIRE_SECURE_CONFIG` 或 Casbin coverage 来绕过启动失败，应修正配置。

### 2.2 外部环境文件

先通过脚本生成权限为 600 的本地文件，再填写真实值：

```bash
node scripts/preflight-external.mjs init .env.external .env.external.example
chmod 600 .env.external
node scripts/preflight-external.mjs .env.external
```

`docker-compose.external.yml` 对所有安全关键变量采用必填插值，不提供数据库口令、Session/Salt、Bootstrap secret，也不默认关闭或开启 secure config、Migration、Seed。`.env.external.example` 明确选择 development，仅用于复制后填写；生产必须改为 `APP_ENV=production`，启用 `APP_REQUIRE_SECURE_CONFIG`，设置 PostgreSQL/Redis TLS 和独立强秘密。development 环境检查需要显式设置 `SWEET_ADMIN_PREFLIGHT_ALLOW_NON_PRODUCTION=true`。

生产写操作还会检查 `SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE` 和显式确认。不要把 `.env.external` 放入仓库，也不要在工单、聊天或日志中粘贴完整文件。

对 `production` 目标执行受控写操作时，脚本要求额外设置：

```bash
CONFIRM_EXTERNAL_PRODUCTION_WRITE=I_UNDERSTAND_THIS_WRITES_PRODUCTION
```

该确认只防止误操作，不代替审批、备份和停机窗口。

## 3. 本地开发启动

### 3.1 直接运行

先确保本机 PostgreSQL 和 Redis 与 `backend/config-dev.yaml` 一致。

```bash
cd backend
go mod download
go run ./migrate
go run ./migrate seed
go run main.go
```

另一个终端启动前端：

```bash
cd frontend
yarn
yarn dev
```

默认前端基路径为 `/sweet_admin`。`make docker-up` 暴露 PostgreSQL/Redis 的默认宿主机端口是 `15432/16380`，而 `config-dev.yaml` 默认使用 `5432/6379`；混用时应显式覆盖环境变量。

### 3.2 本地完整 Docker

```bash
make docker-up
docker compose ps
curl -fsS http://127.0.0.1:9009/healthz
curl -fsS http://127.0.0.1:9009/readyz
```

本地完整 Compose 会先启动 PostgreSQL/Redis，再通过 `migrate adopt` 接入或初始化
Migration Ledger，最后启动 backend/frontend。该步骤只属于仓库内置的本地开发环境；
它会执行真实幂等 Migration，不会仅按表名盲写 Ledger。外部及生产数据库仍必须按下文
的备份、Preflight 和显式 Migration/Adopt 流程执行，不会由应用启动自动认领。

默认地址：

| 服务 | 地址 |
| --- | --- |
| 前端 | `http://localhost:8008/sweet_admin` |
| 后端 API | `http://localhost:9009/sweet_admin` |
| Liveness | `http://localhost:9009/healthz` |
| Readiness | `http://localhost:9009/readyz` |
| Swagger | `http://localhost:9009/swagger/index.html` |
| PostgreSQL | `localhost:15432` |
| Redis | `localhost:16380` |

端口可通过 `SWEET_ADMIN_FRONTEND_PORT`、`SWEET_ADMIN_BACKEND_PORT`、`SWEET_ADMIN_POSTGRES_PORT`、`SWEET_ADMIN_REDIS_PORT` 覆盖。

常用命令：

```bash
make docker-rebuild-backend
make docker-rebuild-frontend
make docker-logs
make docker-down
```

完整 Compose 的 backend entrypoint 按本地配置执行 Migration 和 Seed；外部 Compose 要求环境文件显式声明 `APP_RUN_MIGRATIONS` 与 `APP_RUN_SEEDS`。标准 external Make 入口要求两者均为 `false`，写库只通过单独的受控命令执行。

## 4. 连接外部 PostgreSQL/Redis

1. 从模板初始化 `.env.external`，填写目标和强秘密。
2. 设置 `SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE` 为真实用途。
3. 先执行只读环境检查，不要一上来执行 Migration。
4. 备份并验证备份证据。
5. 显式执行 Migration、Seed。
6. 启动 backend/frontend。
7. 检查 readiness，再执行只读 smoke。

```bash
make external-preflight EXTERNAL_ENV_FILE=.env.external
node scripts/db-backup-external.mjs plan .env.external backups
node scripts/db-backup-external.mjs backup .env.external backups
make db-migrate-external EXTERNAL_ENV_FILE=.env.external
make db-seed-external EXTERNAL_ENV_FILE=.env.external
docker compose --env-file .env.external -f docker-compose.external.yml \
  run --rm --no-deps -e APP_DB_PREFLIGHT_REQUIRE_MIGRATED=true \
  backend /app/db-preflight
make docker-up-external EXTERNAL_ENV_FILE=.env.external
SWEET_ADMIN_EXTERNAL_ENV_FILE=.env.external node scripts/smoke-readonly.mjs
```

`.env.external` 中 `APP_RUN_MIGRATIONS` / `APP_RUN_SEEDS` 必须显式设为 `false`，避免应用每次启动隐式修改目标库。所有 external Make 入口先执行同一 preflight；生产还要求 PostgreSQL TLS mode 不是 `disable`、Redis TLS 为 true 且有 server name，并校验 TLS client cert/key 成对配置和 `APP_BOOTSTRAP_APPLICATION_SECRET`。

## 5. 健康检查与启动关闭

后端启动顺序为：读取配置、初始化日志/数据库/Redis/依赖、启动 Integration Worker、启动 Sync Runner、初始化现有 Cron、启动 HTTP Server。

- `/healthz` 是进程存活检查。
- `/readyz` 会检查 PostgreSQL 和 Redis，依赖异常时返回 503。
- 前端容器健康检查访问 `/`。

收到 `SIGINT` 或 `SIGTERM` 后，应用在统一 45 秒预算内先关闭 HTTP listener，随后取消 Cron、Sync Runner、Worker 和 Chunk cleanup，等待受控 in-flight 请求/任务结束，再依次关闭 Redis、SQL 连接池和异步日志。容器 entrypoint 使用 `exec` 让 Go 进程成为 PID 1；Compose 的 `stop_grace_period` 不得短于应用预算。发布平台仍应先摘除流量、等待 readiness 退出，再终止容器。

若 `/healthz` 正常但 `/readyz` 失败，优先检查 readiness 返回的 component、数据库连通性、Redis 和配置，而不是反复重启。

## 6. PostgreSQL 运维

### 6.1 版本与迁移

生产基线为 PostgreSQL 16。正式 Migration 由 `backend/migrate/registry.go` 的顺序 Registry 驱动，每一步必须幂等；`schema_migration` 记录 version、key、checksum、applied_at，严格 db-preflight 会拒绝缺失、未知、checksum 漂移或未完整应用的 ledger。

- Fresh DB：直接运行 `migrate`，每一步成功后在同一事务写入 ledger；失败步骤不登记。
- 已有且已达到当前 Canonical Schema、但尚无 ledger 的数据库：先备份并验证，再由运维人员显式运行 `migrate adopt`。Adopt 会在 advisory lock 下重跑全部幂等步骤并逐步登记，不会只因表存在就盲目标记完成。
- 部分升级数据库：同样只能在备份和变更窗口内运行 `migrate adopt`，由实际迁移步骤补齐；若任何步骤失败，停止并修复数据库状态，不手写 ledger。
- 已有 ledger：普通 `migrate` 只执行尚未登记的后续步骤；版本/key/checksum 不匹配或出现未知版本时 fail closed。

正式 Migration 与 Seed 共享数据库 advisory lock，避免多个实例并发修改 Schema/基础事实。`dbs.primary.prefix` 当前不属于生产 Migration/Preflight 契约，生产 secure config 会拒绝非空值。

```bash
make db-migrate
make db-seed
```

对外部环境使用：

```bash
make db-migrate-external EXTERNAL_ENV_FILE=.env.external
make db-seed-external EXTERNAL_ENV_FILE=.env.external
```

迁移和 Seed 后检查数据库结构与依赖：

```bash
docker compose --env-file .env.external -f docker-compose.external.yml \
  run --rm --no-deps -e APP_DB_PREFLIGHT_REQUIRE_MIGRATED=true \
  backend /app/db-preflight
```

该命令检查 PostgreSQL、Redis、migration ledger、核心表、Seed 基线、Casbin、AccessLog 索引、File 回填和 Metadata 完整性。

Seed 负责初始菜单、按钮、角色关系、字典、Metadata、Organization、Integration 和 Data Permission 的基础事实。Seed 可重跑，但仍应纳入变更窗口和备份流程。

### 6.2 备份与验证

```bash
node scripts/db-backup-external.mjs plan .env.external backups
node scripts/db-backup-external.mjs backup .env.external backups
node scripts/db-backup-external.mjs verify backups/<backup.sql>
```

脚本使用 PostgreSQL 16 client 容器并遵循 `APP_DBS_PRIMARY_TLS_*`，生成 SQL 和 schema v2 SHA-256 manifest。manifest 同时记录实际数据库身份和 `schema_migration` 的条目数、首末版本及规范化摘要；备份文件和恢复证据包含环境信息，应保存到受控位置，不提交 Git。

恢复是破坏性操作，只能在已验证目标、变更审批和停写窗口内执行：

```bash
CONFIRM_EXTERNAL_RESTORE=I_UNDERSTAND_THIS_OVERWRITES_DATA \
CONFIRM_EXTERNAL_PRODUCTION_WRITE=I_UNDERSTAND_THIS_WRITES_PRODUCTION \
BACKUP_FILE='backups/<backup.sql>' \
node scripts/db-backup-external.mjs restore .env.external
```

生产目标还需要脚本要求的生产写确认。恢复使用 `psql --single-transaction`；SQL 成功后脚本强制核对 migration ledger、运行 `APP_DB_PREFLIGHT_REQUIRE_MIGRATED=true` 的 db-preflight，并确认 `/readyz` 全组件健康，全部通过后才写恢复 evidence。只读 smoke 仍作为恢复后的独立应用验收执行。

### 6.3 回滚策略

当前 Migration 以向前修复和幂等重跑为主，没有自动 down migration。发布回滚应分开处理：

1. 可兼容的应用版本先回滚镜像。
2. Schema 变更优先使用新的修复 Migration。
3. 只有确认需要数据级恢复时，才从已验证备份恢复。
4. 不手工删除未知约束或回写生产表来“让旧版本先跑”。

PostgreSQL TLS 支持 `disable`、`require`、`verify-ca`、`verify-full`；生产 external preflight 拒绝 `disable`。自定义 CA 或 mTLS 文件路径必须在 backend 容器和 PostgreSQL client 容器中可读，client cert/key 必须成对提供。

## 7. Redis 运维

Redis 当前用于：

- 用户、角色、菜单、配置、Metadata 等 Cache；
- 登录失败计数和锁定；
- Access/Refresh Token 撤销与 Session cutoff；
- 短信验证码消费；
- DingTalk Token/身份映射等运行状态。

Redis 不是业务数据库真值，但它是当前应用必需依赖。Redis 异常会使 `/readyz` 失败，并让认证、Refresh/Logout、短信以及缓存相关路径 fail closed。

生产 Redis 必须启用 TLS 并配置用于证书验证的 server name；可选自定义 CA 和 client cert/key，client cert/key 必须成对提供。排错顺序：连接地址和 DB 编号、密码、TLS/server name/证书、网络、连接池耗尽、Redis 内存/淘汰策略、应用日志。不要直接删除登录锁、Token blacklist 或未知 key 作为恢复手段；先确定 key 所属能力和安全影响。

## 8. File 存储

`upload.driver` 当前支持 `local` 和阿里云 OSS。两种模式都必须配置允许扩展名、MIME、单文件大小、Chunk 大小和访问 URL；生产保持 `public_preview=false`。

未完成分片只存在于受控 `upload.dir/chunks/<upload_id>` 暂存目录。应用启动时立即清理一次，并按 `chunk_cleanup_minutes` 周期删除最近活动时间超过 `chunk_ttl_hours` 的会话；合并成功会立即清理。当前没有正式客户端 cancel API，用户放弃、断连或进程崩溃后遗留的分片由 TTL 定期清理。清理器跳过 symlink，且不会扫描或删除持久文件目录。

### 8.1 Local

- `upload.dir` 必须挂载持久卷并纳入容量、权限和备份监控。
- 不要通过 Web Server 直接公开物理目录。
- Preview 和 Download 必须使用各自 purpose 的受控访问签名。

### 8.2 OSS

完整配置 endpoint、access key、bucket、base URL 和 base path。Access Secret 只从秘密管理注入。当前分片暂存仍使用应用节点本地目录，即使最终文件写入 OSS；多实例上传需要会话粘性，平台尚未提供共享分片暂存。

### 8.3 删除与清理失败

文件删除先短事务软删除 metadata，再在事务外删除物理对象。物理删除失败时 metadata 已不可见，再次通过同一受控删除路径可重试清理。当前未发现独立后台清理 Worker，运维需要关注 File 错误日志和存储容量，不得直接删除数据库记录掩盖孤立对象。

## 9. Integration Worker 与 Sync Runner

### 9.1 Worker

`integration.worker` 负责领取和执行 `IntegrationExecution`：

| 配置 | 当前含义 |
| --- | --- |
| `enabled` | 是否启动 Worker |
| `worker_id` | 实例稳定身份；启用时必填，生产多实例必须唯一 |
| `poll_interval` | 轮询间隔，允许 1 秒至 5 分钟 |
| `claim_batch_size` | 单次领取数，1 至 32 |
| `instance_concurrency` | 实例并发，1 至 16 |
| `lease_recovery_interval` | 过期 Lease 恢复间隔，10 秒至 5 分钟 |
| `lease_duration` | Execution Lease 时长 |
| `shutdown_timeout` | 停机等待，1 至 60 秒 |

### 9.2 Sync Runner

`integration.sync_runner` 负责按 Cron 建立 SyncBatch、切片和协调 Checkpoint：

| 配置 | 当前含义 |
| --- | --- |
| `enabled` | 是否启动 Runner |
| `runner_id` | 实例稳定身份；启用时必填，生产多实例必须唯一 |
| `poll_interval` | 轮询间隔，1 秒至 5 分钟 |
| `schedule_batch_size` | 单次调度批量，1 至 64 |
| `coordinate_batch_size` | 单次协调批量，1 至 64 |
| `shutdown_timeout` | 停机等待，1 至 60 秒 |

页面目前提供 Worker 状态能力，没有等价的 Sync Runner 状态页面。Runner/Worker 的启用来自服务端配置，不是页面按钮。手工运行 SyncTask 同样需要 Runner 创建/协调批次，并需要 Worker 执行具体请求。

## 10. Integration 排错

按固定顺序查看，避免把业务错误当网络重试：

```text
Worker / Runner 配置
  -> SyncTask / Checkpoint
  -> SyncBatch
  -> Execution
  -> Attempt / IntegrationLog
  -> Credential / Interface / RetryPolicy
  -> Consumer 业务结果
```

| 现象 | 常见原因 | 去哪看 |
| --- | --- | --- |
| SyncBatch 未创建或长期 pending | Runner 未启用、Cron/Timezone、任务停用 | 服务配置、SyncTask、应用日志 |
| Execution 长期 pending | Worker 未启用、worker_id 冲突或无可用实例 | Worker 状态、配置、应用日志 |
| 401/403 | Credential 停用、吊销或外部权限变化 | Credential 状态、Attempt；秘密不可回读 |
| 429/503 | 外部限流/暂时故障 | Attempt、RetryPolicy、Retry-After |
| 超时或地址拒绝 | 接口超时、HTTPS/SSRF 安全校验 | InterfaceDefinition、IntegrationLog |
| HTTP 200 但 Execution 失败 | Consumer 业务校验失败 | Consumer result、业务 Batch/Record |
| Checkpoint 不推进 | Slice 失败、Retry 尚未完成或业务失败 | SyncBatch、Execution、业务结果 |
| response too large | Body 超 Interface/Consumer/Transport 限制 | Execution reason、接口响应策略 |

业务处理失败不会进入 Integration Retry。Transport 上限为 64 MiB，Consumer 可设置更低限制；不要提高上限、保存 Response Artifact 或落地临时 Payload 绕过失败。

## 11. Auth 排错

认证链统一执行 Credential、账号状态、安全策略、Token、Login State 和 Audit。Password、SMS、DingTalk 只是不同 Credential 来源；AppToken 是应用身份，不是用户登录身份。

### 登录失败/锁定

检查用户 enabled/deleted/locked、初始密码/密码状态、Redis、验证码 Provider 和 Auth/Login Audit。错误密码会参与账号失败计数；短信 Challenge 有自己的尝试限制，不应机械等同密码锁定计数。需要解锁时使用有权限的用户解锁功能，不要直接改数据库或删除 Redis key。

### Refresh 失败

检查用户是否停用或锁定、密码是否更新、Refresh 是否已单次消费、Session 是否被 Logout 撤销、Redis 是否可用。停用用户和已撤销 Session 不能通过 Refresh 继续活跃。

### Logout

Logout 会更新 Login State 并撤销相关 Token。不要因为 JWT 可离线解析就跳过 Logout；若 Logout/Refresh 竞争，查看 Auth Audit 和 Token 状态，不签发补偿 Token。

## 12. Organization 与 Data Permission 排错

用户“能进页面但看不到数据”时依次检查：

1. 菜单和按钮功能权限是否存在。
2. 后端 API 是否有 Casbin policy，当前账号是否获得该 policy。
3. SysUser 是否绑定正确 Employee；Employee 与 User 不是同一个对象。
4. Resolver 的 Subject 是否包含预期 Employee/Organization 身份。
5. Resource 和 Operation 是否匹配当前请求。
6. Ownership Field 是否指向正确的 Metadata/registered field。
7. Dimension、Policy、Rule、Grant 是否启用且范围正确。
8. Metadata 表/字段状态与稳定 `table_code + field_code` 是否一致。
9. 业务查询是否真正通过 Resolver/Adapter 应用了 `DataScopeResult`。

不要通过硬编码管理员角色、扩大 Grant 或取消过滤来临时“修好”。Organization 业务归属引用 `org_unit_id`，不要把 `structure_node_id` 当稳定组织 ID。

## 13. 日志与审计如何区分

| 记录 | 用途 | 不应包含 |
| --- | --- | --- |
| AccessLog | HTTP 请求、响应状态、耗时和安全摘要 | Token、Credential、完整 Body |
| Auth/Login Audit | 认证渠道、成功/失败、稳定原因 | 密码、验证码、原始未知用户名 |
| Business Audit | 谁对业务对象执行了什么写操作 | 敏感 Payload、物理路径 |
| IntegrationLog / Attempt | 外部调用的技术尝试事实 | Credential、Authorization、Response Body |
| OrgSyncRecord | HR 业务对象的 action/status/reason | 姓名、手机、邮箱、原始 Source ID |

排错时先选正确记录。HTTP 503 应查看 Integration Attempt；HTTP 200 后的组织引用失败应查看 OrgSyncRecord；登录锁定应查看 Auth/Login Audit。

## 14. 运行前检查与只读基础可用性测试

本地Docker运行检查：

```bash
make docker-up
docker compose ps
curl -fsS http://127.0.0.1:9009/healthz
curl -fsS http://127.0.0.1:9009/readyz
```

检查外部部署配置：

```bash
node scripts/preflight-external.mjs template-check .env.external.example
node scripts/preflight-external.mjs .env.external
```

只读基础可用性测试会检查health/readiness、公开配置脱敏、登录、菜单、Metadata、Application脱敏和Audit查询：

```bash
SWEET_ADMIN_EXTERNAL_ENV_FILE=.env.external node scripts/smoke-readonly.mjs
```

若启用验证码，第一次运行可通过`SWEET_ADMIN_SMOKE_CAPTCHA_IMAGE_FILE`保存本次图片，再显式提供配对的captcha id/code重跑。测试账号应使用专门的受控管理员账号，不要复用个人密码。

## 15. 测试与发布验证矩阵

| 场景 | 命令 |
| --- | --- |
| 后端全量 | `cd backend && go test ./... -count=1` |
| 后端 Race | `cd backend && go test -race ./... -count=1` |
| PostgreSQL 强制 | `cd backend && SWEET_REQUIRE_POSTGRES_TESTS=true SWEET_TEST_POSTGRES_DSN='<dsn>' go test ./... -count=1` |
| 前端测试 | `cd frontend && yarn test` |
| 前端静态检查 | `cd frontend && yarn lint && yarn typecheck` |
| 前端构建 | `cd frontend && yarn build` |
| 文档 | `make docs-check` |
| Git tracked secret/static scan | `make secret-scan` |
| 基础聚合 | `make verify` |
| Pull Request日常检查 | `SWEET_TEST_POSTGRES_DSN='postgres://<user>:<password>@<host>:<port>/<database>?sslmode=<mode>' make ci-check` |
| 发布前完整检查 | `SWEET_TEST_POSTGRES_DSN='postgres://<user>:<password>@<host>:<port>/<database>?sslmode=<mode>' make release-check` |

`make verify`是本地快速验证，不包含前端Vitest、Race和强制PostgreSQL测试。Pull Request通过`.github/workflows/ci.yml`调用`make ci-check`；main/master通过`.github/workflows/release.yml`调用`make release-check`。`release-check`先完成全部日常检查，再增加后端`count=3`和全仓Race。两个workflow复用`.github/workflows/shared-checks.yml`读取根`.nvmrc`并准备PostgreSQL 16与Redis，具体命令只在Makefile维护。缺少PostgreSQL DSN或URL格式错误时，检查会直接失败。

## 16. 发布 Checklist

### 发布前

- [ ] 工作区干净，发布 Commit/Tag 已确定。
- [ ] 后端全量测试和 Race 通过。
- [ ] PostgreSQL 16 强制测试实际执行，未跳过。
- [ ] 前端 test、lint、typecheck、build 全部通过。
- [ ] `make docs-check` 通过。
- [ ] `make secret-scan` 通过，DSN、token、私钥和 production secret fallback 未进入 tracked 文件。
- [ ] Migration/Seed 在等价环境验证幂等。
- [ ] `.env.external` 通过运行前检查，权限为600，秘密未进入Git或日志。
- [ ] 数据库备份、manifest 和恢复演练证据可用。
- [ ] CORS、上传 allowlist、Casbin coverage 和 `public_preview=false` 已确认。
- [ ] 多实例 Worker/Runner ID 唯一。
- [ ] Local upload 持久卷或 OSS、Chunk 会话粘性已确认。

### 发布中

- [ ] 先备份，再显式执行 Migration/Seed。
- [ ] 启动 backend，等待 `/readyz`。
- [ ] 启动/切换 frontend，检查静态资源基路径。
- [ ] 观察数据库、Redis、错误日志和 Integration Worker。

### 发布后

- [ ] 执行只读基础可用性测试。
- [ ] 用有权限和无权限账号检查关键页面/API。
- [ ] 检查 Access/Auth/Integration Audit 是否正常且无秘密。
- [ ] 检查 Runner/Worker 和 Checkpoint 是否按预期运行。
- [ ] 保存测试、Migration、备份和发布证据。

## 17. 当前运维限制与后续项

这些是当前真实限制，不应靠运行手册绕过：

- File Chunk 使用节点本地暂存，多实例需要会话粘性，尚无共享暂存。
- 文件物理清理失败没有独立后台重试 Worker，需要受控重试和监控。
- 只有 Worker 状态 API/页面能力，没有同等 Sync Runner 状态页面。
- `make verify` 仍定位为日常快速检查；发布必须使用要求真实 PostgreSQL DSN 的 `make release-check`。
- 当前没有平台级 Prometheus/OpenTelemetry 指标端点；运行观测主要依赖 health/readiness、结构化日志和业务审计。

这些限制必须在部署方案中明确，不能通过关闭安全检查或扩大权限绕过。
