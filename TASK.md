# TASK.md

# 在线文档共享编辑服务开发任务

> Hard constraint: the project must not deploy, configure, or call ONLYOFFICE. The legacy `.doc` and `.xls` formats are not supported until a different provider has passed a separate POC. Any historical ONLYOFFICE references below only describe retired work; they are not executable requirements and must not be restored.

## 0. 项目目标

开发一个私有化部署的在线文档共享编辑服务。

核心能力：

- 后端使用 Go
- 前端使用 React + TypeScript
- 支持本地用户登录
- 支持 OIDC 登录
- 支持上传、下载 Word / Excel / Markdown 文档
- 支持 Word / Excel 在线编辑
- 支持 Markdown 在线编辑
- 支持多人共享编辑
- 支持权限管理
- 支持版本管理
- 支持分享链接
- 支持审计日志
- 支持 Docker / Docker Compose 部署

## 1. 总体技术路线

Word / Excel 在线编辑不自研，使用自托管开源 Office 编辑器提供商。
当前只使用 Casual Office 支持 docx / xlsx；项目禁止部署、配置或调用 ONLYOFFICE。`.doc` / `.xls` 暂不支持，待新的替代方案通过 POC 后再评估。

本系统负责：

- 用户认证
- 用户管理
- 文档元数据
- 文件存储
- 权限管理
- 版本管理
- 分享链接
- 审计日志
- Office 编辑器配置生成
- Office 编辑器保存回调处理

Office 编辑器提供商负责：

- docx 在线预览
- docx 在线编辑
- xlsx 在线预览
- xlsx 在线编辑
- Word / Excel 多人协同编辑

Office 编辑器提供商要求：

- 必须自托管，不能依赖外部 SaaS 保存用户文档。
- 必须支持 doc/docx/xls/xlsx 的在线打开。
- 必须至少支持 docx/xlsx 在线编辑和保存回后端。
- 必须接入统一权限判断。
- 必须能和版本管理、审计日志、对象存储链路集成。
- 禁止在本项目内自研 docx / xlsx 编辑器。

Markdown 编辑可以自研。

Markdown 第一版：

- Markdown 源码编辑
- Markdown 预览
- 保存到后端
- 下载原始 md 文件

Markdown 第二版：

- 基于 Yjs / CRDT / WebSocket 实现多人协同编辑
- 支持 presence
- 支持断线重连
- 支持 snapshot 持久化

## 2. 推荐目录结构

如果仓库为空，创建下面结构。

如果已有类似结构，优先沿用已有结构，不要重复创建。

```text
.
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/
│   │   ├── audit/
│   │   ├── auth/
│   │   │   ├── local/
│   │   │   ├── oidc/
│   │   │   └── session/
│   │   ├── config/
│   │   ├── db/
│   │   ├── document/
│   │   ├── health/
│   │   ├── markdown/
│   │   │   ├── collab/
│   │   │   └── snapshot/
│   │   ├── middleware/
│   │   ├── office/
│   │   ├── permission/
│   │   ├── server/
│   │   ├── share/
│   │   ├── storage/
│   │   ├── user/
│   │   └── worker/
│   ├── migrations/
│   ├── tests/
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── api/
│   │   ├── app/
│   │   ├── auth/
│   │   ├── components/
│   │   ├── editors/
│   │   │   ├── markdown/
│   │   │   └── office/
│   │   ├── pages/
│   │   │   ├── LoginPage/
│   │   │   ├── DocumentListPage/
│   │   │   ├── DocumentDetailPage/
│   │   │   ├── OfficeEditorPage/
│   │   │   ├── MarkdownEditorPage/
│   │   │   ├── PermissionPage/
│   │   │   └── AdminPage/
│   │   ├── routes/
│   │   ├── stores/
│   │   ├── utils/
│   │   └── main.tsx
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
│
├── deploy/
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   ├── env/
│   │   └── app.env.example
│   └── nginx/
│       └── nginx.conf
│
├── docs/
│   ├── architecture.md
│   ├── api.md
│   ├── deployment.md
│   ├── permission.md
│   ├── office-editor.md
│   └── markdown-collab.md
│
├── AGENTS.md
├── TASK.md
├── README.md
└── Makefile
```

## 3. 全局开发规则

1. 每次只完成当前 milestone 的任务。
2. 不要顺手实现后续 milestone。
3. 不要大范围重构无关代码。
4. 不要硬编码 secret、token、密码、证书私钥。
5. 所有配置必须来自环境变量、配置文件或 Docker Compose。
6. 所有数据库 schema 变更必须写 migration。
7. 所有文档相关接口必须接入权限检查。
8. 所有文件上传必须校验文件大小、扩展名和 MIME 类型。
9. 外部可访问 token 只能保存 hash，不能保存明文。
10. 测试日志过长时，只输出失败原因和最终摘要。
11. 完成后必须输出修改文件、完成内容、验证命令、测试结果、未完成项、下一步建议。

## 4. Milestone 列表

```text
M0  项目骨架
M1  本地用户认证
M2  OIDC 登录
M3  文档上传下载
M4  权限系统
M5  Office 编辑器集成（已由替代 POC 取代）
M6  Markdown 普通编辑
M7  Markdown 协同编辑
M8  分享链接
M9  版本管理
M10 审计日志
M11 前端完善
M12 Docker 部署与安全加固
```

---

# M0：项目骨架

## M0.1 目标

完成可以启动、可以开发、可以测试、可以 Docker Compose 拉起基础依赖的最小项目骨架。

本阶段不实现业务功能。

## M0.2 任务范围

需要实现：

- Go backend 最小服务
- React + TypeScript + Vite frontend 最小应用
- PostgreSQL
- Redis
- MinIO
- Docker Compose
- Makefile
- README
- health check
- ready check
- 基础架构文档

不要实现：

- 本地登录
- OIDC 登录
- 文档上传下载
- 权限系统
- Office 编辑器集成
- Markdown 编辑器
- 分享链接
- 审计日志

## M0.3 Backend 要求

创建最小 Go HTTP server。

必须支持：

- 配置加载
- structured logging
- graceful shutdown
- health check
- ready check
- JSON response
- 基础错误处理

环境变量：

```text
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgres://docs:docs@postgres:5432/docs?sslmode=disable
REDIS_ADDR=redis:6379
S3_ENDPOINT=http://minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=docs
S3_USE_SSL=false
```

必须实现 API：

```text
GET /healthz
GET /readyz
```

`GET /healthz` 返回：

```json
{
  "status": "ok"
}
```

`GET /readyz` 返回：

```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "storage": "ok"
  }
}
```

如果某个依赖还没有接入，返回：

```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "redis": "skipped",
    "storage": "skipped"
  }
}
```

## M0.4 Frontend 要求

创建 React + TypeScript + Vite 应用。

首页显示：

```text
Docs Collab Service
```

首页需要显示后端健康状态。

必须有集中 API client。

建议文件：

```text
frontend/src/api/client.ts
frontend/src/api/health.ts
```

首页至少处理：

- loading
- success
- error

前端环境变量：

```text
VITE_API_BASE_URL=http://localhost:8080
```

## M0.5 Docker Compose 要求

创建：

```text
deploy/docker-compose.yml
```

包含服务：

- backend
- frontend
- postgres
- redis
- minio

M0 不加入：

- office editor
- keycloak
- nginx

postgres 配置：

```text
POSTGRES_DB=docs
POSTGRES_USER=docs
POSTGRES_PASSWORD=docs
```

minio 配置：

```text
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
```

暴露端口：

```text
frontend: 3000
backend: 8080
postgres: 5432
redis: 6379
minio api: 9000
minio console: 9001
```

## M0.6 Makefile 要求

根目录增加 Makefile。

至少包含：

```makefile
dev
up
down
logs
test
lint
backend-test
frontend-build
```

建议行为：

```text
make dev             启动 docker compose
make up              启动 docker compose
make down            停止 docker compose
make logs            查看日志
make test            运行 backend test 和 frontend build
make backend-test    运行 go test ./...
make frontend-build  运行 npm run build
```

## M0.7 README 要求

README 至少包含：

- 项目简介
- 技术栈
- 本地启动方式
- 环境变量说明
- 常用命令
- 当前里程碑
- 下一步开发计划

README 中明确说明：

```text
当前只完成 M0 项目骨架。
尚未实现认证、文档上传下载、权限、Office 编辑器、Markdown 协同。
```

## M0.8 docs/architecture.md 要求

写简短架构说明。

必须包含：

- 总体架构图，text diagram 即可
- 模块说明
- 后续里程碑
- 为什么 Word / Excel 编辑使用自托管 Office 提供商
- 为什么 Markdown 编辑单独实现

## M0.9 验收标准

```bash
make dev
```

可以启动基础服务。

Backend 可访问：

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Frontend 可访问：

```text
http://localhost:3000
```

Backend 测试：

```bash
make backend-test
```

Frontend 构建：

```bash
make frontend-build
```

整体测试：

```bash
make test
```

---

# M1：本地用户认证

## M1.1 目标

实现本地用户认证。

支持：

- 创建本地用户
- 本地用户登录
- 登出
- 获取当前用户
- session 或 JWT
- 登录态 middleware
- 前端登录页

不实现：

- OIDC
- 文档权限
- 管理后台

## M1.2 数据库

新增 migration。

创建 users 表：

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    password_hash TEXT,
    auth_source TEXT NOT NULL,
    oidc_subject TEXT,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

如果使用 session，创建 sessions 表：

```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
```

## M1.3 Backend API

实现：

```text
POST /api/auth/local/register
POST /api/auth/local/login
POST /api/auth/logout
GET  /api/auth/me
```

注册请求：

```json
{
  "email": "user@example.com",
  "displayName": "User",
  "password": "password"
}
```

登录请求：

```json
{
  "email": "user@example.com",
  "password": "password"
}
```

当前用户返回：

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "displayName": "User",
  "authSource": "local",
  "isAdmin": false
}
```

## M1.4 安全要求

1. 密码不能明文存储。
2. 使用安全 password hash。
3. 登录失败不能泄露用户是否存在。
4. disabled 用户不能登录。
5. cookie 需要设置 HttpOnly。
6. 生产环境 cookie 需要 Secure。
7. session token 只能保存 hash。
8. 错误返回统一 JSON。

## M1.5 Frontend 要求

新增页面：

```text
/login
```

登录页包含：

- email 输入
- password 输入
- 登录按钮
- 错误提示
- loading 状态

前端新增 auth store 或 auth context。

路由保护：

- 未登录访问文档页时跳转登录页
- 已登录访问登录页时跳转文档列表页

## M1.6 测试要求

Backend 测试：

- 注册成功
- 重复 email 注册失败
- 登录成功
- 密码错误登录失败
- disabled 用户登录失败
- /api/auth/me 未登录返回 401
- /api/auth/me 登录后返回用户

Frontend 至少保证：

```bash
make frontend-build
```

## M1.7 验收标准

```bash
make test
```

手动验证：

```text
可以注册本地用户
可以登录
刷新页面后保持登录态
可以退出登录
```

---

# M2：OIDC 登录

## M2.1 目标

实现 OIDC 登录。

支持：

- OIDC provider 配置
- 登录跳转
- callback
- id_token 校验
- userinfo 获取
- OIDC 用户自动创建
- 本地用户和 OIDC 用户共存

不实现：

- 自动账号合并，除非配置明确允许

## M2.2 配置

新增环境变量：

```text
OIDC_ENABLED=true
OIDC_ISSUER_URL=https://idp.example.com
OIDC_CLIENT_ID=docs-collab
OIDC_CLIENT_SECRET=change-me
OIDC_REDIRECT_URL=http://localhost:8080/api/auth/oidc/callback
OIDC_SCOPES=openid,email,profile
OIDC_AUTO_MERGE_BY_EMAIL=false
```

## M2.3 Backend API

实现：

```text
GET /api/auth/oidc/login
GET /api/auth/oidc/callback
```

## M2.4 安全要求

必须校验：

- state
- nonce，如果当前流程使用 nonce
- issuer
- audience
- expiry
- id_token signature

用户绑定规则：

- OIDC 用户 `auth_source=oidc`
- 使用 `oidc_subject` 作为外部身份唯一 ID
- email 已存在时，默认不要自动合并
- 如果 `OIDC_AUTO_MERGE_BY_EMAIL=true`，才允许合并

## M2.5 Frontend 要求

登录页增加：

```text
使用 OIDC 登录
```

点击后跳转：

```text
/api/auth/oidc/login
```

OIDC 登录成功后回到前端首页或文档列表页。

## M2.6 测试要求

覆盖：

- OIDC disabled 时接口不可用
- callback state 错误
- token 校验失败
- 新 OIDC 用户创建
- 已有 OIDC 用户登录
- email 冲突不自动合并

## M2.7 验收标准

```text
本地登录不受影响
OIDC 登录成功后 users 表有对应用户
/api/auth/me 能返回 OIDC 用户信息
```

---

# M3：文档上传下载

## M3.1 目标

实现文档基础管理。

支持：

- 上传 doc / docx / xls / xlsx / md / markdown
- 下载文档
- 文档列表
- 文档详情
- 文档删除，软删除
- 文档版本基础记录

不实现：

- 在线编辑
- 分享链接
- 复杂权限
- 历史版本恢复

当前阶段可以只允许 owner 访问自己的文档。

## M3.2 数据库

新增 documents 表：

```sql
CREATE TABLE documents (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    file_ext TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    current_version_id UUID,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

新增 document_versions 表：

```sql
CREATE TABLE document_versions (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    version_no BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);
```

## M3.3 Backend API

实现：

```text
GET    /api/documents
POST   /api/documents/upload
GET    /api/documents/:id
GET    /api/documents/:id/download
DELETE /api/documents/:id
GET    /api/documents/:id/versions
```

## M3.4 上传要求

必须校验：

- 文件大小
- 扩展名
- MIME 类型

允许扩展名：

```text
doc
docx
xls
xlsx
md
markdown
```

对象存储 key 不要直接使用用户文件名。

建议 key 格式：

```text
documents/{document_id}/versions/{version_id}/{safe_filename}
```

## M3.5 Storage 要求

实现 storage 抽象：

```go
type Storage interface {
    PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
    GetObject(ctx context.Context, key string) (io.ReadCloser, error)
    DeleteObject(ctx context.Context, key string) error
    PresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}
```

第一版实现 MinIO / S3。

## M3.6 Frontend 要求

新增页面：

```text
/documents
/documents/:id
```

文档列表页：

- 显示文档标题
- 显示文件类型
- 显示更新时间
- 上传按钮
- 下载按钮
- 删除按钮

上传组件：

- 选择文件
- 上传进度，可选
- 错误提示
- 成功后刷新列表

## M3.7 测试要求

Backend 覆盖：

- 上传成功
- 未登录上传失败
- 不允许的扩展名失败
- 超过大小限制失败
- 下载成功
- 非 owner 下载失败
- 删除成功
- 删除后列表不显示

## M3.8 验收标准

```text
可以上传 docx / xlsx / md
可以在列表看到文档
可以下载原文件
删除后列表不再显示
对象存储中有对应文件
数据库中有 documents 和 document_versions 记录
```

---

# M4：权限系统

## M4.1 目标

实现文档权限系统。

支持：

- owner
- editor
- viewer
- 用户授权
- 取消授权
- 所有文档接口接入 PermissionService

## M4.2 数据库

新增 document_permissions 表：

```sql
CREATE TABLE document_permissions (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    subject_type TEXT NOT NULL,
    subject_id UUID NOT NULL,
    permission TEXT NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);
```

第一版 subject_type 只实现：

```text
user
```

后续再扩展：

```text
group
workspace
```

## M4.3 权限等级

```text
owner
editor
viewer
none
```

## M4.4 权限矩阵

```text
操作                  owner   editor   viewer
查看文档              yes     yes      yes
下载文档              yes     yes      yes
编辑文档              yes     yes      no
删除文档              yes     no       no
修改权限              yes     no       no
创建分享链接          yes     no       no
查看历史版本          yes     yes      yes
恢复历史版本          yes     yes      no
```

## M4.5 Backend API

实现：

```text
GET    /api/documents/:id/permissions
POST   /api/documents/:id/permissions
DELETE /api/documents/:id/permissions/:permissionId
```

授权请求：

```json
{
  "subjectType": "user",
  "subjectId": "user_uuid",
  "permission": "editor"
}
```

## M4.6 PermissionService

实现统一接口：

```go
type PermissionService interface {
    CanView(ctx context.Context, userID, documentID string) (bool, error)
    CanEdit(ctx context.Context, userID, documentID string) (bool, error)
    CanManage(ctx context.Context, userID, documentID string) (bool, error)
    CanDelete(ctx context.Context, userID, documentID string) (bool, error)
    CanShare(ctx context.Context, userID, documentID string) (bool, error)
}
```

要求：

1. owner 永远拥有所有权限。
2. editor 可以查看、下载、编辑。
3. viewer 只能查看、下载。
4. 无权限返回 403。
5. 不要在 handler 中散落权限判断。
6. document API 必须统一接入 PermissionService。

## M4.7 Frontend 要求

新增权限管理页面：

```text
/documents/:id/permissions
```

功能：

- 查看当前权限列表
- 添加用户权限
- 修改用户权限，可选
- 删除用户权限
- 非 owner 不显示权限管理入口

## M4.8 测试要求

覆盖权限矩阵：

- owner 全部通过
- editor 可以查看和编辑，不能授权和删除
- viewer 可以查看，不能编辑和删除
- 无权限不能访问
- 授权后立即生效
- 删除授权后立即失效

## M4.9 验收标准

```text
owner 可以授权 viewer/editor
editor 可以编辑但不能授权
viewer 只能查看和下载
无权限用户不能访问文档
所有文档接口都有权限校验
```

---

# M6：Markdown 普通编辑

## M6.1 目标

实现 Markdown 普通在线编辑。

支持：

- 打开 md / markdown 文档
- 编辑 Markdown 源码
- 预览 Markdown
- 保存
- 下载当前内容

不实现多人协同。

## M6.2 Backend API

实现：

```text
GET /api/documents/:id/markdown
PUT /api/documents/:id/markdown
```

GET 要求：

- 检查文档存在
- 检查文件类型为 md/markdown
- 检查 CanView
- 返回当前 Markdown 内容

PUT 要求：

- 检查文档存在
- 检查文件类型为 md/markdown
- 检查 CanEdit
- 保存内容到对象存储
- 创建新 document_versions 记录
- 更新 documents.current_version_id

## M6.3 Frontend 要求

新增页面：

```text
/documents/:id/markdown
```

页面包含：

- Markdown 编辑区
- Markdown 预览区
- 保存按钮
- 保存状态
- 错误提示
- 只读模式

第一版可以使用 textarea。

不要引入复杂富文本编辑器，除非项目已有。

## M6.4 权限要求

- viewer 可以打开，只读。
- editor 可以编辑和保存。
- owner 可以编辑和保存。
- 无权限不能打开。

## M6.5 测试要求

覆盖：

- viewer 只读
- editor 可保存
- 非 Markdown 文档返回错误
- 保存后生成新版本
- 下载得到最新内容

## M6.6 验收标准

```text
md 文件可以在线打开
editor 修改后可以保存
保存后刷新页面内容不丢
viewer 打开为只读
保存后下载得到最新 md 内容
```

---

# M7：Markdown 协同编辑

## M7.1 目标

实现 Markdown 多人协同编辑。

支持：

- 多人同时打开同一个 Markdown
- 实时同步编辑内容
- presence / awareness
- 断线重连
- update 持久化
- snapshot 持久化
- 只读用户不能提交编辑

## M7.2 技术路线

使用：

- Yjs
- WebSocket
- Go WebSocket server
- PostgreSQL 持久化 update / snapshot
- Redis presence，可选

第一版可以只支持单 backend 实例。

如果只支持单实例，必须在文档中说明限制。

## M7.3 数据库

新增 markdown_snapshots：

```sql
CREATE TABLE markdown_snapshots (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    version_no BIGINT NOT NULL,
    content TEXT NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);
```

新增 markdown_updates：

```sql
CREATE TABLE markdown_updates (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    update_seq BIGINT NOT NULL,
    update_data BYTEA NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);
```

## M7.4 Backend API

实现：

```text
GET /api/documents/:id/markdown/snapshot
WS  /api/documents/:id/markdown/ws
```

WebSocket 连接要求：

1. 连接建立前检查 CanView。
2. 用户信息绑定到连接。
3. 服务端知道当前用户是否 CanEdit。
4. 只读用户可以接收 update，但不能提交 update。
5. editor / owner 可以提交 update。
6. update 需要持久化。
7. 服务端广播 update 给其他客户端。
8. 支持 awareness / presence。
9. 断线清理 presence。

## M7.5 Snapshot 策略

需要实现：

- 打开文档时加载最新 snapshot
- replay snapshot 之后的 updates
- 定期生成 snapshot
- snapshot 后可以压缩或清理旧 updates，可选

第一版策略：

```text
每 N 个 updates 生成一次 snapshot
或者每隔固定时间生成一次 snapshot
```

N 可以通过环境变量配置：

```text
MARKDOWN_SNAPSHOT_UPDATE_INTERVAL=100
```

## M7.6 Frontend 要求

MarkdownEditorPage 改成协同模式。

需要处理：

- loading
- connected
- disconnected
- reconnecting
- readonly
- saving
- error
- presence users

## M7.7 测试要求

覆盖：

- 无权限不能建立 WebSocket
- viewer 建立连接但不能提交 update
- editor 可以提交 update
- 多客户端收到广播
- update 持久化
- snapshot 恢复
- 断线重连后内容一致

## M7.8 验收标准

```text
两个浏览器打开同一个 md
A 修改后 B 实时看到
viewer 只能看不能改
刷新后内容不丢
重启 backend 后可以从 snapshot 和 updates 恢复
```

---

# M8：分享链接

## M8.1 目标

实现文档分享链接。

支持：

- 创建分享链接
- 设置权限 viewer / editor
- 设置过期时间
- 禁用分享链接
- 未登录用户通过分享链接访问
- 登录用户通过分享链接访问

## M8.2 数据库

新增 share_links 表：

```sql
CREATE TABLE share_links (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES documents(id),
    token_hash TEXT NOT NULL UNIQUE,
    permission TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);
```

## M8.3 安全要求

1. 分享 token 只展示一次。
2. 数据库只保存 token hash。
3. 支持过期时间。
4. 支持禁用。
5. 创建分享链接需要 CanShare。
6. editor 分享链接是否允许编辑由 owner 创建时决定。
7. 不能通过分享链接修改权限。
8. 分享链接访问需要记录审计日志。

## M8.4 Backend API

实现：

```text
POST   /api/documents/:id/share-links
GET    /api/documents/:id/share-links
DELETE /api/share-links/:id
GET    /api/share/:token
```

创建请求：

```json
{
  "permission": "viewer",
  "expiresAt": "2026-12-31T23:59:59Z"
}
```

创建响应：

```json
{
  "id": "uuid",
  "url": "http://localhost:3000/share/token",
  "token": "plain-token-visible-once"
}
```

## M8.5 Frontend 要求

新增：

```text
/documents/:id/share
/share/:token
```

文档详情页显示分享入口。

分享管理页支持：

- 创建链接
- 查看链接列表
- 禁用链接
- 复制链接

分享访问页：

- 根据 token 加载文档
- viewer 只读
- editor 可编辑，如果允许

## M8.6 测试要求

覆盖：

- owner 创建分享链接
- editor 不能创建分享链接
- token 明文只返回一次
- 数据库存 hash
- 过期链接不可访问
- disabled 链接不可访问
- viewer link 不能编辑
- editor link 可以编辑

## M8.7 验收标准

```text
owner 可以生成分享链接
未登录用户可以通过有效链接查看文档
过期链接无法访问
禁用链接无法访问
分享链接 token 不明文入库
```

---

# M9：版本管理

## M9.1 目标

完善文档版本管理。

支持：

- 查看版本列表
- 下载历史版本
- 恢复历史版本
- 上传、Office 编辑保存、Markdown 保存都生成版本

## M9.2 Backend API

实现：

```text
GET  /api/documents/:id/versions
GET  /api/documents/:id/versions/:versionId/download
POST /api/documents/:id/versions/:versionId/restore
```

## M9.3 行为要求

1. 查看版本需要 CanView。
2. 下载历史版本需要 CanView。
3. 恢复历史版本需要 CanEdit。
4. 恢复历史版本不覆盖旧版本，而是生成一个新版本。
5. 每个版本包含：
   - version_no
   - created_by
   - created_at
   - size_bytes
   - storage_key
6. 普通上传、Office 保存、Markdown 保存都必须生成版本。

## M9.4 Frontend 要求

文档详情页增加版本列表。

版本列表显示：

- 版本号
- 创建时间
- 创建人
- 文件大小
- 下载按钮
- 恢复按钮

## M9.5 测试要求

覆盖：

- 查看版本列表
- 下载历史版本
- viewer 不能恢复
- editor 可以恢复
- restore 后生成新版本
- current_version_id 更新

## M9.6 验收标准

```text
可以查看版本列表
可以下载历史版本
可以恢复历史版本
恢复不会覆盖旧版本
恢复后 current_version_id 指向新版本
```

---

# M10：审计日志

## M10.1 目标

实现审计日志。

记录关键操作：

- 登录
- 登出
- OIDC 登录
- 上传文档
- 下载文档
- 删除文档
- 在线编辑保存
- Office 保存
- Markdown 保存
- 创建权限
- 删除权限
- 创建分享链接
- 禁用分享链接
- 访问分享链接
- 恢复历史版本

## M10.2 数据库

新增 audit_logs 表：

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    ip_addr TEXT,
    user_agent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL
);
```

## M10.3 Backend API

实现管理员接口：

```text
GET /api/admin/audit-logs
```

支持查询参数：

```text
actorUserId
action
targetType
targetId
from
to
limit
offset
```

## M10.4 要求

1. 审计日志写入失败不应影响主流程，但必须记录 error log。
2. metadata 不能存敏感信息。
3. 不能记录密码、token、secret。
4. 普通用户不能访问全局审计日志。
5. owner 可以看自己文档相关审计日志，可选。

## M10.5 Frontend 要求

新增管理员页面：

```text
/admin/audit-logs
```

显示：

- 时间
- 操作者
- action
- target
- IP
- user agent
- metadata

## M10.6 测试要求

覆盖：

- 登录写审计
- 上传写审计
- 授权写审计
- 分享链接访问写审计
- 普通用户不能访问 admin audit API
- admin 可以访问

## M10.7 验收标准

```text
关键操作有审计记录
管理员可以查看审计日志
审计日志不包含 secret
```

---

# M11：前端完善

## M11.1 目标

完善前端整体体验。

包括：

- 登录页
- 文档列表页
- 文档详情页
- 上传组件
- 下载按钮
- Office 编辑页
- Markdown 编辑页
- 权限管理页
- 分享管理页
- 版本管理页
- 审计日志页
- 全局错误处理
- 全局 loading
- 路由保护

## M11.2 页面列表

```text
/login
/documents
/documents/:id
/documents/:id/edit
/documents/:id/markdown
/documents/:id/permissions
/documents/:id/share
/documents/:id/versions
/share/:token
/admin/audit-logs
```

## M11.3 前端要求

1. 所有 API 调用集中在 `src/api`。
2. 不要在页面组件中散落 fetch。
3. 所有页面处理 loading。
4. 所有页面处理 error。
5. 所有页面处理 forbidden。
6. 所有列表页面处理 empty。
7. 所有 destructive action 需要确认。
8. 根据后端返回权限显示按钮。
9. 不能只依赖前端权限控制。
10. API base URL 使用环境变量。

## M11.4 测试 / 构建要求

至少保证：

```bash
make frontend-build
```

如果已有 lint/test：

```bash
npm run lint
npm run test
```

## M11.5 验收标准

```text
完整前端流程可用
未登录会跳转登录页
无权限页面显示 forbidden
接口错误有提示
文档相关功能能通过页面操作完成
```

---

# M12：Docker 部署与安全加固

## M12.1 目标

完成开发环境和生产环境 Docker 部署。

支持：

- backend
- frontend
- postgres
- redis
- minio
- office-collab
- nginx
- optional keycloak dev profile

## M12.2 docker-compose

开发 compose：

```text
deploy/docker-compose.yml
```

生产 compose：

```text
deploy/docker-compose.prod.yml
```

开发环境包含：

- backend
- frontend
- postgres
- redis
- minio
- office-collab
- keycloak，可选 profile
- nginx，可选

生产环境包含：

- backend
- frontend static
- postgres
- redis
- minio or external S3
- office-collab
- nginx

## M12.3 Nginx 要求

Nginx 负责统一入口。

路由建议：

```text
/                  frontend
/api/              backend
/office-collab     office-collab
```

需要支持：

- WebSocket upgrade
- 大文件上传限制
- 反向代理 timeout
- Office 保存回调
- gzip，可选

## M12.4 环境变量文档

提供：

```text
deploy/env/app.env.example
```

必须包含：

```text
APP_ENV
HTTP_ADDR
PUBLIC_APP_URL
PUBLIC_API_URL
DATABASE_URL
REDIS_ADDR
S3_ENDPOINT
S3_ACCESS_KEY
S3_SECRET_KEY
S3_BUCKET
S3_USE_SSL
SESSION_SECRET
OIDC_ENABLED
OIDC_ISSUER_URL
OIDC_CLIENT_ID
OIDC_CLIENT_SECRET
OIDC_REDIRECT_URL
OFFICE_COLLAB_ENABLED
OFFICE_COLLAB_PUBLIC_URL
BACKEND_INTERNAL_URL
MAX_UPLOAD_BYTES
MARKDOWN_SNAPSHOT_UPDATE_INTERVAL
```

## M12.5 安全加固

必须检查：

1. cookie HttpOnly。
2. 生产环境 cookie Secure。
3. SameSite 合理配置。
4. CORS 限制来源。
5. 上传文件大小限制。
6. MIME 校验。
7. token hash 存储。
8. secret 不进 git。
9. 日志不输出 secret。
10. callback 设置 timeout。
11. 对外 API 有统一错误格式。
12. 管理员接口需要 is_admin。
13. nginx 限制上传大小。
14. WebSocket 连接校验权限。
15. 删除操作需要权限。

## M12.6 README / docs

更新：

```text
README.md
docs/deployment.md
docs/architecture.md
docs/permission.md
docs/markdown-collab.md
```

必须说明：

- 开发环境启动
- 生产环境启动
- 环境变量
- OIDC 配置方式
- Office 编辑器配置方式
- MinIO / S3 配置方式
- 数据持久化 volume
- 备份建议
- 常见问题

## M12.7 验收标准

```bash
docker compose -f deploy/docker-compose.yml up -d
```

开发环境可以启动。

```bash
docker compose -f deploy/docker-compose.prod.yml up -d
```

生产 compose 可以启动，或至少配置完整并通过 compose config 校验。

检查：

```bash
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.prod.yml config
```

服务可用：

```text
可以登录
可以上传文档
可以在线编辑 Office 文档
可以编辑 Markdown
可以分享文档
可以查看版本
可以查看审计日志
重启容器后数据不丢
```

---

# 5. 当前迭代任务

M12 完成后的体验修补按计划逐个执行，不混合提交。

## Plan 2：Markdown 编辑器在线成员显示

状态：已完成。

范围：

- 使用现有 Markdown WebSocket presence 数据。
- Markdown 编辑器显示当前在线人数。
- 当前用户标记为“我”。
- 每个在线用户显示可编辑 / 只读状态。
- 不修改进入编辑器的入口逻辑。
- 不修改 Office 编辑器内部协同成员展示。

验收标准：

```text
打开 Markdown 编辑器后能看到在线人数
当前用户显示在成员列表中并标记“我”
同一文档的其他查看/编辑用户显示在成员列表中
每个成员都显示可编辑或只读
前端构建通过
```

## Plan 3：简化进入编辑器操作

状态：已完成。

范围：

- 文档列表和详情页减少进入编辑器的操作步骤。
- 按文档类型选择合适的打开目标。
- 保留下载、权限、分享、版本等管理入口。

当前记录：

- 已修复文档列表标题直达编辑器导致权限 / 分享入口不明显的问题：标题进入详情页，列表操作区直接显示 owner 的权限 / 分享入口。

## Plan 4：页面视觉优化

状态：已完成。

范围：

- 优化登录页、文档列表、详情页、Markdown 编辑器和 Office 编辑器外层页面的视觉层次。
- 保持中后台工具型信息密度，不做营销落地页。
- 保持中文界面一致性。
- 不改变后端接口和权限逻辑。
- 不实现 Plan 3 的入口简化。

验收标准：

```text
登录页和主页面视觉更统一
列表、空状态、按钮、编辑器外层有更清晰的层次
移动端布局不溢出
前端构建通过
```

## Plan 5：权限授权时可选择用户

状态：已完成。

范围：

- 权限管理页授权时可以查看可授权用户。
- 支持按邮箱或显示名搜索用户。
- 授权表单不再要求操作者手动猜测用户 ID。
- 后端新增用户查询接口时必须要求登录态，并避免返回密码 hash、session、token 等敏感信息。
- 保持文档权限授予仍然通过统一权限判断，只有 owner / manager 可授权。

验收标准：

```text
权限管理页能搜索用户
搜索结果显示邮箱和显示名
搜索结果不显示当前登录用户
搜索结果显示候选用户当前已有权限或未授权状态
当前权限列表显示用户显示名和邮箱，不直接暴露为难以识别的用户 ID
选择用户后可以授予只读或可编辑权限
无权限用户不能查询或授权
前端构建和后端测试通过
```

## Plan 6：页面视觉年轻化

状态：已完成。

范围：

- 在 Plan 4 的基础上继续优化视觉风格，让界面更轻、更鲜明。
- 使用更明快的色彩、按钮状态、徽标和背景层次。
- 保持中后台工具型信息密度，不做营销页。
- 不改变后端接口、权限逻辑或进入编辑器流程。
- 不实现 Plan 5。

验收标准：

```text
页面整体视觉比 Plan 4 更年轻明快
按钮、文件徽标、列表 hover 状态更清晰
登录页和主页面风格一致
前端构建通过
前端容器实际返回新 CSS 资源
```

## Plan 7：限制上传类型并验证 Excel 编辑链路

状态：已完成。

范围：

- 上传文件类型仅允许 `docx`、`xlsx`、`md`、`txt`。
- 前端文件选择器增加同样的 accept 限制。
- 后端继续以扩展名和 MIME 双重校验为准。
- 验证 xlsx 文档可以取得 Office session、打开编辑器并保存新版本。
- 不实现 Plan 5。

验收标准：

```text
允许上传 docx/xlsx/md/txt
拒绝未列入白名单的扩展名
权限和审计逻辑不变
xlsx Office session 正常生成
xlsx 编辑器可访问受控下载地址
前端构建和后端测试通过
```

## Plan 8：Office 编辑器替换 POC

状态：已完成。

方向：

- 当前产品重点先收敛到 `xlsx` 和 `md`。
- `xlsx` 必须支持在线多人共享编辑，是 Office 替换的高优先级目标。
- `md` 后续升级为类飞书文档的富文本协同编辑体验。
- `doc` / `docx` 保留上传、下载和后续兼容空间，但不作为当前编辑体验优化重点。

范围：

- 选择自托管开源 Office 编辑器候选方案优先做 POC。
- 优先验证 Casual Office；如果无法满足基本链路，再评估 Collabora Online + WOPI。
- POC 必须复用现有文档、权限、版本、审计和对象存储边界。
- 不自研 docx / xlsx 编辑器。
- 已移除旧 Office 提供商代码、配置、容器和回退路径；不得重新引入。
- 不实现完整迁移，只完成候选方案可行性验证和最小接入。

验收标准：

```text
docx 可以通过候选编辑器打开
xlsx 可以通过候选编辑器打开
editor/owner 可以编辑并保存生成新版本
viewer 只能只读打开
无权限用户不能获取编辑器 session/config/WOPI 信息
保存动作写入审计日志
前端构建和后端测试通过
记录是否建议进入正式替换计划
```

当前 POC 结果：

- 已新增 Casual Office iframe 方式的 docx / xlsx 最小接入。
- 已验证 docx / xlsx 可以获取 Casual session、下载内容、保存回后端并生成新版本。
- 已验证未登录用户不能获取 Office session。
- 已验证前端构建和后端测试通过。
- 已在编辑页增加中文 Office 编辑器标识，并明确提示 doc / xls 暂不支持 Casual POC。
- 已新增 xlsx 协同 session 权限接口，返回 view/write role、room 和协同服务配置状态。
- 已将 xlsx 编辑页从 iframe embed 切到 React 直渲染 CasualSheets，并等待协同 session 后再挂载编辑器，确保 collab prop 首次挂载生效。
- 已新增自托管 office-collab Hocuspocus/Yjs WebSocket 服务，并通过 cookie 转发到后端校验 xlsx view/write 权限。
- 开发 compose 已启用 `ws://localhost:1234`，xlsx session 可返回 `enabled: true`。
- 已验证 xlsx 编辑页浏览器烟测：表格 canvas 正常渲染、无英文菜单栏、协同 WebSocket 返回 101。
- 已新增 xlsx workbook snapshot 实时同步：编辑后通过 Hocuspocus/Yjs 房间广播当前 workbook，另一端无需保存和刷新即可接收并应用。
- 已验证双页面 xlsx 协同烟测：两个页面均显示协同已连接，编辑后发送端产生 WebSocket 数据帧，接收端收到对应数据帧。

当前限制：

- Excel 当前 Casual React POC 已实现 snapshot 级实时同步；还不是单元格级 CRDT 合并，并发编辑同一区域时以后到达的 workbook snapshot 为准。
- office-collab 当前为单实例内存协同状态；重启会丢失实时 Yjs 状态，最终保存仍走后端 office/content 版本链路。
- Word 当前仍走 Casual iframe POC，只支持单人编辑保存，不支持多人实时共享编辑。
- CasualSheets 的 `lazyPlugins` 暂时关闭以避开缺失的 `@univerjs/docs-mention-ui` 懒加载白屏问题；后续需要恢复高级表格能力时单独修复插件清单或升级依赖。
- 还没有支持旧格式 doc / xls 的 Casual 打开链路。
- 旧 Office 提供商已按硬约束移除，不保留回退路径。

下一阶段：Excel 协同 POC：

```text
已新增自托管 Hocuspocus/Yjs WebSocket 服务
已新增后端 Office collab 权限校验接口，基于现有 PermissionService 判断 view/write
已将 xlsx 前端从 iframe embed 切到 React 直渲染 CasualSheets
已接入 CasualSheets collab prop 连接 Hocuspocus 房间
viewer 以 view role 加入，只能接收远端更新
editor/owner 以 write role 加入，可以广播修改
保存仍走后端 office/content，并继续生成版本和审计
前端显示协同连接状态和在线编辑限制
```

---

## Plan 9：账户注册与角色区分

状态：已完成。

范围：

- 完善前端注册入口，复用后端本地注册接口。
- 注册成功后自动登录并进入文档列表。
- 后续区分管理员和普通用户的导航、入口和管理能力。
- 管理员能力必须通过后端权限判断，不只做前端隐藏。

验收标准：

```text
未登录用户可以从登录页进入注册页
注册页校验显示名、邮箱和密码
邮箱重复时显示明确错误
注册成功后自动登录并进入文档列表
管理员和普通用户后续有明确的菜单与能力边界
```

当前记录：

- 登录页已提供注册入口，注册页复用后端本地注册接口。
- 注册页已校验显示名、邮箱和至少 8 位密码。
- 邮箱重复和无效输入会显示明确错误。
- 注册成功后自动登录并进入文档列表。
- 文档列表仅对管理员显示审计日志入口，审计日志接口后端继续校验 `is_admin`。
- 管理员 / 普通用户更细的菜单与能力边界留到后续产品化迭代。

---

## Plan 10：Markdown 富文本协同编辑

状态：已完成。

方向：

- Markdown 编辑体验升级为类似飞书共享文档的富文本编辑。
- 继续保留 `.md` 文件作为存储和下载格式，不引入闭源 SaaS。
- 富文本编辑必须支持多人实时协同、在线成员状态和权限控制。

范围：

- 评估并接入开源富文本编辑框架，优先考虑 ProseMirror / TipTap / Milkdown 这类可和 Yjs 集成的方案。
- 将现有 Markdown 源码编辑页面升级为所见即所得编辑体验。
- 支持基础块和行内格式：标题、正文、粗体、斜体、删除线、引用、代码块、列表、链接、表格。
- 复用现有 Markdown 权限：owner/editor 可编辑，viewer 只读。
- 复用现有 Markdown WebSocket 或新增 Yjs 协同通道，但必须经过统一权限校验。
- 保存仍写回后端 Markdown 内容并生成版本。
- 保留必要的 Markdown 导入 / 导出链路。

不做：

- 不实现完整飞书文档所有能力。
- 不实现复杂评论、任务看板、附件、数据库表格。
- 不影响 xlsx 编辑器。

验收标准：

```text
md 文件打开后显示富文本编辑器
editor/owner 修改内容后其他在线用户无需保存和刷新即可看到
viewer 只读并能实时看到更新
在线成员状态可见
保存后后端 Markdown 内容更新并生成版本
下载 md 文件内容可读
前端构建和相关后端测试通过
```

当前记录：

- 已将 Markdown 编辑页从源码 textarea 升级为 ProseMirror 富文本编辑器。
- 已支持标题、正文、引用、代码块、粗体、斜体、删除线、行内代码、链接、无序列表、有序列表和表格的基础编辑。
- 富文本内容会序列化为 Markdown 字符串，继续复用现有 Markdown WebSocket、权限校验、presence 和快照持久化链路。
- viewer 打开时编辑器只读，editor / owner 可以编辑并向同文档在线用户广播更新。
- 已新增显式保存按钮，保存当前 Markdown 内容到后端并生成文档版本。
- 分享链接访问 Markdown 时也复用同一富文本编辑器，viewer 链接只读，editor 链接可保存并继续生成版本。
- 当前仍是 Markdown 字符串快照级协同，不是 ProseMirror step / Yjs XML fragment 级 CRDT 合并；并发编辑同一段落时以后到达的内容为准。

---

## Plan 11：xlsx 中文菜单和丰富编辑能力

状态：已完成。

方向：

- 在当前 xlsx 可打开、可保存、可实时同步的基础上补齐更像表格软件的编辑体验。
- 优先保证中文界面、常用单元格编辑、格式化、筛选排序、工作表操作。

范围：

- 补一个本项目自有的中文 xlsx 工具栏 / 菜单栏，避免继续使用 CasualSheets 内置英文 chrome。
- 工具栏优先支持：撤销、重做、保存、字体加粗、斜体、下划线、字号、文本颜色、填充色、对齐、数字格式。
- 逐步恢复或替代 `lazyPlugins` 依赖的高级能力：筛选、排序、批注、数据校验、工作表管理。
- 修复 `lazyPlugins` 依赖缺失导致白屏的问题，恢复前必须验证不会破坏实时协同。
- 菜单和工具栏动作必须走 CasualSheets / Univer 命令接口，不自研 xlsx 编辑内核。
- 继续保留当前 xlsx 实时同步和保存版本链路。

不做：

- 不自研 xlsx 文件解析或编辑内核。
- 不在本计划内处理 Markdown 富文本。
- 不重新引入已移除的 Office 提供商或任何回退配置。

验收标准：

```text
xlsx 编辑页有中文菜单栏或工具栏
常用格式化命令可操作当前选区
编辑和格式化后其他在线用户无需保存和刷新即可看到
保存后生成新版本
viewer 只读时工具栏编辑动作不可用
前端构建通过
浏览器烟测无白屏、无英文硬编码菜单、无运行时异常
```

当前记录：

- 已新增项目自有的 xlsx 中文工具栏，继续关闭 CasualSheets 内置 chrome，避免内置英文菜单。
- 工具栏通过 CasualSheets / Univer 命令接口提供撤销、重做、保存、粗体、斜体、下划线、字号、文字色、填充色、水平对齐和数字格式。
- 格式修改继续触发既有 workbook snapshot 协同广播；保存继续调用既有 office/content 接口，生成版本和审计记录。
- viewer 会同时使用表格只读模式，并禁用工具栏所有编辑和保存控件。
- 已浏览器烟测编辑态工具栏和格式化保存：页面无运行时异常，保存状态为“已保存”，版本数从 1 增加到 2。

---

## Plan 12：用户中心与本地密码修改

状态：已完成。

范围：

- 新增用户中心页面，展示当前用户显示名、邮箱、登录方式和角色。
- 本地账号支持输入当前密码后修改新密码。
- OIDC 登录账户不在本系统内修改密码，提示由身份提供方管理。
- 修改密码必须走后端登录态校验，不能只在前端处理。
- 密码继续使用现有 hash 逻辑存储，不能记录明文密码。

验收标准：

```text
已登录用户可以从文档页进入用户中心
本地账号可以修改密码
当前密码错误时修改失败
OIDC 账号不显示本地修改密码表单
修改密码后旧密码不能登录，新密码可以登录
后端测试和前端构建通过
```

当前记录：

- 已新增 `PUT /api/auth/password`，通过当前 session 获取用户并校验当前密码。
- 已新增用户中心页面 `/profile`。
- 已在文档列表和审计日志页增加用户中心入口。
- 修改密码成功写入审计动作 `auth.password_change`，不记录任何密码内容。

后续未完成项：

- 用户中心暂不支持头像或个人资料扩展字段。
- 暂不支持 OIDC 账号绑定、解绑或手动合并本地账号。
- 暂不支持查看和撤销当前用户的其他登录 session。
- 本地账号修改密码后暂不强制失效其他已登录 session。

---

## Plan 13：用户中心显示名修改

状态：已完成。

范围：

- 用户中心基本信息支持修改当前用户显示名。
- 修改显示名必须走后端登录态校验。
- 更新后前端当前用户信息同步刷新。
- 显示名不能为空，后端限制最大长度。

不做：

- 不支持头像或个人资料扩展字段。
- 不支持修改邮箱。
- 不支持 OIDC 账号绑定、解绑或合并。
- 不处理其他登录 session 管理。

验收标准：

```text
已登录用户可以在用户中心修改显示名
空显示名保存失败
保存成功后用户中心显示新的显示名
后端测试和前端构建通过
```

当前记录：

- 已新增 `PUT /api/auth/profile`，通过当前 session 获取用户并更新 `display_name`。
- 已新增审计动作 `auth.profile_update`，不记录敏感信息。
- 已在用户中心基本信息面板增加显示名修改表单。

---

## Plan 14：用户中心会话管理

状态：已完成。

范围：

- 用户中心展示当前用户有效登录 session。
- 会话列表只返回 session id、创建时间、过期时间和是否当前会话。
- 当前用户可以撤销自己的其他登录 session。
- 禁止撤销当前 session。
- session token 和 token hash 不返回前端、不写审计 metadata。

不做：

- 不新增 session IP、User-Agent、设备名字段。
- 不支持管理员代用户撤销 session。
- 不支持一键退出全部其他设备。

验收标准：

```text
用户中心可以看到当前会话和其他会话
撤销其他会话后该 session 不能继续访问接口
撤销当前会话失败
后端测试和前端构建通过
```

当前记录：

- 已新增 `GET /api/auth/sessions`。
- 已新增 `DELETE /api/auth/sessions/:id`。
- 已新增审计动作 `auth.session_revoke`，仅记录 session id。
- 已在用户中心增加会话管理面板。

---

## Plan 15：编辑器页面显示与实用性优化

状态：已完成（2026-08-05）。

目标：

- 优化页面在桌面、平板和手机上的信息层级、可读性与连续操作体验。
- 以 Markdown 富文本编辑和 xlsx 表格编辑为最高优先级，保持现有权限、保存、版本和协同边界不变。

范围：

- 全局页面：统一标题、返回入口、状态提示、空/错误态和主要操作的视觉优先级；确保 375px、768px、1440px 宽度下无横向溢出。
- Markdown：改善工具栏的分组、悬停/禁用/选中反馈和小屏折叠策略；让在线成员、连接状态、保存状态与编辑区保持可扫读；优化源码辅助区域，使其在小屏上位于编辑区之后且不挤压主编辑空间。
- Markdown：补齐键盘快捷键提示、链接插入的无阻塞输入体验，以及表格插入后继续编辑、撤销和保存的浏览器回归用例。
- xlsx：改善中文工具栏的分组、按钮文案、禁用反馈和窄屏横向滚动策略；确保表格画布始终优先获得可用高度和宽度。
- xlsx：优先验证单元格输入、选区格式化、撤销/重做、保存状态、只读禁用和双页面实时同步；高级筛选、排序、批注、数据校验与工作表管理仅在依赖可稳定加载后逐项开放。
- 为 Markdown 与 xlsx 增加可重复的 Playwright 浏览器回归脚本，覆盖登录、上传、打开、编辑、保存、权限只读和关键小屏布局。

不做：

- 不修改后端权限模型、版本模型或存储格式。
- 不自研 docx / xlsx 文件解析或编辑内核。
- 不在本计划内引入新的 Office 提供商。

验收标准：

```text
375px、768px、1440px 页面无横向溢出，主要操作始终可见或可达
Markdown 富文本编辑、格式化、表格、保存、下载、只读状态均可通过浏览器回归验证
xlsx 可输入单元格、应用常用格式、撤销/重做、保存并生成新版本，viewer 工具栏不可编辑
两名 editor 打开同一 xlsx 后可看到实时同步，关键操作无白屏和运行时异常
前端构建、后端相关测试和 Playwright 回归均通过
```

当前完成记录（2026-08-05）：

- Markdown 富文本工具栏补齐分组、选中/禁用/悬停反馈、键盘快捷键提示和不阻塞的链接插入表单；移动端工具栏可横向滚动，源码辅助区域位于编辑区之后。
- xlsx 工具栏按历史保存、文字格式、对齐数字分组，补齐可访问名称、禁用状态与窄屏滚动策略；画布保留最小可用宽度，编辑器就绪后会聚焦工作簿。
- 增加 Playwright 回归：Markdown 覆盖编辑、加粗、表格、链接、保存、下载和 1440px/768px/375px 布局；xlsx 覆盖上传、保存、editor 双页协同连接、viewer 只读与 375px 布局。
- 已通过 `pnpm lint`、`pnpm build`、`E2E_BASE_URL=http://localhost:3001 pnpm --config.verify-deps-before-run=false exec playwright test` 与 `make backend-test`。

---

## Plan 16：清新界面、头像与管理员控制台

状态：已完成（2026-08-05）。

目标：

- 重构主要页面视觉，形成简洁、清新且在桌面/移动端均易用的界面。
- 用户可管理自己的头像；管理员可安全管理用户与 MinIO 存储。

范围：

- 用户资料新增头像上传、查看和替换；头像只允许安全的图片类型与大小，并通过登录态访问。
- 新增管理员控制台：用户列表、搜索、账号来源/状态/角色查看；仅可为本地账号重置密码，OIDC 账号明确不可操作。
- 管理员可查看对象存储总对象数、总占用和对象列表；支持删除已确认的对象，并写入不包含敏感信息的审计日志。
- 统一文档列表、用户中心和管理后台的页头、卡片、空状态与移动端布局。

不做：

- 不允许管理员读取、导出或显示密码、session token、S3 密钥或其他 secret。
- 不提供 OIDC 密码重置，不绕过既有文档权限。
- 不在本计划中修改 Office 编辑内核或文档存储格式。

验收标准：

```text
用户可以上传、替换并查看自己的头像
普通用户不能访问任何管理员 API 或后台页面数据
管理员可以查询全部用户并仅重置 local 用户密码
管理员可以查看 MinIO 用量和对象列表，并能确认删除对象
管理员操作有审计记录且不包含密码或 secret
主要页面在 375px、768px、1440px 无页面横向溢出
后端测试、前端构建和管理员浏览器回归通过
```

当前完成记录：

- 用户中心支持 PNG、JPEG、WebP 头像上传和替换（最大 2MB），头像经登录态接口访问，旧头像对象会在替换后清理。
- 新增 `/admin` 管理控制台，提供用户查询、本地账号密码重置、OIDC 账号不可重置提示，以及 MinIO 对象数、占用、对象列表和确认删除。
- 管理员接口全部经过服务端 `is_admin` 路由保护；密码、session token 和 MinIO 密钥均不返回或写入审计 metadata。
- 新增管理员密码重置回归；已通过前端 lint/build、后端全量测试与现有 Playwright 浏览器回归。

---

## Plan 17：Casual Office 原生协同与用户发现修复

状态：阶段性完成，待继续验收。

> 本计划替换当前仅使用 SDK / iframe 的 Office 编辑器集成。实施前必须完成每个 POC 验收项；不得恢复 ONLYOFFICE。

### 17.1 用户发现与权限管理

- `GET /api/users` 改为受控分页查询，权限页默认加载首批用户并提供“加载更多”与搜索，不能只展示固定 20 人。
- 保留当前用户排除、已授权状态和 owner/editor/viewer 后端校验。
- 验收：新建用户无需精确搜索即可在权限页找到；超过一页时可继续加载；授权后两端立即取得正确权限。

### 17.2 Casual Docs 原生服务 POC（`.docx` / `.md`）

- 在 compose 中增加自托管 `casualoffice/docs` 服务，并通过前端同源路径反向代理；配置 Redis 持久化与每文档独立房间。
- 以 Casual Docs 的 WOPI 或 JWT-API host integration 对接本系统：每次打开、读取、写回、协同 WebSocket 建立前均由 Go 统一权限服务决定 viewer/editor。
- `.docx` 使用 Docs 原生 OOXML 链路；`.md` 使用 Docs 官方 WASM Markdown 导入/导出链路，保存后仍为 `.md`。
- 验收：两个用户分别以 editor/viewer 打开同一 `.docx` 和 `.md`；编辑内容、用户名称、远程光标实时同步；viewer 无法修改；保存生成版本并写入审计日志。

### 17.3 Casual Sheets 原生 Docker 服务 POC（`.xlsx`）

- 在 compose 中增加自托管 `casualoffice/sheets` 服务（Web + Hocuspocus + Fastify），采用其房间、seed、snapshot 和 Yjs 协同链路，不再把整本工作簿快照通过项目自建 WebSocket 广播。
- 使用受权限保护的 host integration / JWT API 提供初始 xlsx、保存回调和 editor/viewer 角色；前端通过同源代理打开对应房间。
- 验收：两个 editor 在同一 `.xlsx` 中可实时看到单元格值、格式、远程选区和名称；viewer 能看到更新但不能写入；断线重连后内容不丢失；保存生成版本与审计日志。

POC 记录（2026-08-06）：

- 已启动 Casual Sheets `0.3.4` 与基于固定 Casual Docs 源码提交 `d11605185698cfc4b16a83a975cfecc8056ac348` 的自托管镜像；两个上游 Fastify 运行日志均已脱敏 `req.url`，不记录 WOPI/JWT 查询令牌。
- 权限用户发现已完成：`GET /api/users` 使用受控分页，权限页可加载更多用户、搜索、排除当前用户并展示已授权状态。
- Sheets 已完成原生 room/seed host adapter：浏览器通过同源网关访问官方 Web + Hocuspocus，Go 对 room seed、WOPI 和 Yjs 建连都执行统一权限检查；不再由项目广播整本 workbook snapshot。
- Sheets WOPI 启动路径已修复为官方根路由；浏览器现会加载宿主上传的 xlsx 工作表，而不会进入 `/sheet/:id` 的空白工作簿路由。此前基于该错误路由的双用户同步记录作废，须在真实宿主文件链路上重新验证内容同步、viewer、远程选区、断线重连和保存/版本/审计。
- Sheets 已改为从固定上游提交及递归子模块构建，镜像内补丁会保留宿主身份、脱敏上游请求日志，并将 WOPI 会话令牌传至经 Go 权限代理的协作 WebSocket。实际双用户浏览器验证已确认宿主 xlsx 工作表加载、两端真实显示名、持续 WebSocket 连接、远程名称和远程选区；内容编辑同步、viewer、断线重连及保存/版本/审计仍待验收。
- Docs Markdown 已做独立浏览器上下文双用户验证：editor 修改会实时到达另一 editor，WOPI 保存仍保留 `.md`；Docs Yjs 建连通过 Go 代理授权。Markdown viewer、远程光标和审计/版本仍需纳入正式回归。
- Docs docx 已补充 room seed 和 WOPI save host adapter。已修复官方 Docs 将带查询参数的 `collab` WebSocket URL 错误拼接到 room seed REST 地址的问题；使用最小有效 `.docx` 的实际浏览器验证已确认受保护 seed 返回 `200`。独立 owner/viewer 均可建立 Docs WebSocket；owner 修改会实时同步到 viewer，viewer 可渲染远程光标及已签发会话中的真实用户显示名；viewer 页面无可编辑区域，WOPI 写回被服务端拒绝为 `403`。owner WOPI 保存返回 `200`、生成第 2 个版本并写入 `office.save` 审计。正式 Playwright 回归与断线重连验收仍未完成，POC 尚未标记通过。
- Casual Sheets 官方 SDK 的 `attachCollab` 是所需的单元格 mutation bridge，但当前 npm 发布的 `@casualoffice/sheets@0.20.0` 在直挂页面时动态依赖 `@univerjs/docs-mention-ui`；该包在 npm registry 不存在，导致画布不挂载。因此不能以这个不完整 SDK 发行物作为正式集成。
- 后续必须补齐 Docs docx 的断线重连与正式 Playwright 回归；随后补齐 Sheets 远程选区/名称、断线重连、保存、版本、审计，以及正式 Playwright 回归、前端构建和 compose 验收。以上全部完成前，Plan 17 不得标记完成。

### 17.4 端到端与部署验收

- 新增 Playwright 双用户回归：权限用户列表、Markdown、DOCX、XLSX 的内容同步、远程光标/名称、viewer 只读、保存和版本链路。
- 执行 compose 启动、健康检查、后端测试、前端 lint/build 和浏览器回归；记录镜像版本与配置项。

---

# 6. 每个 Milestone 的 Codex Goal 用法

## Goal：M0

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M0：项目骨架。

不要实现 M1 及之后的功能。

完成后按 TASK.md 规定的格式输出修改文件、完成内容、验证命令、测试结果、未完成项和下一步建议。
```

## Goal：M1

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M1：本地用户认证。

要求不要实现 OIDC，不要实现文档上传下载，不要实现权限系统。

完成后运行后端测试和前端构建，并按 TASK.md 规定格式输出结果。
```

## Goal：M2

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M2：OIDC 登录。

要求不要破坏已有本地登录流程，不要自动合并 email 相同的本地用户和 OIDC 用户，除非配置明确允许。

完成后运行测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M3

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M3：文档上传下载。

当前阶段可以只允许 owner 访问自己的文档，不要实现完整权限系统。

完成后运行测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M4

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M4：权限系统。

要求所有文档接口必须接入 PermissionService，不允许在 handler 中散落权限判断。

完成后运行权限矩阵测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M6

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M6：Markdown 普通编辑。

不要实现多人协同，只实现 Markdown 读取、编辑、预览、保存和下载最新内容。

完成后运行测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M7

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M7：Markdown 协同编辑。

使用 Yjs / CRDT / WebSocket 实现多人协同。必须保证权限校验，viewer 只能接收更新不能提交编辑。

完成后运行测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M8

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M8：分享链接。

要求 token 只展示一次，数据库只保存 token hash。支持过期和禁用。

完成后运行测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M9

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M9：版本管理。

要求恢复历史版本时不要覆盖旧版本，而是创建新版本。

完成后运行测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M10

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M10：审计日志。

要求审计日志不能记录密码、token、secret。管理员可以查询全局审计日志。

完成后运行测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M11

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M11：前端完善。

要求所有页面处理 loading、error、empty、forbidden 状态。API 调用集中封装，不要在组件里散落 fetch。

完成后运行前端构建和测试，并按 TASK.md 规定格式输出结果。
```

## Goal：M12

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M12：Docker 部署与安全加固。

要求完善 docker-compose、nginx、env example、部署文档和安全配置检查。

完成后运行 compose config 校验，并按 TASK.md 规定格式输出结果。
```

---

# 7. 完成后统一输出格式

每次完成一个 milestone 后，必须按下面格式输出：

```text
## 修改文件

- xxx
- xxx

## 完成内容

- xxx
- xxx

## 验证命令

- xxx
- xxx

## 测试结果

- xxx

## 未完成项

- xxx

## 下一步建议

- xxx
```

---

# 8. 当前推荐起始 Goal

第一次开发时，把下面这段给 Codex：

```text
请阅读 AGENTS.md 和 TASK.md，只完成 TASK.md 中的 M0：项目骨架。

不要实现 M1 及之后的功能。

完成后按 TASK.md 规定的格式输出修改文件、完成内容、验证命令、测试结果、未完成项和下一步建议。
```
