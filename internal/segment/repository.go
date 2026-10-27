package segment

import (
	"context"
	"time"
)

// Repository is the outbound port for persisting segment memberships.
type Repository interface {
	// Upsert stores or refreshes a membership (re-tag resets the two-week window).
	Upsert(ctx context.Context, m Membership) error
	// CountActive returns how many memberships in segment are still active at now.
	CountActive(ctx context.Context, segment string, now time.Time) (int64, error)
}
