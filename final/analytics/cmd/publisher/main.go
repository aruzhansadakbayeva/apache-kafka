package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"log"
	"os"
	"strings"
	"time"

	"github.com/colinmarc/hdfs/v2"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

func mustEnv(k string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		log.Fatalf("missing env %s", k)
	}
	return v
}

func main() {
	brokers := strings.Split(mustEnv("KAFKA_BROKERS"), ",")
	outTopic := mustEnv("KAFKA_OUT_TOPIC")

	hdfsAddr := mustEnv("HDFS_ADDR")
	hdfsRecoDir := mustEnv("HDFS_RECO_DIR") // "/data/reco/dt=latest"

	user := os.Getenv("KAFKA_SASL_USER")
	pass := os.Getenv("KAFKA_SASL_PASS")

	dialer := &kafka.Dialer{Timeout: 10 * time.Second}
	if user != "" && pass != "" {
		dialer.SASLMechanism = plain.Mechanism{Username: user, Password: pass}
		dialer.TLS = &tls.Config{InsecureSkipVerify: true}
	}

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: brokers,
		Topic:   outTopic,
		Dialer:  dialer,
		// RequiredAcks: kafka.RequireAll, // если нужно
	})
	defer w.Close()

	hc, err := hdfs.New(hdfsAddr)
	if err != nil {
		log.Fatalf("hdfs connect: %v", err)
	}
	defer hc.Close()

	entries, err := hc.ReadDir(hdfsRecoDir)
	if err != nil {
		log.Fatalf("readdir: %v", err)
	}

	ctx := context.Background()

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "part-") {
			continue
		}
		path := hdfsRecoDir + "/" + e.Name()
		f, err := hc.Open(path)
		if err != nil {
			log.Printf("open %s: %v", path, err)
			continue
		}

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			if err := w.WriteMessages(ctx, kafka.Message{Value: append([]byte(nil), line...)}); err != nil {
				log.Printf("kafka write: %v", err)
			}
		}
		_ = f.Close()
		log.Printf("published from %s", path)
	}
}
