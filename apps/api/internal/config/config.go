package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	APIPort      int
	DBHost       string
	DBPort       int
	DBName       string
	DBUser       string
	DBPassword   string
	DBSSLMode    string
	PublicOrigin string
	CookieSecure bool
}

func Load() (Config, error) {
	apiPort, err := port("API_PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	dbPort, err := port("DB_PORT", 5432)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		APIPort: apiPort, DBHost: value("DB_HOST", "127.0.0.1"), DBPort: dbPort,
		DBName: value("DB_NAME", "blood"), DBUser: value("DB_USER", "blood"),
		DBPassword: os.Getenv("DB_PASSWORD"), DBSSLMode: value("DB_SSLMODE", "disable"),
		PublicOrigin: value("PUBLIC_ORIGIN", "http://localhost:3000"),
	}
	if cfg.DBPassword == "" {
		return Config{}, fmt.Errorf("не задана переменная DB_PASSWORD")
	}
	origin, err := url.Parse(cfg.PublicOrigin)
	if err != nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" || (origin.Scheme != "http" && origin.Scheme != "https") || strings.ContainsAny(cfg.PublicOrigin, "*\\") {
		return Config{}, fmt.Errorf("PUBLIC_ORIGIN должен содержать точный origin без пути")
	}
	if origin.Scheme != "https" && origin.Hostname() != "localhost" && !net.ParseIP(origin.Hostname()).IsLoopback() {
		return Config{}, fmt.Errorf("для публичного PUBLIC_ORIGIN необходим HTTPS")
	}
	cfg.CookieSecure = origin.Scheme == "https"
	switch cfg.DBSSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return Config{}, fmt.Errorf("недопустимый режим DB_SSLMODE")
	}
	return cfg, nil
}

// DatabaseURL корректно экранирует пароль. Результат нельзя писать в журналы.
func (cfg Config) DatabaseURL() string {
	connection := url.URL{
		Scheme: "postgres", User: url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host: net.JoinHostPort(cfg.DBHost, strconv.Itoa(cfg.DBPort)), Path: "/" + cfg.DBName,
	}
	query := connection.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	query.Set("connect_timeout", "5")
	connection.RawQuery = query.Encode()
	return connection.String()
}

func port(name string, fallback int) (int, error) {
	text := os.Getenv(name)
	if text == "" {
		return fallback, nil
	}
	number, err := strconv.Atoi(text)
	if err != nil || number < 1 || number > 65535 {
		return 0, fmt.Errorf("%s должен содержать порт от 1 до 65535", name)
	}
	return number, nil
}

func value(name, fallback string) string {
	if text := os.Getenv(name); text != "" {
		return text
	}
	return fallback
}
