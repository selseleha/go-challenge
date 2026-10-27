package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds runtime settings loaded from environment / .env.
type Config struct {
	HTTPAddr      string
	GRPCAddr      string
	Store         string // memory | redis | postgres_redis
	RedisAddr     string
	DatabaseURL   string
	MembershipTTL time.Duration
	AdminToken    string

	OutboxPollInterval time.Duration
	OutboxBatchSize    int

	USSGRPCAddr string
	USSWorkers  int
	USSEvents   int
}

// Load reads optional .env then environment variables.
// Missing .env is ignored so production can rely on real env only.
func Load() (Config, error) {
	_ = godotenv.Load()

	ttl, err := parseDuration(envOr("MEMBERSHIP_TTL", "14d"))
	if err != nil {
		return Config{}, fmt.Errorf("MEMBERSHIP_TTL: %w", err)
	}
	if ttl <= 0 {
		return Config{}, fmt.Errorf("MEMBERSHIP_TTL must be positive")
	}

	workers, err := envInt("USS_WORKERS", 8)
	if err != nil {
		return Config{}, fmt.Errorf("USS_WORKERS: %w", err)
	}
	events, err := envInt("USS_EVENTS", 1000)
	if err != nil {
		return Config{}, fmt.Errorf("USS_EVENTS: %w", err)
	}
	batch, err := envInt("OUTBOX_BATCH_SIZE", 100)
	if err != nil {
		return Config{}, fmt.Errorf("OUTBOX_BATCH_SIZE: %w", err)
	}
	poll, err := parseDuration(envOr("OUTBOX_POLL_INTERVAL", "2s"))
	if err != nil {
		return Config{}, fmt.Errorf("OUTBOX_POLL_INTERVAL: %w", err)
	}

	cfg := Config{
		HTTPAddr:           envOr("HTTP_ADDR", ":8080"),
		GRPCAddr:           envOr("GRPC_ADDR", ":9090"),
		Store:              strings.ToLower(envOr("STORE", "memory")),
		RedisAddr:          envOr("REDIS_ADDR", "localhost:6379"),
		DatabaseURL:        envOr("DATABASE_URL", ""),
		MembershipTTL:      ttl,
		AdminToken:         envOr("ADMIN_TOKEN", "dev-admin-token"),
		OutboxPollInterval: poll,
		OutboxBatchSize:    batch,
		USSGRPCAddr:        dialAddr(envOr("USS_GRPC_ADDR", envOr("GRPC_ADDR", "localhost:9090"))),
		USSWorkers:         workers,
		USSEvents:          events,
	}

	switch cfg.Store {
	case "memory", "redis":
	case "postgres_redis":
		if cfg.DatabaseURL == "" {
			return Config{}, fmt.Errorf("DATABASE_URL is required when STORE=postgres_redis")
		}
	default:
		return Config{}, fmt.Errorf("STORE must be memory, redis, or postgres_redis, got %q", cfg.Store)
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("must be positive")
	}
	return n, nil
}

// parseDuration accepts Go durations (e.g. 336h) or day shorthand (e.g. 14d).
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") || strings.HasSuffix(s, "D") {
		days, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSuffix(s, "d"), "D"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// dialAddr turns ":9090" into "localhost:9090" for client connections.
func dialAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}
