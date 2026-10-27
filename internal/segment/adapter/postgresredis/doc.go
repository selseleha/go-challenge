// Package postgresredis implements segment.Repository with Postgres as source of
// truth and Redis as the estimate cache.
//
// Design choice vs "write Redis then failure table":
// we use a transactional outbox in the same DB transaction as the membership
// write. A background Worker (in-process ticker) projects pending rows to Redis
// and reschedules failures with backoff — no separate failure table needed.
package postgresredis
