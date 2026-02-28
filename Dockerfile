# Этап сборки
FROM golang:1.24-alpine AS builder

# Установка зависимостей
RUN apk add --no-cache git

# Создание рабочей директории
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Компиляция
RUN go build -o murmansk-bot ./cmd

# Финальный образ (без компилятора)
FROM alpine:latest

# Устанавливаем необходимые библиотеки
RUN apk --no-cache add ca-certificates tzdata

# Устанавливаем часовой пояс (опционально)
ENV TZ=Europe/Moscow

# Рабочая директория
WORKDIR /app

# Копируем бинарник из стадии сборки
COPY --from=builder /app/murmansk-bot .

# Точка входа
ENTRYPOINT ["./murmansk-bot"]
