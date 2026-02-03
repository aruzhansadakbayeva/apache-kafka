package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/colinmarc/hdfs/v2"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

type CartEvent struct {
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
}

func mustEnv(k string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		log.Fatalf("missing env %s", k)
	}
	return v
}

func main() {
	// Kafka
	brokers := strings.Split(mustEnv("KAFKA_BROKERS"), ",")
	topic := mustEnv("KAFKA_TOPIC")
	groupID := mustEnv("KAFKA_GROUP")

	// HDFS
	hdfsAddr := mustEnv("HDFS_ADDR") // "namenode:8020"
	hdfsDir := mustEnv("HDFS_DIR")   // "/data/cart"

	flushEvery := 3 * time.Second
	maxBatch := 1000

	// SASL/SSL (если у тебя kafka-destination SASL_SSL)
	user := os.Getenv("KAFKA_SASL_USER")
	pass := os.Getenv("KAFKA_SASL_PASS")

	dialer := &kafka.Dialer{Timeout: 10 * time.Second}
	if user != "" && pass != "" {
		dialer.SASLMechanism = plain.Mechanism{
			Username: user,
			Password: pass,
		}
		dialer.TLS = &tls.Config{InsecureSkipVerify: true} // лучше указать CA, но для учебного стенда ок
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
		Dialer:  dialer,
		// MinBytes/MaxBytes можно тюнить
	})
	defer reader.Close()

	hc, err := hdfs.New(hdfsAddr)
	if err != nil {
		log.Fatalf("hdfs connect: %v", err)
	}
	defer hc.Close()

	if err := ensureDir(hc, hdfsDir); err != nil {
		log.Fatalf("ensure dir: %v", err)
	}

	ctx := context.Background()
	var batch []string
	lastFlush := time.Now()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		path := fmt.Sprintf("%s/dt=%s/part-%d.jsonl", hdfsDir, time.Now().Format("2006-01-02"), time.Now().UnixNano())
		if err := writeLinesToHDFS(hc, path, batch); err != nil {
			log.Printf("flush error: %v", err)
			return
		}
		log.Printf("flushed %d events -> hdfs:%s", len(batch), path)
		batch = batch[:0]
		lastFlush = time.Now()
	}

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Fatalf("read kafka: %v", err)
		}

		// валидируем JSON на входе (опционально)
		var ev CartEvent
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			log.Printf("bad json (skip): %v", err)
			continue
		}

		batch = append(batch, string(m.Value))

		if len(batch) >= maxBatch || time.Since(lastFlush) >= flushEvery {
			flush()
		}
	}
}

func ensureDir(hc *hdfs.Client, base string) error {
	// создаем base и dt=... папки лениво в write
	return hc.MkdirAll(base, 0o755)
}

func writeLinesToHDFS(hc *hdfs.Client, path string, lines []string) error {
	dir := path[:strings.LastIndex(path, "/")]
	if err := hc.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := hc.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 1<<20)
	for _, ln := range lines {
		if _, err := w.WriteString(ln + "\n"); err != nil {
			return err
		}
	}
	return w.Flush()
}
