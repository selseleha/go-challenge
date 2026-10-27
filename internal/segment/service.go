package segment

import "context"

// Service implements ingest, estimate, health, and admin reset use cases.
type Service struct {
	repo      Repository
	clock     Clock
	ready     *Readiness
	hot       HotStoreControl
	storeName string
}

// NewService wires outbound ports into the segment use cases.
func NewService(repo Repository, clock Clock, ready *Readiness, hot HotStoreControl, storeName string) *Service {
	return &Service{
		repo:      repo,
		clock:     clock,
		ready:     ready,
		hot:       hot,
		storeName: storeName,
	}
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

func (s *Service) Health(context.Context) HealthInfo {
	return HealthInfo{Status: s.ready.Status(), Store: s.storeName}
}

// Reset clears the hot store. For postgres_redis, reload from Postgres runs asynchronously.
// Returns ErrResetInProgress if a reset is already running.
func (s *Service) Reset(ctx context.Context) error {
	if err := s.ready.BeginReset(); err != nil {
		return err
	}

	if err := s.hot.Flush(ctx); err != nil {
		s.ready.MarkReady()
		return err
	}

	if s.hot.AsyncWarm() {
		go func() {
			defer s.ready.MarkReady()
			_ = s.hot.WarmFromSource(context.Background(), s.clock.Now())
		}()
		return nil
	}

	if err := s.hot.WarmFromSource(ctx, s.clock.Now()); err != nil {
		s.ready.MarkReady()
		return err
	}
	s.ready.MarkReady()
	return nil
}
