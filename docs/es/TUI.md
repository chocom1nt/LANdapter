# Interfaz de terminal (lndp)

`lndp` es un cliente interactivo de terminal para administrar clientes y paquetes cargados mediante la API del Master. Las acciones de instalación y eliminación afectan a sistemas y archivos reales.

## Pestañas

- **Clients** — listar/buscar clientes, seleccionar destinos y actualizar el estado de conexión.
- **Devices** — consultar el inventario y las pistas de diagnóstico del Agent seleccionado cuando está conectado.
- **Monitor** — ver la última instantánea de CPU/RAM/uptime. Las métricas se solicitan bajo demanda; todavía no se guardan como serie histórica.
- **Files / Install** — gestor de dos paneles para archivos locales y paquetes del Master: marcar, subir, enviar instalaciones y eliminar paquetes cargados tras confirmar.

## Atajos de teclado

| Tecla | Acción |
|---|---|
| `1`–`4` / `Tab` | Cambiar de pestaña |
| `↑` / `↓` o `j` / `k` | Moverse por las listas |
| `Space` | Marcar cliente o paquete |
| `/` | Buscar clientes |
| `U` | Subir archivos locales marcados |
| `I` | Enviar los paquetes seleccionados a los clientes elegidos |
| `M` | Cambiar el modo de instalación cuando sea compatible |
| `X` y después `Y` | Confirmar la eliminación de un paquete del Master |
| `R` | Actualizar la vista actual |
| `?` / `F1` | Abrir la ayuda de teclado |
| `Q` | Salir |

## Notas

- Comprueba los clientes y paquetes seleccionados antes de pulsar `I`; se enviarán comandos de instalación reales.
- MSI admite modos silencioso, UI básica y UI completa. El comportamiento de EXE y de instaladores Linux depende del proveedor y de sus argumentos.
- Un timeout al consultar dispositivos puede indicar que el Agent está desconectado, usa una versión antigua o tarda en consultar Windows PnP/WMI. Un timeout no equivale a un inventario vacío.
- El ancho y Unicode dependen del emulador de terminal. Usa un terminal moderno compatible con UTF-8 y amplía la ventana si se recortan columnas.
