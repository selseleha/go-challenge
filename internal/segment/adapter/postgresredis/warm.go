package postgresredis

import (
	"context"
	"fmt"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
)

// activePager loads active memberships page-by-page (keyset) so warm never
// holds millions of rows in one slice.
type activePager interface {
	ListActivePage(ctx context.Context, now time.Time, ttl time.Duration, afterUser, afterSegment string, limit int) ([]segment.Membership, error)
}

// warmPages streams active memberships into Redis in pageSize batches.
func warmPages(ctx context.Context, src activePager, cache RedisCache, now time.Time, ttl time.Duration, pageSize int) (int, error) {
	if pageSize <= 0 {
		pageSize = 50000
	}
	var (
		afterUser, afterSeg string
		total               int
	)
	for {
		page, err := src.ListActivePage(ctx, now, ttl, afterUser, afterSeg, pageSize)
		if err != nil {
			return total, fmt.Errorf("list active page: %w", err)
		}
		if len(page) == 0 {
			return total, nil
		}

		// Pipeline size: large enough for millions-scale warm, small enough for one RTT payload.
		const writeChunk = 5000
		for i := 0; i < len(page); i += writeChunk {
			end := i + writeChunk
			if end > len(page) {
				end = len(page)
			}
			chunk := page[i:end]
			errs := cache.UpsertBatch(ctx, chunk)
			for j, err := range errs {
				if err != nil {
					m := chunk[j]
					return total, fmt.Errorf("warm upsert %s/%s: %w", m.Segment, m.UserID, err)
				}
			}
		}

		total += len(page)
		last := page[len(page)-1]
		afterUser, afterSeg = last.UserID, last.Segment
		if len(page) < pageSize {
			return total, nil
		}
	}
}
