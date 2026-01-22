# Tasks: Fix bd init BEADS_DIR

## Phase 1: Tracer Bullet

**Objective**: Add BEADS_DIR check to `checkExistingBeadsData()` with minimal test.

### Tasks

- [ ] **1.1** Read `cmd/bd/init.go` lines 811-870 (current implementation)
- [ ] **1.2** Add BEADS_DIR check at start of `checkExistingBeadsData()`
  - Check `os.Getenv("BEADS_DIR")`
  - Use `utils.CanonicalizePath()` for consistent path handling
  - If set and valid, check that path instead of CWD/worktree
- [ ] **1.3** Write test `TestCheckExistingBeadsData_WithBEADS_DIR` in `cmd/bd/init_test.go`
  - TC-001: BEADS_DIR set, target empty, local has .beads → succeeds
  - TC-002: BEADS_DIR set, target has data → error references BEADS_DIR
- [ ] **1.4** Run existing tests to verify no regression
  ```bash
  go test ./cmd/bd/... -run Init -v
  ```
- [ ] **1.5** Manual validation in beads-next
  ```bash
  cd /Volumes/atlas/beads-next
  eval "$(direnv export bash)"
  echo $BEADS_DIR  # Should be .beads-planning/.beads
  bd init --backend dolt --dry-run  # Should check BEADS_DIR target
  ```

### Validation

```bash
go test ./cmd/bd/... -run Init -v
go test ./cmd/bd/... -run TestCheckExistingBeadsData_WithBEADS_DIR -v
```

### Deliverables

| Artifact | Location |
|----------|----------|
| Code change | `cmd/bd/init.go` |
| Test | `cmd/bd/init_test.go` |

---

## Phase 2: Closing

**Objective**: Merge feature branch and cleanup.

### Tasks

- [ ] **2.1** Run `go test ./cmd/bd/... -v` and verify pass
- [ ] **2.2** Create PR with conventional commit message
- [ ] **2.3** Remove worktree after merge

### Validation

```bash
gh pr checks
git worktree list
```
