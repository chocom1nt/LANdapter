# Установка и запуск LANdapter


## Рекомендуемый быстрый запуск (Docker)

Для новой локальной установки начните в корне репозитория:

```bash
make setup
make up
make install-frontend
make copy-token
make run-frontend
```

`make setup` создаёт или дополняет `.env`, не меняя существующие корректные секреты, и копирует YAML-конфиги из примеров только при отсутствии локальных файлов. На **новом Docker volume** PostgreSQL автоматически применяет SQL-миграции при инициализации. Существующий том не очищается и не мигрируется этими init-скриптами. `make copy-token` копирует admin token, чтобы один раз вставить его в настройки веб-интерфейса; `lndp` читает токен из `.env` автоматически.

Дальнейшие разделы описывают ручную настройку PostgreSQL и существующие установки. Не выполняйте `docker compose down -v`, если не собираетесь удалить том базы данных.

**Языки:** [English](../en/INSTALL.md) · [Русский](INSTALL.md) · [中文](../zh/INSTALL.md) · [日本語](../ja/INSTALL.md) · [Español](../es/INSTALL.md)

В этом руководстве описаны шаги по установке и запуску LANdapter в среде разработки или для тестирования. Для production-развертывания см. [Развёртывание](DEPLOY.md).

---

## Требования

- **Go** 1.26.4 или выше – для сборки бэкенда
- **PostgreSQL** 15 или выше – для хранения данных
- **Node.js** 18 или выше и **npm** – для фронтенда (React)
- **Git** – для клонирования репозитория
- **Make** (опционально, но рекомендуется) – для упрощённого запуска команд

Для Windows рекомендуется использовать **Git Bash** или **WSL2** для выполнения команд Makefile.

---

## 1. Клонирование репозитория

```bash
git clone https://github.com/chocom1nt/LANdapter.git
cd LANdapter
```

---

## 2. Настройка базы данных

Создайте базу данных PostgreSQL и пользователя для LANdapter.

```sql
CREATE USER landapter WITH PASSWORD 'your-database-password';
CREATE DATABASE landapter OWNER landapter;
```

Убедитесь, что PostgreSQL запущен и доступен по адресу, указанному в конфигурации.

---

## 3. Настройка конфигурационных файлов

Скопируйте примеры конфигов и отредактируйте их под ваше окружение.

```bash
cp configs/master.yaml.example configs/master.yaml
cp configs/agent.yaml.example configs/agent.yaml
```

### `configs/master.yaml`

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: "" # задайте DB_PASSWORD в окружении
  dbname: "landapter"
  sslmode: "disable"
```

### `configs/agent.yaml`

```yaml
master_host: "localhost"
master_port: 8081
master_http_port: 8080
master_scheme: "ws" # use wss behind a TLS reverse proxy
uuid: "" # filled from the credential issued by Master Settings
token: "" # per-agent secret; keep this file private
uuid_file: "./agent.uuid"
installer_args:
  ".exe": "/S"
```

Если мастер запускается на другом хосте, укажите его IP вместо `localhost`.

---

## 4. Применение миграций базы данных

В папке `migrations/` находятся SQL-скрипты для создания структуры БД. Примените их в порядке возрастания номера.

### Через psql

Для локального запуска задайте пароль БД и отдельный административный токен (не менее 32 символов):

```bash
export DB_PASSWORD='your-database-password' # тот же пароль, что задан пользователю landapter выше
export LANDAPTER_ADMIN_TOKEN="$(openssl rand -hex 32)"
make migrate-up
```

Для отката миграций:

```bash
make migrate-down
```

---

## 5. Установка зависимостей Go

```bash
go mod download
```

---

## 6. Сборка бэкенда

Соберите исполняемые файлы мастера и агента:

```bash
make build
```

Или вручную:

```bash
go build -o bin/master cmd/master/main.go
go build -o bin/agent cmd/agent/main.go
```

Бинарники появятся в папке `bin/`.

---

## 7. Запуск мастера

```bash
./bin/master
# или
go run cmd/master/main.go
```

Мастер запустит HTTP-сервер (по умолчанию на порту 8080) и WebSocket-сервер (порт 8081).

---

## 8. Установка и запуск фронтенда

Перейдите в папку `web/` и установите зависимости:

```bash
cd web
npm install
```

Запустите сервер разработки:

```bash
npm run dev
```

Откройте `http://localhost:3000`, перейдите в «Настройки», укажите `LANDAPTER_ADMIN_TOKEN`, сохраните подключение и выпустите персональный токен агента. Фронтенд проксирует API мастера (`/api` → `http://localhost:8080`); настройте учётные данные агента до его запуска.

Для production-сборки:

```bash
npm run build
```

Готовая статика появится в `web/dist/`. Её можно раздать через любой веб-сервер (Nginx, Caddy и т.д.).

---

## 9. Запуск агента

Сначала в настройках Master выпустите персональный токен и перенесите `agent_id` и `token` в `configs/agent.yaml` как поля `uuid` и `token`. Затем в отдельном терминале (или на другой машине):

```bash
./bin/agent
# или
go run cmd/agent/main.go
```

Агент подключится к мастеру и будет ждать команд.

---

## 10. Запуск через Docker Compose

Быстрый способ запустить PostgreSQL и мастер – использовать Docker Compose. Агентов запускают отдельно после выдачи персональных токенов.

```bash
cp .env.example .env
# Отредактируйте .env: задайте разные случайные POSTGRES_PASSWORD и LANDAPTER_ADMIN_TOKEN
docker-compose up -d postgres
set -a
. ./.env
set +a
export DB_PASSWORD="$POSTGRES_PASSWORD"
make migrate-up
docker-compose up -d master
```

Это запустит PostgreSQL и мастер. Опубликованные порты по умолчанию привязаны только к `127.0.0.1`; не открывайте их напрямую для удалённых агентов — используйте TLS reverse proxy. В настройках панели выпустите персональный токен, заполните `uuid` и `token` в `configs/agent.yaml`, затем запускайте каждый агент отдельно.

Фронтенд нужно запускать отдельно (см. шаг 8) или собирать и раздавать статику через Nginx.

---

## 11. Проверка работоспособности

### Проверка API мастера

```bash
curl -H "Authorization: Bearer $LANDAPTER_ADMIN_TOKEN" \
  http://localhost:8080/api/v1/clients
```

Должен вернуться пустой массив `[]` или список клиентов.

### Проверка WebSocket агента

Перед первым запуском откройте «Настройки» в панели Master, выпустите токен агента и перенесите `agent_id`/`token` в `configs/agent.yaml` как поля `uuid` и `token`. Токен показывается один раз. Для удалённых подключений используйте TLS (`https`/`wss`) через обратный прокси; не передавайте токены по обычному HTTP через недоверенные сети.

В логах агента должно быть сообщение:

```
Connected, waiting for commands
```

### Проверка фронтенда

Откройте браузер по адресу `http://localhost:3000`. Должна загрузиться панель управления.

---

## 12. Устранение неполадок

### Ошибка подключения к БД

- Убедитесь, что PostgreSQL запущен: `systemctl status postgresql` (Linux) или служба запущена в Windows.
- Проверьте правильность параметров в `master.yaml` (хост, порт, пользователь, пароль, имя БД).
- Убедитесь, что база данных создана и миграции применены.

### Агент не подключается

- Проверьте, что мастер запущен и слушает порт 8081: `netstat -an | grep 8081`.
- Убедитесь, что в `agent.yaml` указан правильный `master_host` (IP мастера).
- Проверьте брандмауэр – порты должны быть открыты.

### Фронтенд не отображается

- Убедитесь, что сервер разработки запущен (`npm run dev`).
- Проверьте, что порт 3000 не занят другим приложением.
- В консоли браузера (F12) посмотрите ошибки загрузки ресурсов.

---

## Дальнейшие шаги

- Изучите [документацию API](README.API.md) для работы с эндпоинтами.
- Настройте [конфигурацию](CONFIG.md) под ваши нужды.
- Для развертывания в продакшн обратитесь к [руководству по развертыванию](DEPLOY.md).