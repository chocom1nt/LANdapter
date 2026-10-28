# Inventario de dispositivos, políticas y planificador

Esta guía describe las funciones actuales. El inventario sirve para diagnóstico; no es una recomendación automática de instalación de controladores.

## Inventario de dispositivos

- En Windows se usa la información disponible de Plug and Play/WMI: Hardware ID, fabricante, proveedor/versión del controlador y códigos de problema. En Linux se inspeccionan dispositivos PCI/USB y asociaciones de controladores mediante interfaces del sistema.
- Algunos campos pueden faltar por permisos, tipo de dispositivo, estado de WMI o herramientas no instaladas. Un resultado parcial no es lo mismo que una lista vacía obtenida correctamente.
- LANdapter ofrece pistas de diagnóstico, pero no busca catálogos de fabricantes, descarga controladores, verifica firmas ni los instala automáticamente.

## Políticas de grupo

- Las políticas asocian paquetes cargados con un filtro de subcadena del sistema operativo y pueden enviarse manualmente.
- Aún no se implementan grupos dinámicos persistentes, selección mediante AD/LDAP, etiquetas ni reglas avanzadas de destino.
- Valida cada política primero en un equipo de prueba: origen y firma del paquete, argumentos, reinicios e idempotencia.

## Planificador

- Las tareas programadas repiten la instalación de paquetes de una política con el intervalo configurado. No son todavía un motor de tareas de propósito general ni ofrecen semántica de calendario tipo cron.
- Las ejecuciones repetidas pueden reinstalar o reparar software y provocar reinicios. Usa instaladores confiables e idempotentes y supervisa los resultados. Desactiva el plan antes de sustituir paquetes o cambiar los destinos.

## Seguridad operativa

- Usa un cliente de prueba antes de desplegar a gran escala. Mantén privados los endpoints HTTP/WebSocket del Master o protégelos con TLS/WSS en redes no confiables.
- Conserva copias de seguridad de PostgreSQL y uploads. Los comandos de limpieza/restablecimiento de la base de datos son destructivos y no deben usarse como mantenimiento rutinario.
