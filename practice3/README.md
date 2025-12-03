# apache-kafka
go mod tidy
go mod init
docker compose up -d

docker exec -it practice3-kafka-0-1 /opt/bitnami/kafka/bin/kafka-topics.sh --create --bootstrap-server localhost:9094 --replication-factor 1 --partitions 1 --topic messages

docker exec -it practice3-kafka-0-1 /opt/bitnami/kafka/bin/kafka-topics.sh --create --bootstrap-server localhost:9094 --replication-factor 1 --partitions 1 --topic filtered_messages

docker exec -it practice3-kafka-0-1 /opt/bitnami/kafka/bin/kafka-topics.sh --create --bootstrap-server localhost:9094 --replication-factor 1 --partitions 1 --topic blocked_users



docker build -t myapp .
docker run -it --rm myapp ls -l /app

docker compose build app
docker compose up -d





go build -o chat-filter.exe main.go
.\chat-filter.exe

docker exec -it practice3-kafka-0-1 /opt/bitnami/kafka/bin/kafka-topics.sh --list --bootstrap-server localhost:9094

