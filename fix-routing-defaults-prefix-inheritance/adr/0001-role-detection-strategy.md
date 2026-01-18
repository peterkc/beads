# ADR-0001: Role Detection Strategy for Contributor Routing

> **Note**: This ADR is for GH#1174 but lives in the GH#1165 spec directory
> because both issues share the routing/detection domain.
>
> - GH#1165: Fix routing defaults (Phase 1)
> - GH#1174: Improve detection (Phase 2, blocked by #1165)

## Status

Proposed

## Context

GH#1174 needs to improve contributor/maintainer role detection for routing.
Current SSH/HTTPS heuristic has edge cases (fork + SSH = wrong detection).

### Problem Statement

| Scenario | Current Detection | Actual Role | Result |
|----------|-------------------|-------------|--------|
| Fork contributor with SSH | Maintainer (SSH URL) | Contributor | WRONG |
| Maintainer with HTTPS+token | Contributor (HTTPS URL) | Maintainer | WRONG |

### Constraints

1. Can't assume `gh` CLI is installed
2. Should work offline after initial setup
3. Must not add significant latency to every `bd create`
4. Should integrate with existing `RepoContext` pattern

## Decision Drivers

- **Accuracy**: Correctly detect role in all scenarios
- **Performance**: One-time cost, not per-command
- **Offline support**: Work without network after caching
- **Minimal dependencies**: Avoid requiring `gh` CLI

## Options Considered

### Option A: Check for `upstream` Remote (Git-only)

```go
// Check if upstream remote exists
upstreamURL, err := git("remote", "get-url", "upstream")
if err == nil && upstreamURL != "" {
    return Contributor, nil  // Has upstream = likely fork
}
// Fall back to SSH/HTTPS heuristic
```

**Pros:**
- Pure git, no external dependencies
- Works offline
- Zero latency

**Cons:**
- Relies on user setting up `upstream` remote (unreliable)
- Doesn't detect push permissions

**Verdict**: Improvement over current, but still heuristic-based.

### Option B: GitHub API via `gh` CLI

```bash
gh repo view --json fork,parent,viewerPermission
```

**Pros:**
- Authoritative (GitHub knows fork graph)
- Simple one-liner

**Cons:**
- Requires `gh` CLI installed
- Requires `gh` authenticated
- External process spawn

**Verdict**: Best accuracy, but dependency on `gh` is problematic.

### Option C: GitHub API via go-github Library

```go
import "github.com/google/go-github/v81/github"

client := github.NewClient(nil)  // or with auth
repo, _, err := client.Repositories.Get(ctx, owner, name)
if repo.GetFork() {
    // Check parent permissions
    parent := repo.GetParent()
    // ...
}
```

**Pros:**
- No external CLI dependency
- Native Go, testable
- Handles auth via http.Client

**Cons:**
- New dependency (~5MB)
- Requires GitHub token for private repos
- Rate limited (60/hr unauthenticated, 5000/hr authenticated)

**Sources:**
- [google/go-github](https://github.com/google/go-github)
- [go-github package docs](https://pkg.go.dev/github.com/google/go-github/v53/github)

### Option D: Hybrid - Git First, API Fallback with Caching

```go
func detectRole(repoPath string) (UserRole, error) {
    // 1. Check cached role (instant, works offline)
    if cached := gitConfig("beads.role"); cached != "" {
        return parseRole(cached), nil
    }

    // 2. Try git-only heuristics
    if hasUpstreamRemote() {
        // Strong fork signal - cache and return
        setGitConfig("beads.role", "contributor")
        return Contributor, nil
    }

    // 3. API check for ambiguous cases (one-time)
    if token := getGitHubToken(); token != "" {
        role := checkGitHubAPI(token)
        setGitConfig("beads.role", role)  // Cache forever
        return role, nil
    }

    // 4. Fall back to SSH/HTTPS heuristic
    return detectByURL()
}
```

**Pros:**
- Best of all worlds
- Works offline after first run
- Degrades gracefully (no token = heuristic)
- One-time API cost, cached forever

**Cons:**
- More complex implementation
- Still needs token for best accuracy

**Verdict**: RECOMMENDED

## Decision

**Option D: Hybrid approach with caching**

### Detection Priority

```
1. git config beads.role     → Cached result (instant)
2. upstream remote exists    → Fork signal (git-only)
3. GitHub API (if token)     → Authoritative check
4. SSH/HTTPS heuristic       → Fallback
```

### Caching Strategy

| Storage | Key | Value | Lifetime |
|---------|-----|-------|----------|
| git config (local) | `beads.role` | `maintainer` or `contributor` | Forever |

**Invalidation**: User runs `git config --unset beads.role`

### GitHub Token Discovery

**Important**: go-github and git CLI use **separate authentication systems**.

| System | Used For | Auth Method |
|--------|----------|-------------|
| Git CLI | push/pull | SSH keys, credential helpers |
| go-github | API calls | OAuth/PAT tokens |

**Discovery Order** (check each, use first found):

```go
func discoverGitHubToken() string {
    // 1. Explicit env var (CI/CD, user-set)
    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        return token
    }

    // 2. gh CLI (most common for developers)
    if output, err := exec.Command("gh", "auth", "token").Output(); err == nil {
        return strings.TrimSpace(string(output))
    }

    // 3. Git credential helper (may have stored PAT)
    if token := queryGitCredential("github.com"); token != "" {
        return token
    }

    // 4. No token available - skip API, use heuristics
    return ""
}

func queryGitCredential(host string) string {
    cmd := exec.Command("git", "credential", "fill")
    cmd.Stdin = strings.NewReader("protocol=https\nhost=" + host + "\n\n")
    output, err := cmd.Output()
    if err != nil {
        return ""
    }
    // Parse "password=<token>" from output
    for _, line := range strings.Split(string(output), "\n") {
        if strings.HasPrefix(line, "password=") {
            return strings.TrimPrefix(line, "password=")
        }
    }
    return ""
}
```

**Sources**:
- Git does NOT share auth with GitHub API
- Git credential helper can be queried programmatically
- `gh auth token` works if user has authenticated gh CLI

### API Fields Needed

```json
{
  "fork": true,
  "parent": {
    "full_name": "steveyegge/beads",
    "permissions": {
      "push": false
    }
  }
}
```

For non-forks:
```json
{
  "fork": false,
  "permissions": {
    "push": true
  }
}
```

## RepoContext Integration

Add role detection to `RepoContext` (single source of truth):

```go
type RepoContext struct {
    // Existing fields...
    BeadsDir     string
    RepoRoot     string
    IsWorktree   bool
    IsRedirected bool

    // NEW: Remote information
    OriginURL    string
    UpstreamURL  string   // Empty if not a fork
    HasUpstream  bool

    // NEW: Role detection
    UserRole     UserRole   // maintainer or contributor
    RoleSource   string     // "config", "upstream", "api", or "heuristic"
    IsFork       bool       // From API or upstream remote
}
```

### Benefits

- Role cached with other context (one lookup)
- Available everywhere `RepoContext` is used
- Consistent with existing patterns (`IsWorktree`, `IsRedirected`)

## Test Matrix

| Scenario | upstream | API | Token | Expected Detection |
|----------|----------|-----|-------|-------------------|
| Maintainer, SSH | none | fork=false, push=true | yes | Maintainer (api) |
| Contributor, HTTPS | none | fork=true, push=false | yes | Contributor (api) |
| Fork + SSH | yes | - | no | Contributor (upstream) |
| Fork + SSH | none | fork=true, push=false | yes | Contributor (api) |
| Offline, cached | - | - | - | (cached value) |
| Offline, no cache | none | - | no | Heuristic (SSH/HTTPS) |

## Dependencies

### Required

None - git-only heuristics work without dependencies.

### Optional (for API check)

- `github.com/google/go-github/v81/github` - GitHub API client
- Token for authenticated requests

### Not Required

- `gh` CLI - nice to have for token discovery, but not required

## Consequences

### Positive

- Correct detection for fork+SSH scenario
- One-time latency, then instant
- Works offline after caching
- Graceful degradation without token

### Negative

- New optional dependency (go-github)
- Complexity increase in detection logic
- Users may need to set token for best accuracy

### Neutral

- Existing `git config beads.role` override still works
- Current behavior preserved if no token available

## Implementation Notes

### Phase 1 (GH#1165 - Current PR)

- Change `routing.mode` default to `""`
- No detection changes

### Phase 2 (GH#1174 - Follow-up)

1. Add `upstream` remote check to `DetectUserRole()`
2. Add optional go-github dependency
3. Integrate with `RepoContext`
4. Add caching to git config

## Related

- GH#1165: Fix routing defaults (blocks this)
- GH#1174: Improve contributor detection (this ADR)
- `internal/routing/routing.go`: Current detection logic
- `internal/beads/context.go`: RepoContext implementation
