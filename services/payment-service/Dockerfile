FROM golang:1.24-alpine AS builder

WORKDIR /app

# 1. Сначала копируем зависимость (user-service), так как на нее ссылается replace
# Мы кладем её в /app/services/user-service, чтобы относительный путь ../user-service сработал
COPY services/user-service /app/services/user-service

# 2. Копируем сам сервис (payment-service)
COPY services/payment-service /app/services/payment-service

# 3. Переходим в рабочую директорию сервиса
WORKDIR /app/services/payment-service

# 4. Теперь go mod download сработает, так как папка ../user-service существует внутри контейнера
RUN go mod download

# 5. Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./cmd/app/main.go

# --- 2. Финальный образ ---
FROM alpine:latest

WORKDIR /root/
RUN apk --no-cache add ca-certificates

# Копируем бинарник из этапа сборки
COPY --from=builder /app/services/payment-service/payment-service .

# Порт HTTP (если используется grpc, порт может быть другим, но EXPOSE чисто информативен)
EXPOSE 50054

CMD ["./payment-service"]