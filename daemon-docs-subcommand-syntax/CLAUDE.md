---
name: daemon-docs-subcommand-syntax
version: "1.0"
spec_type: documentation
status: draft
upstream_issue: https://github.com/steveyegge/beads/issues/1050

location:
  remote: github.com/peterkc/beads
  branch: specs
  path: daemon-docs-subcommand-syntax

beads:
  worktree_path: .worktrees/daemon-docs
  worktree_branch: docs/daemon-subcommand-syntax

phases:
  - name: "Phase 1: Tracer Bullet"
    type: tracer
    status: pending
    description: "Update one file to validate replacement pattern"

  - name: "Phase 2: Bulk Updates"
    type: mvs
    status: pending
    description: "Update remaining documentation files"

  - name: "Phase 3: Verification"
    type: mvs
    status: pending
    description: "Validate all changes and test commands"

success_criteria:
  - "SC-001: Zero occurrences of deprecated daemon flags in docs (excluding CHANGELOG)"
  - "SC-002: All replacement commands execute without error"
  - "SC-003: Documentation renders correctly (no broken formatting)"
---

# Daemon Documentation Subcommand Syntax Update

Update all documentation to use the new subcommand syntax for `bd daemon` commands.

## Context

The `bd daemon` CLI evolved from flag-based (`--start`) to subcommand-based (`daemon start`) syntax. The old flags still work but emit deprecation warnings. Documentation should reflect current best practices.

## Upstream Issue

[GH#1050](https://github.com/steveyegge/beads/issues/1050) - `bd daemon --start` is now `bd daemon start`, but the docs need updating

## Scope

### Files to Modify

| File | Occurrences | Priority |
|------|-------------|----------|
| `docs/PROTECTED_BRANCHES.md` | 12 | High |
| `docs/DAEMON.md` | 5 | High |
| `integrations/beads-mcp/SETUP_DAEMON.md` | 4 | Medium |
| `claude-plugin/commands/daemon.md` | 4 | Medium |
| `examples/team-workflow/README.md` | 4 | Low |
| `examples/protected-branch/README.md` | 3 | Low |
| `claude-plugin/skills/beads/resources/TROUBLESHOOTING.md` | 1 | Low |
| `integrations/beads-mcp/README.md` | 1 | Low |
| `examples/multiple-personas/README.md` | 1 | Low |
| `examples/multi-phase-development/README.md` | 1 | Low |
| `docs/WORKTREES.md` | 1 | Low |
| `docs/TROUBLESHOOTING.md` | 1 | Low |

**Total**: 40 occurrences across 12 files

### Files to Preserve (Historical)

- `CHANGELOG.md` — Contains historical records of when flags existed; should remain unchanged

## Replacement Mapping

| Deprecated | New Syntax |
|------------|------------|
| `bd daemon --start` | `bd daemon start` |
| `bd daemon --stop` | `bd daemon stop` |
| `bd daemon --status` | `bd daemon status` |
| `bd daemon --stop-all` | `bd daemon killall` |
| `bd daemon --health` | `bd daemon status --all` |

## Related Files

- [requirements.md](requirements.md) — EARS format requirements
- [design.md](design.md) — Replacement strategy
- [tasks.md](tasks.md) — Phase breakdown
