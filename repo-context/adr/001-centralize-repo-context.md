# ADR-001: Centralize Repository Context Resolution

**Status**: Proposed

**Date**: 2025-01-14

## Context

The beads codebase has 50+ git commands across 12+ files that need to determine
which repository to operate on. Currently, this logic is scattered:

```
BEFORE (scattered):
┌─────────────────────────────────────────────────────────────┐
│  sync_git.go           syncbranch/          git/gitdir.go  │
│  ┌──────────────┐      ┌──────────────┐     ┌────────────┐ │
│  │getRepoRoot   │      │GetRepoRoot() │     │GetRepoRoot │ │
│  │ForWorktree() │      │  (duplicate!)│     │()          │ │
│  └──────────────┘      └──────────────┘     └────────────┘ │
│         │                    │                    │        │
│         └────────────────────┴────────────────────┘        │
│                    No single source of truth!              │
└─────────────────────────────────────────────────────────────┘
```

This causes bugs when `BEADS_DIR` points to a different repo than CWD:
- `bd sync` runs git commands in the wrong repository
- Worktree operations may target the wrong repo
- No consistent handling of redirects

## Decision Drivers

- **Correctness**: Git commands must run in the correct repository
- **Maintainability**: Single source of truth for repo resolution
- **Testability**: Easy to mock/reset for CWD-invariant tests
- **Backward compatibility**: Existing behavior unchanged for normal cases

## Considered Options

### Option 1: Centralized RepoContext Struct

Create `internal/beads/context.go` with a `RepoContext` struct that caches
all repo resolution logic:

```go
type RepoContext struct {
    BeadsDir    string // Actual .beads directory
    RepoRoot    string // Repository containing BeadsDir
    CWDRepoRoot string // Repository containing CWD
    IsRedirected bool
    IsWorktree   bool
}

func GetRepoContext() (*RepoContext, error)
func (rc *RepoContext) GitCmd(ctx, args...) *exec.Cmd
```

- **Good, because** single source of truth
- **Good, because** caching via `sync.Once` avoids repeated filesystem access
- **Good, because** methods provide ergonomic API
- **Good, because** `ResetCaches()` enables test isolation
- **Bad, because** requires migrating 50+ callsites

### Option 2: Pass repoRoot Parameter

Add `repoRoot` parameter to all git-calling functions:

```go
func gitPull(ctx context.Context, repoRoot string) error
func gitPush(ctx context.Context, repoRoot string) error
```

- **Good, because** explicit, no hidden state
- **Bad, because** requires changing 50+ function signatures
- **Bad, because** callers must compute repoRoot (duplicated logic)
- **Bad, because** no caching benefit

### Option 3: Global Variable

Set a global `var CurrentRepoRoot string` at startup:

- **Good, because** simple to implement
- **Bad, because** not testable (global state)
- **Bad, because** no compile-time safety
- **Bad, because** can't distinguish beads repo vs CWD repo

### Option 4: Status Quo

Keep scattered helpers, fix each bug individually:

- **Good, because** no refactoring required
- **Bad, because** bugs will recur (same root cause)
- **Bad, because** maintenance burden increases

## Decision

**Chosen option**: Option 1 (Centralized RepoContext Struct), because it provides
a single source of truth with caching, testability, and an ergonomic API.

The struct approach allows distinguishing between `RepoRoot` (where beads lives)
and `CWDRepoRoot` (where user is working), which is essential for BEADS_DIR
redirect scenarios.

## Consequences

### Positive

- All git commands run in correct repository
- Single place to fix repo resolution bugs
- Testable with `ResetCaches()`
- Clear separation: `GitCmd()` for beads repo, `GitCmdCWD()` for user's repo

### Negative

- Migration effort: 50+ callsites across 12 files
- Learning curve: contributors must use new API
- Potential circular import issues (mitigated by placement in `internal/beads`)

### Neutral

- Existing helpers (`GetRedirectInfo`, `FindBeadsDir`) remain unchanged
- `RepoContext` composes them internally

## Related

- Spec: `specs/beads-repo-context/CLAUDE.md`
- Issue: `acf-1b7u`
- PR: #1102 (extends original path resolution fixes)
