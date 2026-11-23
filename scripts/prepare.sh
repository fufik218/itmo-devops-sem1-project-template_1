#!/bin/bash
# prepare.sh — установка зависимостей и подготовка базы данных

echo "Установка зависимостей..."
go mod tidy # подтягивает зависимости

echo "Создание роли validator..."
# Создаём роль, если её нет
sudo -u postgres psql -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'validator') THEN CREATE ROLE validator LOGIN PASSWORD 'val1dat0r'; END IF; END \$\$;"

echo "Создание БД project-sem-1..."
# Создаём БД, если её нет (через DO-блок, а не \gexec)
sudo -u postgres psql -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'project-sem-1') THEN PERFORM dblink_exec('dbname=postgres', 'CREATE DATABASE \"project-sem-1\" OWNER validator'); END IF; END \$\$;" 2>/dev/null || \
sudo -u postgres psql -c "CREATE DATABASE \"project-sem-1\" OWNER validator;" 2>/dev/null || \
echo "БД project-sem-1 уже существует (пропускаем)"

echo "Создание таблицы prices..."
# Подключаемся от validator по TCP
PGPASSWORD=val1dat0r psql -h 127.0.0.1 -U validator -d "project-sem-1" -c "
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL
);"

echo "Готово."
