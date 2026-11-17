package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadinessDoesNotLeakInternalErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"ready", nil, 200, "ok"},
		{"database unavailable", errors.New("private database credentials"), 503, "not_ready"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deadlineSet := false
			router := New(func(ctx context.Context) error {
				_, deadlineSet = ctx.Deadline()
				return tc.err
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if response.Code != tc.status || response.Body.String() != "{\"status\":\""+tc.body+"\"}\n" {
				t.Fatalf("неожиданный ответ: %d %s", response.Code, response.Body)
			}
			if !deadlineSet || strings.Contains(response.Body.String(), "private") {
				t.Fatal("нет таймаута проверки или раскрыта внутренняя ошибка")
			}
		})
	}
}

func TestLivenessIsIndependentOfDatabase(t *testing.T) {
	router := New(func(context.Context) error {
		t.Fatal("liveness не должна обращаться к БД")
		return nil
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("неожиданный ответ liveness")
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/health/live", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatal("изменяющий метод не должен приниматься")
	}
}
