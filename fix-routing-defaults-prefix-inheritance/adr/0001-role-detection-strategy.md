# ADR-0001: Role Detection Strategy for Contributor Routing

> **Note**: This ADR is for GH#1174 but lives in the GH#1165 spec directory
> because both issues share the routing/detection domain.
>
> - GH#1165: Fix routing defaults (Phase 1)
> - GH#1174: Improve detection (Phase 2, blocked by #1165)

## Status

Proposed

> **Dependency**: This work is blocked by PR #1177 (GH#1165). Spec execution
> should begin after PR #1177 is merged.

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

| Option | Approach | Verdict |
|--------|----------|---------|
| A | Check for `upstream` remote (git-only) | Improvement, but still heuristic |
| B | GitHub API via `gh` CLI | Best accuracy, but requires `gh` |
| C | GitHub API via go-github library | Native Go, testable, but new dependency |
| **D** | **Hybrid: Git first, API fallback, cached** | **RECOMMENDED** |

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

## Risks

| Risk | L | I | Mitigation |
|------|---|---|------------|
| go-github breaking changes | M | M | Pin to v81, test in CI |
| GitHub API rate limiting | M | L | Cache forever, warn, fallback |
| Token discovery fails silently | L | M | Verbose logging |
| Stale cached role | L | M | Document invalidation |
| Private repo API access | M | L | Detect 404, clear warning |
| Upstream remote false positive | L | L | API confirms if token available |
| gh CLI not installed | M | - | Fallback paths exist |
| Network latency on first run | H | L | One-time cost, cached |

## Consequences

### Positive

- Correct detection for fork+SSH scenario
- One-time latency, then instant
- Works offline after caching
- Graceful degradation without token

### Negative

- New optional dependency (go-github v81)
- Complexity increase in detection logic
- Users may need to set token for best accuracy

### Neutral

- Existing `git config beads.role` override still works
- Current behavior preserved if no token available

## Related

- GH#1165: Fix routing defaults (blocks this)
- GH#1174: Improve contributor detection (this ADR)
- PR #1177: Implementation of GH#1165 (pending review)
- `internal/routing/routing.go`: Current detection logic
- `internal/beads/context.go`: RepoContext implementation

## Companion Documents

| Document | Content |
|----------|---------|
| [Test Strategy](0001-test-strategy.md) | Test matrix, error reporting, bare repo testing |
| [Implementation Notes](0001-implementation-notes.md) | Code samples, RepoContext, phases, docs updates |
