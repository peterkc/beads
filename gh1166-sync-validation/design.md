# Design: GH#1166 Sync-Branch Validation

## Problem Analysis

### Root Cause

Two validation gaps allow `sync.branch = main` despite intended rejection:

1. **Config validation bypass**: `validateYamlConfigValue()` in `yaml_config.go` only validates `hierarchy.max-depth`, not sync-branch keys
2. **Runtime validation missing**: `bd sync` doesn't check if sync-branch equals current branch before entering worktree

### Code Path Analysis

```
bd config set sync.branch main
    │
    ▼
config.go:59 ── IsYamlOnlyKey("sync.branch") → true
    │
    ▼
config.SetYamlConfig() ── validateYamlConfigValue() ── ONLY checks hierarchy.max-depth!
    │
    ▼
Returns success ── sync.branch = main saved to config.yaml
```

The database path (`syncbranch.Set()`) correctly calls `ValidateSyncBranchName()`, but YAML-only keys bypass this entirely.

## Existing Utilities

Both required validation functions already exist:

| Function | Location | Purpose |
|----------|----------|---------|
| `ValidateSyncBranchName(name)` | `syncbranch.go:72` | Rejects main/master with clear error |
| `IsSyncBranchSameAsCurrent(ctx, name)` | `worktree.go:1141` | Compares sync-branch to current branch |

**Key insight**: The utilities exist and are tested—they just aren't wired into the right places.

## Solution Design

### Fix 1: Config-Time Validation

Add sync-branch case to `validateYamlConfigValue()`:

```go
// yaml_config.go
func validateYamlConfigValue(key, value string) error {
    switch key {
    case "hierarchy.max-depth":
        // existing validation...
    case "sync-branch", "sync.branch":
        return syncbranch.ValidateSyncBranchName(value)
    }
    return nil
}
```

**Trade-off**: Adds import of `syncbranch` package to `config` package.

- Acceptable: `config` already imports other internal packages
- Alternative: Inline the validation (rejected—violates DRY)

### Fix 2: Runtime Validation

Add check in `sync.go` after fetching sync-branch config, before worktree operations:

```go
// sync.go, after line 247
hasSyncBranchConfig := syncBranchName != ""
if hasSyncBranchConfig {
    if syncbranch.IsSyncBranchSameAsCurrent(ctx, syncBranchName) {
        FatalError("Cannot sync to '%s': it's your current branch. "+
            "Checkout a different branch first, or use a dedicated sync branch like 'beads-sync'.",
            syncBranchName)
    }
}
```

**Why this location**: At line 247, CWD is still user's working directory. After `CommitToSyncBranch()`, CWD changes to worktree and `GetCurrentBranch()` would return wrong value.

### Reference Implementation

`bd doctor` already implements the runtime check pattern:

```go
// doctor/git.go:611
if syncBranch != "" && currentBranch == syncBranch {
    // Creates warning/error
}
```

## Key Decisions

### KD-1: Reuse vs. New Utilities

**Decision**: Reuse existing `ValidateSyncBranchName()` and `IsSyncBranchSameAsCurrent()`

**Rationale**:
- Already tested (see `syncbranch_test.go`, `worktree_divergence_test.go`)
- Single source of truth for validation logic
- Follows DRY principle

### KD-2: Config Package Import

**Decision**: Add `syncbranch` import to `config` package

**Rationale**:
- Minimal coupling (single function call)
- Alternative (inline validation) duplicates logic
- Pattern established by other internal package imports

### KD-3: Error Message Style

**Decision**: Match existing error message patterns in beads

**Rationale**:
- Consistency with `ValidateSyncBranchName()` error format
- Include actionable suggestion (use dedicated branch)

## Before/After Diagrams

### BEFORE: Config Path (Bug)

```
bd config set sync.branch main
         |
         v
    config.go:59
    IsYamlOnlyKey("sync.branch")
         |
         v (true)
    SetYamlConfig()
         |
         v
    validateYamlConfigValue()
         |
         +---> case "hierarchy.max-depth": validate
         |
         +---> default: return nil    <-- BUG: sync-branch not checked!
         |
         v
    config.yaml updated
    sync.branch = main   <-- SAVED (should have been rejected)
```

### AFTER: Config Path (Fixed)

```
bd config set sync.branch main
         |
         v
    config.go:59
    IsYamlOnlyKey("sync.branch")
         |
         v (true)
    SetYamlConfig()
         |
         v
    validateYamlConfigValue()
         |
         +---> case "hierarchy.max-depth": validate
         |
         +---> case "sync-branch", "sync.branch":    <-- NEW
         |              |
         |              v
         |     syncbranch.ValidateSyncBranchName(value)
         |              |
         |              v
         |     ERROR: "cannot use 'main' as sync branch"
         |
         v
    Command exits with error   <-- CORRECT
```

### BEFORE: Runtime Path (Bug)

```
bd sync (user on main, sync.branch=main)
         |
         v
    sync.go:240-247
    syncBranchName = syncbranch.Get()  --> "main"
    hasSyncBranchConfig = true
         |
         v (no check!)
    CommitToSyncBranch()
         |
         v
    Creates worktree for "main"
         |
         v
    ERROR: "main is already checked out"   <-- CONFUSING
    (or worse: commits all staged files)
```

### AFTER: Runtime Path (Fixed)

```
bd sync (user on main, sync.branch=main)
         |
         v
    sync.go:240-247
    syncBranchName = syncbranch.Get()  --> "main"
    hasSyncBranchConfig = true
         |
         v
    IsSyncBranchSameAsCurrent(ctx, "main")   <-- NEW CHECK
         |
         v (true - matches current branch)
    FatalError: "Cannot sync to 'main': it's your current branch.
                 Checkout a different branch first."
         |
         v
    Command exits cleanly   <-- CORRECT (clear message)
```

---

## Test Matrix

### Config-Time Validation Tests

| Test ID | Input | Current Branch | Expected Result | Validates |
|---------|-------|----------------|-----------------|-----------|
| C-01 | `sync.branch = main` | any | ERROR: cannot use main | Fix 1 |
| C-02 | `sync.branch = master` | any | ERROR: cannot use master | Fix 1 |
| C-03 | `sync-branch = main` | any | ERROR: cannot use main | Fix 1 (alias) |
| C-04 | `sync.branch = beads-sync` | any | SUCCESS | No regression |
| C-05 | `sync.branch = feature/sync` | any | SUCCESS | No regression |
| C-06 | `sync.branch = ""` | any | SUCCESS (unset) | No regression |
| C-07 | `hierarchy.max-depth = 5` | any | SUCCESS | Existing validation |
| C-08 | `hierarchy.max-depth = -1` | any | ERROR | Existing validation |

### Runtime Validation Tests

| Test ID | sync.branch | Current Branch | Expected Result | Validates |
|---------|-------------|----------------|-----------------|-----------|
| R-01 | `main` | `main` | ERROR: on sync branch | Fix 2 |
| R-02 | `main` | `feature/foo` | SUCCESS (sync proceeds) | No regression |
| R-03 | `beads-sync` | `main` | SUCCESS (sync proceeds) | No regression |
| R-04 | `beads-sync` | `beads-sync` | ERROR: on sync branch | Fix 2 |
| R-05 | (not set) | `main` | SUCCESS (no sync-branch mode) | No regression |
| R-06 | `feature/x` | `feature/x` | ERROR: on sync branch | Fix 2 (dynamic) |

### Edge Cases

| Test ID | Scenario | Expected Result | Notes |
|---------|----------|-----------------|-------|
| E-01 | Detached HEAD + sync.branch=main | SUCCESS | No current branch to match |
| E-02 | Config set via YAML edit (bypass CLI) | Runtime catches it | Defense in depth |
| E-03 | Config set before fix, upgrade beads | Runtime catches it | Backward compat |
| E-04 | sync.branch with trailing whitespace | Normalized, then validated | Edge case |

---

## Test Strategy

### Unit Tests

1. **Config validation test**: `yaml_config_test.go`
   - Test `validateYamlConfigValue("sync.branch", "main")` returns error
   - Test `validateYamlConfigValue("sync-branch", "master")` returns error
   - Test `validateYamlConfigValue("sync.branch", "beads-sync")` returns nil

2. **Runtime check integration**: Extend existing sync tests
   - Test sync fails when on sync-branch
   - Test sync succeeds when on different branch

### Manual Verification

```bash
# Should fail at config time
bd config set sync.branch main
# Expected: Error about main/master not allowed

# Should fail at runtime (if config manually edited)
# Edit config.yaml: sync.branch: main
git checkout main
bd sync
# Expected: Error about being on sync branch
```

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Import cycle with syncbranch | Verified: config → syncbranch is safe (no reverse dep) |
| Breaking existing workflows | Tests ensure valid configs still work |
| Performance impact | Negligible: single string comparison |
