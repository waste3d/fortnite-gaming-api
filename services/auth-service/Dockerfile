# --- 1. Сборка (Build Stage) ---
    FROM golang:1.22-alpine AS builder

    WORKDIR /app
    
    # Сначала копируем зависимости (для кэширования Docker слоев)
    COPY go.mod go.sum ./
    RUN go mod download
    
    # Копируем исходный код
    COPY . .
    
    # Компилируем в статический бинарник
    # CGO_ENABLED=0 - важно для работы в alpine
    RUN CGO_ENABLED=0 GOOS=linux go build -o auth-app ./cmd/app/main.go
    
    # --- 2. Финальный образ (Run Stage) ---
    FROM alpine:latest
    
    WORKDIR /root/
    
    # Сертификаты нужны, если сервис делает HTTPS запросы
    RUN apk --no-cache add ca-certificates
    
    # Копируем только бинарник из первого этапа
    COPY --from=builder /app/auth-app .
    
    # Порт gRPC
    EXPOSE 50051
    
    CMD ["./auth-app"]