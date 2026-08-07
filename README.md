# 在线协作文档服务

这是一个可私有化部署的文档协作服务。你可以管理本地与 OIDC 账户、上传文档、分配权限、通过链接分享内容，并审计后台操作。

## 开始使用

本地运行需要 Podman Compose 或 Docker Compose、Go、Node.js 24 和 pnpm：

1. 复制 `deploy/env/app.env.example` 到仓库根目录的 `.env`
2. 在 `.env` 中为 `CASUAL_JWT_SECRET` 设置至少 16 个随机字符，并替换所有生产占位值
3. 运行 `make dev`
4. 打开 `http://localhost:3000` 并注册本地账户

开发服务、端口和常见故障见[开发指南](docs/development.md)。生产部署前请阅读[部署指南](docs/deployment.md)。

## 支持范围

- `.docx`：通过自托管 Casual Docs 打开、编辑和保存
- `.xlsx`：通过自托管 Casual Sheets 打开、编辑和保存
- `.md`：内置编辑器和实时协作
- `.doc`、`.xls`：不支持上传或打开

项目不部署、配置或调用 ONLYOFFICE，也不自研 Word 或 Excel 编辑器。

## 文档导航

- [开发指南](docs/development.md)：本地启动、测试、端口与排障入口
- [部署指南](docs/deployment.md)：生产配置、数据持久化、备份与已知限制
- [使用指南](docs/usage.md)：账户、文档、协作、分享与管理后台
- [系统架构](docs/architecture.md)：服务边界与数据流
- [权限模型](docs/permission.md)：角色、分享链接与服务端授权边界
- [Markdown 协作](docs/markdown-collab.md)：连接、权限、持久化与扩展限制
- [管理控制台](docs/admin-console.md)：后台能力与审计范围
- [开发与编辑器排障](docs/development-troubleshooting.md)：已知问题与验证顺序
- [许可证与第三方声明](THIRD_PARTY_NOTICES.md)：开源许可证、补丁和分发义务

## 常用命令

```bash
make dev
make down
make logs
make test
make lint
```

`make dev` 在前台运行 Compose。需要后台运行时，执行 `podman compose -f deploy/docker-compose.yml up --build -d`。

## 当前限制

- Markdown 实时协作只支持单个 backend 实例。多实例部署需要 Redis pub/sub 或其他跨实例消息总线
- 全局审计日志仅向管理员开放。文档 owner 的范围审计视图尚未提供
- 生产 compose 是基础部署配置。Office 编辑器必须完成外部地址、网关和保存链路验证后才能对外启用，详见[部署限制](docs/deployment.md#生产环境中的-office-编辑器)
