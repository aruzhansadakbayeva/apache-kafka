// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "strconv"
    "time"

    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const topicWaitingTime = 60 * time.Second

func main() {

    if len(os.Args) != 5 {
        log.Fatalf(
            "Usage: %s <bootstrap-servers> <topic> <partition-count> <replication-factor>\n",
            os.Args[0])
    }

    bootstrapServers := os.Args[1]
    topic := os.Args[2]
    numParts, err := strconv.Atoi(os.Args[3])
    if err != nil {
        log.Fatalf("Invalid partition count: %s: %v\n", os.Args[3], err)
    }
    replicationFactor, err := strconv.Atoi(os.Args[4])
    if err != nil {
        log.Fatalf("Invalid replication factor: %s: %v\n", os.Args[4], err)
    }

    // Cоздаём новый админский клиент.
    a, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": bootstrapServers})
    if err != nil {
        log.Fatalf("Ошибка создания админского клиента: %s\n", err)
    }

    results, err := a.CreateTopics(
        context.Background(),
        // Можно одновременно создать несколько топиков, указав несколько структур TopicSpecification
        []kafka.TopicSpecification{{
            Topic:             topic,
            NumPartitions:     numParts,
            ReplicationFactor: replicationFactor}},
        // Устанавливаем опцию админского клиента — ждать не более topicWaitingTime
        kafka.SetAdminOperationTimeout(topicWaitingTime))
    if err != nil {
        log.Fatalf("Ошибка создания топика: %s\n", err)
    }

    // Результат (тип TopicResult) содержит информацию о результате операции создания
    //топика. Если бы мы создавали несколько топиков, то получили бы информацию по каждому из них.
for _, result := range results {
    if result.Error.Code() != kafka.ErrNoError {
        fmt.Printf("Не удалось создать топик %s: %v\n", result.Topic, result.Error)
    } else {
        fmt.Printf("Топик '%s' успешно создан!\n", result.Topic)
        fmt.Printf("  Количество партиций: %d\n", numParts)
        fmt.Printf("  Репликация: %d\n", replicationFactor)
    }
}

    a.Close()
}