# Installing and Running LANdapter

**Languages:** [English](INSTALL.md) · [Русский](INSTALL-RU.md) · [中文](INSTALL-ZH.md) · [日本語](INSTALL-JA.md) · [Español](INSTALL-ES.md)

This guide covers installing and running LANdapter in a development or test environment. For production deployment, see [DEPLOY.md](DEPLOY.md).

---

## Requirements

- **Go** 1.21 or newer – backend build
- **PostgreSQL** 15 or newer – data storage
- **Node.js** 18+ and **npm** – React frontend
- **Git** – clone the repository
- **Make** (optional, recommended) – convenience targets

On Windows, **Git Bash** or **WSL2** is recommended for Makefile commands.

---

## 1. Clone the repository

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
```

---

## 2. Set up the database

Create a PostgreSQL user and database for LANdapter:

```sql
CREATE USER landapter WITH PASSWORD 'your_password';
CREATE DATABASE landapter OWNER landapter;
```

Ensure PostgreSQL is running and reachable at the host configured for the master.

---

## 3. Configuration files

Copy the sample configs and edit them for your environment:

```bash
cp configs/master.yaml.example configs/master.yaml
cp configs/agent.yaml.example configs/agent.yaml
```

### `configs/master.yaml`

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: "your_password"
  dbname: "landapter"
  sslmode: "disable"
```

### `configs/agent.yaml`

```yaml
master_host: "localhost"
master_port: 8081
uuid_file: "./agent.uuid"
installer_args:
  ".exe": "/S"
```

If the master runs on another host, set its IP instead of `localhost`.

---

## 4. Apply database migrations

SQL scripts in `migrations/` create the schema. Apply them in order:

### Using psql

```bash
psql -h localhost -U landapter -d landapter -f migrations/001_init.up.sql
psql -h localhost -U landapter -d landapter -f migrations/002_add_mac_up.sql
psql -h localhost -U landapter -d landapter -f migrations/003_add_devices_up.sql
psql -h localhost -U landapter -d landapter -f migrations/004_add_snapshots.up.sql
```

### Using Make

```bash
make migrate-up
```

To roll back:

```bash
make migrate-down
```

---

## 5. Install Go dependencies

```bash
go mod download
```

---

## 6. Build the backend

```bash
make build
```

Or manually:

```bash
go build -o bin/master cmd/master/main.go
go build -o bin/agent cmd/agent/main.go
```

Binaries are written to `bin/`.

---

## 7. Start the master

```bash
./bin/master
# or
go run cmd/master/main.go
```

The master serves HTTP (default port 8080) and WebSocket (8081).

---

## 8. Start an agent

In another terminal (or on another machine):

```bash
./bin/agent
# or
go run cmd/agent/main.go
```

The agent connects to the master and waits for commands.

---

## 9. Frontend

Install dependencies in `web/`:

```bash
cd web
npm install
```

Development server:

```bash
npm run dev
```

Open `http://localhost:3000`. The dev server proxies `/api` to `http://localhost:8080`.

Production build:

```bash
npm run build
```

Static files are in `web/dist/` and can be served by Nginx, Caddy, etc.

---

## 10. Docker Compose

Quick full stack (PostgreSQL + master + one agent):

```bash
docker-compose up -d
```

This starts:

- PostgreSQL on port 5432
- Master on 8080 (HTTP) and 8081 (WebSocket)
- One agent connected to the master

Run the frontend separately (step 9) or serve `web/dist/` via Nginx.

---

## 11. Verify

### Master API

```bash
curl http://localhost:8080/api/v1/clients
```

Expect `[]` or a list of clients.

### Agent WebSocket

Agent logs should show:

```
Connected, waiting for commands
```

### Frontend

Open `http://localhost:3000` – the dashboard should load.

---

## 12. Troubleshooting

### Database connection errors

- Ensure PostgreSQL is running (`systemctl status postgresql` on Linux).
- Check `master.yaml` (host, port, user, password, database name).
- Confirm migrations were applied.

### Agent will not connect

- Master running and listening on 8081: `netstat -an | grep 8081`.
- Correct `master_host` in `agent.yaml`.
- Firewall allows the ports.

### Frontend issues

- Dev server running (`npm run dev`).
- Port 3000 is free.
- Check the browser console (F12) for load errors.

---

## Next steps

- Read the [API reference](README.API.md).
- Tune [configuration](CONFIG.md).
- For production, see [DEPLOY.md](DEPLOY.md).
