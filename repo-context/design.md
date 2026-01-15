# Design: Centralize CWD/BEADS_DIR Resolution

## ADRs

- [ADR-001: Centralize Repository Context Resolution](adr/001-centralize-repo-context.md)
- [ADR-002: Use cmd.Dir Pattern for Git Commands](adr/002-git-command-pattern.md)

## Before/After Architecture

### BEFORE: Scattered Resolution (Broken)

```
User runs: cd /acf && BEADS_DIR=oss/.beads bd sync

                           sync_git.go
  ─────────────────────────────────────────────────────────────────────

   hasGitRemote()         gitPull()             gitPush()

   exec.Command(          exec.Command(         exec.Command(
     "git",                 "git",                "git",
     "remote"               "pull"...             "push"...
   )                      )                     )

   CWD: /acf  ❌          CWD: /acf  ❌         CWD: /acf  ❌

  ─────────────────────────────────────────────────────────────────────
                                 │
                                 ▼

ERROR: Git commands run in /acf (CWD), not /acf/oss (BEADS_DIR's repo)
```

### AFTER: Centralized RepoContext (Fixed)

```
User runs: cd /acf && BEADS_DIR=oss/.beads bd sync

               RepoContext (internal/beads/context.go)
  ─────────────────────────────────────────────────────────────────────

  GetRepoContext() → *RepoContext (cached via sync.Once)

  RepoContext {
    BeadsDir:     "/acf/oss/.beads"     ← from BEADS_DIR
    RepoRoot:     "/acf/oss"            ← ✅ correct repo
    CWDRepoRoot:  "/acf"                ← user's CWD repo
    IsRedirected: true
  }

  rc.GitCmd(ctx, args...)  → cmd.Dir = RepoRoot  ✅

  ─────────────────────────────────────────────────────────────────────
                                 │
                                 ▼

                    sync_git.go (refactored)
  ─────────────────────────────────────────────────────────────────────

   hasGitRemote()         gitPull()             gitPush()

   rc.GitCmd(ctx,         rc.GitCmd(ctx,        rc.GitCmd(ctx,
     "remote")              "pull"...)            "push"...)

   Dir: /oss  ✅          Dir: /oss  ✅         Dir: /oss  ✅

  ─────────────────────────────────────────────────────────────────────
                                 │
                                 ▼

SUCCESS: Git commands run in correct repo (/acf/oss)
```

### Resolution Flow

```
                       GetRepoContext()
                             │
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                  ▼
    FindBeadsDir()    GetRedirectInfo()   GetMainRepoRoot()
          │                  │                  │
          └──────────────────┴──────────────────┘
                             │
                             ▼
                    Cached RepoContext
                      (sync.Once)
                             │
          ┌──────────────────┴──────────────────┐
          ▼                                     ▼
    rc.GitCmd()                          rc.GitCmdCWD()
    → beads repo                         → user's repo
```

## Architecture Overview

```
                      Call Sites (50+)
   sync_git.go, worktree_cmd.go, hooks.go, merge.go, etc.
                            │
                            │ rc.GitCmd(ctx, args...)
                            ▼

  ─────────────────────────────────────────────────────────────
                      RepoContext API
              internal/beads/context.go

   GetRepoContext() → *RepoContext (cached)
   rc.GitCmd(ctx, args...) → *exec.Cmd
   rc.GitCmdCWD(ctx, args...) → *exec.Cmd
   rc.RelPath(absPath) → (string, error)
  ─────────────────────────────────────────────────────────────
                            │
                            │ uses
                            ▼

        Existing Helpers (internal/beads, internal/git)
   ─────────────────────────────────────────────────────────
    GetRedirectInfo()              GetMainRepoRoot()
    internal/beads                 internal/git
```

## Key Decisions

### KD-001: Helper Location
**Decision**: `internal/beads/context.go`
**Rationale**: Near existing `FindBeadsDir()`, `GetRedirectInfo()`. Avoids circular imports.
**Alternatives**: `internal/git` (rejected: can't import internal/beads), `cmd/bd` (rejected: not reusable)

### KD-002: Git Command Pattern
**Decision**: Use `cmd.Dir` not `-C` flag
**Rationale**:
- More Go-idiomatic
- Works with all git commands (some don't support `-C`)
- Composable with other exec.Cmd properties (Env, Stdin)
**Alternatives**: `-C` flag (rejected: not universal), wrapper script (rejected: overhead)

### KD-003: Caching Strategy
**Decision**: `sync.Once` per process
**Rationale**: CWD and BEADS_DIR don't change during command execution
**Alternatives**: No cache (rejected: repeated filesystem access), TTL cache (rejected: unnecessary complexity)

### KD-004: API Shape
**Decision**: Struct with methods
**Rationale**: Single source of truth, ergonomic, extensible
**Alternatives**: Multiple functions (rejected: scattered state), global variables (rejected: testing)

### KD-005: Redirect File Path Format
**Decision**: Prefer absolute paths in `.beads/redirect` files
**Rationale**:
- The redirect file is gitignored (local-only), so cross-clone portability doesn't apply
- Absolute paths are immediately clear without mental resolution
- Example: `/Volumes/atlas/acf/oss/.beads` vs `oss/.beads`
**Alternatives**: Relative paths (supported but less clear)
**Note**: Both formats work—`FollowRedirect()` handles either. This is guidance, not enforcement.

## RepoContext Implementation

```go
// internal/beads/context.go

package beads

import (
    "context"
    "os/exec"
    "path/filepath"
    "sync"

    "github.com/steveyegge/beads/internal/git"
)

type RepoContext struct {
    BeadsDir    string
    RepoRoot    string
    CWDRepoRoot string
    IsRedirected bool
    IsWorktree   bool
}

var (
    repoCtx     *RepoContext
    repoCtxOnce sync.Once
    repoCtxErr  error
)

func GetRepoContext() (*RepoContext, error) {
    repoCtxOnce.Do(func() {
        repoCtx, repoCtxErr = buildRepoContext()
    })
    return repoCtx, repoCtxErr
}

func buildRepoContext() (*RepoContext, error) {
    beadsDir := FindBeadsDir()
    if beadsDir == "" {
        return nil, fmt.Errorf("no .beads directory found")
    }

    // Security: Validate path boundary (SEC-003)
    if !isPathInSafeBoundary(beadsDir) {
        return nil, fmt.Errorf("BEADS_DIR points to unsafe location: %s", beadsDir)
    }

    redirectInfo := GetRedirectInfo()

    var repoRoot string
    if redirectInfo.IsRedirected {
        repoRoot = filepath.Dir(beadsDir)
    } else {
        var err error
        repoRoot, err = git.GetMainRepoRoot()
        if err != nil {
            return nil, fmt.Errorf("cannot determine repository root: %w", err)
        }
    }

    cwdRepoRoot, _ := git.GetRepoRoot() // May differ from repoRoot

    isWorktree := git.IsWorktree()

    return &RepoContext{
        BeadsDir:     beadsDir,
        RepoRoot:     repoRoot,
        CWDRepoRoot:  cwdRepoRoot,
        IsRedirected: redirectInfo.IsRedirected,
        IsWorktree:   isWorktree,
    }, nil
}

func (rc *RepoContext) GitCmd(ctx context.Context, args ...string) *exec.Cmd {
    cmd := exec.CommandContext(ctx, "git", args...)
    cmd.Dir = rc.RepoRoot
    // Security: Disable git hooks and templates to prevent code execution
    // in potentially malicious repositories (SEC-001, SEC-002)
    cmd.Env = append(os.Environ(),
        "GIT_HOOKS_PATH=",      // Disable hooks
        "GIT_TEMPLATE_DIR=",    // Disable templates
    )
    return cmd
}

func (rc *RepoContext) GitCmdCWD(ctx context.Context, args ...string) *exec.Cmd {
    cmd := exec.CommandContext(ctx, "git", args...)
    if rc.CWDRepoRoot != "" {
        cmd.Dir = rc.CWDRepoRoot
    }
    return cmd
}

func (rc *RepoContext) RelPath(absPath string) (string, error) {
    return filepath.Rel(rc.RepoRoot, absPath)
}

func ResetCaches() {
    repoCtxOnce = sync.Once{}
    repoCtx = nil
    repoCtxErr = nil
}

// Security: Validate path is not in sensitive system directories (SEC-003)
var unsafePrefixes = []string{
    "/etc", "/usr", "/var", "/root", "/System", "/Library",
    "/bin", "/sbin", "/opt", "/private",
}

func isPathInSafeBoundary(path string) bool {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return false
    }
    for _, prefix := range unsafePrefixes {
        if strings.HasPrefix(absPath, prefix+"/") || absPath == prefix {
            return false
        }
    }
    // Also reject other users' home directories
    homeDir, _ := os.UserHomeDir()
    if strings.HasPrefix(absPath, "/Users/") || strings.HasPrefix(absPath, "/home/") {
        if homeDir != "" && !strings.HasPrefix(absPath, homeDir) {
            return false
        }
    }
    return true
}

// Daemon API: Fresh resolution per workspace (no sync.Once caching)
func GetRepoContextForWorkspace(workspacePath string) (*RepoContext, error) {
    // Change to workspace directory temporarily
    originalDir, err := os.Getwd()
    if err != nil {
        return nil, err
    }
    defer os.Chdir(originalDir)

    if err := os.Chdir(workspacePath); err != nil {
        return nil, fmt.Errorf("cannot access workspace %s: %w", workspacePath, err)
    }

    // Clear caches for fresh resolution
    git.ResetCaches()

    // Build context (same logic, no caching)
    return buildRepoContext()
}

// Validate checks if the cached context is still valid
func (rc *RepoContext) Validate() error {
    if _, err := os.Stat(rc.BeadsDir); os.IsNotExist(err) {
        return fmt.Errorf("BeadsDir no longer exists: %s", rc.BeadsDir)
    }
    if _, err := os.Stat(rc.RepoRoot); os.IsNotExist(err) {
        return fmt.Errorf("RepoRoot no longer exists: %s", rc.RepoRoot)
    }
    return nil
}
```

## Migration Pattern

**Before:**
```go
func gitPull(ctx context.Context) error {
    branchCmd := exec.CommandContext(ctx, "git", "symbolic-ref", "--short", "HEAD")
    // ... runs in CWD, not beads repo
}
```

**After:**
```go
func gitPull(ctx context.Context) error {
    rc, err := beads.GetRepoContext()
    if err != nil {
        return err
    }
    branchCmd := rc.GitCmd(ctx, "symbolic-ref", "--short", "HEAD")
    // ... runs in correct repo
}
```

## Files to Modify

### New Files
- `internal/beads/context.go` — RepoContext implementation
- `internal/beads/context_test.go` — CWD-invariant tests

### Modified Files (by priority)
1. `cmd/bd/sync_git.go` — 22 locations
2. `cmd/bd/migrate_sync.go` — 6 locations
3. `cmd/bd/hooks.go` — 6 locations
4. `cmd/bd/worktree_cmd.go` — 9 locations
5. `cmd/bd/init_team.go` — 4 locations
6. `cmd/bd/init_git_hooks.go` — 3 locations
7. `cmd/bd/gate_discover.go` — 2 locations
8. `cmd/bd/merge.go` — 2 locations
9. `cmd/bd/nodb.go` — 2 locations
10. `cmd/bd/prime.go` — 1 location
11. `cmd/bd/version.go` — 1 location
12. `cmd/bd/create.go` — 1 location
13. `internal/compact/git.go` — 1 location

### Removed Code (Phase 5)
- `cmd/bd/sync_git.go:getRepoRootForWorktree()` — replaced by RepoContext
- `internal/syncbranch/worktree.go:GetRepoRoot()` — duplicate
