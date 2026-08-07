# 配置本地开发环境

本指南说明如何启动本地完整栈、运行测试，并避免前端、编辑器和协作服务使用不同地址。

## 准备本地配置

准备 Podman Compose 或 Docker Compose、Go、Node.js 24 和 pnpm。复制环境变量模板后设置本地专用的 Casual JWT 密钥：

```bash
cp deploy/env/app.env.example .env
```

`.env` 已被 Git 忽略。不要提交其中的密码、令牌或密钥。`CASUAL_JWT_SECRET` 必须至少 16 个随机字符，否则 backend 无法建立 Office 编辑会话。

## 启动完整开发栈

在仓库根目录运行：

```bash
make dev
```

该命令在前台构建并启动 backend、frontend、PostgreSQL、Redis、MinIO、Casual Docs、Casual Sheets 和两个编辑器网关。需要后台运行时，执行：

```bash
podman compose -f deploy/docker-compose.yml up --build -d
```

使用 `http://localhost:3000` 访问应用。不要把 backend、编辑器网关或协作服务端口当作浏览器入口。

## 使用宿主机 Vite

前端样式或组件开发可以改用宿主机 Vite。先启动 backend、依赖服务和 Office 网关，再运行 Vite：

```bash
podman compose -f deploy/docker-compose.yml up -d backend office-collab casual-docs casual-sheets-gateway casual-docs-gateway
cd frontend
corepack pnpm run dev
```

Vite 固定使用 `3000` 端口。端口被占用时，停止旧进程后重试。不要让 Vite 回退到 `3001` 或其他端口，因为 Markdown WebSocket 会校验 `FRONTEND_ORIGIN`。

## 验证修改

按修改范围运行命令：

```bash
make backend-test
make frontend-build
make lint
make test
```

`make test` 运行 backend 测试和前端生产构建。端到端测试需要完整开发栈：

```bash
cd frontend
pnpm exec playwright test
```

## 服务地址

| 服务 | 地址 | 用途 |
| --- | --- | --- |
| 前端 | `http://localhost:3000` | 浏览器入口 |
| Backend 健康检查 | `http://localhost:8080/healthz` | 进程存活检查 |
| Backend 就绪检查 | `http://localhost:8080/readyz` | 数据库、Redis、对象存储检查 |
| MinIO API | `http://localhost:9000` | S3 兼容 API |
| MinIO Console | `http://localhost:9001` | 本地对象存储管理 |
| Casual Sheets 网关 | `http://localhost:1234` | 编辑器内部服务 |
| Casual Docs 网关 | `http://localhost:1235` | 编辑器内部服务 |

## 排查问题

编辑器中文界面、中文用户名和远程光标、Vite worker、端口和容器版本不一致等问题见[开发与编辑器排障](development-troubleshooting.md)。
