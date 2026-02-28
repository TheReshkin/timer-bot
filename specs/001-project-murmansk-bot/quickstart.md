# Quickstart: murmansk-bot

## Запуск бота

1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/TheReshkin/timer-bot.git
   cd timer-bot
   ```
2. Установите зависимости:
   ```bash
   go mod tidy
   ```
3. Создайте `.env` файл с токеном:
   ```bash
   echo "TELEGRAM_BOT_TOKEN=your_bot_token_here" > .env
   ```
4. Запустите бота:
   ```bash
   go run ./cmd
   ```
5. Для запуска через Docker:
   ```bash
   docker-compose -f tg-bot.docker-compose.yml up --build
   ```

## Основные команды
- `/set_date <дата> <имя> [описание]` — добавить событие
  - Поддерживаемые форматы: `YYYY-MM-DD`, `YYYY-MM-DD HH:MM`, `DD.MM.YYYY`
  - Пример: `/set_date 2025-12-31 new_year "Новый год"`
- `/<event_name>` — информация о событии (динамическая команда)
- `/list` — краткий список событий
- `/all` — полный список событий
- `/active` — список активных событий
- `/outdated` — список устаревших событий
- `/help` — справка по командам

## Структура проекта
- `cmd/` — точка входа
- `internal/models/` — модели данных
- `internal/services/` — бизнес-логика
- `internal/storage/` — хранение данных
- `tests/unit/` — юнит-тесты
- `tests/integration/` — интеграционные тесты
- `specs/` — спецификации и документация

## Тестирование
```bash
# Все тесты
go test ./...

# Только юнит-тесты
go test ./tests/unit/

# Только интеграционные тесты
go test ./tests/integration/
```

## Расширение
- Хранение событий: JSON (текущее), PostgreSQL (планируется)
- Локализация: структура подготовлена для будущих языков
- Многочатность: поддержка нескольких чатов
- Логирование: структурированные логи с Zap

## Примеры использования
```
/set_date 2025-12-31 new_year "Новый год 2025"
/set_date 2025-09-07 14:30 birthday "День рождения"
/set_date 07.09.2025 vacation "Отпуск"
/new_year
/list
/active
```
