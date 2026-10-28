# Despliegue de LANdapter en producción

**Idiomas:** [English](../en/DEPLOY.md) · [Русский](../ru/DEPLOY.md) · [中文](../zh/DEPLOY.md) · [日本語](../ja/DEPLOY.md) · [Español](DEPLOY.md)

Este documento es una lista práctica de preparación para una instalación persistente. LANdapter ejecuta instaladores remotos: un error de selección o de permisos puede modificar equipos. Valida primero en máquinas de prueba y conserva copias de seguridad.

## Topología recomendada

- Master en una VM o servidor Linux dedicado.
- PostgreSQL accesible solo desde Master y desde el operador que aplica migraciones.
- Interfaz web servida por Nginx/Caddy.
- Un proxy TLS expone HTTPS/WSS; los puertos internos 8080 y 8081 no se publican directamente.
- Agents se ejecutan con una cuenta de servicio y los privilegios mínimos necesarios.

## Secretos y base de datos

Genera dos secretos independientes:

```bash
openssl rand -hex 32
openssl rand -hex 32
```

Guarda uno como contraseña PostgreSQL y el otro como `LANDAPTER_ADMIN_TOKEN`. El Master no arranca si la variable administrativa falta o tiene menos de 32 caracteres. No guardes secretos en Git, logs, documentación ni variables del frontend. Para rotar el token administrativo, actualiza el entorno del Master y también la configuración guardada en la interfaz; reinicia Master. Protege los tokens de Agent: el secreto se muestra una sola vez y solo el hash SHA-256 se guarda en PostgreSQL.

## Base de datos y actualizaciones

1. Crea una cuenta PostgreSQL no administrativa para la aplicación y una base de datos dedicada.
2. Configura `db.sslmode` acorde a la topología. `disable` solo es adecuado en un entorno local controlado.
3. Haz backup antes de aplicar migraciones nuevas.
4. Ejecuta `make migrate-up` con `PG_HOST`, `PG_PORT`, `PG_USER`, `PG_DB` y `DB_PASSWORD` correctos. El Makefile ejecuta `psql` en los scripts numerados; no hay migración automática en el arranque de Master.
5. Verifica las tablas y conserva un registro de qué versión de código y migraciones se desplegó.

No uses `make migrate-down` como rollback automático en producción sin revisar los scripts `*.down.sql`: pueden eliminar columnas y datos.

## Ejecutar Master como servicio

Construye con la versión de Go indicada por el proyecto y despliega el binario, `configs/master.yaml`, la carpeta `uploads/` y un archivo de entorno protegido. Ejemplo de variables para un servicio systemd:

```dotenv
DB_PASSWORD=replace-with-database-secret
LANDAPTER_ADMIN_TOKEN=replace-with-independent-random-admin-secret
```

El servicio debe ejecutarse desde el directorio de la aplicación, porque las rutas de configuración y `uploads/` son relativas al directorio de trabajo. Configura `Restart=on-failure` o política equivalente, límites de acceso a archivos y un usuario de servicio sin privilegios. Guarda `uploads/` en almacenamiento persistente y monitoriza espacio disponible.

## Reverse proxy TLS

Sirve el frontend desde `web/dist/` y conserva la ruta SPA a `index.html`. Reenvía `/api/` a `http://127.0.0.1:8080` y la ruta exacta `/ws` a `http://127.0.0.1:8081` con HTTP/1.1 y encabezados `Upgrade` / `Connection`. La ruta WebSocket es `/ws` (sin barra final obligatoria). Si el proxy publica ambos servicios por HTTPS en el puerto 443, configura Agent con `master_scheme: "wss"`, `master_port: 443` y `master_http_port: 443`.

Ejemplo de fragmento Nginx (ajusta dominio y certificados):

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Real-IP $remote_addr;
}
location = /ws {
    proxy_pass http://127.0.0.1:8081;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

La configuración TLS completa depende de tu dominio y política de certificados. Verifica desde un Agent remoto que el WebSocket y la descarga de archivos funcionen. No actives el modo `wss` si el proxy no ofrece TLS.

## Firewall y exposición

- Permite HTTPS (normalmente TCP 443) para operadores y Agents.
- Limita 8080/8081 a loopback o a una red privada cuando el proxy corre en la misma máquina.
- Permite 5432 solo desde hosts de administración de base de datos autorizados.
- Usa VPN si administras equipos a través de redes públicas. No publiques PostgreSQL.
- La API administrativa usa `Authorization: Bearer ...`; no compartas el token con navegadores o usuarios no confiables. CORS no sustituye autenticación ni TLS.

## Agents y tareas

Emite credenciales individuales en Settings y almacena cada secreto de forma segura. En Windows, las instalaciones de controladores pueden requerir privilegios de administrador. En Linux, instalar paquetes o ejecutar scripts puede requerir privilegios elevados. Valida origen, firma, hash y parámetros del instalador antes de subir paquetes. Empieza con un único Agent y una instalación no crítica.

**Limitación actual:** las tareas dirigidas a Agents que están fuera de línea pueden persistir en estado pendiente, pero la entrega automática cuando el Agent se reconecta todavía no está implementada. No confíes en esto para despliegues programados o garantizados.

## Backups, monitorización y recuperación

- Programa backups consistentes de PostgreSQL y prueba periódicamente su restauración.
- Respalda `uploads/` y coordina la restauración con los IDs de archivo de la base de datos.
- Recoge logs stdout/stderr con systemd o el sistema de contenedores; redacta tokens y datos sensibles antes de enviarlos a un servicio externo.
- Alerta por reinicios repetidos, errores de base de datos, agotamiento del disco y Agents que dejan de informar.
- Para actualizar: backup → desplegar versión → aplicar migraciones → iniciar/reiniciar servicio → verificar API, UI y conexión de un Agent de prueba.

Para instalación inicial, consulta [Instalación](INSTALL.md); para los parámetros, [Configuración](CONFIG.md).
