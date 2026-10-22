# LANdapter 常见问题

**语言：** [English](FAQ.md) · [Русский](FAQ-RU.md) · [中文](FAQ-ZH.md) · [日本語](FAQ-JA.md) · [Español](FAQ-ES.md)

## 基本问题

### LANdapter 是什么？

LANdapter 是用于集中管理 Agents、向 Windows 与 Linux 终端部署驱动或应用的客户端-服务器工具。

### 支持哪些操作系统？

Master 可在 Windows 和 Linux 上构建。当前 Agent 安装器支持 Windows 的 `.exe`、`.msi`、`.inf`，以及 Linux 的 `.deb`、`.run` 和受支持的 tar 归档。macOS 安装执行尚未实现。

### 如何进行身份验证？

管理员 REST API 要求 `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>`。Master 进程必须设置至少 32 个随机字符的 `LANDAPTER_ADMIN_TOKEN`。每个 Agent 都有在 Settings 中签发的独立凭据。不要把 API 暴露到不可信网络；应使用 TLS/WSS 和网络访问限制。

### Master 会直接提供网页吗？

不会。开发模式下前端由 Vite 提供，通常是 `http://localhost:3000`。生产部署时需构建 `web/dist/`，再由 Web 服务器或反向代理发布。

## 安装与配置

### 如何修改端口？

修改 `configs/master.yaml` 中的 `http_port` 和 `ws_port`，并同步更新 Agent 的 `master_port` 与必要时的 `master_http_port`。防火墙仅开放需要的端口。

### 数据库密码能否不写入 YAML？

建议通过 Master 进程环境变量 `DB_PASSWORD` 设置。管理员秘密使用另一个变量 `LANDAPTER_ADMIN_TOKEN`；两者必须不同。

### 为什么 Agent 连接失败？

检查 Master 是否监听、`master_host`、`master_port`、`master_http_port` 和 `master_scheme` 是否正确、token 是否与 UUID 对应，以及防火墙规则。使用 `wss` 时，TLS 代理必须支持 WebSocket 和 HTTPS 文件下载。

### WebSocket 已连接，为什么文件仍无法下载？

Agent 使用独立的 `master_http_port` 下载文件。该请求还需在 `Authorization` 中提供 Agent token，并在 `X-Agent-ID` 中提供 UUID。

### 没有 PostgreSQL 能否运行 Master？

目前不行。PostgreSQL 存储 Agent、任务和结果。

## 使用与限制

### 支持哪些安装格式？

Windows：`.exe`、`.msi`、`.inf`。Linux：`.deb`、`.run`、`.tar`、`.tar.gz`、`.tar.bz2`、`.tgz`。分发前应确认软件包和安装参数。

### 静默模式和交互模式有什么区别？

`quiet` 使用安装器的静默默认参数；`interactive` 请求安装器提供的基本交互界面。实际支持的参数取决于安装包提供商。

### Agent 重新上线后，待处理任务会自动运行吗？

目前不会。任务结果可能被记录为 `pending`，但重新连接后的自动重新投递尚未实现。

### Wake-on-LAN 如何工作？

Master 向目标 MAC 地址发送 magic packet。网卡、BIOS/UEFI、操作系统和网络都必须支持并启用 WOL；跨路由子网时不一定能工作。

## 故障排查

- **提示未配置认证：** 在 Master 进程环境中设置 `LANDAPTER_ADMIN_TOKEN` 并重启。
- **HTTP 401：** 前端/API 发送的管理员 token 与 Master 环境中的值不一致。
- **验证 Agent 时 HTTP 503：** 检查 Master 是否能访问 PostgreSQL。
- **Agent 显示离线：** 检查 Agent 日志、网络连接以及凭据是否被轮换。
- **文件列表为空或读取失败：** 确认服务能访问 `uploads/`。
- **`.inf` 安装失败：** Windows 上以管理员权限运行 Agent，并检查驱动要求及命令输出。
- **前端网络错误：** 检查 `/api/` 代理配置、Master 地址/端口及浏览器控制台。

更多信息：[INSTALL-ZH.md](INSTALL-ZH.md)、[CONFIG-ZH.md](CONFIG-ZH.md)、[README.API-ZH.md](README.API-ZH.md)。
