package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
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

// --- Blacklist storage (JSON file) ---

type BannedItem struct {
	ProductID string `json:"product_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Reason    string `json:"reason,omitempty"`
	AddedAt   string `json:"added_at,omitempty"`
}

type BannedStore struct {
	path        string
	mu          sync.RWMutex
	data        []BannedItem
	lastModTime time.Time
}

func NewBannedStore(path string) *BannedStore {
	return &BannedStore{path: path}
}

func (s *BannedStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = []BannedItem{}
			s.lastModTime = time.Time{}
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		s.data = []BannedItem{}
	} else {
		var items []BannedItem
		if err := json.Unmarshal(b, &items); err != nil {
			return fmt.Errorf("failed to parse %s: %w", s.path, err)
		}
		s.data = items
	}

	// remember file mod time for hot-reload
	if fi, err := os.Stat(s.path); err == nil {
		s.lastModTime = fi.ModTime()
	}
	return nil
}

func (s *BannedStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func norm(s1 string) string {
	return strings.ToLower(strings.TrimSpace(s1))
}

func (s *BannedStore) List() []BannedItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BannedItem, len(s.data))
	copy(out, s.data)
	return out
}

func (s *BannedStore) Add(item BannedItem) error {
	if strings.TrimSpace(item.ProductID) == "" && strings.TrimSpace(item.Name) == "" {
		return errors.New("either --id or --name must be provided")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idN := norm(item.ProductID)
	nameN := norm(item.Name)

	for _, it := range s.data {
		if idN != "" && norm(it.ProductID) == idN {
			return fmt.Errorf("product_id already banned: %s", item.ProductID)
		}
		if nameN != "" && norm(it.Name) == nameN {
			return fmt.Errorf("name already banned: %s", item.Name)
		}
	}

	item.AddedAt = time.Now().Format(time.RFC3339)
	s.data = append(s.data, item)
	return nil
}

func (s *BannedStore) RemoveByID(id string) (bool, error) {
	idN := norm(id)
	if idN == "" {
		return false, errors.New("--id is required for remove")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	out := s.data[:0]
	for _, it := range s.data {
		if norm(it.ProductID) == idN {
			changed = true
			continue
		}
		out = append(out, it)
	}
	s.data = out
	return changed, nil
}

func (s *BannedStore) IsBanned(p Product) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pid := norm(p.ProductID)
	pname := norm(p.Name)

	for _, it := range s.data {
		if it.ProductID != "" && norm(it.ProductID) == pid {
			return true, fmt.Sprintf("banned by product_id (%s)", it.ProductID)
		}
		if it.Name != "" && norm(it.Name) == pname {
			return true, fmt.Sprintf("banned by name (%s)", it.Name)
		}
	}
	return false, ""
}

// ReloadIfChanged reloads the banned list when the underlying JSON file changes.
// This enables managing the list through CLI (separate process) without restarting the stream.
func (s *BannedStore) ReloadIfChanged() {
	fi, err := os.Stat(s.path)
	if err != nil {
		return
	}

	s.mu.RLock()
	last := s.lastModTime
	s.mu.RUnlock()

	if fi.ModTime().After(last) {
		if err := s.Load(); err == nil {
			log.Printf("banned list reloaded: %d items", len(s.List()))
		} else {
			log.Printf("banned list reload failed: %v", err)
		}
	}
}

// --- ENV helpers ---

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

// --- Kafka dialer (SASL_SSL PLAIN) ---

func buildDialer() (*kafka.Dialer, error) {
	securityProtocol := strings.ToUpper(getenv("KAFKA_SECURITY_PROTOCOL", ""))
	saslUser := getenv("KAFKA_SASL_USER", "")
	saslPass := getenv("KAFKA_SASL_PASS", "")
	saslMech := strings.ToUpper(getenv("KAFKA_SASL_MECHANISM", "PLAIN"))

	d := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	if securityProtocol == "SASL_SSL" {
		// Dev-friendly: certificates usually don't match docker hostname.
		d.TLS = &tls.Config{InsecureSkipVerify: true}

		if saslMech != "PLAIN" {
			return nil, fmt.Errorf("unsupported SASL mechanism: %s (only PLAIN supported here)", saslMech)
		}
		if saslUser == "" || saslPass == "" {
			return nil, errors.New("SASL_SSL requires KAFKA_SASL_USER and KAFKA_SASL_PASS")
		}
		d.SASLMechanism = plain.Mechanism{Username: saslUser, Password: saslPass}
	}

	return d, nil
}

// --- Commands ---

func usage() {
	fmt.Println(`Usage:
  filter list
  filter add --id <product_id> [--name <name>] [--reason <text>]
  filter remove --id <product_id>
  filter run

Env for run:
  KAFKA_BROKERS (default: kafka-1:1092)
  KAFKA_IN_TOPIC (default: products-topic)
  KAFKA_OUT_TOPIC (default: products-filtered-topic)
  GOKA_GROUP (default: products-filter-v1)

  KAFKA_SECURITY_PROTOCOL=SASL_SSL
  KAFKA_SASL_USER=admin
  KAFKA_SASL_PASS=admin-secret
  KAFKA_SASL_MECHANISM=PLAIN

Optional:
  BANNED_FILE (default: /app/banned.json or ./banned.json locally)
  BANNED_RELOAD_SECONDS (default: 3)
`)
}

func defaultBannedPath() string {
	if _, err := os.Stat("/app"); err == nil {
		return "/app/banned.json"
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "banned.json")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]

	bannedPath := getenv("BANNED_FILE", defaultBannedPath())
	store := NewBannedStore(bannedPath)
	if err := store.Load(); err != nil {
		log.Fatalf("failed to load banned list: %v", err)
	}

	switch cmd {
	case "list":
		items := store.List()
		if len(items) == 0 {
			fmt.Println("banned list is empty")
			return
		}
		for i, it := range items {
			fmt.Printf("%d) id=%q name=%q reason=%q added_at=%s\n", i+1, it.ProductID, it.Name, it.Reason, it.AddedAt)
		}
		return

	case "add":
		fs := flag.NewFlagSet("add", flag.ExitOnError)
		id := fs.String("id", "", "product_id to ban")
		name := fs.String("name", "", "exact name to ban (optional)")
		reason := fs.String("reason", "", "reason (optional)")
		_ = fs.Parse(os.Args[2:])

		item := BannedItem{ProductID: *id, Name: *name, Reason: *reason}
		if err := store.Add(item); err != nil {
			log.Fatalf("add failed: %v", err)
		}
		if err := store.Save(); err != nil {
			log.Fatalf("save failed: %v", err)
		}
		fmt.Printf("added banned item: id=%q name=%q\n", item.ProductID, item.Name)
		return

	case "remove":
		fs := flag.NewFlagSet("remove", flag.ExitOnError)
		id := fs.String("id", "", "product_id to unban")
		_ = fs.Parse(os.Args[2:])

		changed, err := store.RemoveByID(*id)
		if err != nil {
			log.Fatalf("remove failed: %v", err)
		}
		if !changed {
			fmt.Printf("no banned item found for id=%q\n", *id)
			return
		}
		if err := store.Save(); err != nil {
			log.Fatalf("save failed: %v", err)
		}
		fmt.Printf("removed banned item id=%q\n", *id)
		return

	case "run":
		runStream(store)
		return

	default:
		usage()
		os.Exit(1)
	}
}

func runStream(store *BannedStore) {
	brokers := parseBrokers(getenv("KAFKA_BROKERS", "kafka-1:1092"))
	inTopic := getenv("KAFKA_IN_TOPIC", "products-topic")
	outTopic := getenv("KAFKA_OUT_TOPIC", "products-filtered-topic")
	groupID := getenv("GOKA_GROUP", "products-filter-v1")

	reloadSeconds := 3
	if v := strings.TrimSpace(os.Getenv("BANNED_RELOAD_SECONDS")); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &reloadSeconds); err != nil || n != 1 || reloadSeconds < 1 {
			reloadSeconds = 3
		}
	}

	dialer, err := buildDialer()
	if err != nil {
		log.Fatalf("dialer init failed: %v", err)
	}

	log.Printf("started banned-filter: brokers=%v input=%s output=%s group=%s banned_file=%s reload=%ds",
		brokers, inTopic, outTopic, groupID, store.path, reloadSeconds)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         brokers,
		Topic:           inTopic,
		GroupID:         groupID,
		Dialer:          dialer,
		MinBytes:        1e3,
		MaxBytes:        10e6,
		MaxWait:         500 * time.Millisecond,
		CommitInterval:  0,
		ReadLagInterval: -1,
	})
	defer func() { _ = reader.Close() }()

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown signal received, stopping...")
		cancel()
	}()

	reloadTicker := time.NewTicker(time.Duration(reloadSeconds) * time.Second)
	defer reloadTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("context canceled, exiting")
			return
		case <-reloadTicker.C:
			store.ReloadIfChanged()
		default:
		}

		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("context canceled, exiting")
				return
			}
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

		if banned, why := store.IsBanned(p); banned {
			log.Printf("BLOCKED: product_id=%q name=%q (%s)", p.ProductID, p.Name, why)
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
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("CommitMessages error (offset=%d): %v", msg.Offset, err)
		}
	}
}
