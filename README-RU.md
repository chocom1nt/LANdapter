# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.26.4+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/react-18.2-61DAFB?style=flat&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/platform-windows%20%7C%20linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/postgresql-15-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL">
</p>

**Языки:** [English](README.md) · [Русский](README-RU.md) · [中文](README-ZH.md) · [日本語](README-JA.md) · [Español](README-ES.md)

**LANdapter** – клиент-серверная система для централизованной удалённой установки драйверов и программного обеспечения в локальной сети.

Система состоит из **мастера** (сервер) и **агентов** (клиентские машины). Мастер управляет подключёнными агентами, раздаёт файлы и отслеживает ход установок. Агенты получают команды через WebSocket, скачивают файлы и выполняют установку в тихом или интерактивном режиме.

Проект создан для упрощения администрирования рабочих станций в офисных и промышленных сетях, где требуется оперативно обновлять драйверы или устанавливать приложения на десятки и сотни машин.

---

## Возможности

- **Централизованное управление** – все агенты подключаются к единому мастер-серверу.
- **Удалённая установка** – передача файлов (EXE, MSI, INF, DEB, RUN, TAR) на агенты и запуск с нужными параметрами.
- **Два режима установки** – тихая (без UI) и интерактивная (с пользовательским интерфейсом).
- **Сбор статистики** – получение системных метрик (CPU, RAM, uptime) и списка устройств (PnP, lsusb, lspci).
- **История и отчёты** – каждое задание сохраняется в базе данных вместе со снимками состояния системы до и после установки.
- **Точки восстановления** – автоматическое создание точки восстановления перед установкой на Windows.
- **Веб-интерфейс** – удобная панель управления на React с тёмной темой, библиотекой файлов и мастером установки.
- **Гибкая конфигурация** – YAML-файлы, поддержка переменных окружения.
- **Кроссплатформенность** – Master и Agent поддерживают Windows и Linux. Выполнение установок на macOS пока не поддерживается.

---

## Быстрый старт

### Требования

- Go 1.26.4+
- PostgreSQL 15+
- Node.js 18+ (для фронтенда)
- Make (опционально)

### Установка и запуск

Клонируйте репозиторий:

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
```

Скачайте зависимости Go:

```bash
go mod download
```

Установите зависимости фронтенда:

```bash
cd web
npm ci
cd ..
```

Перед применением миграций создайте отдельного пользователя и базу PostgreSQL от имени администратора PostgreSQL (например, в `psql`):

```sql
CREATE USER landapter WITH PASSWORD 'ваш-пароль-БД';
CREATE DATABASE landapter OWNER landapter;
```

Ниже в `DB_PASSWORD` укажите тот же пароль. Если пользователь или база уже существуют, не создавайте их повторно — проверьте владельца и права.

Перед применением миграций и запуском Master задайте пароль PostgreSQL и отдельный секрет администратора (не менее 32 символов):

```bash
export DB_PASSWORD='ваш-пароль-БД'
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
```

Примените миграции базы данных:

```bash
make migrate-up
```

Запустите мастер:

```bash
go run cmd/master/main.go
```

Запустите фронтенд (в отдельном терминале):

```bash
cd web
npm run dev
```

Откройте `http://localhost:3000`, перейдите в «Настройки», укажите `LANDAPTER_ADMIN_TOKEN`, сохраните подключение и выпустите персональный токен агента. Токен показывается один раз: перенесите `agent_id` и `token` в `configs/agent.yaml` как поля `uuid` и `token`.

Только после сохранения учётных данных запустите агента в другом терминале:

```bash
go run cmd/agent/main.go
```

Подробная инструкция по развёртыванию доступна в [docs/INSTALL-RU.md](docs/INSTALL-RU.md).

---

## Сборка из исходников

### Бинарники Go

Соберите мастер и агента:

```bash
make build
```

Либо вручную:

```bash
go build -o bin/master cmd/master/main.go
go build -o bin/agent cmd/agent/main.go
```

### Фронтенд

Соберите React-приложение:

```bash
cd web
npm run build
```

Готовая статика появится в `web/dist/`.

### Docker-образы

Корневой `Dockerfile` собирает образ Master (в него также включён бинарник агента), а `Dockerfile.agent` — отдельный образ Agent:

```bash
docker build -t landapter-master .
docker build -f Dockerfile.agent -t landapter-agent .
```

`docker-compose.yml` запускает **только PostgreSQL и Master**. Фронтенд и агенты автоматически не запускаются. Скопируйте `configs/master.yaml.example` в `configs/master.yaml`, настройте `.env`, примените SQL-миграции и затем запускайте сервисы по [docs/INSTALL-RU.md](docs/INSTALL-RU.md). Фронтенд запускается отдельно либо раздаётся веб-сервером из `web/dist/`.

---

## Структура проекта

```text
LANdapter/
├── cmd/                  # Точки входа
│   ├── master/
│   └── agent/
├── internal/             # Внутренние пакеты
│   ├── common/           # Общие структуры, логирование, конфиг
│   ├── master/           # Логика мастера (HTTP, WebSocket, хендлеры)
│   └── agent/            # Логика агента (подключение, установка)
├── storage/              # Слой данных (интерфейс + PostgreSQL)
├── migrations/           # SQL-миграции
├── web/                  # React-фронтенд
├── docs/                 # Документация
├── configs/              # Примеры конфигурационных файлов
├── uploads/              # Папка для загруженных файлов (создаётся автоматически)
├── docker-compose.yml
├── makefile
├── go.mod
└── README.md
```

---

## Документация

Полный каталог документации: [docs/README.md](docs/README.md). Начните с руководства, которое соответствует вашей задаче:

- [Установка](docs/INSTALL-RU.md) — требования, база данных, миграции, запуск Master/UI и регистрация агента.
- [Конфигурация](docs/CONFIG-RU.md) — параметры Master и Agent и переменные окружения.
- [Архитектура](docs/ARCHITECTURE-RU.md) — компоненты, учётные данные, WebSocket и текущие ограничения.
- [Развёртывание в production](docs/DEPLOY-RU.md) — TLS-прокси, защита сервисов, резервные копии и обновления.
- [API](docs/README.API-RU.md) — REST-эндпоинты и протокол WebSocket.
- [FAQ](docs/FAQ-RU.md) и [участие в разработке](docs/CONTRIBUTING-RU.md).

---

## Тестирование

Запуск юнит-тестов:

```bash
make test
```

Интеграционные тесты (требуют реальную БД):

```bash
make test-integration
```

Отчёт о покрытии:

```bash
make test-cover
```

---

## Планы развития

- Поддержка групповых политик (выбор клиентов по группам)
- CLI-утилита для мастера (управление без веб-интерфейса)
- Интеграция с Active Directory / LDAP
- Поддержка дополнительных форматов пакетов (AppImage, Flatpak)
- Планирование заданий по расписанию
- Улучшенный парсинг драйверов с веб-сайтов

---

## Лицензия

Проект распространяется под лицензией MIT. Подробнее см. файл [LICENSE](LICENSE).

---

Если у вас есть вопросы или предложения, создавайте [Issues](https://github.com/chocom1nt/LANdapter/issues) или отправляйте Pull Request.
