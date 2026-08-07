# 部署在线协作文档服务

本指南说明如何校验 Compose 配置、准备生产环境变量、启动基础服务和备份数据。生产环境的 Office 编辑器需要单独完成端到端验证。

## 准备生产环境变量

从模板创建受控的生产环境文件：

```bash
cp deploy/env/app.env.example deploy/env/production.env
```

不要把 `production.env` 提交到 Git。部署前替换模板中的数据库密码、S3 密钥、密码 pepper、OIDC 密钥和 Casual JWT 密钥。

以下变量必须与公网入口一致：

| 变量 | 作用 |
| --- | --- |
| `PUBLIC_APP_URL` | 用户在浏览器中访问的 HTTPS 地址 |
| `PUBLIC_API_URL` | backend 在浏览器和编辑器回调中使用的 HTTPS 地址 |
| `FRONTEND_ORIGIN` | 允许携带会话 cookie 的前端来源 |
| `OIDC_REDIRECT_URL` | 身份提供方登记的 callback 地址 |
| `NGINX_HTTP_PORT` | Nginx 映射到宿主机的端口 |
| `DOCUMENT_MAX_UPLOAD_BYTES` | backend 上传限制，默认 50 MB |

生产 compose 将 `SESSION_SECRET` 传给 backend 的 `PASSWORD_HASH_PEPPER`，并将 `MAX_UPLOAD_BYTES` 传给 `DOCUMENT_MAX_UPLOAD_BYTES`。模板只保留这两个部署入口，避免同时维护别名。

## 校验并启动基础服务

生产 compose 使用 Nginx 作为统一入口，并创建 PostgreSQL、Redis 和 MinIO named volumes。先渲染配置：

```bash
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml config
```

确认输出不含模板占位值后启动：

```bash
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml up --build -d
```

Nginx 默认映射到 `8088`。生产通常应设置 `NGINX_HTTP_PORT=80` 或由外部 TLS 终止代理转发到该端口。

## 验证部署

启动后检查服务和 backend 依赖：

```bash
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml ps
curl -fsS http://localhost:8088/api/healthz
curl -fsS http://localhost:8088/api/readyz
```

`readyz` 只有在 PostgreSQL、Redis 和对象存储都可用时才会返回成功。通过 HTTPS 对外提供服务时，使用实际域名替换本地地址。

## 生产环境中的 Office 编辑器

当前 `deploy/docker-compose.prod.yml` 只包含 Casual Sheets 协作服务，未创建开发 compose 中的 Casual Docs、Sheets gateway 或 Docs gateway。因此它不是 docx/xlsx Office 编辑器的完整生产编排。

在向外启用 Office 编辑前，你必须提供并验证以下内容：

1. 可从浏览器访问的 Casual Docs 和 Casual Sheets 地址
2. backend 使用的对应内部 WebSocket 地址
3. `CASUAL_JWT_SECRET`、`CASUAL_DOCS_EDITOR_URL`、`CASUAL_SHEETS_EDITOR_URL`、`CASUAL_DOCS_INTERNAL_WS_URL` 和 `CASUAL_SHEETS_INTERNAL_WS_URL`
4. Nginx 或其他反向代理中的 WebSocket upgrade、长连接 timeout 和编辑器回调路由
5. docx 与 xlsx 的打开、编辑、保存、权限和版本回归

在这些验证完成前，不要把生产 compose 视为 Office 编辑器可用的部署方案。Markdown 编辑、账户、文档存储、分享、版本与后台能力不依赖这组 Office 网关。

## 数据持久化和备份

生产 compose 使用以下 named volumes：

- **`postgres-data`**: 用户、文档元数据、权限、版本和审计日志
- **`redis-data`**: Redis append-only 数据
- **`minio-data`**: 文档对象内容

备份应同时覆盖数据库、对象存储和受控环境变量：

1. 定期执行 `pg_dump` 并验证可恢复
2. 备份 MinIO bucket 或其底层 volume
3. 将生产环境变量保存在受控 secret manager
4. 在隔离环境中恢复数据库和对象存储，再验证文档下载与版本恢复

## OIDC 配置

将 `OIDC_ENABLED` 设为 `true`，并配置 issuer、client ID、client secret 和 redirect URL。issuer 必须能提供 discovery、JWKS 和 userinfo 端点。不要启用开发用 mock OIDC profile 作为生产身份提供方。

## 排查部署问题

- **登录后没有会话**: 检查 HTTPS、`PUBLIC_APP_URL`、`FRONTEND_ORIGIN` 和 cookie 域名
- **上传失败**: 使 Nginx `client_max_body_size` 与 `DOCUMENT_MAX_UPLOAD_BYTES` 保持一致
- **Markdown WebSocket 断开**: 确认 `/api/` 保留 Upgrade 与 Connection 请求头
- **重启后文件缺失**: 确认未删除 PostgreSQL 或 MinIO volume
- **Office 无法打开**: 按“生产环境中的 Office 编辑器”逐项验证地址、JWT、网关和保存回调
