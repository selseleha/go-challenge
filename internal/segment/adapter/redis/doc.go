// Package redis is an outbound segment.Repository backed by Redis.
//
// Why Redis (vs Postgres for this challenge):
//   - memberships are ephemeral (two-week TTL) — Redis sorted sets + score-based
//     ranges match that model better than relational rows
//   - ingest and per-segment counts stay O(log N) for the segment, not full scans
//   - expired members are pruned on read (ZREMRANGEBYSCORE)
//   - Upsert uses ZADD GT so a stale outbox replay cannot regress taggedAt
//
// Swap this adapter for memory/Postgres at the composition root without changing
// HTTP/gRPC or domain code.
package redis
