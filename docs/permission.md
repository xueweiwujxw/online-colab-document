# Permission

文档权限等级：

```text
owner
editor
viewer
none
```

## 权限矩阵

- owner: 查看、下载、编辑、删除、管理权限、创建分享链接、恢复版本。
- editor: 查看、下载、编辑、恢复版本。
- viewer: 查看、下载，只读打开编辑器。
- none: 不能访问文档接口。

## 后端约束

所有文档相关操作必须经过 backend 的统一权限判断：

- 文档详情、下载、Markdown snapshot、Office session 检查 view。
- Markdown 保存、Office 编辑、版本恢复检查 edit。
- 权限管理和分享管理检查 owner/manage。
- 删除检查 delete。
- Markdown WebSocket 建立时检查权限；viewer 可连接但不能提交更新。

前端隐藏按钮不是安全边界。

## 分享链接

分享链接 token 只在创建响应中明文返回一次，数据库只保存 hash。公开分享访问不需要登录：

- viewer link: 只读和下载。
- editor link: 可保存 Markdown。

分享链接访问、下载和保存会写审计日志，但 metadata 不记录 token。

## 管理员

全局审计日志接口 `GET /api/admin/audit-logs` 要求 `users.is_admin = true`。普通用户会收到 forbidden。
