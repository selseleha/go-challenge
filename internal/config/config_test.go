package config_test

import (
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/config"
)

func clearStoreEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"HTTP_ADDR", "GRPC_ADDR", "STORE", "REDIS_ADDR", "DATABASE_URL",
		"MEMBERSHIP_TTL", "ADMIN_TOKEN",
		"OUTBOX_POLL_INTERVAL", "OUTBOX_BATCH_SIZE",
		"OUTBOX_CLEANUP_INTERVAL", "OUTBOX_CLEANUP_RETENTION", "OUTBOX_CLEANUP_BATCH",
		"WARM_PAGE_SIZE",
		"USS_GRPC_ADDR", "USS_WORKERS", "USS_EVENTS", "USS_TIMEOUT",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearStoreEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Store != "memory" {
		t.Fatalf("Store = %q", cfg.Store)
	}
	if cfg.MembershipTTL != 14*24*time.Hour {
		t.Fatalf("MembershipTTL = %v", cfg.MembershipTTL)
	}
	if cfg.AdminToken != "dev-admin-token" {
		t.Fatalf("AdminToken = %q", cfg.AdminToken)
	}
	if cfg.OutboxBatchSize != 1000 {
		t.Fatalf("OutboxBatchSize = %d", cfg.OutboxBatchSize)
	}
	if cfg.OutboxPollInterval != 100*time.Millisecond {
		t.Fatalf("OutboxPollInterval = %v", cfg.OutboxPollInterval)
	}
	if cfg.OutboxCleanupInterval != time.Minute || cfg.OutboxCleanupRetention != 15*time.Minute {
		t.Fatalf("cleanup interval/retention = %v/%v", cfg.OutboxCleanupInterval, cfg.OutboxCleanupRetention)
	}
	if cfg.OutboxCleanupBatch != 10000 || cfg.WarmPageSize != 50000 {
		t.Fatalf("cleanupBatch=%d warmPage=%d", cfg.OutboxCleanupBatch, cfg.WarmPageSize)
	}
}

func TestLoad_PostgresRedisRequiresDatabaseURL(t *testing.T) {
	clearStoreEnv(t)
	t.Setenv("STORE", "postgres_redis")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL missing")
	}
}

func TestLoad_RejectsUnknownStore(t *testing.T) {
	clearStoreEnv(t)
	t.Setenv("STORE", "mongo")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for unknown store")
	}
}
