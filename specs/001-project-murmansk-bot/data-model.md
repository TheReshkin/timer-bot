# Data Model: murmansk-bot

## Entities

### Event
- event_id: string (unique)
- name: string (латинские буквы, цифры, underscore)
- date: string (DD.MM.YYYY)
- description: string
- status: string (active, outdated)
- chat_id: int64

#### Validation Rules
- Имя уникально в пределах чата
- Имя: только латинские буквы, цифры, underscore
- Дата: формат DD.MM.YYYY
- Статус вычисляется по дате

### User
- user_id: int64
- chat_id: int64
- events: []Event

## Relationships
- Event принадлежит чату (chat_id)
- User может создавать события в чате

## State Transitions
- При добавлении события → status = active
- Если дата прошла → status = outdated

## Storage
- Основной вариант: events.json
- Интерфейс Storage для расширения (PostgreSQL)
