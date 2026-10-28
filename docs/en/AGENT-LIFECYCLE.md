# Agent lifecycle, enrollment, and service operation

This guide describes what the Agent does today, why credentials are issued explicitly, and the recommended direction for unattended workstation deployment.

## Current behavior

- The Agent is a foreground process when started with `make run-agent` or `go run ./cmd/agent`. The repository does not currently provide a built-in Windows Service installer or a systemd unit.
- The Agent connects to Master over WebSocket and reconnects after connection loss. Master authenticates the handshake against a per-agent token hash stored in PostgreSQL.
- The Agent reports its identity and basic host information at connection time. Device inventory and CPU/RAM/uptime are requested by the UI/TUI on demand; continuous telemetry streaming and proactive alerting are not implemented yet.
- Each Agent requires its own `uuid` and `token` in `configs/agent.yaml` or the `AGENT_UUID` / `AGENT_TOKEN` environment variables. Environment variables override YAML values.

## Why not issue a token to every client automatically?

A Master that grants a permanent credential to any machine that connects would allow an unauthorized host on the network to enroll and potentially receive remote installation commands. The current enrollment endpoint is protected by the administrator token; the Agent cannot safely call it without already possessing an authorized enrollment secret. Per-agent credentials also allow individual revocation and rotation.

## Recommended unattended enrollment design

1. An administrator creates a short-lived, single-use enrollment code or deployment package scoped to an environment or group.
2. The Agent installer receives that enrollment secret through a protected channel (MDM, configuration management, a secured share, or a deployment system), not a public URL or source code.
3. The Agent exchanges the enrollment secret for a unique per-agent credential. Master stores only the credential hash; the raw token is returned once and saved locally with restrictive permissions.
4. Depending on policy, the new Agent is either approved automatically under tightly controlled rules or appears as pending until an administrator approves it. Enrollment codes expire, have a use limit, and can be revoked.
5. The Agent is installed as a Windows Service or a systemd service with automatic startup, restart-on-failure, a dedicated low-privilege account where possible, and centralized logs.

## Running the current Agent in the background

Until a native service installer is included, use your organization’s service manager or deployment tooling. On Linux, a systemd unit should run the built binary with a fixed working directory, read-only access to configuration where possible, restart-on-failure, and a dedicated account. On Windows, use an approved service wrapper or package it as a native service. Protect `configs/agent.yaml` and never put production tokens into Git, public installer arguments, screenshots, or logs. Test interactive installers separately: a background service session may not have access to the logged-in user desktop.

## Current limitations and roadmap

- Automatic enrollment without a manually provisioned per-agent token is not implemented.
- Native Windows Service and systemd installation are not included in the current repository.
- Continuous heartbeat telemetry, event forwarding, alert rules, and administrator notifications are not implemented; the current UI requests metrics on demand.
- A future enrollment flow should use expiring single-use bootstrap credentials, per-agent identity, approval controls, rate limiting, TLS/WSS, and audit logs.
