# ADR-0001 Companion: Implementation Notes

> Companion document for [ADR-0001: Role Detection Strategy](0001-role-detection-strategy.md)

## GitHub Token Discovery

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

## API Fields Needed

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

## Go Module Updates

When adding go-github, update both module files:

```bash
# Add dependency
go get github.com/google/go-github/v81@latest

# Clean up unused deps, update checksums
go mod tidy

# Verify build
go build ./...

# Commit both files together
git add go.mod go.sum
```

| File | Changes |
|------|---------|
| `go.mod` | New direct dependency entry |
| `go.sum` | Checksums for go-github + transitive deps |

**Note**: Some Google packages already exist as indirect deps (`btree`, `go-cmp`, `uuid`),
so transitive dependency overlap may reduce net additions.

## Dependencies

### Required

None - git-only heuristics work without dependencies.

### Optional (for API check)

- `github.com/google/go-github/v81/github` - GitHub API client
- Token for authenticated requests

### Not Required

- `gh` CLI - nice to have for token discovery, but not required

## Phase Breakdown

### Phase 1 (GH#1165 - Current PR)

- Change `routing.mode` default to `""`
- No detection changes

### Phase 2 (GH#1174 - Follow-up)

1. Add `upstream` remote check to `DetectUserRole()`
2. Add optional go-github dependency
3. Integrate with `RepoContext`
4. Add caching to git config
5. Update documentation (see below)

## Documentation Updates

| File | Section | Changes |
|------|---------|---------|
| `docs/ROUTING.md` | User Role Detection (lines 9-26) | Update strategy to 4-tier hybrid approach |
| `docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md` | Role detection (lines 100-125, 330-339) | Add upstream remote + API detection |
| `docs/TROUBLESHOOTING.md` | Manual override (line 647) | Add API fallback troubleshooting |

**Key changes**:
- Document `upstream` remote as fork signal
- Document GitHub API integration (optional)
- Document token discovery order (`GITHUB_TOKEN` → `gh auth token` → git credential)
- Add troubleshooting for API rate limits and token issues

## Implementation Skills

| Skill | Purpose |
|-------|---------|
| `spec` | Create/execute phased spec with EARS requirements, tracer bullet pattern |
| `golang` | Go idioms, table-driven tests, golangci-lint compliance |
| `github` | PR creation, upstream contribution patterns |
| `beads` | Track phases as issues, manage dependencies |
| `commit` | Conventional commit messages |

## Conversation Context

*Captured from spec review session (2026-01-18)*

### Key Decisions

1. **go-github version**: v81.0.0 (confirmed latest as of Jan 2025)
2. **Testing both paths**: Tests must verify both `gh auth token` (for discovery) and go-github (for API calls)
3. **No token required for CI**: Unit tests use mocks; integration tests use `httptest`
4. **Bare repos for fork simulation**: Test `upstream` remote detection with real git, not mocks
5. **Error reporting**: Warnings for accuracy-affecting failures, verbose for debug info

### Blocking Dependency

PR #1177 must merge before spec execution begins. Note this in spec header.
