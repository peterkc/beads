# Requirements: Improve Contributor Detection

## Functional Requirements

### FR-001: Upstream Remote Detection
WHEN a repository has an `upstream` remote configured
THE SYSTEM SHALL detect the user as a contributor
AND log "Fork detected via upstream remote" at verbose level

### FR-002: GitHub API Fork Check
WHEN a GitHub token is available AND the upstream remote detection is inconclusive
THE SYSTEM SHALL query GitHub API for repository fork status
AND cache the result in git config `beads.role`

### FR-003: Token Discovery Chain
WHEN the system needs a GitHub token for API calls
THE SYSTEM SHALL check in order:
1. `GITHUB_TOKEN` environment variable
2. `gh auth token` command output
3. Git credential helper for github.com

### FR-004: Role Caching
WHEN role detection completes via API
THE SYSTEM SHALL store the result in `git config beads.role`
AND use cached value for subsequent calls

### FR-005: Graceful Degradation
WHEN GitHub API is unavailable or rate limited
THE SYSTEM SHALL fall back to SSH/HTTPS heuristic
AND warn "GitHub API {error}, using heuristic (may be inaccurate)"

### FR-006: Cache Invalidation
WHEN user runs `git config --unset beads.role`
THE SYSTEM SHALL perform fresh detection on next command

### FR-007: RepoContext Integration
WHEN RepoContext is constructed
THE SYSTEM SHALL populate role detection fields (UserRole, RoleSource, IsFork)
AND make them available to all consumers

## Non-Functional Requirements

### NFR-001: Performance
WHEN role detection runs for the first time
THE SYSTEM SHALL complete within 2 seconds (network call)
AND subsequent calls SHALL complete within 10ms (cached)

### NFR-002: Offline Support
WHEN no network is available AND role is cached
THE SYSTEM SHALL use cached role without error

### NFR-003: Minimal Dependencies
THE SYSTEM SHALL NOT require `gh` CLI to be installed
AND go-github dependency SHALL be optional (for API path only)

### NFR-004: Backward Compatibility
WHEN `git config beads.role` is manually set
THE SYSTEM SHALL respect the manual override
AND skip all detection logic

## Test Requirements

### TR-001: Unit Tests
THE SYSTEM SHALL have unit tests for:
- Token discovery (all 3 sources)
- Role detection priority (4 tiers)
- Cache read/write

### TR-002: Integration Tests
THE SYSTEM SHALL have integration tests using:
- Bare repo fixtures for upstream remote detection
- httptest mocks for GitHub API
- No real tokens required

### TR-003: E2E Tests (Optional)
THE SYSTEM MAY have E2E tests with:
- Real GitHub API calls
- Requires `GITHUB_TOKEN` environment variable
- Build tag: `integration,live`
