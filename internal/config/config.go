package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP           HTTPConfig
	Database       DatabaseConfig
	Telegram       TelegramConfig
	LLM            LLMConfig
	TZName         string
	AllowedUserIDs []int64
}

type HTTPConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	DSN string
}

type TelegramConfig struct {
	BotToken string
}

type LLMConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

func Load() (Config, error) {
	// Загружаем .env. Если файла нет, используем переменные,
	// переданные окружением, например через Docker Compose.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	cfg := Config{
		HTTP: HTTPConfig{
			Host: strings.TrimSpace(os.Getenv("HTTP_HOST")),
		},
		Database: DatabaseConfig{
			DSN: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		},
		Telegram: TelegramConfig{
			BotToken: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		},
		LLM: LLMConfig{
			BaseURL: strings.TrimSpace(os.Getenv("LLM_BASE_URL")),
			APIKey:  strings.TrimSpace(os.Getenv("LLM_API_KEY")),
			Model:   strings.TrimSpace(os.Getenv("LLM_MODEL")),
		},
		TZName: strings.TrimSpace(os.Getenv("TZ_NAME")),
	}

	port, err := strconv.Atoi(strings.TrimSpace(os.Getenv("HTTP_PORT")))
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_PORT must be a valid integer: %w", err)
	}
	cfg.HTTP.Port = port

	allowedUserIDs, err := parseAllowedUserIDs(os.Getenv("ALLOWED_USER_IDS"))
	if err != nil {
		return Config{}, fmt.Errorf("ALLOWED_USER_IDS: %w", err)
	}
	cfg.AllowedUserIDs = allowedUserIDs

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	if cfg.HTTP.Host == "" {
		return fmt.Errorf("HTTP_HOST must not be empty")
	}

	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		return fmt.Errorf("HTTP_PORT must be between 1 and 65535")
	}

	if net.ParseIP(cfg.HTTP.Host) == nil &&
		strings.ContainsAny(cfg.HTTP.Host, " /\\") {
		return fmt.Errorf("HTTP_HOST contains invalid characters")
	}

	if cfg.Database.DSN == "" {
		return fmt.Errorf("DATABASE_URL must not be empty")
	}

	databaseURL, err := url.Parse(cfg.Database.DSN)
	if err != nil ||
		(databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") ||
		databaseURL.Host == "" ||
		databaseURL.User == nil ||
		databaseURL.Path == "" ||
		databaseURL.Path == "/" {
		return fmt.Errorf("DATABASE_URL must be a valid PostgreSQL connection URL")
	}

	if cfg.Telegram.BotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN must not be empty")
	}

	if cfg.LLM.BaseURL == "" {
		return fmt.Errorf("LLM_BASE_URL must not be empty")
	}

	llmURL, err := url.ParseRequestURI(cfg.LLM.BaseURL)
	if err != nil ||
		llmURL.Host == "" ||
		(llmURL.Scheme != "http" && llmURL.Scheme != "https") {
		return fmt.Errorf("LLM_BASE_URL must be a valid HTTP or HTTPS URL")
	}

	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("LLM_API_KEY must not be empty")
	}

	if cfg.LLM.Model == "" {
		return fmt.Errorf("LLM_MODEL must not be empty")
	}

	if cfg.TZName == "" {
		return fmt.Errorf("TZ_NAME must not be empty")
	}

	if _, err := time.LoadLocation(cfg.TZName); err != nil {
		return fmt.Errorf("TZ_NAME is invalid: %w", err)
	}

	return nil
}

func parseAllowedUserIDs(value string) ([]int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	ids := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("contains an empty user ID")
		}

		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("user IDs must be positive integers")
		}

		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("contains duplicate user ID: %d", id)
		}

		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids, nil
}
