# GH#1100: Dead Code Cleanup

Remove orphaned `restoreBeadsDirFromBranch` function and add regression test.

## Context

The function `restoreBeadsDirFromBranch()` in `cmd/bd/sync_git.go` was inadvertently
orphaned when PR #918 refactored the sync flow. The bug it caused (GH#1100) no longer
reproduces, but the dead code remains.

---

```yaml
version: "2.0"
name: gh1100-dead-code-cleanup
status: draft
spec_type: implementation

location:
  remote: github.com/steveyegge/beads
  path: specs/gh1100-dead-code-cleanup

phases:
  - name: "Phase 1: Delete Dead Code"
    type: tracer
    status: pending
    description: "Remove restoreBeadsDirFromBranch function"

  - name: "Phase 2: Regression Test"
    type: mvs
    status: pending
    description: "Add test verifying config.yaml preserved during sync"

success_criteria:
  - "SC-001: restoreBeadsDirFromBranch function removed from codebase"
  - "SC-002: No compilation errors after removal"
  - "SC-003: Regression test passes verifying config.yaml preservation"
  - "SC-004: All existing sync tests pass"

beads:
  worktree_path: .worktrees/gh1100-dead-code-cleanup
  worktree_branch: fix/gh1100-dead-code-cleanup
```

---

## Scope

| File | Action |
|------|--------|
| `cmd/bd/sync_git.go` | Delete lines 515-540 (restoreBeadsDirFromBranch) |
| `cmd/bd/sync_test.go` | Add TestConfigPreservedDuringSync |

## Existing Test Coverage

- `TestSyncBranchConfigChange` - Tests config change detection
- `TestSyncBranchConfigPriorityOverUpstream` - Tests config precedence
- **Gap**: No test verifying uncommitted config.yaml survives sync

## References

- GitHub Issue: https://github.com/steveyegge/beads/issues/1100
- PR #918: Refactor that orphaned the function
- Beads Issue: oss-o0b
