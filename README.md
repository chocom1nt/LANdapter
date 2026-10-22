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

**Languages:** [English](README.md) · [Русский](README-RU.md) · [中文](README-ZH.md) · [日本語](README-JA.md) · [Español](README-ES.md)

**LANdapter** is a client–server system for centralized remote installation of drivers and software on a local network.

The system consists of a **master** (server) and **agents** (client machines). The master manages connected agents, distributes files, and tracks installation progress. Agents receive commands over WebSocket, download files, and run installations in quiet or interactive mode.

The project simplifies workstation administration in office and industrial networks where you need to quickly update drivers or deploy applications to dozens or hundreds of machines.

---

## Features

- **Centralized management** – all agents connect to a single master server.
- **Remote installation** – push files (EXE, MSI, INF, DEB, RUN, TAR) to agents and run them with the right parameters.
- **Two installation modes** – quiet (no UI) and interactive (with user interface).
- **Statistics collection** – system metrics (CPU, RAM, uptime) and device lists (PnP, lsusb, lspci).
- **History and reports** – each job is stored in the database with system snapshots before and after installation.
- **Restore points** – automatic restore point creation before installation on Windows.
- **Web UI** – React dashboard with dark theme, file library, and installation wizard.
- **Flexible configuration** – YAML files and environment variable support.
- **Cross-platform** – Master and Agent support Windows and Linux. macOS installation execution is not currently supported.

---

## Quick start

### Requirements

- Go 1.26.4+
- PostgreSQL 15+
- Node.js 18+ (for the frontend)
- Make (optional)

### Install and run

Clone the repository:

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
```

Download Go dependencies:

```bash
go mod download
```

Install frontend dependencies:

```bash
cd web
npm ci
cd ..
```

Before applying migrations, create a dedicated PostgreSQL user and database as a PostgreSQL administrator (for example, in `psql`):

```sql
CREATE USER landapter WITH PASSWORD 'your-database-password';
CREATE DATABASE landapter OWNER landapter;
```

Use the same database password in `DB_PASSWORD` below. If the role or database already exists, do not create it again; verify its owner and permissions instead.

Before applying migrations or starting Master, set the PostgreSQL password and a separate admin secret of at least 32 characters:

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
```

Apply database migrations:

```bash
make migrate-up
```

Start the master:

```bash
go run cmd/master/main.go
```

Start the frontend (in a separate terminal):

```bash
cd web
npm run dev
```

Open `http://localhost:3000`, go to Settings, enter `LANDAPTER_ADMIN_TOKEN`, save the connection, and issue a per-agent token. The token is displayed once; copy the returned `agent_id` and `token` into `configs/agent.yaml` as `uuid` and `token`.

Only after saving the agent credentials, start an agent in another terminal:

```bash
go run cmd/agent/main.go
```

See [docs/INSTALL.md](docs/INSTALL.md) for detailed deployment instructions.

---

## Building from source

### Go binaries

Build master and agent:

```bash
make build
```

Or manually:

```bash
go build -o bin/master cmd/master/main.go
go build -o bin/agent cmd/agent/main.go
```

### Frontend

Build the React app:

```bash
cd web
npm run build
```

Static assets are output to `web/dist/`.

### Docker images

The root `Dockerfile` builds the Master image (and includes the agent binary); `Dockerfile.agent` builds a standalone Agent image:

```bash
docker build -t landapter-master .
docker build -f Dockerfile.agent -t landapter-agent .
```

`docker-compose.yml` starts **PostgreSQL and Master only**. It does not start the web frontend or register/run agents. Copy `configs/master.yaml.example` to `configs/master.yaml`, configure `.env`, apply the SQL migrations, and then start the services as described in [docs/INSTALL.md](docs/INSTALL.md). Run the frontend separately or serve `web/dist/` from a web server.

---

## Project structure

```text
LANdapter/
├── cmd/                  # Entry points
│   ├── master/
│   └── agent/
├── internal/             # Internal packages
│   ├── common/           # Shared types, logging, config
│   ├── master/           # Master logic (HTTP, WebSocket, handlers)
│   └── agent/            # Agent logic (connection, installation)
├── storage/              # Data layer (interface + PostgreSQL)
├── migrations/           # SQL migrations
├── web/                  # React frontend
├── docs/                 # Documentation
├── configs/              # Sample configuration files
├── uploads/              # Uploaded files (created automatically)
├── docker-compose.yml
├── makefile
├── go.mod
└── README.md
```

---

## Documentation

Full documentation is indexed in [docs/README.md](docs/README.md). Start with the guide for your task:

- [Installation](docs/INSTALL.md) – requirements, database setup, migrations, Master/UI startup, and Agent enrollment.
- [Configuration](docs/CONFIG.md) – all Master and Agent settings and supported environment variables.
- [Architecture](docs/ARCHITECTURE.md) – components, credential flow, WebSocket messages, and current limitations.
- [Production deployment](docs/DEPLOY.md) – TLS reverse proxy, service hardening, backups, and upgrades.
- [API reference](docs/README.API.md) – REST routes and WebSocket protocol; translations are linked at the top of the API guide.
- [FAQ](docs/FAQ.md) and [contributing](docs/CONTRIBUTING.md).

---

## Testing

Run unit tests:

```bash
make test
```

Integration tests (require a real database):

```bash
make test-integration
```

Coverage report:

```bash
make test-cover
```

---

## Roadmap

- Group policies (select clients by group)
- Master CLI (manage without the web UI)
- Active Directory / LDAP integration
- Additional package formats (AppImage, Flatpak)
- Scheduled jobs
- Improved driver parsing from vendor websites

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

Questions or suggestions? Open an [Issue](https://github.com/chocom1nt/LANdapter/issues) or send a Pull Request.
