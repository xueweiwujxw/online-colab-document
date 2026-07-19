# Deployment

## 开发环境

启动默认开发栈：

```bash
podman compose -f deploy/docker-compose.yml up --build
```

默认入口：

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`
- MinIO Console: `http://localhost:9001`
- ONLYOFFICE: `http://localhost:8081`

可选 nginx 统一入口：

```bash
podman compose -f deploy/docker-compose.yml --profile proxy up --build
```

统一入口为 `http://localhost:8088`，`/api/` 代理到 backend，`/onlyoffice/` 代理到 ONLYOFFICE。

可选 mock OIDC 开发 profile：

```bash
OIDC_ENABLED=true podman compose -f deploy/docker-compose.yml --profile oidc up --build
```

mock OIDC 服务默认在 `http://localhost:8082`，会自动签发固定测试用户：

- Email: `oidc-user@example.com`
- Display name: `OIDC 测试用户`

它模拟标准 discovery、authorization code、token、JWKS 和 userinfo 接口。用途仅限本地开发测试，不要用于生产。启用后点击登录页的 “使用 OIDC 登录” 会直接完成 mock OIDC 授权并回到文档列表。

## 生产环境

生产 compose 提供完整服务：backend、frontend、postgres、redis、minio、onlyoffice、nginx。

```bash
podman compose -f deploy/docker-compose.prod.yml config
podman compose -f deploy/docker-compose.prod.yml up --build -d
```

默认 nginx 入口为 `http://localhost:8088`。生产部署时应通过环境变量设置公网地址，例如：

```bash
PUBLIC_APP_URL=https://docs.example.com
PUBLIC_API_URL=https://docs.example.com
ONLYOFFICE_PUBLIC_URL=https://docs.example.com/onlyoffice
NGINX_HTTP_PORT=80
```

## 环境变量

完整样例在 `deploy/env/app.env.example`。关键变量：

- `APP_ENV=production` 会启用 Secure cookie。
- `FRONTEND_ORIGIN` 控制 CORS 允许来源。
- `SESSION_SECRET` 作为密码 hash pepper 的兼容配置；也可继续使用 `PASSWORD_HASH_PEPPER`。
- `MAX_UPLOAD_BYTES` 是上传上限别名；也可继续使用 `DOCUMENT_MAX_UPLOAD_BYTES`。
- `S3_ENDPOINT`、`S3_ACCESS_KEY`、`S3_SECRET_KEY`、`S3_BUCKET` 配置 MinIO/S3。
- `ONLYOFFICE_JWT_SECRET` 必须和 ONLYOFFICE Document Server 的 `JWT_SECRET` 一致。

不要把生产 `.env`、真实 password、token、secret 或私钥提交到 Git。

## 数据持久化

生产 compose 使用 named volumes：

- `postgres-data`: PostgreSQL 元数据、用户、权限、版本、审计。
- `redis-data`: Redis append-only 数据。
- `minio-data`: 文档对象内容。

备份建议：

- 定期 `pg_dump` PostgreSQL。
- 定期备份 MinIO bucket 或底层 volume。
- 保存生产环境变量和 ONLYOFFICE JWT secret 到受控 secret manager。
- 恢复演练必须同时验证数据库、对象存储和 ONLYOFFICE 保存回调。

## Migrations

backend 启动时会自动执行 `backend/migrations`。镜像构建会把 migrations 复制到 `/app/migrations`，也可以通过 `MIGRATIONS_DIR` 覆盖路径。

已有旧开发库没有 `schema_migrations` 表时，当前 SQL 使用 `IF NOT EXISTS`，会安全补建迁移记录。

## 常见问题

- 登录后 cookie 不生效：检查 `PUBLIC_APP_URL`、`FRONTEND_ORIGIN` 和是否通过 HTTPS 访问生产环境。
- WebSocket 连接失败：确认 nginx `/api/` location 保留 `Upgrade` 和 `Connection` 头。
- ONLYOFFICE 无法保存：确认 `BACKEND_INTERNAL_URL` 对 ONLYOFFICE 容器可达，且 `ONLYOFFICE_JWT_SECRET` 一致。
- 上传失败：确认 nginx `client_max_body_size` 和 backend `MAX_UPLOAD_BYTES` 一致。
- 重启后文件丢失：确认 MinIO 和 PostgreSQL volumes 没有被删除。
