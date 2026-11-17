package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/contracts"
	"golang.org/x/time/rate"
)

const sessionCookie = "blood_session"
const csrfCookie = "blood_csrf"

type bucket struct {
	limiter *rate.Limiter
	seen    time.Time
}
type limits struct {
	mu      sync.Mutex
	entries map[string]*bucket
}

func (l *limits) allow(key string, interval time.Duration, burst int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.entries == nil {
		l.entries = make(map[string]*bucket)
	}
	b, ok := l.entries[key]
	if !ok {
		// Ограничиваем память и удаляем только давно неиспользуемые ключи.
		if len(l.entries) >= 10000 {
			for k, entry := range l.entries {
				if now.Sub(entry.seen) > time.Hour {
					delete(l.entries, k)
				}
			}
			if len(l.entries) >= 10000 {
				return false
			}
		}
		b = &bucket{limiter: rate.NewLimiter(rate.Every(interval), burst)}
		l.entries[key] = b
	}
	b.seen = now
	return b.limiter.AllowN(now, 1)
}

func remoteIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	// X-Forwarded-For от клиента намеренно не является источником доверия.
	return ip
}

func cookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *server) setCookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true,
		Secure: s.options.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: age,
		Expires: time.Now().UTC().Add(time.Duration(age) * time.Second)})
}

func apiError(w http.ResponseWriter, status int, code contracts.ErrorCode, message string) {
	writeJSON(w, status, contracts.APIError{Code: code, Message: message})
}

func limited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	apiError(w, 429, contracts.ErrorCodeRateLimited, "Слишком много запросов. Повторите через минуту.")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, schema string, value any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		apiError(w, 415, contracts.ErrorCodeInvalidRequest, "Ожидается application/json.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		apiError(w, 400, contracts.ErrorCodeInvalidRequest, "Некорректный JSON или превышен размер запроса.")
		return false
	}
	if contracts.Validate(schema, data) != nil || json.Unmarshal(data, value) != nil {
		apiError(w, 400, contracts.ErrorCodeInvalidRequest, "Проверьте данные: логин — 3–32 латинских символа, цифры, _ или -; пароль — 12–128 символов.")
		return false
	}
	return true
}

func (s *server) identity(w http.ResponseWriter, r *http.Request) (auth.Identity, bool) {
	if s.options.Auth == nil {
		apiError(w, 401, contracts.ErrorCodeUnauthenticated, "Войдите в аккаунт.")
		return auth.Identity{}, false
	}
	id, err := s.options.Auth.Resolve(r.Context(), cookieValue(r, sessionCookie))
	if err == auth.ErrSession {
		apiError(w, 401, contracts.ErrorCodeUnauthenticated, "Войдите в аккаунт.")
		return id, false
	}
	if err != nil {
		apiError(w, 503, contracts.ErrorCodeInternalError, "Сервис временно недоступен.")
		return id, false
	}
	return id, true
}

func (s *server) security(next http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	if s.options.PublicOrigin != "" {
		_ = protection.AddTrustedOrigin(s.options.PublicOrigin)
	}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, 403, contracts.ErrorCodeForbidden, "Запрос с другого сайта запрещён.")
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cache-Control", "no-store")
		if !s.limits.allow("http:"+remoteIP(r), time.Second/2, 120) {
			limited(w)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if origin := r.Header.Get("Origin"); origin != "" && origin != s.options.PublicOrigin {
				apiError(w, 403, contracts.ErrorCodeForbidden, "Запрос с другого сайта запрещён.")
				return
			}
			expected := cookieValue(r, csrfCookie)
			if token := cookieValue(r, sessionCookie); token != "" {
				expected = auth.CSRFToken(token)
			}
			provided := r.Header.Get("X-CSRF-Token")
			if len(expected) != 43 || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
				apiError(w, 403, contracts.ErrorCodeForbidden, "Обновите страницу и повторите запрос.")
				return
			}
		}
		protection.Handler(next).ServeHTTP(w, r)
	})
}
