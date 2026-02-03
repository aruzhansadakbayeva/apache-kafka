package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/IBM/sarama"
)

const (
	kafkaBroker = "kafka-1:1092"
	topic       = "products-topic"
	jsonFile    = "products.json"

	username = "producer"
	password = "producer-secret"

	truststorePath = "/etc/kafka/secrets/kafka.truststore.jks"
)

func main() {
	// 1. читаем файл
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	// 2. пробуем распарсить как массив
	var products []map[string]interface{}
	if err := json.Unmarshal(data, &products); err != nil {
		// если не массив — пробуем одиночный объект
		var single map[string]interface{}
		if err := json.Unmarshal(data, &single); err != nil {
			log.Fatalf("invalid json format: %v", err)
		}
		products = append(products, single)
	}

	// 3. Kafka config
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	// SASL
	config.Net.SASL.Enable = true
	config.Net.SASL.User = username
	config.Net.SASL.Password = password
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	// SSL
	tlsConfig, err := createTLSConfig(truststorePath)
	if err != nil {
		log.Fatalf("failed to create tls config: %v", err)
	}
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = tlsConfig

	// 4. Producer
	producer, err := sarama.NewSyncProducer([]string{kafkaBroker}, config)
	if err != nil {
		log.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close()

	// 5. отправка сообщений
	for _, product := range products {
		value, _ := json.Marshal(product)

		msg := &sarama.ProducerMessage{
			Topic: topic,
			Key: sarama.StringEncoder(fmt.Sprintf("%v", product["product_id"])),
			Value: sarama.ByteEncoder(value),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("failed to send message: %v", err)
			continue
		}

		fmt.Printf("Sent to partition %d offset %d\n", partition, offset)
	}

	fmt.Println("All products sent to Kafka")
}

func createTLSConfig(truststorePath string) (*tls.Config, error) {
	truststoreData, err := os.ReadFile(truststorePath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(truststoreData)

	return &tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: true, 
	}, nil
}
