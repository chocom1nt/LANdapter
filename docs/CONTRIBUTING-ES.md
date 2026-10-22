# Guía de contribución

**Idiomas:** [English](CONTRIBUTING.md) · [Русский](CONTRIBUTING-RU.md) · [中文](CONTRIBUTING-ZH.md) · [日本語](CONTRIBUTING-JA.md) · [Español](CONTRIBUTING-ES.md)

Gracias por contribuir a LANdapter. Se aceptan informes de errores, correcciones, mejoras de documentación y nuevas funciones.

## Antes de abrir un issue

Busca problemas duplicados y confirma que puedes reproducirlo con una versión actual. Incluye versiones de Go, PostgreSQL, Node.js y sistema operativo; pasos para reproducir; comportamiento esperado y observado; logs relevantes y capturas de la interfaz, si ayudan. **Redacta secretos, tokens, direcciones privadas y datos de usuarios** antes de adjuntar logs.

## Flujo de desarrollo

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
git checkout -b fix/short-description
go mod download
cd web && npm ci && cd ..
```

Ramas sugeridas: `fix/...`, `feature/...`, `docs/...`. Mantén los cambios enfocados y explica la razón de la solución.

## Pruebas y formato

```bash
gofmt -w ./cmd ./internal ./storage
go test ./...
make build
cd web && npm run build
```

Usa `golangci-lint run ./...` si tienes golangci-lint instalado. La ejecución de pruebas depende de tener la versión de Go especificada por `go.mod` y de poder descargar las dependencias. No afirmes que una compilación o una prueba pasó si no la ejecutaste. El `package.json` no define actualmente un script `lint`; confirma los scripts existentes antes de usarlo.

## Convenciones

- Go: errores explícitos, formato `gofmt`, evitar logs de credenciales y documentar APIs públicas.
- React: componentes funcionales, nombres coherentes y manejo de errores en llamadas API.
- SQL: añade migraciones `NNN_description.up.sql` y el correspondiente `.down.sql`; no modifiques de forma destructiva una migración ya publicada sin explicar el cambio.
- API/protocolo: actualiza `README.API.md` y las variantes `README.API-RU.md`, `README.API-ZH.md`, `README.API-JA.md` y `README.API-ES.md`.
- Configuración: actualiza `CONFIG.md` y sus traducciones.
- No subas secretos, binarios de compilación ni `node_modules`.

## Pull request

Explica el problema, la solución, las pruebas realizadas, cualquier migración/configuración requerida y los riesgos de compatibilidad. En cambios de UI, incluye capturas sin datos reales. Para cambios de seguridad, describe el modelo de amenaza y añade pruebas de regresión cuando sea posible.
