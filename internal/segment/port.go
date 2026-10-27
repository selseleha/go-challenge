package segment

import (
	"context"
	"time"
)

// Estimator is the inbound port used by the HTTP estimate adapter.
type Estimator interface {
	Estimate(ctx context.Context, segment string) (count int64, err error)
}

// Ingester is the inbound port used by the gRPC ingest adapter.
type Ingester interface {
	Ingest(ctx context.Context, userID, segment string) error
}

// HealthInfo is returned by the health use case (HTTP maps it to a response DTO).
type HealthInfo struct {
	Status string
	Store  string
}

// HealthChecker reports process/store readiness.
type HealthChecker interface {
	Health(ctx context.Context) HealthInfo
}

// Resetter rebuilds or clears the hot store via an admin API.
type Resetter interface {
	Reset(ctx context.Context) error
}

// HotStoreControl is the outbound port used by admin reset.
// memory/redis: Flush clears hot data; WarmFromSource is a no-op.
// postgres_redis: Flush clears Redis; WarmFromSource reloads active rows from Postgres.
type HotStoreControl interface {
	Flush(ctx context.Context) error
	WarmFromSource(ctx context.Context, now time.Time) error
	// AsyncWarm reports whether WarmFromSource should run in a background goroutine.
	AsyncWarm() bool
}
