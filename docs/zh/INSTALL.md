# LANdapter 安装与运行指南


## 推荐的快速启动方式（Docker）

首次本地部署时，请在仓库根目录执行：

```bash
make setup
make up
make install-frontend
make copy-token
make run-frontend
```

`make setup` 会补齐 `.env` 中缺失的配置，但不会轮换已有的有效密钥；只有本地 YAML 配置不存在时才从示例复制。对于**新的 Docker volume**，PostgreSQL 会在初始化时执行 SQL 迁移。已有 volume 不会被这些初始化脚本清空或升级。运行 `make copy-token` 将管理员令牌复制到剪贴板，再在 Web Settings 中粘贴一次；`lndp` 会自动从 `.env` 读取令牌。

下面的详细步骤适用于手动配置 PostgreSQL 或已有安装。除非确实要删除数据库卷，否则不要运行 `docker compose down -v`。

**语言：** [English](../en/INSTALL.md) · [Русский](../ru/INSTALL.md) · [中文](INSTALL.md) · [日本語](../ja/INSTALL.md) · [Español](../es/INSTALL.md)

本文介绍如何在开发或测试网络中安装和运行 LANdapter。生产环境部署请参阅 [部署](DEPLOY.md)。

## 1. 环境要求

- Go **1.26.4 或更高版本**（后端）。
- PostgreSQL **15 或更高版本**。
- Node.js 18+ 和 npm（React 前端）。
- Git；建议安装 `make` 和 PostgreSQL 客户端 `psql`。
- Windows 下建议用 Git Bash/WSL2 运行 Makefile 命令；PowerShell 环境变量语法不同。

## 2. 获取源码

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
go mod download
cp configs/master.yaml.example configs/master.yaml
cp configs/agent.yaml.example configs/agent.yaml
```

上述 `cp` 和 `export` 示例适用于 Bash。PowerShell 可用 `Copy-Item` 复制文件，并用 `$env:NAME = "value"` 设置环境变量。

## 3. 创建数据库

在 PostgreSQL 中创建专用用户与数据库：

```sql
CREATE USER landapter WITH PASSWORD 'replace-with-a-strong-db-password';
CREATE DATABASE landapter OWNER landapter;
```

确认 PostgreSQL 正在运行，且 Master 的配置主机和端口可达。该数据库用户需要具备创建和修改 LANdapter 数据表的权限。

## 4. 配置密钥并执行迁移

Master 读取 `configs/master.yaml`。环境变量 `DB_PASSWORD` 会覆盖 YAML 中的数据库密码。`LANDAPTER_ADMIN_TOKEN` 是独立的管理员密钥，必须是至少 32 个字符的随机秘密；不要与数据库密码共用。

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
make migrate-up
```

请将密钥保存在秘密管理器或权限受限的环境文件中，不要提交到 Git。`make migrate-up` 会按编号执行 SQL 迁移，包括 `005_agent_credentials`。升级现有部署前先备份 PostgreSQL。`make migrate-down` 会回滚迁移，可能删除数据或字段，不应当作为日常恢复操作。

## 5. 构建并启动 Master

```bash
go mod download
make build
./bin/master
```

也可以运行 `go run cmd/master/main.go`。Master 默认在 8080 端口提供 HTTP API，在 8081 端口提供 WebSocket。Master 进程必须能读取 `DB_PASSWORD` 和 `LANDAPTER_ADMIN_TOKEN`。程序启动时不会自动执行数据库迁移，因此必须先准备数据库。

## 6. 启动 Web 前端

在另一个终端执行：

```bash
cd web
npm ci
npm run dev
```

访问 `http://localhost:3000`。Vite 开发服务器会把 `/api` 请求转发到 `http://localhost:8080`。开发服务器不应用于生产部署。生产环境执行 `npm run build`，然后通过 Nginx、Caddy 等 Web 服务器发布 `web/dist/`，并将 `/api/` 转发给 Master。

在 Settings（设置）页面输入 Master 进程使用的同一个 `LANDAPTER_ADMIN_TOKEN`，保存连接配置。

## 7. 注册并启动 Agent

1. 在网页 Settings 中为 Agent 签发独立凭据。
2. 立即复制签发结果：Agent 的明文 token **只显示一次**。
3. 在目标主机编辑 `configs/agent.yaml`，将 `agent_id` 写入 `uuid`，将 token 写入 `token`。设置正确的 `master_host`、`master_port` 和 `master_http_port`。可信的局域网可用 `master_scheme: "ws"`；经 TLS 代理访问时使用 `"wss"`。
4. 限制配置文件权限，只允许 Agent 服务账户读取。
5. 启动 Agent：

```bash
./bin/agent
# 或
go run cmd/agent/main.go
```

Agent 必须同时能够访问 WebSocket 端口和 HTTP 文件下载端口。Windows 驱动安装可能需要管理员权限；Linux 软件包安装可能需要提升权限。

## 8. Docker Compose

`docker-compose.yml` 只会启动 PostgreSQL 与 Master，**不会**启动 Web 前端或 Agents。

```bash
cp .env.example .env
cp configs/master.yaml.example configs/master.yaml
# 编辑 .env，设置两个不同的随机密钥
# 确认本机 5432 端口未被占用
docker compose up -d postgres
```

启动 Master 之前，先对容器中的数据库执行迁移。由于 Compose 默认把 `5432` 绑定到本机回环地址，需要显式加载 `.env`，并将 `POSTGRES_PASSWORD` 传给 Makefile 使用的 `DB_PASSWORD`：

```bash
set -a
. ./.env
set +a
export DB_PASSWORD="$POSTGRES_PASSWORD"
make migrate-up
```

也可以用 `psql` 按编号顺序执行 `migrations/*.up.sql`。然后：

```bash
docker compose up -d master
docker compose logs -f master
```

Compose 端口默认绑定到 `127.0.0.1`。不要直接将其公开到不可信网络。前端需单独运行；Agents 需分别部署在目标机器上。

## 9. 快速验证

```bash
curl -H "Authorization: Bearer $LANDAPTER_ADMIN_TOKEN" \
  http://localhost:8080/api/v1/clients
```

空数据库通常返回 `[]`。Agent 日志应显示连接成功并等待命令。API 细节见 [API](README.API.md)。

## 故障排查

- **数据库连接失败：** 检查 PostgreSQL 服务、主机、端口、用户、密码、数据库名称和所有迁移。
- **Master 无法启动：** 检查 `configs/master.yaml` 是否存在，Master 进程是否获得 `DB_PASSWORD`，以及 `LANDAPTER_ADMIN_TOKEN` 是否至少 32 个字符。
- **Agent 无法连接：** 检查 `master_host`、端口、防火墙，以及 token 是否属于配置的 UUID。
- **文件下载失败：** Agent 除 WebSocket 外还需要访问 `master_http_port`。
- **网页返回 401：** Settings 中保存的管理员 token 必须与 Master 进程的值完全一致。
- **复制的 `node_modules` 导致前端构建失败：** 在目标平台重新运行 `npm ci`。

接下来请阅读 [配置](CONFIG.md)、[部署](DEPLOY.md) 和 [API](README.API.md)。
