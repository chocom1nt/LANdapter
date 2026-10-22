# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.26.4+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/react-18.2-61DAFB?style=flat&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/platform-windows%20%7C%20linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/postgresql-15-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL">
</p>

**Idiomas:** [English](README.md) · [Русский](README-RU.md) · [中文](README-ZH.md) · [日本語](README-JA.md) · [Español](README-ES.md)

**LANdapter** es un sistema cliente-servidor para la instalación remota centralizada de controladores y software en una red local.

Consta de un **maestro** (servidor) y **agentes** (equipos cliente). El maestro gestiona los agentes conectados, distribuye archivos y supervisa las instalaciones. Los agentes reciben comandos por WebSocket, descargan archivos y ejecutan instalaciones en modo silencioso o interactivo.

El proyecto simplifica la administración de estaciones de trabajo en redes de oficina e industriales, donde hay que actualizar controladores o desplegar aplicaciones en decenas o cientos de máquinas.

---

## Características

- **Gestión centralizada** – todos los agentes se conectan a un único servidor maestro.
- **Instalación remota** – envío de archivos (EXE, MSI, INF, DEB, RUN, TAR) y ejecución con los parámetros adecuados.
- **Dos modos de instalación** – silencioso (sin UI) e interactivo (con interfaz).
- **Recopilación de estadísticas** – métricas del sistema (CPU, RAM, tiempo activo) y listas de dispositivos (PnP, lsusb, lspci).
- **Historial e informes** – cada trabajo se guarda en la base de datos con instantáneas del sistema antes y después.
- **Puntos de restauración** – creación automática de punto de restauración en Windows antes de instalar.
- **Interfaz web** – panel React con tema oscuro, biblioteca de archivos y asistente de instalación.
- **Configuración flexible** – archivos YAML y variables de entorno.
- **Multiplataforma** – Master y Agent admiten Windows y Linux. La ejecución de instalaciones en macOS todavía no está admitida.

---

## Inicio rápido

### Requisitos

- Go 1.26.4+
- PostgreSQL 15+
- Node.js 18+ (frontend)
- Make (opcional)

### Instalación y ejecución

Clonar el repositorio:

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
```

Dependencias Go:

```bash
go mod download
```

Dependencias del frontend:

```bash
cd web
npm ci
cd ..
```

Antes de aplicar las migraciones, crea un usuario y una base de datos dedicados en PostgreSQL como administrador (por ejemplo, desde `psql`):

```sql
CREATE USER landapter WITH PASSWORD 'your-database-password';
CREATE DATABASE landapter OWNER landapter;
```

Usa la misma contraseña en `DB_PASSWORD` abajo. Si el usuario o la base ya existen, no los vuelvas a crear: comprueba el propietario y los permisos.

Antes de aplicar migraciones o iniciar el maestro, configure la contraseña de PostgreSQL y un secreto administrativo independiente de al menos 32 caracteres:

```bash
export DB_PASSWORD='your-database-password'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
```

Aplicar migraciones:

```bash
make migrate-up
```

Iniciar el maestro:

```bash
go run cmd/master/main.go
```

Iniciar el frontend (en otra terminal):

```bash
cd web
npm run dev
```

Abre `http://localhost:3000`, entra en Settings, introduce `LANDAPTER_ADMIN_TOKEN`, guarda la conexión y emite un token individual para el agente. El token se muestra una sola vez: copia `agent_id` y `token` a `configs/agent.yaml` como `uuid` y `token`.

Después de guardar esas credenciales, inicia el agente en otra terminal:

```bash
go run cmd/agent/main.go
```

Instrucciones detalladas en [docs/INSTALL-ES.md](docs/INSTALL-ES.md).

---

## Compilar desde el código fuente

### Binarios Go

```bash
make build
```

O manualmente:

```bash
go build -o bin/master cmd/master/main.go
go build -o bin/agent cmd/agent/main.go
```

### Frontend

```bash
cd web
npm run build
```

Los estáticos quedan en `web/dist/`.

### Imágenes Docker

```bash
# Maestro
docker build -t landapter-master .

# Agente
docker build -f Dockerfile.agent -t landapter-agent .
```

`docker-compose.yml` inicia **solo PostgreSQL y Master**; no inicia el frontend ni agentes. Antes, copia `configs/master.yaml.example` a `configs/master.yaml`, configura `.env` y aplica las migraciones SQL. Consulta [docs/INSTALL-ES.md](docs/INSTALL-ES.md). Ejecuta el frontend por separado o publica `web/dist/` con un servidor web.

---

## Estructura del proyecto

```text
LANdapter/
├── cmd/                  # Puntos de entrada
│   ├── master/
│   └── agent/
├── internal/             # Paquetes internos
│   ├── common/           # Tipos comunes, logging, config
│   ├── master/           # Lógica del maestro (HTTP, WebSocket)
│   └── agent/            # Lógica del agente (conexión, instalación)
├── storage/              # Capa de datos (interfaz + PostgreSQL)
├── migrations/           # Migraciones SQL
├── web/                  # Frontend React
├── docs/                 # Documentación
├── configs/              # Ejemplos de configuración
├── uploads/              # Subidas (se crea automáticamente)
├── docker-compose.yml
├── makefile
├── go.mod
└── README.md
```

---

## Documentación

El índice completo está en [docs/README.md](docs/README.md). Empieza por la guía correspondiente a tu tarea:

- [Instalación](docs/INSTALL-ES.md): requisitos, base de datos, migraciones, Master/UI y registro de Agents.
- [Configuración](docs/CONFIG-ES.md): parámetros de Master y Agent y variables de entorno.
- [Arquitectura](docs/ARCHITECTURE-ES.md): componentes, credenciales, WebSocket y límites actuales.
- [Despliegue en producción](docs/DEPLOY-ES.md): proxy TLS, endurecimiento, copias de seguridad y actualizaciones.
- [API](docs/README.API-ES.md): rutas REST y protocolo WebSocket.
- [FAQ](docs/FAQ-ES.md) y [contribución](docs/CONTRIBUTING-ES.md).

---

## Pruebas

Tests unitarios:

```bash
make test
```

Tests de integración (requieren BD real):

```bash
make test-integration
```

Informe de cobertura:

```bash
make test-cover
```

---

## Hoja de ruta

- Políticas de grupo (selección de clientes por grupo)
- CLI del maestro (gestión sin interfaz web)
- Integración con Active Directory / LDAP
- Formatos adicionales (AppImage, Flatpak)
- Trabajos programados
- Mejor parsing de controladores desde sitios web

---

## Licencia

Licencia MIT. Ver [LICENSE](LICENSE).

---

¿Preguntas o sugerencias? Abra un [Issue](https://github.com/chocom1nt/LANdapter/issues) o envíe un Pull Request.
