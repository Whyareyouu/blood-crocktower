package config

import (
	"net/url"
	"testing"
)

func configure(t *testing.T) {
	t.Helper()
	for _, key := range []string{"API_PORT", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_SSLMODE"} {
		t.Setenv(key, "")
	}
	t.Setenv("DB_PASSWORD", "local-test")
}

func TestDatabaseURLPreservesSpecialCharacters(t *testing.T) {
	configure(t)
	t.Setenv("DB_PASSWORD", "test@:/?#% +")
	t.Setenv("DB_HOST", "::1")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(cfg.DatabaseURL())
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	if password != "test@:/?#% +" || parsed.Host != "[::1]:5432" {
		t.Fatal("параметры подключения повреждены при экранировании")
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"DB_PASSWORD", ""}, {"API_PORT", "0"}, {"DB_PORT", "65536"},
		{"DB_PORT", "abc"}, {"DB_SSLMODE", "unexpected"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			configure(t)
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("недопустимая конфигурация принята")
			}
		})
	}
}
