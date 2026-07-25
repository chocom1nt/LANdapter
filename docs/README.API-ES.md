# Documentación de la API

**Idiomas:** [English](README.API.md) · [Русский](README.API-RU.md) · [中文](README.API-ZH.md) · [日本語](README.API-JA.md) · [Español](README.API-ES.md)

## Descripción general

LANdapter ofrece una API HTTP RESTful para gestionar clientes y distribuir instalaciones de software, además de un endpoint WebSocket para la comunicación en tiempo real con los agentes. Todos los endpoints HTTP se sirven en el puerto configurado en `master.yaml` (predeterminado `8080`). WebSocket usa un puerto separado (predeterminado `8081`).

> **Autenticación**: la versión actual no implementa autenticación. Está pensada para redes internas de confianza. No exponga la API a Internet sin medidas de seguridad adicionales.

---

## URL base

- API HTTP: `http://<master-host>:<http-port>/api/v1/`
- WebSocket: `ws://<master-host>:<ws-port>/ws`

---

## Endpoints HTTP de la API

Todos los endpoints devuelven JSON. Los errores incluyen el código HTTP correspondiente y un mensaje de texto plano.

---

### `GET /api/v1/clients`

Lista todos los clientes registrados.

**Parámetros de consulta**

| Nombre   | Tipo    | Descripción |
|----------|---------|-------------|
| `online` | boolean | Opcional. Filtra por estado en línea (`true` o `false`). Si se omite, se devuelven todos. |

**Respuesta** – `200 OK`

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

**Errores**

- `500 Internal Server Error` – fallo en la consulta a la base de datos.

---

### `POST /api/v1/install`

Crea un trabajo de instalación y envía comandos a los agentes en línea seleccionados.

**Cuerpo de la solicitud** – `application/json`

```json
{
  "file_ids": ["uuid1", "uuid2"],
  "client_ids": ["client-uuid1", "client-uuid2"],
  "mode": "quiet"
}
```

| Campo        | Tipo     | Descripción |
|--------------|----------|-------------|
| `file_ids`   | []string | Identificadores de archivos (subidos previamente con `/api/v1/upload`). |
| `client_ids` | []string | UUID de clientes donde instalar. |
| `mode`       | string   | Modo: `"quiet"` (silencioso) o `"interactive"` (con interfaz). Predeterminado `"quiet"`. |

**Respuesta** – `200 OK`

```json
{
  "job_id": "job-uuid"
}
```

El trabajo se crea y encola. Los clientes en línea reciben el comando de inmediato; los offline lo recibirán al reconectar.

**Errores**

- `400 Bad Request` – faltan campos obligatorios o JSON inválido.
- `500 Internal Server Error` – error de base de datos o almacenamiento.

---

### `POST /api/v1/upload`

Sube un archivo al servidor maestro. Se guarda en `uploads/` con un nombre único y metadatos (nombre original, tamaño, fecha de subida).

**Solicitud** – `multipart/form-data` con el campo `file`.

**Respuesta** – `200 OK`

```json
{
  "file_id": "generated-uuid",
  "name": "original_filename.exe"
}
```

**Errores**

- `400 Bad Request` – falta el archivo o supera el límite (100 MB).
- `500 Internal Server Error` – error al escribir en disco.

---

### `GET /api/v1/files`

Lista los archivos subidos con sus metadatos.

**Respuesta** – `200 OK`

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

**Errores**

- `500 Internal Server Error` – no se puede leer el directorio de subidas.

---

### `DELETE /api/v1/files/{id}`

Elimina un archivo del servidor (binario y metadatos).

**Respuesta** – `200 OK`

```json
{
  "status": "deleted"
}
```

**Errores**

- `400 Bad Request` – ID de archivo inválido.
- `404 Not Found` – el archivo no existe.
- `500 Internal Server Error` – fallo al eliminar.

---

### `GET /api/v1/clients/{id}/devices`

Solicita la lista de dispositivos de hardware de un cliente. El maestro reenvía la petición por WebSocket y espera hasta 5 segundos la respuesta del agente.

**Respuesta** – `200 OK`

El formato depende del SO del agente:
- **Windows**: array JSON de `Get-PnpDevice | ConvertTo-Json`.
- **Linux**: texto plano de `lsusb`, `lspci`, `lscpu`.

Ejemplo (Windows):

```json
[
  {
    "FriendlyName": "Intel(R) USB 3.1",
    "Class": "USB",
    "Status": "OK"
  }
]
```

**Errores**

- `400 Bad Request` – UUID de cliente inválido.
- `408 Request Timeout` – el agente no respondió en 5 segundos.
- `500 Internal Server Error` – fallo del comando WebSocket.

---

### `GET /api/v1/clients/{id}/stats`

Obtiene estadísticas del sistema del cliente (CPU, memoria, tiempo activo). La estructura depende de la plataforma.

**Respuesta** – `200 OK`

**Ejemplo Windows**:

```json
{
  "cpu_percent": 12.5,
  "mem_available_mb": 4096,
  "mem_total_mb": 8192,
  "uptime_seconds": 3600,
  "uptime_human": "1:00:00"
}
```

**Ejemplo Linux**:

```json
{
  "cpu_percent": 8.2,
  "mem_used_mb": 2048,
  "mem_total_mb": 4096,
  "uptime_seconds": 7200
}
```

**Errores** – iguales que `/devices`.

---

### `POST /api/v1/wol`

Envía paquetes Wake-on-LAN a clientes seleccionados (por MAC). El maestro busca las MAC por UUID y emite UDP al puerto 9.

**Cuerpo de la solicitud**

```json
{
  "client_ids": ["uuid1", "uuid2"]
}
```

**Respuesta** – `200 OK`

```json
{
  "status": "sent"
}
```

**Errores** – ninguno específico; se registran en el log sin afectar la respuesta HTTP.

---

### `POST /api/v1/parse-driver` (stub)

Endpoint provisional para descubrir controladores desde un sitio web. Actualmente devuelve una lista estática.

**Cuerpo de la solicitud**

```json
{
  "url": "https://example.com/drivers"
}
```

**Respuesta** – `200 OK`

```json
[
  {"name": "Driver1.inf", "url": "http://example.com/driver1.inf"},
  {"name": "Driver2.exe", "url": "http://example.com/driver2.exe"}
]
```

---

## Protocolo WebSocket

El maestro expone WebSocket en `/ws` para conexiones persistentes con agentes.

### Handshake

Tras conectar, el agente debe enviar un mensaje JSON de handshake:

```json
{
  "type": "handshake",
  "uuid": "agent-uuid",
  "hostname": "my-pc",
  "os": "windows",
  "mac": "00:11:22:33:44:55"
}
```

El maestro actualiza o inserta el cliente en la base de datos, lo marca en línea e inicia los bucles de lectura/escritura.

### Comandos entrantes (maestro → agente)

El maestro envía mensajes con esta estructura:

```json
{
  "type": "install",
  "job_id": "job-uuid",
  "payload": { ... }
}
```

- `type`: `"install"`, `"stats"` o `"devices"`.
- payload de `install`: `{"files": ["file-id1", ...], "mode": "quiet"}`.
- `stats` y `devices` tienen payload vacío.

### Respuestas salientes (agente → maestro)

El agente responde con un mensaje de resultado:

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

Para peticiones `stats` y `devices`, el agente envía:

```json
{
  "type": "stats",
  "job_id": "cmd-uuid",
  "data": { ... }
}
```

El campo `data` contiene las estadísticas o la lista de dispositivos solicitados.

### Gestión de la conexión

- El maestro envía ping periódicos (intervalo ~54 s) para mantener la conexión.
- Los agentes deben responder pong automáticamente (lo hace la biblioteca WebSocket).
- Si hay timeout de lectura (120 s), el maestro cierra la conexión.
- Los agentes reconectan con backoff exponencial.

---

## Modelos de datos

### Client (Cliente)

| Campo       | Tipo      | Descripción |
|-------------|-----------|-------------|
| `id`        | uuid.UUID | Identificador único |
| `hostname`  | string    | Nombre del equipo |
| `os`        | string    | Sistema operativo (windows, linux, etc.) |
| `mac`       | string    | Dirección MAC (para WOL) |
| `online`    | boolean   | Estado en línea |
| `last_seen` | timestamp | Última actividad (RFC3339) |

### Job (Trabajo)

| Campo        | Tipo      | Descripción |
|--------------|-----------|-------------|
| `id`         | uuid.UUID | ID del trabajo |
| `files`      | []string  | IDs de archivos a instalar |
| `created_at` | timestamp | Fecha de creación |

### Job Result (Resultado del trabajo)

| Campo             | Tipo      | Descripción |
|-------------------|-----------|-------------|
| `id`              | uuid.UUID | ID del registro de resultado |
| `job_id`          | uuid.UUID | Trabajo asociado |
| `client_id`       | uuid.UUID | Cliente destino |
| `status`          | string    | `pending`, `running`, `success`, `failed` |
| `output`          | string    | Salida de la instalación (stdout/stderr) |
| `error`           | string    | Mensaje de error si falló |
| `started_at`      | timestamp | Envío del comando |
| `finished_at`     | timestamp | Recepción del resultado |
| `snapshot_before` | JSON      | Estado del sistema antes |
| `snapshot_after`  | JSON      | Estado del sistema después |

---

## Versión

Versión actual de la API: **v1**. Todos los endpoints tienen el prefijo `/api/v1/`.
