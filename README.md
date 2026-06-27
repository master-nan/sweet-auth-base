# Sweet Admin

Sweet Admin 是一个通用后台底座，包含权限、菜单、配置、字典、审计日志、文件上传和低代码 CRUD 能力。后端使用 Go/Gin/Gorm，前端使用 Vue 3 + Quasar，默认数据库为 PostgreSQL，缓存为 Redis。

详细运行、测试、Docker 发布和数据库说明见 [docs/RUNBOOK.md](docs/RUNBOOK.md)。

低代码发布、字段配置、按钮动作、参数 Schema、关联下拉和权限排查见 [docs/LOW_CODE_MANUAL.md](docs/LOW_CODE_MANUAL.md)。

通用数据权限模型和可手测 Demo 见 [docs/data-permission-design.md](docs/data-permission-design.md) 与 [docs/DATA_PERMISSION_DEMO.md](docs/DATA_PERMISSION_DEMO.md)。

## 快速启动

启动完整 Docker 本地环境：

```bash
make docker-up
```

默认访问：

- 前端：http://localhost:8008/sweet_admin
- 后端：http://localhost:9009/sweet_admin
- Swagger：http://localhost:9009/swagger/index.html
- PostgreSQL：localhost:15432
- Redis：localhost:16380

如果本机已有服务占用默认端口，可以临时覆盖：

```bash
SWEET_ADMIN_FRONTEND_PORT=18008 \
SWEET_ADMIN_BACKEND_PORT=19009 \
SWEET_ADMIN_POSTGRES_PORT=15433 \
SWEET_ADMIN_REDIS_PORT=16381 \
make docker-up
```

默认账号：

```text
账号：admin
密码：admin123
```

`admin123` 只用于本地初始化。连接已有数据库、预发或生产环境时，必须通过 `APP_BOOTSTRAP_ADMIN_PASSWORD` 设置强密码，并在首次登录后立即修改管理员密码。

## 本地开发

后端：

```bash
cd backend
go mod tidy
go run main.go
```

前端：

```bash
cd frontend
yarn
quasar dev
```

前端本地开发默认读取 [frontend/.env.development](frontend/.env.development)，后端本地开发默认读取 [backend/config-dev.yaml](backend/config-dev.yaml)。

## 常用命令

```bash
make help
make backend-test
make frontend-ci
make verify
make docker-up
make docker-rebuild-frontend
make docker-rebuild-backend
make docker-down
make docker-logs
```

## Docker 环境

[docker-compose.yml](docker-compose.yml) 用于本地完整验证，会启动 PostgreSQL、Redis、backend、frontend。

[docker-compose.external.yml](docker-compose.external.yml) 用于连接已有 PostgreSQL/Redis，只启动 backend、frontend。第一次使用时：

```bash
cp .env.external.example .env.external
make docker-up-external
```

如果要连接容器里的 PostgreSQL/Redis，使用 `make docker-up`。如果要连接你本机或其他机器已有的 PostgreSQL/Redis，修改 `.env.external` 后使用 `make docker-up-external`。

## 数据库

本项目默认使用 PostgreSQL。初始化分为两类：

- `make db-migrate`：创建或补齐基础表结构
- `make db-seed`：初始化内置菜单、按钮、角色、字典和系统关系

本地 Docker 环境默认会在 backend 启动前执行迁移和初始化。连接已有数据库时，默认不自动改库，需要你按版本说明手动执行。

## 生产配置

生产/预发配置模板在 [backend/config-pro.yaml](backend/config-pro.yaml)。真实数据库密码、Redis 密码、JWT 密钥、OSS 密钥、初始化管理员密码等不要提交到 Git，建议通过环境变量、平台 Secret 或私有 `.env.external` 注入。

## 文档

- [运行手册](docs/RUNBOOK.md)
- [低代码配置手册](docs/LOW_CODE_MANUAL.md)
- [通用数据权限 Demo](docs/DATA_PERMISSION_DEMO.md)
- [字段类型说明](docs/field-type-guide.md)
- [联动配置说明](docs/linkage_config.md)
