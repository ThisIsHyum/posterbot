# --- Сборка бинарника ---
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Собираем бинарник с уникальным именем
RUN go build -o posterbot

# --- Финальный образ ---
FROM alpine:latest

RUN apk add --no-cache sqlite ca-certificates

WORKDIR /app

# Копируем бинарник
COPY --from=builder /app/posterbot /app/posterbot

# Делаем исполняемым
RUN chmod +x /app/posterbot

# Точка входа
CMD ["/app/posterbot"]
