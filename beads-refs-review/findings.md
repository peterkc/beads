# Consolidated Findings

## PR #2473 Feedback — Delta Status

| #   | Original Issue                         | Status  | Evidence                                                                     |
| --- | -------------------------------------- | ------- | ---------------------------------------------------------------------------- |
| F1  | `isRebaseInProgress()` worktree bug    | FIXED   | Rewritten to use `git.GetGitDir()` at `hooks.go:1041-1057`                   |
| F2  | `config.yaml` not created by `bd init` | FIXED   | `createConfigYaml()` added at `init_templates.go`, called from `init.go:544` |
| F3  | Test coverage gaps                     | PARTIAL | Substantial new tests added, but see B1 below                                |

## Bugs

### Bug-1 — `TestIsRebaseInProgress` silently passes without validating (HIGH)

**Files**: `cmd/bd/hooks_checkout_test.go:46-80`
**Found by**: Delta review, Build agent, CR CLI (3 independent sources)

The test creates bare `.git/` directories instead of real git repos. `isRebaseInProgress()`
calls `git.GetGitDir()` which runs `git rev-parse --git-dir`. In a non-git directory, this
fails and caches the error via `sync.Once`. Subsequent subtests reuse the cached error,
so `isRebaseInProgress()` always returns `false` — the "rebase-merge exists" and
"rebase-apply exists" subtests pass vacuously.

**Fix**: Either `git init` the temp dirs, or import `"github.com/steveyegge/beads/internal/git"`
and call `git.ResetCaches()` between subtests. See `init_test.go:698-702` for the correct pattern.

### Bug-2 — `TestWriteBeadsRefsDisabled` never calls `writeBeadsRefs` (MEDIUM)

**File**: `cmd/bd/git_reset_test.go:406-428`
**Found by**: CR CLI

Asserts no HEAD file in an empty temp dir but never invokes the function under test.
Passes vacuously. Also, `t.Parallel()` with `config.Initialize()` risks races with
other tests modifying global config state.

### Bug-3 — CHANGELOG example contradicts description (LOW)

**File**: `CHANGELOG.md:14-15`
**Found by**: CR CLI

Example shows `reset_dolt_with_git false` but text describes auto-resetting.
Users copying the example configure the opposite behavior.

## Security

### Security-1 — Path traversal via branch names (HIGH)

**Files**: `cmd/bd/dolt_head.go:70, 103`
**Found by**: Peer review

`writeBeadsRefs` constructs file paths from Dolt branch names without sanitization:

```go
refPath := filepath.Join(refsDir, branch)
```

A branch named `../../etc/something` writes outside `.beads/refs/heads/`.
`readBeadsRefs` has the same issue reading crafted `.beads/HEAD` content.

**Fix**: Validate branch names against `../`, null bytes, and control characters before
constructing paths. Consider using `filepath.Rel` to verify the result stays within
the refs directory.

### Security-2 — Committed backup data with PII (MEDIUM)

**File**: `.beads/backup/events.jsonl`
**Found by**: CR CLI

241 lines of runtime backup data containing owner emails and local filesystem paths.
Should be in `.gitignore` and removed from history.

## Concurrency

### Concurrency-1 — `git add` silently fails under concurrent agents (HIGH)

**File**: `cmd/bd/dolt_head.go:78-80`
**Found by**: colgrep, Peer review

```go
_ = cmd.Run()  // git add .beads/HEAD .beads/refs/heads/<branch>
```

Multiple agents running `bd create` simultaneously contend on git's `index.lock`.
Losers silently fail — refs exist on disk but are never staged. The "git history
carries the ref" guarantee is broken.

**Options**: Retry with backoff, use `git add` only in PersistentPostRun (not inline),
or document that multi-agent setups should commit refs separately.

### Concurrency-2 — `lastWrittenRefs` global has no synchronization (LOW)

**File**: `cmd/bd/dolt_head.go:20`
**Found by**: colgrep, Peer review

Package-level `var lastWrittenRefs string` — safe today (single goroutine path),
but a latent data race if `writeBeadsRefs` is ever called from the background flush
goroutine. Worth a comment or `sync.Mutex`.

### Concurrency-3 — Non-atomic two-file write (LOW)

**File**: `cmd/bd/dolt_head.go:57-73`
**Found by**: Peer review

`writeBeadsRefs` writes `.beads/HEAD` then `.beads/refs/heads/<branch>` sequentially.
A reader between writes sees stale state. Window is small; git uses lock files for this.

## UX

### UX-1 — Reset prompt doesn't explain data loss (MEDIUM)

**File**: `cmd/bd/dolt_head.go:218-226`
**Found by**: Peer review

Prompt says "Reset dolt HEAD to X?" but `DOLT_RESET('--hard')` discards all issue
changes after that commit. Users may think this is a soft pointer move.

### UX-2 — `bd reset` without branch_strategy gives no feedback (LOW)

**File**: `cmd/bd/git_reset.go:66`
**Found by**: Peer review

Git reset runs, but Dolt sync silently skips because `IsBranchStrategyEnabled()` is
false. User expected Dolt sync but got none.

### UX-3 — `gitResetLine` parameter is unused (LOW)

**File**: `cmd/bd/dolt_head.go:134`
**Found by**: Peer review

`checkBeadsRefSyncWithGitLine` accepts `gitResetLine` for prompt context but the
prompt code never references it. Dead parameter.

## Test Gaps

### TestGap-1 — No worktree scenario for `isRebaseInProgress` (MEDIUM)

**File**: `cmd/bd/hooks_checkout_test.go`
**Found by**: Delta review

The fix uses `GetGitDir()` for worktree-aware paths, but no test exercises the worktree
code path (where `.git` is a gitfile pointing elsewhere).

### TestGap-2 — `bd init` test doesn't assert config.yaml creation (LOW)

**File**: `cmd/bd/init_test.go:138-185`
**Found by**: Delta review

Fix is correct but no regression guard against future breakage.

### TestGap-3 — 10 core tests require Docker (MEDIUM)

**Found by**: Build agent

Round-trip, sync correctness, and E2E tests all skip without Docker. The core
behavioral guarantees are untested in CI environments without Dolt containers.

### TestGap-4 — `cmd/bd/vc.go` — 102 lines changed, no dedicated tests (LOW)

**Found by**: Build agent

Version control refactoring has no test file. Behavior tested only indirectly.

## Design Notes (non-actionable)

| #   | Note                                                                     | Source      |
| --- | ------------------------------------------------------------------------ | ----------- |
| D1  | Post-checkout hook is intentionally a stub (Phase 2 integration point)   | colgrep     |
| D2  | Config stored in YAML, not Dolt — survives `DOLT_RESET --hard`           | colgrep     |
| D3  | Downgrade path is clean — ref files become inert in older versions       | Peer review |
| D4  | `bd dolt pull` correctly updates refs via unconditional PostRun call     | Peer review |
| D5  | Ref files travel via `git push`, not `bd dolt push` — correct separation | Peer review |
| D6  | Hand-rolled YAML writer (280+ lines) preserves comments but is complex   | colgrep     |

## Config Key Validation Gap

**File**: `internal/config/config.go:748-752`
**Found by**: Peer review

`IsBranchStrategyEnabled` returns `true` if ANY key under `branch_strategy` exists,
including typos (`branch_strategy.prompts` instead of `branch_strategy.prompt`).
Misspelled keys silently enable ref writing with no prompt/reset behavior.

## Severity Summary

| Severity | Count | IDs                                                                          |
| -------- | ----- | ---------------------------------------------------------------------------- |
| HIGH     | 3     | Bug-1, Security-1, Concurrency-1                                             |
| MEDIUM   | 5     | Bug-2, Security-2, UX-1, TestGap-1, TestGap-3                               |
| LOW      | 7     | Bug-3, Concurrency-2, Concurrency-3, UX-2, UX-3, TestGap-2, TestGap-4       |
