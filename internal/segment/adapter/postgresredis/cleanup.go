package postgresredis

import (
	"context"
	"log"
	"time"
)

// ProcessedDeleter removes already-projected outbox rows.
type ProcessedDeleter interface {
	DeleteProcessedOutbox(ctx context.Context, olderThan time.Time, limit int) (int64, error)
}

// Cleaner periodically deletes processed outbox rows so the table does not grow forever.
type Cleaner struct {
	store     ProcessedDeleter
	interval  time.Duration
	retention time.Duration
	batch     int
	now       func() time.Time
}

// NewCleaner creates an outbox retention worker.
func NewCleaner(store ProcessedDeleter, interval, retention time.Duration, batch int) *Cleaner {
	return &Cleaner{
		store:     store,
		interval:  interval,
		retention: retention,
		batch:     batch,
		now:       time.Now,
	}
}

// Run blocks until ctx is cancelled.
func (c *Cleaner) Run(ctx context.Context) {
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		if _, err := c.Tick(ctx); err != nil {
			log.Printf("outbox cleanup: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick deletes up to batch processed rows older than retention.
func (c *Cleaner) Tick(ctx context.Context) (int64, error) {
	cutoff := c.now().UTC().Add(-c.retention)
	n, err := c.store.DeleteProcessedOutbox(ctx, cutoff, c.batch)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		log.Printf("outbox cleanup: deleted %d processed rows older than %s", n, c.retention)
	}
	return n, nil
}

// SetNowForTest overrides the clock (tests only).
func (c *Cleaner) SetNowForTest(now func() time.Time) {
	if now != nil {
		c.now = now
	}
}
