# API Documentation

**Languages:** [English](README.API.md) · [Русский](../ru/README.API.md) · [中文](../zh/README.API.md) · [日本語](../ja/README.API.md) · [Español](../es/README.API.md)


> **Authentication.** All administrative REST endpoints require `Authorization: Bearer $LANDAPTER_ADMIN_TOKEN`. Set this variable for the Master process to a random secret of at least 32 characters. WebSocket agents use individual tokens issued in the UI; only SHA-256 token hashes are stored in PostgreSQL. Agents download installers from `/api/v1/agent/files/{id}` with `Authorization: Bearer <agent-token>` and `X-Agent-ID: <uuid>`. Use HTTPS/WSS through a TLS reverse proxy for remote networks.


## Overview

LANdapter provides a RESTful HTTP API for managing clients and distributing software installations, plus a WebSocket endpoint for real‑time agent communication. All HTTP endpoints are served on the port configured in `master.yaml` (default `8080`). WebSocket runs on a separate port (default `8081`).

## Base URLs

- HTTP API: `https://<master-host>/api/v1/` behind a TLS proxy (or `http://localhost:8080/api/v1/` for local development)
- WebSocket: `wss://<master-host>/ws` behind a TLS proxy (or `ws://localhost:8081/ws` for local development)

---

## HTTP API Endpoints

Most management endpoints return JSON. File-download endpoints return binary content with `Content-Disposition`; errors use an HTTP status code and a plain-text message.

---

### `POST /api/v1/agents/enroll`

Issue a per-agent credential (admin bearer token required). Send `{}` to create a new agent ID, or `{"agent_id":"<existing-uuid>"}` to rotate the credential for an existing ID. The plaintext token is returned only by this response; the database stores only its SHA-256 hash. Rotating a credential invalidates the old token and closes an active connection.

**Response** – `200 OK`

```json
{
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "token": "one-time-secret",
  "agent_config": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "token": "one-time-secret"
  }
}
```

Save `agent_id` and `token` in the agent's private `configs/agent.yaml` file. Do not commit or share the token.

---

### `GET /api/v1/clients`

List all registered clients.

**Query Parameters**

| Name     | Type    | Description |
|----------|---------|-------------|
| `online` | boolean | Optional. Filter clients by online status (`true` or `false`). If omitted, all clients are returned. |

**Response** – `200 OK`

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "hostname": "pc-01",
    "os": "windows",
    "mac": "00:11:22:33:44:55",
    "online": true,
    "last_seen": "2025-01-15T10:30:00Z"
  }
]
```

**Errors**

- `500 Internal Server Error` – database query failed.

---

### `POST /api/v1/install`

Create a new installation job and send commands to selected online agents.

**Request Body** – `application/json`

```json
{
  "file_ids": ["uuid1", "uuid2"],
  "client_ids": ["client-uuid1", "client-uuid2"],
  "mode": "quiet"
}
```

| Field        | Type     | Description |
|--------------|----------|-------------|
| `file_ids`   | []string | List of file identifiers (previously uploaded via `/api/v1/upload`). |
| `client_ids` | []string | List of client UUIDs to install on. |
| `mode`       | string   | Installation mode: `"quiet"` (silent) or `"interactive"` (with UI). Default is `"quiet"`. |

**Response** – `200 OK`

```json
{
  "job_id": "job-uuid"
}
```

The job is created and stored. Online agents receive the installation command immediately. Offline targets may remain `pending`; automatic delivery when an agent reconnects is not implemented yet.

**Errors**

- `400 Bad Request` – missing required fields or invalid JSON.
- `500 Internal Server Error` – database or storage error.

---

### `POST /api/v1/upload`

Upload a file to the master server. The file is stored in the `uploads/` directory with a unique name, and metadata (original name, size, upload timestamp) is saved alongside it.

**Request** – `multipart/form-data` with a field named `file`.

**Response** – `200 OK`

```json
{
  "file_id": "generated-uuid",
  "name": "original_filename.exe"
}
```

**Errors**

- `400 Bad Request` – missing file field or file too large (limit 100 MB).
- `500 Internal Server Error` – disk write error.

---

### `GET /api/v1/files`

List all uploaded files with their metadata.

**Response** – `200 OK`

```json
[
  {
    "id": "file-uuid",
    "name": "driver.exe",
    "type": ".exe",
    "size": 10485760,
    "uploadedAt": "2025-01-15T12:00:00Z",
    "version": "1.0.0",
    "description": "Network driver"
  }
]
```

**Errors**

- `500 Internal Server Error` – cannot read upload directory.

---

### `GET /api/v1/files/{id}` and `GET /api/v1/agent/files/{id}`

Download an uploaded file as binary content. The administrative route `/api/v1/files/{id}` requires the admin bearer token. Agents must use `/api/v1/agent/files/{id}` with both headers:

```http
Authorization: Bearer <agent-token>
X-Agent-ID: <agent-uuid>
```

The response is an attachment (`Content-Disposition`). `400` means an invalid ID; `404` means the file was not found; `401` means the Agent credential is missing or invalid.

---

### `DELETE /api/v1/files/{id}`

Delete a file from the server (both the binary and its metadata).

**Response** – `200 OK`

```json
{
  "status": "deleted"
}
```

**Errors**

- `400 Bad Request` – invalid file ID.
- `404 Not Found` – file does not exist.
- `500 Internal Server Error` – deletion failed.

---

### `GET /api/v1/clients/{id}/devices`

Request a list of hardware devices attached to a specific client. The master forwards the request via WebSocket and waits up to 5 seconds for the agent’s response.

**Response** – `200 OK`

The format depends on the agent’s operating system:
- **Windows**: JSON array from `Get-PnpDevice | ConvertTo-Json`.
- **Linux**: plain text output of `lsusb`, `lspci`, `lscpu`.

Example (Windows):

```json
[
  {
    "FriendlyName": "Intel(R) USB 3.1",
    "Class": "USB",
    "Status": "OK"
  }
]
```

**Errors**

- `400 Bad Request` – invalid client UUID.
- `408 Request Timeout` – agent did not respond within 5 seconds.
- `500 Internal Server Error` – WebSocket command failed.

---

### `GET /api/v1/clients/{id}/stats`

Retrieve system statistics from a client (CPU, memory, uptime). Response structure is platform‑specific.

**Response** – `200 OK`

**Windows example**:

```json
{
  "cpu_percent": 12.5,
  "mem_available_mb": 4096,
  "mem_total_mb": 8192,
  "uptime_seconds": 3600,
  "uptime_human": "1:00:00"
}
```

**Linux example**:

```json
{
  "cpu_percent": 8.2,
  "mem_used_mb": 2048,
  "mem_total_mb": 4096,
  "uptime_seconds": 7200
}
```

**Errors** – same as `/devices`.

---

### `POST /api/v1/wol`

Send Wake‑on‑LAN magic packets to selected clients (by MAC address). The master looks up MAC addresses for the given client IDs and broadcasts UDP packets to port 9.

**Request Body**

```json
{
  "client_ids": ["uuid1", "uuid2"]
}
```

**Response** – `200 OK`

```json
{
  "status": "sent"
}
```

**Errors** – none specific; errors are logged but do not affect the HTTP response.

---

### `POST /api/v1/parse-driver` (stub)

A placeholder endpoint intended for driver discovery from a website. Currently returns a static list.

**Request Body**

```json
{
  "url": "https://example.com/drivers"
}
```

**Response** – `200 OK`

```json
[
  {"name": "Driver1.inf", "url": "http://example.com/driver1.inf"},
  {"name": "Driver2.exe", "url": "http://example.com/driver2.exe"}
]
```

---

## WebSocket Protocol

The master exposes a WebSocket endpoint at `/ws` for persistent agent connections.

### Handshake

Upon connection, the agent must send a handshake message as JSON:

```json
{
  "type": "handshake",
  "uuid": "agent-uuid",
  "token": "agent-secret-issued-by-master",
  "hostname": "my-pc",
  "os": "windows",
  "mac": "00:11:22:33:44:55"
}
```

The `token` is required and must match the per-agent credential stored by its hash in PostgreSQL. The master then updates or inserts the client record, marks it online, and begins the read/write pumps.

### Incoming Commands (from master to agent)

The master sends command messages with the following structure:

```json
{
  "type": "install",
  "job_id": "job-uuid",
  "payload": { ... }
}
```

- `type`: `"install"`, `"stats"`, or `"devices"`.
- `install` payload: `{"files": ["file-id1", ...], "mode": "quiet"}`.
- `stats` and `devices` have an empty payload.

### Outgoing Responses (from agent to master)

The agent replies with a result message:

```json
{
  "type": "result",
  "job_id": "job-uuid",
  "status": "success",
  "output": "...",
  "error": "",
  "snapshot_before": { ... },
  "snapshot_after": { ... }
}
```

For `stats` and `devices` requests, the agent sends:

```json
{
  "type": "stats",
  "job_id": "cmd-uuid",
  "data": { ... }
}
```

The `data` field contains the requested statistics or device list.

### Connection Management

- The master sends periodic ping messages (default interval ~54 seconds) to keep the connection alive.
- Agents should reply with pong automatically (handled by the WebSocket library).
- If a read timeout occurs (120 seconds), the master closes the connection.
- Agents implement exponential backoff reconnection.

---

## Data Models

### Client

| Field      | Type      | Description |
|------------|-----------|-------------|
| `id`       | uuid.UUID | Unique identifier |
| `hostname` | string    | Machine hostname |
| `os`       | string    | Operating system (e.g., windows, linux) |
| `mac`      | string    | MAC address (used for WOL) |
| `online`   | boolean   | Current online status |
| `last_seen`| timestamp | Last activity timestamp (RFC3339) |

### Job

| Field       | Type      | Description |
|-------------|-----------|-------------|
| `id`        | uuid.UUID | Job identifier |
| `files`     | []string  | List of file IDs to install |
| `created_at`| timestamp | Creation timestamp |

### Job Result

| Field               | Type       | Description |
|---------------------|------------|-------------|
| `id`                | uuid.UUID  | Result record ID |
| `job_id`            | uuid.UUID  | Associated job |
| `client_id`         | uuid.UUID  | Target client |
| `status`            | string     | `pending`, `running`, `success`, `failed` |
| `output`            | string     | Installation output (stdout/stderr) |
| `error`             | string     | Error message if failed |
| `started_at`        | timestamp  | When the command was sent |
| `finished_at`       | timestamp  | When the result was received |
| `snapshot_before`   | JSON       | System state before installation |
| `snapshot_after`    | JSON       | System state after installation |

---

## Version

Current API version: **v1**. All endpoints are prefixed with `/api/v1/`.


---

## Group policies and schedules (migration 007)

All endpoints below require the administrative Bearer token. For an existing database, first apply `make migrate-apply NAME=007_policies_schedules`.

### `GET /api/v1/policies` and `POST /api/v1/policies`

`GET` lists policies. `POST` creates a declarative install policy from previously uploaded files:

```json
{
  "name": "Windows baseline",
  "description": "Approved packages",
  "os_filter": "windows",
  "files": ["550e8400-e29b-41d4-a716-446655440000"],
  "mode": "quiet"
}
```

`os_filter` is optional and matches a case-insensitive substring of the client's OS field. `mode` is `quiet`, `normal`, or `interactive`. Policies reference uploaded packages and do not execute arbitrary shell commands.

### `DELETE /api/v1/policies/{id}`

Deletes a policy and its associated schedules. Returns `404` if the policy does not exist.

### `POST /api/v1/policies/{id}/run`

Immediately dispatches the enabled policy's packages to all matching online clients and returns `202 Accepted` with a `job_id`. Returns `409 Conflict` when no online targets match or no command can be queued. This endpoint launches real installers on matching clients.

### `GET /api/v1/schedules` and `POST /api/v1/schedules`

`GET` lists schedules. `POST` creates an interval schedule for an existing enabled policy:

```json
{
  "policy_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Hourly rollout",
  "interval_minutes": 60,
  "enabled": true
}
```

Intervals range from 5 minutes to 43200 minutes (30 days). The first execution is due after the chosen interval; the scheduler checks for due work about every 30 seconds.

### `PATCH /api/v1/schedules/{id}` and `DELETE /api/v1/schedules/{id}`

`PATCH` enables/disables a schedule with `{"enabled":false}`. `DELETE` removes it. When due, Master selects only online clients whose OS matches the policy filter and dispatches package installation. Every firing repeats the install, so validate the workflow on a test client and use trusted, idempotent installers.
