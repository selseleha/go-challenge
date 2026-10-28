package redis

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
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
	// GT: never let a stale outbox replay regress taggedAt after a newer write.
	return r.client.ZAddArgs(ctx, key(m.Segment), goredis.ZAddArgs{
		GT: true,
		Members: []goredis.Z{{
			Score:  float64(m.TaggedAt.UTC().UnixNano()),
			Member: m.UserID,
		}},
	}).Err()
}

// UpsertBatch pipelines ZADDs when the client supports Pipeline; otherwise falls back per row.
func (r *Repository) UpsertBatch(ctx context.Context, ms []segment.Membership) []error {
	errs := make([]error, len(ms))
	if len(ms) == 0 {
		return errs
	}
	type piper interface {
		Pipeline() goredis.Pipeliner
	}
	p, ok := r.client.(piper)
	if !ok {
		for i, m := range ms {
			errs[i] = r.Upsert(ctx, m)
		}
		return errs
	}
	pipe := p.Pipeline()
	cmds := make([]*goredis.IntCmd, len(ms))
	for i, m := range ms {
		cmds[i] = pipe.ZAddArgs(ctx, key(m.Segment), goredis.ZAddArgs{
			GT: true,
			Members: []goredis.Z{{
				Score:  float64(m.TaggedAt.UTC().UnixNano()),
				Member: m.UserID,
			}},
		})
	}
	_, _ = pipe.Exec(ctx)
	for i, cmd := range cmds {
		errs[i] = cmd.Err()
	}
	return errs
}

// Flush implements segment.HotStoreControl (and clears the Redis DB).
func (r *Repository) Flush(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *Repository) WarmFromSource(context.Context, time.Time) error { return nil }

func (r *Repository) BeforeWarm(context.Context) error { return nil }

func (r *Repository) AfterWarm(context.Context) error { return nil }

func (r *Repository) AsyncWarm() bool { return false }

func (r *Repository) Stats(ctx context.Context) (segment.StoreStats, error) {
	start := time.Now()
	if err := r.client.Ping(ctx).Err(); err != nil {
		return segment.StoreStats{Kind: "redis", OK: false, Detail: err.Error()}, nil
	}
	pingMS := float64(time.Since(start).Microseconds()) / 1000.0

	var members int64
	var names []string
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, "segment:*", 100).Result()
		if err != nil {
			return segment.StoreStats{Kind: "redis", OK: false, Detail: err.Error()}, nil
		}
		for _, k := range keys {
			name := strings.TrimPrefix(k, "segment:")
			if name == "" || name == k {
				continue
			}
			names = append(names, name)
			n, err := r.client.ZCard(ctx, k).Result()
			if err != nil {
				return segment.StoreStats{Kind: "redis", OK: false, Detail: err.Error()}, nil
			}
			members += n
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	slices.Sort(names)

	return segment.StoreStats{
		Kind:         "redis",
		OK:           true,
		LatencyMS:    pingMS,
		Segments:     int64(len(names)),
		Members:      members,
		SegmentNames: names,
		Detail:       "sorted sets segment:*",
	}, nil
}

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
