package segment_test

import (
	"testing"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
)

func TestMembership_Active(t *testing.T) {
	taggedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
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
		{
			name: "just tagged is active",
			now:  taggedAt,
			want: true,
		},
		{
			name: "one day later is active",
			now:  taggedAt.Add(24 * time.Hour),
			want: true,
		},
		{
			name: "just under two weeks is active",
			now:  taggedAt.Add(segment.MembershipTTL - time.Nanosecond),
			want: true,
		},
		{
			name: "exactly two weeks is expired",
			now:  taggedAt.Add(segment.MembershipTTL),
			want: false,
		},
		{
			name: "after two weeks is expired",
			now:  taggedAt.Add(segment.MembershipTTL + time.Hour),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Active(tt.now); got != tt.want {
				t.Fatalf("Active(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}
