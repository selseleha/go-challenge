package config_test

import (
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("GRPC_ADDR", "")
	t.Setenv("STORE", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("MEMBERSHIP_TTL", "")
	t.Setenv("USS_WORKERS", "")
	t.Setenv("USS_EVENTS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != ":9090" {
		t.Fatalf("GRPCAddr = %q", cfg.GRPCAddr)
	}
	if cfg.Store != "memory" {
		t.Fatalf("Store = %q", cfg.Store)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Fatalf("RedisAddr = %q", cfg.RedisAddr)
	}
	if cfg.MembershipTTL != 14*24*time.Hour {
		t.Fatalf("MembershipTTL = %v", cfg.MembershipTTL)
	}
	if cfg.USSGRPCAddr != "localhost:9090" {
		t.Fatalf("USSGRPCAddr = %q", cfg.USSGRPCAddr)
	}
	if cfg.USSWorkers != 8 || cfg.USSEvents != 1000 {
		t.Fatalf("uss workers/events = %d/%d", cfg.USSWorkers, cfg.USSEvents)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":18080")
	t.Setenv("GRPC_ADDR", ":19090")
	t.Setenv("STORE", "redis")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("MEMBERSHIP_TTL", "7d")
	t.Setenv("USS_GRPC_ADDR", "localhost:19090")
	t.Setenv("USS_WORKERS", "16")
	t.Setenv("USS_EVENTS", "500")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":18080" || cfg.GRPCAddr != ":19090" {
		t.Fatalf("addrs = %q %q", cfg.HTTPAddr, cfg.GRPCAddr)
	}
	if cfg.Store != "redis" || cfg.RedisAddr != "redis:6379" {
		t.Fatalf("store = %q %q", cfg.Store, cfg.RedisAddr)
	}
	if cfg.MembershipTTL != 7*24*time.Hour {
		t.Fatalf("MembershipTTL = %v, want 7d", cfg.MembershipTTL)
	}
	if cfg.USSGRPCAddr != "localhost:19090" || cfg.USSWorkers != 16 || cfg.USSEvents != 500 {
		t.Fatalf("uss = %+v", cfg)
	}
}

func TestLoad_RejectsUnknownStore(t *testing.T) {
	t.Setenv("STORE", "mongo")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for unknown store")
	}
}

func TestLoad_RejectsInvalidTTL(t *testing.T) {
	t.Setenv("MEMBERSHIP_TTL", "nope")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid ttl")
	}
}
