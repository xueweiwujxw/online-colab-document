# AGENTS.md

本文件定义 Codex / Agent 在本仓库中的长期开发约束。  
具体任务、里程碑、接口、表结构、验收标准全部看 `TASK.md`。

## 1. 工作原则

1. 每次只完成用户指定的 goal。
2. 开始前先读：
   - `AGENTS.md`
   - `TASK.md`
   - `README.md`
   - 当前任务相关代码
3. 不要顺手实现后续任务。
4. 不要大范围重构无关代码。
5. 不要重复造已有模块。
6. 不要访问仓库外文件，除非用户明确允许。
7. 不要读取、打印、提交 secret。
8. 完成后必须说明：
   - 修改文件
   - 完成内容
   - 验证命令
   - 测试结果
   - 未完成项
   - 下一步建议

## 2. 技术约束

项目技术栈：

- Backend: Go
- Frontend: React + TypeScript
- Frontend package manager: pnpm
- Database: PostgreSQL
- Cache: Redis
- Storage: MinIO / S3
- Office editor: ONLYOFFICE Document Server
- Markdown collab: Yjs / WebSocket
- Deploy: Docker Compose / Nginx

禁止自研 docx / xlsx 编辑器。  
Word / Excel 在线编辑必须通过 ONLYOFFICE。

## 3. 代码约束

Backend 分层：

```text
handler -> service -> repository/storage
```

要求：

1. Handler 只处理参数、登录态、调用 service、返回 JSON。
2. 业务逻辑放 service。
3. 数据库访问放 repository。
4. 所有 schema 变更必须写 migration。
5. 配置必须来自环境变量或配置文件，不要硬编码。

Frontend 要求：

1. API 调用集中放在 `frontend/src/api/`。
2. 页面组件不要散落 `fetch`。
3. 页面必须处理 loading / error / empty / forbidden。
4. API 地址来自环境变量，不要硬编码。
5. 前端依赖使用 pnpm 管理，不要改用 npm / yarn。

## 4. 权限约束

所有文档相关操作必须经过统一权限判断。

权限等级：

```text
owner
editor
viewer
none
```

禁止：

1. 只靠前端隐藏按钮做权限。
2. 未检查权限就返回下载地址。
3. 未检查权限就返回 ONLYOFFICE config。
4. 未检查权限就建立 Markdown WebSocket。
5. 在 handler 中到处散落权限判断。

## 5. 安全约束

禁止提交：

```text
password
token
secret
private key
生产环境 .env
```

要求：

1. 密码必须 hash 存储。
2. session/share token 只存 hash。
3. 日志不能输出 password/token/secret/cookie。
4. 上传文件必须校验大小、扩展名、MIME。
5. 对外错误返回统一 JSON，不直接暴露内部错误。

## 6. Git 规范

分支命名：

```text
feature/m0-project-skeleton
feature/m1-local-auth
feature/m2-oidc-login
feature/m3-document-upload
feature/m4-permission
feature/m5-onlyoffice
feature/m6-markdown-editor
feature/m7-markdown-collab
feature/m8-share-link
feature/m9-versioning
feature/m10-audit-log
feature/m11-frontend
feature/m12-deploy
fix/<short-name>
docs/<short-name>
chore/<short-name>
```

Commit 使用 Conventional Commits：

```text
<type>(<scope>): <summary>
```

type：

```text
feat
fix
docs
test
refactor
chore
build
ci
style
perf
```

示例：

```text
feat(auth): add local login
feat(document): add upload API
feat(permission): add permission service
feat(onlyoffice): add editor config endpoint
fix(storage): validate object key
docs(deploy): add compose guide
test(permission): cover role matrix
```

不要使用：

```text
update
fix bug
wip
temp
misc
修改
提交
```

开发过程中每完成一个可验证的小部分，应及时提交一次 commit。
提交前必须确认本次提交范围，只包含当前小部分相关改动，不混入无关文件。
Commit message 必须遵守 Conventional Commits。

## 7. 测试规范

完成任务后尽量运行相关测试。

常用命令：

```bash
make test
make backend-test
make frontend-build
```

前端相关命令使用 pnpm：

```bash
cd frontend && pnpm install
cd frontend && pnpm run build
```

如果测试失败，说明：

```text
失败命令
失败原因
已完成的部分
建议下一步
```

不要粘贴超长日志，只保留关键错误和摘要。

## 8. 完成输出格式

每次完成任务后按这个格式回复：

```text
## 修改文件

- xxx

## 完成内容

- xxx

## 验证命令

- xxx

## 测试结果

- xxx

## 未完成项

- xxx

## 下一步建议

- xxx
```
