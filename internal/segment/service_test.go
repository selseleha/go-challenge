package segment_test

import (
	"context"
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
)

type stubRepo struct {
	upserted []segment.Membership
	count    int64
	countSeg string
	countAt  time.Time
	err      error
}

func (s *stubRepo) Upsert(_ context.Context, m segment.Membership) error {
	s.upserted = append(s.upserted, m)
	return s.err
}

func (s *stubRepo) CountActive(_ context.Context, name string, now time.Time) (int64, error) {
	s.countSeg = name
	s.countAt = now
	return s.count, s.err
}

func TestService_Ingest_UsesClockAndUpserts(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubRepo{}
	svc := segment.NewService(repo, segment.FixedClock{T: now})

	if err := svc.Ingest(context.Background(), "u104010", "sports"); err != nil {
		t.Fatal(err)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("upserts = %d, want 1", len(repo.upserted))
	}
	got := repo.upserted[0]
	if got.UserID != "u104010" || got.Segment != "sports" || !got.TaggedAt.Equal(now) {
		t.Fatalf("upserted = %+v", got)
	}
}

func TestService_Estimate_DelegatesToRepoWithNow(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubRepo{count: 7}
	svc := segment.NewService(repo, segment.FixedClock{T: now})

	got, err := svc.Estimate(context.Background(), "sports")
	if err != nil {
		t.Fatal(err)
	}
	if got != 7 {
		t.Fatalf("Estimate = %d, want 7", got)
	}
	if repo.countSeg != "sports" || !repo.countAt.Equal(now) {
		t.Fatalf("repo got segment=%q at=%v", repo.countSeg, repo.countAt)
	}
}
