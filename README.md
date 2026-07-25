# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
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
- **Cross-platform** – master and agents run on Windows and Linux (the agent also supports macOS with limited functionality).

---

## Quick start

### Requirements

- Go 1.21+
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
npm install
cd ..
```

Apply database migrations:

```bash
make migrate-up
# or manually:
psql -h localhost -U postgres -d landapter -f migrations/001_init.up.sql
psql -h localhost -U postgres -d landapter -f migrations/002_add_mac_up.sql
psql -h localhost -U postgres -d landapter -f migrations/003_add_devices_up.sql
psql -h localhost -U postgres -d landapter -f migrations/004_add_snapshots.up.sql
```

Start the master:

```bash
go run cmd/master/main.go
```

Start an agent (in a separate terminal):

```bash
go run cmd/agent/main.go
```

Start the frontend (in a third terminal):

```bash
cd web
npm run dev
```

Open `http://localhost:3000` in your browser.

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

Separate Dockerfiles are provided for master and agent:

```bash
# Master
docker build -f Dockerfile.master -t landapter-master .

# Agent
docker build -f Dockerfile.agent -t landapter-agent .
```

Or run the full stack with `docker-compose` (includes PostgreSQL):

```bash
docker-compose up -d
```

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
├── Makefile
├── go.mod
└── README.md
```

---

## Documentation

Full documentation is in [docs/](docs/):

- [API Reference](docs/README.API.md) – REST and WebSocket endpoints ([Русский](docs/README.API-RU.md) · [中文](docs/README.API-ZH.md) · [日本語](docs/README.API-JA.md) · [Español](docs/README.API-ES.md)).
- [Installation guide](docs/INSTALL.md) – detailed deployment walkthrough.
- [Configuration](docs/CONFIG.md) – `master.yaml` and `agent.yaml` parameters.
- [Architecture](docs/ARCHITECTURE.md) – design, job lifecycle, WebSocket protocol.
- [Production deployment](docs/DEPLOY.md) – services, proxy, monitoring.
- [FAQ](docs/FAQ.md) – common questions and troubleshooting.

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
