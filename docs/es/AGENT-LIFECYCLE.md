# Ciclo de vida del agente, registro y ejecución como servicio

Esta guía explica el comportamiento actual del Agent, por qué las credenciales se emiten explícitamente y cómo debería evolucionar el despliegue desatendido de estaciones de trabajo.

## Comportamiento actual

- Al ejecutar `make run-agent` o `go run ./cmd/agent`, el Agent funciona como proceso en primer plano. El repositorio aún no incluye un instalador integrado de servicio de Windows ni una unidad systemd.
- El Agent se conecta al Master mediante WebSocket y vuelve a conectarse tras una desconexión. El Master valida el handshake mediante el hash del token del agente guardado en PostgreSQL.
- Al conectarse, el Agent comunica su identidad y datos básicos del equipo. La lista de dispositivos y las métricas CPU/RAM/uptime se solicitan bajo demanda desde la UI/TUI; todavía no hay telemetría continua ni alertas proactivas.
- Cada Agent necesita su propio `uuid` y `token` en `configs/agent.yaml` o en las variables `AGENT_UUID` / `AGENT_TOKEN`. Las variables de entorno prevalecen sobre YAML.

## ¿Por qué no emitir un token automáticamente a cualquier cliente?

Si el Master concediera una credencial permanente a cualquier equipo que se conecte, un host no autorizado de la red podría registrarse y recibir comandos de instalación remota. El endpoint actual de registro está protegido por el token de administrador; el Agent no puede llamarlo de forma segura sin un secreto de inscripción autorizado. Las credenciales individuales también permiten revocar o rotar un único agente.

## Diseño recomendado para el registro desatendido

1. Un administrador crea un código de registro de un solo uso y corta duración, o un paquete de despliegue limitado a un entorno o grupo.
2. El instalador recibe ese secreto por un canal protegido (MDM, gestión de configuración, recurso compartido seguro o sistema de despliegue), nunca desde una URL pública o el código fuente.
3. El Agent intercambia el secreto de inscripción por una credencial única. El Master guarda solo el hash del token; el token en claro se devuelve una vez y se almacena localmente con permisos restrictivos.
4. Según la política, el nuevo Agent se aprueba automáticamente bajo reglas estrictas o queda pendiente de aprobación administrativa. Los códigos deben caducar, limitar usos y poder revocarse.
5. El Agent se instala como servicio de Windows o systemd, con inicio automático, reinicio tras fallos, cuenta dedicada de mínimos privilegios cuando sea posible y registros centralizados.

## Ejecutar el Agent actual en segundo plano

Hasta que exista un instalador de servicio nativo, usa el gestor de servicios o las herramientas de despliegue aprobadas por tu organización. En Linux, la unidad systemd debería fijar el directorio de trabajo, restringir el acceso a la configuración, usar `restart-on-failure` y una cuenta dedicada. En Windows, utiliza un envoltorio de servicio aprobado o empaqueta el Agent como servicio nativo. Protege `configs/agent.yaml`; nunca incluyas tokens de producción en Git, argumentos públicos del instalador, capturas de pantalla o registros. Prueba los instaladores interactivos por separado: un servicio en segundo plano quizá no tenga acceso al escritorio del usuario conectado.

## Limitaciones actuales y hoja de ruta

- El registro automático sin proporcionar previamente un token del Agent no está implementado.
- El repositorio no incluye instalación nativa como servicio de Windows ni unidad systemd.
- La telemetría continua, el reenvío de eventos, las reglas de alerta y las notificaciones al administrador no están implementados; la UI solicita métricas bajo demanda.
- El futuro flujo de registro debería usar secretos bootstrap de un solo uso y caducidad, identidad individual, controles de aprobación, límites de frecuencia, TLS/WSS y auditoría.
