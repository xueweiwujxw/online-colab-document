# 第三方许可证声明

本仓库的原创代码使用 Apache License 2.0。第三方代码仍按其各自许可证分发。本文件记录当前直接依赖、构建输入和再分发要求，不替代各依赖发布包中的完整许可证文本。

## 许可证策略

Apache-2.0 允许使用、修改、复制、发布、再许可和商业分发，并授予贡献者可许可专利的使用权。它与本仓库直接使用的 MIT 和 Apache-2.0 依赖兼容。

当你分发源代码、二进制包或容器镜像时：

1. 附带本仓库的 `LICENSE` 和 `NOTICE`
2. 保留第三方的版权、专利、商标和归属声明
3. 为修改过的 Apache-2.0 文件保留显著修改说明
4. 收集并附带新依赖自己的许可证和 NOTICE 文件

Apache-2.0 不授予第三方商标使用权。不要把 Casual Office、React、Univer 或其他上游项目的名称作为本项目的背书。

## 直接依赖和构建输入

| 组件 | 用途 | 许可证 | 分发处理 |
| --- | --- | --- | --- |
| Casual Docs | `.docx` 编辑器构建输入 | Apache-2.0 | 保留上游 LICENSE/NOTICE，声明本仓库补丁 |
| Casual Sheets | `.xlsx` 编辑器构建输入与 SDK | Apache-2.0 | 保留上游 LICENSE/NOTICE，声明本仓库补丁 |
| React | 前端 UI | MIT | 在分发包中保留 MIT 声明 |
| Yjs | Markdown 协作数据模型 | MIT | 在分发包中保留 MIT 声明 |
| Hocuspocus | Office 与 Markdown 协作服务 | MIT | 在分发包中保留 MIT 声明 |
| Univer | Casual Sheets 的上游组件 | Apache-2.0 | 跟随 Casual Sheets 的上游声明 |
| Go OIDC、WebSocket、MinIO SDK | backend 网络与对象存储依赖 | 各自许可证 | 通过发布构件中的模块许可证清单复核 |

版本与源地址由 `frontend/pnpm-lock.yaml`、`office-collab/pnpm-lock.yaml` 和 `backend/go.sum` 锁定。升级依赖或更换基础镜像时，必须重新运行许可证审查。

## Casual Office 衍生构件

`deploy/casual-docs` 和 `deploy/casual-sheets` 从固定上游 revision 构建，并在构建前应用仓库内补丁。它们属于 Apache-2.0 上游的修改构件。每个镜像必须包含 `/licenses` 中的上游 LICENSE、仓库 `NOTICE` 和补丁副本。

本仓库未复制或修改 React、Yjs、Hocuspocus、Univer 或 Go 模块的源文件。包管理器链接或下载这些依赖不会把其许可证改写为本仓库许可证。

## 发布前检查

发布源代码、镜像或安装包前检查：

```bash
test -f LICENSE
test -f NOTICE
test -f THIRD_PARTY_NOTICES.md
podman compose -f deploy/docker-compose.yml build casual-docs casual-sheets
```

然后在每个最终镜像中确认 `/licenses` 存在，并检查新增依赖的许可证。对 GPL、AGPL、LGPL、MPL、SSPL 或自定义许可证依赖执行单独法务审查；不要仅凭包名推断兼容性。
