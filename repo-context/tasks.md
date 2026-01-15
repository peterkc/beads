# Tasks: Centralize CWD/BEADS_DIR Resolution

## Phase 0: Prerequisite — Daemon Unification

**Goal**: Unify daemon's worktree detection with RepoContext pattern BEFORE main migration

**Why Phase 0**: The daemon has duplicate worktree logic in `internal/daemon/discovery.go`.
If we migrate without unifying, we'll have two parallel implementations that can drift.

### Tasks

- [ ] Analyze `internal/daemon/discovery.go:findBeadsDirForWorkspace()`:
  - [ ] Map all os.Chdir() calls
  - [ ] Identify differences from `internal/beads.FindBeadsDir()`
  - [ ] Document edge cases daemon handles
- [ ] Add `GetRepoContextForWorkspace(workspacePath string)` to context.go:
  - [ ] Fresh resolution (no sync.Once caching)
  - [ ] Accepts explicit workspace path instead of using CWD
  - [ ] Returns same `RepoContext` struct
- [ ] Add `Validate()` method to RepoContext:
  - [ ] Verify BeadsDir still exists
  - [ ] Verify RepoRoot still exists
  - [ ] Return error if context is stale
- [ ] Refactor daemon to use new API:
  - [ ] Replace `findBeadsDirForWorkspace()` with `GetRepoContextForWorkspace()`
  - [ ] Add context validation before sync operations
- [ ] Add daemon-specific tests:
  - [ ] Test: Context resolution for different workspaces
  - [ ] Test: Stale context detection

**Validation**:
```bash
go test ./internal/beads/... -v -run TestRepoContextForWorkspace
go test ./internal/daemon/... -v
bd daemon status  # Verify daemon still works
```

**Size**: ~100 lines new code, ~100 lines tests

---

## Phase 1: Tracer Bullet — RepoContext API + Security + hasGitRemote()

**Goal**: Working GetRepoContext() with security mitigations + tests + one migrated function

### Security Tasks (CRIT-001, CRIT-003)

- [ ] Add security mitigations to `GitCmd()`:
  - [ ] Set `GIT_HOOKS_PATH=` to disable hooks (SEC-001)
  - [ ] Set `GIT_TEMPLATE_DIR=` to disable templates (SEC-002)
- [ ] Add `isPathInSafeBoundary(path string)` function:
  - [ ] Reject /etc, /usr, /var, /root, /System, /Library
  - [ ] Reject paths in other users' home directories
  - [ ] Call from `buildRepoContext()` before caching
- [ ] Add redirect boundary validation:
  - [ ] Ensure relative redirects stay within repo root
  - [ ] Add test for path traversal attempt (TS-SEC-003)

### Core Tasks

- [ ] Complete `internal/beads/context.go`:
  - [ ] `RepoContext` struct with BeadsDir, RepoRoot, CWDRepoRoot, IsRedirected, IsWorktree
  - [ ] `GetRepoContext()` with `sync.Once` caching
  - [ ] `GitCmd(ctx, args...)` method using `cmd.Dir` + security env vars
  - [ ] `GitCmdCWD(ctx, args...)` for CWD-relative commands
  - [ ] `RelPath(absPath)` for path relativization
  - [ ] `ResetCaches()` for test isolation
- [ ] Create `internal/beads/context_test.go`:
  - [ ] Test: Normal repo (CWD = repo root) — TS-001
  - [ ] Test: Worktree (CWD = worktree) — TS-002
  - [ ] Test: BEADS_DIR redirect (CWD ≠ beads repo) — TS-003
  - [ ] Test: Combined worktree + redirect — TS-004
  - [ ] Test: Subdirectory — TS-005
  - [ ] Test: Non-git with BEADS_DIR — TS-006
  - [ ] Test: Boundary conditions — TS-BC-001 to TS-BC-004
  - [ ] Test: Security scenarios — TS-SEC-001 to TS-SEC-003
  - [ ] Test: Test isolation — TS-ISO-001, TS-ISO-002
- [ ] Migrate `hasGitRemote()` in sync_git.go (line 289)

### Backward Compatibility Tasks

- [ ] Add deprecated wrapper for `syncbranch.GetRepoRoot()`:
  - [ ] Wrapper delegates to `beads.GetRepoContext().RepoRoot`
  - [ ] Add `// Deprecated: Use beads.GetRepoContext().RepoRoot` comment

**Validation**:
```bash
go test ./internal/beads/... -v -run TestRepoContext
go test ./internal/beads/... -v -run TestSecurity
go test ./cmd/bd/... -v -run TestHasGitRemote
```

**Size**: ~250 lines new code, ~400 lines tests

---

## Phase 2a: Query Functions (sync_git.go read-only)

**Goal**: All status/query git commands use RepoContext

### Tasks

- [ ] `isGitRepo()` (line 19)
- [ ] `gitHasUnmergedPaths()` (lines 25, 42)
- [ ] `gitHasUpstream()` (line 53)
- [ ] `gitBranchHasUpstream()` (lines 69-70)
- [ ] `gitHasChanges()` (line 80)
- [ ] `gitHasBeadsChanges()` — update redirect handling (lines 118, 131)
- [ ] `hasJSONLConflict()` (line 322)
- [ ] `gitHasUncommittedBeadsChanges()` — update redirect handling (lines 514, 523)
- [ ] `getDefaultBranchForRemote()` (lines 565, 577, 582)
- [ ] `checkMergeDriverConfig()` (line 448)

**Validation**:
```bash
go test ./cmd/bd/... -v -run "TestGit|TestSync"
cd /Volumes/atlas/acf && BEADS_DIR=/Volumes/atlas/acf/oss/.beads bd --no-daemon status
```

**Size**: ~10 functions, ~50 line changes

---

## Phase 2b: Mutation Functions (sync_git.go writes)

**Goal**: Pull, push, commit operations use correct repo context

### Tasks

- [ ] `gitPull()` — branch detection + pull command (lines 377, 390, 401)
- [ ] `gitPush()` — branch detection + push commands (lines 422, 429, 438)
- [ ] `runGitRebaseContinue()` (line 357)
- [ ] `restoreBeadsDirFromBranch()` (line 487)

**Validation**:
```bash
cd /Volumes/atlas/acf
BEADS_DIR=/Volumes/atlas/acf/oss/.beads bd --no-daemon sync
# Should succeed without "unstaged changes" error
```

**Size**: ~4 functions, ~30 line changes

---

## Phase 3: Other Sync Files

**Goal**: Remaining sync-related files use RepoContext

### Tasks

- [ ] `migrate_sync.go` (6 locations: lines 176, 183, 189, 301, 308, 311)
- [ ] `prime.go` (1 location: line 179)
- [ ] `merge.go` (2 locations: lines 128, 142)
- [ ] `hooks.go` (6 locations: lines 388, 541, 631, 645, 1057)

**Validation**:
```bash
go test ./cmd/bd/... -v
bd prime  # From various directories
```

**Size**: ~15 locations across 4 files

---

## Phase 4: Worktree Operations

**Goal**: Worktree commands work correctly with BEADS_DIR

### Tasks

- [ ] Review existing `cmd.Dir` usage in `worktree_cmd.go`
- [ ] Migrate to RepoContext where appropriate (9 locations)
- [ ] Ensure worktree create/remove use correct repo context

**Validation**:
```bash
cd /Volumes/atlas/Pommel  # BEADS_DIR routes to oss/
bd worktree add test-wt
bd worktree list
bd worktree remove test-wt
```

**Size**: ~9 locations, may be mostly correct already

---

## Phase 5: General CWD Cleanup

**Goal**: All remaining git commands use RepoContext

### Tasks

- [ ] `internal/compact/git.go:GetCurrentCommitHash()` (line 15)
- [ ] `cmd/bd/create.go` — os.Getwd usage (line 868)
- [ ] `cmd/bd/nodb.go` — os.Getwd usages (lines 30, 160)
- [ ] `cmd/bd/version.go` — git symbolic-ref (line 155)
- [ ] `cmd/bd/init_team.go` — git commands (lines 191, 203, 210, 218)
- [ ] `cmd/bd/gate_discover.go` — git rev-parse (lines 252, 262)
- [ ] `cmd/bd/init_git_hooks.go` — git config (lines 414, 446, 451)

**Validation**:
```bash
go test ./...
bd version  # From various directories
bd init team  # Verify git operations
```

**Size**: ~14 locations across 7 files

---

## Phase 6: Cleanup

**Goal**: Single source of truth, no scattered helpers

### Tasks

- [ ] Remove `getRepoRootForWorktree()` from sync_git.go
- [ ] Remove `GetRepoRoot()` from `internal/syncbranch/worktree.go`
- [ ] Update all callers to use `beads.GetRepoContext()`
- [ ] Run staticcheck for dead code
- [ ] Update PR #1102 description with expanded scope

**Validation**:
```bash
go build ./...  # No unused function warnings
go test ./...
staticcheck ./...
```

**Size**: ~50 lines removed

---

## Phase 7: Documentation

**Goal**: Document RepoContext API and update existing docs

### Tasks

- [ ] Check existing docs/ for files that need updating:
  ```bash
  ls /Volumes/atlas/beads/docs/
  ```
- [ ] Update any docs referencing CWD behavior or path resolution
- [ ] Consider new doc: `docs/repo-context.md` covering:
  - [ ] When to use `GitCmd()` vs `GitCmdCWD()`
  - [ ] BEADS_DIR redirect scenarios
  - [ ] Worktree context handling
  - [ ] Migration guide for contributors
- [ ] Add godoc comments to `internal/beads/context.go`

**Validation**:
```bash
# Check doc links
grep -r "RepoContext\|GitCmd" docs/
# Verify godoc renders
go doc ./internal/beads
```

**Size**: ~100-200 lines of documentation

---

## Post-Implementation

- [ ] Review open GitHub issues:
  ```bash
  gh issue list --repo steveyegge/beads --state open --search "CWD path worktree BEADS_DIR"
  ```
- [ ] Update PR #1102 to link resolved issues
- [ ] Restart bd daemons with new binary
- [ ] Verify oss/ beads routing now works
