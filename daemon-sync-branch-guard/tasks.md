# Tasks: Daemon Sync Branch Guard

## Phase 1: Tracer Bullet

Add guard to one daemon entry point and verify the guard works end-to-end.

### Tasks

- [ ] Create `shouldSkipDueToSameBranch()` helper function in `daemon_sync.go`
- [ ] Add guard to `performExport()` function
- [ ] Write test: `TestDaemonExportSkipsSameBranch`
- [ ] Verify existing `TestDaemonExport*` tests still pass

### Validation

```bash
go test -v ./cmd/bd -run TestDaemonExport
go test -v ./cmd/bd -run TestDaemonExport -cover  # New code must be covered
```

---

## Phase 2: Complete Guards

Add guard to all remaining daemon entry points.

### Tasks

- [ ] Add guard to `performAutoImport()` in `daemon_sync.go`
- [ ] Add guard to `performSync()` in `daemon_sync.go`
- [ ] Add guard to `syncBranchCommitAndPushWithOptions()` in `daemon_sync_branch.go`
- [ ] Add guard to `syncBranchPull()` in `daemon_sync_branch.go`
- [P] Write tests for each entry point (can parallelize)
  - [ ] `TestDaemonAutoImportSkipsSameBranch`
  - [ ] `TestDaemonSyncSkipsSameBranch`
  - [ ] `TestSyncBranchCommitSkipsSameBranch`
  - [ ] `TestSyncBranchPullSkipsSameBranch`
- [P] Write edge case tests (scenarios 8-12 from test matrix)
  - [ ] `TestDaemonExportWorktreeDifferentBranch` (scenario 8)
  - [ ] `TestDaemonExportWorktreeSameBranch` (scenario 9)
  - [ ] `TestDaemonExportDynamicBranchSwitch` (scenario 10)
  - [ ] `TestDaemonExportAfterBranchChange` (scenario 11)
  - [ ] `TestDaemonExportConfigReload` (scenario 12)

### Validation

```bash
go test -v ./cmd/bd -run "TestDaemon.*SameBranch|TestSyncBranch.*SameBranch"
```

---

## Phase 3: Startup Validation

Add daemon startup warning when sync-branch is misconfigured.

### Tasks

- [ ] Add startup check in `daemon.go` `startDaemon()` function
- [ ] Log warning (not error) when sync-branch == current-branch at startup
- [ ] Write test: `TestDaemonStartupWarnsSameBranch`
- [ ] Verify daemon still starts (warn, don't block)

### Validation

```bash
go test -v ./cmd/bd -run TestDaemonStartup
```

---

## Phase 4: Closing

Create PR and clean up worktree.

### Tasks

- [ ] Run full daemon test suite
- [ ] Verify no regressions in sync tests
- [ ] Create PR targeting `main`
- [ ] Link PR to GH#1258
- [ ] Remove worktree after merge

### Validation

```bash
# Full test suite
go test -v ./cmd/bd -run "TestDaemon|TestSync"

# Create PR
gh pr create --draft --title "fix: add sync-branch guard to daemon code paths" --body "..."
```

---

## Test Matrix Reference

All 12 core scenarios from `design.md` mapped to test functions:

| # | Scenario | Expected | Test Function |
|---|----------|----------|---------------|
| 1 | Normal config (different branch) | Allow | `TestDaemonExportAllowsDifferentBranch` |
| 2 | Same branch (config) | Block | `TestDaemonExportSkipsSameBranch` |
| 3 | Same branch (env override) | Block | `TestDaemonExportSkipsEnvSameBranch` |
| 4 | No sync-branch configured | Allow | `TestDaemonExportAllowsNoSyncBranch` |
| 5 | Detached HEAD | Allow | `TestDaemonExportAllowsDetachedHead` |
| 6 | Non-git directory | Allow | `TestDaemonExportAllowsNonGit` |
| 7 | Local-only mode | Allow | `TestDaemonLocalExportAllows` |
| 8 | Worktree (different branch) | Allow | `TestDaemonExportWorktreeDifferentBranch` |
| 9 | Worktree (same as sync) | Block | `TestDaemonExportWorktreeSameBranch` |
| 10 | Dynamic switch TO sync-branch | Block | `TestDaemonExportDynamicBranchSwitch` |
| 11 | Dynamic switch FROM sync-branch | Allow | `TestDaemonExportAfterBranchChange` |
| 12 | Config hot reload | New value | `TestDaemonExportConfigReload` |

**Phase coverage**: Scenarios 1-7 covered in Phase 1-2. Scenarios 8-12 covered in Phase 2 (edge cases).
