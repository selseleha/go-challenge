package postgresredis

import (
	"context"
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
)

type pageSource struct {
	pages [][]segment.Membership
	i     int
	calls int
}

func (p *pageSource) ListActivePage(context.Context, time.Time, time.Duration, string, string, int) ([]segment.Membership, error) {
	p.calls++
	if p.i >= len(p.pages) {
		return nil, nil
	}
	out := p.pages[p.i]
	p.i++
	return out, nil
}

type warmCache struct {
	batches int
	upserts []segment.Membership
}

func (c *warmCache) Upsert(context.Context, segment.Membership) error { return nil }

func (c *warmCache) UpsertBatch(_ context.Context, ms []segment.Membership) []error {
	c.batches++
	c.upserts = append(c.upserts, ms...)
	return make([]error, len(ms))
}

func (c *warmCache) CountActive(context.Context, string, time.Time) (int64, error) { return 0, nil }

func (c *warmCache) Flush(context.Context) error { return nil }

func TestWarmPages_StreamsPages(t *testing.T) {
	src := &pageSource{pages: [][]segment.Membership{
		{
			{UserID: "a", Segment: "sports", TaggedAt: time.Unix(1, 0)},
			{UserID: "b", Segment: "sports", TaggedAt: time.Unix(2, 0)},
		},
		{
			{UserID: "c", Segment: "music", TaggedAt: time.Unix(3, 0)},
		},
	}}
	cache := &warmCache{}
	n, err := warmPages(context.Background(), src, cache, time.Now(), time.Hour, 2)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("warmed = %d", n)
	}
	if src.calls != 2 {
		t.Fatalf("calls = %d, want 2", src.calls)
	}
	if cache.batches != 2 || len(cache.upserts) != 3 {
		t.Fatalf("batches=%d upserts=%d", cache.batches, len(cache.upserts))
	}
}
