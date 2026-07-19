# Architecture

## 总体架构

```text
Browser
  |
  | HTTP
  v
React + TypeScript frontend
  |
  | JSON API
  v
Go backend
  |-- PostgreSQL: users, documents, permissions, versions, audit logs
  |-- Redis: sessions, cache, realtime coordination
  |-- MinIO / S3: uploaded document objects
  |-- ONLYOFFICE Document Server: Word / Excel preview and editing
  |-- Share links: token-hash based public access
  `-- Markdown collaboration service: Yjs / WebSocket snapshots
```

## 模块说明

- `backend/cmd/server`: 后端进程入口。
- `backend/internal/config`: 从环境变量加载配置。
- `backend/internal/health`: health check 与 ready check。
- `backend/internal/server`: HTTP server、路由、中间件、优雅退出。
- `frontend/src/api`: 集中管理前端 API 调用。
- `frontend/src/app`: 当前 M0 首页应用。
- `deploy`: Docker Compose、环境变量样例和部署配置。

## 后续里程碑

- M1: 本地用户认证。
- M2: OIDC 登录。
- M3: 文档上传下载。
- M4: 权限系统。
- M5: ONLYOFFICE 集成。
- M6: Markdown 普通编辑。
- M7: Markdown 协同编辑。
- M8: 分享链接。
- M9: 版本管理。
- M10: 审计日志。
- M11: 前端完善。
- M12: Docker 部署与安全加固。

## Word / Excel 为什么使用 ONLYOFFICE

Word / Excel 的格式兼容、渲染、编辑和多人协同复杂度很高。项目要求不自研 docx / xlsx 编辑器，因此后续通过 ONLYOFFICE Document Server 提供在线预览、编辑和协同能力，系统只负责认证、权限、文档元数据、文件存储、配置生成和保存回调。

## Markdown 为什么单独实现

Markdown 是文本格式，服务端保存和前端编辑成本较低。普通编辑阶段实现源码编辑、预览、保存和下载；协同阶段基于 Yjs、WebSocket 和 PostgreSQL snapshot/update 持久化实现多人协同、presence 和断线重连。

当前 Markdown 协同第一版只支持单 backend 实例内的实时广播。多 backend 实例部署时，需要增加 Redis pub/sub 或其他跨实例消息总线来同步 update 与 presence。

## 分享链接

分享链接由 owner 创建，数据库只保存 token hash。创建响应会返回一次明文 token 和完整 URL；后续列表只显示链接元数据。公开分享访问不要求登录，viewer 链接只读，editor 链接可以保存 Markdown。
