# apache-kafka

Для запуска всего приложения в двух экземплярах, создала Dockerfile и прописала конфигурации в docker-compose.yaml

Понадобились установки:
go get github.com/confluentinc/confluent-kafka-go/schemaregistry
go get github.com/confluentinc/confluent-kafka-go/schemaregistry/serde
go get github.com/confluentinc/confluent-kafka-go/schemaregistry/serde/protobuf

Установки и команды для работы с Proto (такие же структуры Order и Item как в примерах курса, см.order.proto и сгенерированный orderpb/order.pb.go - с отправкой сообщения и сериализацией все получилось, а вот при корректной десериализации возникли проблемы, описала ошибку в функциях консьюмеров в комментариях файла main.go):
go get google.golang.org/protobuf
go get google.golang.org/protobuf/proto
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go mod tidy

export PATH="$PATH:$(go env GOPATH)/bin"

which protoc-gen-go
which protoc-gen-go-grpc

mkdir -p orderpb
protoc --go_out=./orderpb --go_opt=paths=source_relative order.proto


Команда для запуска приложения:
docker compose up --build