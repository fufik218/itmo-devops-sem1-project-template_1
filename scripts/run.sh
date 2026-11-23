#!/bin/bash
# run.sh — запуск Go-сервера локально

# Определяем директорию, где лежит сам скрипт
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Поднимаемся на уровень выше — в корень проекта, где лежит main.go
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Переходим в корень проекта
cd "$PROJECT_ROOT" || { echo "Не удалось перейти в $PROJECT_ROOT"; exit 1; }

# Проверяем, что main.go существует
if [ ! -f "main.go" ]; then
    echo "Ошибка: main.go не найден в $PROJECT_ROOT"
    exit 1
fi

echo "Запуск сервера из $PROJECT_ROOT..."
go run main.go # компилирует и запускает сервер
