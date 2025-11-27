FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем всё содержимое текущей папки (включая созданную папку vendor)
COPY . .

# Собираем с флагом -mod=vendor
# Это заставит Go брать зависимости из локальной папки vendor, а не качать их
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o payment-service ./cmd/app/main.go

# --- 2. Финальный образ ---
FROM alpine:latest

WORKDIR /root/
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/payment-service .

EXPOSE 50054

CMD ["./payment-service"]