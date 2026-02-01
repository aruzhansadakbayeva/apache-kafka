FROM golang:latest


WORKDIR /app
COPY shop/main.go .
COPY shop/products.json .
COPY customer/main.go .
COPY customer/cart_products.json .

RUN go mod init producer
RUN go get github.com/IBM/sarama

CMD ["go", "run", "main.go"]
