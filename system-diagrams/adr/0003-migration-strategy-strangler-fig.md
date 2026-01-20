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

| Option | Risk | Merge Back | Parallel Dev | v0 Contributors |
|--------|------|------------|--------------|-----------------|
| Feature Flags | Medium | Easy | No | Blocked |
| Parallel Fork (bdx) | Low initially, High later | Hard | Yes | Unaffected |
| Strangler Fig (in-place) | Low | Easy | Partial | Blocked |
| Strangler Fig (in fork) | Low | Medium | Yes | Unaffected |
| Big Bang | HIGH | N/A | No | Blocked |

## Decision

**Use Strangler Fig pattern in a fork with branch-based phases:**

1. Develop v1 in `peterkc/beads` fork (not upstream)
2. Create v1 implementations alongside v0 (versioned files)
3. Regularly sync fork's `main` with `upstream/main`
4. PR phases back to upstream when stable

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
              └── next              # Integration branch (upcoming major)
                     │
                     ├── next/phase-1   # Interface segregation
                     ├── next/phase-2   # Row mapper DRY
                     ├── next/phase-3   # Adapters
                     ├── next/phase-4   # Event bus
                     ├── next/phase-5   # Use cases
                     ├── next/phase-6   # Migrate callers
                     ├── next/phase-7   # Plugins
                     └── next/phase-8   # Swap/cleanup
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
2. **Develop phases**: Work in `next/phase-*` branches
3. **Integrate**: Merge phases into `next` for testing
4. **Rebase before PR**: Keep phases rebased on latest `main`
5. **PR to upstream**: When phase is stable, PR to `steveyegge/beads`
6. **Final PR**: Rename `bdx` → `bd` when v1.0 ships

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

| Benefit | How |
|---------|-----|
| v0 contributors unblocked | They work on upstream, we work on fork |
| No merge conflicts during dev | Isolated branch until PR time |
| Incremental PRs still possible | Each phase can be PR'd separately |
| Rollback easy | Fork can reset; upstream unchanged |
| Integration testing | `next` validates all phases together |

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

## Migration Phases (Plugin-First)

### Two-Stage Migration Strategy

Instead of rewriting from scratch, **wrap v0 code in plugins first**, then refactor internals:

```
STAGE 1: PLUGINIZE                   STAGE 2: MODERNIZE
──────────────────                   ───────────────────
v0 monolith                          v0-plugins
    │                                    │
    ▼                                    ▼
┌─────────────┐                     ┌─────────────┐
│ bd (v0)     │     Wrap in         │ bd (v0)     │     Refactor
│ ├─ create   │ ──────────────►     │ ├─ core.Plugin    │ ──────────►
│ ├─ list     │     plugin          │ │   └─ calls v0   │     to v1
│ ├─ linear   │     interfaces      │ ├─ linear.Plugin  │     ports
│ └─ ...      │                     │ │   └─ calls v0   │
└─────────────┘                     │ └─ ...            │
                                    └─────────────┘
Same behavior,                       Same behavior,       New architecture,
monolithic code                      plugin structure     clean internals
```

**Benefits:**
- v0 never breaks (plugins just wrap existing code)
- Architecture locked early (plugin interfaces are the contract)
- Each plugin can be refactored independently
- Can ship v0-plugins (same behavior) then gradually upgrade
- Clear progress tracking (which plugins are modernized?)

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

---

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

---

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

---

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

---

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

---

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

---

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

---

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

---

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

---

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

---

### Phase 2.3: Implement v1 Adapters (Parallel!)

**Goal:** Fill in adapter stubs - can be done in parallel per plugin

| Plugin | Adapter to Implement | Port From |
|--------|---------------------|-----------|
| core | `issue_repo.go` | `storage/sqlite/queries.go` |
| work | `work_repo.go` | `storage/sqlite/ready.go` |
| work | `dependency_repo.go` | `storage/sqlite/dependencies.go` |
| sync | `sync_tracker.go` | `storage/sqlite/dirty.go` |

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

---

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

---

### Phase 2.5: Validate and Cleanup

**Goal:** Integration testing, remove v0 code

- Run bdx against existing .beads/ databases
- Compare output with bd (should match)
- Performance benchmarks
- Delete `internal/storage/` (v0 code)

**Risk:** Low (testing and cleanup)

---

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

| bd (v0) | bdx (v1) |
|---------|----------|
| `internal/storage/storage.go` | `internal/ports/*.go` |
| `internal/storage/sqlite/queries.go` | `internal/adapters/sqlite/issue_repo.go` |
| `internal/storage/sqlite/dependencies.go` | `internal/adapters/sqlite/dependency_repo.go` |
| `internal/storage/sqlite/ready.go` | `internal/adapters/sqlite/work_repo.go` |
| `internal/storage/sqlite/config.go` | `internal/adapters/sqlite/config_store.go` |
| `internal/storage/sqlite/dirty.go` | `internal/adapters/sqlite/sync_tracker.go` |
| `internal/rpc/server_*.go` | `internal/usecases/*.go` |
| `internal/linear/` | `plugins/linear/` |
| `internal/compact/` | `plugins/compact/` |
| `internal/molecules/` | `plugins/molecules/` |
| `cmd/bd/` | `cmd/bdx/` |

---

## File Count Summary

### Stage 1: PLUGINIZE

| Phase | New Files | Modified | Risk | Deliverable |
|-------|-----------|----------|------|-------------|
| 1.1 Plugin Infrastructure | 3 | 0 | Low | Registry compiles |
| 1.2 core.Plugin | 6 | 0 | Low | create/list/show work |
| 1.3 work.Plugin | 4 | 0 | Low | ready/dep/blocked work |
| 1.4 sync.Plugin | 4 | 0 | Low | sync/export/import work |
| 1.5 Integration Plugins | 3 | 0 | Low | linear/molecules/compact |
| 1.6 Wire Main | 0 | 1 | Low | bd uses plugins |

**Stage 1 Total: ~20 new files, 1 modified, same behavior**

### Stage 2: MODERNIZE

| Phase | New Files | Modified | Deleted | Risk |
|-------|-----------|----------|---------|------|
| 2.1 Ports (interfaces) | 7 | 0 | 0 | Low |
| 2.2 Adapter stubs | 6 | 0 | 0 | Low |
| 2.3 Implement adapters | 0 | 6 | 0 | Medium |
| 2.4 Update PluginContext | 0 | ~20 | 0 | Medium |
| 2.5 Validate & cleanup | 0 | 0 | ~30 | Low |

**Stage 2 Total: ~13 new files, ~26 modified, ~30 deleted**

**Grand Total: ~33 new files, ~27 modified, ~30 deleted**

## Consequences

### Positive

- **Always working software** — Every phase ships working code
- **Two safe checkpoints** — Stage 1 (plugins) and Stage 2 (v1) are independently valuable
- **Parallel development** — Different plugins can be modernized concurrently
- **Low risk Stage 1** — v0 code unchanged, just wrapped
- **Incremental PRs** — Each phase is a reviewable PR
- **Ship early** — Can release v0-plugins (same behavior, better structure)
- **Future extensibility** — Plugin interface enables external plugins later

### Negative

- Temporary code duplication (plugin wrappers + v0 code)
- Longer timeline than greenfield rewrite
- Must maintain wrapper code until Stage 2 complete

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
