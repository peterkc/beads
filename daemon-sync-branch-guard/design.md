# Design: Daemon Sync Branch Guard

## Architecture Decision

### Guard Placement Strategy

**Decision**: Add guard at implementation level, not wrapper level.

**Rationale**: The daemon has multiple wrapper functions that call shared implementations:

```
createExportFunc ─────┐
createLocalExportFunc ┼───► performExport ◄── ADD GUARD HERE
```

Adding the guard to `performExport`, `performAutoImport`, `performSync` ensures consistent protection regardless of which wrapper calls them.

### Guard Function Reuse

**Decision**: Use existing `syncbranch.IsSyncBranchSameAsCurrent()` function.

**Rationale**:
- Already tested (worktree_divergence_test.go)
- Handles edge cases (non-git repos, detached HEAD)
- Fail-open behavior matches safety requirements

### Fail-Open Pattern

**Decision**: If branch detection fails, allow operation to proceed.

**Rationale**:
- Matches existing sync.go guard behavior
- Better to allow operation than block valid usage
- Edge cases (detached HEAD, non-git) should not break daemon

## Test Matrix

### Comprehensive Scenario Coverage

| # | Scenario | sync-branch | current-branch | Expected | Notes |
|---|----------|-------------|----------------|----------|-------|
| 1 | Normal config | `beads-sync` | `main` | Allow | Standard usage |
| 2 | Same branch (config) | `main` | `main` | Block | The bug case |
| 3 | Same branch (env) | `main` (env) | `main` | Block | BEADS_SYNC_BRANCH override |
| 4 | No sync-branch | (not set) | `main` | Allow | Fallback to regular sync |
| 5 | Detached HEAD | `beads-sync` | (detached) | Allow | Fail-open |
| 6 | Non-git directory | N/A | N/A | Allow | Fail-open |
| 7 | Local-only mode | `beads-sync` | `main` | Allow | --local flag skips sync-branch |
| 8 | Worktree context | `beads-sync` | `feature-x` | Allow | In worktree, different branch |
| 9 | Worktree same branch | `feature-x` | `feature-x` | Block | Worktree on sync-branch |

### Dynamic Branch Change Scenarios

| # | Scenario | Start State | Action | End State | Expected |
|---|----------|-------------|--------|-----------|----------|
| 10 | Switch TO sync-branch | on `main`, sync=`main` | daemon export | - | Block (checked each op) |
| 11 | Switch FROM sync-branch | on `sync`, sync=`sync` | checkout `main` → export | - | Allow (re-checked) |
| 12 | Config hot reload | sync=`A` | change config to sync=`B` | - | Uses new value |

### Entry Point Coverage

| # | Entry Point | Same-Branch | Different-Branch |
|---|-------------|-------------|------------------|
| 13 | `performExport` | Block + log | Allow |
| 14 | `performAutoImport` | Block + log | Allow |
| 15 | `performSync` | Block + log | Allow |
| 16 | `syncBranchCommitAndPushWithOptions` | Skip + log | Allow |
| 17 | `syncBranchPull` | Skip + log | Allow |

### Startup Validation

| # | Scenario | Config | Expected |
|---|----------|--------|----------|
| 18 | Misconfigured at startup | sync=`main`, on `main` | Warn log, continue |
| 19 | Valid at startup | sync=`sync`, on `main` | No warning |
| 20 | No sync-branch | (not set) | No warning |

## Implementation Approach

### Helper Function

Create a shared helper to avoid code duplication:

```go
// shouldSkipDueToSameBranch checks if operation should be skipped because
// sync-branch == current-branch. Returns true if should skip, logs reason.
func shouldSkipDueToSameBranch(ctx context.Context, store storage.Storage, operation string, log daemonLogger) bool {
    syncBranch, err := syncbranch.Get(ctx, store)
    if err != nil || syncBranch == "" {
        return false // No sync branch configured, allow
    }

    if syncbranch.IsSyncBranchSameAsCurrent(ctx, syncBranch) {
        log.log("Skipping %s: sync-branch '%s' is your current branch. Use a dedicated sync branch.", operation, syncBranch)
        return true
    }

    return false
}
```

### Guard Placement

```go
func performExport(...) func() {
    return func() {
        // ADD: Early return if same branch
        if shouldSkipDueToSameBranch(ctx, store, "export", log) {
            return
        }
        // ... existing code ...
    }
}
```

## Error Messages

### User-Facing Messages

| Context | Message |
|---------|---------|
| Daemon log (operation blocked) | `Skipping {op}: sync-branch '{branch}' is your current branch. Use a dedicated sync branch.` |
| Daemon startup (warning) | `Warning: sync-branch '{branch}' is your current branch. Daemon sync operations will be skipped. Configure a dedicated sync branch (e.g., 'beads-sync') to enable sync.` |

## Existing Guard Reference

The guard at `sync.go:348` provides the pattern:

```go
if syncbranch.IsSyncBranchSameAsCurrent(ctx, sbc.Branch) {
    return fmt.Errorf("Cannot sync to '%s': it's your current branch. "+
        "When sync.branch equals your working branch, bd sync would overwrite "+
        "your uncommitted changes.\n\n"+
        "Solutions:\n"+
        "1. Use a dedicated sync branch: bd config set sync.branch beads-sync\n"+
        "2. Or remove sync.branch config: bd config unset sync.branch", sbc.Branch)
}
```

The daemon version should log rather than return error, since daemon operations are background tasks.
