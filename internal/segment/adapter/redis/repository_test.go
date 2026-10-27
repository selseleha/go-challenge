package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/selseleha/go-challenge/internal/segment"
	segmentredis "github.com/selseleha/go-challenge/internal/segment/adapter/redis"
)

func newTestRepo(t *testing.T) (*segmentredis.Repository, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return segmentredis.NewRepository(client), mr
}

func TestRepository_CountActive_IgnoresExpiredAndCountsFresh(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	mustUpsert(t, repo, segment.Membership{
		UserID: "fresh", Segment: "sports", TaggedAt: now.Add(-24 * time.Hour),
	})
	mustUpsert(t, repo, segment.Membership{
		UserID: "expired", Segment: "sports", TaggedAt: now.Add(-segment.MembershipTTL),
	})
	mustUpsert(t, repo, segment.Membership{
		UserID: "other", Segment: "music", TaggedAt: now,
	})

	got, err := repo.CountActive(ctx, "sports", now)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("CountActive = %d, want 1", got)
	}
}

func TestRepository_Upsert_RefreshesTaggedAt(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	mustUpsert(t, repo, segment.Membership{
		UserID: "u1", Segment: "sports", TaggedAt: now.Add(-segment.MembershipTTL),
	})
	mustUpsert(t, repo, segment.Membership{
		UserID: "u1", Segment: "sports", TaggedAt: now,
	})

	got, err := repo.CountActive(ctx, "sports", now)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("CountActive after refresh = %d, want 1", got)
	}
}

func TestRepository_CountActive_EmptySegment(t *testing.T) {
	repo, _ := newTestRepo(t)
	got, err := repo.CountActive(context.Background(), "missing", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("CountActive = %d, want 0", got)
	}
}

func TestRepository_CountActive_RemovesExpiredMembers(t *testing.T) {
	repo, mr := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	mustUpsert(t, repo, segment.Membership{
		UserID: "expired", Segment: "sports", TaggedAt: now.Add(-segment.MembershipTTL),
	})

	if _, err := repo.CountActive(ctx, "sports", now); err != nil {
		t.Fatal(err)
	}

	if mr.Exists("segment:sports") {
		members, err := mr.ZMembers("segment:sports")
		if err != nil {
			t.Fatal(err)
		}
		if len(members) != 0 {
			t.Fatalf("members after cleanup = %v, want empty", members)
		}
	}
}

func mustUpsert(t *testing.T, repo *segmentredis.Repository, m segment.Membership) {
	t.Helper()
	if err := repo.Upsert(context.Background(), m); err != nil {
		t.Fatal(err)
	}
}
