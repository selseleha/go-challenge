package segment

import (
	"context"
	"log"
	"runtime"
	"time"
)

// Service implements ingest, estimate, health, and admin reset use cases.
type Service struct {
	repo      Repository
	clock     Clock
	ready     *Readiness
	hot       HotStoreControl
	stats     StoreStatser
	storeName string
	product   ProductConfig
	startedAt time.Time
}

// NewService wires outbound ports into the segment use cases.
func NewService(repo Repository, clock Clock, ready *Readiness, hot HotStoreControl, stats StoreStatser, storeName string) *Service {
	return &Service{
		repo:      repo,
		clock:     clock,
		ready:     ready,
		hot:       hot,
		stats:     stats,
		storeName: storeName,
		product:   ProductConfig{Store: storeName},
		startedAt: time.Now(),
	}
}

// WithProduct attaches product config shown on the health dashboard.
func (s *Service) WithProduct(p ProductConfig) *Service {
	if p.Store == "" {
		p.Store = s.storeName
	}
	s.product = p
	return s
}

var (
	_ Ingester      = (*Service)(nil)
	_ Estimator     = (*Service)(nil)
	_ HealthChecker = (*Service)(nil)
	_ Resetter      = (*Service)(nil)
)

func (s *Service) Ingest(ctx context.Context, userID, name string) error {
	if !s.ready.IsReady() {
		return ErrNotReady
	}
	return s.repo.Upsert(ctx, Membership{
		UserID:   userID,
		Segment:  name,
		TaggedAt: s.clock.Now(),
	})
}

func (s *Service) Estimate(ctx context.Context, name string) (int64, error) {
	if !s.ready.IsReady() {
		return 0, ErrNotReady
	}
	return s.repo.CountActive(ctx, name, s.clock.Now())
}

func (s *Service) Health(ctx context.Context) HealthInfo {
	info := HealthInfo{
		Status:     s.ready.Status(),
		Store:      s.storeName,
		Product:    s.product,
		Uptime:     time.Since(s.startedAt),
		GoRoutines: runtime.NumGoroutine(),
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	info.AllocMiB = float64(mem.Alloc) / (1024 * 1024)
	info.SysMiB = float64(mem.Sys) / (1024 * 1024)
	info.NumGC = mem.NumGC

	if s.stats != nil {
		if st, err := s.stats.Stats(ctx); err == nil {
			info.Hot = st
		} else {
			info.Hot = StoreStats{Kind: s.storeName, OK: false, Detail: err.Error()}
		}
	}

	start := time.Now()
	_, _ = s.repo.CountActive(ctx, "__health_probe__", s.clock.Now())
	info.ProbeMS = float64(time.Since(start).Microseconds()) / 1000.0

	return info
}

// Reset clears the hot store. For postgres_redis, reload from Postgres runs asynchronously.
// Returns ErrResetInProgress if a reset is already running.
func (s *Service) Reset(ctx context.Context) error {
	if err := s.ready.BeginReset(); err != nil {
		return err
	}

	// Pause outbox / discard stale projections before touching Redis so the
	// worker cannot race Flush or WarmFromSource.
	if err := s.hot.BeforeWarm(ctx); err != nil {
		_ = s.hot.AfterWarm(ctx)
		s.ready.MarkReady()
		return err
	}
	if err := s.hot.Flush(ctx); err != nil {
		_ = s.hot.AfterWarm(ctx)
		s.ready.MarkReady()
		return err
	}

	if s.hot.AsyncWarm() {
		go func() {
			defer func() {
				_ = s.hot.AfterWarm(context.Background())
				// TODO(ops): on warm failure keep status=resetting (or a dedicated
				// degraded state) instead of MarkReady, so estimate cannot return
				// undercounts with HTTP 200. For now we only log and become ready.
				s.ready.MarkReady()
			}()
			if err := s.hot.WarmFromSource(context.Background(), s.clock.Now()); err != nil {
				log.Printf("async warm failed (system marked ready anyway; see TODO): %v", err)
			}
		}()
		return nil
	}

	if err := s.hot.WarmFromSource(ctx, s.clock.Now()); err != nil {
		_ = s.hot.AfterWarm(ctx)
		s.ready.MarkReady()
		return err
	}
	_ = s.hot.AfterWarm(ctx)
	s.ready.MarkReady()
	return nil
}
