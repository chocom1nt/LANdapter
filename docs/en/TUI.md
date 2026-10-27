# Master TUI (`lndp`)

`lndp` is an experimental terminal client for the authenticated Master API. It can browse client records, filter them, request device inventory, and inspect an online agent's current statistics. It is read-only: installs and policy editing remain in the web UI.

## Quick start

From the repository root:

```sh
make setup
make up
make build-tui
make run-tui
```

The TUI loads `LANDAPTER_ADMIN_TOKEN` from `.env` automatically. Override `LANDAPTER_API_URL` only when the Master API is not at `http://127.0.0.1:8080/api/v1`.

On Windows:

```powershell
.\bin\lndp.exe
```

## Controls

- `↑` / `↓` or `j` / `k`: select a client
- `/`: filter by hostname, OS, MAC, or ID
- `1` or `c`: clients list
- `2` or `d`: query device inventory and driver diagnostics for the selected client
- `3` or `s`: query the selected agent's current statistics
- `Tab`: cycle through clients, devices, and stats
- `Enter`: show or hide selected-client details on the client list
- `r`: refresh client list
- `Esc`: clear filter
- `q` or `Ctrl+C`: quit

The devices and stats tabs require the selected agent to be online and to run a compatible agent build. Windows inventory is based on Plug and Play problem information; Linux inventory examines PCI/USB devices and the sysfs driver binding. A flagged device is a diagnostic hint, not a recommendation to install an arbitrary driver. Confirm hardware IDs and vendor packages before deployment.

The TUI and inventory schema are experimental and may evolve.
