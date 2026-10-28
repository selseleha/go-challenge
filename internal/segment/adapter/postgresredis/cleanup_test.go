package postgresredis_test

import (
	"context"
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment/adapter/postgresredis"
)

type stubDeleter struct {
	cutoff time.Time
	limit  int
	n      int64
	err    error
}

func (s *stubDeleter) DeleteProcessedOutbox(_ context.Context, olderThan time.Time, limit int) (int64, error) {
	s.cutoff = olderThan
	s.limit = limit
	return s.n, s.err
}

func TestCleaner_Tick_DeletesOlderThanRetention(t *testing.T) {
	del := &stubDeleter{n: 42}
	c := postgresredis.NewCleaner(del, time.Minute, time.Hour, 5000)
	fixed := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	c.SetNowForTest(func() time.Time { return fixed })

	n, err := c.Tick(context.Background())
	if err != nil || n != 42 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	wantCutoff := fixed.Add(-time.Hour)
	if !del.cutoff.Equal(wantCutoff) || del.limit != 5000 {
		t.Fatalf("cutoff=%v limit=%d", del.cutoff, del.limit)
	}
}
