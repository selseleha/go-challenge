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
	// UpsertBatch writes many memberships (preferably pipelined). errs[i] matches ms[i].
	UpsertBatch(ctx context.Context, ms []segment.Membership) []error
	CountActive(ctx context.Context, segment string, now time.Time) (int64, error)
	Flush(ctx context.Context) error
}

// Repository implements segment.Repository: Postgres is source of truth,
// Redis is the hot estimate cache filled via transactional outbox.
type Repository struct {
	store  *postgres.Store
	cache  RedisCache
	ttl    time.Duration
	worker *Worker
}

// NewRepository wires Postgres + Redis behind the segment.Repository port.
func NewRepository(store *postgres.Store, cache RedisCache, ttl time.Duration) *Repository {
	return &Repository{store: store, cache: cache, ttl: ttl}
}

// SetWorker attaches the outbox worker so reset can pause/resume it safely.
func (r *Repository) SetWorker(w *Worker) { r.worker = w }

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

// BeforeWarm pauses the outbox worker and discards pending projections.
func (r *Repository) BeforeWarm(ctx context.Context) error {
	if r.worker != nil {
		r.worker.Pause()
	}
	n, err := r.store.DiscardPendingOutbox(ctx)
	if err != nil {
		return fmt.Errorf("discard pending outbox: %w", err)
	}
	log.Printf("outbox: discarded %d pending rows for cache reset", n)
	return nil
}

// AfterWarm resumes the outbox worker.
func (r *Repository) AfterWarm(context.Context) error {
	if r.worker != nil {
		r.worker.Resume()
	}
	return nil
}

// WarmFromSource reloads active memberships from Postgres into Redis.
func (r *Repository) WarmFromSource(ctx context.Context, now time.Time) error {
	active, err := r.store.ListActive(ctx, now, r.ttl)
	if err != nil {
		return fmt.Errorf("list active: %w", err)
	}
	const warmChunk = 1000
	for i := 0; i < len(active); i += warmChunk {
		end := i + warmChunk
		if end > len(active) {
			end = len(active)
		}
		chunk := active[i:end]
		errs := r.cache.UpsertBatch(ctx, chunk)
		for j, err := range errs {
			if err != nil {
				m := chunk[j]
				return fmt.Errorf("warm upsert %s/%s: %w", m.Segment, m.UserID, err)
			}
		}
	}
	log.Printf("redis: warmed %d active memberships from postgres", len(active))
	return nil
}

// AsyncWarm is true so admin reset does not block HTTP on large reloads.
func (r *Repository) AsyncWarm() bool { return true }

func (r *Repository) Stats(ctx context.Context) (segment.StoreStats, error) {
	if st, ok := r.cache.(segment.StoreStatser); ok {
		stats, err := st.Stats(ctx)
		if err != nil {
			return stats, err
		}
		stats.Kind = "postgres_redis"
		if stats.Detail == "" {
			stats.Detail = "redis hot cache (postgres is source of truth)"
		} else {
			stats.Detail = "redis hot cache; " + stats.Detail
		}
		if n, err := r.store.CountAll(ctx); err == nil {
			stats.SourceMembers = n
			stats.HasSource = true
		} else {
			stats.Detail += "; postgres count: " + err.Error()
		}
		// Prefer durable segment list when hot cache is empty (e.g. mid-reset).
		if len(stats.SegmentNames) == 0 {
			if names, err := r.store.ListSegments(ctx); err == nil {
				stats.SegmentNames = names
				stats.Segments = int64(len(names))
			}
		}
		return stats, nil
	}
	return segment.StoreStats{Kind: "postgres_redis", OK: false, Detail: "cache does not expose stats"}, nil
}
