package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/memory"
)

func TestRepository_CountActive_IgnoresExpiredAndCountsFresh(t *testing.T) {
	repo := memory.NewRepository(segment.DefaultMembershipTTL)
	ctx := context.Background()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	if err := repo.Upsert(ctx, segment.Membership{
		UserID: "fresh", Segment: "sports", TaggedAt: now.Add(-24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, segment.Membership{
		UserID: "expired", Segment: "sports", TaggedAt: now.Add(-segment.DefaultMembershipTTL),
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, segment.Membership{
		UserID: "other", Segment: "music", TaggedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.CountActive(ctx, "sports", now)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("CountActive = %d, want 1", got)
	}
}

func TestRepository_Upsert_RefreshesTaggedAt(t *testing.T) {
	repo := memory.NewRepository(segment.DefaultMembershipTTL)
	ctx := context.Background()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	if err := repo.Upsert(ctx, segment.Membership{
		UserID: "u1", Segment: "sports", TaggedAt: now.Add(-segment.DefaultMembershipTTL),
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, segment.Membership{
		UserID: "u1", Segment: "sports", TaggedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.CountActive(ctx, "sports", now)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("CountActive after refresh = %d, want 1", got)
	}
}

func TestRepository_CountActive_EmptySegment(t *testing.T) {
	repo := memory.NewRepository(segment.DefaultMembershipTTL)
	got, err := repo.CountActive(context.Background(), "missing", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("CountActive = %d, want 0", got)
	}
}
