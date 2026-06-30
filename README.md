# Sweet Admin

Sweet Admin 是一个通用后台底座，包含权限、菜单、配置、字典、审计日志、文件上传和低代码 CRUD 能力。后端使用 Go/Gin/Gorm，前端使用 Vue 3 + Quasar，默认数据库为 PostgreSQL，缓存为 Redis。

常用入口：

- 运行、测试、Docker、数据库和发布说明：[docs/runbook.md](docs/runbook.md)
- 低代码发布、字段、按钮和权限排查：[docs/low-code-manual.md](docs/low-code-manual.md)
- 字段输入类型：[docs/field-type-guide.md](docs/field-type-guide.md)
- 字段联动配置：[docs/linkage-config.md](docs/linkage-config.md)
- 通用数据权限模型与 Demo：[docs/data-permission-design.md](docs/data-permission-design.md)、[docs/data-permission-demo.md](docs/data-permission-demo.md)

## 快速启动

启动完整 Docker 本地环境：

```bash
make docker-up
```

默认访问：

- 前端：http://localhost:8008/sweet_admin
- 后端 API 前缀：http://localhost:9009/sweet_admin
- 后端健康检查：http://localhost:9009/healthz、http://localhost:9009/readyz
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

注意：`make docker-up` 暴露给宿主机的 PostgreSQL/Redis 端口是 `15432/16380`，而 `go run main.go` 默认按 [backend/config-dev.yaml](backend/config-dev.yaml) 连接 `5432/6379`。如果本地后端要复用 Docker 里的数据库和 Redis，需要通过环境变量覆盖端口。

## 文档

- [运行手册](docs/runbook.md)
- [低代码配置手册](docs/low-code-manual.md)
- [通用数据权限设计与实现说明](docs/data-permission-design.md)
- [通用数据权限 Demo](docs/data-permission-demo.md)
- [字段类型说明](docs/field-type-guide.md)
- [联动配置说明](docs/linkage-config.md)
