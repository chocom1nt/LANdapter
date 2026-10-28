# Agent 生命周期、注册与服务运行

本文说明当前 Agent 的工作方式、为什么要显式发放凭据，以及无人值守部署工作站的推荐方向。

## 当前行为

- 通过 `make run-agent` 或 `go run ./cmd/agent` 启动时，Agent 作为前台进程运行。仓库目前没有内置的 Windows 服务安装器或 systemd unit。
- Agent 通过 WebSocket 连接 Master，并在断线后重连。Master 使用 PostgreSQL 中保存的 Agent token 哈希验证握手。
- 连接时 Agent 上报身份和基本主机信息。设备清单以及 CPU/RAM/uptime 由 Web UI/TUI 按需请求；持续遥测和主动告警尚未实现。
- 每个 Agent 都需要独立的 `uuid` 和 `token`，配置在 `configs/agent.yaml` 或 `AGENT_UUID` / `AGENT_TOKEN` 环境变量中。环境变量优先于 YAML。

## 为什么不向每台客户端自动发放 token？

如果 Master 向任何连接上来的设备发放永久凭据，局域网中的未授权主机也可能注册并接收远程安装命令。当前注册 API 受管理员 token 保护；Agent 没有已授权的注册密钥时，不能安全地调用它。每台 Agent 使用独立凭据也便于单独撤销和轮换。

## 推荐的无人值守注册设计

1. 管理员创建有有效期、只能使用一次并限定环境/组的注册码或部署包。
2. 通过 MDM、配置管理、受保护的共享目录或部署系统等安全渠道向 Agent 安装程序提供注册密钥，不要放在公开 URL 或源代码中。
3. Agent 使用注册密钥换取独立的 Agent 凭据。Master 只保存 token 哈希；原始 token 仅返回一次，并以严格权限保存在本机。
4. 根据策略，新 Agent 可以在严格规则下自动批准，也可以等待管理员审批。注册码应支持过期、次数限制和撤销。
5. 将 Agent 安装为 Windows 服务或 systemd 服务，配置开机启动、故障重启、尽可能使用低权限专用账户，并集中收集日志。

## 让当前 Agent 在后台运行

在仓库提供原生服务安装器之前，请使用组织批准的服务管理器或部署工具。Linux 的 systemd unit 应设置固定工作目录、尽可能只读的配置权限、`restart-on-failure` 和专用账户。Windows 可使用获批的服务包装器，或将 Agent 打包为原生服务。保护 `configs/agent.yaml`，不要把生产 token 放进 Git、公开安装参数、截图或日志。交互式安装程序需单独测试：后台服务可能无法访问已登录用户的桌面。

## 当前限制与路线图

- 尚未实现无需预先配置 Agent token 的自动注册。
- 当前仓库不包含原生 Windows 服务安装和 systemd unit。
- 持续遥测、事件转发、告警规则和管理员通知尚未实现；当前 UI 按需请求指标。
- 未来注册流程应使用限时一次性 bootstrap 凭据、独立 Agent 身份、审批控制、速率限制、TLS/WSS 和审计日志。
