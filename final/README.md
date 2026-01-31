chmod +x generate-certs.sh
./generate-certs.sh

docker compose up -d

В certs/kafka-1 добавила admin.properties:
# Протокол безопасности
security.protocol=SASL_SSL

# Механизм аутентификации
sasl.mechanism=PLAIN

# Данные для входа admin
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required \
  username="admin" \
  password="admin-secret";

# SSL настройки
ssl.truststore.location=/etc/kafka/secrets/kafka.truststore.jks
ssl.truststore.password=password






docker exec -it final-kafka-1-1 bash

kafka-topics \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --create \
  --if-not-exists \
  --topic products-topic \
  --partitions 3 \
  --replication-factor 3 \
  --config min.insync.replicas=2




kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:producer \
  --operation Write \
  --topic products-topic


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
  --list

Вывод:

Current ACLs for resource `ResourcePattern(resourceType=TOPIC, name=cart-topic, patternType=LITERAL)`: 
        (principal=User:producer, host=*, operation=WRITE, permissionType=ALLOW) 

Current ACLs for resource `ResourcePattern(resourceType=TOPIC, name=products-topic, patternType=LITERAL)`: 
        (principal=User:consumer, host=*, operation=READ, permissionType=ALLOW)
        (principal=User:producer, host=*, operation=WRITE, permissionType=ALLOW) 


# Запустила 
docker compose up --build producer

# Тест - получила сообщение из products.json
kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic \
  --from-beginning
















kafka-topics \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --create \
  --if-not-exists \
  --topic cart-topic \
  --partitions 3 \
  --replication-factor 3 \
  --config min.insync.replicas=2

kafka-console-producer \
  --bootstrap-server kafka-1:1092 \
  --producer.config /etc/kafka/secrets/admin.properties \
  --producer-property acks=all \
  --topic products-topic

  kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:customer-producer \
  --operation Write \
  --topic cart-topic


  # Тест 1 - записала сообщение hello в topic-1
kafka-console-producer \
  --bootstrap-server kafka-1:1092 \
  --producer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic

  # Тест 2 - записала сообщение hello в topic-2
kafka-console-producer \
  --bootstrap-server kafka-1:1092 \
  --producer.config /etc/kafka/secrets/admin.properties \
  --topic cart-topic


kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic cart-topic \
  --from-beginning