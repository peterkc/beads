# ADR 0002: Interface Segregation (Ports & Adapters)

## Status

Proposed

## Context

The current bd v0 architecture tightly couples components:

```
cmd/bd/*.go → internal/storage/sqlite/*.go → internal/git/*.go
                    ↑                              ↑
              direct imports                 direct imports
```

**Problems with v0 architecture:**

1. **Testing difficulty** — Can't test CLI without real SQLite, can't test storage without Git
2. **Circular dependencies** — RPC server imports storage, storage imports RPC for daemon communication
3. **No boundary protection** — Any package can reach into any other's internals
4. **Difficult to extend** — Adding new storage backend (PostgreSQL, S3) requires surgery across codebase
5. **Large `internal/` surface** — 50+ files with unclear boundaries

### Current Package Structure (v0)

```
internal/
├── beads/       # Context, repo detection (mixed concerns)
├── config/      # YAML config
├── daemon/      # Discovery, registry
├── flush/       # FlushManager
├── git/         # Git operations (shelling out)
├── importer/    # JSONL import
├── lockfile/    # File locking
├── rpc/         # Client + Server + Protocol (mixed)
├── storage/     # SQLite implementation
├── syncbranch/  # Sync branch feature
├── types/       # Domain types
└── utils/       # Grab bag
```

## Decision

Adopt **Interface Segregation Principle** via Ports & Adapters (Hexagonal Architecture):

```
                    ┌─────────────────────────────────┐
                    │           Core Domain           │
                    │  (internal/domain/)             │
                    │                                 │
                    │  - Issue, Dependency, Label     │
                    │  - IssueService interface       │
                    │  - Pure business logic          │
                    └───────────────┬─────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
              v                     v                     v
    ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
    │   Storage Port  │   │    Git Port     │   │    RPC Port     │
    │                 │   │                 │   │                 │
    │ - IssueStore    │   │ - Repository    │   │ - Server        │
    │ - DependencyStore│   │ - WorktreeDetector│ │ - Client       │
    │ - LabelStore    │   │ - SyncBranch    │   │ - Protocol      │
    └────────┬────────┘   └────────┬────────┘   └────────┬────────┘
             │                     │                     │
             v                     v                     v
    ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
    │ SQLite Adapter  │   │  Git Adapter    │   │  Unix Adapter   │
    │                 │   │                 │   │                 │
    │ internal/       │   │ internal/       │   │ internal/       │
    │ adapters/sqlite/│   │ adapters/git/   │   │ adapters/rpc/   │
    └─────────────────┘   └─────────────────┘   └─────────────────┘
```

### Target Package Structure (v1)

```
internal/
├── domain/           # Pure domain (no external deps)
│   ├── issue.go      # Issue entity + business rules
│   ├── dependency.go # Dependency entity
│   └── services.go   # Service interfaces (ports)
│
├── ports/            # Interface definitions
│   ├── storage.go    # IssueStore, DependencyStore, LabelStore
│   ├── git.go        # Repository, WorktreeDetector
│   └── rpc.go        # Server, Client interfaces
│
├── adapters/         # Implementations
│   ├── sqlite/       # SQLite storage adapter
│   ├── git/          # Git shell adapter
│   └── rpc/          # Unix socket RPC adapter
│
└── app/              # Application wiring
    └── container.go  # Dependency injection
```

### Key Interfaces (Ports)

```go
// ports/storage.go
type IssueStore interface {
    Create(ctx context.Context, issue *domain.Issue) error
    Get(ctx context.Context, id string) (*domain.Issue, error)
    Update(ctx context.Context, issue *domain.Issue) error
    List(ctx context.Context, filter IssueFilter) ([]*domain.Issue, error)
    Delete(ctx context.Context, id string) error
}

// ports/git.go
type Repository interface {
    IsWorktree() bool
    MainWorktreePath() string
    CurrentBranch() string
    Commit(files []string, message string) error
}

// ports/rpc.go
type DaemonServer interface {
    Start(socketPath string) error
    Stop() error
    Health() HealthStatus
}
```

### Migration Strategy

Use Strangler Fig pattern (see `research/system-diagrams/adr/0003`):

1. **Stage 1**: Define interfaces, create adapters that wrap existing code
2. **Stage 2**: Migrate callers to use interfaces
3. **Stage 3**: Replace adapter implementations with clean code

## Consequences

### Positive

- **Testable** — Mock any port for unit testing
- **Extensible** — Add PostgreSQL adapter without touching core
- **Clear boundaries** — Imports flow inward (adapters → ports → domain)
- **Maintainable** — Small, focused packages

### Negative

- **More files** — Interface + implementation separation
- **Indirection** — One more layer to navigate
- **Migration cost** — Significant refactoring effort

### Neutral

- **No behavior change** — v1 must pass all v0 characterization tests
- **Wire-compatible** — RPC protocol remains unchanged (daemon supersession handles version)

## References

- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Clean Architecture in Go](https://threedots.tech/post/introducing-clean-architecture/)
- ADR 0003 (Migration Strategy) — `research/system-diagrams/adr/0003-migration-strategy-strangler-fig.md`
