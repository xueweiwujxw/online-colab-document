# 开发与编辑器排障手册

本文记录本项目开发中已经反复出现的问题。遇到编辑器、登录态或协同异常时，先按本文检查，不要通过改端口、直接访问内部服务等方式绕过现有配置。

## 统一入口与端口

本地开发只通过 `http://localhost:3000` 访问前端。`FRONTEND_ORIGIN` 固定为该地址，Markdown 协作服务会校验 Origin。

- `3000` 被占用时，停止旧的前端进程或容器后重启 Vite；不要使用 Vite 自动回退的 `3001`、`3002` 等端口。
- 不要直接在浏览器中访问 backend `8080`、Casual Sheets `1234` 或 Casual Docs `1235` 作为应用入口。它们会造成 `localhost` 与 `127.0.0.1`、跨源 cookie 或 WebSocket Origin 不一致。
- 典型症状是登录后 Office 下载 `401`、`Failed to fetch`、协作 WebSocket `403`，或者界面显示为只读。

## Casual Sheets 的 xlsx 解析

Casual Sheets 依赖包内的 `parser.worker.js`。Vite 必须保留 `frontend/vite.config.ts` 中的：

```ts
optimizeDeps: { exclude: ['@casualoffice/sheets/xlsx'] }
```

修改该项或更新 Sheets 依赖后必须重启 Vite。否则浏览器可能请求不存在的 `.vite/deps/parser.worker.js`，表现为表格持续加载、协同异常，或出现 `xlsx parser worker ran out of memory parsing this file`。

## Casual 编辑器中文界面

Casual Docs 与 Casual Sheets 是独立服务，主前端的语言设置不会传入它们。开发 compose 通过两个 gateway 注入 `deploy/casual-gateway/zh-CN.js`：

- `docs.conf` 和 `sheets.conf` 为 HTML 注入脚本，并禁用上游 HTML 压缩，以确保 Nginx 能完成注入。
- `zh-CN.js` 只翻译工具栏、菜单、弹窗和辅助标签；它会跳过 `contenteditable`、ProseMirror 和文本输入区域，不能改动用户文档正文。
- 同一份脚本同时挂载到两个 Nginx 容器时，Compose volume 必须使用共享 SELinux 标签 `:z`，不能使用私有标签 `:Z`。后者会导致后创建的容器可读、另一个返回 `403`，错误日志中会出现 `open() ... Permission denied`。
- 修改该脚本或任一 gateway 配置后，重建/重建网关容器：

```bash
podman compose -f deploy/docker-compose.yml up -d --force-recreate casual-sheets-gateway casual-docs-gateway
```

- 如果仍看到英文，先在浏览器网络面板确认 `http://localhost:1234/casual-zh-CN.js` 或 `http://localhost:1235/casual-zh-CN.js` 返回 `200`，然后执行强制刷新。不要把翻译逻辑放入用户文档内容或只改 `frontend/index.html`。

## 修改 Office 集成后的验证顺序

1. 校验 compose：`podman compose -f deploy/docker-compose.yml config`。
2. 构建前端：`make frontend-build`。
3. 重新创建受影响的 Casual 网关或编辑器容器。
4. 通过 `http://localhost:3000` 上传一个 `.docx` 和一个 `.xlsx`，分别打开编辑页；确认菜单显示中文、编辑用户可写、viewer 只读。
5. 对 xlsx 运行 `frontend/e2e/editor-experience.spec.ts` 的原生协同用例，确认画布、公式栏、协同和保存版本均可用。

## DOCX 端到端样例损坏

`frontend/e2e/editor-experience.spec.ts` 曾使用内嵌 Base64 的 DOCX fixture。若 Casual Docs 显示 `Failed to read .docx archive` 或 `expected 7 records in central dir, got 0`，先将它视为测试样例损坏，而不是网关、权限或汉化故障：该错误发生在编辑器解析上传文件之前。应使用由已验证工具生成的 `.docx` fixture 替换内嵌字节后，再执行 DOCX 协同回归。

## 中文用户名与协同光标

协同测试账户的 `displayName` 必须包含中文字符。测试应在文档权限页、编辑器在线成员标签和远程光标标签中分别断言完整显示名，而不是只验证连接成功。若出现乱码，按链路检查：注册接口 JSON 编码、JWT `display_name` claim、URL 查询参数编码、WebSocket awareness payload，以及编辑器 DOM 的文本/属性渲染。不要通过把用户名替换成 ASCII 来绕过该问题。

Casual Docs 当前宿主补丁解析 JWT payload 时必须使用 `TextDecoder`：`atob()` 的返回值是二进制字符串，直接传给 `JSON.parse()` 会把 UTF-8 中文显示名解成 Latin-1 乱码（例如 `王文` 变成 `çæ`）。修复后重建 `casual-docs` 容器，再验证在线成员与悬浮光标。

为了兼容尚未重建的旧 Casual Docs 镜像，网关注入的 `zh-CN.js` 还会恢复在线成员和 Yjs 光标标签中的这类乱码；它只处理这些 UI 覆盖层，绝不遍历或改写普通编辑正文。镜像重新构建后，该兼容逻辑仍是无害的兜底。

## 容器与本地代码版本不一致

浏览器实际访问的是正在运行容器中的代码。修改 `frontend/vite.config.ts`、`deploy/casual-gateway/*` 或 Dockerfile 后，仅刷新浏览器不会应用改动；需要重启 Vite 或按影响范围重建对应容器。排障前先用 `podman compose ... ps` 确认服务名和运行状态，避免误测旧容器。
