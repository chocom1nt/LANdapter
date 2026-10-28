# LANdapter

<p align="center">
  <img src="../assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.26.4+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/react-18.2-61DAFB?style=flat&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/platform-windows%20%7C%20linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/postgresql-15-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL">
</p>

**Idiomas:** [English](../../README.md) · [Русский](../ru/README.md) · [中文](../zh/README.md) · [日本語](../ja/README.md) · [Español](README.md)

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

- Docker Engine con el complemento Docker Compose
- Node.js 18+ para la interfaz web
- Go 1.26.4+ solo si quieres compilar o ejecutar binarios Go fuera de Docker

### Preparar e iniciar LANdapter

Desde la raíz del repositorio:

```bash
make quickstart
```

`make quickstart` crea o repara `.env`, copia configuraciones de ejemplo si faltan, instala las dependencias frontend e inicia PostgreSQL y Master. En un **volumen Docker nuevo**, PostgreSQL inicializa el esquema con las migraciones SQL. Los scripts de inicialización no se vuelven a ejecutar en bases existentes.

Instala e inicia la interfaz web en terminales separados:

```bash
make run-frontend
```

Abre `http://localhost:3000`, ejecuta `make copy-token` y pega una vez el token de administrador en **Settings**. El navegador lo conserva durante la sesión actual. `lndp` lee el token desde `.env` automáticamente.

Para compilar y ejecutar la TUI:

```bash
make build-tui
make run-tui
```

En Settings, emite credenciales distintas para cada Agent y guárdalas en `configs/agent.yaml`; después ejecuta `make run-agent`.

Para instalaciones existentes o PostgreSQL configurado manualmente, consulta [la guía de instalación](INSTALL.md). No ejecutes `docker compose down -v` salvo que quieras borrar el volumen de la base de datos.

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

`make quickstart` prepara el entorno y arranca PostgreSQL + Master; en un volumen nuevo PostgreSQL inicializa el esquema. El frontend se inicia por separado con `make run-frontend`. Para instalaciones existentes, consulta [la guía de instalación](INSTALL.md).

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

El índice completo está en [docs/README.md](../README.md). Empieza por la guía correspondiente a tu tarea:

- [Instalación](INSTALL.md): requisitos, base de datos, migraciones, Master/UI y registro de Agents.
- [Configuración](CONFIG.md): parámetros de Master y Agent y variables de entorno.
- [Arquitectura](ARCHITECTURE.md): componentes, credenciales, WebSocket y límites actuales.
- [Despliegue en producción](DEPLOY.md): proxy TLS, endurecimiento, copias de seguridad y actualizaciones.
- [API](README.API.md): rutas REST y protocolo WebSocket.
- [FAQ](FAQ.md) y [contribución](CONTRIBUTING.md).

---

## Más documentación

- [Ciclo de vida del Agent y despliegue desatendido](AGENT-LIFECYCLE.md): registro, ejecución como servicio y diseño seguro de autorregistro.
- [TUI](TUI.md): clientes, dispositivos, monitorización y gestor de paquetes.
- [Inventario, políticas y planificador](DEVICES-POLICIES-SCHEDULES.md): funciones y limitaciones actuales.

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

Licencia MIT. Ver [LICENSE](../../LICENSE).

---

¿Preguntas o sugerencias? Abra un [Issue](https://github.com/chocom1nt/LANdapter/issues) o envíe un Pull Request.
