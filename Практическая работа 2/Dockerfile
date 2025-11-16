FROM golang:latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app main.go

CMD ["./app", "kafka-0:9092", "my_topic3"]
