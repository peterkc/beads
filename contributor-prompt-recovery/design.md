# Design: Contributor Prompt Recovery

## Architecture

Two-gate system with no upfront detection:

```
bd init
   |
   +-- --contributor flag? --> Contributor wizard (no prompt)
   +-- --team flag?        --> Team wizard (no prompt)
   +-- (plain)             --> "Contributing to someone else's repo? [y/N]"
                                  |
                                  +-- [Y] --> Contributor wizard
                                  +-- [N] --> Proceed as maintainer

bd sync (if push fails)
   |
   +-- Parse error: "Permission denied" / "403" / "not allowed"
   +-- Show recovery guidance pointing to existing docs/commands

DetectUserRole()
   |
   +-- git config beads.role? --> use config
   +-- (no config)            --> ErrRoleNotConfigured
                                   (forces user through init prompt)
```

## Current Implementation (Before Changes)

The current init flow is **flag-based only**—no interactive prompt:

```
bd init (current)
   |
   +-- Lines 44-469: Setup Phase
   |     +-- Validate flags, create .beads/
   |     +-- Initialize database (SQLite/Dolt)
   |     +-- Set prefix, metadata, sync branch
   |     +-- Import existing issues
   |     +-- Handle stealth mode
   |
   +-- Lines 471-486: Wizard Execution (flag-gated)
   |     contributor, _ := cmd.Flags().GetBool("contributor")
   |     team, _ := cmd.Flags().GetBool("team")
   |
   |     if contributor { runContributorWizard(ctx, store) }
   |     if team { runTeamWizard(ctx, store) }
   |
   +-- Lines 492-582: Post-Init Tasks
         +-- Fork detection, hooks, diagnostics
```

### Current DetectUserRole() (routing.go:33-70)

```go
func DetectUserRole(repoPath string) (UserRole, error) {
    // 1. Check git config for explicit beads.role
    // 2. SSH (git@...) → Maintainer
    // 3. HTTPS with @ → Maintainer
    // 4. Plain HTTPS → Contributor
}
```

**Gap**: `DetectUserRole` exists but is **not called during init**. Wizard selection is purely flag-based.

### Insertion Point

**Lines 470-486 in init.go** — after database init, before closing store:

```go
// NEW: Add before existing wizard checks
if shouldPrompt() {
    role := promptContributorMode()  // Returns "contributor", "maintainer", or ""
    if role == "contributor" {
        contributor = true  // Triggers existing wizard
    }
}

// EXISTING: Flag-based checks remain for explicit overrides
if contributor {
    if err := runContributorWizard(ctx, store) { ... }
}
```

## Design Decisions

### Why Not Auto-Detection?

| Original Approach | Issue |
|-------------------|-------|
| 5-tier detection (config→cache→upstream→API→heuristic) | Over-engineered |
| Upstream remote check | Unreliable — many contributors don't set it up |
| GitHub API | Provider-specific, requires token |

### Why Prompt + Push-Fail?

| Benefit | Explanation |
|---------|-------------|
| User declares intent | Respects explicit configuration |
| Provider-agnostic | Error parsing works everywhere |
| No token required | Git-only operation |
| Handles human error | Recovery guidance catches mistakes |

## Key Components

### 1. Init Prompt (`cmd/bd/init.go`)

```go
func promptContributorMode() (bool, error) {
    // Check existing config first
    if role := getGitConfig("beads.role"); role != "" {
        fmt.Printf("Already configured as %s. Change? [y/N]: ", role)
        // ...
    }

    fmt.Print("Contributing to someone else's repo? [y/N]: ")
    // ...
}
```

### 2. Push Error Parser (`cmd/bd/sync_git.go`)

```go
func isPushPermissionDenied(output string) bool {
    patterns := []string{
        "permission denied",
        "403",
        "not allowed to push",
        "you are not allowed to push",
        "could not read from remote",
    }
    lower := strings.ToLower(output)
    for _, p := range patterns {
        if strings.Contains(lower, p) {
            return true
        }
    }
    return false
}
```

### 3. Recovery Flow (`cmd/bd/sync.go`)

```go
// In gitPush error handling
if isPushPermissionDenied(output) {
    fmt.Println("\n⚠ Push access denied.")
    fmt.Println("")
    fmt.Println("You don't have push access to this repository.")
    fmt.Println("")
    fmt.Println("If you're contributing to someone else's repo:")
    fmt.Println("  git config beads.role contributor")
    fmt.Println("  bd init --contributor")
    fmt.Println("")
    fmt.Println("See: docs/ROUTING.md for contributor setup")
    return err
}
```

No wizard, no --force, no new commands. Just helpful guidance pointing to existing docs.

## Reinit Behavior

```
bd init (beads.role exists?)
   |
   +-- No  --> Prompt: "Contributing to someone else's repo? [y/N]"
   |              |
   |              +-- [Y] --> set beads.role=contributor --> Contributor wizard
   |              +-- [N] --> set beads.role=maintainer  --> Continue
   |
   +-- Yes --> Show: "Already configured as {role}."
                  |
                  +-- "Change? [y/N]"
                        |
                        +-- [Y] --> Clear config --> Re-prompt
                        +-- [N] --> Keep config --> Continue
```

## Config Lifecycle

```
beads.role config
   |
   +-- SET via:
   |      +-- bd init prompt (user answers Y/N)
   |      +-- bd init --contributor flag
   |      +-- bd init --team flag
   |      +-- Manual: git config beads.role contributor
   |
   +-- READ via:
   |      +-- DetectUserRole() --> only source of truth
   |
   +-- CLEARED via:
   |      +-- Manual: git config --unset beads.role
   |
   +-- STALE detection:
          +-- .beads/ missing but config exists? --> Warn user
```

Local to repo (`.git/config`), not global.

## Files Modified

| File | Changes |
|------|---------|
| `cmd/bd/init.go` | Add contributor prompt before wizard selection |
| `cmd/bd/sync_git.go` | Add `isPushPermissionDenied()` function |
| `cmd/bd/sync.go` | Add recovery guidance message on push failure |
| `cmd/bd/doctor.go` | Add `checkBeadsRole()` check, fix points to `bd init` |
| `internal/beads/context.go` | Add `Role()`, `IsContributor()`, `IsMaintainer()`, `RequireRole()` |
| `internal/routing/routing.go` | Keep URL heuristic as fallback (graceful degradation) |

## RepoContext Integration

**Option**: Add role helpers to `RepoContext` (methods, not fields).

```go
type RepoContext struct {
    BeadsDir    string
    RepoRoot    string
    CWDRepoRoot string
    IsRedirected bool
    IsWorktree   bool
    // Role accessed via methods, not cached fields
}
```

**Helpers** (functions, not cached fields — avoids staleness):

```go
// Role reads beads.role from git config (fresh each call, ~1ms)
// If BEADS_DIR is set, returns Contributor implicitly (no config needed)
func (rc *RepoContext) Role() (UserRole, bool) {
    // BEADS_DIR implies contributor (external repo mode)
    if rc.IsRedirected {
        return Contributor, true
    }

    output, err := rc.GitOutput(context.Background(), "config", "--get", "beads.role")
    if err != nil {
        return "", false  // Not configured
    }
    return UserRole(strings.TrimSpace(output)), true
}

// IsContributor returns true if user is configured as contributor
func (rc *RepoContext) IsContributor() bool {
    role, ok := rc.Role()
    return ok && role == Contributor
}

// IsMaintainer returns true if user is configured as maintainer
func (rc *RepoContext) IsMaintainer() bool {
    role, ok := rc.Role()
    return ok && role == Maintainer
}

// RequireRole returns error if role not configured (forces init prompt)
func (rc *RepoContext) RequireRole() error {
    if _, ok := rc.Role(); !ok {
        return ErrRoleNotConfigured
    }
    return nil
}
```

**Why functions instead of cached fields**:
- Git config reads are fast (~1ms)
- Eliminates staleness concern entirely
- No sync.Once complexity for role
- Consistent: RepoContext paths are stable, role is config-based

**Trade-offs**:

| Pro | Con |
|-----|-----|
| Centralized access via `rc.Role()` | Minor overhead (~1ms per call) |
| Always fresh, no staleness | Role detection moves from routing.go |
| Simple API for callers | — |

**Recommendation**: Yes, add to RepoContext as functions. Fresh reads eliminate staleness with negligible overhead.

## Migration Strategy

Existing users don't have `beads.role` configured. Use existing infrastructure:

### bd doctor: Add Role Check

```go
func checkBeadsRole(path string) doctorCheck {
    role, err := getGitConfig(path, "beads.role")
    if err != nil || role == "" {
        return doctorCheck{
            Name:    "Role Configuration",
            Status:  statusWarning,
            Message: "beads.role not configured",
            Detail:  "Run 'bd init' to configure your role.",
            Fix:     "bd init",
        }
    }
    return doctorCheck{
        Name:    "Role Configuration",
        Status:  statusOK,
        Message: fmt.Sprintf("Configured as %s", role),
    }
}
```

### Migration Flow (uses existing bd init)

```
Existing user runs bd doctor
   |
   +-- "⚠ beads.role not configured"
   +-- "Fix: bd init"
   |
   v
bd init (detects existing .beads/)
   |
   +-- No beads.role? → Prompt for role → Set config
   +-- Has beads.role? → "Already configured. Change? [y/N]"
   |
   v
Future bd commands use explicit config
```

**No new commands**: `bd init` handles both new and existing users.

**No breaking change**: URL heuristic continues working until user runs `bd init`.

## DetectUserRole Changes

Current detection has a problematic fallback for SSH forks:

```go
// BEFORE
func DetectUserRole(repoPath string) (UserRole, error) {
    // 1. Check beads.role config ✅
    // 2. SSH URL → maintainer ❌ (wrong for forks!)
    // 3. HTTPS → contributor
}
```

New approach: Config first, heuristic fallback with warning (graceful degradation):

```go
// AFTER (graceful)
func DetectUserRole(repoPath string) (UserRole, error) {
    // 1. Check beads.role config - preferred source
    if role := getGitConfig("beads.role"); role != "" {
        return UserRole(role), nil
    }

    // 2. Fallback to URL heuristic (deprecated, with warning)
    //    This keeps existing users working while encouraging migration
    fmt.Fprintln(os.Stderr, "⚠ beads.role not configured. Run 'bd init' to set.")
    return detectFromURL(repoPath), nil
}
```

**Why graceful**:
- Existing users keep working (no breaking change)
- Warning encourages migration
- `bd doctor` shows the issue with fix command
- `bd init` provides easy migration path
