# Estimation Service (Go Challenge)

Answers: **how many users are in segment X?**  
A user stays in a segment for **two weeks** after being tagged; expired memberships are not counted.

![architecture](./Untitled%20Diagram.drawio.png)

## Quick start

```bash
# terminal 1 — Estimation Service
go run ./cmd/estimator

# terminal 2 — USS simulator (random ingest)
go run ./cmd/uss

# estimate
curl -s localhost:8080/v1/segments/sports/estimate
```

| Env | Default | Used by |
|-----|---------|---------|
| `HTTP_ADDR` | `:8080` | estimator (REST) |
| `GRPC_ADDR` | `:9090` / `localhost:9090` | estimator (listen) / uss (dial) |
| `WORKERS` | `8` | uss |
| `EVENTS` | `1000` | uss |

```bash
go test ./...
```

## API

### Ingest (gRPC) — from USS

- Service: `segment.v1.MembershipService`
- RPC: `Ingest({ user_id, segment }) → {}`
- Proto: [`api/proto/segment/v1/membership.proto`](api/proto/segment/v1/membership.proto)
- Invalid input → `InvalidArgument`

### Estimate (REST)

```http
GET /v1/segments/{segment}/estimate
```

```json
{"segment":"sports","count":42}
```

- Empty segment → `400` `{"error":"segment is required"}`
- Unexpected failure → `500`

## Design choices

| Choice | Why |
|--------|-----|
| **gRPC for ingest** | Service-to-service from USS; explicit protobuf contract; easy to add streaming later under load |
| **REST for estimate** | Simple read; easy to call with curl/browser; cache-friendly |
| **Hexagonal + package-by-domain** | `internal/segment` owns model, ports, service; adapters under `adapter/{http,grpc,memory}` |
| **In-memory store (for now)** | Fast local/dev path behind `Repository`; can swap Redis/Postgres without touching HTTP/gRPC |
| **Storage shape** `segment → userID → taggedAt` | Ingest O(1); estimate scans one segment only — fits “millions of users, hundreds of segments” |
| **Injected `Clock`** | Two-week expiry is unit-testable without sleeping |
| **TDD** | Behaviour covered next to the code (`*_test.go`) |

Membership rule: active while `now - taggedAt < 14 days`. Re-ingest of the same user/segment refreshes the window.

## Layout

```
cmd/estimator/          # wires adapters + starts HTTP & gRPC
cmd/uss/                # load/simulator client
api/proto/segment/v1/   # protobuf source
internal/segment/       # domain + ports + service
  adapter/http/         # REST estimate
  adapter/grpc/         # gRPC ingest (+ gen/)
  adapter/memory/       # in-memory repository
```

## AI usage

See [`ai.md`](./ai.md). (`sample.ai.md` is the template only.)

## Original challenge brief

The upstream problem statement lives in git history / the fork source ([ArmanCreativeSolutions/go-challenge](https://github.com/ArmanCreativeSolutions/go-challenge)). This README focuses on the implemented solution.
