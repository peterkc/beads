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

## Migration Phases (Scaffold-First)

### Scaffold-First Strategy

Instead of sequential phases, create the **complete v1 structure upfront** with graceful error stubs:

```
SEQUENTIAL (old):                    SCAFFOLD-FIRST (new):
─────────────────                    ────────────────────
Phase 1 → Phase 2 → ...              Phase 1: Create ALL stubs
      (serial)                              ↓
                                     Phase 2-N: Fill in stubs
                                          (parallel!)
```

**Benefits:**
- Architecture locked early (can't drift)
- Parallel development (anyone can fill any stub)
- Easy contributor sync (bd commit → map to bdx location)
- CI validates structure (compiles even with stubs)
- Clear progress tracking (which stubs are done?)

### Stub Pattern (Graceful Errors)

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

### Phase 1: SCAFFOLD

**Goal:** Create complete v1 directory structure with graceful error stubs

**Directory structure:**

```
cmd/bdx/                              # CLI entry point
├── main.go                           # Minimal, calls root command
└── commands/                         # Cobra commands (stubs)

internal/
├── core/                             # Domain (pure Go, no deps)
│   ├── issue/                        # Issue entity + value objects
│   ├── dependency/                   # Dependency value object
│   ├── label/                        # Label value object
│   └── work/                         # Ready/blocked logic
│
├── ports/                            # Interfaces (REAL - define contracts)
│   ├── errors.go                     # ErrNotImplemented
│   ├── issue_repository.go           # 5 methods
│   ├── dependency_repository.go      # 4 methods
│   ├── work_repository.go            # 3 methods
│   ├── config_store.go               # 3 methods
│   ├── sync_tracker.go               # 5 methods
│   └── event_bus.go                  # 2 methods
│
├── adapters/                         # Implementations (stubs initially)
│   ├── sqlite/
│   │   ├── issue_repo.go             # Stub → implements ports.IssueRepository
│   │   ├── dependency_repo.go        # Stub
│   │   ├── work_repo.go              # Stub
│   │   ├── config_store.go           # Stub
│   │   ├── sync_tracker.go           # Stub
│   │   └── row_mapper.go             # Generic scanner (implement early)
│   ├── jsonl/
│   │   ├── exporter.go               # Stub
│   │   └── importer.go               # Stub
│   ├── git/
│   │   └── integration.go            # Stub
│   └── memory/
│       └── issue_repo.go             # Stub (for testing)
│
├── usecases/                         # Business operations (stubs)
│   ├── issue_ops.go                  # CreateIssue, UpdateIssue, CloseIssue
│   ├── work_ops.go                   # GetReadyWork, MarkInProgress
│   ├── sync_ops.go                   # Export, Import, AutoSync
│   └── dependency_ops.go             # AddDep, RemoveDep, DetectCycles
│
├── events/                           # Event bus (stubs)
│   ├── bus.go                        # EventBus implementation
│   └── events.go                     # IssueCreated, IssueUpdated, etc.
│
└── plugins/                          # Plugin system (stubs)
    ├── api.go                        # PluginContext, BeadsPlugin interface
    └── registry.go                   # Plugin discovery
```

**Risk:** Low (all stubs, compiles but returns errors)
**Deliverable:** `bdx` binary that compiles, commands return "not implemented"

---

### Phase 2: IMPLEMENT CORE

**Goal:** Fill in domain types and ports (shared foundation)

**Files to implement:**

```
internal/core/issue/issue.go          # Issue entity
internal/core/issue/status.go         # Status enum + transitions
internal/core/dependency/dep.go       # Dependency value object
internal/core/work/ready.go           # Ready/blocked logic
internal/ports/*.go                   # Already done in scaffold
```

**Can port from:**
```
internal/types/types.go               # bd's current types
```

**Risk:** Low (pure Go, no I/O)

---

### Phase 3: IMPLEMENT ADAPTERS (Parallel!)

**Goal:** Fill in adapter stubs - can be done in parallel

| Adapter | Priority | Port From |
|---------|----------|-----------|
| `sqlite/row_mapper.go` | High | New (DRY helper) |
| `sqlite/issue_repo.go` | High | `storage/sqlite/queries.go` |
| `sqlite/dependency_repo.go` | High | `storage/sqlite/dependencies.go` |
| `sqlite/work_repo.go` | Medium | `storage/sqlite/ready.go` |
| `sqlite/config_store.go` | Medium | `storage/sqlite/config.go` |
| `sqlite/sync_tracker.go` | Medium | `storage/sqlite/dirty.go` |
| `memory/issue_repo.go` | Low | New (for testing) |
| `jsonl/exporter.go` | Medium | `export/export.go` |
| `jsonl/importer.go` | Medium | `importer/importer.go` |

**Risk:** Medium (most complex phase)

---

### Phase 4: IMPLEMENT USE CASES

**Goal:** Fill in business operation stubs

| Use Case | Port From |
|----------|-----------|
| `issue_ops.go` | `rpc/server_issues_epics.go` |
| `work_ops.go` | `rpc/server_core.go` |
| `sync_ops.go` | `rpc/server_export_import_auto.go` |
| `dependency_ops.go` | `rpc/server_labels_deps_comments.go` |

**Risk:** Medium

---

### Phase 5: WIRE CLI

**Goal:** Connect bdx commands to use cases

```
cmd/bdx/commands/
├── create.go      # bd create → usecases.CreateIssue
├── list.go        # bd list → usecases.ListIssues
├── show.go        # bd show → usecases.GetIssue
├── update.go      # bd update → usecases.UpdateIssue
├── close.go       # bd close → usecases.CloseIssue
├── ready.go       # bd ready → usecases.GetReadyWork
├── dep.go         # bd dep → usecases.DependencyOps
└── sync.go        # bd sync → usecases.SyncOps
```

**Risk:** Low (thin wrappers)

---

### Phase 6: PORT FEATURES

**Goal:** Migrate Linear, compact, molecules as plugins

```
plugins/
├── linear/        # Port from internal/linear/
├── compact/       # Port from internal/compact/
└── molecules/     # Port from internal/molecules/
```

**Risk:** Medium (plugin API must be stable)

---

### Phase 7: VALIDATE

**Goal:** Integration testing, .beads/ compatibility

- Run bdx against existing .beads/ databases
- Compare output with bd (should match)
- Performance benchmarks
- Edge case testing

**Risk:** Low (testing, not implementation)

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

| Phase | New Files | Stubs | Implemented | Risk |
|-------|-----------|-------|-------------|------|
| 1. Scaffold | ~35 | 35 | 0 | Low |
| 2. Core | 0 | -5 | 5 | Low |
| 3. Adapters | 0 | -10 | 10 | Medium |
| 4. Use Cases | 0 | -4 | 4 | Medium |
| 5. Wire CLI | ~10 | 0 | 10 | Low |
| 6. Plugins | ~6 | 0 | 6 | Medium |
| 7. Validate | 0 | 0 | 0 | Low |

**Total: ~51 new files, all stubs filled by end**

## Consequences

### Positive

- Always working software at every phase
- Each phase is reviewable PR
- Can pause at any phase if priorities shift
- Atomic swap at end (no gradual breakage)

### Negative

- Temporary code duplication (v0 + v1 coexist)
- Longer timeline than greenfield rewrite
- Must maintain both codepaths until Phase 8

### Mitigations

- Clear naming convention (`*_v1.go`)
- Track progress in beads issue
- Automated tests verify both paths work

## References

- [Strangler Fig Pattern - Martin Fowler](https://martinfowler.com/bliki/StranglerFigApplication.html)
- [ADR 0002: Hybrid Architecture Patterns](0002-hybrid-architecture-patterns.md)
- [beads-v1-architecture.md](../beads-v1-architecture.md)
