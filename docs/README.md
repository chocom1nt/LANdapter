# LANdapter documentation

LANdapter documentation is available in English, Russian, Chinese (简体中文), Japanese, and Spanish. Choose a starting point below.

## Documentation by language

| Guide | English | Русский | 中文 | 日本語 | Español |
|---|---|---|---|---|---|
| Main project overview | [README](../README.md) | [README](../README-RU.md) | [README](../README-ZH.md) | [README](../README-JA.md) | [README](../README-ES.md) |
| Installation and first run | [INSTALL](INSTALL.md) | [INSTALL](INSTALL-RU.md) | [INSTALL](INSTALL-ZH.md) | [INSTALL](INSTALL-JA.md) | [INSTALL](INSTALL-ES.md) |
| Configuration | [CONFIG](CONFIG.md) | [CONFIG](CONFIG-RU.md) | [CONFIG](CONFIG-ZH.md) | [CONFIG](CONFIG-JA.md) | [CONFIG](CONFIG-ES.md) |
| Architecture and protocol | [ARCHITECTURE](ARCHITECTURE.md) | [ARCHITECTURE](ARCHITECTURE-RU.md) | [ARCHITECTURE](ARCHITECTURE-ZH.md) | [ARCHITECTURE](ARCHITECTURE-JA.md) | [ARCHITECTURE](ARCHITECTURE-ES.md) |
| Production deployment | [DEPLOY](DEPLOY.md) | [DEPLOY](DEPLOY-RU.md) | [DEPLOY](DEPLOY-ZH.md) | [DEPLOY](DEPLOY-JA.md) | [DEPLOY](DEPLOY-ES.md) |
| REST and WebSocket API | [API](README.API.md) | [API](README.API-RU.md) | [API](README.API-ZH.md) | [API](README.API-JA.md) | [API](README.API-ES.md) |
| FAQ and troubleshooting | [FAQ](FAQ.md) | [FAQ](FAQ-RU.md) | [FAQ](FAQ-ZH.md) | [FAQ](FAQ-JA.md) | [FAQ](FAQ-ES.md) |
| Contributing | [CONTRIBUTING](CONTRIBUTING.md) | [CONTRIBUTING](CONTRIBUTING-RU.md) | [CONTRIBUTING](CONTRIBUTING-ZH.md) | [CONTRIBUTING](CONTRIBUTING-JA.md) | [CONTRIBUTING](CONTRIBUTING-ES.md) |

## Recommended reading order

1. Read the project README for the high-level concept and requirements.
2. Follow the installation guide to prepare PostgreSQL, configure secrets, apply migrations, and start Master.
3. Configure and start the web UI, then issue an individual credential for each Agent. Copy the token when it is issued: it is shown only once.
4. Read the API and architecture guides before integrating tools or changing the protocol.
5. Use the production deployment checklist before exposing the service to a network beyond a trusted local environment.

## Important operational notes

- Current backend requirement: Go 1.26.4 or newer; PostgreSQL 15 or newer; Node.js 18+ for the frontend.
- Administrative REST calls require `Authorization: Bearer <LANDAPTER_ADMIN_TOKEN>`. The value is configured in the Master process environment and must be a random secret of at least 32 characters.
- Agents use individual tokens; PostgreSQL stores their SHA-256 hashes, not their original token strings.
- Authentication does not encrypt traffic. Use HTTPS/WSS through a TLS reverse proxy on untrusted networks.
- Docker Compose starts PostgreSQL and Master only. It does not start the frontend or separate Agent processes. Apply SQL migrations before relying on Master.
- Offline installation jobs may remain `pending`, but automatic redelivery on Agent reconnect is not yet implemented.
- Review the supported file types and OS notes in the installation and FAQ guides before deploying packages.
