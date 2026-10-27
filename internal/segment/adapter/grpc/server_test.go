package grpc_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	segmentgrpc "github.com/selseleha/go-challenge/internal/segment/adapter/grpc"
	"github.com/selseleha/go-challenge/internal/segment/adapter/grpc/gen"
)

type stubIngester struct {
	userID  string
	segment string
	err     error
	called  bool
}

func (s *stubIngester) Ingest(_ context.Context, userID, segment string) error {
	s.called = true
	s.userID = userID
	s.segment = segment
	return s.err
}

func TestServer_Ingest_Success(t *testing.T) {
	ing := &stubIngester{}
	srv := segmentgrpc.NewServer(ing)

	resp, err := srv.Ingest(context.Background(), &gen.IngestRequest{
		UserId:  "u104010",
		Segment: "sports",
	})
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !ing.called || ing.userID != "u104010" || ing.segment != "sports" {
		t.Fatalf("ingester got user=%q segment=%q called=%v", ing.userID, ing.segment, ing.called)
	}
}

func TestServer_Ingest_RejectsMissingUserID(t *testing.T) {
	ing := &stubIngester{}
	srv := segmentgrpc.NewServer(ing)

	_, err := srv.Ingest(context.Background(), &gen.IngestRequest{Segment: "sports"})
	assertInvalidArgument(t, err, "user_id is required")
	if ing.called {
		t.Fatal("ingester should not be called")
	}
}

func TestServer_Ingest_RejectsMissingSegment(t *testing.T) {
	ing := &stubIngester{}
	srv := segmentgrpc.NewServer(ing)

	_, err := srv.Ingest(context.Background(), &gen.IngestRequest{UserId: "u104010"})
	assertInvalidArgument(t, err, "segment is required")
	if ing.called {
		t.Fatal("ingester should not be called")
	}
}

func TestServer_Ingest_RejectsWhitespaceOnlyFields(t *testing.T) {
	ing := &stubIngester{}
	srv := segmentgrpc.NewServer(ing)

	_, err := srv.Ingest(context.Background(), &gen.IngestRequest{
		UserId:  "  ",
		Segment: "sports",
	})
	assertInvalidArgument(t, err, "user_id is required")
	if ing.called {
		t.Fatal("ingester should not be called")
	}
}

func TestServer_Ingest_MapsServiceErrorToInternal(t *testing.T) {
	ing := &stubIngester{err: errors.New("boom")}
	srv := segmentgrpc.NewServer(ing)

	_, err := srv.Ingest(context.Background(), &gen.IngestRequest{
		UserId:  "u104010",
		Segment: "sports",
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("code = %v, want Internal", st.Code())
	}
}

func assertInvalidArgument(t *testing.T, err error, wantMsg string) {
	t.Helper()
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", st.Code())
	}
	if st.Message() != wantMsg {
		t.Fatalf("message = %q, want %q", st.Message(), wantMsg)
	}
}
