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

## Migration Strategy

### Three Stages

| Stage             | Goal                         | Deliverable                         | Duration  |
| ----------------- | ---------------------------- | ----------------------------------- | --------- |
| **1: Foundation** | Safety net + contracts       | Tests, core domain, port interfaces | 7-12 days |
| **2: Pluginize**  | Same behavior, new structure | v0 wrapped in plugin architecture   | 5-7 days  |
| **3: Modernize**  | Replace internals            | v1 adapters, cleanup v0             | 7-10 days |

```
Foundation ──► Pluginize ──► Modernize
(tests+ports)   (wrap v0)    (replace v0)
```

**Key principles:**

- **Testing-first** — Characterization tests capture v0 behavior before changes
- **Always shippable** — Each stage produces working software
- **Parallel-friendly** — Plugins can be modernized independently

**See:** [docs/implementation-plan.md](../docs/implementation-plan.md) for detailed execution plan.

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

## Timeline Summary

| Stage             | Duration  | Files                | Outcome                |
| ----------------- | --------- | -------------------- | ---------------------- |
| **1: Foundation** | 7-12 days | ~29 new              | Safety net + contracts |
| **2: Pluginize**  | 5-7 days  | ~20 new              | v0 wrapped in plugins  |
| **3: Modernize**  | 7-10 days | ~14 new, ~30 deleted | v1 architecture        |

**Total:** ~63 new files, ~23 modified, ~30 deleted, 19-29 days

**Detailed execution plan:** [docs/implementation-plan.md](../docs/implementation-plan.md)

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

### Abort Procedures

Each stage has an explicit rollback path if migration needs to stop:

| Stage             | Abort Complexity | Procedure                                                   |
| ----------------- | ---------------- | ----------------------------------------------------------- |
| **0: Foundation** | Trivial          | Delete `internal/next/` directory, remove test dependencies |
| **1: Pluginize**  | Easy             | Revert `cmd/bdx/main.go`, delete `internal/next/plugins/`   |
| **2.1-2.3**       | Medium           | Keep `internal/v0/`, revert plugin imports to v0            |
| **2.4-2.5**       | Hard             | Restore `internal/v0/` from git, revert plugin wiring       |
| **Post-Ship**     | Complex          | Maintain `v0-archive` branch, document downgrade path       |

**Stage 0 Abort:**

```bash
# Foundation is additive only - safe to delete
rm -rf internal/next/
git checkout go.mod go.sum  # Remove test dependencies
```

**Stage 1 Abort:**

```bash
# Plugins wrap v0 - v0 code is unchanged
git revert <plugin-commits>
# bd still works, bdx removed
```

**Stage 2 Abort (before 2.5):**

```bash
# v0 code still exists in internal/v0/
# Revert plugin imports from next/ back to v0/
git revert <stage-2-commits>
```

**Post-Ship Abort (emergency):**

```bash
# If v1 has critical bugs after replacing main
git checkout v0-archive
git checkout -b hotfix/critical-bug
# Fix in v0, release as patch
# Document: "v1.0.1 requires v0 hotfix due to X"
```

**Decision Point:** Before Stage 2.5 (cleanup), explicitly confirm:

- [ ] All characterization tests pass
- [ ] Performance benchmarks acceptable
- [ ] No regressions in user workflows
- [ ] Stakeholder sign-off on "point of no return"

## References

- [Strangler Fig Pattern - Martin Fowler](https://martinfowler.com/bliki/StranglerFigApplication.html)
- [Internal Plugin Architecture - kubectl pattern](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [ADR 0002: Hybrid Architecture Patterns](0002-hybrid-architecture-patterns.md)
- [ADR 0005: Feature Flags](0005-feature-flags-go-feature-flag.md) — Gradual v0→v1 rollout
- [beads-v1-architecture.md](../beads-v1-architecture.md)
