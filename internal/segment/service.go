package segment

import "context"

// Service implements the inbound Ingester and Estimator ports.
type Service struct {
	repo  Repository
	clock Clock
}

// NewService wires outbound ports into the segment use cases.
func NewService(repo Repository, clock Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

var (
	_ Ingester  = (*Service)(nil)
	_ Estimator = (*Service)(nil)
)

func (s *Service) Ingest(ctx context.Context, userID, name string) error {
	return s.repo.Upsert(ctx, Membership{
		UserID:   userID,
		Segment:  name,
		TaggedAt: s.clock.Now(),
	})
}

func (s *Service) Estimate(ctx context.Context, name string) (int64, error) {
	return s.repo.CountActive(ctx, name, s.clock.Now())
}
