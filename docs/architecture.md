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
  |-- Casual Office: docx / xlsx editing
  |-- Share links: token-hash based public access
  `-- Markdown collaboration service: Yjs / WebSocket snapshots
```

## 模块说明

- `backend/cmd/server`: 后端进程入口。
- `backend/internal/config`: 从环境变量加载配置。
- `backend/internal/db`: 启动时执行数据库 migrations。
- `backend/internal/health`: health check 与 ready check。
- `backend/internal/server`: HTTP server、路由、中间件、优雅退出。
- `frontend/src/api`: 集中管理前端 API 调用。
- `frontend/src/app`: 当前 M0 首页应用。
- `deploy`: Docker Compose、环境变量样例和部署配置。

## 里程碑

M0 到 M12 已覆盖项目骨架、本地/OIDC 登录、文档上传下载、权限、Casual Office、Markdown 编辑、Markdown 协同、分享链接、版本管理、审计日志、前端完善和 Docker 部署。

## Office 编辑器范围

项目不自研 Office 编辑器。当前通过 Casual Office 支持 docx / xlsx 编辑和保存；`.doc` / `.xls` 暂不支持，且项目不部署或使用 ONLYOFFICE。

## Markdown 为什么单独实现

Markdown 是文本格式，服务端保存和前端编辑成本较低。普通编辑阶段实现源码编辑、预览、保存和下载；协同阶段基于 Yjs、WebSocket 和 PostgreSQL snapshot/update 持久化实现多人协同、presence 和断线重连。

当前 Markdown 协同第一版只支持单 backend 实例内的实时广播。多 backend 实例部署时，需要增加 Redis pub/sub 或其他跨实例消息总线来同步 update 与 presence。

## 分享链接

分享链接由 owner 创建，数据库只保存 token hash。创建响应会返回一次明文 token 和完整 URL；后续列表只显示链接元数据。公开分享访问不要求登录，viewer 链接只读，editor 链接可以保存 Markdown。

## 版本管理

每次上传、Markdown 保存、Office 保存和历史版本恢复都会写入 `document_versions`。恢复历史版本会读取旧版本对象，写入新的对象存储 key，再创建一个新版本并更新 `documents.current_version_id`，不会覆盖旧版本。
