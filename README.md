# itmo-devops-sem1-project-template

REST API на Go + PostgreSQL для простого уровня.

## Требования

- Go 1.21+
- PostgreSQL (порт 5432)
- Утилиты: `zip`, `curl`, `psql`

## Быстрый старт

```bash
# 1. Подготовка БД и зависимостей (один раз)
./scripts/prepare.sh

# 2. Запуск сервера (в отдельном терминале, не закрывать)
./scripts/run.sh

# 3. Тесты (в третьем терминале)
./scripts/tests.sh 1
