package auth_test

import (
	"context"
	"errors"
	"github.com/alexedwards/argon2id"
	"strings"
	"sync"
	"testing"
	"time"

	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/contracts"
	"blood/apps/api/internal/testutil"
)

func TestPersistentSessions(t *testing.T) {
	db := testutil.Database(t)
	s, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := contracts.Credentials{Login: "Player_1", Password: "длинный надёжный пароль"}
	session, token, err := s.Authenticate(ctx, c, true, "")
	if err != nil {
		t.Fatal(err)
	}
	var stored auth.User
	if err := db.Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.PasswordHash == c.Password || !strings.HasPrefix(stored.PasswordHash, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Fatal("пароль не сохранён как Argon2id RFC 9106")
	}
	_, salt, key, err := argon2id.DecodeHash(stored.PasswordHash)
	if err != nil || len(salt) != 16 || len(key) != 32 {
		t.Fatal("неверная соль или длина ключа")
	}
	var record auth.Session
	if err := db.Take(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.TokenHash == token || record.TokenHash != auth.Digest(token) {
		t.Fatal("в БД должен храниться только SHA-256 токена")
	}
	if session.CsrfToken != auth.CSRFToken(token) {
		t.Fatal("CSRF не связан с сессией")
	}
	_, _, err = s.Authenticate(ctx, c, true, "")
	if !errors.Is(err, auth.ErrConflict) {
		t.Fatalf("повторный логин: %v", err)
	}
	c.Login = "PLAYER_1"
	other, otherToken, err := s.Authenticate(ctx, c, false, "")
	if err != nil || other.User != session.User || otherToken == token || other.CsrfToken == session.CsrfToken {
		t.Fatal("новое устройство должно иметь тот же аккаунт и отдельную сессию")
	}
	// Новый экземпляр сервиса моделирует перезапуск API, а не память процесса.
	restarted, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := restarted.Resolve(ctx, token); err != nil || id.User != session.User {
		t.Fatal("сессия не восстановлена после перезапуска")
	}
	_, rotated, err := restarted.Authenticate(ctx, c, false, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(ctx, token); !errors.Is(err, auth.ErrSession) {
		t.Fatal("старая сессия не отозвана при входе")
	}
	if _, err := s.Resolve(ctx, otherToken); err != nil {
		t.Fatal("вход завершил сессию другого устройства")
	}
	if err := s.Logout(ctx, rotated); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(ctx, rotated); !errors.Is(err, auth.ErrSession) {
		t.Fatal("выход не отозвал сессию")
	}
	if err := db.Model(&auth.Session{}).Where(&auth.Session{TokenHash: auth.Digest(otherToken)}).Update("expires_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(ctx, otherToken); !errors.Is(err, auth.ErrSession) {
		t.Fatal("истёкшая сессия принята")
	}
	for _, login := range []string{"player_1", "unknown"} {
		_, _, err := s.Authenticate(ctx, contracts.Credentials{Login: login, Password: "wrong password 123"}, false, "")
		if !errors.Is(err, auth.ErrCredentials) {
			t.Fatal("ошибка входа раскрывает существование аккаунта")
		}
	}
}

func TestConcurrentRegistration(t *testing.T) {
	db := testutil.Database(t)
	s, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	errorsCh := make(chan error, 4)
	var group sync.WaitGroup
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, _, err := s.Authenticate(context.Background(), contracts.Credentials{Login: "same_login", Password: "password long enough"}, true, "")
			errorsCh <- err
		}()
	}
	group.Wait()
	close(errorsCh)
	success, conflict := 0, 0
	for err := range errorsCh {
		if err == nil {
			success++
		} else if errors.Is(err, auth.ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 3 {
		t.Fatalf("гонка регистрации: success=%d conflict=%d", success, conflict)
	}
}

func TestCredentialValidation(t *testing.T) {
	for _, c := range []contracts.Credentials{
		{Login: "ab", Password: strings.Repeat("a", 12)}, {Login: "Привет", Password: strings.Repeat("a", 12)},
		{Login: "valid", Password: strings.Repeat("a", 11)}, {Login: "valid", Password: strings.Repeat("я", 129)},
		{Login: strings.Repeat("a", 33), Password: strings.Repeat("a", 12)},
	} {
		if auth.ValidCredentials(c) {
			t.Fatalf("приняты недопустимые данные: %s", c.Login)
		}
	}
	if !auth.ValidCredentials(contracts.Credentials{Login: "Valid_1-", Password: strings.Repeat("я", 128)}) {
		t.Fatal("длина должна считаться в символах Unicode")
	}
}
