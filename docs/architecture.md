# 系统架构

服务将身份、权限、文档元数据、对象存储和编辑器集成分开处理。浏览器只通过 frontend 或 Nginx 入口访问 API 和协作连接。

## 服务边界

```text
Browser
  |
  v
React frontend / Nginx
  |
  v
Go backend
  |-- PostgreSQL: users, sessions, documents, permissions, versions, audit logs
  |-- Redis: session and collaboration support
  |-- MinIO / S3: document objects
  |-- Casual Office: docx and xlsx editor integration
  `-- Markdown collaboration: WebSocket, Yjs updates and snapshots
```

backend 按 `handler -> service -> repository/storage` 分层。handler 处理 HTTP 参数和登录态，service 执行业务规则，repository 与 storage 负责数据库和对象存储访问。

## 授权边界

backend 在返回文档详情、下载地址、Office 会话、Markdown snapshot 或建立 Markdown WebSocket 前验证权限。编辑、保存、恢复版本、权限管理、分享管理和删除使用更高的权限等级。

前端只根据权限隐藏不适用的操作。服务端权限检查仍是唯一授权边界，完整矩阵见[权限模型](permission.md)。

## 文档与版本

上传、Markdown 保存、Office 保存和版本恢复都会创建 `document_versions` 记录。恢复操作会创建新版本并更新当前版本，不会覆盖历史对象。

对象内容保存在 MinIO 或 S3，数据库保存元数据、版本和权限。分享链接和会话只保存哈希，不保存明文令牌。

## 编辑器集成

Office 文档由自托管 Casual Office 处理：Casual Docs 用于 `.docx`，Casual Sheets 用于 `.xlsx`。项目不包含 Office 编辑内核，也不使用 ONLYOFFICE。旧格式 `.doc` 和 `.xls` 不受支持。

Markdown 使用内置编辑器。协同更新通过 WebSocket 广播，并周期性写入 PostgreSQL snapshot。单实例 backend 可以提供实时协作，多实例需要 Redis pub/sub 或其他跨实例消息总线。

## 运维接口

`/healthz` 用于进程存活检查，`/readyz` 验证 PostgreSQL、Redis 和对象存储。管理员可在 `/admin` 查看用户、文档、对象存储、OIDC 状态和审计日志。
