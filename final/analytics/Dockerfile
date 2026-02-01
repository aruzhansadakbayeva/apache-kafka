FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ingestor ./cmd/ingestor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o publisher ./cmd/publisher

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/ingestor /app/ingestor
COPY --from=builder /app/publisher /app/publisher
