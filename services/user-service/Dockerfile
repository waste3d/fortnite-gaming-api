FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# ВАЖНО: Указываем путь к main.go именно User Service
# И называем бинарник user-app
RUN go build -o user-app ./services/user-service/cmd/app/main.go

# --- Финальный образ ---
FROM alpine:latest

WORKDIR /root/

# Копируем бинарник из билдера
COPY --from=builder /app/user-app .
# Копируем конфиг (если нужен, обычно viper читает ENV, но файл тоже можно)
COPY --from=builder /app/services/user-service/config/app.env . 

# Запускаем правильный бинарник
CMD ["./user-app"]