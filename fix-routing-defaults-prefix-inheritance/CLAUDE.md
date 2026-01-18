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
  - name: "Phase 1: Tracer Bullet - Disable Auto-Routing Default"
    type: tracer
    status: pending
    description: "Change routing.mode default from 'auto' to empty, add test"

  - name: "Phase 2: Fix Prefix Inheritance"
    type: mvs
    status: pending
    description: "Fix ensureBeadsDirForPath to correctly inherit prefix with var/ layout"

  - name: "Phase 3: Closing"
    type: closing
    status: pending
    merge_strategy: pr

success_criteria:
  - "SC-001: Fresh bd init + bd create works without routing to ~/.beads-planning"
  - "SC-002: If routing is explicitly enabled, prefix inheritance works correctly"
  - "SC-003: Existing bd init --contributor workflow still works"
  - "SC-004: All existing routing tests pass"
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
3. Auto-routing creates `~/.beads-planning` but fails to inherit prefix

## Scope

### Files to Modify

| File | Change |
|------|--------|
| `internal/config/config.go` | Change `routing.mode` default from `"auto"` to `""` |
| `cmd/bd/create.go` | Fix `ensureBeadsDirForPath()` prefix inheritance |
| `cmd/bd/create.go` | Add debug logging for routing decisions |

### Files to Add

| File | Purpose |
|------|---------|
| `cmd/bd/routing_test.go` | Test routing defaults behavior |

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
