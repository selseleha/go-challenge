package main

import (
	"context"
	"fmt"
	"log"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/selseleha/go-challenge/internal/config"
	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/memory"
	"github.com/selseleha/go-challenge/internal/segment/adapter/postgres"
	"github.com/selseleha/go-challenge/internal/segment/adapter/postgresredis"
	segmentredis "github.com/selseleha/go-challenge/internal/segment/adapter/redis"
)

// appBundle is the composition-root wiring result.
type appBundle struct {
	Service *segment.Service
	Cleanup func()
}

func wireApp(ctx context.Context, cfg config.Config) (appBundle, error) {
	ready := segment.NewReadiness()

	switch cfg.Store {
	case "memory":
		repo := memory.NewRepository(cfg.MembershipTTL)
		log.Print("store: memory")
		return appBundle{
			Service: newSegmentService(repo, ready, repo, repo, cfg),
			Cleanup: func() {},
		}, nil

	case "redis":
		client, cache, err := openRedisCache(ctx, cfg)
		if err != nil {
			return appBundle{}, err
		}
		log.Printf("store: redis (%s)", cfg.RedisAddr)
		return appBundle{
			Service: newSegmentService(cache, ready, cache, cache, cfg),
			Cleanup: func() { _ = client.Close() },
		}, nil

	case "postgres_redis":
		store, err := postgres.NewStore(ctx, cfg.DatabaseURL)
		if err != nil {
			return appBundle{}, fmt.Errorf("postgres: %w", err)
		}
		client, cache, err := openRedisCache(ctx, cfg)
		if err != nil {
			store.Close()
			return appBundle{}, err
		}
		repo := postgresredis.NewRepository(store, cache, cfg.MembershipTTL)
		repo.SetWarmPageSize(cfg.WarmPageSize)
		worker := postgresredis.NewWorker(store, cache, cfg.OutboxPollInterval, cfg.OutboxBatchSize)
		repo.SetWorker(worker)
		go worker.Run(ctx)
		cleaner := postgresredis.NewCleaner(
			store,
			cfg.OutboxCleanupInterval,
			cfg.OutboxCleanupRetention,
			cfg.OutboxCleanupBatch,
		)
		go cleaner.Run(ctx)
		log.Printf("store: postgres_redis (db + redis=%s, outbox batch=%d every %s, cleanup every %s keep %s, warm page=%d)",
			cfg.RedisAddr, cfg.OutboxBatchSize, cfg.OutboxPollInterval,
			cfg.OutboxCleanupInterval, cfg.OutboxCleanupRetention, cfg.WarmPageSize)
		return appBundle{
			Service: newSegmentService(repo, ready, repo, repo, cfg),
			Cleanup: func() {
				store.Close()
				_ = client.Close()
			},
		}, nil

	default:
		return appBundle{}, fmt.Errorf("unknown STORE %q", cfg.Store)
	}
}

func newSegmentService(
	repo segment.Repository,
	ready *segment.Readiness,
	hot segment.HotStoreControl,
	stats segment.StoreStatser,
	cfg config.Config,
) *segment.Service {
	return segment.NewService(repo, segment.SystemClock{}, ready, hot, stats, cfg.Store).WithProduct(segment.ProductConfig{
		Store:              cfg.Store,
		MembershipTTL:      formatTTL(cfg.MembershipTTL),
		OutboxBatchSize:    cfg.OutboxBatchSize,
		OutboxPollInterval: cfg.OutboxPollInterval.String(),
		WarmPageSize:       cfg.WarmPageSize,
		CleanupRetention:   cfg.OutboxCleanupRetention.String(),
	})
}

func formatTTL(d time.Duration) string {
	day := 24 * time.Hour
	if d >= day && d%day == 0 {
		return fmt.Sprintf("%dd", int(d/day))
	}
	return d.String()
}

func openRedisCache(ctx context.Context, cfg config.Config) (*goredis.Client, *segmentredis.Repository, error) {
	client := goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("redis ping %s: %w", cfg.RedisAddr, err)
	}
	return client, segmentredis.NewRepository(client, cfg.MembershipTTL), nil
}
