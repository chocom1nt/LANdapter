# Развёртывание LANdapter в продакшн

**Языки:** [English](DEPLOY.md) · [Русский](DEPLOY-RU.md) · [中文](DEPLOY-ZH.md) · [日本語](DEPLOY-JA.md) · [Español](DEPLOY-ES.md)

В этом руководстве описаны шаги по развёртыванию LANdapter в рабочей среде с учётом надёжности, безопасности и масштабируемости.

---


## Секреты и регистрация агентов

Перед запуском Master задайте `LANDAPTER_ADMIN_TOKEN` случайной строкой минимум из 32 символов и `DB_PASSWORD` — пароль пользователя PostgreSQL. Без административного токена сервер не стартует. Не публикуйте порты 8080/8081 напрямую в недоверенную сеть; используйте HTTPS/WSS через TLS reverse proxy. После запуска примените миграции `make migrate-up`, затем в панели Master → Настройки выпустите токен каждому агенту. Секрет агента показывается только один раз и должен быть записан в `configs/agent.yaml` в поля `uuid` и `token` с ограниченными правами чтения файла.


## Подготовка сервера

### Требования к оборудованию

- **CPU**: 2 ядра (рекомендуется 4+).
- **RAM**: 2 ГБ (рекомендуется 4+).
- **Диск**: 20 ГБ + место для загружаемых файлов.
- **ОС**: Ubuntu 22.04 LTS (или аналогичная) / Windows Server 2022 (поддерживается, но рекомендуем Linux).

### Установка ПО

**Обновление системы:**

```bash
sudo apt update && sudo apt upgrade -y
```

**Установка Go:**

```bash
wget https://go.dev/dl/go1.26.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Установка PostgreSQL:**

```bash
sudo apt install postgresql-15 -y
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

**Установка Node.js (для сборки фронтенда, если он будет собираться на сервере):**

```bash
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs
```

---

## Настройка базы данных

Создайте пользователя и базу данных:

```sql
sudo -u postgres psql
CREATE USER landapter WITH PASSWORD 'your-database-password';
CREATE DATABASE landapter OWNER landapter;
GRANT ALL PRIVILEGES ON DATABASE landapter TO landapter;
\q
```

Примените миграции:

```bash
psql -h localhost -U landapter -d landapter -f migrations/001_init.up.sql
psql -h localhost -U landapter -d landapter -f migrations/002_add_mac.up.sql
psql -h localhost -U landapter -d landapter -f migrations/003_add_devices.up.sql
psql -h localhost -U landapter -d landapter -f migrations/004_add_snapshots.up.sql
psql -h localhost -U landapter -d landapter -f migrations/005_agent_credentials.up.sql
```

---

## Сборка и настройка мастера

### Конфигурация

Отредактируйте `/opt/landapter/configs/master.yaml` (мастер ищет конфигурацию относительно рабочей директории):

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: "" # задайте DB_PASSWORD в окружении сервиса
  dbname: "landapter"
  sslmode: "disable"
```

**Важно:** Пароль лучше передавать через переменную окружения `DB_PASSWORD`, чтобы не хранить его в файле. Тогда в YAML можно оставить пустым или закомментировать.

### Сборка бинарника

```bash
cd /opt/landapter
make build
# или вручную:
go build -o bin/master cmd/master/main.go
```

Бинарник появится в `bin/master`.

### Настройка systemd-сервиса (Linux)

Создайте файл `/etc/systemd/system/landapter-master.service`:

```ini
[Unit]
Description=LANdapter Master Server
After=network.target postgresql.service

[Service]
Type=simple
User=landapter
Group=landapter
WorkingDirectory=/opt/landapter
ExecStart=/opt/landapter/bin/master
Restart=always
RestartSec=5
EnvironmentFile=/etc/landapter/landapter.env

[Install]
WantedBy=multi-user.target
```

Создайте пользователя сервиса и защищённый файл окружения. В `DB_PASSWORD` укажите тот же пароль, что использовали при создании пользователя PostgreSQL, а в `LANDAPTER_ADMIN_TOKEN` — отдельный случайный секрет минимум из 32 символов (можно сгенерировать командой `openssl rand -hex 32`):

```bash
sudo useradd -r -s /bin/false landapter
sudo chown -R landapter:landapter /opt/landapter
sudo install -d -o root -g landapter -m 0750 /etc/landapter
sudo install -o root -g landapter -m 0640 /dev/null /etc/landapter/landapter.env
sudoedit /etc/landapter/landapter.env
```

Файл должен содержать (замените оба примера реальными значениями):

```dotenv
DB_PASSWORD=your-database-password
LANDAPTER_ADMIN_TOKEN=replace-with-output-of-openssl-rand-hex-32
```

Запустите сервис:

```bash
sudo systemctl daemon-reload
sudo systemctl enable landapter-master
sudo systemctl start landapter-master
sudo systemctl status landapter-master
```

---

## Настройка агентов

Агенты могут быть установлены на каждой целевой машине.

### Конфигурация агента

Используйте `/opt/landapter/configs/agent.yaml` относительно рабочей директории агента:

```yaml
master_host: "192.168.1.100"   # IP мастера
master_port: 8081
master_http_port: 8080
master_scheme: "ws" # для удалённой сети используйте wss и порт TLS-прокси
uuid: "<agent-id-issued-by-master>"
token: "<one-time-agent-token-issued-by-master>"
uuid_file: "/var/lib/landapter/agent.uuid"
installer_args:
  ".exe": "/S"
```

### Сборка агента

Аналогично мастеру, соберите бинарник и разместите его на клиентских машинах.

### Запуск агента как сервиса (Windows)

Для Windows можно создать задачу в планировщике или использовать NSSM (Non-Sucking Service Manager).

**Пример с NSSM:**

```cmd
nssm install LANdapterAgent "C:\landapter\bin\agent.exe"
# В NSSM задайте AppDirectory = C:\landapter, чтобы находился configs\agent.yaml.
nssm start LANdapterAgent
```

---

## Настройка фронтенда и веб-сервера

### Сборка статики

```bash
cd web
npm install
npm run build
```

Статика будет в `web/dist/`.

### Настройка Nginx

Установите Nginx:

```bash
sudo apt install nginx -y
```

Создайте конфигурацию `/etc/nginx/sites-available/landapter`:

```nginx
server {
    listen 80;
    server_name landapter.example.com;

    root /var/www/landapter;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location = /ws {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

Активируйте:

```bash
sudo ln -s /etc/nginx/sites-available/landapter /etc/nginx/sites-enabled/
sudo systemctl reload nginx
```

Путь WebSocket — точно `/ws` (завершающий слэш не требуется). Если API и WebSocket публикуются через один HTTPS-порт (обычно 443), укажите в конфигурации агента `master_scheme: "wss"`, `master_port: 443` и `master_http_port: 443`.

Путь WebSocket — точно `/ws` (завершающий слэш не требуется). Если API и WebSocket публикуются через один HTTPS-порт (обычно 443), укажите в конфигурации агента `master_scheme: "wss"`, `master_port: 443` и `master_http_port: 443`.

---

## Безопасность

### Брандмауэр

Разрешите только необходимые порты:

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp   # если используете HTTPS
sudo ufw enable
```

Внутренние порты (8080, 8081, 5432) должны быть закрыты для внешнего мира.

### HTTPS

Настройте Let's Encrypt с помощью Certbot:

```bash
sudo apt install certbot python3-certbot-nginx -y
sudo certbot --nginx -d landapter.example.com
```

---

## Резервное копирование

Регулярно делайте бэкап:

- **Базы данных** – через `pg_dump`.
- **Загруженных файлов** – папка `uploads/`.

Пример скрипта:

```bash
#!/bin/bash
BACKUP_DIR=/backup/landapter
mkdir -p $BACKUP_DIR
pg_dump -U landapter landapter > $BACKUP_DIR/db_$(date +%Y%m%d).sql
tar -czf $BACKUP_DIR/uploads_$(date +%Y%m%d).tar.gz /opt/landapter/uploads
```

Настройте cron для ежедневного выполнения.

---

## Мониторинг и логи

### Логи

- Мастер пишет логи в stdout. Для systemd логи можно смотреть через `journalctl -u landapter-master -f`.
- Логи агентов аналогично, если они запущены как сервисы.

### Метрики

При необходимости добавьте экспорт метрик в Prometheus (можно доработать). Пока можно использовать системные утилиты (`top`, `htop`, `netstat`).

---

## Обновление

1. Остановите сервис мастера: `sudo systemctl stop landapter-master`.
2. Скачайте новую версию, соберите бинарник.
3. Примените новые миграции (если есть).
4. Запустите сервис.

Агенты обновляются отдельно, но рекомендуется обновлять их в то же время.

