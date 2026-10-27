package segment_test

import (
	"context"
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/memory"
)

func TestService_Estimate_ExcludesExpiredMemberships(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	repo := memory.NewRepository(segment.DefaultMembershipTTL)
	svc := segment.NewService(repo, segment.FixedClock{T: now.Add(-segment.DefaultMembershipTTL)})

	ctx := context.Background()
	if err := svc.Ingest(ctx, "old", "sports"); err != nil {
		t.Fatal(err)
	}

	// Advance clock: previous ingest is now exactly expired; new ingest is fresh.
	svc = segment.NewService(repo, segment.FixedClock{T: now})
	if err := svc.Ingest(ctx, "fresh", "sports"); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Estimate(ctx, "sports")
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("Estimate = %d, want 1 (only fresh)", got)
	}
}
