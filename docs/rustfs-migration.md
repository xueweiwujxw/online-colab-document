# MinIO → RustFS

默认部署固定为 `docker.io/rustfs/rustfs:1.0.0`，可以通过 `RUSTFS_IMAGE` 显式覆盖。后端仍使用 `S3_ENDPOINT`、`S3_ACCESS_KEY`、`S3_SECRET_KEY`、`S3_BUCKET`、`S3_USE_SSL`；MinIO Go SDK 是 S3 客户端依赖，不会启动 MinIO 服务。接口、数据库 schema 和对象 key 不变。

## 新部署

使用更新后的 Compose 和 env 模板。开发端口 9000/9001 仅绑定 localhost；生产存储不暴露主机端口。RustFS 使用独立 `rustfs-data` named volume，后端等待 `/health/ready` 成功后启动。官方容器以 UID/GID 10001 运行；改为 bind mount 时需事先准备可写目录，不要使用旧 MinIO 数据目录。

生产必须设置 S3 凭据，后端与 RustFS 使用同一组变量。控制台使用 9001，S3 使用 9000。外部 S3 可通过 Compose override 同时覆盖后端 endpoint/SSL/凭据，并按部署实际移除内置服务依赖。

## 已有数据（停机迁移）

1. 保留旧版本的 Compose、镜像和环境配置。停止 backend、编辑器和其他写入方，保留旧 MinIO 服务可读；备份 PostgreSQL 和 MinIO 数据。禁止执行 `down -v`。
2. 用新卷启动 RustFS，迁移期间分配不同端口，确保新旧 S3 endpoint 都可访问。不要把 `minio-data` 重命名或直接挂载给 RustFS。
3. 使用受控的 rclone 配置建立 `old`、`new` 两个 S3 remote（provider 设为 `Other`，endpoint 分别指向旧 MinIO、新 RustFS，force_path_style 为 true）。凭据从本地受限配置或环境注入，不写入仓库或日志。
4. 对实际 `S3_BUCKET` 执行以下命令；目标应为新建空 bucket，不使用 `sync` 或删除源对象：

   ```bash
   rclone mkdir "new:${S3_BUCKET}"
   rclone copy "old:${S3_BUCKET}" "new:${S3_BUCKET}" --metadata
   rclone check "old:${S3_BUCKET}" "new:${S3_BUCKET}" --download
   ```

5. 检查命令退出码均为 0，确认对象 key、数量、大小、内容及 Content-Type。必须复制整个 bucket，包括 `documents/` 的全部历史版本、`avatars/` 和其他前缀；不能只复制数据库的当前版本。项目历史版本使用不同对象 key。若自行开启了 S3 原生 versioning、Object Lock、生命周期或 IAM 策略，须另行迁移这些配置/版本，普通 copy 不覆盖这些语义。
6. 保持 PostgreSQL 原数据与 `S3_BUCKET`，将 backend 切换到 RustFS endpoint 和凭据。启动后验证现有 docx/xlsx/md 的下载、编辑保存、历史版本恢复、头像、分享链接与管理员对象管理，再恢复访问。
7. 验收期间保留旧存储和备份。恢复写入前可切回旧配置；恢复写入后如需回滚，先停写并把新增/变更对象同步回旧存储，使数据库引用与对象保持一致，不能直接切回缺失新对象的旧桶。

参考：[RustFS 1.0.0](https://github.com/rustfs/rustfs/releases/tag/1.0.0)、[官方容器定义](https://github.com/rustfs/rustfs/blob/1.0.0/Dockerfile)、[rclone copy](https://rclone.org/commands/rclone_copy/)、[rclone check](https://rclone.org/commands/rclone_check/)。
