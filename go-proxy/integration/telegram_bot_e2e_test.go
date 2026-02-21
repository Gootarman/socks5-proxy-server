//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/internal/bot"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/commands/createuser"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/commands/deleteuser"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/commands/generatepass"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/commands/getusers"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/commands/start"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/commands/usersstats"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/handlers/message"
	mw "github.com/nskondratev/socks5-proxy-server/internal/bot/middleware"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/store"
	"github.com/nskondratev/socks5-proxy-server/internal/services/admin"
)

func TestTelegramBotE2E_AllCommandsAndTextMessages(t *testing.T) {
	t.Setenv("PUBLIC_URL", "https://proxy.example.com/")
	t.Setenv("APP_PORT", "1080")

	redis := newFakeRedis()
	adminService := admin.New(redis)

	if err := adminService.Add(context.Background(), "admin"); err != nil {
		t.Fatalf("failed to seed admin user: %v", err)
	}

	if err := redis.HSet(context.Background(), redisUserAuthKey, "taken-user", mustHashPassword(t, "taken-pass")); err != nil {
		t.Fatalf("failed to seed taken user: %v", err)
	}
	if err := redis.HSet(context.Background(), redisUserUsageKey, "taken-user", "2048"); err != nil {
		t.Fatalf("failed to seed usage data: %v", err)
	}
	if err := redis.HSet(context.Background(), redisUserAuthDate, "taken-user", "2026-02-01T00:00:00.000Z"); err != nil {
		t.Fatalf("failed to seed auth date: %v", err)
	}

	mock := newTelegramAPIMock(t, "test-token")
	defer mock.Close()

	b := newIntegrationBot(t, redis, mock.URL(), "test-token")

	go b.Start()
	defer b.Stop()

	adminUser := tele.User{ID: 1001, Username: "admin", FirstName: "Admin"}
	regularUser := tele.User{ID: 1002, Username: "regular", FirstName: "Regular"}

	reply := sendAndAwaitReply(t, mock, regularUser, "/start")
	if !strings.Contains(reply.Text, "available only for admin users") {
		t.Fatalf("unexpected non-admin reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "/start")
	if !strings.Contains(reply.Text, "Hello! You can manage proxy server.") {
		t.Fatalf("unexpected /start reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "hello")
	if !strings.Contains(reply.Text, "Enter command") {
		t.Fatalf("unexpected idle text reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "/create_user")
	if !strings.Contains(reply.Text, "Enter username for the new proxy user.") {
		t.Fatalf("unexpected /create_user reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, " ")
	if !strings.Contains(reply.Text, "Username can not be empty") {
		t.Fatalf("unexpected empty username reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "taken-user")
	if !strings.Contains(reply.Text, "already taken") {
		t.Fatalf("unexpected occupied username reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "new-user")
	if !strings.Contains(reply.Text, "Enter the password") {
		t.Fatalf("unexpected username accepted reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, " ")
	if !strings.Contains(reply.Text, "Password can not be empty") {
		t.Fatalf("unexpected empty password reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "new-pass")
	if !strings.Contains(reply.Text, "User created. Send these settings to the user:") {
		t.Fatalf("unexpected create completion reply: %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "proxy.example.com") || !strings.Contains(reply.Text, "new-user") {
		t.Fatalf("connection details are missing in create reply: %q", reply.Text)
	}

	storedHash, err := redis.HGet(context.Background(), redisUserAuthKey, "new-user")
	if err != nil {
		t.Fatalf("failed to get created user hash: %v", err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("new-pass")); err != nil {
		t.Fatalf("created user password mismatch: %v", err)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "/get_users")
	if !strings.Contains(reply.Text, "new-user") || !strings.Contains(reply.Text, "taken-user") {
		t.Fatalf("unexpected /get_users reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "/users_stats")
	if !strings.Contains(reply.Text, "Data usage by users") || !strings.Contains(reply.Text, "taken-user") {
		t.Fatalf("unexpected /users_stats reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "/generate_pass 12")
	if len(strings.TrimSpace(reply.Text)) != 12 {
		t.Fatalf("unexpected generated password length: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "/delete_user")
	if !strings.Contains(reply.Text, "Enter username to delete.") {
		t.Fatalf("unexpected /delete_user reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "missing-user")
	if !strings.Contains(reply.Text, "does not exists") {
		t.Fatalf("unexpected missing user reply: %q", reply.Text)
	}

	reply = sendAndAwaitReply(t, mock, adminUser, "new-user")
	if !strings.Contains(reply.Text, "User deleted.") {
		t.Fatalf("unexpected delete completion reply: %q", reply.Text)
	}

	if _, err = redis.HGet(context.Background(), redisUserAuthKey, "new-user"); err == nil {
		t.Fatal("expected new-user to be deleted")
	}
}

func newIntegrationBot(t *testing.T, redis *fakeRedis, apiURL, token string) *tele.Bot {
	t.Helper()

	b, err := tele.NewBot(tele.Settings{
		URL:   apiURL,
		Token: token,
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
		Poller:      &tele.LongPoller{Timeout: time.Second},
		OnError:     bot.OnErrorCb,
		Synchronous: true,
	})
	if err != nil {
		t.Fatalf("failed to create integration bot: %v", err)
	}

	if err = b.RemoveWebhook(); err != nil {
		t.Fatalf("failed to remove webhook: %v", err)
	}

	adminService := admin.New(redis)
	b.Use(
		mw.SetTimeoutCtx(2*time.Second),
		mw.RestrictByAdminUserID(adminService),
	)

	botStore := store.New(redis)

	b.Handle(start.Command, start.New(botStore).Handle)
	b.Handle(usersstats.Command, usersstats.New(botStore).Handle)
	b.Handle(createuser.Command, createuser.New(botStore).Handle)
	b.Handle(deleteuser.Command, deleteuser.New(botStore).Handle)
	b.Handle(getusers.Command, getusers.New(botStore).Handle)
	b.Handle(generatepass.Command, generatepass.New().Handle)
	b.Handle(tele.OnText, message.New(botStore).Handle)

	return b
}

func sendAndAwaitReply(t *testing.T, mock *telegramAPIMock, user tele.User, text string) telegramMessage {
	t.Helper()

	before := mock.SentCount()
	mock.EnqueueUpdate(makeTextUpdate(user, text))

	msg, ok := mock.WaitNewMessage(before, defaultWaitTimeout)
	if !ok {
		t.Fatalf("timed out while waiting for reply to %q", text)
	}

	return msg
}

func makeTextUpdate(user tele.User, text string) tele.Update {
	now := time.Now().Unix()
	updateID := int(telegramUpdateSeq.Add(1))

	return tele.Update{
		ID: updateID,
		Message: &tele.Message{
			ID:       updateID + 100,
			Unixtime: now,
			Sender:   &user,
			Chat:     &tele.Chat{ID: user.ID, Type: tele.ChatPrivate, Username: user.Username},
			Text:     text,
		},
	}
}

var telegramUpdateSeq atomic.Int64

type telegramMessage struct {
	Text      string
	ChatID    int64
	ParseMode string
}

type telegramAPIMock struct {
	t       *testing.T
	token   string
	server  *httptest.Server
	mu      sync.Mutex
	updates []tele.Update
	sent    []telegramMessage
}

func newTelegramAPIMock(t *testing.T, token string) *telegramAPIMock {
	t.Helper()

	mock := &telegramAPIMock{
		t:     t,
		token: token,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handle))

	return mock
}

func (m *telegramAPIMock) URL() string {
	return m.server.URL
}

func (m *telegramAPIMock) Close() {
	m.server.Close()
}

func (m *telegramAPIMock) EnqueueUpdate(update tele.Update) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updates = append(m.updates, update)
}

func (m *telegramAPIMock) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.sent)
}

func (m *telegramAPIMock) WaitNewMessage(previous int, timeout time.Duration) (telegramMessage, bool) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		m.mu.Lock()
		if len(m.sent) > previous {
			msg := m.sent[previous]
			m.mu.Unlock()
			return msg, true
		}
		m.mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}

	return telegramMessage{}, false
}

func (m *telegramAPIMock) handle(w http.ResponseWriter, r *http.Request) {
	prefix := "/bot" + m.token + "/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}

	method := strings.TrimPrefix(r.URL.Path, prefix)

	switch method {
	case "getMe":
		m.respondJSON(w, map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"id":         777001,
				"is_bot":     true,
				"first_name": "ProxyBot",
				"username":   "proxy_bot",
			},
		})
	case "deleteWebhook":
		m.respondJSON(w, map[string]interface{}{"ok": true, "result": true})
	case "getUpdates":
		offset := 0
		payload := map[string]string{}
		_ = json.NewDecoder(r.Body).Decode(&payload)

		if rawOffset, ok := payload["offset"]; ok {
			if parsedOffset, err := strconv.Atoi(rawOffset); err == nil {
				offset = parsedOffset
			}
		}

		updates := m.popUpdates(offset)

		m.respondJSON(w, map[string]interface{}{
			"ok":     true,
			"result": updates,
		})
	case "sendMessage":
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		text := fmt.Sprint(payload["text"])
		parseMode := fmt.Sprint(payload["parse_mode"])
		chatID := parseChatID(payload["chat_id"])

		m.mu.Lock()
		m.sent = append(m.sent, telegramMessage{
			Text:      text,
			ChatID:    chatID,
			ParseMode: parseMode,
		})
		messageID := len(m.sent)
		m.mu.Unlock()

		m.respondJSON(w, map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": messageID,
				"date":       time.Now().Unix(),
				"chat": map[string]interface{}{
					"id":   chatID,
					"type": "private",
				},
				"text": text,
			},
		})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *telegramAPIMock) popUpdates(offset int) []tele.Update {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.updates) == 0 {
		return []tele.Update{}
	}

	result := make([]tele.Update, 0, len(m.updates))
	for _, update := range m.updates {
		if update.ID >= offset {
			result = append(result, update)
		}
	}

	m.updates = nil

	return result
}

func (m *telegramAPIMock) respondJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		m.t.Errorf("failed to write JSON response: %v", err)
	}
}

func parseChatID(raw interface{}) int64 {
	switch v := raw.(type) {
	case string:
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}

		return id
	case float64:
		return int64(v)
	default:
		return 0
	}
}
