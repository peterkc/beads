# ADR 0003: Migration Strategy — Strangler Fig in Fork

## Status

Accepted (Updated)

## Context

Beads v0 is actively used with multiple contributors. We need to migrate to v1 architecture without:

- Breaking existing functionality
- Requiring "big bang" cutover
- Blocking active v0 contributors
- Creating unmergeable divergence

**Options considered:**

| Option                   | Risk                      | Merge Back | Parallel Dev | v0 Contributors |
| ------------------------ | ------------------------- | ---------- | ------------ | --------------- |
| Feature Flags            | Medium                    | Easy       | No           | Blocked         |
| Parallel Fork (bdx)      | Low initially, High later | Hard       | Yes          | Unaffected      |
| Strangler Fig (in-place) | Low                       | Easy       | Partial      | Blocked         |
| Strangler Fig (in fork)  | Low                       | Medium     | Yes          | Unaffected      |
| Big Bang                 | HIGH                      | N/A        | No           | Blocked         |

## Decision

**Use Strangler Fig pattern in a fork with branch-based phases:**

1. Develop v1 in `peterkc/beads` fork (not upstream)
1. Create v1 implementations alongside v0 (versioned files)
1. Regularly sync fork's `main` with `upstream/main`
1. PR phases back to upstream when stable

### Repository Strategy

```
steveyegge/beads (upstream)
       │
       ├── Active v0 development by contributors
       │   (unaffected by next branch work)
       │
       └── peterkc/beads (fork)
              │
              ├── main              # Tracks upstream/main (stable)
              │
              └── next              # Branched from main (v0 + v1 coexist)
                     │
                     │  STAGE 0: FOUNDATION (Testing-First)
                     ├── next/stage-0.1   # Testing infrastructure
                     ├── next/stage-0.2   # Characterization tests
                     ├── next/stage-0.3   # Core domain layer
                     ├── next/stage-0.4   # Ports (interfaces)
                     │
                     │  STAGE 1: PLUGINIZE
                     ├── next/stage-1.1   # Plugin infrastructure
                     ├── next/stage-1.2   # Core plugin (create/list/show)
                     ├── next/stage-1.3   # Work plugin (ready/dep)
                     ├── next/stage-1.4   # Sync plugin
                     ├── next/stage-1.5   # Integration plugins
                     ├── next/stage-1.6   # Wire main entry point
                     │
                     │  STAGE 2: MODERNIZE
                     ├── next/stage-2.1   # Adapters (SQLite impl)
                     ├── next/stage-2.2   # Use cases
                     ├── next/stage-2.3   # Wire to v1 ports
                     ├── next/stage-2.4   # Validate + v0 compat
                     └── next/stage-2.5   # Cleanup v0 code
```

**Why `next` over `v1`:**

- Version-agnostic (reusable for v2, v3...)
- Industry standard (Node.js, React use this pattern)
- Clearer intent: "what's coming next"

### CLI Coexistence: bd vs bdx

v1 CLI is named `bdx` (beads experimental) to coexist with `bd`:

```
bd   → v0 (current, stable)
bdx  → v1 (experimental, new architecture)
```

**Benefits:**

- Users can test v1 without losing v0
- Side-by-side comparison on same `.beads/` data
- Gradual adoption: use `bdx` for new features, `bd` for stable ops
- When v1 is stable: `bdx` becomes `bd`, old `bd` becomes `bd-legacy` (or removed)

**Build targets:**

```makefile
# In fork's Makefile
build:
    go build -o bd ./cmd/bd           # v0 CLI
    go build -o bdx ./cmd/bdx         # v1 CLI (new entry point)
```

### Internal Plugin Architecture (Single Binary)

Every command is a plugin, but all plugins compile into a single binary:

```
USER SEES:                         INTERNALLY:
──────────                         ──────────
$ bdx create ...                   ┌─────────────────────────────┐
$ bdx list ...                     │  bdx (single binary)        │
$ bdx linear sync                  │  ┌─────────────────────────┐│
                                   │  │ Plugin Registry         ││
                                   │  │  ├─ core.Plugin         ││
                                   │  │  ├─ work.Plugin         ││
                                   │  │  ├─ sync.Plugin         ││
                                   │  │  ├─ linear.Plugin       ││
                                   │  │  └─ molecules.Plugin    ││
                                   │  └─────────────────────────┘│
                                   └─────────────────────────────┘
```

**Plugin interface:**

```go
// internal/plugins/plugin.go
type Plugin interface {
    Metadata() Metadata
    Commands() []Command
}

type Metadata struct {
    Name        string
    Version     string
    Description string
}

type Command struct {
    Name    string
    Aliases []string
    Short   string
    Run     func(ctx *Context, args []string) error
}

// internal/plugins/context.go — DI container
type Context struct {
    Issues       ports.IssueRepository
    Dependencies ports.DependencyRepository
    Work         ports.WorkRepository
    Config       ports.ConfigStore
    Events       ports.EventBus
}
```

**Registry and routing:**

```go
// internal/plugins/registry.go
type Registry struct {
    plugins map[string]Plugin
}

func (r *Registry) Register(p Plugin) {
    r.plugins[p.Metadata().Name] = p
}

func (r *Registry) Execute(name string, ctx *Context, args []string) error {
    for _, p := range r.plugins {
        for _, cmd := range p.Commands() {
            if cmd.Name == name || contains(cmd.Aliases, name) {
                return cmd.Run(ctx, args)
            }
        }
    }
    return fmt.Errorf("unknown command: %s", name)
}
```

**Main entry point:**

```go
// cmd/bdx/main.go
func main() {
    registry := plugins.NewRegistry()

    // All plugins compiled in (single binary)
    registry.Register(core.Plugin{})
    registry.Register(work.Plugin{})
    registry.Register(sync.Plugin{})
    registry.Register(linear.Plugin{})
    registry.Register(molecules.Plugin{})

    ctx := wireContext()  // DI setup

    if err := registry.Execute(os.Args[1], ctx, os.Args[2:]); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**Benefits:**

- Single binary distribution (no plugin installation)
- Clean separation of concerns (each command isolated)
- Future extensibility (can add external plugins later)
- Testable (mock PluginContext for unit tests)

**Plugin directory structure:**

```
internal/plugins/
├── plugin.go              # Plugin interface
├── context.go             # PluginContext (DI container)
├── registry.go            # Command routing
│
├── core/                  # Core commands plugin
│   ├── plugin.go          # Registers: create, list, show, update, close
│   ├── create.go
│   ├── list.go
│   └── ...
│
├── work/                  # Work management plugin
│   ├── plugin.go          # Registers: ready, dep
│   └── ...
│
├── sync/                  # Sync plugin
│   ├── plugin.go          # Registers: sync, export, import
│   └── ...
│
├── linear/                # Linear integration plugin
│   └── plugin.go
│
└── molecules/             # Molecules plugin
    └── plugin.go
```

### Workflow

1. **Sync regularly**: Automated via GitHub Action (daily) or manual
1. **Develop phases**: Work in `next/phase-*` branches
1. **Integrate**: Merge phases into `next` for testing
1. **Rebase before PR**: Keep phases rebased on latest `main`
1. **PR to upstream**: When phase is stable, PR to `steveyegge/beads`
1. **Final PR**: Rename `bdx` → `bd` when v1.0 ships

### Automated Sync (GitHub Action)

A GitHub Action automates keeping the fork in sync with upstream:

```yaml
# .github/workflows/sync-upstream.yml
# See: research/system-diagrams/workflows/sync-upstream.yml

Schedule: Daily at 6 AM UTC
Manual: workflow_dispatch with rebase_next option

Jobs:
1. sync-main      → Merge upstream/main into fork's main
2. rebase-next    → Rebase next on updated main
3. notify-sync    → Summary + create issue on conflict
```

**On conflict:**

- Workflow creates GitHub issue with `sync-conflict` label
- Developer resolves manually, closes issue

**Setup:**

```bash
# Copy workflow to fork
cp research/system-diagrams/workflows/sync-upstream.yml .github/workflows/
git add .github/workflows/sync-upstream.yml
git commit -m "ci: add upstream sync workflow"
git push
```

### Why Fork + Strangler Fig (Hybrid)

| Benefit                        | How                                    |
| ------------------------------ | -------------------------------------- |
| v0 contributors unblocked      | They work on upstream, we work on fork |
| No merge conflicts during dev  | Isolated branch until PR time          |
| Incremental PRs still possible | Each phase can be PR'd separately      |
| Rollback easy                  | Fork can reset; upstream unchanged     |
| Integration testing            | `next` validates all phases together   |

### Versioned Files Pattern (Within Fork)

```
Option A: Wrappers (delegate to v0)          Option B: Versioned Files (swap when ready)
─────────────────────────────────            ─────────────────────────────────────────
v1.IssueRepository                           storage/sqlite/issues.go      (v0 - current)
    └── wraps v0.Storage                     storage/sqlite/issues_v1.go   (v1 - new impl)

Pros: Immediate v1 interface                 Pros: Clean implementations
Cons: Runtime delegation overhead            Cons: Duplicate code temporarily
      Wrapper code is throwaway
                                             Swap: Rename issues_v1.go → issues.go
```

**Decision: Option B (Versioned Files)** — Cleaner implementations, atomic swap.

## Migration Phases (Testing-First)

### Three-Stage Migration Strategy

**Testing-first approach:** Establish safety net before any structural changes.

```
STAGE 0: FOUNDATION      STAGE 1: PLUGINIZE         STAGE 2: MODERNIZE
───────────────────      ──────────────────         ──────────────────

┌─────────────────┐      ┌─────────────────┐        ┌─────────────────┐
│ 0.1 Testing     │      │ 1.1 Plugin Infra│        │ 2.1 Adapters    │
│ 0.2 Char Tests  │  ─►  │ 1.2 Core Plugin │   ─►   │ 2.2 Use Cases   │
│ 0.3 Core Domain │      │ 1.3 Work Plugin │        │ 2.3 Wire to v1  │
│ 0.4 Ports       │      │ 1.4 Sync Plugin │        │ 2.4 Validate    │
└─────────────────┘      │ 1.5 Other Plugs │        │ 2.5 Cleanup     │
                         │ 1.6 Wire Main   │        └─────────────────┘
Safety net +             └─────────────────┘
contracts first          Wrap v0 in plugins         Replace internals
```

**Benefits:**

- **Stage 0 provides safety net** — characterization tests catch regressions
- v0 never breaks (plugins just wrap existing code)
- Architecture locked early (plugin interfaces are the contract)
- Each plugin can be refactored independently
- Can ship v0-plugins (same behavior) then gradually upgrade
- Clear progress tracking (which plugins are modernized?)

**See:** `docs/migration-phases-revised.md` for detailed phase breakdown

### Stub Pattern (Graceful Errors)

For v1 adapters that aren't implemented yet:

```go
// internal/ports/errors.go
var ErrNotImplemented = errors.New("not implemented")

// internal/adapters/sqlite/issue_repo.go
func (r *IssueRepo) Get(ctx context.Context, id string) (*Issue, error) {
    return nil, fmt.Errorf("IssueRepo.Get: %w", ErrNotImplemented)
}
```

**Why errors, not panics:**

- Program continues (partial functionality)
- Clear error messages show what's missing
- Testable (can assert on ErrNotImplemented)
- Commands that don't use unimplemented code still work

______________________________________________________________________

## Stage 0: FOUNDATION (Testing-First)

**Goal:** Establish testing infrastructure and contracts BEFORE any structural changes.

### Phase 0.1: Testing Infrastructure

**Duration:** 1-2 days

**New dependencies:**

```go
// go.mod additions
require (
    github.com/stretchr/testify v1.9.0
    go.uber.org/mock v0.5.0
    pgregory.net/rapid v1.2.0
)
```

**New files:**

```
internal/testutil/
├── fixtures.go        # Test data generators
├── assertions.go      # Custom assertions
└── db.go              # In-memory SQLite helper

Makefile additions:
    test-unit, test-integration, test-e2e targets
```

**Deliverable:** Testing stack ready, CI workflow passes

______________________________________________________________________

### Phase 0.2: Characterization Tests

**Duration:** 3-5 days

**Goal:** Capture v0 behavior as executable specifications (safety net).

**New files:**

```
characterization/
├── create_test.go         # Create behavior
├── list_test.go           # List/search behavior
├── update_test.go         # Update behavior
├── dependency_test.go     # Dependency behavior
├── sync_test.go           # Sync behavior
└── helpers_test.go        # Shared setup
```

**Example:**

```go
//go:build characterization

func TestCreate_V0Behavior(t *testing.T) {
    store := setupV0Store(t)

    tests := []struct {
        name     string
        input    types.Issue
        validate func(*testing.T, *types.Issue)
    }{
        {
            name:  "priority defaults to 2",
            input: types.Issue{Title: "Test"},
            validate: func(t *testing.T, got *types.Issue) {
                assert.Equal(t, 2, got.Priority)
            },
        },
        // ... capture ALL behaviors
    }
    // ...
}
```

**Deliverable:** 100% of v0 behaviors documented as tests

______________________________________________________________________

### Phase 0.3: Core Domain Layer

**Duration:** 2-3 days

**Goal:** Pure Go domain with zero external dependencies.

**New files:**

```
internal/core/
├── issue/
│   ├── issue.go           # Issue entity
│   ├── status.go          # Status enum
│   └── priority.go        # Priority enum
├── dependency/
│   ├── dependency.go      # Dependency entity
│   └── graph.go           # Graph algorithms
├── label/
│   └── label.go           # Label value object
└── events/
    ├── event.go           # Domain events
    └── types.go           # Event type enum
```

**Rule:** `internal/core/` has ZERO imports from outside `core/`.

**Deliverable:** Domain entities with business logic, fully tested

______________________________________________________________________

### Phase 0.4: Ports (Interfaces)

**Duration:** 1-2 days

**Goal:** Define contracts before implementation.

**New files:**

```
internal/ports/
├── repositories/
│   ├── issue.go           # IssueRepository (5 methods)
│   ├── dependency.go      # DependencyRepository (4 methods)
│   ├── label.go           # LabelRepository (3 methods)
│   ├── comment.go         # CommentRepository (3 methods)
│   ├── config.go          # ConfigRepository (3 methods)
│   └── sync.go            # SyncRepository (4 methods)
├── services/
│   ├── linear.go          # LinearService
│   └── git.go             # GitService
└── events/
    └── bus.go             # EventBus interface

internal/mocks/               # Generated by mockgen
├── issue_repo_mock.go
├── dependency_repo_mock.go
└── ...
```

**Deliverable:** All interfaces defined, mocks generated

______________________________________________________________________

### Stage 0 Checkpoint

Before proceeding to Stage 1:

```
✅ Testing infrastructure (testify, gomock, rapid)
✅ Characterization tests (v0 behavior captured)
✅ Core domain (pure Go, fully tested)
✅ Port interfaces (contracts defined)
✅ Generated mocks (ready for use case tests)
```

**Validation gate:**

```bash
go test -tags=characterization ./characterization/...  # 100% pass
go test ./internal/core/...                            # 100% pass
go generate ./internal/ports/...                       # Mocks generate
```

______________________________________________________________________

## Stage 1: PLUGINIZE (v0-plugins)

### Phase 1.1: Plugin Infrastructure

**Goal:** Create plugin system that wraps existing v0 code

**New files:**

```
internal/plugins/
├── plugin.go              # Plugin interface
├── context.go             # PluginContext (wraps v0 Storage for now)
└── registry.go            # Command routing
```

**Example v0-wrapper PluginContext:**

```go
// internal/plugins/context.go
type Context struct {
    // Stage 1: wraps v0 Storage directly
    Storage *storage.Storage  // existing 62-method interface

    // Stage 2: will become individual ports
    // Issues       ports.IssueRepository
    // Dependencies ports.DependencyRepository
    // ...
}

func NewContext(db *sql.DB) *Context {
    return &Context{
        Storage: storage.New(db),  // existing v0 code
    }
}
```

**Risk:** Low (infrastructure only)
**Deliverable:** Plugin registry compiles

______________________________________________________________________

### Phase 1.2: Wrap Core Commands

**Goal:** Wrap create, list, show, update, close in core.Plugin

**New files:**

```
internal/plugins/core/
├── plugin.go              # Registers commands
├── create.go              # Calls ctx.Storage.CreateIssue()
├── list.go                # Calls ctx.Storage.ListIssues()
├── show.go                # Calls ctx.Storage.GetIssue()
├── update.go              # Calls ctx.Storage.UpdateIssue()
└── close.go               # Calls ctx.Storage.CloseIssue()
```

**Example wrapper:**

```go
// internal/plugins/core/create.go
func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    // Parse args (same as current cmd/bd/create.go)
    title, description, priority := parseArgs(args)

    // Delegate to existing v0 code
    issue, err := ctx.Storage.CreateIssue(title, description, priority)
    if err != nil {
        return err
    }

    fmt.Printf("Created: %s\n", issue.ID)
    return nil
}
```

**Risk:** Low (thin wrappers over existing code)
**Deliverable:** `bd create`, `bd list`, etc. work via plugins

______________________________________________________________________

### Phase 1.3: Wrap Work Commands

**Goal:** Wrap ready, dep, blocked in work.Plugin

**New files:**

```
internal/plugins/work/
├── plugin.go              # Registers: ready, dep, blocked
├── ready.go               # Calls ctx.Storage.GetReadyIssues()
├── dep.go                 # Calls ctx.Storage.AddDependency()
└── blocked.go             # Calls ctx.Storage.GetBlockedIssues()
```

**Risk:** Low

______________________________________________________________________

### Phase 1.4: Wrap Sync Commands

**Goal:** Wrap sync, export, import in sync.Plugin

**New files:**

```
internal/plugins/sync/
├── plugin.go              # Registers: sync, export, import
├── sync.go                # Calls existing sync logic
├── export.go              # Calls ctx.Storage.Export()
└── import.go              # Calls existing importer
```

**Risk:** Low

______________________________________________________________________

### Phase 1.5: Wrap Integration Plugins

**Goal:** Wrap linear, molecules, compact in their own plugins

**New files:**

```
internal/plugins/linear/
└── plugin.go              # Calls internal/linear/*

internal/plugins/molecules/
└── plugin.go              # Calls internal/molecules/*

internal/plugins/compact/
└── plugin.go              # Calls internal/compact/*
```

**Risk:** Low

______________________________________________________________________

### Phase 1.6: Wire Main Entry Point

**Goal:** Replace cmd/bd/main.go with plugin-based routing

**Before (v0):**

```go
// cmd/bd/main.go
func main() {
    rootCmd := &cobra.Command{Use: "bd"}
    rootCmd.AddCommand(createCmd, listCmd, ...)
    rootCmd.Execute()
}
```

**After (v0-plugins):**

```go
// cmd/bd/main.go
func main() {
    registry := plugins.NewRegistry()

    // All plugins wrap existing v0 code
    registry.Register(core.Plugin{})
    registry.Register(work.Plugin{})
    registry.Register(sync.Plugin{})
    registry.Register(linear.Plugin{})
    registry.Register(molecules.Plugin{})
    registry.Register(compact.Plugin{})

    ctx := plugins.NewContext(openDB())

    if err := registry.Execute(os.Args[1], ctx, os.Args[2:]); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**Risk:** Low (same behavior, different structure)
**Deliverable:** `bd` binary works identically, but uses plugin architecture

______________________________________________________________________

## Stage 1 Checkpoint

At this point:

- `bd` still works exactly as before
- All commands are plugins (clean separation)
- Plugin interface is the contract for Stage 2
- Can ship this as a release (no behavior change)

```
✅ Plugin infrastructure
✅ core.Plugin (create, list, show, update, close)
✅ work.Plugin (ready, dep, blocked)
✅ sync.Plugin (sync, export, import)
✅ linear.Plugin
✅ molecules.Plugin
✅ compact.Plugin
✅ Main entry point rewired
```

______________________________________________________________________

## Stage 2: MODERNIZE (v1-plugins)

Now refactor each plugin's internals to use v1 architecture (ports/adapters).

### Phase 2.1: Create Ports (Interfaces)

**Goal:** Define v1 interfaces in `internal/ports/`

**New files:**

```
internal/ports/
├── errors.go                     # ErrNotImplemented
├── issue_repository.go           # 5 methods
├── dependency_repository.go      # 4 methods
├── work_repository.go            # 3 methods
├── config_store.go               # 3 methods
├── sync_tracker.go               # 5 methods
└── event_bus.go                  # 2 methods
```

**Risk:** Low (interfaces only, no implementation)

______________________________________________________________________

### Phase 2.2: Create v1 Adapters (with stubs)

**Goal:** Create adapter stubs that return ErrNotImplemented

**New files:**

```
internal/adapters/sqlite/
├── issue_repo.go             # Stub
├── dependency_repo.go        # Stub
├── work_repo.go              # Stub
├── config_store.go           # Stub
├── sync_tracker.go           # Stub
└── row_mapper.go             # Implement early (DRY helper)
```

**Risk:** Low (all stubs)

______________________________________________________________________

### Phase 2.3: Implement v1 Adapters (Parallel!)

**Goal:** Fill in adapter stubs - can be done in parallel per plugin

| Plugin | Adapter to Implement | Port From                        |
| ------ | -------------------- | -------------------------------- |
| core   | `issue_repo.go`      | `storage/sqlite/queries.go`      |
| work   | `work_repo.go`       | `storage/sqlite/ready.go`        |
| work   | `dependency_repo.go` | `storage/sqlite/dependencies.go` |
| sync   | `sync_tracker.go`    | `storage/sqlite/dirty.go`        |

**Each plugin can be modernized independently:**

```go
// internal/plugins/core/create.go

// Before (v0-plugin, Stage 1):
func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    return ctx.Storage.CreateIssue(...)  // v0 code
}

// After (v1-plugin, Stage 2):
func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    return ctx.Issues.Create(...)  // v1 port
}
```

**Risk:** Medium (most complex phase)

______________________________________________________________________

### Phase 2.4: Update PluginContext

**Goal:** PluginContext switches from v0 Storage to v1 ports

**Before (Stage 1):**

```go
type Context struct {
    Storage *storage.Storage  // v0: 62 methods
}
```

**After (Stage 2):**

```go
type Context struct {
    // v1 ports (segregated interfaces)
    Issues       ports.IssueRepository
    Dependencies ports.DependencyRepository
    Work         ports.WorkRepository
    Config       ports.ConfigStore
    Sync         ports.SyncTracker
    Events       ports.EventBus
}
```

**Risk:** Medium (all plugins must be updated)

______________________________________________________________________

### Phase 2.5: Validate and Cleanup

**Goal:** Integration testing, remove v0 code

- Run bdx against existing .beads/ databases
- Compare output with bd (should match)
- Performance benchmarks
- Delete `internal/storage/` (v0 code)

**Risk:** Low (testing and cleanup)

______________________________________________________________________

## Upstream Sync Automation

Keeping `next` (v1) in sync with upstream changes requires multi-layer automation.

### Sync Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│                    UPSTREAM SYNC PIPELINE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  steveyegge/beads ──┐                                           │
│         (upstream)  │                                           │
│                     ▼                                           │
│  ┌──────────────────────────────────┐                          │
│  │ 1. sync-upstream.yml (daily)     │  Git level               │
│  │    main ← upstream/main          │                          │
│  └──────────────────────────────────┘                          │
│                     │                                           │
│                     ▼                                           │
│  ┌──────────────────────────────────┐                          │
│  │ 2. port-upstream.yml (triggered) │  Analysis                │
│  │    - analyze-upstream.sh         │                          │
│  │    - Which plugins affected?     │                          │
│  └──────────────────────────────────┘                          │
│                     │                                           │
│                     ▼                                           │
│  ┌──────────────────────────────────┐                          │
│  │ 3. create-port-issues.sh         │  Tracking                │
│  │    - Beads issue per commit      │                          │
│  │    - Priority by plugin          │                          │
│  └──────────────────────────────────┘                          │
│                     │                                           │
│                     ▼                                           │
│  ┌──────────────────────────────────┐                          │
│  │ 4. Manual: Port to v1 plugin     │  Implementation          │
│  │    - Update plugin code          │                          │
│  │    - Add git note                │                          │
│  │    - Close beads issue           │                          │
│  └──────────────────────────────────┘                          │
│                     │                                           │
│                     ▼                                           │
│  ┌──────────────────────────────────┐                          │
│  │ 5. test-compatibility.sh         │  Validation              │
│  │    - v0 ↔ v1 data compatibility  │                          │
│  └──────────────────────────────────┘                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Scripts

| Script                          | Purpose                          |
| ------------------------------- | -------------------------------- |
| `scripts/analyze-upstream.sh`   | Analyze commits, map to plugins  |
| `scripts/create-port-issues.sh` | Create beads issues for tracking |
| `scripts/port-status.sh`        | Dashboard of port progress       |
| `scripts/test-compatibility.sh` | v0↔v1 data compatibility         |

### File → Plugin Mapping

The analyze script maps upstream files to v1 plugins:

| Upstream File                              | Plugin               |
| ------------------------------------------ | -------------------- |
| `internal/storage/sqlite/queries.go`       | core                 |
| `internal/storage/sqlite/ready.go`         | work                 |
| `internal/storage/sqlite/dependencies.go`  | work                 |
| `internal/export/*`, `internal/importer/*` | sync                 |
| `internal/linear/*`                        | linear               |
| `internal/molecules/*`                     | molecules            |
| `internal/compact/*`                       | compact              |
| `internal/storage/storage.go`              | shared (all plugins) |

### Usage

```bash
# Check current port status
./scripts/port-status.sh

# Analyze new upstream commits
./scripts/analyze-upstream.sh

# Create issues for untracked commits
./scripts/analyze-upstream.sh | ./scripts/create-port-issues.sh

# After porting, add git note and close issue
git notes add -m "ported-from: abc1234 (upstream)"
bd close beads-xxx
```

### GitHub Actions

| Workflow            | Trigger        | Purpose                               |
| ------------------- | -------------- | ------------------------------------- |
| `sync-upstream.yml` | Daily 6 AM UTC | Merge upstream → main                 |
| `port-upstream.yml` | After sync     | Analyze + create issues               |
| `claude-port.yml`   | After sync     | **AI-assisted porting + stacked PRs** |

### Claude Code + Graphite Automation (Advanced)

For higher automation, Claude Code can do the actual porting work:

```
┌─────────────────────────────────────────────────────────────────┐
│              CLAUDE CODE AUTOMATED PORTING                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  For each upstream commit:                                      │
│                                                                  │
│  1. Create branch                                               │
│     git checkout -b v0/port-abc123 next                        │
│                                                                  │
│  2. Claude Code analyzes and ports                              │
│     - Reads upstream diff                                       │
│     - Maps to v1 plugin(s)                                      │
│     - Updates plugin code                                       │
│     - Runs tests                                                │
│                                                                  │
│  3. Creates stacked PR via Graphite                             │
│     gt create -m "v0/port-abc123: Subject"                     │
│                                                                  │
│  4. Local review                                                │
│     ./scripts/review-ports.sh                                   │
│     - Interactive approve/edit/skip                             │
│     - gt merge when ready                                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Setup requirements:**

```bash
# 1. GitHub secrets
ANTHROPIC_API_KEY   # For Claude Code
GRAPHITE_TOKEN      # For stacked PRs

# 2. Local tools
npm install -g @withgraphite/graphite-cli
npm install -g @anthropic-ai/claude-code
gt auth --token <token>
```

**Local review workflow:**

```bash
# List pending port PRs
./scripts/review-ports.sh --list

# Interactive review (approve/edit/skip each)
./scripts/review-ports.sh

# View Graphite stack
gt log

# Merge approved PRs
gt merge --all
```

**Why stacked PRs (Graphite)?**

| Benefit          | How                              |
| ---------------- | -------------------------------- |
| Atomic reviews   | Each upstream commit = one PR    |
| Dependency order | PRs stack in correct merge order |
| Easy reorder     | `gt restack` if needed           |
| Batch merge      | `gt merge --all` when approved   |

______________________________________________________________________

## End-Game Strategy: Replace (Not Merge)

When v1 is ready, the `next` branch **replaces** `main` rather than merging into it.

### Why Replace Over Merge?

| Factor              | Situation                                | Favors  |
| ------------------- | ---------------------------------------- | ------- |
| Architecture        | Hexagonal (ports/adapters) vs monolithic | Replace |
| Code reuse          | Wrapping v0, then rewriting              | Replace |
| Directory structure | `internal/plugins/` vs flat              | Replace |
| Git history value   | v0 patterns being abandoned              | Replace |
| Traceability need   | Yes → git notes solve this               | Replace |

**Key insight:** Merge preserves history in git, but v1 code won't share ancestry with v0 anyway — it's rewritten. Git notes provide traceability without merge complexity.

### Branch Strategy

```
DURING DEVELOPMENT:                        END-GAME:
──────────────────                         ─────────
steveyegge/beads                           steveyegge/beads
├── main (v0)                              ├── v0-archive (old main)
│                                          └── main (was next, now v1)
└── peterkc/beads (fork)
    ├── main (tracks upstream)
    ├── next (v1 development, orphan)
    └── v0/port-* (cherry-picks)
```

### End-Game Workflow

```bash
# In upstream (steveyegge/beads) when v1 is ready:

# 1. Archive v0
git branch v0-archive main
git push origin v0-archive
git tag v0-final -m "Final v0 release before v1 migration"

# 2. Replace main with next
git checkout next
git branch -M main                  # next becomes main
git push --force-with-lease origin main

# 3. Tag the transition
git tag v1.0.0 -m "v1 architecture release"
git push origin v1.0.0

# 4. Update default branch if needed (GitHub settings)
```

### User Migration Path

| User Scenario   | Action                                             |
| --------------- | -------------------------------------------------- |
| Staying on v0   | `git checkout v0-archive` (indefinite support TBD) |
| Migrating to v1 | `git pull` (already on new main)                   |
| Fresh clone     | Gets v1 automatically                              |

### Git Notes Preserve Lineage

Even after replace, notes link v1 code to v0 origins:

```bash
# Find what v0 commit inspired v1 code
git notes show HEAD
# → "derived-from: abc1234 (v0 internal/storage/sqlite/queries.go)"

# Find all v1 commits derived from a specific v0 commit
git log --notes --grep="derived-from: abc1234"
```

### Branch Strategy During Development

**`next` branches from `main`** (NOT orphan) so plugins can wrap v0:

```
next branch (branched from main)
─────────────────────────────────
internal/
├── v0/                 # All v0 code (reorganized)
│   ├── storage/        # 75-method interface
│   ├── types/
│   ├── linear/
│   └── ...
│
└── next/               # All v1 code (NEW)
    ├── core/           # Domain layer
    ├── ports/          # Interfaces
    ├── adapters/       # Implementations
    ├── usecases/       # Application logic
    └── plugins/        # Wraps internal/v0/!

cmd/
├── bd/                 # v0 CLI (imports internal/v0/)
└── bdx/                # v1 CLI (imports internal/next/)
```

**Benefits of `internal/{v0,next}/` structure:**

- Explicit versioning in import paths
- Trivial cleanup: `rm -rf internal/v0/`
- No ambiguity about which code is legacy
- Clear migration: move imports from `v0/` to `next/`

**Why NOT orphan during development:**

- Strangler Fig requires v0 and v1 to coexist
- Plugins must import v0 packages to wrap them
- Orphan branch has no v0 code to wrap

**See:** `docs/package-structure-versioned.md` for detailed structure

### Why Orphan Is No Longer Needed

With versioned directories (`internal/v0/` and `internal/next/`), orphan branch is **unnecessary**:

| Goal                 | Orphan Approach             | Versioned Directories                    |
| -------------------- | --------------------------- | ---------------------------------------- |
| Clean separation     | Separate git history        | Separate import paths (`v0/` vs `next/`) |
| Easy cleanup         | Complex (branch swap)       | Trivial: `rm -rf internal/v0/`           |
| History preservation | Lost                        | ✅ Kept (same branch)                    |
| Git blame/bisect     | Only within orphan          | ✅ Works across versions                 |
| Strangler Fig        | ❌ Breaks (can't import v0) | ✅ Works (plugins import v0/)            |

**The versioned directory structure solves the same problem more elegantly:**

- History is clean because v0 code lives under `internal/v0/`
- Cleanup is trivial: one `rm -rf` command
- Git blame/bisect works across the entire migration
- No branch swap complexity at end-game

**Recommendation:** Do NOT use orphan branch. Versioned directories provide all benefits without the drawbacks.

______________________________________________________________________

## Automation Scripts

### Progress Tracker

```bash
#!/usr/bin/env bash
# scripts/check-stubs.sh - Find unimplemented stubs

grep -r "ErrNotImplemented" internal/adapters internal/usecases \
    --include="*.go" -l | while read file; do
    count=$(grep -c "ErrNotImplemented" "$file")
    echo "❌ $file ($count stubs)"
done

implemented=$(find internal/adapters internal/usecases -name "*.go" \
    -exec grep -L "ErrNotImplemented" {} \;)
for file in $implemented; do
    echo "✅ $file (implemented)"
done
```

### Commit Porter

```bash
#!/usr/bin/env bash
# scripts/port-commit.sh - Map bd commit to bdx location

SHA="$1"
echo "Analyzing commit $SHA..."

git show --name-only "$SHA" | tail -n +7 | while read file; do
    case "$file" in
        internal/storage/sqlite/queries.go)
            echo "  $file → internal/adapters/sqlite/issue_repo.go" ;;
        internal/storage/sqlite/dependencies.go)
            echo "  $file → internal/adapters/sqlite/dependency_repo.go" ;;
        internal/rpc/server_*.go)
            echo "  $file → internal/usecases/*.go" ;;
        *)
            echo "  $file → (manual mapping needed)" ;;
    esac
done
```

### File Mapping Table

| bd (v0)                                   | bdx (v1)                                      |
| ----------------------------------------- | --------------------------------------------- |
| `internal/storage/storage.go`             | `internal/ports/*.go`                         |
| `internal/storage/sqlite/queries.go`      | `internal/adapters/sqlite/issue_repo.go`      |
| `internal/storage/sqlite/dependencies.go` | `internal/adapters/sqlite/dependency_repo.go` |
| `internal/storage/sqlite/ready.go`        | `internal/adapters/sqlite/work_repo.go`       |
| `internal/storage/sqlite/config.go`       | `internal/adapters/sqlite/config_store.go`    |
| `internal/storage/sqlite/dirty.go`        | `internal/adapters/sqlite/sync_tracker.go`    |
| `internal/rpc/server_*.go`                | `internal/usecases/*.go`                      |
| `internal/linear/`                        | `plugins/linear/`                             |
| `internal/compact/`                       | `plugins/compact/`                            |
| `internal/molecules/`                     | `plugins/molecules/`                          |
| `cmd/bd/`                                 | `cmd/bdx/`                                    |

______________________________________________________________________

## File Count Summary

### Stage 0: FOUNDATION

| Phase             | New Files | Modified | Risk | Deliverable            |
| ----------------- | --------- | -------- | ---- | ---------------------- |
| 0.1 Testing Infra | 3         | 2        | Low  | testify, gomock, rapid |
| 0.2 Char Tests    | 6         | 0        | Low  | v0 behavior captured   |
| 0.3 Core Domain   | 10        | 0        | Low  | Pure Go entities       |
| 0.4 Ports         | 10        | 0        | Low  | Interfaces + mocks     |

**Stage 0 Total: ~29 new files, 2 modified (go.mod, Makefile)**

### Stage 1: PLUGINIZE

| Phase                     | New Files | Modified | Risk | Deliverable              |
| ------------------------- | --------- | -------- | ---- | ------------------------ |
| 1.1 Plugin Infrastructure | 3         | 0        | Low  | Registry compiles        |
| 1.2 core.Plugin           | 6         | 0        | Low  | create/list/show work    |
| 1.3 work.Plugin           | 4         | 0        | Low  | ready/dep/blocked work   |
| 1.4 sync.Plugin           | 4         | 0        | Low  | sync/export/import work  |
| 1.5 Integration Plugins   | 3         | 0        | Low  | linear/molecules/compact |
| 1.6 Wire Main             | 0         | 1        | Low  | bd uses plugins          |

**Stage 1 Total: ~20 new files, 1 modified, same behavior**

### Stage 2: MODERNIZE

| Phase                 | New Files | Modified | Deleted | Risk   |
| --------------------- | --------- | -------- | ------- | ------ |
| 2.1 Adapters (impl)   | 6         | 0        | 0       | Medium |
| 2.2 Use Cases         | 8         | 0        | 0       | Medium |
| 2.3 Wire to v1 ports  | 0         | ~20      | 0       | Medium |
| 2.4 Validate + compat | 0         | 0        | 0       | Low    |
| 2.5 Cleanup v0 code   | 0         | 0        | ~30     | Low    |

**Stage 2 Total: ~14 new files, ~20 modified, ~30 deleted**

### Timeline Summary

| Stage             | Duration  | Files                | Outcome                |
| ----------------- | --------- | -------------------- | ---------------------- |
| **0: Foundation** | 7-12 days | ~29 new              | Safety net + contracts |
| **1: Pluginize**  | 5-7 days  | ~20 new              | v0 wrapped in plugins  |
| **2: Modernize**  | 7-10 days | ~14 new, ~30 deleted | v1 architecture        |

**Grand Total: ~63 new files, ~23 modified, ~30 deleted, 19-29 days**

## Consequences

### Positive

- **Testing-first safety** — Characterization tests catch regressions instantly
- **Always working software** — Every phase ships working code
- **Three safe checkpoints** — Stage 0 (tests), Stage 1 (plugins), Stage 2 (v1)
- **Parallel development** — Different plugins can be modernized concurrently
- **Low risk Stage 1** — v0 code unchanged, just wrapped
- **Incremental PRs** — Each phase is a reviewable PR
- **Ship early** — Can release v0-plugins (same behavior, better structure)
- **Future extensibility** — Plugin interface enables external plugins later
- **Better design** — TDD forces clean interfaces from the start

### Negative

- Temporary code duplication (plugin wrappers + v0 code)
- **Longer timeline** than original (~5-10 days for Stage 0)
- Must maintain wrapper code until Stage 2 complete
- Characterization tests need ongoing maintenance

### Mitigations

- Stage 1 wrappers are thin (minimal maintenance)
- Clear progress tracking per plugin
- Automated tests verify behavior unchanged
- Can ship Stage 1 as interim release

## References

- [Strangler Fig Pattern - Martin Fowler](https://martinfowler.com/bliki/StranglerFigApplication.html)
- [Internal Plugin Architecture - kubectl pattern](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [ADR 0002: Hybrid Architecture Patterns](0002-hybrid-architecture-patterns.md)
- [ADR 0005: Feature Flags](0005-feature-flags-go-feature-flag.md) — Gradual v0→v1 rollout
- [beads-v1-architecture.md](../beads-v1-architecture.md)
