# ONLYOFFICE

系统不自研 Word / Excel 编辑器，`doc/docx/xls/xlsx` 在线预览和编辑由 ONLYOFFICE Document Server 提供。

## 配置

开发环境默认：

```text
ONLYOFFICE_ENABLED=true
ONLYOFFICE_PUBLIC_URL=http://localhost:8081
ONLYOFFICE_INTERNAL_URL=http://onlyoffice
ONLYOFFICE_JWT_SECRET=change-me
PUBLIC_API_URL=http://localhost:8080
BACKEND_INTERNAL_URL=http://backend:8080
```

生产 nginx 统一入口示例：

```text
ONLYOFFICE_PUBLIC_URL=https://docs.example.com/onlyoffice
ONLYOFFICE_INTERNAL_URL=http://onlyoffice
ONLYOFFICE_JWT_SECRET=<same-as-document-server-JWT_SECRET>
PUBLIC_API_URL=https://docs.example.com
BACKEND_INTERNAL_URL=http://backend:8080
```

`ONLYOFFICE_PUBLIC_URL` 是浏览器加载 Document Server 脚本的地址。`BACKEND_INTERNAL_URL` 是 ONLYOFFICE 容器回调 backend 的地址。

## 权限

backend 生成 ONLYOFFICE config 前会检查统一权限：

- viewer 只读。
- editor / owner 可编辑。
- 无权限返回 forbidden。

前端按钮只做显示控制，真实权限由 backend 判断。

## 保存回调

ONLYOFFICE 状态为 `2` 或 `6` 时，backend 下载回调 URL 内容并创建新版本。callback HTTP client 有超时限制，下载大小受 `MAX_UPLOAD_BYTES` / `DOCUMENT_MAX_UPLOAD_BYTES` 控制。

保存成功会写审计日志 `onlyoffice.save`。

## 常见问题

- 编辑器空白：确认 `ONLYOFFICE_PUBLIC_URL` 从浏览器可访问。
- 保存失败：确认 `BACKEND_INTERNAL_URL` 从 ONLYOFFICE 容器可访问。
- JWT 校验失败：确认 backend 和 ONLYOFFICE 使用同一个 `ONLYOFFICE_JWT_SECRET`。
