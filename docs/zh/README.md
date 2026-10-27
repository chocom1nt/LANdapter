# LANdapter

<p align="center">
  <img src="../assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.26.4+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/react-18.2-61DAFB?style=flat&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/platform-windows%20%7C%20linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/postgresql-15-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL">
</p>

**语言：** [English](../../README.md) · [Русский](../ru/README.md) · [中文](README.md) · [日本語](../ja/README.md) · [Español](../es/README.md)

**LANdapter** 是一套客户端–服务器系统，用于在局域网内集中远程安装驱动和软件。

系统由 **主节点（master）** 和 **代理（agent）** 组成。主节点管理已连接的代理、分发文件并跟踪安装进度。代理通过 WebSocket 接收命令、下载文件，并以静默或交互模式执行安装。

本项目旨在简化办公和工业网络中的工作站管理，便于在数十或数百台机器上快速更新驱动或部署应用。

---

## 功能

- **集中管理** – 所有代理连接到单一主服务器。
- **远程安装** – 向代理推送文件（EXE、MSI、INF、DEB、RUN、TAR）并以合适参数运行。
- **两种安装模式** – 静默（无界面）与交互（带用户界面）。
- **统计采集** – 系统指标（CPU、内存、运行时间）及设备列表（PnP、lsusb、lspci）。
- **历史与报告** – 每个任务写入数据库，并保存安装前后的系统快照。
- **还原点** – 在 Windows 上安装前自动创建系统还原点。
- **Web 界面** – 基于 React 的管理面板，深色主题、文件库与安装向导。
- **灵活配置** – YAML 配置文件，支持环境变量。
- **跨平台** – Master 和 Agent 支持 Windows 与 Linux。目前不支持在 macOS 上执行安装。

---

## 快速开始

### 要求

- 安装 Docker Engine 和 Docker Compose 插件
- Web 前端需要 Node.js 18+
- 只有在 Docker 外构建或运行 Go 二进制文件时才需要 Go 1.26.4+

### 准备并启动 LANdapter

在仓库根目录执行：

```bash
make quickstart
```

`make quickstart` 会创建或修复 `.env`，在缺少本地配置时复制示例文件，安装前端依赖并启动 PostgreSQL 和 Master。对于**新的 Docker volume**，PostgreSQL 会通过 SQL 迁移自动初始化表结构。已有数据库不会重复运行初始化脚本。

在不同终端安装并启动 Web UI：

```bash
make run-frontend
```

打开 `http://localhost:3000`，运行 `make copy-token`，再将剪贴板中的管理员令牌粘贴到 **Settings** 一次。浏览器只在当前会话中保存令牌。`lndp` 会自动从 `.env` 读取令牌。

构建并运行 TUI：

```bash
make build-tui
make run-tui
```

在 Settings 中为每个 Agent 签发独立凭据并保存到 `configs/agent.yaml`，然后运行 `make run-agent`。

已有安装或手动配置的 PostgreSQL 请参考[安装指南](INSTALL.md)。除非确实要删除数据库卷，否则不要运行 `docker compose down -v`。

---

## 从源码构建

## 从源码构建

### Go 二进制

```bash
make build
```

或手动：

```bash
go build -o bin/master cmd/master/main.go
go build -o bin/agent cmd/agent/main.go
```

### 前端

```bash
cd web
npm run build
```

静态资源输出到 `web/dist/`。

### Docker 镜像

```bash
# 主节点
docker build -t landapter-master .

# 代理
docker build -f Dockerfile.agent -t landapter-agent .
```

`make quickstart` 会准备环境并启动 PostgreSQL 和 Master。新的 volume 会自动初始化数据库结构。通过 `make run-frontend` 单独启动 Web UI。已有安装请参考[安装指南](INSTALL.md)。

---

## 项目结构

```text
LANdapter/
├── cmd/                  # 入口
│   ├── master/
│   └── agent/
├── internal/             # 内部包
│   ├── common/           # 公共类型、日志、配置
│   ├── master/           # 主节点逻辑（HTTP、WebSocket、处理器）
│   └── agent/            # 代理逻辑（连接、安装）
├── storage/              # 数据层（接口 + PostgreSQL）
├── migrations/           # SQL 迁移
├── web/                  # React 前端
├── docs/                 # 文档
├── configs/              # 配置示例
├── uploads/              # 上传目录（自动创建）
├── docker-compose.yml
├── makefile
├── go.mod
└── README.md
```

---

## 文档

完整文档索引见 [docs/README.md](../README.md)。请根据任务从相应指南开始：

- [安装指南](INSTALL.md)：环境要求、数据库、迁移、Master/UI 启动与 Agent 注册。
- [配置](CONFIG.md)：Master 和 Agent 配置项及环境变量。
- [架构](ARCHITECTURE.md)：组件、凭据流程、WebSocket 与当前限制。
- [生产部署](DEPLOY.md)：TLS 代理、服务加固、备份与升级。
- [API 参考](README.API.md)：REST 端点和 WebSocket 协议。
- [常见问题](FAQ.md) 与 [贡献指南](CONTRIBUTING.md)。

---

## 更多文档

- [Agent 生命周期与无人值守部署](AGENT-LIFECYCLE.md) — 注册、服务运行与安全的自动注册设计。
- [终端界面 TUI](TUI.md) — 客户端、设备、监控和软件包管理。
- [设备清单、策略与计划任务](DEVICES-POLICIES-SCHEDULES.md) — 当前功能与限制。

## 测试

单元测试：

```bash
make test
```

集成测试（需要真实数据库）：

```bash
make test-integration
```

覆盖率报告：

```bash
make test-cover
```

---

## 当前限制与路线图

- 策略目前使用操作系统子字符串过滤；持久化客户端分组和标签尚未实现。
- 仪表板显示当前指标，尚未保存历史时间序列。
- 尚不支持自动搜索、验证签名或安装驱动。
- 原生 Windows 服务/systemd 安装和自动注册流程仍是后续工作。
- 主节点 CLI（无 Web 界面管理）
- Active Directory / LDAP 集成
- 更多包格式（AppImage、Flatpak）
- 计划任务
- 改进从厂商网站解析驱动

---

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](../../LICENSE)。

---

有问题或建议？请提交 [Issue](https://github.com/chocom1nt/LANdapter/issues) 或 Pull Request。
