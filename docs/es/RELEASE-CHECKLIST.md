# Lista de comprobación para el lanzamiento

## Compilación y dependencias

- [ ] CI pasa con la versión de Go requerida por `go.mod`.
- [ ] Pasan las pruebas frontend, la compilación de producción y la auditoría de dependencias.
- [ ] Se revisan los avisos de seguridad de dependencias y se corrigen o se documenta la decisión de riesgo.
- [ ] Las imágenes Docker se construyen desde un checkout limpio.

## Funcionalidad

- [ ] Se validan la instalación nueva y las migraciones de actualización con bases desechables.
- [ ] Los Agents de Windows/Linux se autentican con credenciales únicas y se reconectan tras reiniciar Master.
- [ ] Se verifican inventario, métricas, descarga de paquetes, resultados de instalación y errores con Agents reales.
- [ ] Se prueban políticas y ejecuciones programadas repetidas en un cliente aislado.
- [ ] La navegación TUI y las confirmaciones destructivas funcionan en terminales compatibles.

## Seguridad y operaciones

- [ ] Los secretos de administración y de Agent no están en Git ni en los registros.
- [ ] Se documentan TLS/WSS y reglas de firewall para redes no aisladas.
- [ ] Se prueban copia de seguridad y restauración.
- [ ] Los comandos destructivos requieren confirmación explícita.
- [ ] Se documentan las limitaciones y los pasos de actualización.

## Metadatos de lanzamiento

- [ ] Las versiones y el changelog están sincronizados.
- [ ] Las cinco versiones lingüísticas incluyen instrucciones y limitaciones actuales.
- [ ] Solo se crea la etiqueta `vX.Y.Z` cuando pasan todas las comprobaciones.
- [ ] Se publican sumas de verificación y notas de la versión.
