# Estimation Service (Go Challenge)

Answers: **how many users are in segment X?**  
A user stays in a segment for **two weeks** after being tagged; expired memberships are not counted.

![architecture](./Untitled%20Diagram.drawio.png)

## Quick start

```bash
# full stack (Redis + estimator)
docker compose up --build

# or locally without Docker for the app
go run ./cmd/estimator                                    # memory
STORE=redis REDIS_ADDR=localhost:6379 go run ./cmd/estimator  # needs: docker compose up redis -d

# USS simulator (host → compose-published gRPC)
go run ./cmd/uss

curl -s localhost:8080/v1/segments/sports/estimate
```

| Env | Default | Used by |
|-----|---------|---------|
| `HTTP_ADDR` | `:8080` | estimator (REST) |
| `GRPC_ADDR` | `:9090` / `localhost:9090` | estimator (listen) / uss (dial) |
| `STORE` | `memory` | estimator (`memory` or `redis`) |
| `REDIS_ADDR` | `localhost:6379` | estimator when `STORE=redis` |
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
| **`STORE=memory` or `redis`** | Same `Repository` port; swap at composition root without touching HTTP/gRPC/domain |
| **Redis sorted set** `segment:{name}` score=`taggedAt` | Ingest `ZADD`; estimate `ZCOUNT` + prune expired with `ZREMRANGEBYSCORE` — fits two-week TTL and per-segment scale |
| **Why Redis over Postgres here** | Memberships are ephemeral; score-range counts beat scanning relational rows for this access pattern |
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
  adapter/redis/        # Redis sorted-set repository
```

## AI usage

See [`ai.md`](./ai.md). (`sample.ai.md` is the template only.)

## Original challenge brief

The upstream problem statement lives in git history / the fork source ([ArmanCreativeSolutions/go-challenge](https://github.com/ArmanCreativeSolutions/go-challenge)). This README focuses on the implemented solution.
