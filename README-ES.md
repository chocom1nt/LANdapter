# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
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
- **Multiplataforma** – maestro y agentes en Windows y Linux (el agente también admite macOS con funcionalidad limitada).

---

## Inicio rápido

### Requisitos

- Go 1.21+
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
npm install
cd ..
```

Aplicar migraciones:

```bash
make migrate-up
# o manualmente:
psql -h localhost -U postgres -d landapter -f migrations/001_init.up.sql
psql -h localhost -U postgres -d landapter -f migrations/002_add_mac_up.sql
psql -h localhost -U postgres -d landapter -f migrations/003_add_devices_up.sql
psql -h localhost -U postgres -d landapter -f migrations/004_add_snapshots.up.sql
```

Iniciar el maestro:

```bash
go run cmd/master/main.go
```

Iniciar un agente (otra terminal):

```bash
go run cmd/agent/main.go
```

Iniciar el frontend (tercera terminal):

```bash
cd web
npm run dev
```

Abrir `http://localhost:3000` en el navegador.

Instrucciones detalladas en [docs/INSTALL.md](docs/INSTALL.md).

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
docker build -f Dockerfile.master -t landapter-master .

# Agente
docker build -f Dockerfile.agent -t landapter-agent .
```

O el stack completo con `docker-compose` (incluye PostgreSQL):

```bash
docker-compose up -d
```

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
├── Makefile
├── go.mod
└── README.md
```

---

## Documentación

Documentación completa en [docs/](docs/):

- [Referencia API](docs/README.API-ES.md) – endpoints REST y WebSocket ([English](docs/README.API.md) · [Русский](docs/README.API-RU.md) · [中文](docs/README.API-ZH.md) · [日本語](docs/README.API-JA.md)).
- [Guía de instalación](docs/INSTALL.md) – despliegue detallado.
- [Configuración](docs/CONFIG.md) – parámetros de `master.yaml` y `agent.yaml`.
- [Arquitectura](docs/ARCHITECTURE.md) – diseño, ciclo de vida de trabajos, WebSocket.
- [Despliegue en producción](docs/DEPLOY.md) – servicios, proxy, monitorización.
- [FAQ](docs/FAQ.md) – preguntas frecuentes.

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
