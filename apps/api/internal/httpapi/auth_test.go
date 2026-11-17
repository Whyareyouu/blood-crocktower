package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/contracts"
	"blood/apps/api/internal/database"
	"blood/apps/api/internal/testutil"
	"gorm.io/gorm"
)

const testOrigin = "http://localhost:3000"
const testPassword = "надёжный пароль 12345"

type browser struct {
	cookies map[string]*http.Cookie
	csrf    string
}

func newBrowser() *browser { return &browser{cookies: map[string]*http.Cookie{}} }

func (b *browser) request(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Origin", testOrigin)
	r.Header.Set("Content-Type", "application/json")
	if b.csrf != "" {
		r.Header.Set("X-CSRF-Token", b.csrf)
	}
	for _, c := range b.cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	for _, c := range w.Result().Cookies() {
		if c.MaxAge < 0 {
			delete(b.cookies, c.Name)
		} else {
			b.cookies[c.Name] = c
		}
	}
	return w
}

func setup(t *testing.T) (http.Handler, *gorm.DB) {
	t.Helper()
	db := testutil.Database(t)
	a, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	return New(database.Ready(db), Options{Auth: a, Access: access.New(db), PublicOrigin: testOrigin, CookieSecure: true}), db
}

func (b *browser) prepare(t *testing.T, h http.Handler) {
	t.Helper()
	w := b.request(h, "GET", "/api/v1/auth/csrf", "")
	if w.Code != 200 {
		t.Fatalf("csrf: %d %s", w.Code, w.Body)
	}
	var value contracts.CSRFToken
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	b.csrf = value.CsrfToken
}

func (b *browser) signIn(t *testing.T, h http.Handler, login string, register bool) contracts.Session {
	t.Helper()
	b.prepare(t, h)
	path := "/api/v1/auth/login"
	expected := 200
	if register {
		path = "/api/v1/auth/register"
		expected = 201
	}
	body, _ := json.Marshal(contracts.Credentials{Login: login, Password: testPassword})
	w := b.request(h, "POST", path, string(body))
	if w.Code != expected {
		t.Fatalf("вход %s: %d %s", login, w.Code, w.Body)
	}
	if err := contracts.Validate("Session", w.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	var value contracts.Session
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	b.csrf = value.CsrfToken
	return value
}

func TestHTTPAccountLifecycleAndDevices(t *testing.T) {
	h, db := setup(t)
	first := newBrowser()
	for _, path := range []string{"/api/v1/auth/session", "/api/v1/users/me/rooms", "/api/v1/rooms", "/api/v1/rooms/other/messages"} {
		if w := first.request(h, "GET", path, ""); w.Code != 401 {
			t.Fatalf("%s без входа: %d", path, w.Code)
		}
	}
	created := first.signIn(t, h, "Player", true)
	c := first.cookies[sessionCookie]
	if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" || c.MaxAge != int(auth.SessionLifetime.Seconds()) {
		t.Fatal("cookie не защищена")
	}
	if w := first.request(h, "GET", "/api/v1/auth/session", ""); w.Code != 200 || contracts.Validate("Session", w.Body.Bytes()) != nil {
		t.Fatal("сессия не восстановлена после обновления страницы")
	}
	room := access.Room{ID: "mine", Name: "Моя комната", CreatorID: created.User.Id, Mode: contracts.ModeProgramStoryteller, Script: contracts.ScriptTroubleBrewing, Capacity: 10, State: contracts.RoomStateWaiting}
	if err := db.Create(&room).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&access.Membership{RoomID: room.ID, UserID: created.User.Id, Player: true}).Error; err != nil {
		t.Fatal(err)
	}
	second := newBrowser()
	restored := second.signIn(t, h, "PLAYER", false)
	if restored.User != created.User {
		t.Fatal("другое устройство получило другой аккаунт")
	}
	if w := second.request(h, "GET", "/api/v1/users/me/rooms", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "Моя комната") || contracts.Validate("RoomList", w.Body.Bytes()) != nil {
		t.Fatal("членство не восстановлено на другом устройстве")
	}
	outsider := newBrowser()
	outsider.signIn(t, h, "outsider", true)
	if w := outsider.request(h, "GET", "/api/v1/users/me/rooms", ""); w.Body.String() != "{\"rooms\":[]}\n" {
		t.Fatal("посторонний увидел чужое членство")
	}
	saved := first.csrf
	first.csrf = second.csrf
	if w := first.request(h, "POST", "/api/v1/auth/logout", ""); w.Code != 403 {
		t.Fatal("CSRF другого устройства принят")
	}
	first.csrf = saved
	oldCookie := *first.cookies[sessionCookie]
	first.signIn(t, h, "player", false)
	stale := newBrowser()
	stale.cookies[sessionCookie] = &oldCookie
	if w := stale.request(h, "GET", "/api/v1/auth/session", ""); w.Code != 401 {
		t.Fatal("фиксация сессии: старый токен действителен")
	}
	if w := first.request(h, "POST", "/api/v1/auth/logout", ""); w.Code != 204 {
		t.Fatalf("выход: %d", w.Code)
	}
	if _, ok := first.cookies[sessionCookie]; ok {
		t.Fatal("cookie не удалена")
	}
	if w := second.request(h, "GET", "/api/v1/auth/session", ""); w.Code != 200 {
		t.Fatal("выход завершил чужое устройство")
	}
}

func TestHTTPRejectsCSRFAndMalformedRequests(t *testing.T) {
	h, _ := setup(t)
	b := newBrowser()
	b.prepare(t, h)
	for _, tc := range []struct {
		name, body, origin, csrf string
		status                   int
	}{
		{"без csrf", `{"login":"valid","password":"123456789012"}`, testOrigin, "", 403},
		{"чужой origin", `{"login":"valid","password":"123456789012"}`, "https://evil.example", b.csrf, 403},
		{"null origin", `{}`, "null", b.csrf, 403},
		{"лишнее поле", `{"login":"valid","password":"123456789012","userId":"victim"}`, testOrigin, b.csrf, 400},
		{"регистр имён полей", `{"LOGIN":"valid","password":"123456789012"}`, testOrigin, b.csrf, 400},
		{"нет пароля", `{"login":"valid"}`, testOrigin, b.csrf, 400},
		{"null", `null`, testOrigin, b.csrf, 400},
		{"несколько объектов", `{} {}`, testOrigin, b.csrf, 400},
		{"огромное тело", strings.Repeat(" ", 9000), testOrigin, b.csrf, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(tc.body))
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-CSRF-Token", tc.csrf)
			for _, c := range b.cookies {
				r.AddCookie(c)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || contracts.Validate("APIError", w.Body.Bytes()) != nil {
				t.Fatalf("%d %s", w.Code, w.Body)
			}
		})
	}
	if w := b.request(h, "POST", "/api/v1/auth/login", `{"login":"unknown","password":"123456789012"}`); w.Code != 401 || strings.Contains(w.Body.String(), "unknown") {
		t.Fatal("раскрытие аккаунта")
	}
}

func TestRateLimitsDoNotTrustForwardedIP(t *testing.T) {
	h := New(func(context.Context) error { return nil })
	var last int
	for i := 0; i < 130; i++ {
		r := httptest.NewRequest("GET", "/api/v1/status", nil)
		r.Header.Set("X-Forwarded-For", auth.RandomToken())
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		last = w.Code
	}
	if last != 429 {
		t.Fatal("нет ограничения HTTP или доверен поддельный адрес")
	}
	// Отдельный более строгий лимит дорогих операций.
	h = New(func(context.Context) error { return nil })
	b := newBrowser()
	b.prepare(t, h)
	for i := 0; i < 11; i++ {
		last = b.request(h, "POST", "/api/v1/auth/login", `{"login":"valid","password":"123456789012"}`).Code
	}
	if last != 429 {
		t.Fatal("нет ограничения входа")
	}
}

func TestJSONContentType(t *testing.T) {
	h := New(func(context.Context) error { return nil })
	b := newBrowser()
	b.prepare(t, h)
	r := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(`{}`))
	r.Header.Set("X-CSRF-Token", b.csrf)
	r.Header.Set("Content-Type", "text/plain")
	for _, c := range b.cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal("принят простой cross-site Content-Type")
	}
}
