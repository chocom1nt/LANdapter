# API Documentation

[Russian](README.API-RU.md)

## Overview

LANdapter provides a RESTful HTTP API for managing clients and distributing software installations, plus a WebSocket endpoint for real‑time agent communication. All HTTP endpoints are served on the port configured in `master.yaml` (default `8080`). WebSocket runs on a separate port (default `8081`).

> **Authentication**: The current version does not implement authentication. It is designed for trusted internal networks. Do not expose the API to the public internet without additional security measures.

---

## Base URLs

- HTTP API: `http://<master-host>:<http-port>/api/v1/`
- WebSocket: `ws://<master-host>:<ws-port>/ws`

---

## HTTP API Endpoints

All endpoints return JSON responses. Errors are returned with appropriate HTTP status codes and a plain text error message.

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

The job is created and queued. Each selected client receives the installation command immediately if online; otherwise, the job remains pending.

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
  "hostname": "my-pc",
  "os": "windows",
  "mac": "00:11:22:33:44:55"
}
```

The master updates or inserts the client record in the database, marks it online, and begins the read/write pumps.

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

