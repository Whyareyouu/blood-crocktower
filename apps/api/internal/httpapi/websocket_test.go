package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/contracts"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func dial(t *testing.T, base string, b *browser, origin string, protocols []string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	if b != nil {
		for _, c := range b.cookies {
			header.Add("Cookie", c.String())
		}
	}
	return websocket.Dial(ctx, strings.Replace(base, "http://", "ws://", 1)+"/api/v1/ws", &websocket.DialOptions{HTTPHeader: header, Subprotocols: protocols})
}

func exchange(t *testing.T, c *websocket.Conn, body string) json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, []byte(body)); err != nil {
		t.Fatal(err)
	}
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := contracts.Validate("ServerMessage", data); err != nil {
		t.Fatalf("невалидный ответ: %s %v", data, err)
	}
	return data
}

func command(kind, payload string) string {
	return fmt.Sprintf(`{"version":1,"type":%q,"requestId":"123e4567-e89b-42d3-a456-426614174000","payload":%s}`, kind, payload)
}

func TestWebSocketOriginAuthorizationAndRevocation(t *testing.T) {
	h, db := setup(t)
	b := newBrowser()
	user := b.signIn(t, h, "socket_user", true)
	srv := httptest.NewServer(h)
	defer srv.Close()
	for _, tc := range []struct {
		origin    string
		b         *browser
		protocols []string
		status    int
	}{
		{"", b, []string{"blood.v1"}, 403},
		{"null", b, []string{"blood.v1"}, 403},
		{"https://evil.example", b, []string{"blood.v1"}, 403},
		{testOrigin, nil, []string{"blood.v1"}, 401},
		{testOrigin, b, []string{"wrong"}, 426},
	} {
		c, response, err := dial(t, srv.URL, tc.b, tc.origin, tc.protocols)
		if c != nil {
			c.CloseNow()
		}
		if err == nil || response == nil || response.StatusCode != tc.status {
			t.Fatalf("upgrade: response=%v err=%v", response, err)
		}
	}
	c, _, err := dial(t, srv.URL, b, testOrigin, []string{"blood.v1"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	if c.Subprotocol() != "blood.v1" {
		t.Fatal("подпротокол не согласован")
	}
	if data := exchange(t, c, command("ping", `{}`)); !strings.Contains(string(data), `"pong"`) {
		t.Fatal("нет pong")
	}
	for _, body := range []string{
		command("room.subscribe", `{"roomId":"other"}`),
		command("conversation.subscribe", `{"conversationId":"other"}`),
		command("room.ready", `{"roomId":"other","ready":true}`),
		command("conversation.message.send", `{"conversationId":"other","text":"привет"}`),
	} {
		if data := exchange(t, c, body); !strings.Contains(string(data), `"forbidden"`) {
			t.Fatal("нет проверки доступа к чужой команде")
		}
	}
	for _, body := range []string{command("ping", `{"userId":"victim"}`), command("room.ready", `{"roomId":"other"}`), command("unknown", `{}`), strings.Replace(command("ping", `{}`), `"version":1`, `"version":2`, 1)} {
		if data := exchange(t, c, body); !strings.Contains(string(data), `"invalid_request"`) {
			t.Fatal("некорректная команда принята")
		}
	}
	room := access.Room{ID: "mine", Name: "Комната", CreatorID: user.User.Id, Mode: contracts.ModeProgramStoryteller, Script: contracts.ScriptTroubleBrewing, Capacity: 10, State: contracts.RoomStateWaiting}
	if err := db.Create(&room).Error; err != nil {
		t.Fatal(err)
	}
	member := access.Membership{RoomID: room.ID, UserID: user.User.Id, Player: true}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	if data := exchange(t, c, command("room.subscribe", `{"roomId":"mine"}`)); strings.Contains(string(data), `"forbidden"`) {
		t.Fatal("своё членство не учтено")
	}
	if err := db.Delete(&member).Error; err != nil {
		t.Fatal(err)
	}
	if data := exchange(t, c, command("room.subscribe", `{"roomId":"mine"}`)); !strings.Contains(string(data), `"forbidden"`) {
		t.Fatal("права не перепроверены после удаления членства")
	}
	if w := b.request(h, "POST", "/api/v1/auth/logout", ""); w.Code != 204 {
		t.Fatal("выход не выполнен")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _, err = c.Read(ctx)
	if websocket.CloseStatus(err) != 4401 {
		t.Fatalf("выход не закрыл WS с 4401: %v", err)
	}
}

func TestWebSocketExpiredSessionAndFrameLimits(t *testing.T) {
	h, db := setup(t)
	b := newBrowser()
	b.signIn(t, h, "frame_user", true)
	srv := httptest.NewServer(h)
	defer srv.Close()
	for _, tc := range []struct {
		name   string
		kind   websocket.MessageType
		data   string
		status websocket.StatusCode
	}{
		{"binary", websocket.MessageBinary, `{}`, 1008},
		{"oversize", websocket.MessageText, strings.Repeat("x", 65537), 1009},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, err := dial(t, srv.URL, b, testOrigin, []string{"blood.v1"})
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = c.Write(ctx, tc.kind, []byte(tc.data))
			_, _, err = c.Read(ctx)
			if websocket.CloseStatus(err) != tc.status {
				t.Fatalf("код закрытия: %v", err)
			}
		})
	}
	c, _, err := dial(t, srv.URL, b, testOrigin, []string{"blood.v1"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	if err := db.Model(&auth.Session{}).Where(&auth.Session{TokenHash: auth.Digest(b.cookies[sessionCookie].Value)}).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var response any
	err = wsjson.Read(ctx, c, &response)
	if websocket.CloseStatus(err) != 4401 {
		t.Fatalf("истёкшая молчащая сессия не закрыта: %v", err)
	}
}

func TestWebSocketCommandRateLimit(t *testing.T) {
	h, _ := setup(t)
	b := newBrowser()
	b.signIn(t, h, "rate_user", true)
	srv := httptest.NewServer(h)
	defer srv.Close()
	c, _, err := dial(t, srv.URL, b, testOrigin, []string{"blood.v1"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < 30; i++ {
		err = c.Write(ctx, websocket.MessageText, []byte(command("ping", `{}`)))
		if err != nil {
			break
		}
		_, _, err = c.Read(ctx)
		if err != nil {
			break
		}
	}
	if websocket.CloseStatus(err) != 1008 {
		t.Fatalf("нет ограничения WS-команд: %v", err)
	}
}

func TestWebSocketConnectionLimitAndShutdown(t *testing.T) {
	h, _ := setup(t)
	b := newBrowser()
	b.signIn(t, h, "connections_user", true)
	srv := httptest.NewServer(h)
	defer srv.Close()
	var connections []*websocket.Conn
	defer func() {
		for _, c := range connections {
			c.CloseNow()
		}
	}()
	for i := 0; i < 4; i++ {
		c, _, err := dial(t, srv.URL, b, testOrigin, []string{"blood.v1"})
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, c)
		_ = exchange(t, c, command("ping", `{}`))
	}
	extra, _, err := dial(t, srv.URL, b, testOrigin, []string{"blood.v1"})
	if err != nil {
		t.Fatal(err)
	}
	defer extra.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _, err = extra.Read(ctx)
	if websocket.CloseStatus(err) != 1008 {
		t.Fatalf("нет ограничения соединений: %v", err)
	}
	h.(*Handler).Close()
	_, _, err = connections[0].Read(ctx)
	if websocket.CloseStatus(err) != 1001 {
		t.Fatalf("остановка не закрыла соединения: %v", err)
	}
}

// Отдельная проверка уже запущенного production-стека, включая upgrade
// через настоящий Next.js proxy. Создаёт один тестовый аккаунт.
func TestWebSocketThroughProxy(t *testing.T) {
	base := os.Getenv("SMOKE_WEB_URL")
	if base == "" {
		t.Skip("для проверки запущенного proxy задайте SMOKE_WEB_URL")
	}
	origin := os.Getenv("PUBLIC_ORIGIN")
	if origin == "" {
		origin = base
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	response, err := client.Get(base + "/api/v1/auth/csrf")
	if err != nil {
		t.Fatal(err)
	}
	var csrf contracts.CSRFToken
	err = json.NewDecoder(response.Body).Decode(&csrf)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 {
		t.Fatal("не получен предварительный CSRF")
	}
	credentials, _ := json.Marshal(contracts.Credentials{Login: "proxy_" + auth.RandomToken()[:16], Password: "temporary proxy test password"})
	request, err := http.NewRequest("POST", base+"/api/v1/auth/register", strings.NewReader(string(credentials)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", origin)
	request.Header.Set("X-CSRF-Token", csrf.CsrfToken)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var session contracts.Session
	err = json.NewDecoder(response.Body).Decode(&session)
	response.Body.Close()
	if err != nil || response.StatusCode != 201 {
		t.Fatal("не выполнена регистрация через proxy")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	endpoint := strings.Replace(strings.Replace(base, "https://", "wss://", 1), "http://", "ws://", 1) + "/api/v1/ws"
	c, _, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Origin": []string{origin}}, Subprotocols: []string{"blood.v1"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	if data := exchange(t, c, command("ping", `{}`)); !strings.Contains(string(data), `"pong"`) {
		t.Fatal("нет pong через proxy")
	}
	request, err = http.NewRequest("POST", base+"/api/v1/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", origin)
	request.Header.Set("X-CSRF-Token", session.CsrfToken)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatal("не выполнен выход через proxy")
	}
	_, _, err = c.Read(ctx)
	if websocket.CloseStatus(err) != 4401 {
		t.Fatalf("выход через proxy не отозвал WebSocket: %v", err)
	}
}
