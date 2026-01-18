# Tasks: GH#1166 Sync-Branch Validation

## Phase 1: Tracer Bullet — Config-Time Validation

End-to-end path: YAML config set → validation → error

### Tasks

- [ ] Add `syncbranch` import to `internal/config/yaml_config.go`
- [ ] Add sync-branch case to `validateYamlConfigValue()`:
  ```go
  case "sync-branch", "sync.branch":
      return syncbranch.ValidateSyncBranchName(value)
  ```
- [ ] Add unit test in `yaml_config_test.go`:
  - `TestValidateYamlConfigValue_SyncBranch_RejectsMain`
  - `TestValidateYamlConfigValue_SyncBranch_AcceptsValid`

### Validation

```bash
# Unit test for new validation
go test ./internal/config/... -run TestValidateYamlConfigValue -v

# Verify import doesn't create cycle
go build ./...

# Manual verification
cd /tmp/test-repo && bd init
bd config set sync.branch main
# Expected: error message about main not allowed
```

### Success Criteria

- [ ] `bd config set sync.branch main` returns error
- [ ] `bd config set sync.branch beads-sync` succeeds
- [ ] No import cycles introduced

---

## Phase 2: Runtime Guard — Sync Command Protection

Defense in depth: catch cases where config was set before validation existed or manually edited.

### Tasks

- [ ] In `cmd/bd/sync.go`, after line 247 (after `hasSyncBranchConfig` is set):
  ```go
  if hasSyncBranchConfig {
      if syncbranch.IsSyncBranchSameAsCurrent(ctx, syncBranchName) {
          FatalError("Cannot sync to '%s': it's your current branch. "+
              "Checkout a different branch first, or use a dedicated sync branch like 'beads-sync'.",
              syncBranchName)
      }
  }
  ```
- [ ] Add integration test in `sync_test.go` or `sync_modes_test.go`:
  - `TestSync_FailsWhenOnSyncBranch`

### Validation

```bash
# Run sync-related tests
go test ./cmd/bd/... -run TestSync -v

# Run full test suite to catch regressions
go test ./... -v

# Manual verification (requires manual config.yaml edit to bypass Phase 1)
cd /tmp/test-repo
# Manually edit .beads/config.yaml: sync: { branch: main }
git checkout main
bd sync
# Expected: error about being on sync branch
```

### Success Criteria

- [ ] `bd sync` fails gracefully when on sync-branch
- [ ] Error message is clear and actionable
- [ ] Sync tests succeed: `go test ./cmd/bd/... -run TestSync -v`

---

## Phase 3: Closing — PR and Cleanup

### Tasks

- [ ] Run full test suite: `go test ./...`
- [ ] Create upstream PR with:
  - Title: `fix: Validate sync-branch at config-time and runtime (GH#1166)`
  - Reference upstream issue
  - Include test reproduction steps
- [ ] Comment on GH#1166 with PR link
- [ ] Cleanup worktree after PR merged

### Validation

```bash
# Full test suite
go test ./... -v

# Verify no regressions in specific areas
go test ./cmd/bd/... -run TestSync -v
go test ./internal/config/... -v
go test ./internal/syncbranch/... -v

# Lint check
golangci-lint run ./...
```

### PR Description Template

```markdown
## Summary

Fixes #1166 - `bd sync` commits files outside `.beads/` when sync.branch equals current branch.

**Root cause**: Two validation gaps:
1. `validateYamlConfigValue()` didn't validate sync-branch keys
2. No runtime check in `bd sync` for sync-branch == current-branch

**Fix**:
- Add sync-branch validation to YAML config path (reuses `ValidateSyncBranchName()`)
- Add runtime guard before worktree operations (reuses `IsSyncBranchSameAsCurrent()`)

## Test Plan

- [x] Unit test: YAML config validation rejects main/master
- [x] Integration test: `bd sync` fails when on sync-branch
- [x] Manual verification: `bd config set sync.branch main` → error
- [x] Full test suite succeeds: `go test ./...`
```

### Success Criteria

- [ ] PR created and linked to GH#1166
- [ ] CI passes (`go test ./...` and `golangci-lint run`)
- [ ] Worktree cleaned up after merge
