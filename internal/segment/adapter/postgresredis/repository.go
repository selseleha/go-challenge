package postgresredis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/postgres"
)

// RedisCache is the Redis side used for estimates and async projection.
type RedisCache interface {
	Upsert(ctx context.Context, m segment.Membership) error
	CountActive(ctx context.Context, segment string, now time.Time) (int64, error)
	Flush(ctx context.Context) error
}

// Repository implements segment.Repository: Postgres is source of truth,
// Redis is the hot estimate cache filled via transactional outbox.
type Repository struct {
	store *postgres.Store
	cache RedisCache
	ttl   time.Duration
}

// NewRepository wires Postgres + Redis behind the segment.Repository port.
func NewRepository(store *postgres.Store, cache RedisCache, ttl time.Duration) *Repository {
	return &Repository{store: store, cache: cache, ttl: ttl}
}

func (r *Repository) Upsert(ctx context.Context, m segment.Membership) error {
	return r.store.UpsertWithOutbox(ctx, m)
}

func (r *Repository) CountActive(ctx context.Context, name string, now time.Time) (int64, error) {
	n, err := r.cache.CountActive(ctx, name, now)
	if err == nil {
		return n, nil
	}
	log.Printf("redis count failed, falling back to postgres: %v", err)
	return r.store.CountActiveSQL(ctx, name, now, r.ttl)
}

// Flush clears the Redis hot store (admin reset).
func (r *Repository) Flush(ctx context.Context) error {
	return r.cache.Flush(ctx)
}

// WarmFromSource reloads active memberships from Postgres into Redis.
func (r *Repository) WarmFromSource(ctx context.Context, now time.Time) error {
	active, err := r.store.ListActive(ctx, now, r.ttl)
	if err != nil {
		return fmt.Errorf("list active: %w", err)
	}
	for _, m := range active {
		if err := r.cache.Upsert(ctx, m); err != nil {
			return fmt.Errorf("warm upsert %s/%s: %w", m.Segment, m.UserID, err)
		}
	}
	log.Printf("redis: warmed %d active memberships from postgres", len(active))
	return nil
}

// AsyncWarm is true so admin reset does not block HTTP on large reloads.
func (r *Repository) AsyncWarm() bool { return true }
