
# AI Usage Report

## 1. Tools & Models

| Tool / Product | Model | Purpose | Frequency |
|---|---|---|---|
| Claude | Claude Opus 5.5 | Architecture ideas and trade-offs | Early design |
| ChatGPT (web) | GPT-6 | Softer English, wording, and “how do I say this” | When writing prompts/docs |
| Cursor | Auto | Coding, refactors, tests, debugging | Most of the work |

## 2. Stages of Development

### Planning / Requirements Analysis
- Talked architecture with Opus (ports/adapters, store options).
- Used ChatGPT when my English/prompt wording was messy.
- Kept decisions myself: TDD, hexagonal, package-by-domain, later Postgres+Redis+outbox.

### Scaffolding / Boilerplate
- Asked Cursor only for empty folders first.
- First layout was wrong (layer folders). I pushed back with a DDD rule and fixed it.

### Implementation
- Cursor wrote most of the code step by step.
- As the repo grew I gave shorter prompts and avoided “read the whole project”.
- I ran a lot of the builds/tests/curl/USS myself to save tokens.

### Debugging
- When something felt wrong (health quiet, outbox slow, reset racing) I described the symptom, not the whole codebase.
- Found real issues like compose ignoring `.env` for outbox settings.

### Testing
- Cursor wrote unit tests with TDD.
- I ran `go test ./...`, Docker, USS loads, and manual reset/health checks myself.

### Documentation
- README/design notes with Cursor help.
- ChatGPT for cleaner sentences when needed.
- This file is my log of what I actually asked.

### Code Review / Refactoring
- Asked Cursor if wiring like `mustRepository` was clean; we moved DI to `cmd` wire.
- Changed boot flush/warm to admin reset after we saw it was a bad ops story.

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
- **Evaluation:** Read the rule files; they matched what I wanted.
- **Fixes:** Later added more rules (DDD layout, project context) when needed.

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
- **Evaluation:** Checked the folder tree in the IDE.
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
- **Evaluation:** Ran the tests myself; glanced at handlers to make sure no business logic leaked in.
- **Fixes:** None in the code. GoLand module setup was on me.

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
- **Evaluation:** Ran tests; checked expiry behaviour made sense with the clock.
- **Fixes:** None.

### Prompt #5

- **Stage:** Testing
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Write a simple main that acts like USS: send random user-segment data to ES over gRPC.
  One file is enough. Use goroutines for speed if it helps.
  ```

- **Result:** Added `cmd/uss/main.go` (worker pool, random ingest, ok/err stats).
- **Evaluation:** Ran USS against a local estimator and watched ok/err counts.
- **Fixes:** Later widened user space / timeout myself when I needed bigger loads.

### Prompt #6

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Add Redis behind the same Repository (real persistence + TTL).
  Keep memory as an option. Add docker-compose to make running easier.
  ```

- **Result:** Added Redis sorted-set adapter, `STORE=memory|redis` wiring, Dockerfile, and docker-compose. Tests with miniredis passed.
- **Evaluation:** Tests + `docker compose up`; tried estimate after ingest.
- **Fixes:** None big.

### Prompt #7

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Before the next feature, add proper config.
  Put ports, connections, and the membership TTL (now two weeks) in .env.
  ```

- **Result:** Added `internal/config`, `.env.example`, injected TTL into memory/redis repos, wired estimator and uss to config.
- **Evaluation:** Checked `.env.example` keys and that the app booted with them.
- **Fixes:** None.

### Prompt #8

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  For postgres+redis, add config flags to flush Redis on startup and warm it from Postgres.
  Also support choosing store mode from config.
  ```

- **Result:** Added `STORE=postgres_redis`, transactional outbox, `REDIS_FLUSH_ON_START` / `REDIS_WARM_ON_START`.
- **Evaluation:** Thought through boot with large data; warm-on-start felt risky for startup time and wrong counts.
- **Fixes:** Replaced in Prompt #9 with admin reset + health instead of boot flush/warm.

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
- **Evaluation:** Called reset/health myself; confirmed estimate returns 503 while resetting.
- **Fixes:** Merged duplicate health routes; put admin token in config.

### Prompt #10

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Standard reset for postgres_redis: pause outbox, discard pending, flush Redis,
  warm from memberships, resume. Also speed outbox (bigger batch, faster poll, pipeline).
  ```

- **Result:** BeforeWarm/AfterWarm, DiscardPendingOutbox, worker pause/resume, Redis pipeline + MarkDoneBatch.
- **Evaluation:** Reset while outbox was busy; watched Redis catch up and health counts.
- **Fixes:** Changing only `.env` was not enough at first — compose had hardcoded outbox settings, and per-row Redis/SQL was slow. Fixed both.

### Prompt #11

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  On health monitor show Postgres row count and Redis member count.
  Each refresh call estimate for every segment and chart them.
  ```

- **Result:** Live HTML polls health JSON + per-segment estimates; source_members / segment_names in stats.
- **Evaluation:** Opened `/v1/health` in the browser and watched numbers/charts update.
- **Fixes:** None.

### Prompt #12

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Clean up processed outbox rows on a schedule.
  Also warm after reset page-by-page so millions of rows are not loaded into one Go slice.
  ```

- **Result:** Outbox cleaner worker + `ListActivePage` keyset warm; config for cleanup/warm page size.
- **Evaluation:** Ran tests; thought about 4M warm — default page 5k felt slow so I bumped it to 50k.
- **Fixes:** Raised `WARM_PAGE_SIZE` default and Redis write chunk after checking the math for millions.

### Prompt #13

- **Stage:** Implementation
- **Tool:** Cursor (Auto)
- **Prompt:**

  ```
  Make the health top row four fixed columns.
  Fourth column: product config (store type, membership TTL, outbox/warm settings).
  ```

- **Result:** Product config on health JSON + HTML fourth card (store, TTL, outbox batch/poll, warm page, cleanup keep).
- **Evaluation:** Rebuilt and checked the dashboard in the browser.
- **Fixes:** None.

## 4. Token Usage Monitoring

- **Monitoring method:** Mostly gut feel + Cursor chat length. I did not track exact token bills in a spreadsheet.
- **Tools/links:** Cursor usage in the IDE.
- **Helper prompts:** None special. I just kept prompts short.
- **Management strategies:**
  - Architecture with Opus, wording with ChatGPT, coding with Cursor Auto.
  - As the code grew, I stopped asking it to reread everything.
  - Short prompts for the next small step only.
  - I ran tests, Docker, USS, and curls myself instead of asking the agent to do every check.
- **Rough total usage:** Continuous Cursor Auto across the challenge; Opus/ChatGPT only when needed.
