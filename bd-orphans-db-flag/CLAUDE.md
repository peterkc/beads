# Spec: Fix bd orphans ignoring --db flag

## Metadata

```yaml
spec_type: implementation
status: in_progress
created: 2026-01-19
upstream_issue: https://github.com/steveyegge/beads/issues/1196
tracking_issue: oss-zo5

phases:
  - name: "Phase 1: Tracer Bullet + All Callers"
    type: tracer
    status: completed
    description: "Define interface, update signature, update ALL callers atomically (build never breaks)"
    commit: a7375078

  - name: "Phase 2: Test Coverage"
    type: mvs
    status: completed
    description: "Mock provider tests, cross-repo integration tests"
    commit: 7e61c130

  - name: "Phase 3: Closing"
    type: closing
    status: pending
    merge_strategy: pr
    description: "PR creation and upstream submission"

success_criteria:
  - "SC-001: IssueProvider interface defined with GetOpenIssues() and GetIssuePrefix()"
  - "SC-002: bd --db /path orphans reads issues from specified database via provider"
  - "SC-003: FindOrphanedIssues() accepts IssueProvider instead of hardcoded path"
  - "SC-004: All existing orphans tests pass (no regressions)"
  - "SC-005: New cross-repo test case passes with mock provider"
  - "SC-006: PR accepted by upstream maintainer"

beads:
  # Tracked via ACF oss/ repo (oss-zo5) since beads specs/ is plain git
  external_tracking: oss-zo5
  worktree_path: .worktrees/orphans-db-flag
  worktree_branch: fix/orphans-db-flag

location:
  remote: github.com/peterkc/beads
  path: specs/bd-orphans-db-flag
```

## Problem Statement

The `bd orphans --db /path/to/other.db` command ignores the `--db` flag entirely. The `FindOrphanedIssues()` function in `cmd/bd/doctor/git.go` hardcodes the database path to the local `.beads/` directory, preventing cross-repo orphan detection.

This breaks the separate beads repo workflow where:
- Planning repo contains `.beads/` with issues (e.g., `PLAN-xxx`)
- Code repo contains `.git/` with commits referencing `PLAN-xxx`
- User wants to detect orphans from the code repo using the planning repo's database

## Scope

### In Scope

| File | Changes |
|------|---------|
| `cmd/bd/doctor/git.go` | Add `dbPath` parameter to `FindOrphanedIssues()` |
| `cmd/bd/orphans.go` | Pass global `dbPath` to `FindOrphanedIssues()` |
| `cmd/bd/doctor.go` | Update call site (if any) |
| `cmd/bd/orphans_test.go` | Add cross-repo test case |

### Out of Scope

- Refactoring orphans to use global `store` (larger change, separate PR)
- Adding `--db` support to other doctor subcommands
- Documentation updates (will be handled in PR description)

## Interview Context

**Q1: What problem does this solve?**
Cross-repo orphan detection is the symptom, but the root issue is `--db` flag consistency across all commands.

**Q2: Who benefits?**
All bd users who use the `--db` flag for any reason.

**Q3: What's the smallest useful version?**
Fix `FindOrphanedIssues()` signature to accept `dbPath`, wire it through `orphans.go`.

**Q4: What would make this fail?**
- Breaking existing callers of `FindOrphanedIssues()` (signature change)
- Test coverage gaps (no existing cross-repo tests)
- Upstream rejection (maintainer may prefer different approach)

**Q5: How will we know it worked?**
Both manual sandbox test AND new unit test pass, AND upstream accepts the PR.

## Related Files

- [requirements.md](./requirements.md) - EARS format requirements
- [design.md](./design.md) - Architecture decisions
- [tasks.md](./tasks.md) - Phase breakdown
