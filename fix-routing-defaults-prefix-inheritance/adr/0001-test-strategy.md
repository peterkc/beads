# ADR-0001 Companion: Test Strategy

> Companion document for [ADR-0001: Role Detection Strategy](0001-role-detection-strategy.md)

## Test Matrix

| Scenario | upstream | API | Token | Expected Detection |
|----------|----------|-----|-------|-------------------|
| Maintainer, SSH | none | fork=false, push=true | yes | Maintainer (api) |
| Contributor, HTTPS | none | fork=true, push=false | yes | Contributor (api) |
| Fork + SSH | yes | - | no | Contributor (upstream) |
| Fork + SSH | none | fork=true, push=false | yes | Contributor (api) |
| Offline, cached | - | - | - | (cached value) |
| Offline, no cache | none | - | no | Heuristic (SSH/HTTPS) |

## Error Reporting

**Principle**: Silent success, visible failures that affect accuracy.

| Scenario | Visibility | Message |
|----------|------------|---------|
| Cached role used | `--verbose` | `Using cached role: contributor` |
| Upstream remote detected | `--verbose` | `Fork detected via upstream remote` |
| Token not found | `--verbose` | `No GitHub token, using URL heuristic` |
| API rate limited | **Warning** | `GitHub API rate limited, using heuristic (may be inaccurate)` |
| API network error | **Warning** | `GitHub API unreachable, using heuristic` |
| Invalid token | **Warning** | `GitHub token invalid, using heuristic` |
| gh CLI failed | `--verbose` | `gh auth token failed, trying other sources` |

Users should know when detection falls back to a potentially inaccurate heuristic.

## Test Levels

| Level | Build Tag | Token Required? | What's Tested |
|-------|-----------|-----------------|---------------|
| Unit | (none) | No | Logic, parsing, fallback paths (mocked) |
| Integration | `integration` | No | Component integration, `httptest` mocks |
| E2E (optional) | `integration` + `live` | Yes | Real GitHub API calls |

## Testability Design

Interfaces enable mocking without real tokens:

```go
// Token discovery - mock command execution
type TokenDiscoverer interface {
    FromEnv() string
    FromGhCLI() (string, error)
    FromGitCredential() (string, error)
}

// GitHub API - mock the client
type GitHubChecker interface {
    CheckForkStatus(owner, repo string) (*ForkInfo, error)
}
```

## Token Discovery Testing

Both `gh` CLI and go-github paths must be tested:

| Test Scenario | Token Source | API Client | Purpose |
|---------------|--------------|------------|---------|
| gh CLI available | `gh auth token` | go-github | Common developer setup |
| GITHUB_TOKEN env | Environment var | go-github | CI/CD scenario |
| Git credential helper | `git credential fill` | go-github | Alternative auth |
| No token available | — | — | Graceful degradation |

## Bare Repo Testing

Use bare repos to test `upstream` remote detection without mocking git:

```go
func setupForkScenario(t *testing.T) (forkDir string, cleanup func()) {
    tmpDir := t.TempDir()

    // 1. Create bare "upstream" repo (simulates steveyegge/beads)
    upstreamDir := filepath.Join(tmpDir, "upstream.git")
    exec.Command("git", "init", "--bare", upstreamDir).Run()

    // 2. Create bare "origin" repo (simulates peterkc/beads fork)
    originDir := filepath.Join(tmpDir, "origin.git")
    exec.Command("git", "init", "--bare", originDir).Run()

    // 3. Clone origin to create working directory
    forkDir = filepath.Join(tmpDir, "fork")
    exec.Command("git", "clone", originDir, forkDir).Run()

    // 4. Add upstream remote (the fork signal!)
    cmd := exec.Command("git", "remote", "add", "upstream", upstreamDir)
    cmd.Dir = forkDir
    cmd.Run()

    return forkDir, func() { /* cleanup */ }
}
```

**Why bare repos**: Tests real git behavior instead of mocking `git remote get-url`.
Beads already uses this pattern in `sync_git_test.go`, `sync_local_only_test.go`.

## Existing Infrastructure

Extend existing test file: `cmd/bd/contributor_routing_e2e_test.go`

- Already has `//go:build integration` tag
- Has `contributorRoutingEnv` test fixture helper
- Has `TestExplicitRoleOverride` as foundation
- Uses table-driven tests pattern

## New Unit Test Files

```
internal/routing/
├── detection_test.go      # Role detection logic
├── token_discovery_test.go # Token discovery paths
└── github_client_test.go   # go-github integration (httptest)
```
