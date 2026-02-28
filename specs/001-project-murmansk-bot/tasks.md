# Tasks: murmansk-bot MVP

**Input**: Design documents from `/specs/001-project-murmansk-bot/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/

## Phase 3.1: Setup
- [x] T001 Создать структуру проекта: cmd/, internal/, pkg/, tests/
- [x] T002 Инициализировать Go-модуль и зависимости (go mod tidy)
- [ ] T003 [P] Настроить линтер и форматирование (golangci-lint)

## Phase 3.2: Tests First (TDD)
- [x] T004 [P] Реализовать контрактные тесты для /set_date и /list в specs/001-project-murmansk-bot/contracts/events_contract_test.go (должны падать)
- [x] T005 [P] Написать юнит-тесты для парсинга дат и расчёта времени до события в tests/unit/date_test.go
- [x] T006 [P] Написать интеграционный тест для команды /set_date в tests/integration/set_date_test.go
- [x] T007 [P] Написать интеграционный тест для команды /list в tests/integration/list_test.go

## Phase 3.3: Core Implementation
- [x] T008 [P] Реализовать модель Event в internal/models/event.go
- [x] T009 [P] Реализовать модель User в internal/models/user.go
- [x] T010 Реализовать интерфейс Storage (json, расширяемо до PostgreSQL) в internal/storage/storage.go
- [x] T011 Реализовать сервис управления событиями (EventService) в internal/services/event_service.go
- [x] T012 Реализовать сервис управления пользователями (UserService) в internal/services/user_service.go
- [x] T013 Реализовать обработчики команд /set_date, /list, /all, /active, /outdated, /help в cmd/main.go
- [x] T014 Реализовать регистрацию динамических команд в cmd/main.go
- [x] T015 Реализовать логирование событий и ошибок (log/logrus/zap) в cmd/main.go и сервисах

## Phase 3.4: Integration & Polish
- [x] T016 [P] Реализовать интеграцию с Docker (Dockerfile, docker-compose)
- [x] T017 [P] Добавить README и quickstart.md с инструкциями и списком команд
- [ ] T018 [P] Добавить поддержку нескольких чатов (разделение событий по chat_id)
- [ ] T019 [P] Добавить локализацию (структура для будущих языков)
- [ ] T020 [P] Провести рефакторинг и оптимизацию кода
- [ ] T021 [P] Провести тестирование производительности и устойчивости

## Phase 3.5: Refactoring & Enhancements
- [x] T022 Обновить парсинг даты и времени: добавить поддержку часов и минут (формат YYYY-MM-DD HH:MM), по умолчанию 00:00, часовой пояс Europe/Moscow
- [x] T023 Изменить формат даты в моделях и хранилище на YYYY-MM-DD HH:MM
- [x] T024 Обновить команду /list для отображения динамических команд событий
- [x] T025 Обновить все тесты для нового формата даты и времени
- [ ] T026 Обновить обработчики команд для работы с новым форматом

## Parallel Execution Guidance
- Все задачи, отмеченные [P], могут выполняться параллельно, если не зависят от одних и тех же файлов.
- Пример: тесты, модели, интеграция с Docker, документация, локализация, оптимизация.

## Dependency Notes
- T001, T002 — всегда первыми
- Тесты (T004-T007) — до реализации
- Модели (T008-T009) — до сервисов
- Сервисы (T011-T012) — до обработчиков команд
- Логирование, интеграция, документация — после основных функций
- Рефакторинг (T022-T026) — после завершения основных функций

## Task Agent Commands
- Для параллельных задач используйте: `/run-tasks T004 T005 T006 T007`
- Для последовательных: `/run-task T001`, затем `/run-task T002`, затем `/run-task T008`
- Для рефакторинга: `/run-task T022`, затем `/run-task T023`, затем `/run-task T024`

