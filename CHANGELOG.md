# Changelog

All notable changes to LANdapter are documented in this file. The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

This project snapshot has not been assigned an official release version here. Changes remain under **Unreleased** until they are matched to an actual release/tag in the canonical Git repository.

## [Unreleased]

### Added

#### Master, Agent, and API

- Added the Go-based Master service and Agent runtime for managing Windows and Linux workstations over a LAN.
- Added REST API endpoints for client enrollment, client records, package storage, deployment requests, installation results, device inventory, workstation snapshots, policies, and schedules.
- Added authenticated WebSocket communication for Master-to-Agent commands and Agent-to-Master results.
- Added automatic Agent reconnection when its Master connection is interrupted.
- Added separate credentials for each Agent. Enrollment generates a random per-Agent token, stores its SHA-256 hash in PostgreSQL, and returns the raw token only in the enrollment response.
- Added configuration examples for Master and Agent, environment-variable loading, and scripts for preparing local development configuration.

#### Software deployment

- Added central package upload and dispatch to selected online Agents.
- Added installer handling for Windows `.exe`, `.msi`, and `.inf` files and Linux `.deb`, `.run`, `.tar`, `.tar.gz`, `.tar.bz2`, and `.tgz` packages.
- Added quiet, basic-UI, and interactive modes where supported by the installer format.
- Added per-client installation results and command output for deployment troubleshooting.
- Added local package browsing and uploaded-package management in the terminal UI.
- Added streaming uploads from disk in `lndp` so the entire package does not need to be buffered in memory.

#### Device diagnostics and workstation status

- Added Windows Plug and Play inventory with available hardware identifiers, device instance identifiers, driver provider/version information, and device problem codes.
- Added Linux PCI/USB inventory through system information exposed by `sysfs`, including device identifiers and driver-binding diagnostics where available.
- Added diagnostic hints for devices that appear to be missing a driver; these are informational rather than an automatic driver update workflow.
- Added on-demand CPU, memory, and uptime snapshots for connected Agents.
- Added client operating-system distribution and client status summaries to the web panel.
- Added installation-related snapshots and job information for troubleshooting deployment outcomes.

#### Policies and scheduling

- Added PostgreSQL-backed package policies and interval schedules.
- Added policy and schedule management endpoints and web-panel controls.
- Added a Master-side scheduler and a manual action to run a policy immediately.
- Added migration `007_policies_schedules` and Windows-compatible migration helper targets.

#### Web interface

- Added a React-based administration dashboard for client inventory, status, package management, deployment, settings, device diagnostics, policies, and schedules.
- Added an English/Russian/Simplified Chinese language selector and translation dictionaries for the main web-panel interface.
- Added language-aware date and number formatting, translated status/error/action labels, and parameter interpolation in translation strings.
- Added dark/light theme styling, clearer focus states, improved selector contrast, and a refreshed navy/teal visual palette.
- Added a production frontend build using Vite.

#### Terminal UI (`lndp`)

- Added a colorized, four-tab terminal interface: **Clients**, **Devices**, **Monitor**, and **Files / Install**.
- Added online/offline state and last-seen information, client filtering, multi-selection, and keyboard navigation.
- Added device inventory browsing with selected-device diagnostics and paging.
- Added an `htop`-style CPU/RAM/uptime view that refreshes every five seconds while the monitor tab is active.
- Added a two-pane local/Master package browser, multi-file upload, multi-client installation dispatch, installer-mode selection, and confirmed package deletion.
- Added built-in keyboard help on `?` and `F1`, responsive narrow-terminal layouts, and Unicode display-width-aware clipping for CJK and other wide characters.
- Added handling for `NO_COLOR` and `TERM=dumb` terminal environments.

#### Development and operations

- Added `make quickstart` for first-run environment setup, frontend dependency installation, and Docker startup.
- Added `make copy-token` to copy the administrative token for initial web-panel setup.
- Added `make reset` and cross-platform factory-reset scripts with explicit confirmation prompts for development/test environments.
- Added `make clear-clients` and Unix shell/Windows PowerShell helpers. The command requires the operator to type `DELETE ALL CLIENTS` before deleting client records.
- Added `make seed-demo-agents` and `make clear-demo-agents` with deterministic development-only database fixtures for UI smoke testing.
- Added Docker Compose configuration for PostgreSQL and Master, plus Docker build definitions for Master/Agent images.
- Added migration apply/rollback helpers and setup scripts for Bash and PowerShell.
- Added a GitHub Actions CI workflow for Go tests, `go vet`, Go builds, frontend tests, and the frontend production build.

#### Tests and documentation

- Added frontend tests for API URL validation, client availability behavior, translation-key parity, interpolation placeholders, and UI translation-key references.
- Added Go tests for Master handlers, authentication/security helpers, WebSocket behavior, automation endpoints, Agent device diagnostics/installer argument selection, environment loading, and terminal UI behavior.
- Added documentation for installation, configuration, deployment, architecture, REST API, Agent lifecycle, terminal UI, device inventory, policies, scheduling, testing, and release readiness.
- Organized project documentation into English, Russian, Japanese, Simplified Chinese, and Spanish sections under `docs/`.
- Refreshed the root `README.md` with project badges, an architecture diagram, feature overview, quick start, documentation navigation, and a visible summary of current limitations.

### Changed

- Derive client online status from active Master WebSocket connections rather than trusting potentially stale database flags.
- Avoid requesting device inventory or live telemetry from clients that are offline or do not have an active Agent connection.
- Make the web UI and `lndp` distinguish timeouts/inactive connections from a genuinely empty device inventory.
- Make `.env` generation idempotent: fill missing or placeholder database credentials and the administrative token without rotating valid existing secrets.
- Centralize `.env` loading for Master and `lndp`, including support for the common working-directory and `bin/` executable layouts.
- Use PowerShell for Windows Makefile directory operations and wrappers for demo-data and migration tasks rather than relying on Unix-only command syntax.
- Use the current `cmd/lndp` path for terminal UI build/run targets and Docker references.
- Use `/qn`, `/qb`, and full UI behavior consistently for Windows MSI quiet, basic, and interactive modes. Non-quiet EXE launches no longer receive an empty argument; quiet EXE launches use the configured argument string or the default `/S`.
- Make terminal layouts, help, tables, selected rows, status colors, and long-path clipping more consistent across terminal widths.
- Organize translated guides under `docs/<language>/` and align configuration and lifecycle documentation with the current credential workflow.
- Clarify that Docker initialization scripts run only when PostgreSQL initializes a new empty data volume; existing databases require explicit migration steps.

### Fixed

- Corrected the administrative token environment-variable name to `LANDAPTER_ADMIN_TOKEN`.
- Require a sufficiently long administrative bearer token and compare credentials using a constant-time comparison.
- Authenticate Agent-only package downloads against the stored per-Agent token hash.
- Bound Wake-on-LAN request bodies and limit each request to 1–500 client IDs.
- Ensure Master binds its listeners before reporting a successful startup, so listener failures are surfaced at startup rather than logged later from a goroutine.
- Close an active Agent connection when its credential is rotated.
- Prevent offline/demo database fixtures from being treated as live WebSocket-connected Agents for inventory and installation operations.
- Fix the Windows `clear-clients` command to execute its SQL from a file rather than passing SQL through fragile PowerShell/Docker quoting.
- Fix the Linux `.deb` retry flow so successful dependency repair is followed by a retry of the requested package; dependency repair alone is not treated as proof that the package installed.
- Correct installer argument selection and add regression tests for Windows EXE/MSI mode handling.
- Improve API error messages and TUI feedback for HTTP 408/409 responses, disconnected Agents, and inventory timeouts.
- Correct stale or misleading README, TUI, testing, and release-checklist descriptions where they claimed features or checks that are not present.

### Security

- Store only the SHA-256 hash of each Agent token in PostgreSQL; disclose the generated raw token only in the enrollment response.
- Protect administrative API routes with the `LANDAPTER_ADMIN_TOKEN` bearer token and Agent package downloads with per-Agent credentials.
- Add response security headers including `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Referrer-Policy: no-referrer` to protected responses.
- Upgrade the locked PostCSS dependency to `8.5.31` to address the source-map path-disclosure issue documented in the security review.
- Document the risks of package deployment, the need to protect administrative and Agent credentials, and the use of TLS (`https`/`wss`) when operating beyond a trusted isolated network.

### Known limitations

These items are documented so the current implementation is not mistaken for a complete endpoint-management suite:

- Agent enrollment still requires an administrator-issued per-Agent credential. Short-lived bootstrap enrollment and unrestricted self-registration are not implemented.
- The Agent currently runs as a foreground process. The repository does not yet include a native Windows Service installer or a `systemd` unit/installer; an external service manager can be configured separately.
- CPU, memory, uptime, and device inventory are queried on demand. Continuous telemetry, historical time-series storage, proactive administrator alerts, and event forwarding are not implemented.
- Pending installation jobs are not automatically redelivered when an offline Agent reconnects.
- Policy targeting currently uses a case-insensitive OS substring filter; persistent dynamic groups, directory integration, tags, and richer targeting rules are not implemented.
- Schedules repeat policy/package runs at an interval. General-purpose cron expressions and configurable schedule time zones are not implemented; only run trusted, safe, idempotent installers on repeating schedules.
- Driver inventory is diagnostic only. Automatic driver catalog lookup, download, signature verification, selection, and installation are not implemented.
- The database-backed integration test suite is not yet present. CI test/build steps must pass on the target branch, and the manual release checklist must be completed before cutting a release.
- The Vite dependency noted in the release checklist still requires a security-reviewed upgrade and lockfile regeneration before the dependency review can be considered complete. Do not expose the development server to untrusted networks.

