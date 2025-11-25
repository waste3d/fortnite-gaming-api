# --- 1. Сборка ---
    FROM golang:1.24-alpine AS builder

    WORKDIR /app
    COPY go.mod go.sum ./
    COPY . .
    
    RUN go build -mod=vendor -o /main cmd/app/main.go    
    # --- 2. Финальный образ ---
    FROM alpine:latest
    
    WORKDIR /root/
    RUN apk --no-cache add ca-certificates
    
    COPY --from=builder /app/course-app .
    
    # Порт HTTP
    EXPOSE 8080
    
    CMD ["./course-app"]