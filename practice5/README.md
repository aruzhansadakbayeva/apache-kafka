docker compose up -d

curl.exe -X PUT -H "Content-Type: application/json" -d @connector-config.json http://localhost:8083/connectors/pg-connector/config

PS C:\Users\user\Desktop\infra_template> docker exec -it postgres psql -h 127.0.0.1 -U postgres-user -d customers
psql (16.4 (Debian 16.4-1.pgdg110+2))
Type "help" for help.

customers=# CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    email VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
);  order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
CREATE TABLE
CREATE TABLE


docker run -it --rm --name debezium-ui --network custom_network -p 8080:8080 -e KAFKA_CONNECT_URIS=http://kafka-connect:8083 quay.io/debezium/debezium-ui



-- Добавление пользователей
INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com');
INSERT INTO users (name, email) VALUES ('Jane Smith', 'jane@example.com');
INSERT INTO users (name, email) VALUES ('Alice Johnson', 'alice@example.com');
INSERT INTO users (name, email) VALUES ('Bob Brown', 'bob@example.com');


-- Добавление заказов
INSERT INTO orders (user_id, product_name, quantity) VALUES (1, 'Product A', 2);
INSERT INTO orders (user_id, product_name, quantity) VALUES (1, 'Product B', 1);
INSERT INTO orders (user_id, product_name, quantity) VALUES (2, 'Product C', 5);
INSERT INTO orders (user_id, product_name, quantity) VALUES (3, 'Product D', 3);
INSERT INTO orders (user_id, product_name, quantity) VALUES (4, 'Product E', 4);

В http://localhost:8085/ui/clusters/kraft/all-topics - созданные топики customers.public.users и customers.public.orders, и в них отправленные сообщения
