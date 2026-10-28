# LANdapter 生产环境部署

**语言：** [English](../en/DEPLOY.md) · [Русский](../ru/DEPLOY.md) · [中文](DEPLOY.md) · [日本語](../ja/DEPLOY.md) · [Español](../es/DEPLOY.md)

LANdapter 可以远程执行安装程序，因此错误的目标选择或权限配置可能修改受管设备。先在测试机器上验证安装包，再部署到生产环境。

## 建议拓扑

- 在专用 Linux 服务器或虚拟机上运行 Master。
- PostgreSQL 只允许 Master 与获授权的数据库管理员访问。
- Web 前端由 Nginx/Caddy 提供服务。
- 通过 TLS 反向代理公开 HTTPS/WSS；不要直接向不可信网络公开内部 8080/8081 端口。
- Agent 以服务账户运行，并仅授予安装所需权限。

## 密钥与数据库

生成两个互不相同的随机秘密：

```bash
openssl rand -hex 32
openssl rand -hex 32
```

其中一个用于 PostgreSQL 密码，另一个设置为 `LANDAPTER_ADMIN_TOKEN`。如果该变量缺失或短于 32 个字符，Master 不会启动。不要把秘密提交到 Git、写入日志、文档或前端源码。更换管理员 token 时，要同步更新 Web UI 的设置并重启 Master。Agent 的原始 token 只显示一次，PostgreSQL 只存 SHA-256 哈希。

## 数据库与升级

1. 创建专用 PostgreSQL 用户和数据库。
2. 按网络情况配置 `db.sslmode`；`disable` 只适合受控的本地开发环境。
3. 应用迁移前备份数据库。
4. 使用正确的 `PG_HOST`、`PG_PORT`、`PG_USER`、`PG_DB` 与 `DB_PASSWORD` 运行 `make migrate-up`。Makefile 通过 `psql` 顺序执行 SQL 脚本；Master 启动时不会自动迁移。
5. 记录应用版本和已应用的迁移版本。

生产环境不要不经审核直接执行 `make migrate-down`；对应的 `.down.sql` 可能删除字段或数据。

## 将 Master 作为服务运行

用项目要求的 Go 版本构建二进制，并部署程序、`configs/master.yaml`、`uploads/` 和受保护的环境文件。例如：

```dotenv
DB_PASSWORD=replace-with-database-secret
LANDAPTER_ADMIN_TOKEN=replace-with-independent-random-admin-secret
```

服务应从应用目录运行，因为配置与 `uploads/` 路径相对于当前工作目录。使用无登录能力的服务账户，设置自动重启策略、文件权限和磁盘监控。`uploads/` 应位于持久化存储中。

## TLS 反向代理

将前端静态文件发布于 `web/dist/`，并将 SPA 路由回退到 `index.html`。把 `/api/` 转发到 `http://127.0.0.1:8080`，把精确路径 `/ws` 转发到 `http://127.0.0.1:8081`，启用 HTTP/1.1 以及 `Upgrade`、`Connection` 头。WebSocket 路径是 `/ws`，并不要求末尾斜杠。如果 HTTPS 公网入口统一使用 443 端口，Agent 配置通常为 `master_scheme: "wss"`、`master_port: 443`、`master_http_port: 443`。

Nginx 片段示例（请按实际域名和证书配置 TLS）：

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Real-IP $remote_addr;
}
location = /ws {
    proxy_pass http://127.0.0.1:8081;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

从远程 Agent 验证 WebSocket 和文件下载。只有在代理实际启用 TLS 时才配置 `wss`。

## 防火墙与暴露面

- 通常只允许 TCP 443 给运维人员和 Agents 使用。
- 如果反向代理与 Master 在同一主机，将内部 8080/8081 限制为 loopback 或私有网络。
- PostgreSQL 5432 只允许授权主机访问。
- 跨公网管理机器时使用 VPN；不要公开数据库端口。
- 管理 API 需要 `Authorization: Bearer ...`。不要将管理员 token 交给不可信用户；CORS 不能替代身份验证或 TLS。

## Agent 与安装任务

在 Settings 中为每个 Agent 签发独立凭据并安全保存。Windows 驱动安装可能需要管理员权限；Linux 软件包或脚本可能需要提升权限。上传前应验证安装包来源、签名/哈希和命令参数。先针对一台测试 Agent 执行非关键安装。

**当前限制：** 发往离线 Agent 的任务可能被记录为 `pending`，但 Agent 重新连接后的自动重新投递尚未实现。不要依赖该机制保证定时部署或离线任务执行。

## 备份、监控和更新

- 定期备份 PostgreSQL，并实际测试恢复流程。
- 备份 `uploads/`，并在恢复时与数据库中的文件 ID 保持一致。
- 通过 systemd/容器平台收集 stdout/stderr 日志；对外发送前清理 token 等敏感数据。
- 监控反复重启、数据库错误、磁盘空间和长期离线的 Agent。
- 更新顺序：备份 → 部署新版本 → 应用迁移 → 重启服务 → 检查 API、UI 和测试 Agent。

首次安装见 [安装](INSTALL.md)，配置说明见 [配置](CONFIG.md)。
