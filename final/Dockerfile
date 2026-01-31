FROM golang:latest


WORKDIR /app
COPY shop/main.go .
COPY shop/products.json .

RUN go mod init shop-producer
RUN go get github.com/IBM/sarama

CMD ["go", "run", "main.go"]
