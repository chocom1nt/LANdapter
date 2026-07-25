# API 文档

**语言：** [English](README.API.md) · [Русский](README.API-RU.md) · [中文](README.API-ZH.md) · [日本語](README.API-JA.md) · [Español](README.API-ES.md)

## 概述

LANdapter 提供 RESTful HTTP API，用于管理客户端和分发软件安装，并提供 WebSocket 端点以与代理实时通信。所有 HTTP 端点运行在 `master.yaml` 中配置的端口（默认 `8080`）。WebSocket 使用独立端口（默认 `8081`）。

> **身份验证**：当前版本未实现身份验证，适用于受信任的内网环境。请勿在未增加额外安全措施的情况下将 API 暴露到公网。

---

## 基础 URL

- HTTP API：`http://<master-host>:<http-port>/api/v1/`
- WebSocket：`ws://<master-host>:<ws-port>/ws`

---

## HTTP API 端点

所有端点返回 JSON。错误会伴随相应的 HTTP 状态码和纯文本错误消息。

---

### `GET /api/v1/clients`

获取所有已注册客户端列表。

**查询参数**

| 名称     | 类型    | 说明 |
|----------|---------|------|
| `online` | boolean | 可选。按在线状态筛选（`true` 或 `false`）。省略则返回全部客户端。 |

**响应** – `200 OK`

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

**错误**

- `500 Internal Server Error` – 数据库查询失败。

---

### `POST /api/v1/install`

创建安装任务并向所选在线代理发送命令。

**请求体** – `application/json`

```json
{
  "file_ids": ["uuid1", "uuid2"],
  "client_ids": ["client-uuid1", "client-uuid2"],
  "mode": "quiet"
}
```

| 字段         | 类型     | 说明 |
|--------------|----------|------|
| `file_ids`   | []string | 文件标识列表（须先通过 `/api/v1/upload` 上传）。 |
| `client_ids` | []string | 要安装到的客户端 UUID 列表。 |
| `mode`       | string   | 安装模式：`"quiet"`（静默）或 `"interactive"`（带界面）。默认为 `"quiet"`。 |

**响应** – `200 OK`

```json
{
  "job_id": "job-uuid"
}
```

任务创建并排队。在线客户端会立即收到安装命令；离线客户端在下次连接时收到。

**错误**

- `400 Bad Request` – 缺少必填字段或 JSON 无效。
- `500 Internal Server Error` – 数据库或存储错误。

---

### `POST /api/v1/upload`

向主服务器上传文件。文件以唯一名称保存在 `uploads/` 目录，元数据（原始名称、大小、上传时间）一并保存。

**请求** – `multipart/form-data`，字段名为 `file`。

**响应** – `200 OK`

```json
{
  "file_id": "generated-uuid",
  "name": "original_filename.exe"
}
```

**错误**

- `400 Bad Request` – 缺少文件字段或文件过大（上限 100 MB）。
- `500 Internal Server Error` – 磁盘写入错误。

---

### `GET /api/v1/files`

列出所有已上传文件及其元数据。

**响应** – `200 OK`

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

**错误**

- `500 Internal Server Error` – 无法读取上传目录。

---

### `DELETE /api/v1/files/{id}`

从服务器删除文件（二进制文件及元数据）。

**响应** – `200 OK`

```json
{
  "status": "deleted"
}
```

**错误**

- `400 Bad Request` – 无效的文件 ID。
- `404 Not Found` – 文件不存在。
- `500 Internal Server Error` – 删除失败。

---

### `GET /api/v1/clients/{id}/devices`

请求指定客户端的硬件设备列表。主节点通过 WebSocket 转发请求，最多等待代理响应 5 秒。

**响应** – `200 OK`

格式取决于代理操作系统：
- **Windows**：`Get-PnpDevice | ConvertTo-Json` 的 JSON 数组。
- **Linux**：`lsusb`、`lspci`、`lscpu` 的纯文本输出。

示例（Windows）：

```json
[
  {
    "FriendlyName": "Intel(R) USB 3.1",
    "Class": "USB",
    "Status": "OK"
  }
]
```

**错误**

- `400 Bad Request` – 无效的客户端 UUID。
- `408 Request Timeout` – 代理在 5 秒内未响应。
- `500 Internal Server Error` – WebSocket 命令失败。

---

### `GET /api/v1/clients/{id}/stats`

获取客户端系统统计（CPU、内存、运行时间）。响应结构因平台而异。

**响应** – `200 OK`

**Windows 示例**：

```json
{
  "cpu_percent": 12.5,
  "mem_available_mb": 4096,
  "mem_total_mb": 8192,
  "uptime_seconds": 3600,
  "uptime_human": "1:00:00"
}
```

**Linux 示例**：

```json
{
  "cpu_percent": 8.2,
  "mem_used_mb": 2048,
  "mem_total_mb": 4096,
  "uptime_seconds": 7200
}
```

**错误** – 与 `/devices` 相同。

---

### `POST /api/v1/wol`

向所选客户端（按 MAC 地址）发送 Wake-on-LAN 魔术包。主节点根据客户端 ID 查找 MAC 并向端口 9 广播 UDP 包。

**请求体**

```json
{
  "client_ids": ["uuid1", "uuid2"]
}
```

**响应** – `200 OK`

```json
{
  "status": "sent"
}
```

**错误** – 无特定 HTTP 错误；错误会记录日志但不影响 HTTP 响应。

---

### `POST /api/v1/parse-driver`（占位）

用于从网站发现驱动的占位端点，当前返回静态列表。

**请求体**

```json
{
  "url": "https://example.com/drivers"
}
```

**响应** – `200 OK`

```json
[
  {"name": "Driver1.inf", "url": "http://example.com/driver1.inf"},
  {"name": "Driver2.exe", "url": "http://example.com/driver2.exe"}
]
```

---

## WebSocket 协议

主节点在 `/ws` 提供 WebSocket 端点，供代理保持长连接。

### 握手

连接建立后，代理必须发送 JSON 格式的握手消息：

```json
{
  "type": "handshake",
  "uuid": "agent-uuid",
  "hostname": "my-pc",
  "os": "windows",
  "mac": "00:11:22:33:44:55"
}
```

主节点在数据库中更新或插入客户端记录，标记为在线，并启动读写循环。

### 入站命令（主节点 → 代理）

主节点发送如下结构的命令消息：

```json
{
  "type": "install",
  "job_id": "job-uuid",
  "payload": { ... }
}
```

- `type`：`"install"`、`"stats"` 或 `"devices"`。
- `install` 的 payload：`{"files": ["file-id1", ...], "mode": "quiet"}`。
- `stats` 和 `devices` 的 payload 为空。

### 出站响应（代理 → 主节点）

代理以结果消息回复：

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

对于 `stats` 和 `devices` 请求，代理发送：

```json
{
  "type": "stats",
  "job_id": "cmd-uuid",
  "data": { ... }
}
```

`data` 字段包含请求的统计或设备列表。

### 连接管理

- 主节点定期发送 ping（默认间隔约 54 秒）以保持连接。
- 代理应自动回复 pong（由 WebSocket 库处理）。
- 读超时（120 秒）时主节点关闭连接。
- 代理使用指数退避重连。

---

## 数据模型

### Client（客户端）

| 字段        | 类型      | 说明 |
|-------------|-----------|------|
| `id`        | uuid.UUID | 唯一标识 |
| `hostname`  | string    | 主机名 |
| `os`        | string    | 操作系统（如 windows、linux） |
| `mac`       | string    | MAC 地址（用于 WOL） |
| `online`    | boolean   | 当前在线状态 |
| `last_seen` | timestamp | 最后活动时间（RFC3339） |

### Job（任务）

| 字段         | 类型      | 说明 |
|--------------|-----------|------|
| `id`         | uuid.UUID | 任务标识 |
| `files`      | []string  | 要安装的文件 ID 列表 |
| `created_at` | timestamp | 创建时间 |

### Job Result（任务结果）

| 字段              | 类型      | 说明 |
|-------------------|-----------|------|
| `id`              | uuid.UUID | 结果记录 ID |
| `job_id`          | uuid.UUID | 关联任务 |
| `client_id`       | uuid.UUID | 目标客户端 |
| `status`          | string    | `pending`、`running`、`success`、`failed` |
| `output`          | string    | 安装输出（stdout/stderr） |
| `error`           | string    | 失败时的错误消息 |
| `started_at`      | timestamp | 命令发送时间 |
| `finished_at`     | timestamp | 收到结果时间 |
| `snapshot_before` | JSON      | 安装前系统状态 |
| `snapshot_after`  | JSON      | 安装后系统状态 |

---

## 版本

当前 API 版本：**v1**。所有端点前缀为 `/api/v1/`。
