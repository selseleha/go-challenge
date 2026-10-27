
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
