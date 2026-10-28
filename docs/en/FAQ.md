# Frequently Asked Questions (FAQ)

**Languages:** [English](FAQ.md) · [Русский](../ru/FAQ.md) · [中文](../zh/FAQ.md) · [日本語](../ja/FAQ.md) · [Español](../es/FAQ.md)

Common questions about installing, configuring, and using LANdapter.

---

## General

### What is LANdapter?

A client–server system for centralized driver and software deployment on many machines on a LAN. The master controls agents over WebSocket, distributes files, and tracks installations.

### Supported operating systems?

- **Master:** Windows, Linux (Linux recommended).
- **Agents:** Windows and Linux installation handlers are implemented. macOS installation execution is not currently supported.

### Is there a web UI?

Yes – a React dashboard for clients, file uploads, jobs, and results.

### Authentication?

Administrative REST endpoints require `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>` (a random secret of at least 32 characters). Agents use individual tokens issued in Settings. Authentication does not encrypt traffic: use TLS/WSS and network restrictions outside a trusted local network.

---

## Installation and configuration

### Change master ports?

Edit `http_port` and `ws_port` in `master.yaml`. Agents must use the new WebSocket port in `master_port`. Open firewall rules accordingly.

### DB password without YAML?

Set `DB_PASSWORD` in the environment. Viper overrides the YAML `password` field.

### Agent will not connect?

- Master running, port 8081 reachable.
- Correct `master_host` in `agent.yaml`.
- No firewall blocking agent → master.
- Check agent logs for the exact error.

### Agent permissions?

- Windows `.inf` drivers: run agent as administrator.
- Linux `.deb`/`.run`: needs `sudo` – configure passwordless sudo or SUID as appropriate.

### Master without PostgreSQL?

No – PostgreSQL is required. SQLite would need a new `storage.Storage` implementation.

---

## Usage

### Upload a file?

Web UI: Files page, drag-and-drop or file picker. API: `POST /api/v1/upload` (multipart, field `file`).

### Supported file types?

- Windows: `.exe`, `.msi`, `.inf`
- Linux: `.deb`, `.run`, `.tar`, `.tar.gz`, `.tar.bz2`, `.tgz`
- Extend via `installer.go`

### Quiet vs interactive install?

- **Quiet:** no UI, default answers (e.g. `/S` for EXE, `/qn` for MSI).
- **Interactive:** normal installer UI.

### Client statistics?

UI: client details → Statistics. API: `GET /api/v1/clients/{id}/stats`.

### Snapshots?

CPU, memory, and uptime captured before and after install, stored in the DB to compare impact.

### Wake-on-LAN?

Master sends magic packets to the client MAC. Client must support WOL; MAC is sent at handshake.

---

## Troubleshooting

### `failed to connect to DB`

- PostgreSQL running.
- Check `master.yaml` connection settings.
- Database exists, migrations applied.
- Verify `sslmode` if using SSL.

### Agent reconnects every minute

Normal if the master is unreachable. Check master load, port 8081, and network stability.

### Agent cannot download files

- Agent downloads from `GET /api/v1/agent/files/{id}` with its bearer token and `X-Agent-ID` header.
- File exists under `uploads/`.
- Agent can reach the HTTP(S) endpoint configured by `master_http_port` (default 8080), in addition to the WebSocket port.

### `.inf` install fails

- Run agent elevated on Windows.
- Reboot may be required after driver install.

### Web UI blank or broken

- Frontend built (`npm run build`) and served (e.g. Nginx).
- `/api` proxied to master.
- Browser console (F12) for CORS/404 errors.

---

## Development

### New file type?

1. Add a case in `internal/agent/installer.go` (`installOne`).
2. Implement install method.
3. Optional `installer_args` entries.
4. Update docs.

### New API endpoint?

1. Handler in `internal/master/handlers.go`.
2. Register in `server.go` `Start()`.
3. Update [README.API.md](README.API.md) (all language variants).

### Integration tests?

Configure a test DB, set env vars, run `make test-integration`.

---

## Misc

### Logs?

Stdout by default; `journalctl` with systemd. Redirect in your service unit or wrapper script for files.

### Upgrading?

Stop service → deploy binaries → run migrations → start service.

### Cloud deployment?

Possible only with careful hardening. Use TLS/WSS, strong secrets, a firewall or VPN, and never expose PostgreSQL. Authentication is implemented for the admin API and Agent credentials, but LANdapter is still an evolving project.

### Roadmap?

Group policies, Master CLI, scheduling, more package formats, and better driver parsing from vendor sites.

---

## Contact

Open a GitHub Issue if your question is not covered here.
