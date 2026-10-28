# Production Deployment

**Languages:** [English](DEPLOY.md) · [Русский](../ru/DEPLOY.md) · [中文](../zh/DEPLOY.md) · [日本語](../ja/DEPLOY.md) · [Español](../es/DEPLOY.md)

This guide covers deploying LANdapter in production with reliability, security, and scalability in mind.

---


## Secrets and agent enrollment

Set `LANDAPTER_ADMIN_TOKEN` to a random string of at least 32 characters and `DB_PASSWORD` to the PostgreSQL password before starting Master. Master refuses to start without the administrative token. Do not expose ports 8080/8081 directly to untrusted networks; use HTTPS/WSS through a TLS reverse proxy. Apply migrations with `make migrate-up`, then issue one individual token per agent in Master → Settings. The secret is shown only once and must be saved in `configs/agent.yaml` as `uuid` and `token` with restrictive file permissions.


## Server preparation

### Hardware

- **CPU:** 2 cores (4+ recommended).
- **RAM:** 2 GB (4+ recommended).
- **Disk:** 20 GB plus space for uploads.
- **OS:** Ubuntu 22.04 LTS (recommended) or Windows Server 2022.

### Software

**Update system:**

```bash
sudo apt update && sudo apt upgrade -y
```

**Install Go:**

```bash
wget https://go.dev/dl/go1.26.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Install PostgreSQL:**

```bash
sudo apt install postgresql-15 -y
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

**Node.js (if building frontend on server):**

```bash
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs
```

---

## Database setup

```sql
sudo -u postgres psql
CREATE USER landapter WITH PASSWORD 'your-database-password';
CREATE DATABASE landapter OWNER landapter;
GRANT ALL PRIVILEGES ON DATABASE landapter TO landapter;
\q
```

Apply migrations:

```bash
psql -h localhost -U landapter -d landapter -f migrations/001_init.up.sql
psql -h localhost -U landapter -d landapter -f migrations/002_add_mac.up.sql
psql -h localhost -U landapter -d landapter -f migrations/003_add_devices.up.sql
psql -h localhost -U landapter -d landapter -f migrations/004_add_snapshots.up.sql
psql -h localhost -U landapter -d landapter -f migrations/005_agent_credentials.up.sql
```

---

## Master build and setup

### Configuration

Edit `/opt/landapter/configs/master.yaml` (the master reads this path relative to its working directory):

```yaml
host: "0.0.0.0"
http_port: 8080
ws_port: 8081
db:
  host: "localhost"
  port: 5432
  user: "landapter"
  password: "" # set DB_PASSWORD in the service environment
  dbname: "landapter"
  sslmode: "disable"
```

Prefer `DB_PASSWORD` via environment instead of storing secrets in the file.

### Build

```bash
cd /opt/landapter
make build
# or:
go build -o bin/master cmd/master/main.go
```

### systemd (Linux)

`/etc/systemd/system/landapter-master.service`:

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

Create the service user and a protected environment file. Put the same database password used when creating the PostgreSQL user in `DB_PASSWORD`, and a separate random value of at least 32 characters in `LANDAPTER_ADMIN_TOKEN` (generate one with `openssl rand -hex 32`):

```bash
sudo useradd -r -s /bin/false landapter
sudo chown -R landapter:landapter /opt/landapter
sudo install -d -o root -g landapter -m 0750 /etc/landapter
sudo install -o root -g landapter -m 0640 /dev/null /etc/landapter/landapter.env
sudoedit /etc/landapter/landapter.env
```

The environment file must contain (replace both example values):

```dotenv
DB_PASSWORD=your-database-password
LANDAPTER_ADMIN_TOKEN=replace-with-output-of-openssl-rand-hex-32
```

Enable service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable landapter-master
sudo systemctl start landapter-master
sudo systemctl status landapter-master
```

---

## Agents

### Configuration

`/opt/landapter/configs/agent.yaml` (relative to the agent's working directory):

```yaml
master_host: "192.168.1.100"
master_port: 8081
master_http_port: 8080
master_scheme: "ws" # use wss and the TLS proxy's port for remote networks
uuid: "<agent-id-issued-by-master>"
token: "<one-time-agent-token-issued-by-master>"
uuid_file: "/var/lib/landapter/agent.uuid"
installer_args:
  ".exe": "/S"
```

Build and deploy `bin/agent` to each target machine.

### Windows service (NSSM example)

```cmd
nssm install LANdapterAgent "C:\landapter\bin\agent.exe"
# In NSSM, set AppDirectory to C:\landapter so configs\agent.yaml is found.
nssm start LANdapterAgent
```

---

## Frontend and web server

### Build static assets

```bash
cd web
npm install
npm run build
```

Output: `web/dist/`.

### Nginx

```bash
sudo apt install nginx -y
```

`/etc/nginx/sites-available/landapter`:

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

Enable:

```bash
sudo ln -s /etc/nginx/sites-available/landapter /etc/nginx/sites-enabled/
sudo systemctl reload nginx
```

---

## Security

### Firewall

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

Keep 8080, 8081, and 5432 off the public internet.

### HTTPS

```bash
sudo apt install certbot python3-certbot-nginx -y
sudo certbot --nginx -d landapter.example.com
```

---

## Backups

Back up regularly:

- Database: `pg_dump`
- Uploads: `uploads/` directory

Example script:

```bash
#!/bin/bash
BACKUP_DIR=/backup/landapter
mkdir -p $BACKUP_DIR
pg_dump -U landapter landapter > $BACKUP_DIR/db_$(date +%Y%m%d).sql
tar -czf $BACKUP_DIR/uploads_$(date +%Y%m%d).tar.gz /opt/landapter/uploads
```

Schedule with cron.

---

## Monitoring and logs

- Master logs to stdout; use `journalctl -u landapter-master -f` with systemd.
- Agents similarly when run as services.
- Prometheus export can be added; use system tools (`top`, `htop`, `netstat`) meanwhile.

---

## Upgrades

1. `sudo systemctl stop landapter-master`
2. Deploy new build
3. Run new migrations if any
4. Start the service

Update agents on a similar schedule.
