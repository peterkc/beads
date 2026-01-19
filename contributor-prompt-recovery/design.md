# Design: Contributor Prompt Recovery

## Architecture

Two-gate system with no upfront detection:

```
bd init
├─ --contributor flag? → Contributor wizard (no prompt)
├─ --team flag?        → Team wizard (no prompt)
└─ (plain)             → "Contributing to someone else's repo? [y/N]"
                          ├─ [Y] → Contributor wizard
                          └─ [N] → Proceed as maintainer

bd sync (if push fails)
├─ Parse error: "Permission denied" / "403" / "not allowed"
├─ "⚠ Push access denied. Set up contributor mode? [Y/n]"
└─ [Y] → Run contributor wizard + migrate existing issues
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
| Handles human error | Recovery wizard catches mistakes |

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

## Config Storage

```bash
git config beads.role contributor  # or maintainer
git config --unset beads.role      # clear
```

Local to repo (`.git/config`), not global.

## Files Modified

| File | Changes |
|------|---------|
| `cmd/bd/init.go` | Add contributor prompt before wizard selection |
| `cmd/bd/sync_git.go` | Add `isPushPermissionDenied()` function |
| `cmd/bd/sync.go` | Add recovery guidance message on push failure |
| `internal/routing/routing.go` | Remove URL heuristic fallback, require explicit config |

## DetectUserRole Changes

Current detection has a problematic fallback:

```go
// BEFORE (problematic)
func DetectUserRole(repoPath string) (UserRole, error) {
    // 1. Check beads.role config ✅
    // 2. SSH URL → maintainer ❌ (wrong for forks!)
    // 3. HTTPS → contributor
}
```

New approach requires explicit configuration:

```go
// AFTER (explicit)
func DetectUserRole(repoPath string) (UserRole, error) {
    // Check beads.role config - ONLY source of truth
    output, err := gitCommandRunner(repoPath, "config", "--get", "beads.role")
    if err == nil {
        role := strings.TrimSpace(string(output))
        if role == string(Maintainer) {
            return Maintainer, nil
        }
        if role == string(Contributor) {
            return Contributor, nil
        }
    }

    // No config = not configured (forces user through init prompt)
    return "", ErrRoleNotConfigured
}
```

This forces users through the init prompt, which sets `beads.role`.
