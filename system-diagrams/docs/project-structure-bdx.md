# Project Structure: v0 vs bdx

## v0 Structure Analysis

### Key Metrics

| Metric | v0 Value | Problem |
|--------|----------|---------|
| Files in `cmd/bd/` | **329** | God package, no separation |
| `Storage` interface methods | **75** | ISP violation |
| `internal/` packages | 31 | Flat, no hierarchy |
| `util` + `utils` packages | 2 | Naming inconsistency |

### v0 Directory Layout

```
github.com/steveyegge/beads/
├── cmd/bd/                      # 329 files! God package
│   ├── create.go
│   ├── list.go
│   ├── daemon_*.go (7 files)
│   ├── mol_*.go (3 files)
│   ├── sync_*.go (2 files)
│   └── ... (317 more files)
│
├── internal/
│   ├── storage/                 # 75-method interface
│   │   ├── storage.go           # God interface
│   │   ├── sqlite/              # Monolithic implementation
│   │   ├── memory/
│   │   └── dolt/
│   │
│   ├── beads/                   # Core types mixed with logic
│   ├── config/                  # Config loading
│   ├── configfile/              # Config file handling (separate!)
│   ├── types/                   # Domain types
│   │
│   ├── linear/                  # Linear integration
│   ├── molecules/               # Molecules feature
│   ├── compact/                 # Compaction
│   ├── export/                  # Export
│   ├── importer/                # Import (different naming!)
│   │
│   ├── daemon/                  # Daemon logic
│   ├── rpc/                     # RPC server
│   │
│   ├── util/                    # Utilities (duplicate!)
│   ├── utils/                   # Utilities (duplicate!)
│   ├── ui/                      # UI helpers
│   └── ...
│
├── docs/
├── examples/
└── integrations/
```

### v0 Problems

1. **God Package (`cmd/bd/`)**: 329 files with no organization
2. **God Interface (`Storage`)**: 75 methods violates ISP
3. **Flat Internal Structure**: 31 packages at same level
4. **Inconsistent Naming**: `export/` vs `importer/`, `util/` vs `utils/`
5. **Mixed Concerns**: CLI logic mixed with business logic
6. **No Clear Layers**: Storage, business logic, presentation intermingled
7. **Feature Sprawl**: Related files scattered (daemon_*.go, mol_*.go)

---

## bdx Optimized Structure

### Design Principles

1. **Hexagonal Architecture**: Core → Ports → Adapters
2. **Plugin-Based Commands**: Each feature is a plugin
3. **Interface Segregation**: Small, focused interfaces
4. **Vertical Slices**: Features grouped together
5. **Clear Dependencies**: Outer layers depend on inner

### bdx Directory Layout

```
github.com/steveyegge/beads/
├── cmd/
│   ├── bd/                      # v0 CLI (unchanged for compat)
│   │   └── main.go
│   │
│   └── bdx/                     # v1 CLI (new)
│       └── main.go              # ~50 lines: wire plugins, run
│
├── internal/
│   │
│   ├── core/                    # DOMAIN LAYER (pure Go, no deps)
│   │   ├── issue/
│   │   │   ├── issue.go         # Issue entity
│   │   │   ├── status.go        # Status enum
│   │   │   └── priority.go      # Priority enum
│   │   │
│   │   ├── dependency/
│   │   │   ├── dependency.go    # Dependency entity
│   │   │   └── graph.go         # Graph algorithms
│   │   │
│   │   ├── label/
│   │   │   └── label.go         # Label value object
│   │   │
│   │   ├── comment/
│   │   │   └── comment.go       # Comment entity
│   │   │
│   │   └── events/
│   │       ├── event.go         # Domain events
│   │       └── types.go         # Event types enum
│   │
│   ├── ports/                   # PORT INTERFACES (contracts)
│   │   ├── repositories/        # Data access ports
│   │   │   ├── issue.go         # IssueRepository (5 methods)
│   │   │   ├── dependency.go    # DependencyRepository (4 methods)
│   │   │   ├── label.go         # LabelRepository (3 methods)
│   │   │   ├── comment.go       # CommentRepository (3 methods)
│   │   │   ├── config.go        # ConfigRepository (3 methods)
│   │   │   └── sync.go          # SyncRepository (4 methods)
│   │   │
│   │   ├── services/            # External service ports
│   │   │   ├── linear.go        # LinearService
│   │   │   ├── git.go           # GitService
│   │   │   └── notify.go        # NotificationService
│   │   │
│   │   └── events/
│   │       └── bus.go           # EventBus interface
│   │
│   ├── adapters/                # ADAPTER IMPLEMENTATIONS
│   │   ├── sqlite/              # SQLite adapters
│   │   │   ├── issue_repo.go
│   │   │   ├── dependency_repo.go
│   │   │   ├── label_repo.go
│   │   │   ├── comment_repo.go
│   │   │   ├── config_repo.go
│   │   │   ├── sync_repo.go
│   │   │   ├── mapper.go        # Row mapper (DRY)
│   │   │   ├── migrations/      # Schema migrations
│   │   │   └── sqlite.go        # Connection management
│   │   │
│   │   ├── memory/              # In-memory adapters (testing)
│   │   │   ├── issue_repo.go
│   │   │   └── ...
│   │   │
│   │   ├── linear/              # Linear API adapter
│   │   │   ├── client.go
│   │   │   └── mapper.go
│   │   │
│   │   ├── git/                 # Git adapter
│   │   │   ├── repo.go
│   │   │   └── hooks.go
│   │   │
│   │   └── events/              # Event bus implementations
│   │       ├── memory.go        # In-process bus
│   │       └── rpc.go           # RPC-based bus (daemon)
│   │
│   ├── usecases/                # APPLICATION LAYER
│   │   ├── issue/
│   │   │   ├── create.go        # CreateIssueUseCase
│   │   │   ├── list.go          # ListIssuesUseCase
│   │   │   ├── update.go        # UpdateIssueUseCase
│   │   │   └── close.go         # CloseIssueUseCase
│   │   │
│   │   ├── dependency/
│   │   │   ├── add.go           # AddDependencyUseCase
│   │   │   ├── remove.go        # RemoveDependencyUseCase
│   │   │   └── tree.go          # GetDependencyTreeUseCase
│   │   │
│   │   ├── sync/
│   │   │   ├── push.go          # PushSyncUseCase
│   │   │   ├── pull.go          # PullSyncUseCase
│   │   │   └── resolve.go       # ResolveConflictsUseCase
│   │   │
│   │   └── linear/
│   │       ├── sync.go          # SyncLinearUseCase
│   │       └── import.go        # ImportLinearUseCase
│   │
│   ├── plugins/                 # PLUGIN LAYER (CLI commands)
│   │   ├── registry.go          # Plugin registry
│   │   ├── context.go           # Plugin context (DI)
│   │   ├── plugin.go            # Plugin interface
│   │   │
│   │   ├── core/                # Core commands plugin
│   │   │   ├── plugin.go        # Register: create, list, show, update, close
│   │   │   ├── create.go
│   │   │   ├── list.go
│   │   │   ├── show.go
│   │   │   ├── update.go
│   │   │   └── close.go
│   │   │
│   │   ├── work/                # Work management plugin
│   │   │   ├── plugin.go        # Register: ready, dep, blocked
│   │   │   ├── ready.go
│   │   │   ├── dep.go
│   │   │   └── blocked.go
│   │   │
│   │   ├── sync/                # Sync plugin
│   │   │   ├── plugin.go        # Register: sync, export, import
│   │   │   ├── sync.go
│   │   │   ├── export.go
│   │   │   └── import.go
│   │   │
│   │   ├── linear/              # Linear integration plugin
│   │   │   ├── plugin.go        # Register: linear sync, linear import
│   │   │   └── sync.go
│   │   │
│   │   ├── molecules/           # Molecules plugin
│   │   │   ├── plugin.go        # Register: mol current, mol show, mol stale
│   │   │   └── ...
│   │   │
│   │   ├── compact/             # Compact plugin
│   │   │   ├── plugin.go        # Register: compact
│   │   │   └── compact.go
│   │   │
│   │   ├── admin/               # Admin plugin
│   │   │   ├── plugin.go        # Register: doctor, repair, migrate
│   │   │   ├── doctor.go
│   │   │   ├── repair.go
│   │   │   └── migrate.go
│   │   │
│   │   └── daemon/              # Daemon plugin
│   │       ├── plugin.go        # Register: daemon start, daemon stop
│   │       ├── server.go
│   │       └── watcher.go
│   │
│   ├── infra/                   # INFRASTRUCTURE
│   │   ├── config/              # Configuration loading
│   │   │   ├── config.go
│   │   │   ├── file.go
│   │   │   └── env.go
│   │   │
│   │   ├── logging/             # Structured logging
│   │   │   └── logger.go
│   │   │
│   │   ├── telemetry/           # Metrics, tracing
│   │   │   └── trace.go
│   │   │
│   │   └── errors/              # Error types
│   │       └── errors.go
│   │
│   └── ui/                      # PRESENTATION HELPERS
│       ├── table.go             # Table formatting
│       ├── color.go             # Color output
│       └── prompt.go            # Interactive prompts
│
├── pkg/                         # PUBLIC API (if needed)
│   └── beads/
│       └── client.go            # Programmatic API
│
├── docs/
├── examples/
└── integrations/
```

---

## Package Comparison

### Storage Interface: v0 vs bdx

**v0: God Interface (75 methods)**
```go
type Storage interface {
    // Issues (8 methods)
    CreateIssue(...)
    GetIssue(...)
    UpdateIssue(...)
    // ... 5 more

    // Dependencies (10 methods)
    AddDependency(...)
    RemoveDependency(...)
    GetDependencies(...)
    // ... 7 more

    // Labels (3 methods)
    // Comments (4 methods)
    // Config (6 methods)
    // Sync (8 methods)
    // Compact (5 methods)
    // Linear (6 methods)
    // Transactions (2 methods)
    // ... 23 more
}
```

**bdx: Segregated Interfaces**
```go
// ports/repositories/issue.go
type IssueRepository interface {
    Create(ctx context.Context, issue *core.Issue) error
    Get(ctx context.Context, id string) (*core.Issue, error)
    Update(ctx context.Context, id string, updates core.IssueUpdates) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, filter core.IssueFilter) ([]*core.Issue, error)
}

// ports/repositories/dependency.go
type DependencyRepository interface {
    Add(ctx context.Context, dep *core.Dependency) error
    Remove(ctx context.Context, issueID, dependsOnID string) error
    GetBlockers(ctx context.Context, issueID string) ([]*core.Issue, error)
    GetTree(ctx context.Context, issueID string, depth int) (*core.DependencyTree, error)
}

// ports/repositories/label.go
type LabelRepository interface {
    Add(ctx context.Context, issueID, label string) error
    Remove(ctx context.Context, issueID, label string) error
    List(ctx context.Context, issueID string) ([]string, error)
}
```

### Benefits of Segregation

| Aspect | v0 (75 methods) | bdx (5-method interfaces) |
|--------|-----------------|---------------------------|
| Testing | Must mock 75 methods | Mock only what you use |
| Understanding | Overwhelming | Clear purpose per interface |
| Changes | Ripple through everything | Isolated to interface |
| Composition | All or nothing | Mix and match |

---

## File Count Comparison

| Location | v0 | bdx | Reduction |
|----------|------|-----|-----------|
| `cmd/bd/` or `cmd/bdx/` | 329 | 1 | **99.7%** |
| `internal/storage/` | ~50 | 0 | 100% (moved to adapters) |
| `internal/plugins/` | 0 | ~40 | New (organized) |
| `internal/adapters/` | 0 | ~20 | New |
| `internal/ports/` | 0 | ~10 | New |
| `internal/usecases/` | 0 | ~15 | New |
| `internal/core/` | 0 | ~10 | New |

**Total internal/**: v0 ~200 files scattered → bdx ~95 files organized

---

## Import Graph

### v0 Import Graph (Circular Dependencies Risk)

```
cmd/bd → internal/storage → internal/types
       → internal/linear  → internal/storage (!)
       → internal/daemon  → internal/rpc → internal/storage (!)
       → internal/config  → internal/types
```

### bdx Import Graph (Clean Layers)

```
cmd/bdx
    ↓
internal/plugins
    ↓
internal/usecases
    ↓
internal/ports (interfaces only)
    ↓
internal/core (no dependencies)

internal/adapters → internal/ports (implements interfaces)
```

**Rule**: Inner layers never import outer layers.

---

## Migration Mapping

| v0 Location | bdx Location |
|-------------|--------------|
| `cmd/bd/create.go` | `internal/plugins/core/create.go` |
| `cmd/bd/list.go` | `internal/plugins/core/list.go` |
| `cmd/bd/daemon_*.go` | `internal/plugins/daemon/` |
| `cmd/bd/mol_*.go` | `internal/plugins/molecules/` |
| `cmd/bd/sync_*.go` | `internal/plugins/sync/` |
| `internal/storage/storage.go` | `internal/ports/repositories/*.go` |
| `internal/storage/sqlite/` | `internal/adapters/sqlite/` |
| `internal/types/` | `internal/core/` |
| `internal/linear/` | `internal/adapters/linear/` + `internal/plugins/linear/` |
| `internal/rpc/` | `internal/adapters/events/rpc.go` |
| `internal/util/` + `internal/utils/` | `internal/infra/` (consolidated) |

---

## Recommended go.mod for bdx

```go
module github.com/steveyegge/beads

go 1.25

toolchain go1.25.0

ignore (
    research/
    specs/
)

tool (
    golang.org/x/tools/cmd/stringer
    github.com/golangci/golangci-lint/cmd/golangci-lint
)
```

---

## Summary

| Metric | v0 | bdx | Improvement |
|--------|-----|-----|-------------|
| `cmd/` files | 329 | 1 | Plugins handle commands |
| Storage methods | 75 | 5-method interfaces | ISP compliance |
| Package depth | 1 level | 3 levels | Clear hierarchy |
| Circular deps | Possible | Impossible | Layer enforcement |
| Test isolation | Hard | Easy | Small interfaces |
| Feature addition | Scatter files | Add plugin | Vertical slice |

**Key wins:**
1. **From 329 files to 1** in `cmd/bdx/` (plugins do the work)
2. **From 75-method god interface** to 5-method focused interfaces
3. **Clear layer boundaries** (core → ports → adapters → plugins)
4. **Testable by design** (small interfaces, dependency injection)
