# LANdapter 架构说明

**语言：** [English](ARCHITECTURE.md) · [Русский](ARCHITECTURE-RU.md) · [中文](ARCHITECTURE-ZH.md) · [日本語](ARCHITECTURE-JA.md) · [Español](ARCHITECTURE-ES.md)

LANdapter 将控制平面（Master 与 API）、被管理终端（Agents）、Web 前端以及 PostgreSQL 持久化存储分离。

```text
浏览器 ── HTTP/REST + 管理员 Bearer token ──> Master :8080 ──> PostgreSQL
Agent ───── 已认证 WebSocket ──────────────> Master :8081
Agent ───── 带 Agent token 的 HTTP(S) 文件下载 ─> Master :8080
浏览器 ── HTTPS ──> TLS 代理 ──> 静态前端 + /api 和 /ws 路由
```

## 组件

- **Master（`cmd/master`）**：读取 YAML 和环境变量，连接 PostgreSQL，启动 HTTP 与 WebSocket 服务。
- **HTTP API（`internal/master/handlers.go`）**：管理客户端、安装任务、文件库、设备和统计数据、Wake-on-LAN 以及 Agent 凭据签发。管理员 API 必须提供 `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>`。
- **WebSocket（`internal/master/websocket.go`）**：在握手阶段用 UUID 与 Agent 专属 token 验证连接，并处理命令与结果。
- **Agent（`cmd/agent`、`internal/agent`）**：连接并在失败后逐渐延长重试间隔，下载获准文件，在支持的操作系统上运行安装程序并回传结果。
- **Storage（`storage`）**：定义存储接口并实现 PostgreSQL。带编号的 SQL 迁移位于 `migrations/`。
- **Web（`web/`）**：React 单页应用；开发时 Vite 默认在 3000 端口运行，并把 `/api` 转发给本地 Master。

## 凭据流程

1. 运维人员在 Master 进程环境中设置至少 32 个字符的随机 `LANDAPTER_ADMIN_TOKEN`。
2. 管理请求通过 `Authorization: Bearer <token>` 认证。
3. `POST /api/v1/agents/enroll` 创建 Agent UUID 与 token，或为现有 UUID 轮换 token。
4. PostgreSQL 仅保存 Agent token 的 SHA-256 哈希；原始 token 只在签发响应中显示一次。
5. Agent 在 WebSocket 首条 `handshake` 消息中提交 UUID 与 token。Master 以常量时间比较对应哈希。
6. 下载安装文件时，Agent 调用 `/api/v1/agent/files/{id}`，附带 `Authorization: Bearer <agent-token>` 和 `X-Agent-ID: <uuid>`。

身份验证本身不加密网络流量。在不可信网络中必须使用 TLS/WSS 并限制网络访问。

## 通信与任务流程

Agent 连接 `/ws` 并发送含 `type`、`uuid`、`token`、`hostname`、`os`、`mac` 的握手 JSON。验证通过后，Master 与 Agent 通过 WebSocket 交换 JSON 命令和响应，包括安装、统计信息和设备列表。详细字段请查看 [README.API-ZH.md](README.API-ZH.md)。

`POST /api/v1/install` 会验证文件 ID 和客户端 ID，创建任务并为每台客户端记录结果。在线 Agent 立即收到命令，结果写入 PostgreSQL。离线目标可能保留为 `pending`，但**尚未实现 Agent 重连后的自动重新投递**。

当前支持的安装类型：Windows `.exe`、`.msi`、`.inf`；Linux `.deb`、`.run`、`.tar`、`.tar.gz`、`.tar.bz2`、`.tgz`。目前不支持在 macOS 上执行安装。

## 文件与数据

上传文件存放在 `uploads/`，使用唯一 ID 命名，并配有 JSON 元数据文件。应将该目录放在持久化存储中，限制访问权限，并与 PostgreSQL 一起备份。数据库保存客户端、任务、结果和 token 哈希，不保存原始 Agent token。

## 边界与安全

- Master 和 Agent 都依赖正确的配置、网络和凭据。
- 管理 API 与 Agent 凭据签发由 `LANDAPTER_ADMIN_TOKEN` 保护。
- 每台 Agent 使用独立 token；泄露后轮换，旧凭据会失效。
- 远程执行安装程序属于高权限操作：审查文件并限制管理员、文件、网络及操作系统权限。
- 当前版本尚未提供 Master CLI、组策略或定时调度。请以 README 的路线图为准。
