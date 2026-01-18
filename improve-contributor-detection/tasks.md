# Tasks: Improve Contributor Detection

> **BLOCKED**: This spec depends on PR #1177 merging first.

## Phase 1: Upstream Remote Detection

**Type**: Tracer Bullet
**Goal**: Add git-only upstream remote detection with documentation

### Tasks

1. Add `HasUpstreamRemote()` function to `internal/routing/routing.go`
   - Check for `upstream` remote via `git remote get-url upstream`
   - Return true if remote exists, false otherwise

2. Update `DetectUserRole()` to check upstream remote
   - After git config check, before SSH/HTTPS heuristic
   - If upstream exists, return contributor role
   - Log "Fork detected via upstream remote" at verbose level

3. Add unit tests in `internal/routing/detection_test.go`
   - Test upstream detection with bare repo fixtures
   - Test detection priority (config > upstream > heuristic)

4. [P] Add integration test in `cmd/bd/contributor_routing_e2e_test.go`
   - Use bare repo pattern for fork simulation
   - Verify contributor detection via upstream remote

5. Update `docs/ROUTING.md` with upstream detection
   - Add "Upstream Remote Detection" section
   - Document that `upstream` remote signals fork/contributor

### Validation

```bash
go test ./internal/routing/... -v
# Expected: PASS, including TestUpstreamRemoteDetection

go test ./cmd/bd/... -tags=integration -run=TestUpstreamRemoteDetection -v
# Expected: "Fork detected via upstream remote" in verbose output

golangci-lint run ./internal/routing/...
# Expected: Exit 0, no new warnings

lychee --offline docs/ROUTING.md
# Expected: All internal links valid
```

**Success criteria**: All commands exit 0, no test failures.

---

## Phase 2: GitHub API Integration

**Type**: MVS Slice
**Goal**: Add go-github dependency, token discovery, and API documentation

### Tasks

1. Add go-github dependency
   ```bash
   go get github.com/google/go-github/v81@latest
   go mod tidy
   ```

2. Create `internal/routing/github_client.go`
   - Define `GitHubChecker` interface
   - Implement `RealGitHubChecker` using go-github
   - Query repo fork status and permissions

3. Create `internal/routing/token_discovery.go`
   - Define `TokenDiscoverer` interface
   - Implement token discovery chain (env → gh CLI → git credential)
   - Return empty string if no token found

4. Add unit tests for token discovery paths
   - Mock command execution
   - Test all 3 sources

5. [P] Add httptest integration tests for GitHub API
   - Mock API responses for fork/non-fork scenarios
   - Test error handling (rate limit, network error)

6. Update `docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md`
   - Add GitHub API detection section
   - Document token discovery order
   - Add inline code comments for `discoverGitHubToken()`

### Validation

```bash
go build ./...
# Expected: Exit 0, no build errors

go test ./internal/routing/... -v
# Expected: PASS for TestTokenDiscovery*, TestGitHubChecker*

golangci-lint run ./internal/routing/...
# Expected: Exit 0, no new warnings

lychee --offline docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md
# Expected: All internal links valid
```

**Success criteria**: Build passes, all token discovery paths tested.

---

## Phase 3: RepoContext Integration

**Type**: MVS Slice
**Goal**: Integrate with RepoContext, add caching, and troubleshooting docs

### Tasks

1. Extend `RepoContext` struct in `internal/beads/context.go`
   - Add UserRole, RoleSource, IsFork fields
   - Add OriginURL, UpstreamURL, HasUpstream fields

2. Update `NewRepoContext()` to populate role fields
   - Call detection logic during construction
   - Populate all new fields

3. Add git config caching to `DetectUserRole()`
   - After API detection, cache result: `git config beads.role <role>`
   - Check cache first before detection

4. Update consumers to use RepoContext role fields
   - `cmd/bd/create.go` - use RepoContext.UserRole
   - `internal/routing/routing.go` - remove duplicate detection

5. Add tests for caching behavior
   - Verify cache write after API call
   - Verify cache read on subsequent calls

6. Update `docs/TROUBLESHOOTING.md`
   - Add "Role incorrectly detected" section
   - Add API rate limit troubleshooting
   - Add token configuration instructions
   - Add cache invalidation instructions

7. Update `docs/ROUTING.md` with complete 4-tier strategy
   - Replace SSH/HTTPS heuristic section with full detection strategy
   - Add cache invalidation instructions

### Validation

```bash
go test ./internal/beads/... -v
# Expected: PASS for TestRepoContext*, including new role fields

go test ./cmd/bd/... -tags=integration -run=TestRoleCaching -v
# Expected: Cache hit on second call (verify via verbose logging)

golangci-lint run ./...
# Expected: Exit 0, no new warnings

lychee --offline docs/TROUBLESHOOTING.md docs/ROUTING.md
# Expected: All internal links valid
```

**Success criteria**: Caching works (2nd call uses cached value), RepoContext populated correctly.

---

## Phase 4: Closing

**Type**: Closing
**Merge Strategy**: PR

### Tasks

1. Run full test suite and linting
   ```bash
   go test ./... -v
   go test ./... -tags=integration -v
   golangci-lint run ./...
   ```

2. Final documentation review
   ```bash
   lychee --offline docs/*.md
   ```

3. Create PR against upstream
   - Title: `feat(routing): improve contributor detection for SSH forks`
   - Reference GH#1174

4. Clean up worktree after merge
   ```bash
   git checkout main && git pull
   git worktree remove .worktrees/improve-contributor-detection
   ```

---

## Test Matrix

| Scenario | upstream | API | Token | Expected Detection | Phase |
|----------|----------|-----|-------|-------------------|-------|
| Maintainer, SSH | none | fork=false, push=true | yes | Maintainer (api) | P2 |
| Contributor, HTTPS | none | fork=true, push=false | yes | Contributor (api) | P2 |
| Fork + SSH | yes | - | no | Contributor (upstream) | P1 |
| Fork + SSH | none | fork=true, push=false | yes | Contributor (api) | P2 |
| Offline, cached | - | - | - | (cached value) | P3 |
| Offline, no cache | none | - | no | Heuristic (SSH/HTTPS) | P1 |
