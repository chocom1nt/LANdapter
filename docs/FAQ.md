# Frequently Asked Questions (FAQ)

**Languages:** [English](FAQ.md) · [Русский](FAQ-RU.md) · [中文](FAQ-ZH.md) · [日本語](FAQ-JA.md) · [Español](FAQ-ES.md)

Common questions about installing, configuring, and using LANdapter.

---

## General

### What is LANdapter?

A client–server system for centralized driver and software deployment on many machines on a LAN. The master controls agents over WebSocket, distributes files, and tracks installations.

### Supported operating systems?

- **Master:** Windows, Linux (Linux recommended).
- **Agents:** Windows, Linux. macOS is limited/untested.

### Is there a web UI?

Yes – a React dashboard for clients, file uploads, jobs, and results.

### Authentication?

Not in the current release. Designed for trusted private networks. Use VPN or a reverse proxy with auth if needed.

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

- Master serves `GET /api/v1/files/{id}`.
- File exists under `uploads/`.
- Agent reaches master HTTP port (default 8080; verify config/code).

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

Possible if agents reach the master (VPN/SSH tunnel strongly recommended – no built-in auth).

### Roadmap?

Authentication, group policies, scheduling, more package formats, better driver parsing from vendor sites.

---

## Contact

Open a GitHub Issue if your question is not covered here.
