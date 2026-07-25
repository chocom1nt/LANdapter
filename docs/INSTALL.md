# Установка и запуск LANdapter

В этом руководстве описаны шаги по установке и запуску LANdapter в среде разработки или для тестирования. Для production-развертывания см. [DEPLOY.md](DEPLOY.md).

---

## Требования

- **Go** 1.21 или выше – для сборки бэкенда
- **PostgreSQL** 15 или выше – для хранения данных
- **Node.js** 18 или выше и **npm** – для фронтенда (React)
- **Git** – для клонирования репозитория
- **Make** (опционально, но рекомендуется) – для упрощённого запуска команд

Для Windows рекомендуется использовать **Git Bash** или **WSL2** для выполнения команд Makefile.

---

## 1. Клонирование репозитория

```bash
git clone https://github.com/your-username/LANdapter.git
cd LANdapter
```

---

## 2. Настройка базы данных

Создайте базу данных PostgreSQL и пользователя для LANdapter.

```sql
CREATE USER landapter WITH PASSWORD 'your_password';
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
  password: "your_password"
  dbname: "landapter"
  sslmode: "disable"
```

### `configs/agent.yaml`

```yaml
master_host: "localhost"
master_port: 8081
uuid_file: "./agent.uuid"
installer_args:
  ".exe": "/S"
```

Если мастер запускается на другом хосте, укажите его IP вместо `localhost`.

---

## 4. Применение миграций базы данных

В папке `migrations/` находятся SQL-скрипты для создания структуры БД. Примените их в порядке возрастания номера.

### Через psql

```bash
psql -h localhost -U landapter -d landapter -f migrations/001_init.up.sql
psql -h localhost -U landapter -d landapter -f migrations/002_add_mac_up.sql
psql -h localhost -U landapter -d landapter -f migrations/003_add_devices_up.sql
psql -h localhost -U landapter -d landapter -f migrations/004_add_snapshots.up.sql
```

### Через Makefile

```bash
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

## 8. Запуск агента

В отдельном терминале (или на другой машине):

```bash
./bin/agent
# или
go run cmd/agent/main.go
```

Агент подключится к мастеру и будет ждать команд.

---

## 9. Установка и запуск фронтенда

Перейдите в папку `web/` и установите зависимости:

```bash
cd web
npm install
```

Запустите сервер разработки:

```bash
npm run dev
```

Фронтенд будет доступен по адресу `http://localhost:3000`. Он автоматически проксирует запросы к API мастера (`/api` → `http://localhost:8080`).

Для production-сборки:

```bash
npm run build
```

Готовая статика появится в `web/dist/`. Её можно раздать через любой веб-сервер (Nginx, Caddy и т.д.).

---

## 10. Запуск через Docker Compose

Самый быстрый способ поднять полный стек (PostgreSQL + мастер + агент) – использовать Docker Compose.

```bash
docker-compose up -d
```

Это запустит:
- PostgreSQL на порту 5432
- Мастер на портах 8080 (HTTP) и 8081 (WebSocket)
- Агент (один экземпляр, подключенный к мастеру)

Фронтенд нужно запускать отдельно (см. шаг 9) или собирать и раздавать статику через Nginx.

---

## 11. Проверка работоспособности

### Проверка API мастера

```bash
curl http://localhost:8080/api/v1/clients
```

Должен вернуться пустой массив `[]` или список клиентов.

### Проверка WebSocket агента

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