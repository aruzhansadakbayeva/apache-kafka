package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

type Product struct {
	ProductID   string `json:"product_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func parseBrokers(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func main() {
	// --- ENV from your compose ---
	brokers := parseBrokers(getenv("KAFKA_BROKERS", "kafka-1:1092"))
	inTopic := getenv("KAFKA_IN_TOPIC", "products-topic")
	outTopic := getenv("KAFKA_OUT_TOPIC", "products-filtered-topic")
	groupID := getenv("GOKA_GROUP", "products-filter-v1")

	securityProtocol := strings.ToUpper(getenv("KAFKA_SECURITY_PROTOCOL", ""))
	saslUser := getenv("KAFKA_SASL_USER", "")
	saslPass := getenv("KAFKA_SASL_PASS", "")
	saslMech := strings.ToUpper(getenv("KAFKA_SASL_MECHANISM", "PLAIN"))

	latinRe := regexp.MustCompile(`[A-Za-z]`)

	// --- Dialer (TLS + SASL if needed) ---
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	// В твоём случае: SASL_SSL + PLAIN
	// Если захочешь поддержать другие механизмы — расширим.
	if securityProtocol == "SASL_SSL" {
		// ⚠️ В dev/docker часто сертификат не совпадает с hostname kafka-1,
		// поэтому без этого будет tls: handshake failure / x509: certificate is not valid for...
		dialer.TLS = &tls.Config{
			InsecureSkipVerify: true,
		}

		if saslMech != "PLAIN" {
			log.Fatalf("unsupported SASL mechanism: %s (this build supports PLAIN)", saslMech)
		}
		if saslUser == "" || saslPass == "" {
			log.Fatalf("SASL_SSL requires KAFKA_SASL_USER and KAFKA_SASL_PASS")
		}
		dialer.SASLMechanism = plain.Mechanism{
			Username: saslUser,
			Password: saslPass,
		}
	}

	log.Printf("started filter service: brokers=%v input=%s output=%s group=%s protocol=%s sasl=%s",
		brokers, inTopic, outTopic, groupID, securityProtocol, saslMech)

	// --- Reader ---
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         brokers,
		Topic:           inTopic,
		GroupID:         groupID,
		Dialer:          dialer,
		MinBytes:        1e3,
		MaxBytes:        10e6,
		MaxWait:         500 * time.Millisecond,
		CommitInterval:  0, // manual commit
		ReadLagInterval: -1,
	})
	defer func() { _ = reader.Close() }()

	// --- Writer ---
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        outTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		Transport: &kafka.Transport{
			Dial: dialer.DialFunc,
			TLS:  dialer.TLS,
			SASL: dialer.SASLMechanism,
		},
	}
	defer func() { _ = writer.Close() }()

	// --- graceful shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown signal received, stopping...")
		cancel()
	}()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("context canceled, exiting")
				return
			}
			// тут уже не будет EOF, если auth/tls настроены правильно
			log.Printf("FetchMessage error: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var p Product
		if err := json.Unmarshal(msg.Value, &p); err != nil {
			log.Printf("invalid json, skip (offset=%d): %v", msg.Offset, err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		name := strings.TrimSpace(p.Name)
		if name == "" {
			log.Printf("empty name, skip (offset=%d)", msg.Offset)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		// Filter: allow only names WITHOUT latin letters
		if latinRe.MatchString(name) {
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		out := kafka.Message{
			Key:     msg.Key,
			Value:   msg.Value,
			Headers: append([]kafka.Header(nil), msg.Headers...),
			Time:    time.Now(),
		}

		if err := writer.WriteMessages(ctx, out); err != nil {
			log.Printf("WriteMessages error (offset=%d): %v", msg.Offset, err)
			// не коммитим, чтобы сообщение переобработалось
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("CommitMessages error (offset=%d): %v", msg.Offset, err)
		}
	}
}
