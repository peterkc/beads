---
beads:
  epic: spec-4fe
  worktree_branch: feature/improve-contributor-detection
  worktree_path: .worktrees/improve-contributor-detection
blocked_by:
  - pr: 1177
    reason: "Routing defaults must be fixed before detection improvements"
created: 2026-01-18
github_issue: 1174
location:
  path: specs/improve-contributor-detection
  remote: github.com/steveyegge/beads
phases:
  - name: "Phase 1: Upstream Remote Detection"
    type: tracer
    status: pending
    description: "Add upstream remote check to DetectUserRole()"
  - name: "Phase 2: GitHub API Integration"
    type: mvs
    status: pending
    description: "Add optional go-github dependency for authoritative fork detection"
  - name: "Phase 3: RepoContext Integration"
    type: mvs
    status: pending
    description: "Integrate role detection with RepoContext, add caching"
  - name: "Phase 4: Documentation"
    type: mvs
    status: pending
    description: "Update ROUTING.md, CONTRIBUTOR_NAMESPACE_ISOLATION.md, TROUBLESHOOTING.md"
  - name: "Phase 5: Closing"
    type: closing
    status: pending
    merge_strategy: pr
skills:
  feature: [golang]
  foundational: [spec, commit, github]
spec_type: implementation
status: draft
success_criteria:
  - "SC-001: Fork+SSH contributor correctly detected as contributor (not maintainer)"
  - "SC-002: Maintainer+HTTPS correctly detected as maintainer (not contributor)"
  - "SC-003: Detection works offline after caching"
  - "SC-004: Graceful degradation when no token available"
  - "SC-005: All existing routing tests pass"
---
# Improve Contributor Detection for SSH Forks

Fixes GH#1174: Fork contributors using SSH are incorrectly detected as maintainers.

> **BLOCKED**: This spec depends on PR #1177 (GH#1165) merging first.
> Check status: `gh pr view 1177 --repo steveyegge/beads`

## Problem Statement

Current SSH/HTTPS heuristic has edge cases:

| Scenario | Current Detection | Actual Role | Result |
|----------|-------------------|-------------|--------|
| Fork contributor with SSH | Maintainer (SSH URL) | Contributor | **WRONG** |
| Maintainer with HTTPS+token | Contributor (HTTPS URL) | Maintainer | **WRONG** |

## Solution: Hybrid Detection Strategy

4-tier detection with caching:

```
1. git config beads.role     → Cached result (instant)
2. upstream remote exists    → Fork signal (git-only)
3. GitHub API (if token)     → Authoritative check
4. SSH/HTTPS heuristic       → Fallback
```

## Scope

### Files to Modify

| File | Change |
|------|--------|
| `internal/routing/routing.go` | Add upstream remote check, GitHub API integration |
| `internal/beads/context.go` | Extend RepoContext with role detection fields |
| `go.mod`, `go.sum` | Add go-github/v81 optional dependency |
| `docs/ROUTING.md` | Update detection strategy documentation |
| `docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md` | Add upstream remote + API detection |
| `docs/TROUBLESHOOTING.md` | Add API fallback troubleshooting |

### Files to Add

| File | Purpose |
|------|---------|
| `internal/routing/detection_test.go` | Role detection unit tests |
| `internal/routing/token_discovery_test.go` | Token discovery path tests |
| `internal/routing/github_client_test.go` | go-github integration tests (httptest) |

## Constraints

1. Can't assume `gh` CLI is installed
2. Should work offline after initial setup
3. Must not add significant latency to every `bd create`
4. Should integrate with existing `RepoContext` pattern

## Links

- [requirements.md](requirements.md) — EARS format requirements
- [design.md](design.md) — Architecture decisions
- [tasks.md](tasks.md) — Phase breakdown
- [ADR: Role Detection Strategy](../fix-routing-defaults-prefix-inheritance/adr/0001-role-detection-strategy.md)

## Interview Context

**Q1: What problem does this solve?**
Fork contributors using SSH are incorrectly routed to maintainer workflow, causing confusion and potential data loss.

**Q2: Who benefits from this?**
All beads users who fork repos and use SSH authentication, especially open source contributors.

**Q3: What's the smallest useful version?**
Add upstream remote detection (git-only, no new dependencies) - Phase 1.

**Q4: What would make this fail?**
Breaking the existing `git config beads.role` override or adding latency to every command.

**Q5: How will we know it worked?**
Fork+SSH users correctly detected as contributors; all existing tests pass.
