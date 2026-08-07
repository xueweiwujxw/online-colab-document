# 配置 Markdown 协作

本页说明 Markdown 协作连接、权限和持久化边界。

Markdown 协同通过 WebSocket 和 Yjs 文本模型实现。backend 当前使用单实例内存 hub 广播 update 和 presence，并将 update/snapshot 持久化到 PostgreSQL。

## 接口

- `GET /api/documents/:id/markdown/snapshot`
- `WS /api/documents/:id/markdown/ws`

两个接口都要求登录并检查文档权限。

## 权限

- owner/editor 可以提交更新。
- viewer 可以连接和接收更新，但不能提交编辑。
- none 不能建立连接。

## 持久化

`markdown_updates` 保存增量更新序列，`markdown_snapshots` 保存周期 snapshot。`MARKDOWN_SNAPSHOT_UPDATE_INTERVAL` 控制 snapshot 间隔。

## 部署注意

当前第一版只支持单 backend 实例内实时广播。多实例部署时，需要增加 Redis pub/sub 或其他跨实例消息总线，否则不同 backend 实例上的用户无法互相实时同步。

nginx 必须支持 WebSocket upgrade：

```nginx
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
```
