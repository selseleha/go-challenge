package postgresredis

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/postgres"
)

// Outbox is the durable queue of Redis projections.
type Outbox interface {
	Claim(ctx context.Context, limit int) ([]postgres.OutboxItem, error)
	MarkDone(ctx context.Context, id int64) error
	MarkDoneBatch(ctx context.Context, ids []int64) error
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
	paused   atomic.Bool
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

// Pause stops claiming outbox rows (used during cache reset).
func (w *Worker) Pause() { w.paused.Store(true) }

// Resume allows claiming again after reset warm-up.
func (w *Worker) Resume() { w.paused.Store(false) }

// Run blocks until ctx is cancelled.
// When a tick finds work it loops immediately to drain faster; otherwise it waits.
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		if w.paused.Load() {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				continue
			}
		}

		n, err := w.Tick(ctx)
		if err != nil {
			log.Printf("outbox tick: %v", err)
		}
		if n > 0 {
			// Keep draining without waiting the full poll interval.
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick claims a batch, pipelines Redis writes, then marks successes done in one SQL.
func (w *Worker) Tick(ctx context.Context) (int, error) {
	if w.paused.Load() {
		return 0, nil
	}
	items, err := w.outbox.Claim(ctx, w.batch)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	ms := make([]segment.Membership, len(items))
	for i, it := range items {
		ms[i] = segment.Membership{
			UserID:   it.UserID,
			Segment:  it.Segment,
			TaggedAt: it.TaggedAt,
		}
	}
	errs := w.cache.UpsertBatch(ctx, ms)

	done := make([]int64, 0, len(items))
	for i, it := range items {
		if errs[i] != nil {
			next := w.now().UTC().Add(backoff(it.Attempts))
			if markErr := w.outbox.MarkRetry(ctx, it.ID, errs[i], next); markErr != nil {
				log.Printf("outbox mark retry id=%d: %v", it.ID, markErr)
			}
			continue
		}
		done = append(done, it.ID)
	}
	if len(done) > 0 {
		if err := w.outbox.MarkDoneBatch(ctx, done); err != nil {
			log.Printf("outbox mark done batch (%d): %v", len(done), err)
		}
	}
	return len(items), nil
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
