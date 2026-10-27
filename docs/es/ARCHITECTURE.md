# Arquitectura de LANdapter

**Idiomas:** [English](../en/ARCHITECTURE.md) · [Русский](../ru/ARCHITECTURE.md) · [中文](../zh/ARCHITECTURE.md) · [日本語](../ja/ARCHITECTURE.md) · [Español](ARCHITECTURE.md)

LANdapter separa el plano de control (Master y API), los equipos administrados (Agents), la interfaz web y el almacenamiento persistente PostgreSQL.

```text
Navegador ── HTTP/REST + Bearer admin ──> Master :8080 ──> PostgreSQL
Agent ────── WebSocket autenticado ─────> Master :8081
Agent ────── HTTPS/HTTP descarga con token ─> Master :8080
Navegador ── HTTPS ──> Proxy TLS ──> frontend estático + rutas /api y /ws
```

## Componentes

- **Master (`cmd/master`)**: carga YAML y variables de entorno, conecta PostgreSQL y arranca los servidores HTTP y WebSocket.
- **API HTTP (`internal/master/handlers.go`)**: gestiona clientes, trabajos de instalación, biblioteca de archivos, datos de dispositivos y estadísticas, Wake-on-LAN y emisión de credenciales para Agents. Las rutas administrativas requieren `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>`.
- **WebSocket (`internal/master/websocket.go`)**: autentica cada Agent mediante UUID y token individual durante el handshake; administra conexiones y mensajes de comando/resultado.
- **Agent (`cmd/agent`, `internal/agent`)**: conecta, vuelve a intentar con espera creciente, descarga los ficheros autorizados, ejecuta el instalador para el sistema operativo y devuelve resultados.
- **Storage (`storage`)**: abstracción y backend PostgreSQL. Las migraciones SQL versionadas están en `migrations/`.
- **Web (`web/`)**: aplicación React que usa la API; durante desarrollo Vite escucha en 3000 y reenvía `/api` al Master local.

## Flujo de credenciales

1. El operador establece `LANDAPTER_ADMIN_TOKEN` como secreto aleatorio de al menos 32 caracteres en el proceso Master.
2. Las operaciones de administración envían `Authorization: Bearer <token>`.
3. `POST /api/v1/agents/enroll` crea UUID y token de Agent, o rota el token de un UUID existente.
4. PostgreSQL guarda el hash SHA-256 del token de Agent; el token original se muestra una sola vez.
5. El Agent presenta UUID y token en el primer mensaje WebSocket. El servidor compara los hashes en tiempo constante.
6. Para descargar un instalador, Agent llama a `/api/v1/agent/files/{id}` con `Authorization: Bearer <agent-token>` y `X-Agent-ID: <uuid>`.

La autenticación no cifra por sí misma el tráfico. Usa TLS/WSS en redes no confiables y restringe las rutas de red.

## Protocolo y mensajes

El Agent inicia `/ws` con un mensaje `handshake` que incluye `uuid`, `token`, `hostname`, `os` y `mac`. Después del handshake, Master y Agent intercambian mensajes JSON sobre WebSocket. Entre los tipos de comandos están instalación, estadísticas y dispositivos; las respuestas informan resultado, salida, error y datos de snapshot cuando corresponde. La referencia exacta está en [API](README.API.md).

## Trabajos e instalaciones

`POST /api/v1/install` valida los IDs de archivo y de cliente, crea un trabajo y registra el resultado por cliente. A los Agents en línea se les envía el comando; los resultados se guardan en PostgreSQL. Las tareas para equipos fuera de línea pueden quedar pendientes, pero **no existe todavía un mecanismo de reentrega automática al reconectar**.

Formatos de instalación actuales:

- Windows: `.exe`, `.msi`, `.inf`.
- Linux: `.deb`, `.run`, `.tar`, `.tar.gz`, `.tar.bz2`, `.tgz`.
- macOS no está soportado para ejecutar instalaciones actualmente.

El Agent puede capturar estadísticas de CPU, memoria y tiempo de actividad, así como una lista de dispositivos cuando las herramientas del sistema estén disponibles. Las funciones concretas dependen del sistema y de los permisos de la cuenta del servicio.

## Archivos y datos

Los archivos subidos se guardan en `uploads/` bajo un identificador único, con un archivo JSON lateral para metadatos. Mantén `uploads/` persistente, limita quién puede acceder a él y haz backup coordinado con PostgreSQL. La base de datos contiene Agents/clientes, trabajos, resultados e información de credenciales (hashes, no secretos originales).

## Límites y seguridad

- Master y los Agents requieren red, configuración y credenciales correctas.
- La emisión de credenciales y toda API administrativa dependen de `LANDAPTER_ADMIN_TOKEN`.
- Usa un secreto único por Agent y rótalo si se filtra; al rotarlo se invalida la credencial anterior.
- Ejecutar instaladores remotos es una operación privilegiada: revisa los paquetes y limita permisos, firewall y quién puede administrar.
- No hay CLI de Master ni políticas de grupos/programación implementadas en esta versión; consulta el roadmap del README.
