# Tasks

## Phase 1: Delete Dead Code

**Type**: Tracer | **Estimate**: 5 min

### Tasks

- [ ] Delete `restoreBeadsDirFromBranch` function (lines 515-540 in sync_git.go)
- [ ] Verify build succeeds: `go build ./cmd/bd/...`
- [ ] Run existing sync tests: `go test -v ./cmd/bd/... -run Sync`

### Validation

```bash
# Verify function is gone
grep -n "restoreBeadsDirFromBranch" cmd/bd/sync_git.go && exit 1 || echo "Function removed"

# Build check
go build ./cmd/bd/...

# Test check
go test -v ./cmd/bd/... -run TestSync -count=1
```

### Success Criteria

- [ ] SC-001: Function no longer exists in codebase
- [ ] SC-002: Build succeeds without errors

---

## Phase 2: Regression Test

**Type**: MVS | **Estimate**: 15 min

### Tasks

- [ ] Add `TestConfigPreservedDuringSync` to `cmd/bd/sync_test.go`
- [ ] Follow bare-repo fixture pattern from existing E2E tests
- [ ] Test that uncommitted config.yaml survives `bd sync`

### Validation

```bash
# Run the new test
go test -v ./cmd/bd/... -run TestConfigPreservedDuringSync -count=1

# Run all sync tests to check for regressions
go test -v ./cmd/bd/... -run TestSync -count=1
```

### Success Criteria

- [ ] SC-003: New regression test passes
- [ ] SC-004: All existing sync tests pass

---

## Completion Checklist

- [ ] All phases complete
- [ ] PR created with conventional commit format
- [ ] GH#1100 can be closed after user verification
