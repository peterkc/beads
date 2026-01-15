# Design

## bd sync Call Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           bd sync Call Flow                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  User runs: bd sync                                                         │
│       │                                                                     │
│       ▼                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ sync.go: syncCmd.Run()                                              │   │
│  │   • Force direct mode (close daemon connection)                     │   │
│  │   • Find JSONL path                                                 │   │
│  │   • Check for sync-branch config                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│       │                                                                     │
│       ├──────────────────────┬───────────────────────────────────┐         │
│       │                      │                                   │         │
│       ▼                      ▼                                   ▼         │
│  ┌─────────────┐    ┌─────────────────────┐    ┌─────────────────────────┐ │
│  │ Normal sync │    │ Sync-branch mode    │    │ From-main mode          │ │
│  │ (no config) │    │ (sync.branch set)   │    │ (ephemeral branches)    │ │
│  └──────┬──────┘    └──────────┬──────────┘    └────────────┬────────────┘ │
│         │                      │                            │              │
│         ▼                      ▼                            ▼              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ doPullFirstSync()                                                   │   │
│  │   1. Load local state from DB                                       │   │
│  │   2. Load base state (sync_base.jsonl)                              │   │
│  │   3. Pull from remote ────────────┐                                 │   │
│  │   4. Load remote state            │                                 │   │
│  │   5. 3-way merge                  ▼                                 │   │
│  │   6. Import merged state    ┌─────────────────────────────────┐     │   │
│  │   7. Export to JSONL        │ Sync-branch: Uses worktree      │     │   │
│  │   8. Commit & push          │ syncbranch.PullFromSyncBranch() │     │   │
│  │   9. Update base state      │ syncbranch.CommitToSyncBranch() │     │   │
│  │                             └─────────────────────────────────┘     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ╔═══════════════════════════════════════════════════════════════════════╗ │
│  ║ DEAD CODE (v0.46.0 - removed in v0.47.0):                             ║ │
│  ║                                                                        ║ │
│  ║   // Was called at sync.go:910 in v0.46.0                             ║ │
│  ║   if useSyncBranch {                                                  ║ │
│  ║       restoreBeadsDirFromBranch(ctx)  ◄── REMOVED BY PR #918          ║ │
│  ║   }                                                                    ║ │
│  ║                                                                        ║ │
│  ║   // This ran: git checkout HEAD -- .beads/                           ║ │
│  ║   // Restored ENTIRE .beads/ including config.yaml ◄── THE BUG        ║ │
│  ╚═══════════════════════════════════════════════════════════════════════╝ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Worktree Data Flow (Current - Post v0.47.0)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Worktree Sync Flow (Current)                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Main Repo (.beads/)              Worktree (.git/beads-worktrees/sync/)     │
│  ┌───────────────────┐           ┌───────────────────────────────────────┐  │
│  │ config.yaml       │           │ .beads/                               │  │
│  │ issues.jsonl      │◄──────────┤   issues.jsonl  ◄── Only JSONL copied │  │
│  │ metadata.json     │           │   metadata.json                       │  │
│  │ beads.db          │           │                                       │  │
│  │ sync_base.jsonl   │           │ (config.yaml NOT copied back)         │  │
│  └───────────────────┘           └───────────────────────────────────────┘  │
│                                                                             │
│  ✓ Config.yaml stays in main repo, never touched by sync                    │
│  ✓ Only data files (JSONL, metadata) flow through worktree                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Current State

```
┌─────────────────────────────────────────────────────────┐
│ cmd/bd/sync_git.go                                      │
├─────────────────────────────────────────────────────────┤
│ Line 515-540: restoreBeadsDirFromBranch()               │
│   - Defined but NEVER CALLED                            │
│   - Was called in v0.46.0 at sync.go:910                │
│   - Call site removed in PR #918 (pull-first refactor)  │
│   - Function left behind as dead code                   │
└─────────────────────────────────────────────────────────┘
```

## Proposed Change

```
┌─────────────────────────────────────────────────────────┐
│ Phase 1: Delete Dead Code                               │
├─────────────────────────────────────────────────────────┤
│ 1. Remove lines 515-540 from sync_git.go                │
│ 2. Verify no compilation errors                         │
│ 3. Run existing tests                                   │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Phase 2: Regression Test                                │
├─────────────────────────────────────────────────────────┤
│ TestConfigPreservedDuringSync:                          │
│   1. Create bare repo + clone                           │
│   2. Configure sync-branch in clone                     │
│   3. Modify config.yaml (uncommitted)                   │
│   4. Run bd sync                                        │
│   5. Assert config.yaml unchanged                       │
└─────────────────────────────────────────────────────────┘
```

## Test Strategy

### Test Fixture Pattern

Follow existing patterns from `sync_modes_test.go` and `syncbranch_e2e_test.go`:

```go
func TestConfigPreservedDuringSync(t *testing.T) {
    // Setup: bare repo as "remote"
    bareDir := t.TempDir()
    initBareRepo(t, bareDir)

    // Clone and configure sync-branch
    cloneDir := t.TempDir()
    cloneRepo(t, bareDir, cloneDir)
    runBd(t, cloneDir, "config", "set", "sync.branch", "beads-sync")

    // Modify config.yaml AFTER setting sync-branch
    configPath := filepath.Join(cloneDir, ".beads", "config.yaml")
    originalContent := readFile(t, configPath)
    modifyConfig(t, configPath, "test-marker: true")

    // Run sync
    runBd(t, cloneDir, "sync")

    // Assert config preserved
    afterContent := readFile(t, configPath)
    require.Contains(t, afterContent, "test-marker: true")
}
```

### Test Matrix

| Scenario | Config Change | Sync Mode | Expected |
|----------|---------------|-----------|----------|
| Uncommitted sync-branch | `sync-branch: X` | sync-branch | Preserved |
| Uncommitted custom key | `test-key: Y` | sync-branch | Preserved |
| Committed config | N/A | sync-branch | No change |
| Normal sync (no sync-branch) | Any | normal | N/A (different code path) |

## Key Decisions

### KD-001: Minimal Test Scope

**Decision**: Test only sync-branch mode, not all sync modes.

**Rationale**: The bug only affected sync-branch mode (the removed call was inside
`if useSyncBranch {}`). Testing other modes adds complexity without value.

### KD-002: Single Test Function

**Decision**: One comprehensive test, not multiple small tests.

**Rationale**: The regression risk is a single code path. One focused test with
clear assertions is easier to maintain than scattered tests.
