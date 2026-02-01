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


docker exec -it final-kafka-1-1 kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add --allow-principal User:admin --operation Read --topic products-topic




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

docker exec -it final-kafka-1-1 bash

# Тест - получила сообщение из products.json
kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic \
  --from-beginning




kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic \
  --from-beginning \
  --timeout-ms 10000 | jq 'select(.name=="Умные часы XYZ")'









curl -X POST -H "Content-Type: application/json" \
  --data @connector.json \
  http://localhost:8083/connectors


Ответ должен вернуть JSON с "state":"RUNNING" для коннектора и задач (tasks).
4️⃣ Проверка
Статус коннектора:
curl -X GET http://localhost:8083/connectors/products-elasticsearch-sink/status
Список индексов в Elasticsearch:
curl -X GET "http://localhost:9200/_cat/indices?v"
Если все OK, должен появиться индекс products и документы из Kafka должны попадать в него.


 Вывод:
 @aruzhansadakbayeva ➜ curl -X GET http://localhost:8083/connectors/products-elasticsearch-sink/statusectors/products-elasticsearch-sink/status
{"name":"products-elasticsearch-sink","connector":{"state":"RUNNING","worker_id":"kafka-connect:8083"},"tasks":[{"id":0,"state":"RUNNING","worker_id":"kafka-connect:8083"}],curl -X GET "http://localhost:9200/_cat/indices?v"ache-kafka/final (final) $ curl -X GET "http://localhost:9200/_cat/indices?v"
health status index          uuid                   pri rep docs.count docs.deleted store.size pri.store.size dataset.size

Команда для поиска товара по имени:
curl -X GET "http://localhost:9200/products-topic/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match": {
      "name": "Умные часы XYZ"
    }
  }
}
'
 еще:
@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ curl -X GET "http://localhost:9200/products-topic/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match": {
      "name": "Умные часы"
    }
  }
}
'
вывод:
{
  "took" : 3,
  "timed_out" : false,
  "_shards" : {
    "total" : 1,
    "successful" : 1,
    "skipped" : 0,
    "failed" : 0
  },
  "hits" : {
    "total" : {
      "value" : 1,
      "relation" : "eq"
    },
    "max_score" : 0.5753642,
    "hits" : [
      {
        "_index" : "products-topic",
        "_id" : "products-topic+0+0",
        "_score" : 0.5753642,
        "_source" : {
          "store_id" : "store_001",
          "images" : [
            {
              "alt" : "Умные часы XYZ - вид спереди",
              "url" : "https://example.com/images/product1.jpg"
            },
            {
              "alt" : "Умные часы XYZ - вид сбоку",
              "url" : "https://example.com/images/product1_side.jpg"
            }
          ],
          "created_at" : "2023-10-01T12:00:00Z",
          "description" : "Умные часы с функцией мониторинга здоровья, GPS и уведомлениями.",
          "index" : "products",
          "specifications" : {
            "water_resistance" : "IP68",
            "battery_life" : "24 hours",
            "weight" : "50g",
            "dimensions" : "42mm x 36mm x 10mm"
          },
          "tags" : [
            "умные часы",
            "гаджеты",
            "технологии"
          ],
          "updated_at" : "2023-10-10T15:30:00Z",
          "price" : {
            "amount" : 4999.99,
            "currency" : "RUB"
          },
          "product_id" : "12345",
          "name" : "Умные часы XYZ",
          "category" : "Электроника",
          "sku" : "XYZ-12345",
          "stock" : {
            "reserved" : 20,
            "available" : 150
          },
          "brand" : "XYZ"
        }
      }
    ]
  }
}
@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ 

На всякий случай:
curl -X DELETE http://localhost:8083/connectors/products-elasticsearch-sink

перезапуск коннектора чтобы он прочел топик заново:
curl -X POST http://localhost:8083/connectors/products-elasticsearch-sink/restart

если в контейнере нет плагина elasticsearch: 
confluent-hub install confluentinc/kafka-connect-elasticsearch:latest --no-prompt
(из контейнера)

docker restart <container>
docker logs <container>




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






 docker exec -it kafka-destination bash

  kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:admin \
  --operation Read \
  --operation Write \
  --topic products-topic


  kafka-metadata-quorum \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  describe --status







docker exec -it kafka-destination bash


kafka-topics \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --list



kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add --allow-principal User:admin --operation Read --topic products-topic



kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:consumer \
  --operation Read \
  --group '*'



kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --list


kafka-console-consumer \
  --bootstrap-server kafka-destination:1096 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic \
  --from-beginning