package postgresredis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/postgres"
	"github.com/selseleha/go-challenge/internal/segment/adapter/postgresredis"
)

type stubOutbox struct {
	items   []postgres.OutboxItem
	done    []int64
	retries []retryCall
}

type retryCall struct {
	id   int64
	next time.Time
	err  string
}

func (s *stubOutbox) Claim(context.Context, int) ([]postgres.OutboxItem, error) {
	out := s.items
	s.items = nil
	return out, nil
}

func (s *stubOutbox) MarkDone(_ context.Context, id int64) error {
	s.done = append(s.done, id)
	return nil
}

func (s *stubOutbox) MarkDoneBatch(_ context.Context, ids []int64) error {
	s.done = append(s.done, ids...)
	return nil
}

func (s *stubOutbox) MarkRetry(_ context.Context, id int64, cause error, next time.Time) error {
	s.retries = append(s.retries, retryCall{id: id, next: next, err: cause.Error()})
	return nil
}

type stubCache struct {
	err     error
	upserts []segment.Membership
	batches int
}

func (s *stubCache) Upsert(_ context.Context, m segment.Membership) error {
	s.upserts = append(s.upserts, m)
	return s.err
}

func (s *stubCache) UpsertBatch(_ context.Context, ms []segment.Membership) []error {
	s.batches++
	errs := make([]error, len(ms))
	for i, m := range ms {
		s.upserts = append(s.upserts, m)
		errs[i] = s.err
	}
	return errs
}

func (s *stubCache) CountActive(context.Context, string, time.Time) (int64, error) {
	return 0, nil
}

func (s *stubCache) Flush(context.Context) error { return nil }

func TestWorker_Tick_MarksDoneOnSuccess(t *testing.T) {
	outbox := &stubOutbox{items: []postgres.OutboxItem{{
		ID: 1, UserID: "u1", Segment: "sports", TaggedAt: time.Unix(100, 0).UTC(), Attempts: 1,
	}}}
	cache := &stubCache{}
	w := postgresredis.NewWorker(outbox, cache, time.Second, 10)

	n, err := w.Tick(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("claimed = %d", n)
	}
	if len(cache.upserts) != 1 || cache.upserts[0].UserID != "u1" {
		t.Fatalf("upserts = %+v", cache.upserts)
	}
	if len(outbox.done) != 1 || outbox.done[0] != 1 {
		t.Fatalf("done = %v", outbox.done)
	}
	if cache.batches != 1 {
		t.Fatalf("expected one UpsertBatch, got %d", cache.batches)
	}
}

func TestWorker_Tick_BatchesMultipleDone(t *testing.T) {
	outbox := &stubOutbox{items: []postgres.OutboxItem{
		{ID: 1, UserID: "u1", Segment: "sports", TaggedAt: time.Unix(1, 0).UTC()},
		{ID: 2, UserID: "u2", Segment: "sports", TaggedAt: time.Unix(2, 0).UTC()},
		{ID: 3, UserID: "u3", Segment: "music", TaggedAt: time.Unix(3, 0).UTC()},
	}}
	cache := &stubCache{}
	w := postgresredis.NewWorker(outbox, cache, time.Second, 10)
	n, err := w.Tick(context.Background())
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if cache.batches != 1 {
		t.Fatalf("batches = %d", cache.batches)
	}
	if len(outbox.done) != 3 {
		t.Fatalf("done = %v", outbox.done)
	}
}

func TestWorker_Tick_PausedSkipsClaim(t *testing.T) {
	outbox := &stubOutbox{items: []postgres.OutboxItem{{ID: 1, UserID: "u1", Segment: "sports"}}}
	w := postgresredis.NewWorker(outbox, &stubCache{}, time.Second, 10)
	w.Pause()
	n, err := w.Tick(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(outbox.items) != 1 {
		t.Fatal("should not claim while paused")
	}
}

func TestWorker_Tick_RetriesOnRedisFailure(t *testing.T) {
	outbox := &stubOutbox{items: []postgres.OutboxItem{{
		ID: 7, UserID: "u7", Segment: "music", TaggedAt: time.Unix(200, 0).UTC(), Attempts: 3,
	}}}
	cache := &stubCache{err: errors.New("redis down")}
	w := postgresredis.NewWorker(outbox, cache, time.Second, 10)

	if _, err := w.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(outbox.done) != 0 {
		t.Fatalf("done should be empty, got %v", outbox.done)
	}
	if len(outbox.retries) != 1 || outbox.retries[0].id != 7 {
		t.Fatalf("retries = %+v", outbox.retries)
	}
	if outbox.retries[0].err != "redis down" {
		t.Fatalf("retry err = %q", outbox.retries[0].err)
	}
}
