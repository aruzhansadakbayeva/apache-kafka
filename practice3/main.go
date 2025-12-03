package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/lovoo/goka"
)

// Структуры данных

type Message struct {
	FromUserID int64  `json:"from_user_id"`
	ToUserID   int64  `json:"to_user_id"`
	Text       string `json:"text"`
}

type UserBlock struct {
	Reason string    `json:"reason"`          // Причина блокировки
	Until  time.Time `json:"until,omitempty"` // До какого времени блокировка
}

// Ключ: FromUserID, значение: map[ToUserID]UserBlock
type BlockedUsers struct {
	Users map[int64]map[int64]UserBlock `json:"users"`
}

type CensoredMessage struct {
	FromUserID int64  `json:"from_user_id"`
	ToUserID   int64  `json:"to_user_id"`
	Text       string `json:"text"`
}

// JsonCodec для Goka

type JsonCodec[T any] struct{}

func (jc JsonCodec[T]) Encode(value interface{}) ([]byte, error) {
	if v, ok := value.(T); ok {
		return json.Marshal(v)
	}
	return nil, fmt.Errorf("illegal type: %T", value)
}

func (jc JsonCodec[T]) Decode(data []byte) (interface{}, error) {
	var t T
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return t, nil
}

// Константы топиков и групп

var (
	brokers = []string{"localhost:9094", "localhost:9095"}

	topicMessages     goka.Stream = "messages"
	topicFiltered     goka.Stream = "filtered_messages"
	topicBlockedUsers goka.Stream = "blocked_users"

	groupMessageFilter goka.Group = "message-filter-group"
)

// Цензура

var forbiddenWords = []string{"badword", "curse", "spam"}

func censorMessage(text string) string {
	for _, word := range forbiddenWords {
		text = strings.ReplaceAll(text, word, "***")
	}
	return text
}

// Глобальный список заблокированных пользователей


var blockedUsersGlobal = struct {
	users map[int64]map[int64]UserBlock
}{
	users: map[int64]map[int64]UserBlock{},
}

// Проверка блокировки между конкретными пользователями

func isBlocked(from, to int64) bool {
	toMap, ok := blockedUsersGlobal.users[from]
	if !ok {
		return false
	}

	block, ok := toMap[to]
	if !ok {
		return false
	}

	// Проверка срока блокировки
	if !block.Until.IsZero() && time.Now().After(block.Until) {
		delete(toMap, to)
		if len(toMap) == 0 {
			delete(blockedUsersGlobal.users, from)
		}
		return false
	}

	return true
}

// Процессор сообщений и блокировок


func messageFilterProcessor() {
	processFunc := func(ctx goka.Context, msg interface{}) {
		switch v := msg.(type) {
		case Message:
			if isBlocked(v.FromUserID, v.ToUserID) {
				log.Printf("[FILTER] Сообщение от %d к %d заблокировано!",
					v.FromUserID, v.ToUserID)
				return
			}

			censoredText := censorMessage(v.Text)
			ctx.Emit(topicFiltered, strconv.FormatInt(v.ToUserID, 10), CensoredMessage{
				FromUserID: v.FromUserID,
				ToUserID:   v.ToUserID,
				Text:       censoredText,
			})
			log.Printf("[FILTER] Сообщение от %d к %d: %s", v.FromUserID, v.ToUserID, censoredText)

		case BlockedUsers:
			for from, toMap := range v.Users {
				if blockedUsersGlobal.users[from] == nil {
					blockedUsersGlobal.users[from] = map[int64]UserBlock{}
				}
				for to, block := range toMap {
					blockedUsersGlobal.users[from][to] = block
				}
			}
			log.Printf("[BLOCKED] Обновлён список заблокированных пользователей : %+v", blockedUsersGlobal.users)
		}
	}

	g := goka.DefineGroup(groupMessageFilter,
		goka.Input(topicMessages, new(JsonCodec[Message]), processFunc),
		goka.Input(topicBlockedUsers, new(JsonCodec[BlockedUsers]), processFunc),
		goka.Output(topicFiltered, new(JsonCodec[CensoredMessage])),
	)

	p, err := goka.NewProcessor(brokers, g)
	if err != nil {
		log.Fatal(err)
	}
	if err := p.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// Эмиттер сообщений

func messageEmitter() {
	e, err := goka.NewEmitter(brokers, topicMessages, new(JsonCodec[Message]))
	if err != nil {
		log.Fatal(err)
	}
	defer e.Finish()

	for {
		time.Sleep(1 * time.Second)
		msg := Message{
			FromUserID: rand.Int63n(5),
			ToUserID:   rand.Int63n(5),
			Text:       "hello badword world",
		}
		e.EmitSync(strconv.FormatInt(msg.ToUserID, 10), msg)
		log.Printf("[EMITTER] Отправлено сообщение: %+v", msg)
	}
}

// Эмиттер списка блокировок


func blockedUsersEmitter() {
	e, err := goka.NewEmitter(brokers, topicBlockedUsers, new(JsonCodec[BlockedUsers]))
	if err != nil {
		log.Fatal(err)
	}
	defer e.Finish()

	for {
		time.Sleep(5 * time.Second)
		blocked := BlockedUsers{
			Users: map[int64]map[int64]UserBlock{
				1: {3: {Reason: "spam", Until: time.Now().Add(1 * time.Hour)}}, // 1 → 3
				3: {1: {Reason: "abuse"}},                                      // 3 → 1
			},
		}
		e.EmitSync("blocked", blocked)
		log.Printf("[EMITTER BLOCKED] Отправлен список заблокированных пользователей: %+v", blocked.Users)
	}
}


func main() {
	go messageEmitter()
	go blockedUsersEmitter()
	go messageFilterProcessor()

	select {} // блокируем main
}
