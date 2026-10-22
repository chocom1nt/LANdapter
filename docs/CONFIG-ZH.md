# LANdapter 配置指南

**语言：** [English](CONFIG.md) · [Русский](CONFIG-RU.md) · [中文](CONFIG-ZH.md) · [日本語](CONFIG-JA.md) · [Español](CONFIG-ES.md)

LANdapter 从当前工作目录下的 `configs/` 读取 YAML。受支持的环境变量会覆盖 YAML 值。配置在进程启动时加载；修改后需要重启相应进程。

## Master：`configs/master.yaml`

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: ""
  dbname: "landapter"
  sslmode: "disable"
```

| 字段 | 说明 | 常用值 |
|---|---|---|
| `host` | Master 监听地址；`0.0.0.0` 表示监听所有网络接口。 | `0.0.0.0` |
| `http_port` | REST API 与 Web UI 使用的 HTTP 端口。 | `8080` |
| `ws_port` | Agents 的 WebSocket 端口。 | `8081` |
| `db.host`, `db.port` | PostgreSQL 地址与端口。 | `localhost`, `5432` |
| `db.user`, `db.password` | PostgreSQL 账户。建议通过环境变量 `DB_PASSWORD` 注入密码。 | `landapter`、示例中为空 |
| `db.dbname` | 数据库名称。 | `landapter` |
| `db.sslmode` | PostgreSQL TLS 模式：`disable`、`require`、`verify-ca`、`verify-full`。 | 仅开发环境用 `disable` |

Master 支持的环境变量包括 `MASTER_HOST`、`HTTP_PORT`、`WS_PORT`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME` 和 `DB_SSLMODE`。此外，必须单独配置 `LANDAPTER_ADMIN_TOKEN`，并使用至少 32 个字符的随机值；它不能与数据库密码相同。Docker Compose 会把 `.env` 中指定的变量传入容器；在宿主机直接运行 Go 程序时，需要把变量导出到该进程环境。

## Agent：`configs/agent.yaml`

```yaml
master_host: "192.168.1.100"
master_port: 8081
master_http_port: 8080
master_scheme: "ws"
uuid: "<agent-id-issued-by-master>"
token: "<one-time-agent-token>"
uuid_file: "./agent.uuid"
installer_args:
  ".exe": "/S"
```

| 字段 | 说明 |
|---|---|
| `master_host` | 从该 Agent 主机可访问的 Master IP 或 DNS 名称。 |
| `master_port` | WebSocket 端口；需对应 Master 的 `ws_port`，或 TLS 代理的公网端口。 |
| `master_http_port` | Agent 下载文件时使用的 HTTP/HTTPS 端口。经过 TLS 代理时可与 `master_port` 相同。 |
| `master_scheme` | 可信的无 TLS 局域网使用 `ws`；通过 TLS 代理访问使用 `wss`。远程访问请启用 TLS。 |
| `uuid` | Master 签发的 Agent ID。填写后优先于 `uuid_file` 中的 ID。 |
| `token` | Agent 专属秘密。必填；数据库不会保存明文 token。丢失后需重新签发。 |
| `uuid_file` | 未配置 `uuid` 时保存持久化 UUID 的路径；必要时会创建父目录。 |
| `installer_args` | 按文件扩展名配置安装器参数；当前 `.exe` 参数用于静默安装。 |

不要在多台主机之间复用 Agent token。为同一 UUID 重新签发 token 会立即使旧 token 失效。请限制 `configs/agent.yaml` 的读取权限，也不要将其提交到 Git、工单或截图中。

Agent 支持的环境变量：`MASTER_HOST`、`MASTER_PORT`、`MASTER_HTTP_PORT`、`MASTER_SCHEME`、`AGENT_UUID`、`AGENT_TOKEN` 和 `UUID_FILE`。复杂映射（例如 `installer_args`）应保留在 YAML 文件中。

## 地址与 TLS 反向代理

- 本地开发：Vite 前端为 `http://localhost:3000`，API 为 `http://localhost:8080`，Agent 连接 `ws://<master>:8081/ws`。
- 生产环境建议由反向代理提供 HTTPS，并把 `/api/` 转发到内部 `8080`，把 `/ws` 转发到内部 WebSocket `8081`。
- 如果代理通过 `443` 同时公开 API 和 WebSocket，Agent 通常配置 `master_scheme: "wss"`、`master_port: 443`、`master_http_port: 443`。
- 不要把内部端口或数据库直接暴露给互联网。若 PostgreSQL 经过不可信网络，请配置严格的 `db.sslmode`。

## 验证与运维

Master 和 Agent 在启动时读取配置，改动 YAML 或环境变量后应重启。排错时检查 YAML 语法、端口是否为整数、相对于工作目录的文件路径是否正确以及环境变量名称拼写。安装步骤请见 [INSTALL-ZH.md](INSTALL-ZH.md)，生产部署请见 [DEPLOY-ZH.md](DEPLOY-ZH.md)。
