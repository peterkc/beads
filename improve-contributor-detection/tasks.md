# Tasks: Improve Contributor Detection

> **BLOCKED**: This spec depends on PR #1177 merging first.

## Phase 1: Upstream Remote Detection

**Type**: Tracer Bullet
**Goal**: Add git-only upstream remote detection (no new dependencies)

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

### Validation

```bash
go test ./internal/routing/... -v
go test ./cmd/bd/... -tags=integration -run=TestUpstreamRemoteDetection -v
golangci-lint run ./internal/routing/...
```

---

## Phase 2: GitHub API Integration

**Type**: MVS Slice
**Goal**: Add optional go-github dependency for authoritative fork detection

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

### Validation

```bash
go build ./...
go test ./internal/routing/... -v
golangci-lint run ./internal/routing/...
```

---

## Phase 3: RepoContext Integration

**Type**: MVS Slice
**Goal**: Integrate role detection with RepoContext, add caching

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

### Validation

```bash
go test ./internal/beads/... -v
go test ./cmd/bd/... -tags=integration -run=TestRoleCaching -v
golangci-lint run ./...
```

---

## Phase 4: Documentation

**Type**: MVS Slice
**Goal**: Update documentation to reflect new detection strategy

### Tasks

1. Update `docs/ROUTING.md`
   - Replace SSH/HTTPS heuristic section with 4-tier strategy
   - Add token discovery documentation
   - Add cache invalidation instructions

2. Update `docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md`
   - Add upstream remote as fork signal
   - Add GitHub API detection section
   - Update code examples

3. Update `docs/TROUBLESHOOTING.md`
   - Add "Role incorrectly detected" section
   - Add API rate limit troubleshooting
   - Add token configuration instructions

4. [P] Add inline code comments for key functions
   - `DetectUserRole()` - explain detection priority
   - `discoverGitHubToken()` - explain discovery order

### Validation

```bash
# Link check (internal only)
lychee --offline docs/*.md

# Manual review of rendered markdown
```

---

## Phase 5: Closing

**Type**: Closing
**Merge Strategy**: PR

### Tasks

1. Run full test suite and linting
   ```bash
   go test ./... -v
   go test ./... -tags=integration -v
   golangci-lint run ./...
   ```

2. Create PR against upstream
   - Title: `feat(routing): improve contributor detection for SSH forks`
   - Reference GH#1174

3. Clean up worktree after merge
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
