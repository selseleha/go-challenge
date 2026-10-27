// Package grpc is the inbound gRPC adapter for ingesting (user_id, segment) pairs from USS.
//
// Contract: segment.v1.MembershipService / Ingest
// Validation failures return codes.InvalidArgument; unexpected service errors return codes.Internal.
//
// Handlers stay thin: validate input, call segment.Ingester, map results to gRPC status codes.
//
// Protobuf sources live under api/proto; generated stubs are in ./gen. Regenerate from repo root:
//
//	protoc -I api/proto \
//	  --go_out=. --go_opt=module=github.com/selseleha/go-challenge \
//	  --go-grpc_out=. --go-grpc_opt=module=github.com/selseleha/go-challenge \
//	  api/proto/segment/v1/membership.proto
package grpc
