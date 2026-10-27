package postgresredis

import (
	"context"
	"log"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/postgres"
)

// Outbox is the durable queue of Redis projections.
type Outbox interface {
	Claim(ctx context.Context, limit int) ([]postgres.OutboxItem, error)
	MarkDone(ctx context.Context, id int64) error
	MarkRetry(ctx context.Context, id int64, cause error, next time.Time) error
}

// Worker polls the outbox and projects rows into Redis.
// This replaces an external cron: same process, ticker-based retries with backoff.
type Worker struct {
	outbox   Outbox
	cache    RedisCache
	interval time.Duration
	batch    int
	now      func() time.Time
}

// NewWorker creates an outbox → Redis projector.
func NewWorker(outbox Outbox, cache RedisCache, interval time.Duration, batch int) *Worker {
	return &Worker{
		outbox:   outbox,
		cache:    cache,
		interval: interval,
		batch:    batch,
		now:      time.Now,
	}
}

// Run blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		if err := w.Tick(ctx); err != nil {
			log.Printf("outbox tick: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick claims a batch and attempts Redis upserts.
func (w *Worker) Tick(ctx context.Context) error {
	items, err := w.outbox.Claim(ctx, w.batch)
	if err != nil {
		return err
	}
	for _, it := range items {
		m := segment.Membership{
			UserID:   it.UserID,
			Segment:  it.Segment,
			TaggedAt: it.TaggedAt,
		}
		if err := w.cache.Upsert(ctx, m); err != nil {
			next := w.now().UTC().Add(backoff(it.Attempts))
			if markErr := w.outbox.MarkRetry(ctx, it.ID, err, next); markErr != nil {
				log.Printf("outbox mark retry id=%d: %v", it.ID, markErr)
			}
			continue
		}
		if err := w.outbox.MarkDone(ctx, it.ID); err != nil {
			log.Printf("outbox mark done id=%d: %v", it.ID, err)
		}
	}
	return nil
}

func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	d := time.Duration(attempts) * time.Second
	if d > 1*time.Minute {
		return 1 * time.Minute
	}
	return d
}
