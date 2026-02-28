# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`  
**Created**: [DATE]  
**Status**: Draft  

# Feature Specification: murmansk-bot: Telegram bot for tracking dates and countdowns

**Feature Branch**: `001-project-murmansk-bot`
**Created**: 2025-09-06
**Status**: Draft
**Input**: User description: "Telegram bot written in Go for tracking dates and countdowns. Requirements: interaction in private and group chats, commands start with '/', event creation via /set_date <DD.MM.YYYY> <event_name>, dynamic commands for events, event info, system commands (/help, /set_date, /list, /all, /active, /outdated), storage in-memory (optionally file/DB), date format DD.MM.YYYY, error handling for invalid date/name/duplicate."

## Execution Flow (main)
```
1. Parse user description from Input
2. Extract key concepts: actors (users in private/group chats), actions (create event, view info, list events), data (event name, date, time left), constraints (unique event names, allowed characters, error handling)
3. For each unclear aspect:
   → Events should persist after bot restart in .json file, in plans switch to postgresql 
   → Events should be visible to all group members.
4. Fill User Scenarios & Testing section
5. Generate Functional Requirements
6. Identify Key Entities (Event, User)
7. Run Review Checklist
8. Return: SUCCESS (spec ready for planning)
```

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story
Пользователь (в личном или групповом чате) добавляет событие через команду `/set_date 12.12.2025 new_year`. После этого появляется команда `/new_year`, которая показывает описание события и время до даты. Пользователь может просмотреть список событий, получить справку по командам, увидеть активные и устаревшие события.

### Acceptance Scenarios
1. **Given** пользователь в чате, **When** отправляет `/set_date 12.12.2025 new_year`, **Then** появляется команда `/new_year`, которая возвращает описание и время до даты.
2. **Given** несколько событий, **When** пользователь отправляет `/list`, **Then** бот возвращает краткий список событий.
3. **Given** событие с недопустимым именем, **When** пользователь пытается создать событие, **Then** бот возвращает ошибку.
4. **Given** событие с уже существующим именем, **When** пользователь пытается создать событие, **Then** бот возвращает ошибку о дублировании.
5. **Given** событие с некорректной датой, **When** пользователь пытается создать событие, **Then** бот возвращает ошибку.

### Edge Cases
- Что происходит, если пользователь пытается создать событие с именем, содержащим запрещённые символы?
- Как бот реагирует на команду с датой в прошлом?
- Как бот ведёт себя при большом количестве событий?
- Как бот реагирует на команду в групповом чате от разных пользователей?

## Requirements *(mandatory)*

### Functional Requirements
- **FR-001**: Система ДОЛЖНА позволять создавать события через команду `/set_date <DD.MM.YYYY> <event_name>`
- **FR-002**: Система ДОЛЖНА создавать динамическую команду `/event_name` для каждого события
- **FR-003**: Система ДОЛЖНА возвращать информацию о событии (имя, дата, время до события) по динамической команде
- **FR-004**: Система ДОЛЖНА поддерживать команды `/help`, `/list`, `/all`, `/active`, `/outdated`
- **FR-005**: Система ДОЛЖНА хранить события в памяти (опционально — в файле или БД)
- **FR-006**: Система ДОЛЖНА проверять уникальность имени события
- **FR-007**: Система ДОЛЖНА проверять формат даты (DD.MM.YYYY)
- **FR-008**: Система ДОЛЖНА возвращать ошибку при недопустимом имени события
- **FR-009**: Система ДОЛЖНА возвращать ошибку при некорректной дате
- **FR-010**: Система ДОЛЖНА возвращать ошибку при попытке создать событие с уже существующим именем
- **FR-011**: Система ДОЛЖНА поддерживать работу в личных и групповых чатах
- **FR-012**: Система ДОЛЖНА поддерживать только латинские буквы, цифры и нижнее подчёркивание в именах событий
- **FR-013**: Система ДОЛЖНА запрещать пробелы и спецсимволы в именах событий
- **FR-014**: Система ДОЛЖНА возвращать справку по командам через `/help`
- **FR-015**: Система ДОЛЖНА различать активные и устаревшие события
- **FR-016**: Система ДОЛЖНА корректно обрабатывать ошибки
- **FR-017**: Должны события быть видимы всем участникам группы.
- **FR-018**: Нужно сохранять события между перезапусками бота, для этого уже используется .json файл. Далее планируется переход на postgresql

### Key Entities
- **Event**: имя события, дата, описание, время до события, статус (активное/устаревшее)
- **User**: идентификатор пользователя, список созданных событий

---

## Review & Acceptance Checklist

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [ ] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---
- [ ] No [NEEDS CLARIFICATION] markers remain
- [ ] Requirements are testable and unambiguous  
- [ ] Success criteria are measurable
- [ ] Scope is clearly bounded
- [ ] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [ ] User description parsed
- [ ] Key concepts extracted
- [ ] Ambiguities marked
- [ ] User scenarios defined
- [ ] Requirements generated
- [ ] Entities identified
- [ ] Review checklist passed

---
