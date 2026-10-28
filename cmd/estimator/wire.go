package main

import (
	"context"
	"fmt"
	"log"

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
			Service: segment.NewService(repo, segment.SystemClock{}, ready, repo, repo, cfg.Store),
			Cleanup: func() {},
		}, nil

	case "redis":
		client, cache, err := openRedisCache(ctx, cfg)
		if err != nil {
			return appBundle{}, err
		}
		log.Printf("store: redis (%s)", cfg.RedisAddr)
		return appBundle{
			Service: segment.NewService(cache, segment.SystemClock{}, ready, cache, cache, cfg.Store),
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
		worker := postgresredis.NewWorker(store, cache, cfg.OutboxPollInterval, cfg.OutboxBatchSize)
		repo.SetWorker(worker)
		go worker.Run(ctx)
		log.Printf("store: postgres_redis (db + redis=%s, outbox batch=%d every %s)",
			cfg.RedisAddr, cfg.OutboxBatchSize, cfg.OutboxPollInterval)
		return appBundle{
			Service: segment.NewService(repo, segment.SystemClock{}, ready, repo, repo, cfg.Store),
			Cleanup: func() {
				store.Close()
				_ = client.Close()
			},
		}, nil

	default:
		return appBundle{}, fmt.Errorf("unknown STORE %q", cfg.Store)
	}
}

func openRedisCache(ctx context.Context, cfg config.Config) (*goredis.Client, *segmentredis.Repository, error) {
	client := goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("redis ping %s: %w", cfg.RedisAddr, err)
	}
	return client, segmentredis.NewRepository(client, cfg.MembershipTTL), nil
}
