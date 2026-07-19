# Docs Collab Service

Docs Collab Service 是一个面向私有化部署的在线文档共享编辑服务。当前完成到 M6 Markdown 普通编辑。尚未实现 Markdown 协同。

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
- ONLYOFFICE Document Server: http://localhost:8081

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
OIDC_ENABLED=false
OIDC_ISSUER_URL=https://idp.example.com
OIDC_CLIENT_ID=docs-collab
OIDC_CLIENT_SECRET=change-me
OIDC_REDIRECT_URL=http://localhost:8080/api/auth/oidc/callback
OIDC_SCOPES=openid,email,profile
OIDC_AUTO_MERGE_BY_EMAIL=false
DOCUMENT_MAX_UPLOAD_BYTES=52428800
ONLYOFFICE_ENABLED=true
ONLYOFFICE_PUBLIC_URL=http://localhost:8081
ONLYOFFICE_INTERNAL_URL=http://onlyoffice
ONLYOFFICE_JWT_SECRET=change-me
PUBLIC_APP_URL=http://localhost:3000
PUBLIC_API_URL=http://localhost:8080
BACKEND_INTERNAL_URL=http://backend:8080
MARKDOWN_SNAPSHOT_UPDATE_INTERVAL=100
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

M7 Markdown 协同编辑：

- users / sessions 数据库 migration
- 本地用户注册、登录、登出、当前用户接口
- HttpOnly session cookie，服务端仅保存 token hash
- OIDC 登录跳转、callback、id_token 校验和 userinfo 获取
- OIDC 用户自动创建，默认不按 email 合并本地用户
- documents / document_versions 数据库 migration
- 文档上传、列表、详情、下载、软删除和版本列表
- MinIO / S3 storage 抽象与对象存储实现
- 前端 `/documents` 和 `/documents/:id` 页面
- document_permissions 数据库 migration
- owner / editor / viewer 权限矩阵
- 文档接口统一接入 PermissionService
- 前端 `/documents/:id/permissions` 权限管理页面
- ONLYOFFICE Document Server compose 服务
- doc/docx/xls/xlsx 编辑器 config 生成
- viewer 只读、editor/owner 可编辑
- ONLYOFFICE 保存回调生成新版本
- Markdown 文档读取接口 `GET /api/documents/:id/markdown`
- Markdown 文档保存接口 `PUT /api/documents/:id/markdown`
- Markdown 保存生成新版本并更新当前下载内容
- 前端 `/documents/:id/markdown` 源码编辑和预览页面
- viewer 只读打开 Markdown，editor/owner 可以保存
- Markdown 协同 snapshot 接口 `GET /api/documents/:id/markdown/snapshot`
- Markdown 协同 WebSocket `WS /api/documents/:id/markdown/ws`
- editor/owner 修改内容实时广播到同一文档其他客户端
- viewer 可以连接和接收更新，但不能提交编辑
- Markdown update 和周期 snapshot 持久化到 PostgreSQL
- presence 显示当前在线用户和只读/可编辑状态

## 当前限制

- Markdown 协同第一版只支持单 backend 实例内实时广播；多实例部署需要 Redis pub/sub 或其他跨实例消息总线。

## 下一步开发计划

M8 将实现分享链接。
