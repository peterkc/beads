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

See `design.md` for comprehensive test scenarios. Key scenarios to cover:

| Scenario | Expected |
|----------|----------|
| sync-branch == current-branch | Block + log |
| sync-branch != current-branch | Allow |
| No sync-branch configured | Allow |
| Detached HEAD | Allow (fail-open) |
| Local-only mode | Allow (no sync-branch) |
| BEADS_SYNC_BRANCH env override | Apply guard |
