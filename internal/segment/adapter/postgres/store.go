package postgres

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/selseleha/go-challenge/internal/segment"
)

//go:embed schema.sql
var schemaFS embed.FS

// OutboxItem is a pending Redis sync row.
type OutboxItem struct {
	ID       int64
	UserID   string
	Segment  string
	TaggedAt time.Time
	Attempts int
}

// Store is the Postgres persistence for memberships and the transactional outbox.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore connects and applies schema migrations.
func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) migrate(ctx context.Context) error {
	sql, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, string(sql))
	return err
}

// UpsertWithOutbox writes membership and outbox row in one transaction.
// Redis sync happens asynchronously via the outbox worker.
func (s *Store) UpsertWithOutbox(ctx context.Context, m segment.Membership) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO memberships (user_id, segment, tagged_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, segment)
		DO UPDATE SET tagged_at = EXCLUDED.tagged_at
	`, m.UserID, m.Segment, m.TaggedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert membership: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox (user_id, segment, tagged_at)
		VALUES ($1, $2, $3)
	`, m.UserID, m.Segment, m.TaggedAt.UTC())
	if err != nil {
		return fmt.Errorf("enqueue outbox: %w", err)
	}

	return tx.Commit(ctx)
}

// ListActive returns memberships still inside ttl at now (for Redis warm-up).
func (s *Store) ListActive(ctx context.Context, now time.Time, ttl time.Duration) ([]segment.Membership, error) {
	minTagged := now.Add(-ttl)
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, segment, tagged_at
		FROM memberships
		WHERE tagged_at > $1
	`, minTagged.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []segment.Membership
	for rows.Next() {
		var m segment.Membership
		if err := rows.Scan(&m.UserID, &m.Segment, &m.TaggedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountActiveSQL is a fallback estimate directly from Postgres.
func (s *Store) CountActiveSQL(ctx context.Context, name string, now time.Time, ttl time.Duration) (int64, error) {
	minTagged := now.Add(-ttl)
	var n int64
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM memberships
		WHERE segment = $1 AND tagged_at > $2
	`, name, minTagged.UTC()).Scan(&n)
	return n, err
}

// Claim returns up to limit unprocessed outbox rows (SKIP LOCKED).
func (s *Store) Claim(ctx context.Context, limit int) ([]OutboxItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, `
		WITH cte AS (
			SELECT id
			FROM outbox
			WHERE processed_at IS NULL AND available_at <= NOW()
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE outbox AS o
		SET attempts = o.attempts + 1
		FROM cte
		WHERE o.id = cte.id
		RETURNING o.id, o.user_id, o.segment, o.tagged_at, o.attempts
	`, limit)
	if err != nil {
		return nil, err
	}

	var items []OutboxItem
	for rows.Next() {
		var it OutboxItem
		if err := rows.Scan(&it.ID, &it.UserID, &it.Segment, &it.TaggedAt, &it.Attempts); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) MarkDone(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox SET processed_at = NOW(), last_error = NULL
		WHERE id = $1
	`, id)
	return err
}

func (s *Store) MarkRetry(ctx context.Context, id int64, cause error, next time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox
		SET last_error = $2, available_at = $3
		WHERE id = $1
	`, id, truncateErr(cause), next.UTC())
	return err
}

func truncateErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) > 500 {
		return s[:500]
	}
	return s
}
