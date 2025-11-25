FROM golang:1.22-alpine as builder

WORKDIR /app

# УДАЛЯЕМ или комментируем строку RUN go mod download
# УДАЛЯЕМ отдельное копирование go.mod (не обязательно, но лишнее)

# 1. Копируем ВСЕ файлы сразу (включая созданную вами папку vendor)
COPY . .

# 2. Собираем проект с флагом -mod=vendor
# Этот флаг говорит Go: "не лезь в интернет, смотри в папку /vendor"
RUN go build -mod=vendor -o /main cmd/app/main.go

# ... (далее без изменений) ...
FROM alpine:3.18
COPY --from=builder /main /bin/main
CMD ["/bin/main"]