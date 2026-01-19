# Beads v2.0 Architecture Proposal

> First Principles Redesign for Clean Code, DRY, Extensibility

## Executive Summary

Current beads has **117K LoC** across **36 internal packages**. Analysis reveals:

| Issue | Severity | Impact |
|-------|----------|--------|
| Storage interface bloat (62 methods) | High | Hard to implement, test, mock |
| RPC coupling hotspot (33 imports) | High | Changes ripple across codebase |
| SQLite god package (188 methods) | Medium | High cognitive load |
| DRY violations (55 row scans) | Medium | Boilerplate accumulation |

## First Principles Analysis

### Fundamental Truths (Irreducible)

1. **Issues are the core domain** — Everything else serves issue management
2. **Git is the distribution layer** — No central server, travels with code
3. **SQLite is the speed layer** — Fast local queries, disposable cache
4. **JSONL is the truth layer** — Git-trackable, merge-friendly
5. **Agents need automation** — Hooks, events, and RPC are essential

### Current Assumptions to Challenge

| Assumption | Challenge | v2.0 Approach |
|------------|-----------|---------------|
| One storage interface | 62 methods is too many | **Split into focused interfaces** |
| Config injected everywhere | 13 files import config | **Functional options pattern** |
| RPC mirrors storage 1:1 | Protocol coupling | **Use case based commands** |
| Everything in internal/ | Hard to extend | **Plugin architecture** |

---

## Proposed v2.0 Architecture

### Layer 1: Core Domain (Pure Go, No Dependencies)

```
core/
├── issue/          # Issue aggregate root
│   ├── issue.go    # Issue entity + value objects
│   ├── status.go   # Status enum + transitions
│   └── events.go   # Domain events (IssueCreated, StatusChanged)
├── dependency/     # Dependency value object
├── label/          # Label value object
└── work/           # Ready work, blocking logic (pure functions)
```

**Principle**: Domain logic has ZERO external dependencies. Pure functions, no I/O.

### Layer 2: Ports (Interfaces)

Split the 62-method Storage interface into **focused repositories**:

```go
// ports/repository.go

// IssueRepository - Core CRUD (8 methods max)
type IssueRepository interface {
    Create(ctx context.Context, issue *Issue) error
    Get(ctx context.Context, id string) (*Issue, error)
    Update(ctx context.Context, id string, fn func(*Issue) error) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, filter Filter) ([]*Issue, error)
}

// DependencyRepository - Graph operations
type DependencyRepository interface {
    Add(ctx context.Context, dep *Dependency) error
    Remove(ctx context.Context, issueID, dependsOnID string) error
    GetTree(ctx context.Context, id string, depth int) (*Tree, error)
    DetectCycles(ctx context.Context) ([][]string, error)
}

// WorkRepository - Ready/Blocked queries (read-only)
type WorkRepository interface {
    GetReady(ctx context.Context, filter WorkFilter) ([]*Issue, error)
    GetBlocked(ctx context.Context, filter WorkFilter) ([]*BlockedIssue, error)
    IsBlocked(ctx context.Context, id string) (bool, []string, error)
}

// ConfigStore - Separate concern entirely
type ConfigStore interface {
    Get(key string) (string, error)
    Set(key, value string) error
    All() (map[string]string, error)
}

// SyncTracker - Export/import state (separate from issues)
type SyncTracker interface {
    MarkDirty(issueID string) error
    GetDirty() ([]string, error)
    ClearDirty(ids []string) error
    GetExportHash(id string) (string, error)
    SetExportHash(id, hash string) error
}
```

**Benefit**: Each interface is small, mockable, and has a single responsibility.

### Layer 3: Adapters (Implementations)

```
adapters/
├── sqlite/
│   ├── issue_repo.go      # Implements IssueRepository
│   ├── dep_repo.go        # Implements DependencyRepository
│   ├── work_repo.go       # Implements WorkRepository
│   ├── config_store.go    # Implements ConfigStore
│   ├── sync_tracker.go    # Implements SyncTracker
│   └── row_mapper.go      # DRY: Generic row scanning helper
├── jsonl/
│   ├── exporter.go        # SQLite → JSONL
│   └── importer.go        # JSONL → SQLite
├── git/
│   └── integration.go     # Git operations
└── memory/
    └── issue_repo.go      # In-memory for testing
```

**DRY Fix**: Row mapper eliminates 55 scan boilerplate instances:

```go
// adapters/sqlite/row_mapper.go
type RowMapper[T any] struct {
    scanFn func(scanner Scanner) (*T, error)
}

func (m *RowMapper[T]) ScanOne(row *sql.Row) (*T, error) { ... }
func (m *RowMapper[T]) ScanAll(rows *sql.Rows) ([]*T, error) { ... }
```

### Layer 4: Use Cases (Application Services)

```
usecases/
├── issue_ops.go    # CreateIssue, UpdateIssue, CloseIssue
├── work_ops.go     # GetReadyWork, MarkInProgress
├── sync_ops.go     # Export, Import, AutoSync
└── molecule_ops.go # Molecule workflows
```

**Pattern**: Each use case is a function, not a method on a god struct:

```go
// usecases/issue_ops.go
type CreateIssueCommand struct {
    IssueRepo IssueRepository
    EventBus  EventBus
    Config    ConfigStore
}

func (c *CreateIssueCommand) Execute(ctx context.Context, input CreateIssueInput) (*Issue, error) {
    // Validation
    // Create issue
    // Publish event
    return issue, nil
}
```

### Layer 5: Infrastructure

```
infra/
├── daemon/
│   ├── server.go      # RPC server (use-case based, not storage-mirrored)
│   └── client.go      # RPC client
├── cli/
│   └── commands/      # Cobra commands (thin wrappers over use cases)
├── hooks/
│   └── runner.go      # Event-driven hooks
└── plugins/
    ├── registry.go    # Plugin discovery
    └── api.go         # Plugin API (HashiCorp-style)
```

### Layer 6: Plugin System

Inspired by HashiCorp's go-plugin and GolangCI-lint:

```go
// plugins/api.go
type BeadsPlugin interface {
    Name() string
    Version() string

    // Lifecycle
    Init(ctx PluginContext) error
    Shutdown() error
}

type PluginContext struct {
    IssueRepo  IssueRepository  // Read-only access
    EventBus   EventBus         // Subscribe to events
    Config     ConfigStore      // Plugin-specific config
    Logger     Logger
}

// Built-in plugins (can be disabled)
// - linear: Linear sync
// - compact: AI compaction
// - molecules: Workflow templates
```

---

## Configuration Pattern

Replace direct config imports with **functional options**:

```go
// Before (tight coupling)
func NewSQLiteStore(path string) (*Store, error) {
    prefix := config.GetString("issue_prefix")  // ❌ Global access
    ...
}

// After (dependency injection)
type StoreOption func(*storeConfig)

func WithPrefix(prefix string) StoreOption { ... }
func WithJournalMode(mode string) StoreOption { ... }

func NewSQLiteStore(path string, opts ...StoreOption) (*Store, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    ...
}
```

---

## Event Bus (Decoupling)

Replace tight coupling with event-driven architecture:

```go
// events/bus.go
type Event interface {
    EventType() string
    Timestamp() time.Time
}

type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType string, handler EventHandler) (unsubscribe func())
}

// Example events
type IssueCreated struct { Issue *Issue }
type IssueUpdated struct { Issue *Issue; Changes map[string]any }
type IssueClosed  struct { Issue *Issue; Reason string }
```

**Benefit**: Hooks, audit logging, and sync can all subscribe independently.

---

## Testing Strategy

| Layer | Testing Approach |
|-------|------------------|
| Core | Pure unit tests, property-based testing |
| Ports | Interface compliance tests |
| Adapters | Integration tests with testcontainers |
| Use Cases | Mock repositories, test commands |
| CLI | Cobra test patterns (SetOut, SetArgs) |

---

## Migration Path

### Phase 1: Interface Segregation (Low Risk)

1. Split Storage interface into focused interfaces
2. Create adapter implementations that delegate to existing SQLite
3. Run both old and new paths, compare results

### Phase 2: Row Mapper DRY (Medium Risk)

1. Implement generic row mapper
2. Refactor one file at a time (e.g., labels.go first)
3. Verify with existing tests

### Phase 3: Event Bus (Medium Risk)

1. Add event bus alongside existing hooks
2. Migrate hooks to event subscriptions
3. Add audit logging as subscriber

### Phase 4: Plugin Architecture (Higher Risk)

1. Extract Linear integration as first plugin
2. Define stable plugin API
3. Extract compact, molecules as plugins

---

## Package Structure (Final)

```
beads/
├── core/           # Domain (0 dependencies)
├── ports/          # Interfaces
├── adapters/       # Implementations
├── usecases/       # Application logic
├── infra/          # Technical infrastructure
│   ├── daemon/
│   ├── cli/
│   ├── hooks/
│   └── plugins/
├── plugins/        # Built-in plugins
│   ├── linear/
│   ├── compact/
│   └── molecules/
└── cmd/
    └── bd/         # CLI entry point
```

---

## Reference Architectures

Based on research of successful Go CLIs:

| Project | Key Pattern | Applicable to Beads |
|---------|-------------|---------------------|
| **gh CLI** | Focused interfaces, factory pattern | Split Storage interface |
| **git-bug** | Distributed, Git-backed | Validates JSONL approach |
| **HashiCorp Consul** | go-plugin for extensions | Plugin system |
| **GolangCI-lint** | Plugin registry | Extension discovery |
| **Three Dots Labs** | Closure-based transactions | Transaction pattern |

---

## Success Criteria

1. **Storage interface < 15 methods each** (currently 62)
2. **Config imports < 5 files** (currently 13)
3. **No god packages > 50 methods** (currently 188 in sqlite)
4. **Row scanning helpers eliminate 80%+ boilerplate**
5. **Plugin API enables Linear/compact as external plugins**

---

## Next Steps

1. [ ] Create ADR for interface segregation approach
2. [ ] Prototype row mapper with labels.go
3. [ ] Design event bus API
4. [ ] Define plugin API v1

---

*Generated: 2026-01-19*
*Based on: First Principles analysis, Go CLI architecture research, codebase exploration*
