# Tasks

## Phase 1: Tracer Bullet

**Goal**: End-to-end fix that eliminates the warning for the happy path.

### Tasks

1. **Add `isJSONLNewerThanDB()` to staleness.go**
   - File: `cmd/bd/staleness.go`
   - ~15 lines: stat both files, compare mtime
   - Use `os.Lstat` for JSONL (symlink-aware)
   - Return `false` if JSONL missing, `true` if DB missing

2. **Wire into read-only decision in main.go**
   - File: `cmd/bd/main.go` (around line 749)
   - After `useReadOnly := isReadOnlyCommand(cmd.Name())`
   - Add check: if stale, set `useReadOnly = false`
   - Add debug log for visibility

3. **Manual verification**
   ```bash
   # In worktree:
   bd init
   touch .beads/issues.jsonl  # Make JSONL newer
   bd --no-daemon ready 2>&1 | grep -i "warning"
   # Should produce no output (no warning)
   ```

### Validation

```bash
# Run existing staleness tests
go test -v ./cmd/bd/... -run TestStaleness
go test -v ./cmd/bd/... -run TestReadonly
```

---

## Phase 2: Edge Cases & Tests

**Goal**: Comprehensive test coverage and edge case handling.

### Tasks

1. **Add unit tests for `isJSONLNewerThanDB()`**
   - File: `cmd/bd/staleness_test.go`
   - Test cases:
     - [P] JSONL newer than DB → returns true
     - [P] DB newer than JSONL → returns false
     - [P] JSONL missing → returns false
     - [P] DB missing → returns true
     - [P] Both missing → returns false
     - Symlinked JSONL → uses symlink mtime

2. **Add integration test for GH#1089 scenario**
   - File: `cmd/bd/readonly_test.go` or new `cmd/bd/gh1089_test.go`
   - Create temp DB, make JSONL newer
   - Run read-only command via test harness
   - Assert stderr does not contain "attempt to write"

3. **Edge case: Symlinked JSONL**
   - Verify behavior matches `autoimport.CheckStaleness`
   - Add test similar to `internal/autoimport/symlink_test.go`

4. **Edge case: Worktree context**
   - Verify staleness check works in worktrees
   - DB path may differ from main repo

### Validation

```bash
# Run all tests
go test -v ./cmd/bd/...
go test -v ./internal/autoimport/...

# Verify no regressions
go test -race ./...
```

---

## Phase 3: Closing

**Goal**: Merge and cleanup.

### Tasks

1. **Create PR**
   ```bash
   gh pr create --draft \
     --title "fix(staleness): check JSONL freshness before read-only mode (GH#1089)" \
     --body "..."
   ```

2. **Ensure CI passes**
   - All tests green
   - golangci-lint passes

3. **Request review**
   - Mark PR ready for review
   - Link to GH#1089

4. **Cleanup worktree** (after merge)
   ```bash
   git worktree remove .worktrees/gh1089-readonly-staleness
   git branch -d feature/gh1089-readonly-staleness
   ```

### Validation

```bash
# CI must pass
gh pr checks

# After merge, verify on main
git checkout main && git pull
bd --no-daemon ready  # Should work without warnings
```
