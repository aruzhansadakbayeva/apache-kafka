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

kafka-topics \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --create \
  --if-not-exists \
  --topic products-filtered-topic \
  --partitions 3 \
  --replication-factor 3 \
  --config min.insync.replicas=2


kafka-topics \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --create \
  --if-not-exists \
  --topic cart-topic \
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
  --allow-principal User:producer \
  --operation Write \
  --topic products-filtered-topic


kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:producer \
  --operation Write \
  --topic cart-topic


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
  --add --allow-principal User:admin --operation Read --topic products-topic

kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add --allow-principal User:admin --operation Read --topic cart-topic


kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add --allow-principal User:admin --operation Read --topic products-filtered-topic


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
docker compose up --build 

docker exec -it final-kafka-1-1 bash
# Тест - получила сообщение из products.json
kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic \
  --from-beginning


@aruzhansadakbayeva ➜ /workspaces/apache-kafka (final) $ docker inspect products-filter --format '{{json .Mounts}}' | jq
[
  {
    "Type": "bind",
    "Source": "/workspaces/apache-kafka/final/filter/data",
    "Destination": "/app/data",
    "Mode": "rw",
    "RW": true,
    "Propagation": "rprivate"
  }
]
@aruzhansadakbayeva ➜ /workspaces/apache-kafka (final) $ docker exec -it products-filter sh
/app # /app/filter add --id 12345 --reason "banned by policy"
added banned item: id="12345" name=""
/app # /app/filter add --id 12344 --reason "banned by policy"
added banned item: id="12344" name=""
/app # /app/filter list
1) id="12345" name="" reason="banned by policy" added_at=2026-02-02T06:49:01Z
2) id="12344" name="" reason="banned by policy" added_at=2026-02-02T06:49:29Z

удалить из запрещенных: /app/filter remove --id 12345

Шаг 1. Добавим запрещённый товар через CLI
Например, запретим product_id=999
docker exec -it products-filter /app/filter add --id 999 --reason "test ban"
Проверим:
docker exec -it products-filter /app/filter list
Ты увидишь его в списке.
Шаг 2. Открой consumer на выходной топик
Это самое важное — тут будет видно итог фильтрации.
docker exec -it final-kafka-1-1 bash -lc '
kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-filtered-topic \
  --from-beginning
'
Оставь этот терминал открытым.
Шаг 3. Отправим 2 товара во входной топик
В другом терминале:
docker exec -i final-kafka-1-1 bash -lc '
kafka-console-producer \
  --bootstrap-server kafka-1:1092 \
  --producer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic
'
И вставь две строки:
❌ Запрещённый (НЕ должен пройти)
{"product_id":"999","name":"Запрещённые часы","description":"bad"}
✅ Разрешённый (должен пройти)
{"product_id":"100","name":"Разрешённые часы","description":"good"}
Шаг 4. Что должно произойти
В логах products-filter (docker logs -f products-filter) ты увидишь:
BLOCKED: product_id="999" name="Запрещённые часы" (banned by product_id)
и НЕ будет записи про отправку.
В consumer на products-filtered-topic появится ТОЛЬКО:
{"product_id":"100","name":"Разрешённые часы","description":"good"}

Фильтрация успешно обрабатывает сообщения из products-topic, которые в свою очередь читаются из products.json



kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-filtered-topic \
  --from-beginning


kafka-console-consumer \
  --bootstrap-server kafka-1:1092 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic cart-topic \
  --from-beginning











docker exec -it final-kafka-connect-1 bash
confluent-hub install confluentinc/kafka-connect-elasticsearch:latest --no-prompt

@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker exec -it final-kafka-connect-1 bash
[appuser@kafka-connect ~]$ confluent-hub install --no-prompt confluentinc/kafka-connect-elasticsearch:latest
Running in a "--no-prompt" mode 

Implicit acceptance of the license below:  
Confluent Community License 
http://www.confluent.io/confluent-community-license 
Downloading component Kafka Connect Elasticsearch 15.1.1, provided by Confluent, Inc. from Confluent Hub and installing into /usr/share/confluent-hub-components 
Adding installation directory to plugin path in the following files: 
  /etc/kafka/connect-distributed.properties 
  /etc/kafka/connect-standalone.properties 
  /etc/schema-registry/connect-avro-distributed.properties 
  /etc/schema-registry/connect-avro-standalone.properties 
  /etc/kafka-connect/kafka-connect.properties 
 
Completed 
[appuser@kafka-connect ~]$ 
[appuser@kafka-connect ~]$ curl http://localhost:8083/connector-plugins
[{"class":"org.apache.kafka.connect.mirror.MirrorCheckpointConnector","type":"source","version":"7.4.4-ccs"},{"class":"org.apache.kafka.connect.mirror.MirrorHeartbeatConnector","type":"source","version":"7.4.4-ccs"},{"class":"org.apache.kafka.connect.mirror.MirrorSourceConnector","type":"source","version":"7.4.4-ccs"}][appuser@kafka-connect ~]$ 


docker restart final-kafka-connect-1

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





docker exec -it final-kafka-1-1 bash

kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:consumer \
  --operation Read \
  --topic products-topic \
  --group mirror-maker-group

kafka-acls \
  --bootstrap-server kafka-1:1092 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:consumer \
  --operation Read \
  --topic cart-topic \
  --group mirror-maker-group


docker exec -it kafka-destination bash


kafka-topics \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --list




kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:mirrormaker \
  --operation Write \
  --operation Create \
  --topic products-topic

kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add \
  --allow-principal User:mirrormaker \
  --operation Write \
  --operation Create \
  --topic cart-topic


kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add --allow-principal User:admin --operation Read --topic products-topic


kafka-acls \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --add --allow-principal User:admin --operation Read --topic cart-topic



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



docker restart mirror-maker

docker exec -it kafka-destination bash

kafka-console-consumer \
  --bootstrap-server kafka-destination:1096 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic products-topic \
  --from-beginning


kafka-console-consumer \
  --bootstrap-server kafka-destination:1096 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic cart-topic \
  --from-beginning


kafka-topics \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --create --if-not-exists \
  --topic recommendations-topic \
  --partitions 3 \
  --replication-factor 1


  kafka-console-consumer \
  --bootstrap-server kafka-destination:1096 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic recommendations-topic \
  --from-beginning

 





  в certs/kafka-destination добавила admin.properties

и kafka_server_jaas.conf:
KafkaServer {
  org.apache.kafka.common.security.plain.PlainLoginModule required
  username="admin"
  password="admin-secret"
  user_admin="admin-secret"
  user_mirrormaker="mm-secret";
};





go mod init analytics-pipeline
go get github.com/segmentio/kafka-go
go get github.com/colinmarc/hdfs/v2

@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker inspect spark-master --format '{{.Config.Image}}'
bde2020/spark-master:3.3.0-hadoop3.3
@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker exec -it spark-master bash -lc 'find / -name spark-submit 2>/dev/null | head -20'
/spark/bin/spark-submit

docker exec -it spark-master bash -lc \
  '/spark/bin/spark-submit --master spark://spark-master:7077 /opt/analytics/reco_job.py'


 kafka-console-consumer \
  --bootstrap-server kafka-destination:1096 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic recommendations-topic \
  --from-beginning

  docker exec -it namenode bash

hdfs dfs -mkdir -p /data/cart
hdfs dfs -mkdir -p /data/reco/dt=latest
hdfs dfs -ls /data


@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker exec -it namenode bash
root@db631664309a:/# hdfs dfs -mkdir -p /data/cart
root@db631664309a:/# hdfs dfs -ls /data
Found 2 items
drwxr-xr-x   - root supergroup          0 2026-02-01 17:36 /data/cart
drwxr-xr-x   - root supergroup          0 2026-02-01 17:28 /data/reco
root@db631664309a:/# hdfs dfs -ls /data/cart

Found 1 items
drwxr-xr-x   - root supergroup          0 2026-02-01 17:36 /data/cart/dt=2026-02-01
root@db631664309a:/# 


Проверить, что в HDFS появились файлы и в них есть данные
Внутри namenode:
hdfs dfs -ls -h /data/cart/dt=2026-02-01
Если там будут файлы (часто .json, .jsonl, .parquet, .csv или part-*) — супер.
Посмотреть первые строки (выбери файл из списка):
hdfs dfs -cat /data/cart/dt=2026-02-01/* | head -50

root@db631664309a:/# hdfs dfs -cat /data/cart/dt=2026-02-01/* | head -50

2026-02-01 17:44:38,857 INFO sasl.SaslDataTransferClient: SASL encryption trust check: localHostTrusted = false, remoteHostTrusted = false
{"brand":"XYZ","category":"Электроника","created_at":"2023-10-01T12:00:00Z","description":"Умные часы с функцией мониторинга здоровья, GPS и уведомлениями.","images":[{"alt":"Умные часы XYZ - вид спереди","url":"https://example.com/images/product1.jpg"},{"alt":"Умные часы XYZ - вид сбоку","url":"https://example.com/images/product1_side.jpg"}],"index":"products","name":"Умные часы XYZ","price":{"amount":4999.99,"currency":"RUB"},"product_id":"12345","sku":"XYZ-12345","specifications":{"battery_life":"24 hours","dimensions":"42mm x 36mm x 10mm","water_resistance":"IP68","weight":"50g"},"stock":{"available":150,"reserved":20},"store_id":"store_001","tags":["умные часы","гаджеты","технологии"],"updated_at":"2023-10-10T15:30:00Z","user_id":"1"}



Запуск reco_job.py
docker exec -it spark-master bash -lc '
/spark/bin/spark-submit \
  --master spark://spark-master:7077 \
  --name cart-analytics \
  --deploy-mode client \
  --executor-cores 1 \
  --executor-memory 512m \
  --driver-memory 512m \
  /opt/analytics/reco_job.py
'


Проверить, что reco_job записал результат в HDFS

@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker exec -it namenode bash
root@db631664309a:/# hdfs dfs -ls -R /data/reco
eco/dt=latest || true
hdfs dfs -cat /data/reco/dt=latest/part-* 2>/dev/null | head -50 || true
drwxr-xr-x   - root supergroup          0 2026-02-01 18:17 /data/reco/dt=latest
-rw-r--r--   3 root supergroup          0 2026-02-01 18:17 /data/reco/dt=latest/_SUCCESS
-rw-r--r--   3 root supergroup         52 2026-02-01 18:17 /data/reco/dt=latest/part-00000-3c62105c-b6b8-46b7-9e31-cf339e1ef32c-c000.json
root@db631664309a:/# hdfs dfs -ls -h /data/reco/dt=latest || true
Found 2 items
-rw-r--r--   3 root supergroup          0 2026-02-01 18:17 /data/reco/dt=latest/_SUCCESS
-rw-r--r--   3 root supergroup         52 2026-02-01 18:17 /data/reco/dt=latest/part-00000-3c62105c-b6b8-46b7-9e31-cf339e1ef32c-c000.json
root@db631664309a:/# hdfs dfs -cat /data/reco/dt=latest/part-* 2>/dev/null | head -50 || true
{"user_id":"1","product_id":"12345","cnt":1,"rn":1}



Проверить, что рекомендации улетели в Kafka (recommendations-topic)

@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker exec -it kafka-destination bash -lc '
kafka-topics \
  --bootstrap-server kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --describe --topic recommendations-topic
'
Topic: recommendations-topic    TopicId: FcZcecWGRCSTKYVyJlH1uA PartitionCount: 3       ReplicationFactor: 1    Configs: 
        Topic: recommendations-topic    Partition: 0    Leader: 1       Replicas: 1     Isr: 1
        Topic: recommendations-topic    Partition: 1    Leader: 1       Replicas: 1     Isr: 1
        Topic: recommendations-topic    Partition: 2    Leader: 1       Replicas: 1     Isr: 1

@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker exec -it kafka-destination bash -lc '
kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list kafka-destination:1096 \
  --command-config /etc/kafka/secrets/admin.properties \
  --topic recommendations-topic --time -1
'
recommendations-topic:0:0
recommendations-topic:1:0
recommendations-topic:2:0


docker exec -it kafka-destination bash -lc '
kafka-console-consumer \
  --bootstrap-server kafka-destination:1096 \
  --consumer.config /etc/kafka/secrets/admin.properties \
  --topic recommendations-topic \
  --from-beginning \
  --timeout-ms 5000
'




docker compose build



docker compose build products-filter


Запуск и проверка мониторинга:
docker compose up -d --build
Проверка, что метрики отдаются:
http://localhost:9090 — Prometheus
http://localhost:3000 — Grafana (admin/admin)
http://localhost:7071/metrics — kafka-1 метрики
В Prometheus → Status → Targets - UP для всех брокеров.

В Grafana:
Add data source → Prometheus → URL: http://prometheus:9090
Создала дашборд и добавила панели на метрики типа:
jvm_memory_heap_used


@aruzhansadakbayeva ➜ /workspaces/apache-kafka/final (final) $ docker compose ps alertmanager
docker compose logs -f alertmanager
NAME           IMAGE                      COMMAND                  SERVICE        CREATED          STATUS          PORTS
alertmanager   prom/alertmanager:latest   "/bin/alertmanager -…"   alertmanager   26 minutes ago   Up 26 minutes   0.0.0.0:9093->9093/tcp, [::]:9093->9093/tcp
alertmanager  | time=2026-02-03T08:38:15.888Z level=INFO source=main.go:191 msg="Starting Alertmanager" version="(version=0.31.0, branch=HEAD, revision=0ae07a09fbb26a7738c867306f32b5f42583a7d2)"
alertmanager  | time=2026-02-03T08:38:15.889Z level=INFO source=main.go:194 msg="Build context" build_context="(go=go1.25.6, platform=linux/amd64, user=root@47d5a7c91e78, date=20260202-13:00:37, tags=netgo)"
alertmanager  | time=2026-02-03T08:38:15.890Z level=INFO source=cluster.go:192 msg="setting advertise address explicitly" component=cluster addr=172.18.0.8 port=9094
alertmanager  | time=2026-02-03T08:38:15.892Z level=INFO source=cluster.go:682 msg="Waiting for gossip to settle..." component=cluster interval=2s
alertmanager  | time=2026-02-03T08:38:15.923Z level=INFO source=coordinator.go:111 msg="Loading configuration file" component=configuration file=/etc/alertmanager/alertmanager.yml
alertmanager  | time=2026-02-03T08:38:15.929Z level=INFO source=coordinator.go:124 msg="Completed loading of configuration file" component=configuration file=/etc/alertmanager/alertmanager.yml
alertmanager  | time=2026-02-03T08:38:15.935Z level=INFO source=tls_config.go:354 msg="Listening on" address=[::]:9093

Для теста остановила один брокер: docker compose stop kafka-2
Через ~2 минуты (как в rules.yml for: 2m):
В Prometheus алерт стал FIRING
В Alertmanager он появится

Вернула брокер: docker compose start kafka-2
Алерт ушел в resolved

В телеграм-бот пришли соответствующие сообщения