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

type stubHot struct {
	flushed bool
	warmed  bool
	before  bool
	after   bool
	async   bool
}

func (s *stubHot) Flush(context.Context) error {
	s.flushed = true
	return nil
}

func (s *stubHot) WarmFromSource(context.Context, time.Time) error {
	s.warmed = true
	return nil
}

func (s *stubHot) BeforeWarm(context.Context) error {
	s.before = true
	return nil
}

func (s *stubHot) AfterWarm(context.Context) error {
	s.after = true
	return nil
}

func (s *stubHot) AsyncWarm() bool { return s.async }

type stubStats struct{}

func (stubStats) Stats(context.Context) (segment.StoreStats, error) {
	return segment.StoreStats{Kind: "memory", OK: true}, nil
}

func newTestService(repo segment.Repository, hot segment.HotStoreControl) *segment.Service {
	return segment.NewService(repo, segment.FixedClock{T: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}, segment.NewReadiness(), hot, stubStats{}, "memory")
}

func TestService_Ingest_UsesClockAndUpserts(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubRepo{}
	svc := segment.NewService(repo, segment.FixedClock{T: now}, segment.NewReadiness(), &stubHot{}, stubStats{}, "memory")

	if err := svc.Ingest(context.Background(), "u104010", "sports"); err != nil {
		t.Fatal(err)
	}
	got := repo.upserted[0]
	if got.UserID != "u104010" || got.Segment != "sports" || !got.TaggedAt.Equal(now) {
		t.Fatalf("upserted = %+v", got)
	}
}

func TestService_Estimate_DelegatesToRepoWithNow(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubRepo{count: 7}
	svc := segment.NewService(repo, segment.FixedClock{T: now}, segment.NewReadiness(), &stubHot{}, stubStats{}, "memory")

	got, err := svc.Estimate(context.Background(), "sports")
	if err != nil {
		t.Fatal(err)
	}
	if got != 7 || repo.countSeg != "sports" || !repo.countAt.Equal(now) {
		t.Fatalf("got=%d repo=%q %v", got, repo.countSeg, repo.countAt)
	}
}

func TestService_Estimate_NotReady(t *testing.T) {
	ready := segment.NewReadiness()
	_ = ready.BeginReset()
	svc := segment.NewService(&stubRepo{count: 99}, segment.SystemClock{}, ready, &stubHot{}, stubStats{}, "redis")

	_, err := svc.Estimate(context.Background(), "sports")
	if err != segment.ErrNotReady {
		t.Fatalf("err = %v", err)
	}
}

func TestService_Reset_SyncMarksReady(t *testing.T) {
	hot := &stubHot{async: false}
	svc := newTestService(&stubRepo{}, hot)

	if err := svc.Reset(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !hot.flushed || !hot.warmed || !hot.before || !hot.after {
		t.Fatalf("hot = %+v", hot)
	}
	h := svc.Health(context.Background())
	if h.Status != segment.StatusReady || h.Store != "memory" {
		t.Fatalf("health = %+v", h)
	}
}

func TestService_Reset_AsyncStaysResettingUntilWarmDone(t *testing.T) {
	hot := &stubHot{async: true}
	block := make(chan struct{})
	hotBlocked := &blockingHot{stubHot: hot, block: block}
	ready := segment.NewReadiness()
	svc := segment.NewService(&stubRepo{}, segment.SystemClock{}, ready, hotBlocked, stubStats{}, "postgres_redis")

	if err := svc.Reset(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ready.Status() != segment.StatusResetting {
		t.Fatalf("status = %s", ready.Status())
	}
	close(block)
	// wait briefly for goroutine
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ready.IsReady() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected ready after async warm")
}

type blockingHot struct {
	*stubHot
	block <-chan struct{}
}

func (b *blockingHot) WarmFromSource(ctx context.Context, now time.Time) error {
	<-b.block
	return b.stubHot.WarmFromSource(ctx, now)
}

func TestService_Reset_RejectsConcurrent(t *testing.T) {
	ready := segment.NewReadiness()
	_ = ready.BeginReset()
	svc := segment.NewService(&stubRepo{}, segment.SystemClock{}, ready, &stubHot{}, stubStats{}, "memory")
	if err := svc.Reset(context.Background()); err != segment.ErrResetInProgress {
		t.Fatalf("err = %v", err)
	}
}
