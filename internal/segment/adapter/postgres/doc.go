// Package postgres stores memberships durably and uses a transactional outbox
// so Redis can be updated asynchronously without losing events on failure.
//
// Ingest path: BEGIN → upsert memberships → insert outbox → COMMIT.
// A worker later claims outbox rows and writes to Redis; failures stay in
// outbox with backoff (available_at) instead of a separate "failure" table.
//
// Why not GORM:
//   - only two tables and a few explicit SQL paths (upsert, SKIP LOCKED claim,
//     count-by-segment); an ORM adds little besides dependency weight
//   - outbox concurrency (FOR UPDATE SKIP LOCKED) stays clearer in raw SQL
//   - keeps the hexagonal boundary clean: no GORM models/tags leaking upward
//
// Scale notes (millions of users, hundreds of segments):
//   - hot estimates hit Redis (per-segment ZSET), not a full Postgres scan
//   - PK (user_id, segment) serves ingest upserts; (segment, tagged_at) serves
//     SQL count/warm fallbacks
//   - sharding is not required at challenge scale; natural partition is already
//     "one Redis key per segment". Postgres partitioning by segment or time
//     would be a later ops step if the history table grows without bound.
//   - processed outbox rows should be pruned periodically in production.
package postgres
