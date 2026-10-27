# Installing and Running LANdapter


## Recommended quick setup (Docker)

For a new local installation, start with the repository root:

```bash
make quickstart
make run-frontend
make copy-token
```

`make quickstart` creates or repairs `.env` without rotating existing valid secrets, creates local YAML configs from the examples when needed, installs the frontend dependencies, and starts PostgreSQL + Master. On a **new Docker volume**, PostgreSQL runs the SQL initialization migrations automatically. An existing volume is never cleared or migrated by these initialization hooks. `make copy-token` copies the admin token so you can paste it once into web Settings; `lndp` reads it from `.env` automatically.

The detailed steps below describe manual PostgreSQL setup and existing installations. Do not use `docker compose down -v` unless you intend to delete the database volume.

**Languages:** [English](INSTALL.md) · [Русский](../ru/INSTALL.md) · [中文](../zh/INSTALL.md) · [日本語](../ja/INSTALL.md) · [Español](../es/INSTALL.md)

This guide covers installing and running LANdapter in a development or test environment. For production deployment, see [DEPLOY.md](DEPLOY.md).

---

## Requirements

- **Go** 1.26.4 or newer – backend build
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
CREATE USER landapter WITH PASSWORD 'your-database-password';
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
  password: "" # set DB_PASSWORD in the environment
  dbname: "landapter"
  sslmode: "disable"
```

### `configs/agent.yaml`

```yaml
master_host: "localhost"
master_port: 8081
master_http_port: 8080
master_scheme: "ws" # use wss behind a TLS reverse proxy
uuid: "" # filled from the credential issued by Master Settings
token: "" # per-agent secret; keep this file private
uuid_file: "./agent.uuid"
installer_args:
  ".exe": "/S"
```

If the master runs on another host, set its IP instead of `localhost`.

---

## 4. Apply database migrations

SQL scripts in `migrations/` create the schema. Apply them in order:

### Apply schema and configure secrets

Set a database password and a separate administrative token of at least 32 characters, then apply all migrations, including the per-agent credential column:

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
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

## 8. Frontend

Install dependencies in `web/`:

```bash
cd web
npm install
```

Development server:

```bash
npm run dev
```

Open `http://localhost:3000`, go to Settings, enter `LANDAPTER_ADMIN_TOKEN`, save the connection, and issue a per-agent token. The dev server proxies `/api` to `http://localhost:8080`; configure the agent credentials before starting it.

Production build:

```bash
npm run build
```

Static files are in `web/dist/` and can be served by Nginx, Caddy, etc.

---

## 9. Start an agent

In Master Settings, issue a per-agent token first and copy its `agent_id` and `token` into `configs/agent.yaml` as `uuid` and `token`. Then, in another terminal (or on another machine):

```bash
./bin/agent
# or
go run cmd/agent/main.go
```

The agent connects to the master and waits for commands.

---

## 10. Docker Compose

Quick start for PostgreSQL + Master (agents are started separately):

```bash
cp .env.example .env
# Edit .env: set different random POSTGRES_PASSWORD and LANDAPTER_ADMIN_TOKEN values
docker-compose up -d postgres
set -a
. ./.env
set +a
export DB_PASSWORD="$POSTGRES_PASSWORD"
make migrate-up
docker-compose up -d master
```

This starts PostgreSQL and Master. Published ports are bound to `127.0.0.1` by default; keep them private and use a TLS reverse proxy for remote agents. Issue a per-agent token in Master Settings, save `uuid` and `token` in `configs/agent.yaml`, and start each agent separately.

Run the frontend separately (step 8) or serve `web/dist/` via Nginx.

---

## 11. Verify

### Master API

```bash
curl -H "Authorization: Bearer $LANDAPTER_ADMIN_TOKEN" \
  http://localhost:8080/api/v1/clients
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
