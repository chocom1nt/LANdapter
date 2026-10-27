# Pruebas y validación

Ejecuta las comprobaciones desde un checkout limpio y usa las mismas versiones de Go/Node que CI. Que se abra la página web no significa que una versión esté validada.

## Requisitos

- La versión de Go indicada por `go.mod` (actualmente 1.26.4 o posterior).
- Node.js 18+, npm y dependencias instaladas desde el lockfile.
- Docker Compose y una base PostgreSQL desechable para comprobaciones manuales de integración.

## Comprobaciones automáticas

```bash
go test ./...
make test
cd web && npm ci && npm test && npm run build
make test-cover
```

## Prueba manual rápida

1. Inicia PostgreSQL y Master; verifica migraciones, registros y estado.
2. Abre la UI, autentícate con el token de administrador, emite credenciales únicas y conecta un Agent de prueba real.
3. Comprueba online/offline, inventario, CPU/RAM/uptime, subida/descarga y un instalador de prueba inocuo.
4. Comprueba el envío de una política y su repetición programada en un cliente desechable.
5. Prueba pestañas, búsqueda, selección, terminal estrecho, subida y confirmaciones de instalación/eliminación en `lndp`.
6. Prueba copia/restauración y migraciones de actualización sobre una copia de la base, nunca sobre la única base de producción.

## Carencias conocidas

- Todavía no hay una suite completa de integración respaldada por base de datos.
- Valida el Agent de Windows en Windows y el de Linux en Linux; la compilación cruzada no valida el comportamiento de la plataforma.
- Los registros de demostración no demuestran que un Agent real pueda autenticarse o devolver inventario.
