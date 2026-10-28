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

// StoreStats is a snapshot of the hot-store for health dashboards.
type StoreStats struct {
	Kind          string   // memory | redis | postgres_redis
	OK            bool     // connectivity / usability
	LatencyMS     float64  // ping or local probe
	Segments      int64    // number of segment keys/groups
	Members       int64    // total memberships in hot store (redis/memory)
	SourceMembers int64    // durable source count (postgres); meaningful when HasSource
	HasSource     bool     // true when SourceMembers is from a durable store
	SegmentNames  []string // known segment names for per-segment charts
	Detail        string   // short human note (error text, etc.)
}

// StoreStatser gathers hot-store diagnostics for health.
type StoreStatser interface {
	Stats(ctx context.Context) (StoreStats, error)
}

// ProductConfig is the product-facing runtime settings shown on the health dashboard.
type ProductConfig struct {
	Store              string
	MembershipTTL      string // e.g. "14d" or "3h0m0s"
	OutboxBatchSize    int
	OutboxPollInterval string
	WarmPageSize       int
	CleanupRetention   string
}

// HealthInfo is returned by the health use case (HTTP maps it to JSON/HTML).
type HealthInfo struct {
	Status     string
	Store      string
	Product    ProductConfig
	Hot        StoreStats
	Uptime     time.Duration
	GoRoutines int
	AllocMiB   float64
	SysMiB     float64
	NumGC      uint32
	ProbeMS    float64 // timed estimate against a synthetic segment
}

// HealthChecker reports process/store readiness and diagnostics.
type HealthChecker interface {
	Health(ctx context.Context) HealthInfo
}

// Resetter rebuilds or clears the hot store via an admin API.
type Resetter interface {
	Reset(ctx context.Context) error
}

// HotStoreControl is the outbound port used by admin reset.
// memory/redis: Flush clears hot data; WarmFromSource is a no-op.
// postgres_redis: pauses outbox, discards pending rows, flushes Redis, warms from
// memberships, then resumes outbox — so rebuild does not race the worker.
type HotStoreControl interface {
	Flush(ctx context.Context) error
	WarmFromSource(ctx context.Context, now time.Time) error
	// BeforeWarm runs after Flush (pause worker + discard pending outbox).
	BeforeWarm(ctx context.Context) error
	// AfterWarm runs when warm finishes (resume worker).
	AfterWarm(ctx context.Context) error
	// AsyncWarm reports whether WarmFromSource should run in a background goroutine.
	AsyncWarm() bool
}
