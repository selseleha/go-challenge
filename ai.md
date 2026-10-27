
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
