package segment

import "context"

// Estimator is the inbound port used by the HTTP estimate adapter.
type Estimator interface {
	Estimate(ctx context.Context, segment string) (count int64, err error)
}

// Ingester is the inbound port used by the gRPC ingest adapter.
type Ingester interface {
	Ingest(ctx context.Context, userID, segment string) error
}
