# ADR 0002: Hybrid Architecture Patterns for Extensibility and Replaceability

## Status

Accepted

## Context

Beads v1.0 redesign requires:
1. **Extensibility** — Add new integrations (Linear, GitHub, Jira) without modifying core
2. **Replaceability** — Swap any component (storage, events, config) with different implementation
3. **Testability** — Test any layer in isolation with mocks
4. **Maintainability** — Clear boundaries between concerns

The question: Should we pick ONE architectural pattern or combine multiple?

## Decision

**Use a hybrid approach combining complementary patterns at different layers:**

| Layer | Pattern | Purpose |
|-------|---------|---------|
| Foundation | Core Domain (Pure Go) | Business logic with zero dependencies |
| Contracts | Ports (Interfaces) | Define boundaries, enable swapping |
| Implementation | Adapters | Concrete implementations (SQLite, JSONL, Git) |
| Operations | Use Cases + DI | Wire components, enable testing |
| Behavior | Event Bus / Observer | Decouple reactions to changes |
| Extension | Plugin Architecture | Add capabilities without core changes |

### Pattern Responsibilities

```
User Request
     │
     ▼
CLI (Cobra) ──────────────────────────────────────────────────┐
     │                                                        │
     ▼                                                        │
Use Case (DI) ─── calls ──► Port (Interface)                  │
     │                           │                            │
     │                           ▼                            │
     │                      Adapter (SQLite/Memory/Dolt)      │
     │                                                        │
     └─── publishes ──► Event Bus ──► Subscribers             │
                             │                                │
                             ▼                                │
                        Plugin (Linear, Compact) ◄────────────┘
```

### Why These Patterns Are Orthogonal

| Pattern | Scope | Independent Of |
|---------|-------|----------------|
| Ports & Adapters | Storage | How events are handled |
| Event Bus | Communication | Where data is stored |
| Plugin Architecture | Extension | Core implementation |
| Dependency Injection | Wiring | What gets wired |
| Functional Options | Configuration | Runtime behavior |

## Consequences

### Positive

- **Full replaceability**: Any adapter can be swapped (SQLite → Dolt → Memory)
- **Safe extension**: Plugins run in separate processes, crash isolation
- **Loose coupling**: Event subscribers don't know about each other
- **Easy testing**: DI allows mock injection at any boundary
- **Clear boundaries**: Each pattern has single responsibility

### Negative

- **Learning curve**: Contributors must understand multiple patterns
- **More interfaces**: Small interfaces (5-8 methods) multiply file count
- **Indirection**: Request flows through multiple layers

### Mitigations

- Document patterns in ARCHITECTURE.md with examples
- Keep interfaces small (ISP) — easier to understand individually
- Provide "follow the request" tracing guide

## Alternatives Considered

### Single Pattern: Only Ports & Adapters

- **Pro**: Simpler mental model
- **Con**: No extension mechanism, tight coupling for cross-cutting concerns
- **Rejected**: Insufficient for Linear sync, audit logging, hooks

### Single Pattern: Only Plugin Architecture

- **Pro**: Maximum extensibility
- **Con**: Everything becomes a plugin, even core operations
- **Rejected**: Over-engineering for built-in features

### No Formal Pattern

- **Pro**: Fastest initial development
- **Con**: Leads to current v0 situation (62-method interface, tight coupling)
- **Rejected**: Already proven problematic

## References

- [Three Dots Labs - Clean Architecture](https://threedots.tech/post/introducing-clean-architecture/)
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)
- [Repository Pattern in Go](https://threedots.tech/post/repository-pattern-in-go/)
- gh CLI, Terraform, git-bug — all use hybrid approach

## Related

- [ADR 0001: Diagramming Tool Selection](0001-diagramming-tool-selection.md)
- [beads-v1-architecture.md](../beads-v1-architecture.md) — Full v1.0 proposal
