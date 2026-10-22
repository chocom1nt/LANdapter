# Preguntas frecuentes de LANdapter

**Idiomas:** [English](FAQ.md) · [Русский](FAQ-RU.md) · [中文](FAQ-ZH.md) · [日本語](FAQ-JA.md) · [Español](FAQ-ES.md)

## General

### ¿Qué es LANdapter?

Una herramienta cliente-servidor para administrar Agents y desplegar controladores o aplicaciones en equipos Windows y Linux desde un Master central.

### ¿Qué sistemas operativos admite?

Master puede compilarse para Windows y Linux. Las rutas de instalación de Agent disponibles actualmente son Windows (`.exe`, `.msi`, `.inf`) y Linux (`.deb`, `.run`, archivos tar compatibles). La ejecución de instalación en macOS no está implementada.

### ¿Cómo se autentica?

Las rutas administrativas REST exigen `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>`. El proceso Master requiere una variable `LANDAPTER_ADMIN_TOKEN` de al menos 32 caracteres aleatorios. Cada Agent tiene credenciales individuales emitidas en Settings. No expongas la API directamente a una red no confiable; usa TLS/WSS y controles de red.

### ¿El frontend se sirve desde Master?

No. El frontend de desarrollo corre en Vite, normalmente en `http://localhost:3000`. Para producción, compila `web/dist/` y sirve esos archivos en un servidor web/proxy.

## Instalación y configuración

### ¿Cómo cambio los puertos?

Edita `http_port` y `ws_port` en `configs/master.yaml`. Actualiza `master_port` de Agent y `master_http_port` si hace falta. Abre solo los puertos necesarios en el firewall.

### ¿Puedo guardar la contraseña de DB en el YAML?

Es preferible definir `DB_PASSWORD` en el entorno del proceso Master. Para el secreto administrativo, usa `LANDAPTER_ADMIN_TOKEN`. Son dos secretos diferentes.

### ¿Por qué falla la conexión de Agent?

Confirma que Master escucha, que `master_host`, `master_port`, `master_http_port` y `master_scheme` sean correctos, que el token corresponda al UUID y que el firewall permita la conexión. Si usas `wss`, el proxy TLS debe estar configurado para WebSocket y para descargas HTTP(S).

### ¿Por qué el Agent no puede descargar un archivo aunque WebSocket conecte?

El Agent usa el puerto de `master_http_port` para descargar, aparte del puerto WebSocket. El endpoint de descarga exige el mismo token válido de Agent y el UUID en `X-Agent-ID`.

### ¿Puedo ejecutar Master sin PostgreSQL?

No en la implementación actual. PostgreSQL almacena agentes, trabajos y resultados.

## Uso y límites

### ¿Qué formatos se admiten?

Windows: `.exe`, `.msi`, `.inf`. Linux: `.deb`, `.run`, `.tar`, `.tar.gz`, `.tar.bz2`, `.tgz`. Comprueba los argumentos en la configuración del instalador y prueba cada paquete antes de distribuirlo.

### ¿Qué diferencia hay entre modo silencioso e interactivo?

El modo `quiet` usa argumentos silenciosos predeterminados cuando existen. El modo `interactive` solicita el modo de interfaz básico que admite el instalador. Los parámetros específicos del proveedor pueden variar.

### ¿Los trabajos pendientes se ejecutan cuando vuelve a conectarse un Agent?

No automáticamente en la versión actual. Aunque un resultado puede quedar en estado `pending`, la reentrega al reconectar todavía no está implementada.

### ¿Qué hace Wake-on-LAN?

Master envía un paquete mágico al MAC del equipo. El adaptador, BIOS/UEFI, configuración de SO y red deben admitir y permitir Wake-on-LAN; no funcionará necesariamente a través de subredes enrutadas.

## Solución de problemas

- **Master informa de autenticación sin configurar:** define `LANDAPTER_ADMIN_TOKEN` en el entorno de Master y reinicia.
- **HTTP 401:** el token administrativo enviado por el frontend/API no coincide con el token del proceso Master.
- **HTTP 503 al verificar Agent:** comprueba la conectividad de Master con PostgreSQL.
- **Agente aparece offline:** busca en sus logs, verifica la red y comprueba si su credencial fue rotada.
- **No hay archivos en la lista:** confirma que `uploads/` existe y es escribible para el servicio.
- **Instalación `.inf` falla:** ejecuta Agent con permisos elevados en Windows y revisa la salida del comando y los requisitos del controlador.
- **Frontend muestra error de red:** confirma el proxy `/api/`, el host/puerto del Master y la consola del navegador.

Para más detalles: [INSTALL-ES.md](INSTALL-ES.md), [CONFIG-ES.md](CONFIG-ES.md) y [README.API-ES.md](README.API-ES.md).
