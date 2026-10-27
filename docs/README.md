# LANdapter documentation

Choose one of the five documentation languages. English is the reference language; localized guides cover the same current capabilities and clearly call out features that are not implemented.

## Language guides

| Guide | English | Русский | 日本語 | 简体中文 | Español |
|---|---|---|---|---|---|
| Project overview | [README](../README.md) | [README](ru/README.md) | [README](ja/README.md) | [README](zh/README.md) | [README](es/README.md) |
| Installation | [INSTALL](en/INSTALL.md) | [INSTALL](ru/INSTALL.md) | [INSTALL](ja/INSTALL.md) | [INSTALL](zh/INSTALL.md) | [INSTALL](es/INSTALL.md) |
| Configuration | [CONFIG](en/CONFIG.md) | [CONFIG](ru/CONFIG.md) | [CONFIG](ja/CONFIG.md) | [CONFIG](zh/CONFIG.md) | [CONFIG](es/CONFIG.md) |
| Architecture | [ARCHITECTURE](en/ARCHITECTURE.md) | [ARCHITECTURE](ru/ARCHITECTURE.md) | [ARCHITECTURE](ja/ARCHITECTURE.md) | [ARCHITECTURE](zh/ARCHITECTURE.md) | [ARCHITECTURE](es/ARCHITECTURE.md) |
| Production deployment | [DEPLOY](en/DEPLOY.md) | [DEPLOY](ru/DEPLOY.md) | [DEPLOY](ja/DEPLOY.md) | [DEPLOY](zh/DEPLOY.md) | [DEPLOY](es/DEPLOY.md) |
| REST / WebSocket API | [API](en/README.API.md) | [API](ru/README.API.md) | [API](ja/README.API.md) | [API](zh/README.API.md) | [API](es/README.API.md) |
| FAQ | [FAQ](en/FAQ.md) | [FAQ](ru/FAQ.md) | [FAQ](ja/FAQ.md) | [FAQ](zh/FAQ.md) | [FAQ](es/FAQ.md) |
| Contributing | [CONTRIBUTING](en/CONTRIBUTING.md) | [CONTRIBUTING](ru/CONTRIBUTING.md) | [CONTRIBUTING](ja/CONTRIBUTING.md) | [CONTRIBUTING](zh/CONTRIBUTING.md) | [CONTRIBUTING](es/CONTRIBUTING.md) |
| Agent lifecycle / unattended operation | [Guide](en/AGENT-LIFECYCLE.md) | [Руководство](ru/AGENT-LIFECYCLE.md) | [ガイド](ja/AGENT-LIFECYCLE.md) | [指南](zh/AGENT-LIFECYCLE.md) | [Guía](es/AGENT-LIFECYCLE.md) |
| Terminal UI | [TUI](en/TUI.md) | [TUI](ru/TUI.md) | [TUI](ja/TUI.md) | [TUI](zh/TUI.md) | [TUI](es/TUI.md) |
| Devices / policies / scheduler | [Guide](en/DEVICES-POLICIES-SCHEDULES.md) | [Руководство](ru/DEVICES-POLICIES-SCHEDULES.md) | [ガイド](ja/DEVICES-POLICIES-SCHEDULES.md) | [指南](zh/DEVICES-POLICIES-SCHEDULES.md) | [Guía](es/DEVICES-POLICIES-SCHEDULES.md) |
| Testing | [TESTING](en/TESTING.md) | [TESTING](ru/TESTING.md) | [TESTING](ja/TESTING.md) | [TESTING](zh/TESTING.md) | [TESTING](es/TESTING.md) |
| Release readiness | [Checklist](en/RELEASE-CHECKLIST.md) | [Чек-лист](ru/RELEASE-CHECKLIST.md) | [チェックリスト](ja/RELEASE-CHECKLIST.md) | [检查清单](zh/RELEASE-CHECKLIST.md) | [Lista](es/RELEASE-CHECKLIST.md) |

## Recommended first run

1. Run `make quickstart` from the repository root. It creates or repairs `.env`, creates missing local configs, installs frontend dependencies and starts PostgreSQL + Master. On Windows, use a supported GNU Make environment or run the setup steps from PowerShell as documented in the localized installation guide.
2. In another terminal, run `make run-frontend` and open `http://localhost:3000`.
3. Run `make copy-token` and paste the admin token into **Settings**. The web UI stores it for the current browser session; `lndp` reads it from `.env`.
4. Issue one separate Agent credential per workstation in Settings and save the matching `agent_id` / `token` pair in that Agent's private configuration. Do not mix credentials between agents.

Do not use `docker compose down -v` unless you intend to delete the database volume. PostgreSQL init scripts run only for a new empty volume; existing databases require explicit migrations.

## Operational and security notes

- Backend: Go 1.26.4 or newer; database: PostgreSQL 15 or newer; frontend: Node.js 18+.
- Admin REST calls use `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>`. Agent credentials are separate; PostgreSQL stores only SHA-256 hashes of Agent tokens.
- The Agent currently runs as a foreground process unless an external service manager is configured. Native Windows Service/systemd installers are not yet included.
- The Agent reconnects automatically, but current metrics and device inventory are requested on demand; continuous telemetry and proactive administrator alerts are not implemented.
- Admin-issued per-agent credentials are a security boundary. Do not allow unrestricted self-registration. A future bootstrap flow should use short-lived, single-use enrollment secrets and an approval policy.
- HTTP/WebSocket traffic is not encrypted by default. Use HTTPS/WSS and firewall rules outside trusted isolated networks.
- TUI installation/deletion actions affect real clients and uploads; verify targets before dispatch.

## Testing

- [English testing guide](en/TESTING.md)
- [Руководство по тестированию](ru/TESTING.md)
- [テストガイド](ja/TESTING.md)
- [测试指南](zh/TESTING.md)
- [Guía de pruebas](es/TESTING.md)
