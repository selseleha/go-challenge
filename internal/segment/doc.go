// Package segment is the bounded context for user-segment membership and estimation.
//
// Inbound adapters:
//   - adapter/http — REST JSON API for estimating how many users are in a segment.
//     REST is used here because estimation is a simple, cache-friendly read that
//     operators and other services can call with curl/browsers without a gRPC client.
//   - adapter/grpc — gRPC API for ingesting (user_id, segment) pairs from USS.
//     gRPC is preferred for ingest because USS is a service-to-service producer;
//     protobuf contracts are explicit and streaming can be added later if volume grows.
//
// This package currently exposes only inbound ports (Estimator, Ingester). Domain
// logic, use cases, and outbound adapters are intentionally not implemented yet.
package segment
