chmod +x generate-certs.sh
./generate-certs.sh

docker compose up -d

В certs/kafka-1 добавила admin.properties, producer.properties и consumer.properties

docker exec -it ksecurity-kafka-1-1 bash


  kafka-topics \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --create \
  --if-not-exists \
  --topic topic-1 \
  --partitions 3 \
  --replication-factor 3


  kafka-topics \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --create \
  --if-not-exists \
  --topic topic-2 \
  --partitions 3 \
  --replication-factor 3


kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:producer \
  --operation Write \
  --topic topic-1


kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:consumer \
  --operation Read \
  --group '*'



kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:producer \
  --operation Write \
  --topic topic-2


kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --list

Вывод:

Current ACLs for resource `ResourcePattern(resourceType=TOPIC, name=topic-2, patternType=LITERAL)`: 
        (principal=User:producer, host=*, operation=WRITE, permissionType=ALLOW) 

Current ACLs for resource `ResourcePattern(resourceType=TOPIC, name=topic-1, patternType=LITERAL)`: 
        (principal=User:consumer, host=*, operation=READ, permissionType=ALLOW)
        (principal=User:producer, host=*, operation=WRITE, permissionType=ALLOW) 

# Тест 1 - записала сообщение hello в topic-1
kafka-console-producer \
  --bootstrap-server kafka-1:1092 \
  --producer.config /etc/kafka/secrets/producer.properties \
  --topic topic-1

# получила сообщение hello
kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/consumer.properties \
  --topic topic-1 \
  --from-beginning

# Тест 2 - записала сообщение hello в topic-2
kafka-console-producer \
  --bootstrap-server kafka-1:1092 \
  --producer.config /etc/kafka/secrets/producer.properties \
  --topic topic-2

# на чтение прав нет
kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/consumer.properties \
  --topic topic-2 \
  --from-beginning
