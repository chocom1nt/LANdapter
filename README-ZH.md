# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.26.4+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/react-18.2-61DAFB?style=flat&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/platform-windows%20%7C%20linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/postgresql-15-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL">
</p>

**语言：** [English](README.md) · [Русский](README-RU.md) · [中文](README-ZH.md) · [日本語](README-JA.md) · [Español](README-ES.md)

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

### 环境要求

- Go 1.26.4+
- PostgreSQL 15+
- Node.js 18+（前端）
- Make（可选）

### 安装与运行

克隆仓库：

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
```

下载 Go 依赖：

```bash
go mod download
```

安装前端依赖：

```bash
cd web
npm ci
cd ..
```

执行迁移前，请以 PostgreSQL 管理员身份创建专用用户和数据库（例如在 `psql` 中执行）：

```sql
CREATE USER landapter WITH PASSWORD 'your-database-password';
CREATE DATABASE landapter OWNER landapter;
```

下方 `DB_PASSWORD` 应使用同一个数据库密码。如果用户或数据库已经存在，请勿重复创建，而应检查其所有者和权限。

在执行迁移或启动主节点前，设置 PostgreSQL 密码以及独立的、至少 32 个字符的管理员密钥：

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
```

执行数据库迁移：

```bash
make migrate-up
```

启动主节点：

```bash
go run cmd/master/main.go
```

在另一个终端启动前端：

```bash
cd web
npm run dev
```

打开 `http://localhost:3000`，进入 Settings，填入 `LANDAPTER_ADMIN_TOKEN` 并保存连接，然后为代理签发独立令牌。令牌只显示一次；请将返回的 `agent_id` 和 `token` 分别保存到 `configs/agent.yaml` 的 `uuid` 和 `token` 字段。

保存代理凭据后，再在另一个终端启动代理：

```bash
go run cmd/agent/main.go
```

详细部署说明见 [docs/INSTALL-ZH.md](docs/INSTALL-ZH.md)。

---

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

`docker-compose.yml` **只启动 PostgreSQL 和 Master**，不会启动前端或 Agent。请先将 `configs/master.yaml.example` 复制为 `configs/master.yaml`，配置 `.env` 并应用 SQL 迁移。详见 [docs/INSTALL-ZH.md](docs/INSTALL-ZH.md)。前端需单独启动，或通过 Web 服务器发布 `web/dist/`。

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

完整文档索引见 [docs/README.md](docs/README.md)。请根据任务从相应指南开始：

- [安装指南](docs/INSTALL-ZH.md)：环境要求、数据库、迁移、Master/UI 启动与 Agent 注册。
- [配置](docs/CONFIG-ZH.md)：Master 和 Agent 配置项及环境变量。
- [架构](docs/ARCHITECTURE-ZH.md)：组件、凭据流程、WebSocket 与当前限制。
- [生产部署](docs/DEPLOY-ZH.md)：TLS 代理、服务加固、备份与升级。
- [API 参考](docs/README.API-ZH.md)：REST 端点和 WebSocket 协议。
- [常见问题](docs/FAQ-ZH.md) 与 [贡献指南](docs/CONTRIBUTING-ZH.md)。

---

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

## 路线图

- 组策略（按组选择客户端）
- 主节点 CLI（无 Web 界面管理）
- Active Directory / LDAP 集成
- 更多包格式（AppImage、Flatpak）
- 计划任务
- 改进从厂商网站解析驱动

---

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE)。

---

有问题或建议？请提交 [Issue](https://github.com/chocom1nt/LANdapter/issues) 或 Pull Request。
