package segment_test

import (
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
)

func TestMembership_Active(t *testing.T) {
	taggedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ttl := segment.DefaultMembershipTTL
	m := segment.Membership{
		UserID:   "u104010",
		Segment:  "sports",
		TaggedAt: taggedAt,
	}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "just tagged is active", now: taggedAt, want: true},
		{name: "one day later is active", now: taggedAt.Add(24 * time.Hour), want: true},
		{name: "just under ttl is active", now: taggedAt.Add(ttl - time.Nanosecond), want: true},
		{name: "exactly ttl is expired", now: taggedAt.Add(ttl), want: false},
		{name: "after ttl is expired", now: taggedAt.Add(ttl + time.Hour), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Active(tt.now, ttl); got != tt.want {
				t.Fatalf("Active(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestMembership_Active_RespectsCustomTTL(t *testing.T) {
	taggedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := segment.Membership{TaggedAt: taggedAt}
	ttl := 48 * time.Hour

	if !m.Active(taggedAt.Add(47*time.Hour), ttl) {
		t.Fatal("expected active before custom ttl")
	}
	if m.Active(taggedAt.Add(ttl), ttl) {
		t.Fatal("expected expired at custom ttl")
	}
}
