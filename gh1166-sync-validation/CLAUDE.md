---
spec_type: implementation
status: draft
created: 2026-01-18
upstream_issue: GH#1166

location:
  remote: github.com/steveyegge/beads
  path: specs/gh1166-sync-validation

beads:
  epic: oss-0rx
  worktree_path: .worktrees/gh1166-sync-validation
  worktree_branch: feature/gh1166-sync-validation

phases:
  - name: "Phase 1: Tracer Bullet"
    type: tracer
    status: pending
    description: "Add config-time validation for sync-branch YAML path"

  - name: "Phase 2: Runtime Guard"
    type: mvs
    status: pending
    description: "Add runtime check in bd sync before worktree operations"

  - name: "Phase 3: Closing"
    type: closing
    status: pending
    merge_strategy: pr
    description: "Create upstream PR and cleanup"

success_criteria:
  - "SC-001: bd config set sync.branch main returns error"
  - "SC-002: bd sync fails gracefully when on sync-branch"
  - "SC-003: Existing ValidateSyncBranchName() and IsSyncBranchSameAsCurrent() utilities reused"
  - "SC-004: All existing tests pass (go test ./...)"
---

# GH#1166: Fix Sync-Branch Validation Bypass

## Overview

Fix two validation gaps that allow `sync.branch = main` configuration, causing `bd sync` to commit all staged files instead of only `.beads/` files.

**Issue**: https://github.com/steveyegge/beads/issues/1166

## Scope

### Files to Modify

| File | Change |
|------|--------|
| `internal/config/yaml_config.go` | Add sync-branch validation in `validateYamlConfigValue()` |
| `cmd/bd/sync.go` | Add runtime check after sync-branch config fetch |

### Files to Read (Context)

| File | Purpose |
|------|---------|
| `internal/syncbranch/syncbranch.go` | `ValidateSyncBranchName()` utility |
| `internal/syncbranch/worktree.go` | `IsSyncBranchSameAsCurrent()` utility |
| `cmd/bd/doctor/git.go` | Reference implementation of runtime check |

### Out of Scope

- Modifying pathspec fix in `gitCommitBeadsDir()` (already working)
- Changing worktree commit behavior in `commitInWorktree()` (separate issue)
- Adding new validation functions (utilities already exist)

## Existing Test Coverage

- `internal/syncbranch/syncbranch_test.go::TestValidateSyncBranchName` — Tests main/master rejection
- `internal/syncbranch/worktree_divergence_test.go::TestIsSyncBranchSameAsCurrent` — Tests dynamic detection
- `cmd/bd/doctor/fix/fix_integration_test.go::TestSyncBranchHealth_CurrentlyOnSyncBranch` — Tests doctor detection

**Gap**: No test for YAML config path validation bypass.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Import cycle config→syncbranch | Low | High | Verified: no reverse dependency exists |
| Breaking valid sync-branch configs | Low | Medium | Existing tests cover valid cases |
| Runtime check in wrong location | Medium | High | Check placement verified at line 247 (before worktree entry) |

**Rollback**: Revert single commit; no data migration required.

## Related Files

- [requirements.md](requirements.md) — EARS requirements
- [design.md](design.md) — Architecture decisions
- [tasks.md](tasks.md) — Phase breakdown
