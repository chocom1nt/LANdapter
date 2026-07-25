# LANdapter Configuration

**Languages:** [English](CONFIG.md) · [Русский](CONFIG-RU.md) · [中文](CONFIG-ZH.md) · [日本語](CONFIG-JA.md) · [Español](CONFIG-ES.md)

This document describes all settings in `master.yaml` and `agent.yaml`, and how to override them with environment variables.

---

## General

LANdapter uses [Viper](https://github.com/spf13/viper) for configuration:

- YAML files (primary)
- Environment variables (override file values)
- Optional file watching (when enabled)

Configs are loaded from `configs/` relative to the working directory. A `--config` flag may be added in the future to change the path.

---

## Master (`master.yaml`)

Settings for HTTP and WebSocket servers and PostgreSQL.

### Main fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `host` | string | Listen address. `0.0.0.0` = all interfaces. | `"0.0.0.0"` |
| `http_port` | int | HTTP REST API port. | `8080` |
| `ws_port` | int | WebSocket port for agents. | `8081` |
| `db` | object | PostgreSQL settings (below). | – |

### `db` fields

| Field | Type | Description |
|-------|------|-------------|
| `host` | string | Database host (`localhost` or IP). |
| `port` | int | PostgreSQL port (default `5432`). |
| `user` | string | Database user. |
| `password` | string | User password. |
| `dbname` | string | Database name. |
| `sslmode` | string | SSL mode: `"disable"`, `"require"`, `"verify-ca"`, `"verify-full"`. Dev often uses `"disable"`. |

### Example `master.yaml`

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: "secure_password"
  dbname: "landapter"
  sslmode: "disable"
```

---

## Agent (`agent.yaml`)

How the agent connects to the master and runs installers.

### Main fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `master_host` | string | Master IP or hostname. | – |
| `master_port` | int | Master WebSocket port (must match master `ws_port`). | `8081` |
| `uuid_file` | string | Path to persistent agent UUID file (created if missing). | `"./agent.uuid"` |
| `installer_args` | object | Extra CLI args per file extension (below). | `{}` |

### `installer_args`

Maps extensions (with dot) to extra arguments passed to the installer.

Examples:

- `.exe`: `"/quiet"` instead of default `"/S"`.
- `.msi`: `"/qb"` instead of `/qn` in interactive mode.

If omitted, built-in defaults apply for the installation mode.

### Example `agent.yaml`

```yaml
master_host: "192.168.1.100"
master_port: 8081
uuid_file: "/var/lib/landapter/agent.uuid"
installer_args:
  ".exe": "/quiet"
  ".msi": "/qb"
```

---

## Environment variables

YAML values can be overridden via environment variables. Viper maps dots to underscores and uppercases names:

- `db.host` → `DB_HOST`
- `http_port` → `HTTP_PORT`
- `master_host` → `MASTER_HOST`
- `installer_args` → `INSTALLER_ARGS` (maps are awkward; prefer YAML)

**Master example:**

```bash
export DB_HOST=postgres.example.com
export DB_PASSWORD=secret
export HTTP_PORT=9090
go run cmd/master/main.go
```

**Agent example:**

```bash
export MASTER_HOST=192.168.1.200
export UUID_FILE=/tmp/agent.uuid
go run cmd/agent/main.go
```

---

## Security tips

1. **Do not commit plaintext passwords.** Use env vars or secret files in `.gitignore`.
2. **Restrict ports** with a firewall. HTTP and WebSocket have no built-in auth; keep them on trusted networks.
3. **Use TLS for PostgreSQL** in production (`sslmode: "require"` or stricter).
4. **Use strong DB passwords** – avoid defaults like `postgres`/`postgres`.
5. **On Windows**, run the agent elevated when installing drivers (`.inf`, `pnputil`).

---

## Validating configuration

Start master or agent and check logs. Parse errors or missing required fields cause startup to fail with an error message.

---

## Reloading

Configuration is loaded once at startup. Restart the process to apply changes. Hot reload may be added later.
