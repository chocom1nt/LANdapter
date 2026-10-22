#!/bin/bash

# Проверяем, существует ли файл .env
if [ -f .env ]; then
    echo "Файл .env уже существует. Не перезаписываю его — добавь недостающие переменные вручную." >&2
    exit 1
fi

# Функция для генерации случайного секрета (32 байта -> hex строка)
generate_secret() {
    openssl rand -hex 32
}

# Генерируем секреты
dbPassword=$(generate_secret)
adminToken=$(generate_secret)

# Записываем данные в файл .env
cat << EOF > .env
POSTGRES_USER=landapter
POSTGRES_PASSWORD=$dbPassword
LANDAPTER_ADMIN_TOKEN=$adminToken
EOF

echo ".env создан. Секреты сгенерированы."
