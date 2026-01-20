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
       │   (unaffected by v1 work)
       │
       └── peterkc/beads (fork)
              │
              ├── main              # Tracks upstream/main (v0)
              │
              └── v1/develop        # Integration branch
                     │
                     ├── v1/phase-1   # Interface segregation
                     ├── v1/phase-2   # Row mapper DRY
                     ├── v1/phase-3   # v1 adapters
                     ├── v1/phase-4   # Event bus
                     ├── v1/phase-5   # Use cases
                     ├── v1/phase-6   # Migrate callers
                     ├── v1/phase-7   # Plugins
                     └── v1/phase-8   # Swap/cleanup
```

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
2. **Develop phases**: Work in `v1/phase-*` branches
3. **Integrate**: Merge phases into `v1/develop` for testing
4. **Rebase before PR**: Keep phases rebased on latest `main`
5. **PR to upstream**: When phase is stable, PR to `steveyegge/beads`
6. **Final PR**: Rename `bdx` → `bd` when v1.0 ships

### Automated Sync (GitHub Action)

A GitHub Action automates keeping the fork in sync with upstream:

```yaml
# .github/workflows/sync-upstream.yml
# See: research/system-diagrams/workflows/sync-upstream.yml

Schedule: Daily at 6 AM UTC
Manual: workflow_dispatch with rebase_v1 option

Jobs:
1. sync-main      → Merge upstream/main into fork's main
2. rebase-v1      → Rebase v1/develop on updated main
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
| Integration testing | `v1/develop` validates all phases together |

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

## Migration Phases

### Phase 1: Interface Segregation

**Goal:** Define v1 interfaces, no behavior change

**New files to create:**

```
internal/ports/                          # NEW directory
├── issue_repository.go                  # 5 methods
├── dependency_repository.go             # 4 methods
├── work_repository.go                   # 3 methods
├── config_store.go                      # 3 methods
├── sync_tracker.go                      # 5 methods
└── event_bus.go                         # 2 methods
```

**Reference (split this):**

```
internal/storage/storage.go              # 62-method interface → split into ports/*
```

**Risk:** Low (additive only)
**Upstream PR:** Yes — non-breaking, just new interfaces

---

### Phase 2: Row Mapper DRY

**Goal:** Eliminate 213 duplicate row scanning patterns across 73 files

**New file:**

```
internal/adapters/sqlite/row_mapper.go   # Generic[T] scanner
```

**Files to migrate (highest scan counts first):**

```
# High priority (10+ scans each)
internal/storage/sqlite/queries.go           # 15 scans
internal/storage/sqlite/transaction.go       # 11 scans
internal/storage/sqlite/events.go            # 10 scans
internal/storage/sqlite/migration_invariants.go # 10 scans
internal/storage/sqlite/dependencies.go      # 9 scans

# Medium priority (3-9 scans)
internal/storage/sqlite/multirepo.go         # 7 scans
internal/storage/sqlite/comments.go          # 6 scans
internal/storage/sqlite/ready.go             # 5 scans
internal/storage/sqlite/ids.go               # 4 scans
internal/storage/sqlite/adaptive_length.go   # 4 scans

# Lower priority (1-3 scans, 58 more files)
internal/storage/sqlite/labels.go            # 2 scans (good first target - simple)
internal/storage/sqlite/dirty.go             # 3 scans
internal/storage/sqlite/config.go            # 3 scans
internal/storage/sqlite/batch_ops.go         # 3 scans
internal/storage/sqlite/resurrection.go      # 3 scans
internal/storage/sqlite/compact.go           # 3 scans
# ... plus 40 migration files with 1-5 scans each
```

**Risk:** Low (internal refactor, behavior unchanged)
**Upstream PR:** Yes — reduces boilerplate

---

### Phase 3: v1 Adapter Implementations

**Goal:** Create v1 implementations alongside v0

**New files (versioned):**

```
internal/adapters/                       # NEW directory structure
├── sqlite/
│   ├── issue_repo_v1.go                 # Implements ports.IssueRepository
│   ├── dependency_repo_v1.go            # Implements ports.DependencyRepository
│   ├── work_repo_v1.go                  # Implements ports.WorkRepository
│   ├── config_store_v1.go               # Implements ports.ConfigStore
│   └── sync_tracker_v1.go               # Implements ports.SyncTracker
├── memory/
│   └── issue_repo_v1.go                 # For testing
└── dolt/
    └── issue_repo_v1.go                 # If Dolt support continues
```

**Existing files (unchanged for now):**

```
internal/storage/sqlite/*.go             # v0 stays working
internal/storage/memory/memory.go        # v0 stays working
internal/storage/dolt/*.go               # v0 stays working
```

**Risk:** Low (additive, v0 unchanged)
**Upstream PR:** Yes — provides v1 adapters

---

### Phase 4: Event Bus

**Goal:** Add event-driven architecture alongside hooks

**New files:**

```
internal/events/
├── bus.go                               # EventBus interface + impl
├── events.go                            # IssueCreated, IssueUpdated, etc.
└── subscribers/
    ├── audit.go                         # Logging subscriber
    └── hooks.go                         # Existing hooks as subscriber
```

**Files to modify:**

```
internal/hooks/hooks.go                  # Becomes event subscriber
```

**Risk:** Medium (new system, but additive)
**Upstream PR:** Yes — enables loose coupling

---

### Phase 5: Use Cases Layer

**Goal:** Extract business logic from RPC into use cases

**New files:**

```
internal/usecases/
├── issue_ops.go                         # CreateIssue, UpdateIssue, CloseIssue
├── work_ops.go                          # GetReadyWork, MarkInProgress
├── sync_ops.go                          # Export, Import, AutoSync
└── dependency_ops.go                    # AddDep, RemoveDep, DetectCycles
```

**Files to refactor (extract logic from):**

```
internal/rpc/server_issues_epics.go      # 20 methods → use cases
internal/rpc/server_labels_deps_comments.go  # 9 methods → use cases
internal/rpc/server_export_import_auto.go    # 4 methods → use cases
internal/rpc/server_core.go              # 8 methods → use cases
internal/rpc/client.go                   # 41 methods (thins down)
```

**Risk:** Medium (refactoring, but behavior unchanged)
**Upstream PR:** Batched — one package at a time

---

### Phase 6: Migrate Callers to v1

**Goal:** Switch callers from v0 Storage to v1 interfaces

**Files to migrate (by dependency order):**

```
# Layer 1: Internal utilities (least dependencies)
internal/autoimport/autoimport.go
internal/export/export.go

# Layer 2: Features
internal/compact/compactor.go
internal/molecules/molecules.go
internal/linear/linear.go

# Layer 3: Sync
internal/importer/importer.go
internal/syncbranch/syncbranch.go

# Layer 4: RPC (depends on use cases now)
internal/rpc/server_*.go                 # Use usecases, not Storage directly

# Layer 5: CLI
cmd/bd/*.go                              # Use usecases via RPC
```

**Config coupling to fix (13 files):**

```
internal/storage/sqlite/ready.go
internal/storage/sqlite/multirepo_export.go
internal/storage/sqlite/hash_ids.go
internal/storage/sqlite/multirepo.go
internal/storage/sqlite/external_deps.go
internal/storage/memory/memory.go
internal/importer/importer.go
internal/syncbranch/syncbranch.go
# ... (5 more)
```

**Risk:** Medium (many files, but incremental)
**Upstream PR:** Batched — reviewable chunks

---

### Phase 7: Plugin Architecture

**Goal:** Extract integrations as plugins

**Files to extract:**

```
# Move to plugins/
internal/linear/*.go          → plugins/linear/
internal/compact/*.go         → plugins/compact/
internal/molecules/*.go       → plugins/molecules/
```

**New files:**

```
internal/plugins/
├── registry.go                          # Plugin discovery
├── api.go                               # PluginContext, BeadsPlugin interface
└── loader.go                            # go-plugin integration
```

**Risk:** Medium (architectural change)
**Upstream PR:** Feature PR — plugin system

---

### Phase 8: Swap and Cleanup

**Goal:** Remove v0, finalize v1

**Swap operations:**

```
# Rename v1 → production
internal/adapters/sqlite/issue_repo_v1.go    → issue_repo.go
internal/adapters/sqlite/dependency_repo_v1.go → dependency_repo.go
# ... etc

# Delete v0
internal/storage/storage.go              # DELETE (62-method interface)
internal/storage/sqlite/*.go             # DELETE (old implementations)
internal/storage/memory/memory.go        # DELETE (old implementation)
```

**Risk:** Low (all callers already on v1)
**Upstream PR:** Breaking change — v1.0.0 release

---

## File Count Summary

| Phase | New Files | Modified | Deleted | Risk |
|-------|-----------|----------|---------|------|
| 1. Interfaces | 6 | 0 | 0 | Low |
| 2. Row Mapper | 1 | 73 | 0 | Low |
| 3. v1 Adapters | 8 | 0 | 0 | Low |
| 4. Event Bus | 4 | 1 | 0 | Medium |
| 5. Use Cases | 4 | 6 | 0 | Medium |
| 6. Migrate Callers | 0 | ~25 | 0 | Medium |
| 7. Plugins | 3 | 0 | 0 | Medium |
| 8. Swap/Cleanup | 0 | ~10 | ~30 | Low |

**Total: ~26 new files, ~115 modified, ~30 deleted**

*Note: Phase 2 file count from `grep -c 'rows\.Scan\|\.Scan\(' internal/storage/sqlite/*.go`*

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
