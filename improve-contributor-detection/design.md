# Design: Improve Contributor Detection

## Architecture Overview

```
Detection Priority (highest to lowest):
┌─────────────────────────────────────────┐
│ 1. git config beads.role (cached)       │ ← Instant
├─────────────────────────────────────────┤
│ 2. upstream remote exists (git-only)    │ ← Fast
├─────────────────────────────────────────┤
│ 3. GitHub API (if token available)      │ ← Network
├─────────────────────────────────────────┤
│ 4. SSH/HTTPS URL heuristic (fallback)   │ ← Current
└─────────────────────────────────────────┘
```

## Key Decisions

### Decision 1: Hybrid Approach
**Choice**: Option D from ADR - Git first, API fallback, cached

**Rationale**: Balances accuracy with performance. Git-only detection handles most fork cases. API provides authoritative check when ambiguous. Caching ensures one-time latency cost.

### Decision 2: go-github v81.0.0
**Choice**: Use go-github library for API calls

**Rationale**: Native Go, well-maintained, enables mocking via interfaces. Avoids `gh` CLI dependency which may not be installed.

### Decision 3: Token Discovery Order
**Choice**: GITHUB_TOKEN → gh CLI → git credential

**Rationale**: Explicit env var for CI/CD, gh CLI for developers, git credential as fallback. Matches common patterns.

### Decision 4: Forever Caching
**Choice**: Store role in `git config beads.role` forever

**Rationale**: Role rarely changes. Manual invalidation via `git config --unset` is simple. Avoids complexity of TTL-based caching.

## Component Design

### TokenDiscoverer Interface

```go
type TokenDiscoverer interface {
    FromEnv() string
    FromGhCLI() (string, error)
    FromGitCredential() (string, error)
}
```

Enables testing without real tokens.

### GitHubChecker Interface

```go
type GitHubChecker interface {
    CheckForkStatus(owner, repo string) (*ForkInfo, error)
}

type ForkInfo struct {
    IsFork bool
    Parent string  // "owner/repo" if fork
    CanPush bool   // Push access to parent
}
```

### RepoContext Extension

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

## Error Reporting Strategy

| Scenario | Visibility | Message |
|----------|------------|---------|
| Cached role used | `--verbose` | `Using cached role: contributor` |
| Upstream remote detected | `--verbose` | `Fork detected via upstream remote` |
| Token not found | `--verbose` | `No GitHub token, using URL heuristic` |
| API rate limited | **Warning** | `GitHub API rate limited, using heuristic (may be inaccurate)` |
| API network error | **Warning** | `GitHub API unreachable, using heuristic` |
| Invalid token | **Warning** | `GitHub token invalid, using heuristic` |

**Principle**: Silent success, visible failures that affect accuracy.

## Test Strategy

### Unit Tests (No Network)
- Mock TokenDiscoverer for all paths
- Mock GitHubChecker for API scenarios
- Test detection priority logic

### Integration Tests (httptest)
- Use httptest for GitHub API mocking
- Bare git repos for upstream remote detection
- No real tokens required

### Bare Repo Testing Pattern
```go
func setupForkScenario(t *testing.T) (forkDir string) {
    tmpDir := t.TempDir()

    // Create bare "upstream" repo
    upstreamDir := filepath.Join(tmpDir, "upstream.git")
    exec.Command("git", "init", "--bare", upstreamDir).Run()

    // Create bare "origin" repo (fork)
    originDir := filepath.Join(tmpDir, "origin.git")
    exec.Command("git", "init", "--bare", originDir).Run()

    // Clone origin, add upstream remote
    forkDir = filepath.Join(tmpDir, "fork")
    exec.Command("git", "clone", originDir, forkDir).Run()

    cmd := exec.Command("git", "remote", "add", "upstream", upstreamDir)
    cmd.Dir = forkDir
    cmd.Run()

    return forkDir
}
```

## Risks and Mitigations

| Risk | L | I | Mitigation |
|------|---|---|------------|
| go-github breaking changes | M | M | Pin to v81, test in CI |
| GitHub API rate limiting | M | L | Cache forever, warn, fallback |
| Token discovery fails silently | L | M | Verbose logging |
| Stale cached role | L | M | Document invalidation |
| Private repo API access | M | L | Detect 404, clear warning |

## Applied Patterns

- **Interface segregation**: TokenDiscoverer, GitHubChecker enable mocking
- **Graceful degradation**: Always has fallback path
- **Single source of truth**: RepoContext holds all context
- **Table-driven tests**: Go idiom for test matrix

## Related ADRs

- [ADR-0001: Role Detection Strategy](../fix-routing-defaults-prefix-inheritance/adr/0001-role-detection-strategy.md)
- [ADR-0001 Companion: Test Strategy](../fix-routing-defaults-prefix-inheritance/adr/0001-test-strategy.md)
- [ADR-0001 Companion: Implementation Notes](../fix-routing-defaults-prefix-inheritance/adr/0001-implementation-notes.md)
