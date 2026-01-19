# Tasks: bd orphans --db flag fix (Option D: Storage Interface)

## Phase 1: Tracer Bullet + All Callers

Define interface, update signature, and update ALL callers so build never breaks.

### Tasks

- [ ] **1.1** Search for existing storage interfaces
  ```bash
  grep -r "type.*interface" internal/storage/
  grep -r "GetIssue\|SearchIssues" internal/storage/*.go
  ```
  - Document: Existing patterns to follow

- [ ] **1.2** Find all callers of `FindOrphanedIssues`
  ```bash
  grep -r "FindOrphanedIssues" cmd/bd/
  ```
  - Known callers:
    - `cmd/bd/doctor/git.go:940` (CheckOrphanedIssues)
    - `cmd/bd/orphans.go:15` (variable assignment)
    - `cmd/bd/orphans.go:108` (actual call)

- [ ] **1.3** Create `IssueProvider` interface
  - Location: `internal/types/orphans.go` (or extend existing interface)
  - Methods:
    - `GetOpenIssues(ctx context.Context) ([]*Issue, error)`
    - `GetIssuePrefix() string`

- [ ] **1.4** Implement interface on SQLite storage
  - Check: Does `storage.Storage` already have these methods?
  - If yes: Use existing methods
  - If no: Add methods to `internal/storage/sqlite/storage.go`

- [ ] **1.5** Update `FindOrphanedIssues()` signature AND all callers atomically
  - File: `cmd/bd/doctor/git.go`
  - Change: `func FindOrphanedIssues(path string)` → `func FindOrphanedIssues(gitPath string, provider types.IssueProvider)`
  - Remove: Internal DB opening code
  - Add: Use `provider.GetIssuePrefix()` and `provider.GetOpenIssues()`
  - **ALSO UPDATE** (same commit):
    - `CheckOrphanedIssues()` in `doctor/git.go` - build provider from path
    - `doctorFindOrphanedIssues` variable in `orphans.go` - update signature

- [ ] **1.6** Update orphans.go wrapper
  - Add: `getIssueProvider()` function
  - Update: `findOrphanedIssues()` to pass provider
  - Update: `orphansCmd.Run` to call with provider

- [ ] **1.7** Verify build succeeds
  ```bash
  go build ./cmd/bd/...
  ```

- [ ] **1.8** Manual verification (sandbox test)
  ```bash
  # Create test repos
  mkdir -p /tmp/beads-test/{planning,code}
  cd /tmp/beads-test/planning && bd init --prefix=TEST
  bd create -t "Test issue"

  # Create code repo
  cd /tmp/beads-test/code && git init
  echo "test" > file.txt && git add . && git commit -m "feat: impl (TEST-xxx)"

  # Cross-repo orphan detection - THIS IS THE KEY TEST
  cd /tmp/beads-test/code
  bd --db /tmp/beads-test/planning/.beads/var/beads.db orphans
  # Expected: Should find TEST-xxx as orphan

  # Verify --db flag was honored (not using local .beads)
  bd orphans 2>&1 | grep -q "no beads database" && echo "PASS: Local correctly fails"
  ```

### Validation

```bash
# Build succeeds (critical - must not break between phases)
go build ./cmd/bd/...

# Existing tests still pass
go test ./cmd/bd/... -run TestOrphans -v

# Manual sandbox test with --db flag works
# (command above returns TEST-xxx orphan)
```

## Phase 2: Test Coverage

Add comprehensive tests with mock provider.

### Tasks

- [ ] **2.1** Create mock provider for testing
  - File: `cmd/bd/doctor/git_test.go` or `cmd/bd/orphans_test.go`
  - Implementation:
    ```go
    type mockProvider struct {
        issues []*types.Issue
        prefix string
    }
    ```

- [ ] **2.2** Add cross-repo test (IT-02)
  - Test: `TestFindOrphanedIssues_CrossRepo`
  - Uses: Mock provider with custom prefix
  - Verifies: Issues with custom prefix detected
  - **Must assert**: `--db` flag is honored, not local `.beads/`

- [ ] **2.3** Add backward compatibility test (RT-01)
  - Test: `TestFindOrphanedIssues_LocalProvider`
  - Uses: Real SQLite provider with local .beads/
  - Verifies: Default behavior unchanged

- [ ] **2.4** Add error handling tests
  - Test: Provider returns error → function handles gracefully
  - Test: Empty provider → returns empty list

- [ ] **2.5** Update existing test mocks
  - File: `cmd/bd/orphans_test.go`
  - Update: `doctorFindOrphanedIssues` mock signature from `func(string)` to `func(string, IssueProvider)`

- [ ] **2.6** [P] Add integration test with real cross-repo setup
  - Uses: t.TempDir() with two directories
  - Creates: Real DB in one, git repo in other

### Validation

```bash
# All tests pass
go test ./cmd/bd/... -run TestOrphans -v
go test ./cmd/bd/doctor/... -run TestFindOrphaned -v

# Coverage report
go test ./cmd/bd/... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -E "orphan|FindOrphaned"

# Specific assertion: cross-repo test exists and passes
go test ./cmd/bd/... -run TestFindOrphanedIssues_CrossRepo -v
```

## Phase 3: Closing

PR creation and upstream submission.

### Tasks

- [ ] **3.1** Final test run
  ```bash
  go test ./... -v
  go vet ./...
  ```

- [ ] **3.2** Update any affected documentation
  - Check: `docs/` for orphan-related docs
  - Update: If needed

- [ ] **3.3** Create PR
  - Title: `fix(orphans): use storage interface for cross-repo detection`
  - Body:
    - Links to GH#1196
    - Summary of interface approach
    - Test coverage improvements
    - Breaking change notes (internal API only)
  - Labels: `bug`, `orphans`, `enhancement`

- [ ] **3.4** Address review feedback
  - Iterate based on maintainer comments

- [ ] **3.5** Cleanup worktree after merge
  ```bash
  git worktree remove .worktrees/orphans-db-flag
  ```

### Validation

```bash
# PR created
gh pr view --json url

# CI passes
gh pr checks

# PR merged
gh pr view --json state
```
