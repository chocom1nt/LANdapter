# LANdapter Architecture

**Languages:** [English](ARCHITECTURE.md) · [Русский](ARCHITECTURE-RU.md) · [中文](ARCHITECTURE-ZH.md) · [日本語](ARCHITECTURE-JA.md) · [Español](ARCHITECTURE-ES.md)

This document describes system components, interactions, the data exchange protocol, and core design principles.

---

## Overview

```
+-------------------+          +-------------------+
|   Master Server   |          |   PostgreSQL DB   |
|  (Go + React UI)  |<-------->|                   |
+-------------------+          +-------------------+
         ^
         | WebSocket (port 8081) / HTTP API (port 8080)
         |
+-------------------+
|     Agents        |
| (Windows/Linux)   |
+-------------------+
```

**Master** – central server that:

- Exposes REST API for the web UI or direct clients.
- Accepts agent WebSocket connections.
- Stores clients, jobs, and results in PostgreSQL.
- Serves files to agents over HTTP.

**Agents** – client machines that:

- Connect to the master via WebSocket.
- Receive install, stats, and devices commands.
- Download files from the master.
- Run installations and return results.

**Database** – PostgreSQL stores:

- Clients (ID, hostname, OS, MAC, online, last_seen).
- Jobs (ID, file list, created_at).
- Job results (status, output, errors, before/after snapshots).

**Frontend** – React SPA using REST API: dashboard, clients, files, install wizard.

---

## Master components

### HTTP server

- Port: `http_port` in `master.yaml` (default 8080).
- REST routes include:
  - `GET /api/v1/clients`
  - `POST /api/v1/install`
  - `POST /api/v1/upload`
  - `GET /api/v1/files`
  - `DELETE /api/v1/files/{id}`
  - `GET /api/v1/clients/{id}/devices`
  - `GET /api/v1/clients/{id}/stats`
  - `POST /api/v1/wol`
- CORS middleware for browser requests.

### WebSocket server

- Port: `ws_port` (default 8081), path `/ws`.
- Handshake: agent sends UUID, hostname, OS, MAC.
- Master upserts client in DB and marks online.
- Ping/pong (~54 s interval).
- Commands: `install`, `stats`, `devices`.
- Incoming: `result`, `stats`, `devices`.

### Job logic

On `POST /api/v1/install`:

1. Generate job UUID.
2. Persist job (files, timestamp).
3. For each selected client:
   - Verify client exists.
   - Create `job_result` with `pending`.
   - If online, send `install` over WebSocket and set `running`.
   - Offline delivery on reconnect is planned.

On agent result, update `job_result` (status, output, error, snapshots).

### File storage

- Upload via `POST /api/v1/upload`.
- Files in `uploads/` as `<UUID>.<ext>`.
- Sidecar `<UUID>.<ext>.meta.json` (name, type, size, uploadedAt, version, description).
- Download: `GET /api/v1/files/{id}` with `Content-Disposition`.

---

## Agent components

### Connection

- Read/generate UUID from `uuid_file`.
- WebSocket connect + handshake.
- Reconnect with exponential backoff (1 s – 1 min).

### Commands

**install:**

1. Snapshot system state before install.
2. Create temp directory.
3. HTTP download each file from master.
4. Run `installer.Install` with mode `quiet` or `interactive`.
5. Snapshot after install.
6. Send result to master.

**stats:** CPU, memory, uptime (JSON on Windows, text on Linux).

**devices:** Windows `Get-PnpDevice`; Linux `lsusb`, `lspci`, `lscpu`.

### Installer

By extension:

- **Windows:** `.exe`, `.msi`, `.inf`
- **Linux:** `.deb`, `.run`, `.tar`, `.tar.gz`, `.tar.bz2`, `.tgz`

Builds commands per type and mode. `.inf` tries `pnputil`, then `dism`. `.tar` extracts and runs `install.sh` or `setup.sh`. Windows may create a restore point before install.

---

## WebSocket protocol

### Agent → master

**Handshake:**

```json
{
  "type": "handshake",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "hostname": "pc-01",
  "os": "windows",
  "mac": "00:11:22:33:44:55"
}
```

**Install result:**

```json
{
  "type": "result",
  "job_id": "job-uuid",
  "status": "success",
  "output": "installation output",
  "error": "",
  "snapshot_before": { "cpu": 12.5, "mem": 4096 },
  "snapshot_after": { "cpu": 15.2, "mem": 3800 }
}
```

**Stats/devices:**

```json
{
  "type": "stats",
  "job_id": "cmd-uuid",
  "data": { "cpu_percent": 12.5, "mem_available_mb": 4096 }
}
```

### Master → agent

**Install:**

```json
{
  "type": "install",
  "job_id": "job-uuid",
  "payload": { "files": ["uuid1", "uuid2"], "mode": "quiet" }
}
```

**Stats/devices:**

```json
{
  "type": "stats",
  "job_id": "cmd-uuid",
  "payload": {}
}
```

### Ping/pong

- Master ping ~every 54 seconds.
- Agent pong via Gorilla WebSocket.
- 120 s read timeout closes the connection.

---

## Security

There is **no authentication** today. Use only on trusted internal networks.

Recommendations:

- Isolate master and DB on a private network.
- Firewall ports 8080, 8081, 5432.
- PostgreSQL `sslmode: require` in production.
- Strong DB passwords.
- Do not expose ports to the public internet.

JWT/OAuth2 are planned.

---

## Extensibility

- New installer types: `installer.go`
- New HTTP routes: `handlers.go`, register in `server.go`
- New agent commands: `client.go` by `cmd.Type`
- Other databases: implement `storage.Storage`
- New OS support: add platform commands in the agent

---

## Job lifecycle (example)

1. User uploads a file (`/api/v1/upload`) → `file_id`.
2. User selects clients/files in the UI and starts install.
3. Master creates job and per-client results; sends commands to online agents.
4. Agent downloads, installs, reports result.
5. Master updates DB.
6. UI/API show status.
