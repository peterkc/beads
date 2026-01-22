# Tasks: Fix bd init BEADS_DIR

## Phase 1: Tracer Bullet

**Objective**: Make `bd init` respect BEADS_DIR for existence check, database creation, and contributor wizard default.

### Tasks

- [ ] **1.1** Read current implementation
  - `cmd/bd/init.go` lines 72-160 (safety check + path determination)
  - `cmd/bd/init.go` lines 811-870 (`checkExistingBeadsData`)
  - `cmd/bd/init_contributor.go` lines 18-50 (wizard BEADS_DIR handling)

- [ ] **1.2** Fix `checkExistingBeadsData()` (Bug 1)
  - Add BEADS_DIR check at function start
  - Use `utils.CanonicalizePath()` for path handling
  - Return early if BEADS_DIR set (skip CWD/worktree checks)

- [ ] **1.3** Fix `initDBPath` determination (Bug 1b)
  - Add BEADS_DIR check before default `.beads/` path
  - Handle both SQLite and Dolt backends
  - Preserve existing precedence: `--db` > `BEADS_DB` > `BEADS_DIR` > default

- [ ] **1.4** Fix contributor wizard default (Bug 2)
  - In `runContributorWizard()`, use BEADS_DIR as default when set
  - Fall back to `~/.beads-planning` when BEADS_DIR not set

- [ ] **1.5** Write tests for BEADS_DIR scenarios
  - `TestCheckExistingBeadsData_WithBEADS_DIR`
  - `TestInitDBPath_WithBEADS_DIR`
  - Cover test cases TC-001 through TC-008 from requirements.md

- [ ] **1.6** Run existing tests to verify no regression
  ```bash
  go test ./cmd/bd/... -v
  go test ./internal/routing/... -v
  ```

- [ ] **1.7** Manual validation in beads-next
  ```bash
  cd /Volumes/atlas/beads-next
  trash .beads
  eval "$(direnv export bash)"
  echo "BEADS_DIR=$BEADS_DIR"
  bd init --backend dolt --prefix bdx
  # Verify: Database at BEADS_DIR, not CWD
  ```

### Validation

```bash
go test ./cmd/bd/... -run Init -v
go test ./cmd/bd/... -run BEADS_DIR -v
go test ./internal/routing/... -v
```

### Deliverables

| Artifact | Location |
|----------|----------|
| Bug 1 fix | `cmd/bd/init.go` - `checkExistingBeadsData()` |
| Bug 1b fix | `cmd/bd/init.go` - `initDBPath` determination |
| Bug 2 fix | `cmd/bd/init_contributor.go` - wizard default |
| Tests | `cmd/bd/init_test.go` |

---

## Phase 2: Closing

**Objective**: Merge feature branch and cleanup.

### Tasks

- [ ] **2.1** Run `go test ./cmd/bd/... -v` and verify all pass
- [ ] **2.2** Run `go test ./internal/routing/... -v` and verify unchanged
- [ ] **2.3** Create PR with conventional commit message
- [ ] **2.4** Remove worktree after merge

### Validation

```bash
gh pr checks
git worktree list
```
