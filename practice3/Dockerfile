FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app main.go

# Подключение к обоим брокерам
CMD ["/app/app", "localhost:9094,localhost:9095"]

