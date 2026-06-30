# Sweet Admin 操作手册

忘记命令时先看：

```bash
make help
```

低代码发布、字段配置、按钮动作、参数 Schema、关联下拉、权限排查和常用模板见 [low-code-manual.md](low-code-manual.md)。

## 1. 本地开发

后端本地运行：

```bash
cd backend
go run main.go
```

前端本地运行：

```bash
cd frontend
yarn
quasar dev
```

本地后端读取 [backend/config-dev.yaml](../backend/config-dev.yaml)，本地前端读取 [frontend/.env.development](../frontend/.env.development)。

## 2. 测试命令

改后端：

```bash
make backend-test
```

改前端：

```bash
make frontend-ci
```

不确定影响范围：

```bash
make verify
```

脚本级检查：

```bash
node --test scripts/*.test.mjs
```

Docker 环境启动后做只读 smoke：

```bash
source ~/.nvm/nvm.sh && nvm use 22
node scripts/smoke-readonly.mjs
```

## 3. 使用容器 PostgreSQL/Redis

本地完整 Docker 环境：

```bash
make docker-up
```

访问地址：

- 前端：http://localhost:8008/sweet_admin
- 后端：http://localhost:9009/sweet_admin
- Swagger：http://localhost:9009/swagger/index.html
- PostgreSQL：localhost:15432
- Redis：localhost:16380

查看日志：

```bash
make docker-logs
```

停止环境：

```bash
make docker-down
```

## 4. 连接已有 PostgreSQL/Redis

复制外部依赖配置：

```bash
cp .env.external.example .env.external
```

按你的真实地址修改 `.env.external`，然后启动：

```bash
make docker-up-external
```

`docker-compose.external.yml` 不创建 PostgreSQL/Redis 容器，只启动 backend 和 frontend。

外部依赖模式默认不自动迁移、不自动 seed。首次接入已有库前先备份，再按需要显式开启或手动执行：

```bash
make db-migrate
make db-seed
```

`.env.external` 里的 `APP_BOOTSTRAP_ADMIN_PASSWORD`、`APP_SESSION_SECRET`、`APP_CONF_SALT` 必须替换成环境专用值，不要沿用示例占位。

## 5. 修改代码后如何发布到本地测试

只改前端：

```bash
make docker-rebuild-frontend
```

只改后端：

```bash
make docker-rebuild-backend
```

前后端都改：

```bash
make docker-up
```

连接已有 PostgreSQL/Redis 时，对应使用：

```bash
make docker-rebuild-frontend-external
make docker-rebuild-backend-external
make docker-up-external
```

## 6. 数据库变更

迁移结构：

```bash
make db-migrate
```

初始化内置数据：

```bash
make db-seed
```

本地完整 Docker 默认会自动执行迁移和初始化。连接已有数据库时默认不自动改库，发布前先备份，再按版本说明决定是否执行 `db-migrate` 或 `db-seed`。

## 7. 配置文件说明

[backend/config-dev.yaml](../backend/config-dev.yaml)：本地 `go run main.go` 使用。

[backend/config-docker.yaml](../backend/config-docker.yaml)：本地 Docker Compose 使用，数据库 host 是 compose 服务名 `postgres`、`redis`。

[backend/config-pro.yaml](../backend/config-pro.yaml)：生产/预发模板，只放占位值，不提交真实密码和密钥。

[.env.external.example](../.env.external.example)：连接已有 PostgreSQL/Redis 的环境变量模板。

`.env.external`：本机真实覆盖配置，不提交到 Git。

[docker-compose.yml](../docker-compose.yml)：完整本地环境，包含 PostgreSQL、Redis、backend、frontend。

[docker-compose.external.yml](../docker-compose.external.yml)：外部依赖环境，只启动 backend、frontend。

## 8. 正式发布建议

1. 准备生产 PostgreSQL 和 Redis。
2. 按 [backend/config-pro.yaml](../backend/config-pro.yaml) 配置真实环境变量或平台 Secret。
3. 发布前先备份数据库。
4. 明确本次版本是否需要执行 `db-migrate` 和 `db-seed`。
5. 构建前端时确认 `APP_BASE_PATH`，默认是 `/sweet_admin`。
6. 部署后检查 `/healthz`、`/readyz`、登录、菜单、低代码列表、文件上传和审计日志。

## 9. 常见场景速查

| 场景 | 命令 |
| --- | --- |
| 查看命令 | `make help` |
| 后端本地开发 | `cd backend && go run main.go` |
| 前端本地开发 | `cd frontend && quasar dev` |
| 后端测试 | `make backend-test` |
| 前端检查 | `make frontend-ci` |
| 全量检查 | `make verify` |
| 本地 Docker 启动 | `make docker-up` |
| 只发布前端到 Docker | `make docker-rebuild-frontend` |
| 只发布后端到 Docker | `make docker-rebuild-backend` |
| 外部依赖启动 | `make docker-up-external` |
| 数据库迁移 | `make db-migrate` |
| 初始化内置数据 | `make db-seed` |
