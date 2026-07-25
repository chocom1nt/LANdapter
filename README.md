# LANdapter

<p align="center">
  <img src="docs/assets/logo.png" width="128" alt="LANdapter logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/react-18.2-61DAFB?style=flat&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/platform-windows%20%7C%20linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/postgresql-15-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL">
</p>

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
- **Кроссплатформенность** – мастер и агенты работают на Windows и Linux (агент также поддерживает macOS в режиме ограниченной функциональности).

---

## Быстрый старт

### Требования

- Go 1.21+
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
npm install
cd ..
```

Примените миграции базы данных:

```bash
make migrate-up
# или вручную:
psql -h localhost -U postgres -d landapter -f migrations/001_init.up.sql
psql -h localhost -U postgres -d landapter -f migrations/002_add_mac_up.sql
psql -h localhost -U postgres -d landapter -f migrations/003_add_devices_up.sql
psql -h localhost -U postgres -d landapter -f migrations/004_add_snapshots.up.sql
```

Запустите мастер:

```bash
go run cmd/master/main.go
```

Запустите агента (в отдельном терминале):

```bash
go run cmd/agent/main.go
```

Запустите фронтенд (в третьем терминале):

```bash
cd web
npm run dev
```

Откройте в браузере `http://localhost:3000`.

Подробная инструкция по развёртыванию доступна в [docs/INSTALL.md](docs/INSTALL.md).

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

Для мастера и агента предусмотрены отдельные Dockerfile:

```bash
# Мастер
docker build -f Dockerfile.master -t landapter-master .

# Агент
docker build -f Dockerfile.agent -t landapter-agent .
```

Или запустите полный стек через `docker-compose` (с PostgreSQL):

```bash
docker-compose up -d
```

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
├── Makefile
├── go.mod
└── README.md
```

---

## Документация

Полная документация находится в папке [docs/](docs/):

- [API Reference](docs/README.API-RU.md) – описание всех REST и WebSocket эндпоинтов (русский / [английский](docs/README.API.md)).
- [Инструкция по установке](docs/INSTALL.md) – детальное руководство по развёртыванию.
- [Конфигурация](docs/CONFIG.md) – описание параметров `master.yaml` и `agent.yaml`.
- [Архитектура](docs/ARCHITECTURE.md) – принципы работы, жизненный цикл заданий, протокол WebSocket.
- [Развёртывание в продакшн](docs/DEPLOY.md) – настройка сервисов, прокси, мониторинга.
- [FAQ](docs/FAQ.md) – частые вопросы и решения проблем.

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