---
spec_type: implementation
status: draft
created: 2026-01-18
github_issue: 1165

beads:
  epic: null
  worktree_path: .worktrees/fix-routing-defaults-prefix-inheritance
  worktree_branch: fix/routing-defaults-prefix-inheritance

location:
  remote: github.com/steveyegge/beads
  path: specs/fix-routing-defaults-prefix-inheritance

phases:
  - name: "Phase 1: Fix Routing Default"
    type: tracer
    status: pending
    description: "Change routing.mode default from 'auto' to empty, add test"

  - name: "Phase 2: Closing"
    type: closing
    status: pending
    merge_strategy: pr

success_criteria:
  - "SC-001: Fresh bd init + bd create works without routing to ~/.beads-planning"
  - "SC-002: Existing bd init --contributor workflow still works"
  - "SC-003: All existing routing tests pass"

notes:
  - "Prefix inheritance for var/ layout deferred to PR #1153"
---

# Fix Routing Defaults and Prefix Inheritance

Fixes GH#1165: Fresh `bd init` unexpectedly routes to `~/.beads-planning`.

## Problem Statement

Users running `bd init --prefix X` followed by `bd create` get:
```
Error: database not initialized: issue_prefix config is missing
```

This happens because:
1. Viper defaults `routing.mode=auto` even without `--contributor` flag
2. Non-SSH remotes trigger "contributor" role detection
3. Auto-routing creates `~/.beads-planning` without inherited prefix

## Scope

### Files to Modify

| File | Change |
|------|--------|
| `internal/config/config.go` | Change `routing.mode` default from `"auto"` to `""` |
| `docs/ROUTING.md` | Update to clarify auto-routing requires opt-in |
| `docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md` | Fix code example showing `routing.mode` default |

### Files to Add

| File | Purpose |
|------|---------|
| `internal/config/config_test.go` | Test routing mode default is empty |

## Test Matrix

| Scenario | Before Fix | After Fix | Status |
|----------|------------|-----------|--------|
| Fresh `bd init` + `bd create` | Routes to ~/.beads-planning, fails with prefix error | Creates issue locally | Must pass |
| `bd init --contributor` + `bd create` | Routes to ~/.beads-planning | Routes to ~/.beads-planning | Regression check |
| `bd config set routing.mode auto` + `bd create` | Routes based on role | Routes based on role | Regression check |
| Existing repo with `routing.mode=auto` in DB | Routes based on role | Routes based on role | Regression check |
| SSH remote (maintainer role) | Creates locally | Creates locally | Regression check |
| HTTPS remote + `--contributor` flag | Routes to planning repo | Routes to planning repo | Regression check |

## Documentation Updates

### docs/ROUTING.md

**Line ~48-50**: Update configuration section to clarify opt-in:

```markdown
# BEFORE
bd config set routing.mode auto

# AFTER
# Auto-routing is disabled by default (routing.mode="")
# Enable with:
bd init --contributor
# OR manually:
bd config set routing.mode auto
```

### docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md

**Line 127-130**: Fix code example:

```go
// BEFORE
v.SetDefault("routing.mode", "auto")

// AFTER
v.SetDefault("routing.mode", "")  // Empty = disabled by default
```

## Links

- [requirements.md](requirements.md) — EARS format requirements
- [design.md](design.md) — Architecture decisions
- [tasks.md](tasks.md) — Phase breakdown

## Interview Context

**Q1: What problem does this solve?**
Users experience confusing "database not initialized" errors after fresh `bd init`.

**Q2: Who benefits from this?**
All new beads users, especially those using HTTPS git remotes.

**Q3: What's the smallest useful version?**
Change the default so auto-routing requires explicit opt-in.

**Q4: What would make this fail?**
Breaking the `bd init --contributor` workflow for users who want auto-routing.

**Q5: How will we know it worked?**
Fresh init + create works locally without errors; existing tests pass.
