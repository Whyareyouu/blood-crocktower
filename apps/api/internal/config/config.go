package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	APIPort    int
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string
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
	}
	if cfg.DBPassword == "" {
		return Config{}, fmt.Errorf("не задана переменная DB_PASSWORD")
	}
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
