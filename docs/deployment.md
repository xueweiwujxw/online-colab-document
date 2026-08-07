# 部署在线协作文档服务

本指南说明如何校验 Compose 配置、准备生产环境变量、启动基础服务和备份数据。生产 compose 包含 Casual Docs、Casual Sheets 和两个编辑器网关。

## 准备生产环境变量

从模板创建受控的生产环境文件：

```bash
cp deploy/env/production.env.example deploy/env/production.env
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
| `NGINX_HTTPS_PORT` | 主站 Nginx 映射到宿主机的 TLS 端口 |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | 部署主机上的证书链与私钥绝对路径，只读挂载进三个公开网关 |
| `DOCUMENT_MAX_UPLOAD_BYTES` | backend 上传限制，默认 50 MB |
| `CASUAL_DOCS_EDITOR_URL` | 浏览器访问 Casual Docs gateway 的完整地址 |
| `CASUAL_SHEETS_EDITOR_URL` | 浏览器访问 Casual Sheets gateway 的完整地址 |
| `CASUAL_DOCS_HTTPS_PORT` | Casual Docs gateway 的宿主机 TLS 端口 |
| `CASUAL_SHEETS_HTTPS_PORT` | Casual Sheets gateway 的宿主机 TLS 端口 |

生产 compose 将 `SESSION_SECRET` 传给 backend 的 `PASSWORD_HASH_PEPPER`，并将 `MAX_UPLOAD_BYTES` 传给 `DOCUMENT_MAX_UPLOAD_BYTES`。模板只保留这两个部署入口，避免同时维护别名。

## 构建并发布镜像

生产 Compose **不包含 `build`**，部署主机只拉取经过 CI 或发布流程构建的不可变镜像。前端镜像在构建阶段执行 Vite 打包，最终层仅使用 Nginx 提供静态文件；backend 同样由其 Docker 镜像运行。

在 CI 或受控构建机中，为同一个版本号构建并推送四个应用镜像：

```bash
podman build -t registry.example.com/online-colab-document/backend:VERSION backend
podman build --build-arg VITE_API_BASE_URL='' \
  -t registry.example.com/online-colab-document/frontend:VERSION frontend
podman build -t registry.example.com/online-colab-document/casual-sheets:VERSION deploy/casual-sheets
podman build -t registry.example.com/online-colab-document/casual-docs:VERSION deploy/casual-docs
podman push registry.example.com/online-colab-document/backend:VERSION
podman push registry.example.com/online-colab-document/frontend:VERSION
podman push registry.example.com/online-colab-document/casual-sheets:VERSION
podman push registry.example.com/online-colab-document/casual-docs:VERSION
```

将这四个镜像的完整、固定版本标签填入 `production.env` 的 `*_IMAGE` 变量。不要使用浮动的 `latest` 标签；私有镜像仓库先在部署主机执行 `podman login`。

## 校验并启动基础服务

生产 compose 使用 Nginx 作为统一入口，并创建 PostgreSQL、Redis 和 MinIO named volumes。先渲染配置：

```bash
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml config
```

确认输出不含模板占位值与 `build:` 后启动。此命令只拉取并运行镜像，绝不会在部署主机编译源码：

```bash
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml pull
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml up -d
```

## HTTPS 入口

生产清单内置 TLS：主站 Nginx 监听 `80`（永久重定向到 HTTPS）和 `443`，Casual Sheets 与 Casual Docs gateway 分别仅监听 HTTPS 端口（默认 `8444`、`8445`）。三个公开入口都使用 `TLS_CERT_FILE` 和 `TLS_KEY_FILE`，因此证书的 SAN 必须覆盖 `PUBLIC_APP_URL`、`CASUAL_SHEETS_EDITOR_URL` 与 `CASUAL_DOCS_EDITOR_URL` 的主机名。

推荐由 ACME/Let's Encrypt 自动续期服务把证书放在受控绝对路径，并在续期后执行：

```bash
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml restart nginx casual-sheets-gateway casual-docs-gateway
```

不要将私钥放入 Git、镜像或 Compose 环境变量。若网络入口必须使用标准端口，则让外部四层/七层负载均衡器分别转发主站和两个编辑器域名到容器 TLS 端口；该外部代理也必须保留 HTTPS 与 WebSocket Upgrade 头。

## 验证部署

启动后检查服务和 backend 依赖：

```bash
podman compose --env-file deploy/env/production.env -f deploy/docker-compose.prod.yml ps
curl -fsS https://docs.example.com/healthz
curl -fsS https://docs.example.com/readyz
```

`readyz` 只有在 PostgreSQL、Redis 和对象存储都可用时才会返回成功。验证证书链时不要使用 `-k`；应使用受信任的公网证书，或在隔离测试中通过 `--cacert` 显式指定测试 CA。

## 生产环境中的 Office 编辑器

生产 compose 公开三个入口：主站 Nginx、Casual Sheets gateway 和 Casual Docs gateway。backend 生成编辑会话时使用 `CASUAL_DOCS_EDITOR_URL` 或 `CASUAL_SHEETS_EDITOR_URL`，编辑器通过 gateway 回调 backend 的 `/wopi`、`/yjs` 和房间接口。

生产环境可为两个 gateway 提供独立的 HTTPS 域名，或使用同一 HTTPS 主机名加不同端口，例如 `https://docs.example.com:8445` 与 `https://docs.example.com:8444`。将这两个公开地址写入对应的 `CASUAL_*_EDITOR_URL`。不要把容器内部服务名或未加密的生产地址写入这些变量。

上线前执行以下回归：

1. owner 上传 `.docx` 与 `.xlsx`，确认可打开和保存
2. editor 打开相同文档，确认可编辑并产生新版本
3. viewer 打开相同文档，确认只读且保存请求被拒绝
4. 通过 gateway 检查 `/wopi`、`/yjs` 和房间接口不绕过 backend 权限
5. 重启编辑器服务后，确认已保存内容仍能从 MinIO/S3 和版本历史恢复

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
