package main

import pb "github.com/aruzhansadakbayeva/apache-kafka/practice2"

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/schemaregistry/serde/protobuf"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

)

// ---------------- ПРОДЮСЕР ----------------
func runProducer(bootstrap string, topic string) {

	// Конфигурация для Schema Registry
	srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig("http://localhost:8081"))
	if err != nil {
		log.Fatalf("Ошибка при создании клиента Schema Registry: %v", err)
	}

	// Создание сериализатора protobuf
	serializer, err := protobuf.NewSerializer(srClient, serde.ValueSerde, protobuf.NewSerializerConfig())
	if err != nil {
		log.Fatalf("Ошибка при создании сериализатора: %v", err)
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": bootstrap})
	if err != nil {
		log.Fatalf("Producer creation failed: %s", err)
	}
	defer p.Close()

	deliveryChan := make(chan kafka.Event)

	go func() {
		for e := range deliveryChan {
			m := e.(*kafka.Message)
			if m.TopicPartition.Error != nil {
				fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
			} else {
				fmt.Printf("Message delivered to %v\n", m.TopicPartition)
			}
		}
	}()

	// Отправляем сообщения каждые 1 сек
	for i := 1; i <= 50; i++ {
		order := &pb.Order{
			OrderID: fmt.Sprintf("%04d", i),
			UserID:  "u001",
			Items: []*pb.Item{
				{ProductID: "P1", Quantity: 2, Price: 100},
				{ProductID: "P2", Quantity: 1, Price: 200},
			},
			TotalPrice: 400,
		}
		payload, _ := serializer.Serialize(topic, &order)
		p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          payload,
		}, deliveryChan)
		time.Sleep(time.Second)
	}

	close(deliveryChan)
}

// ---------------- SINGLE MESSAGE CONSUMER ----------------
func runSingleConsumer(bootstrap string, topic string, group string) {

	// Конфигурация для Schema Registry
	srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig("http://localhost:8081"))
	if err != nil {
		log.Fatalf("Ошибка при создании клиента Schema Registry: %v", err)
	}

	// Создание десериализатора protobuf
	deserializer, err := protobuf.NewDeserializer(srClient, serde.ValueSerde, protobuf.NewDeserializerConfig())
	if err != nil {
		log.Fatalf("Ошибка при создании десериализатора: %v", err)
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrap,
		"group.id":           group,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": true,
	})
	if err != nil {
		log.Fatalf("SingleConsumer creation failed: %s", err)
	}
	defer c.Close()
	c.SubscribeTopics([]string{topic}, nil)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	run := true

	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Signal %v: stopping SingleConsumer\n", sig)
			run = false
		default:
			ev := c.Poll(100)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				var order pb.Order
				err := deserializer.DeserializeInto(topic, e.Value, &order)
				if err != nil {
					log.Printf("[SingleConsumer] deserialize error: %v", err)
				} else {
					fmt.Printf("[SingleConsumer] Got message: %v\n", order)
				}
			case kafka.Error:
				log.Printf("[SingleConsumer] Kafka error: %v", e)
			}
		}
	}
}

// ---------------- BATCH MESSAGE CONSUMER ----------------
func runBatchConsumer(bootstrap string, topic string, group string, batchSize int) {
	// Конфигурация для Schema Registry
	srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig("http://localhost:8081"))
	if err != nil {
		log.Fatalf("Ошибка при создании клиента Schema Registry: %v", err)
	}

	// Создание десериализатора protobuf
	deserializer, err := protobuf.NewDeserializer(srClient, serde.ValueSerde, protobuf.NewDeserializerConfig())
	if err != nil {
		log.Fatalf("Ошибка при создании десериализатора: %v", err)
	}
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrap,
		"group.id":           group,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false, // ручной коммит
		"fetch.min.bytes":    1,
		"fetch.wait.max.ms":  500,
	})

	if err != nil {
		log.Fatalf("BatchConsumer creation failed: %s", err)
	}
	defer c.Close()
	c.SubscribeTopics([]string{topic}, nil)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	run := true

	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Signal %v: stopping BatchConsumer\n", sig)
			run = false
		default:
			msgs := make([]*kafka.Message, 0, batchSize)
			for len(msgs) < batchSize {
				ev := c.Poll(100)
				if ev == nil {
					continue
				}
				switch e := ev.(type) {
				case *kafka.Message:
					msgs = append(msgs, e)
				case kafka.Error:
					log.Printf("[BatchConsumer] Kafka error: %v", e)
				}
			}
			// Обработка сообщений
			for _, m := range msgs {
				var order pb.Order
				err := deserializer.DeserializeInto(topic, m.Value, &order)
				if err != nil {
					log.Printf("[BatchConsumer] deserialize error: %v", err)
				} else {
					fmt.Printf("[BatchConsumer] Got message: %v\n", order)
				}
			}
			// Коммит оффсетов пачкой
			_, err := c.Commit()
			if err != nil {
				log.Printf("[BatchConsumer] Commit error: %v", err)
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <bootstrap> <topic>\n", os.Args[0])
	}
	bootstrap := os.Args[1]
	topic := os.Args[2]

	go runProducer(bootstrap, topic)
	go runSingleConsumer(bootstrap, topic, "single-group")
	go runBatchConsumer(bootstrap, topic, "batch-group", 10)

	select {} // ждём завершения всех горутин
}
