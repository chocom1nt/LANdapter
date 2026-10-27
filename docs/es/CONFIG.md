# Configuración de LANdapter

**Idiomas:** [English](../en/CONFIG.md) · [Русский](../ru/CONFIG.md) · [中文](../zh/CONFIG.md) · [日本語](../ja/CONFIG.md) · [Español](CONFIG.md)

LANdapter carga archivos YAML desde `configs/` relativos al directorio de trabajo. Las variables de entorno admitidas sustituyen los valores del archivo. La configuración se lee al iniciar el proceso; reinicia el servicio después de cambiarla.

## Master: `configs/master.yaml`

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: ""
  dbname: "landapter"
  sslmode: "disable"
```

| Clave | Descripción | Valor habitual |
|---|---|---|
| `host` | Interfaces de escucha de Master; `0.0.0.0` escucha en todas las interfaces. | `0.0.0.0` |
| `http_port` | Puerto REST para la interfaz y clientes API. | `8080` |
| `ws_port` | Puerto WebSocket de agentes. | `8081` |
| `db.host`, `db.port` | Host y puerto PostgreSQL. | `localhost`, `5432` |
| `db.user`, `db.password` | Cuenta de PostgreSQL. Es preferible inyectar el password como `DB_PASSWORD`. | `landapter`, vacío en el ejemplo |
| `db.dbname` | Nombre de la base. | `landapter` |
| `db.sslmode` | Modo TLS de PostgreSQL (`disable`, `require`, `verify-ca`, `verify-full`). | `disable` solo para desarrollo |

Variables de entorno admitidas en Master: `MASTER_HOST`, `HTTP_PORT`, `WS_PORT`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` y `DB_SSLMODE`. `LANDAPTER_ADMIN_TOKEN` se configura por separado y es obligatorio para iniciar el servicio: genera un valor aleatorio de al menos 32 caracteres, distinto del password de base de datos. En Compose, el archivo `.env` proporciona las variables que se declaran en `docker-compose.yml`; para ejecutar Go fuera de Compose, exporta las variables al entorno del proceso.

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

| Clave | Descripción |
|---|---|
| `master_host` | IP o nombre DNS de Master, accesible desde ese equipo. |
| `master_port` | Puerto WebSocket; debe coincidir con `ws_port` o con el puerto público del proxy TLS. |
| `master_http_port` | Puerto HTTP/HTTPS usado para descargar archivos. Puede ser igual a `master_port` detrás de un proxy TLS. |
| `master_scheme` | `ws` en una red de confianza sin TLS o `wss` detrás de un proxy TLS. Para acceso remoto, usa TLS. |
| `uuid` | ID emitido por Master. Si se configura, tiene prioridad sobre el ID guardado en `uuid_file`. |
| `token` | Secreto individual del agente; obligatorio y no recuperable desde la base de datos en texto plano. Emite otro si se pierde. |
| `uuid_file` | Ruta del UUID persistente cuando no se proporciona `uuid`. El directorio padre se crea si hace falta. |
| `installer_args` | Argumentos de línea de comandos por extensión para formatos que admiten personalización; el valor `.exe` afecta a instalaciones silenciosas. |

No reutilices un token de agente en varios equipos. Si cambias o vuelves a emitir el token para un UUID, el token anterior queda invalidado. Guarda `configs/agent.yaml` con permisos restrictivos y no lo incluyas en Git, tickets ni capturas de pantalla.

Variables de entorno admitidas en Agent: `MASTER_HOST`, `MASTER_PORT`, `MASTER_HTTP_PORT`, `MASTER_SCHEME`, `AGENT_UUID`, `AGENT_TOKEN` y `UUID_FILE`. Los mapas complejos como `installer_args` deben configurarse en YAML.

## Direcciones y proxy TLS

- Acceso local: la interfaz Vite usa `http://localhost:3000`, la API `http://localhost:8080` y el Agent se conecta a `ws://<master>:8081/ws`.
- En producción, publica HTTPS en un reverse proxy y enruta `/api/` a HTTP interno `8080` y `/ws` a WebSocket interno `8081`.
- Detrás del proxy, normalmente configura `master_scheme: "wss"`, y tanto `master_port` como `master_http_port` con el puerto HTTPS público, por ejemplo `443`.
- No expongas los puertos internos ni la base de datos a Internet. Configura `db.sslmode` con verificación adecuada si PostgreSQL cruza una red no confiable.

## Validación y operación

Master y Agent cargan la configuración al arrancar; reinícialos tras editar el YAML o el entorno. Si hay errores, comprueba la sintaxis YAML, que los puertos sean enteros, que el archivo exista desde el directorio de trabajo y que los nombres de las variables estén escritos exactamente. Consulta [Instalación](INSTALL.md) para la instalación y [Despliegue](DEPLOY.md) para producción.

## Autonomía del Agent e instalación como servicio

Actualmente el Agent es un proceso en primer plano que se reconecta; el repositorio todavía no lo instala como servicio nativo de Windows ni como unidad systemd. Las métricas y el inventario se solicitan bajo demanda y las alertas proactivas no están implementadas. Cada Agent necesita su propio UUID/token emitido por un administrador. Consulta la [guía del ciclo de vida del Agent](AGENT-LIFECYCLE.md) para conocer el motivo de seguridad y el futuro diseño con token bootstrap.
