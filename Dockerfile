FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app main.go

# Подключение к обоим брокерам
CMD ["./app", "kafka-0:9092,kafka-1:9092", "my_topic4"]
