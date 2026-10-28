
# AI Usage Report

## 1. Tools & Models

| Tool / Product | Model | Purpose | Frequency |
|---|---|---|---|
| Claude | Claude Opus 5.5 | System architecture design and evaluation of architectural trade-offs | During architecture design |
| ChatGPT (web) | GPT-6 | General discussions, clarifying requirements, and technical questions | As needed throughout development |
| Cursor | Auto | AI-assisted development, code generation, implementation, and refactoring | Throughout implementation |

## 3. Prompts


### Prompt #1

- **Stage:** Planning / Requirements Analysis
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Please write project rules for me:
  - Follow TDD.
  - Use Hexagonal Architecture (Ports and Adapters).
  - Always follow Conventional Commits and keep commit messages to a single line.
  ```

- **Result:** Created project rules covering TDD, Hexagonal Architecture, and commit conventions.
- **Evaluation:** Reviewed the rules for consistency with the project's requirements.
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #2

- **Stage:** Scaffolding
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Create the initial Go project directory structure only, following Hexagonal Architecture (Ports and Adapters) with DDD package-by-domain layout.

  Use a simple, idiomatic Go layout: `cmd/<app>/` and `internal/<domain>/` with adapters nested under each domain (`adapter/http`, `adapter/memory`). Colocate unit tests next to the code they test — no central `tests/` or layer folders (`services/`, `repositories/`, `ports/`).

  Do not create any files or implement any code yet. Only create the required directories, keeping the structure minimal and avoiding unnecessary abstractions.
  ```

- **Result:** Created empty directories: `cmd/estimator`, `internal/segment`, `internal/segment/adapter/http`, `internal/segment/adapter/memory`.
- **Evaluation:** [Add how you reviewed the structure.]
- **Fixes:** First AI output used layer folders (`domain/`, `application/`, `ports/`, `adapters/...`, `tests/`). Corrected via a DDD package-by-domain rule and rebuilt the tree under `internal/segment/`.

### Prompt #3

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Implement only the API layer:
  - gRPC for receiving user-segment activity from USS (with protobuf)
  - REST/JSON for estimating users in a segment
  - Thin handlers, interfaces for DI, no business logic
  - Tests for routing, validation, and errors
  - Follow TDD and project rules; do not implement domain, DB, or other layers
  ```

- **Result:** Added HTTP estimate and gRPC ingest adapters, ports, protobuf, and tests. `go test ./...` passed.
- **Evaluation:** [Add how you reviewed the API contracts and tests.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #4

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Next step: domain + use cases.
  - Membership with 2-week expiry (inject Clock)
  - Ingest and Estimate services
  - In-memory repository
  - Wire HTTP and gRPC in cmd
  Follow TDD. No Postgres yet.
  ```

- **Result:** Added Membership/TTL, Service, memory repository, and `cmd/estimator`. `go test ./...` passed.
- **Evaluation:** [Add how you reviewed the behaviour and tests.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #5

- **Stage:** Testing
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Write a simple main that acts like USS: send random user-segment data to ES over gRPC.
  One file is enough. Use goroutines for speed if it helps.
  ```

- **Result:** Added `cmd/uss/main.go` (worker pool, random ingest, ok/err stats).
- **Evaluation:** [Add how you tried it against estimator.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #6

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Add Redis behind the same Repository (real persistence + TTL).
  Keep memory as an option. Add docker-compose to make running easier.
  ```

- **Result:** Added Redis sorted-set adapter, `STORE=memory|redis` wiring, Dockerfile, and docker-compose. Tests with miniredis passed.
- **Evaluation:** [Add how you reviewed Redis behaviour and compose.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #7

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Before the next feature, add proper config.
  Put ports, connections, and the membership TTL (now two weeks) in .env.
  ```

- **Result:** Added `internal/config`, `.env.example`, injected TTL into memory/redis repos, wired estimator and uss to config.
- **Evaluation:** [Add how you reviewed the config keys.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #8

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  For postgres+redis, add config flags to flush Redis on startup and warm it from Postgres.
  Also support choosing store mode from config.
  ```

- **Result:** Added `STORE=postgres_redis`, transactional outbox, `REDIS_FLUSH_ON_START` / `REDIS_WARM_ON_START`.
- **Evaluation:** [Add how you reviewed boot flush/warm.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #9

- **Stage:** Refactoring
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Undo the boot flush/warm flags from the previous prompt — they make startup slow and can serve wrong counts.

  Instead:
  - remove REDIS_FLUSH_ON_START and REDIS_WARM_ON_START
  - add POST /v1/admin/reset (token from config)
  - add GET /v1/health
  - while resetting, estimate must return a clear 503, not a wrong number
  ```

- **Result:** Boot flags removed. Added health + admin reset with readiness gating (`ready` / `resetting`).
- **Evaluation:** [Add how you verified 503 during reset.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #10

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Standard reset for postgres_redis: pause outbox, discard pending, flush Redis,
  warm from memberships, resume. Also speed outbox (bigger batch, faster poll, pipeline).
  ```

- **Result:** BeforeWarm/AfterWarm, DiscardPendingOutbox, worker pause/resume, Redis pipeline + MarkDoneBatch.
- **Evaluation:** [Add how you verified reset and drain speed.]
- **Fixes:** [Add any changes made, if applicable.]

### Prompt #11

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  On health monitor show Postgres row count and Redis member count.
  Each refresh call estimate for every segment and chart them.
  ```

- **Result:** Live HTML polls health JSON + per-segment estimates; source_members / segment_names in stats.
- **Evaluation:** [Add how you checked the charts.]
- **Fixes:** [Add any changes made, if applicable.]
