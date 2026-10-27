# Instalación y puesta en marcha de LANdapter


## Inicio rápido recomendado (Docker)

Para una instalación local nueva, empieza en la raíz del repositorio:

```bash
make setup
make up
make install-frontend
make copy-token
make run-frontend
```

`make setup` crea o completa `.env` sin rotar secretos válidos y copia las configuraciones YAML de ejemplo solo si no existen archivos locales. En un **volumen Docker nuevo**, PostgreSQL ejecuta las migraciones durante la inicialización. Los volúmenes existentes no se limpian ni se migran con estos scripts. `make copy-token` copia el token de administrador para pegarlo una vez en Settings; `lndp` lee el token desde `.env` automáticamente.

Las secciones siguientes cubren PostgreSQL manual e instalaciones existentes. No uses `docker compose down -v` salvo que quieras borrar el volumen de la base de datos.

**Idiomas:** [English](../en/INSTALL.md) · [Русский](../ru/INSTALL.md) · [中文](../zh/INSTALL.md) · [日本語](../ja/INSTALL.md) · [Español](INSTALL.md)

Esta guía explica cómo ejecutar LANdapter en desarrollo o en una red de pruebas. Para un servicio público o empresarial, consulta [Despliegue](DEPLOY.md).

## 1. Requisitos

- Go **1.26.4 o posterior** para el backend.
- PostgreSQL **15 o posterior**.
- Node.js 18 o posterior y npm para la interfaz React.
- Git; `make` y el cliente `psql` son recomendables.
- En Windows, usa PowerShell para las variables de entorno y Git Bash o WSL2 para los comandos `make`.

## 2. Obtener el código

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
go mod download
cp configs/master.yaml.example configs/master.yaml
cp configs/agent.yaml.example configs/agent.yaml
```

Los comandos `cp` y `export` siguientes corresponden a Bash. En PowerShell, copia los archivos con `Copy-Item` y define las variables con `$env:NOMBRE = "valor"`.

## 3. Crear la base de datos

En PostgreSQL, crea un usuario y una base de datos dedicados:

```sql
CREATE USER landapter WITH PASSWORD 'replace-with-a-strong-db-password';
CREATE DATABASE landapter OWNER landapter;
```

Confirma que PostgreSQL escucha en el host y puerto configurados. El usuario necesita permisos para crear y modificar el esquema de LANdapter.

## 4. Configurar secretos y migraciones

Master lee `configs/master.yaml`. `DB_PASSWORD` del entorno sustituye la contraseña YAML. `LANDAPTER_ADMIN_TOKEN` es un secreto administrativo independiente; debe tener al menos 32 caracteres aleatorios. No uses la contraseña de PostgreSQL para este token.

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
make migrate-up
```

Guarda los dos secretos en un gestor de secretos o en un archivo de entorno protegido. No los subas al repositorio. `make migrate-up` aplica las migraciones numeradas, incluida `005_agent_credentials`. Antes de actualizar una instalación existente, haz una copia de seguridad de PostgreSQL. `make migrate-down` revierte las migraciones y puede eliminar datos o columnas: no lo uses como una operación rutinaria de recuperación.

## 5. Compilar y arrancar Master

```bash
go mod download
make build
./bin/master
```

También puedes ejecutar `go run cmd/master/main.go`. Master abre la API HTTP en el puerto 8080 y WebSocket en 8081 por defecto. El proceso debe heredar `DB_PASSWORD` y `LANDAPTER_ADMIN_TOKEN`. La aplicación no aplica las migraciones automáticamente: la base de datos debe prepararse antes.

## 6. Arrancar la interfaz web

En otra terminal:

```bash
cd web
npm ci
npm run dev
```

Abre `http://localhost:3000`. El servidor de desarrollo Vite reenvía las llamadas `/api` a `http://localhost:8080`. Esta interfaz de desarrollo no está pensada para producción. Para publicar la interfaz, ejecuta `npm run build` y sirve `web/dist/` con Nginx, Caddy u otro servidor web configurado para reenviar `/api/` a Master.

En Settings/Configuración, introduce el mismo `LANDAPTER_ADMIN_TOKEN` que usa el proceso Master y guarda la conexión.

## 7. Registrar y arrancar un Agent

1. En la interfaz, abre Settings y emite credenciales para un agente.
2. Copia la respuesta de emisión inmediatamente: el token secreto se muestra **una sola vez**.
3. En el equipo destino, edita `configs/agent.yaml` y asigna `agent_id` a `uuid` y el secreto a `token`. Ajusta `master_host`, `master_port` y `master_http_port` para que apunten al Master. En una red local confiable se usa `master_scheme: "ws"`; con un proxy TLS se usa `"wss"`.
4. Protege el archivo para que solo la cuenta del servicio pueda leerlo.
5. Arranca el agente:

```bash
./bin/agent
# o
go run cmd/agent/main.go
```

El agente debe poder llegar al puerto WebSocket y al endpoint HTTP de descarga de archivos. En Windows, las instalaciones de controladores pueden requerir ejecutar el servicio como administrador; en Linux, las instalaciones de paquetes pueden necesitar privilegios elevados.

## 8. Docker Compose

`docker-compose.yml` inicia PostgreSQL y Master, pero **no** inicia la interfaz web ni agentes.

```bash
cp .env.example .env
cp configs/master.yaml.example configs/master.yaml
# Edita .env: usa dos secretos aleatorios diferentes
# Asegúrate de que el puerto local 5432 esté libre
docker compose up -d postgres
```

Antes de iniciar Master, aplica las migraciones contra la base de datos del contenedor. Con la publicación local de `5432`, carga explícitamente las variables de `.env` y mapea `POSTGRES_PASSWORD` a `DB_PASSWORD` (el Makefile usa `DB_PASSWORD` para `psql`):

```bash
set -a
. ./.env
set +a
export DB_PASSWORD="$POSTGRES_PASSWORD"
make migrate-up
```

También puedes aplicar en orden los archivos de `migrations/*.up.sql` con `psql`. Después:

```bash
docker compose up -d master
docker compose logs -f master
```

Los puertos de Compose se enlazan a `127.0.0.1` por defecto. No los publiques directamente en una red no confiable. El frontend se ejecuta por separado y los agentes se instalan en sus equipos destino.

## 9. Verificación rápida

```bash
curl -H "Authorization: Bearer $LANDAPTER_ADMIN_TOKEN" \
  http://localhost:8080/api/v1/clients
```

Una base vacía normalmente devuelve `[]`. Los logs de Agent deben indicar que se conectó y espera comandos. Consulta [API](README.API.md) para el API.

## Solución de problemas

- **Error de PostgreSQL:** confirma que servicio, host, puerto, usuario, contraseña y base de datos coincidan; comprueba que aplicaste todas las migraciones.
- **Master no arranca:** confirma que existe `configs/master.yaml`, que `DB_PASSWORD` está en el entorno del proceso y que `LANDAPTER_ADMIN_TOKEN` tiene al menos 32 caracteres.
- **Agent no conecta:** revisa `master_host`, `master_port`, reglas de firewall y que el token pertenezca al UUID configurado.
- **La descarga falla:** el Agent necesita acceso al puerto HTTP `master_http_port`, además del WebSocket.
- **La interfaz devuelve 401:** el token guardado en Settings debe coincidir exactamente con `LANDAPTER_ADMIN_TOKEN` del proceso Master.
- **Frontend no compila en un `node_modules` copiado:** ejecuta `npm ci` en el sistema destino para instalar las dependencias correctas para esa plataforma.

Siguientes pasos: [Configuración](CONFIG.md), [Despliegue](DEPLOY.md) y [API](README.API.md).
