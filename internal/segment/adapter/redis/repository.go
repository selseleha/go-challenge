package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/selseleha/go-challenge/internal/segment"
)

// Repository persists memberships in Redis sorted sets.
//
// Key: segment:{name}
// Member: userID
// Score: taggedAt as Unix nanoseconds
//
// CountActive uses ZCOUNT for members with score in (now-TTL, +inf], then
// ZREMRANGEBYSCORE to drop expired entries so the key does not grow forever.
type Repository struct {
	client goredis.Cmdable
	ttl    time.Duration
}

// NewRepository wraps a Redis client (or cluster/compatible Cmdable).
func NewRepository(client goredis.Cmdable, ttl time.Duration) *Repository {
	return &Repository{client: client, ttl: ttl}
}

func (r *Repository) Upsert(ctx context.Context, m segment.Membership) error {
	score := float64(m.TaggedAt.UTC().UnixNano())
	return r.client.ZAdd(ctx, key(m.Segment), goredis.Z{
		Score:  score,
		Member: m.UserID,
	}).Err()
}

// Flush implements segment.HotStoreControl (and clears the Redis DB).
func (r *Repository) Flush(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *Repository) WarmFromSource(context.Context, time.Time) error { return nil }

func (r *Repository) AsyncWarm() bool { return false }

func (r *Repository) CountActive(ctx context.Context, name string, now time.Time) (int64, error) {
	k := key(name)
	minActive := now.Add(-r.ttl).UTC().UnixNano()
	// Active means taggedAt > now-TTL (same as now.Sub(taggedAt) < TTL).
	minScore := strconv.FormatInt(minActive, 10)
	maxScore := "+inf"

	n, err := r.client.ZCount(ctx, k, "("+minScore, maxScore).Result()
	if err != nil {
		return 0, err
	}

	// Drop expired members: score <= now-TTL
	_ = r.client.ZRemRangeByScore(ctx, k, "-inf", minScore).Err()

	return n, nil
}

func key(segment string) string {
	return fmt.Sprintf("segment:%s", segment)
}
