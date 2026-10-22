# LANdapter Configuration

**Languages:** [English](CONFIG.md) · [Русский](CONFIG-RU.md) · [中文](CONFIG-ZH.md) · [日本語](CONFIG-JA.md) · [Español](CONFIG-ES.md)

LANdapter loads YAML configuration from `configs/`, relative to the process working directory. Supported environment variables override selected YAML fields. Settings are read at startup; restart Master or Agent after editing them.

## Master: `configs/master.yaml`

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: "" # set DB_PASSWORD in the Master process environment
  dbname: "landapter"
  sslmode: "disable"
```

| Key | Description | Typical value |
|---|---|---|
| `host` | Network interfaces for Master listeners. `0.0.0.0` listens on all interfaces. | `0.0.0.0` |
| `http_port` | HTTP REST API port used by the web UI and management clients. | `8080` |
| `ws_port` | WebSocket port used by Agents. | `8081` |
| `db.host` | PostgreSQL hostname or IP. | `localhost` |
| `db.port` | PostgreSQL TCP port. | `5432` |
| `db.user` | PostgreSQL application username. | `landapter` |
| `db.password` | Database password. Prefer `DB_PASSWORD` rather than committing it in YAML. | Empty in the example |
| `db.dbname` | Database name. | `landapter` |
| `db.sslmode` | PostgreSQL TLS policy: `disable`, `require`, `verify-ca`, or `verify-full`. | `disable` only for controlled local development |

### Master environment variables

The current code explicitly supports these overrides:

| Variable | Overrides |
|---|---|
| `MASTER_HOST` | `host` |
| `HTTP_PORT` | `http_port` |
| `WS_PORT` | `ws_port` |
| `DB_HOST` | `db.host` |
| `DB_PORT` | `db.port` |
| `DB_USER` | `db.user` |
| `DB_PASSWORD` | `db.password` |
| `DB_NAME` | `db.dbname` |
| `DB_SSLMODE` | `db.sslmode` |
| `LANDAPTER_ADMIN_TOKEN` | Required administrative API secret; not a YAML field |

`LANDAPTER_ADMIN_TOKEN` must be a random secret of at least 32 characters. It is different from the PostgreSQL password. Master refuses to start if the administrative token is missing or too short. Generate one on a system with OpenSSL, for example `openssl rand -hex 32`. Do not put it in the frontend source, Git, screenshots, or issue reports.

## Agent: `configs/agent.yaml`

```yaml
master_host: "192.168.1.100"
master_port: 8081
master_http_port: 8080
master_scheme: "ws"
uuid: "<agent-id-issued-by-master>"
token: "<one-time-agent-token>"
uuid_file: "./agent.uuid"
installer_args:
  ".exe": "/S"
```

| Key | Description | Notes |
|---|---|---|
| `master_host` | IP address or DNS name of Master reachable from this machine. | Do not use `localhost` if Master is on another host. |
| `master_port` | Master WebSocket port. | Must match `ws_port` or the public WebSocket proxy port. |
| `master_http_port` | HTTP/HTTPS port used to download installer files. | Default `8080`; may be `443` when a TLS proxy exposes both services on one listener. |
| `master_scheme` | `ws` for a trusted non-TLS network, `wss` behind a TLS reverse proxy. | Must be `ws` or `wss`. |
| `uuid` | Agent ID issued by Master Settings. | If non-empty, it overrides the UUID in `uuid_file`. |
| `token` | Secret issued for this Agent. | Required; displayed only once; never share it between machines. |
| `uuid_file` | File where an Agent UUID is persisted when no `uuid` is configured. | Default `./agent.uuid`; parent directories are created as needed. |
| `installer_args` | Map of file extension to custom installer arguments. | Currently `.exe` custom arguments affect quiet installs; MSI uses built-in arguments. |

Issue a credential from the web UI's Settings page. Copy `agent_id` to `uuid` and `token` to `token` in the Agent's private `configs/agent.yaml`. When a credential is reissued for the same UUID, the previous token becomes invalid and an active connection is closed. If the secret is lost, issue a replacement; it cannot be recovered in plaintext from PostgreSQL.

Restrict file permissions to the account running the Agent. On Linux, a common choice is directory permissions `0700` and file permissions `0600`. Do not commit `configs/agent.yaml` with a real token.

### `installer_args`

Keys use file extensions including the leading dot. The current implementation reads `.exe` from this map for quiet Windows EXE installs; if omitted, it uses `/S`. MSI arguments are currently hard-coded (`/qn` for quiet and `/qb` for interactive), so `installer_args[".msi"]` is not currently used. Always test vendor-specific installer arguments on a non-critical machine first.

### Agent environment variables

| Variable | Overrides |
|---|---|
| `MASTER_HOST` | `master_host` |
| `MASTER_PORT` | `master_port` |
| `MASTER_HTTP_PORT` | `master_http_port` |
| `MASTER_SCHEME` | `master_scheme` |
| `AGENT_UUID` | `uuid` |
| `AGENT_TOKEN` | `token` |
| `UUID_FILE` | `uuid_file` |

Complex maps such as `installer_args` should be configured in YAML. Keep the Agent token private even when environment variables are used: service definitions and process environments may be visible to privileged users.

## Local URLs vs. TLS reverse proxy

For local development, the typical addresses are:

- Frontend: `http://localhost:3000` (Vite development server).
- Master REST API: `http://localhost:8080/api/v1/`.
- Agent WebSocket: `ws://localhost:8081/ws`.
- Agent download API: `http://localhost:8080/api/v1/agent/files/{id}`.

For remote agents, terminate TLS at a reverse proxy. Proxy `/api/` to Master HTTP port 8080 and the exact path `/ws` to WebSocket port 8081. If both are exposed on the same public HTTPS port, set `master_scheme: "wss"`, `master_port: 443`, and `master_http_port: 443` in Agent configuration. Use the same hostname that matches the TLS certificate. Do not set `wss` unless the endpoint actually speaks TLS.

## Security and troubleshooting

1. Use separate random secrets for PostgreSQL and `LANDAPTER_ADMIN_TOKEN`.
2. Never expose PostgreSQL or internal Master ports directly to an untrusted network.
3. Use HTTPS/WSS and firewall rules outside a trusted isolated LAN. Authentication does not encrypt traffic.
4. For PostgreSQL over an untrusted network, use an appropriate TLS mode such as `verify-full` and configure certificates correctly.
5. If Master fails at startup, confirm that it is running from the repository/application directory, that `configs/master.yaml` exists, and that the database and all migrations are ready.
6. If Agent cannot connect, confirm `master_host`, both ports, `master_scheme`, the issued UUID/token pair, and firewall rules.

For an end-to-end setup, see [INSTALL.md](INSTALL.md). For production hardening and backups, see [DEPLOY.md](DEPLOY.md).
