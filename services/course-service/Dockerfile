# --- 1. Сборка ---
    FROM golang:1.24-alpine AS builder

    WORKDIR /app
    
    COPY go.mod go.sum ./
    RUN go mod download
    
    COPY . .
    
    RUN CGO_ENABLED=0 GOOS=linux go build -o course-app ./cmd/app/main.go
    
    # --- 2. Финальный образ ---
    FROM alpine:latest
    
    WORKDIR /root/
    RUN apk --no-cache add ca-certificates
    
    COPY --from=builder /app/course-app .
    
    # Порт HTTP
    EXPOSE 8080
    
    CMD ["./course-app"]