# Docs Collab Service

Docs Collab Service 是一个面向私有化部署的在线文档共享编辑服务。当前完成到 M1 本地用户认证。尚未实现 OIDC、文档上传下载、权限、ONLYOFFICE、Markdown 协同。

## 技术栈

- Backend: Go HTTP server
- Frontend: React + TypeScript + Vite
- Frontend package manager: pnpm
- Runtime for frontend tooling: Node.js 24
- Database: PostgreSQL
- Cache: Redis
- Storage: MinIO / S3
- Deploy: Podman Compose / Docker Compose compatible compose file
- Office editor: 后续里程碑使用 ONLYOFFICE Document Server
- Markdown collab: 后续里程碑使用 Yjs / WebSocket

## 本地启动

启动基础服务：

```bash
make dev
```

服务地址：

- Frontend: http://localhost:3000
- Backend health: http://localhost:8080/healthz
- Backend ready: http://localhost:8080/readyz
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- MinIO API: http://localhost:9000
- MinIO Console: http://localhost:9001

## 环境变量

后端：

```text
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgres://docs:docs@postgres:5432/docs?sslmode=disable
REDIS_ADDR=redis:6379
S3_ENDPOINT=http://minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=docs
S3_USE_SSL=false
FRONTEND_ORIGIN=http://localhost:3000
SESSION_COOKIE_NAME=docs_session
SESSION_TTL_HOURS=168
PASSWORD_HASH_PEPPER=
```

前端：

```text
VITE_API_BASE_URL=http://localhost:8080
```

## 常用命令

```bash
make dev
make up
make down
make logs
make backend-test
make frontend-build
make test
make lint
```

## 当前里程碑

M1 本地用户认证：

- users / sessions 数据库 migration
- 本地用户注册、登录、登出、当前用户接口
- HttpOnly session cookie，服务端仅保存 token hash
- 前端 `/login` 页面、auth context、受保护的文档列表占位页
- 后端认证测试和前端构建验证

## 下一步开发计划

M2 将实现 OIDC 登录，并保持已有本地登录流程不受影响。
