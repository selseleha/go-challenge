package memory

import (
	"context"
	"sync"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
)

// Repository is an in-memory outbound adapter for segment.Repository.
//
// Data is keyed segment → userID → taggedAt so ingest is O(1) and estimate
// scans only one segment (not all users). Suitable for local/dev and tests;
// production can use adapter/redis with the same access pattern.
type Repository struct {
	mu   sync.RWMutex
	ttl  time.Duration
	data map[string]map[string]time.Time
}

// NewRepository creates an empty in-memory membership store.
func NewRepository(ttl time.Duration) *Repository {
	return &Repository{
		ttl:  ttl,
		data: make(map[string]map[string]time.Time),
	}
}

func (r *Repository) Upsert(_ context.Context, m segment.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, ok := r.data[m.Segment]
	if !ok {
		users = make(map[string]time.Time)
		r.data[m.Segment] = users
	}
	users[m.UserID] = m.TaggedAt
	return nil
}

func (r *Repository) CountActive(_ context.Context, name string, now time.Time) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := r.data[name]
	var n int64
	for _, taggedAt := range users {
		if (segment.Membership{TaggedAt: taggedAt}).Active(now, r.ttl) {
			n++
		}
	}
	return n, nil
}
