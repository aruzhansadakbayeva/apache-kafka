FROM golang:latest


WORKDIR /app
COPY ./main.go .
COPY ./cart_products.json .

RUN go mod init producer
RUN go get github.com/IBM/sarama

CMD ["go", "run", "main.go"]
