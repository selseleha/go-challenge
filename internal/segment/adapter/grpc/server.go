package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/selseleha/go-challenge/internal/segment"
	"github.com/selseleha/go-challenge/internal/segment/adapter/grpc/gen"
)

// Server is the inbound gRPC adapter for ingesting memberships from USS.
type Server struct {
	gen.UnimplementedMembershipServiceServer
	ingester segment.Ingester
}

// NewServer wires an Ingester into the MembershipService gRPC API.
func NewServer(ingester segment.Ingester) *Server {
	return &Server{ingester: ingester}
}

// Register attaches this server to a gRPC ServiceRegistrar (e.g. *grpc.Server).
func (s *Server) Register(r grpc.ServiceRegistrar) {
	gen.RegisterMembershipServiceServer(r, s)
}

// Ensure Server implements the generated service interface at compile time.
var _ gen.MembershipServiceServer = (*Server)(nil)

func (s *Server) Ingest(ctx context.Context, req *gen.IngestRequest) (*gen.IngestResponse, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	name := strings.TrimSpace(req.GetSegment())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "segment is required")
	}

	if err := s.ingester.Ingest(ctx, userID, name); err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &gen.IngestResponse{}, nil
}
