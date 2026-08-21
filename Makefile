.PHONY: help verify release-check secret-scan docs-check scripts-test backend-test frontend-ci external-preflight db-migrate db-seed db-migrate-external db-seed-external docker-build-backend-assets docker-build-frontend-assets docker-build-assets docker-up docker-rebuild-backend docker-rebuild-frontend docker-up-external docker-rebuild-backend-external docker-rebuild-frontend-external docker-down docker-logs

APP_BASE_PATH ?= /sweet_admin
EXTERNAL_ENV_FILE ?= .env.external
EXTERNAL_DOCKER_COMPOSE = docker compose --env-file "$(EXTERNAL_ENV_FILE)" -f docker-compose.external.yml
EXTERNAL_PREFLIGHT = SWEET_ADMIN_PREFLIGHT_REQUIRE_STARTUP_WRITES_DISABLED=true node scripts/preflight-external.mjs "$(EXTERNAL_ENV_FILE)"

help:
	@printf '%s\n' 'Sweet Admin 常用命令'
	@printf '%s\n' ''
	@printf '%s\n' '本地开发：'
	@printf '%s\n' '  cd backend && go run main.go              启动后端，读取 backend/config-dev.yaml'
	@printf '%s\n' '  cd frontend && yarn && quasar dev         启动前端开发服务'
	@printf '%s\n' ''
	@printf '%s\n' '测试验证：'
	@printf '%s\n' '  make docs-check                           检查文档目录、旧路径和相对链接'
	@printf '%s\n' '  make secret-scan                          扫描 Git tracked 文件中的静态凭据'
	@printf '%s\n' '  make scripts-test                        运行运维脚本 Node 原生测试'
	@printf '%s\n' '  make backend-test                         只跑 Go 测试'
	@printf '%s\n' '  make frontend-ci                          只跑前端 lint/typecheck/build'
	@printf '%s\n' '  make verify                               快速验证，不含 Race/PostgreSQL/Vitest'
	@printf '%s\n' '  SWEET_TEST_POSTGRES_DSN=postgres://... make release-check'
	@printf '%s\n' '                                            完整发布门禁，强制真实 PostgreSQL'
	@printf '%s\n' ''
	@printf '%s\n' '数据库：'
	@printf '%s\n' '  make db-migrate                           执行结构迁移'
	@printf '%s\n' '  make db-seed                              补基础菜单、按钮、角色等数据'
	@printf '%s\n' '  make db-migrate-external                  对 .env.external 指向的外部库执行结构迁移'
	@printf '%s\n' '  make db-seed-external                     对 .env.external 指向的外部库补基础数据'
	@printf '%s\n' ''
	@printf '%s\n' 'Docker 本地完整环境：'
	@printf '%s\n' '  make docker-up                            启动 PostgreSQL、Redis、backend、frontend'
	@printf '%s\n' '  make docker-rebuild-backend               只重建 backend'
	@printf '%s\n' '  make docker-rebuild-frontend              只重建 frontend'
	@printf '%s\n' '  make docker-down                          停止容器'
	@printf '%s\n' '  make docker-logs                          查看日志'
	@printf '%s\n' ''
	@printf '%s\n' 'Docker 连接宿主机 PostgreSQL/Redis：'
	@printf '%s\n' '  cp .env.external.example .env.external    首次复制外部环境配置'
	@printf '%s\n' '  make external-preflight                  验证外部部署配置且禁止启动期写库'
	@printf '%s\n' '  make docker-up-external                   只启动 backend、frontend'
	@printf '%s\n' ''
	@printf '%s\n' '详细说明见 docs/operations/PlatformOperationsGuide.md'

verify: docs-check backend-test frontend-ci

release-check:
	@test -n "$${SWEET_TEST_POSTGRES_DSN}" || (printf '%s\n' 'SWEET_TEST_POSTGRES_DSN is required for release-check' >&2; exit 1)
	@case "$${SWEET_TEST_POSTGRES_DSN}" in postgres://*|postgresql://*) ;; *) printf '%s\n' 'SWEET_TEST_POSTGRES_DSN must be a postgres:// or postgresql:// URL' >&2; exit 1;; esac
	$(MAKE) secret-scan
	$(MAKE) docs-check
	$(MAKE) scripts-test
	@cd backend && SWEET_REQUIRE_POSTGRES_TESTS=true go test ./... -count=1
	@cd backend && SWEET_REQUIRE_POSTGRES_TESTS=true go test -p=1 ./... -count=3
	@cd backend && SWEET_REQUIRE_POSTGRES_TESTS=true go test -race -p=1 ./... -count=1
	cd frontend && yarn quasar prepare && yarn test
	$(MAKE) frontend-ci

secret-scan:
	node scripts/check-tracked-secrets.mjs

docs-check:
	python3 scripts/check_docs.py

scripts-test:
	node --test scripts/*.test.mjs

backend-test:
	cd backend && go test ./...

frontend-ci:
	cd frontend && VITE_API_URL=$(APP_BASE_PATH) VITE_PUBLIC_PATH=$(APP_BASE_PATH) yarn ci

external-preflight:
	$(EXTERNAL_PREFLIGHT)

db-migrate:
	cd backend && go run ./migrate

db-seed:
	cd backend && go run ./migrate seed

db-migrate-external:
	$(EXTERNAL_PREFLIGHT) migration
	$(MAKE) docker-build-backend-assets
	$(EXTERNAL_DOCKER_COMPOSE) run --rm --no-deps backend /app/migrate

db-seed-external:
	$(EXTERNAL_PREFLIGHT) seed
	$(MAKE) docker-build-backend-assets
	$(EXTERNAL_DOCKER_COMPOSE) run --rm --no-deps backend /app/migrate seed

docker-build-backend-assets:
	mkdir -p backend/bin frontend/bin
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/sweet_admin main.go
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/migrate ./migrate
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/db-preflight ./cmd/db-preflight
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/container-entrypoint ./cmd/container-entrypoint
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/healthcheck ./cmd/healthcheck

docker-build-frontend-assets:
	mkdir -p frontend/bin
	cd frontend && VITE_API_URL=$(APP_BASE_PATH) VITE_PUBLIC_PATH=$(APP_BASE_PATH) yarn build
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../frontend/bin/static-server ./cmd/static-server
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../frontend/bin/healthcheck ./cmd/healthcheck

docker-build-assets:
	$(MAKE) docker-build-backend-assets
	$(MAKE) docker-build-frontend-assets

docker-up:
	$(MAKE) docker-build-assets
	docker compose up -d --build
	docker compose ps

docker-up-external:
	$(EXTERNAL_PREFLIGHT)
	$(MAKE) docker-build-assets
	$(EXTERNAL_DOCKER_COMPOSE) up -d --build
	$(EXTERNAL_DOCKER_COMPOSE) ps

docker-rebuild-backend: docker-build-backend-assets
	docker compose up -d --build backend
	docker compose ps backend

docker-rebuild-frontend: docker-build-frontend-assets
	docker compose up -d --build frontend
	docker compose ps frontend

docker-rebuild-backend-external:
	$(EXTERNAL_PREFLIGHT)
	$(MAKE) docker-build-backend-assets
	$(EXTERNAL_DOCKER_COMPOSE) up -d --build backend
	$(EXTERNAL_DOCKER_COMPOSE) ps backend

docker-rebuild-frontend-external:
	$(EXTERNAL_PREFLIGHT)
	$(MAKE) docker-build-frontend-assets
	$(EXTERNAL_DOCKER_COMPOSE) up -d --build frontend
	$(EXTERNAL_DOCKER_COMPOSE) ps frontend

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
