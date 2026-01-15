---
name: beads-repo-context
status: in_progress
created: 2025-01-14
spec_type: implementation
phases:
  - name: "Phase 0: Daemon Unification"
    type: prerequisite
    status: pending
    description: "Unify internal/daemon/discovery.go with RepoContext pattern"
  - name: "Phase 1: Tracer Bullet"
    type: tracer
    status: pending
    description: "RepoContext API + security mitigations + tests + hasGitRemote() migration"
  - name: "Phase 2a: Query Functions"
    type: mvs
    status: pending
    description: "Status checks, branch detection (read-only git commands)"
  - name: "Phase 2b: Mutation Functions"
    type: mvs
    status: pending
    description: "gitPull, gitPush, gitCommit (write operations)"
  - name: "Phase 3: Other Sync Files"
    type: mvs
    status: pending
    description: "migrate_sync.go, prime.go, merge.go, hooks.go"
  - name: "Phase 4: Worktree Operations"
    type: mvs
    status: pending
    description: "worktree_cmd.go git calls"
  - name: "Phase 5: General CWD"
    type: mvs
    status: pending
    description: "Remaining files (compact, create, nodb, version, etc.)"
  - name: "Phase 6: Cleanup"
    type: mvs
    status: pending
    description: "Remove duplicate helpers, verify no dead code"
  - name: "Phase 7: Documentation"
    type: mvs
    status: pending
    description: "Update docs/ and add RepoContext usage guide"
success_criteria:
  - "SC-001: bd sync works with BEADS_DIR pointing to different repo"
  - "SC-002: All git commands use RepoContext API"
  - "SC-003: No duplicate repo root resolution helpers"
  - "SC-004: CWD-invariant tests pass for all scenarios"
  - "SC-005: Git hooks disabled in GitCmd() to prevent code execution"
  - "SC-006: BEADS_DIR validated against safe path boundaries"
  - "SC-007: Daemon uses workspace-specific context (no stale cache)"
  - "SC-008: Deprecated wrapper for syncbranch.GetRepoRoot() for 1 release"
location:
  remote: github.com/peterkc/beads
  branch: specs
  path: repo-context
beads:
  epic: null
  worktree_path: null
  worktree_branch: null
target:
  repo: /Volumes/atlas/beads
  worktree: /Volumes/atlas/beads/.worktrees/oss-lbp
  pr: 1102
---

# Centralize CWD/BEADS_DIR Resolution in Beads

## Problem

50+ git commands across 12+ files assume CWD is repo root. When `BEADS_DIR` points to a different repo, these commands fail.

```bash
cd /Volumes/atlas/acf  # has unstaged changes
BEADS_DIR=/Volumes/atlas/acf/oss/.beads bd sync
# FAILS: git operations run on ACF repo, not oss/ repo
```

## Solution

Create centralized `RepoContext` API in `internal/beads/context.go`:

```go
type RepoContext struct {
    BeadsDir    string // Actual .beads directory (after redirects)
    RepoRoot    string // Repository root containing BeadsDir
    CWDRepoRoot string // Repository root containing CWD
    IsRedirected bool
    IsWorktree   bool
}

func GetRepoContext() (*RepoContext, error)
func (rc *RepoContext) GitCmd(ctx, args...) *exec.Cmd
```

## Scope

| Tier | Files | Locations | Impact |
|------|-------|-----------|--------|
| 1 | sync_git.go, migrate_sync.go, prime.go, merge.go, hooks.go | 37 | Blocks bd sync |
| 2 | worktree_cmd.go | 9 | Worktree operations |
| 3 | 7 files (compact, create, nodb, version, init_team, gate_discover, init_git_hooks) | 18 | General CWD |

## Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| **Backward compat breakage** | High | Medium | Comprehensive test matrix before any migration; test existing workflows |
| **Daemon behavior change** | High | Medium | Daemons may have different context than CLI; add daemon-specific tests |
| **Circular imports** | Medium | Low | New `internal/beads/context.go` must not import from `cmd/bd`; verify with `go build` |
| **Test isolation failures** | Medium | Medium | `ResetCaches()` must clear all state; verify in parallel test runs |
| **Git hook execution** | Critical | Medium | Malicious repo could execute hooks; disable via `GIT_HOOKS_PATH=` |
| **Path traversal** | Critical | Low | BEADS_DIR could point to sensitive dirs; validate boundaries |
| **TOCTOU attacks** | High | Low | Path validated at init, used later; add runtime validation |
| **Daemon cache staleness** | High | High | Long-running daemon caches stale context; use workspace-specific API |

### Mitigation Strategy

**Primary**: Write comprehensive CWD-invariant test matrix BEFORE any migration (Phase 1 includes this).

**Secondary**: Phase-by-phase execution with manual verification at each gate.

**Security**: Implement git hook disabling and path boundary validation in Phase 1.

**Daemon**: Add `GetRepoContextForWorkspace()` variant in Phase 0 before main migration.

### External Dependencies

- No blocking dependencies
- Post-implementation: Check open GH issues that may be resolved by this fix

## Security Considerations

| Threat | Vector | Mitigation |
|--------|--------|------------|
| **Git hook execution** | Malicious `.git/hooks/` in BEADS_DIR target | Set `GIT_HOOKS_PATH=` and `GIT_TEMPLATE_DIR=` in `GitCmd()` |
| **Path traversal** | BEADS_DIR points to `/etc`, `/root`, etc. | Validate resolved path against safe boundaries |
| **Redirect injection** | `.beads/redirect` escapes repo boundary | Ensure relative redirects stay within repository |
| **TOCTOU** | Path swapped between validation and use | Re-validate or use file descriptors |
| **Environment injection** | Modified PATH provides fake `git` | Consider absolute path to git binary |

## Daemon Handling

The daemon is a **long-running process** where `sync.Once` caching is inappropriate.

### Problem

```go
// CLI: sync.Once is fine (process exits quickly)
rc, _ := beads.GetRepoContext()  // Cached at startup

// Daemon: sync.Once causes stale context
// - User creates new worktree → daemon doesn't see it
// - BEADS_DIR changes via direnv → daemon uses old value
// - Project moves → daemon points to stale paths
```

### Solution: Workspace-Specific API

```go
// For daemon: fresh resolution per-operation
func GetRepoContextForWorkspace(workspacePath string) (*RepoContext, error)

// Validation hook for cached contexts
func (rc *RepoContext) Validate() error
```

### Unification Requirement (Phase 0)

The daemon has **duplicate worktree detection** in `internal/daemon/discovery.go`:

```go
func findBeadsDirForWorkspace(workspacePath string) string {
    // Uses os.Chdir pattern - must be unified with RepoContext
}
```

Phase 0 consolidates this into the RepoContext API before main migration.

## Related Files

- [requirements.md](requirements.md) — EARS format requirements
- [design.md](design.md) — Architecture decisions
- [tasks.md](tasks.md) — Phase breakdown

## Related Issues

- Fixes #1101 (gitPull CWD bug)
- Fixes #1098 (worktree redirect)
- Extends PR #1102 (original path resolution fixes)
- May resolve other open issues (check post-implementation)
