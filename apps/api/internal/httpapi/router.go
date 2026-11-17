package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"

	"blood/apps/api/internal/contracts"
)

type Options struct {
	Auth         *auth.Service
	Access       *access.Policy
	PublicOrigin string
	CookieSecure bool
}

type server struct {
	options Options
	limits  limits
	sockets socketHub
}

type Handler struct {
	http.Handler
	server *server
}

// Close завершает также WebSocket, которые http.Server.Shutdown не закрывает.
func (h *Handler) Close() { h.server.sockets.closeAll() }

func New(checkReady func(context.Context) error, options ...Options) *Handler {
	s := &server{options: Options{PublicOrigin: "http://localhost:3000"}}
	if len(options) > 0 {
		s.options = options[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, contracts.Health{Status: "ok"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := checkReady(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, contracts.Health{Status: "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, contracts.Health{Status: "ok"})
	})
	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, contracts.Status{Service: "blood-api", Stage: "accounts"})
	})
	mux.HandleFunc("GET /api/v1/auth/csrf", s.csrf)
	mux.HandleFunc("POST /api/v1/auth/register", s.authenticate)
	mux.HandleFunc("POST /api/v1/auth/login", s.authenticate)
	mux.HandleFunc("GET /api/v1/auth/session", func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.identity(w, r)
		if !ok {
			return
		}
		writeJSON(w, 200, contracts.Session{User: id.User, CsrfToken: auth.CSRFToken(cookieValue(r, sessionCookie))})
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.identity(w, r)
		if !ok {
			return
		}
		if err := s.options.Auth.Logout(r.Context(), cookieValue(r, sessionCookie)); err != nil {
			apiError(w, 503, contracts.ErrorCodeInternalError, "Сервис временно недоступен.")
			return
		}
		s.sockets.revoke(id.TokenHash)
		s.setCookie(w, sessionCookie, "", -1)
		s.setCookie(w, csrfCookie, "", -1)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/users/me/rooms", func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.identity(w, r)
		if !ok {
			return
		}
		if s.options.Access == nil {
			apiError(w, 503, contracts.ErrorCodeInternalError, "Сервис временно недоступен.")
			return
		}
		rooms, err := s.options.Access.MyRooms(r.Context(), id.User.Id)
		if err != nil {
			apiError(w, 503, contracts.ErrorCodeInternalError, "Сервис временно недоступен.")
			return
		}
		writeJSON(w, 200, rooms)
	})
	mux.HandleFunc("GET /api/v1/ws", s.websocket)
	// Даже ещё не реализованные операции находятся за общей границей входа.
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.identity(w, r); !ok {
			return
		}
		apiError(w, 404, contracts.ErrorCodeNotFound, "Операция недоступна.")
	})
	return &Handler{Handler: s.security(mux), server: s}
}

func (s *server) csrf(w http.ResponseWriter, r *http.Request) {
	if token := cookieValue(r, sessionCookie); token != "" && s.options.Auth != nil {
		_, err := s.options.Auth.Resolve(r.Context(), token)
		if err == nil {
			writeJSON(w, 200, contracts.CSRFToken{CsrfToken: auth.CSRFToken(token)})
			return
		}
		if !errors.Is(err, auth.ErrSession) {
			apiError(w, 503, contracts.ErrorCodeInternalError, "Сервис временно недоступен.")
			return
		}
		s.setCookie(w, sessionCookie, "", -1)
	}
	token := cookieValue(r, csrfCookie)
	if len(token) != 43 {
		token = auth.RandomToken()
	}
	s.setCookie(w, csrfCookie, token, 3600)
	writeJSON(w, 200, contracts.CSRFToken{CsrfToken: token})
}

func (s *server) authenticate(w http.ResponseWriter, r *http.Request) {
	if !s.limits.allow("auth-ip:"+remoteIP(r), 6*time.Second, 10) {
		limited(w)
		return
	}
	var c contracts.Credentials
	if !decodeJSON(w, r, "Credentials", &c) {
		return
	}
	if !auth.ValidCredentials(c) {
		apiError(w, 400, contracts.ErrorCodeInvalidRequest, "Логин: 3–32 латинских символа, цифры, _ или -. Пароль: 12–128 символов.")
		return
	}
	if !s.limits.allow("auth-login:"+strings.ToLower(c.Login), 12*time.Second, 5) {
		limited(w)
		return
	}
	if s.options.Auth == nil {
		apiError(w, 503, contracts.ErrorCodeInternalError, "Сервис временно недоступен.")
		return
	}
	oldToken := cookieValue(r, sessionCookie)
	register := r.URL.Path == "/api/v1/auth/register"
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	session, token, err := s.options.Auth.Authenticate(ctx, c, register, oldToken)
	switch {
	case errors.Is(err, auth.ErrCredentials):
		apiError(w, 401, contracts.ErrorCodeUnauthenticated, "Неверный логин или пароль.")
		return
	case errors.Is(err, auth.ErrConflict):
		apiError(w, 409, contracts.ErrorCodeConflict, "Этот логин уже занят.")
		return
	case err != nil:
		apiError(w, 503, contracts.ErrorCodeInternalError, "Сервис временно недоступен.")
		return
	}
	if oldToken != "" {
		s.sockets.revoke(auth.Digest(oldToken))
	}
	s.setCookie(w, sessionCookie, token, int(auth.SessionLifetime.Seconds()))
	s.setCookie(w, csrfCookie, "", -1)
	status := http.StatusOK
	if register {
		status = http.StatusCreated
	}
	writeJSON(w, status, session)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
