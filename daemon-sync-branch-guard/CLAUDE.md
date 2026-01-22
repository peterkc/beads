---
title: 'Daemon Sync Branch Guard'
status: draft
spec_type: implementation
created: '2025-01-22'
upstream_issue: GH#1258

success_criteria:
  - 'SC-001: Guard blocks daemon operations when sync-branch == current-branch'
  - 'SC-002: Daemon startup warns about misconfigured sync-branch'
  - 'SC-003: All test matrix scenarios pass'
  - 'SC-004: Existing daemon tests continue to pass'

phases:
  - name: 'Phase 1: Tracer Bullet'
    type: tracer
    status: pending
    description: "Add guard to one daemon entry point and verify behavior"

  - name: 'Phase 2: Complete Guards'
    type: mvs
    status: pending
    description: "Add guard to all daemon entry points"

  - name: 'Phase 3: Startup Validation'
    type: mvs
    status: pending
    description: "Add daemon startup warning for misconfigured sync-branch"

  - name: 'Phase 4: Closing'
    type: closing
    status: pending
    merge_strategy: pr

beads:
  epic: null
  worktree_path: .worktrees/gh1258-investigation
  worktree_branch: feature/gh1258-investigation

location:
  remote: github.com/peterkc/beads
  path: specs/daemon-sync-branch-guard
---

# Daemon Sync Branch Guard

> Add sync-branch == current-branch guard to daemon code paths (GH#1258).

## Success Criteria

| ID     | Criterion                                            | Validation                                       |
| ------ | ---------------------------------------------------- | ------------------------------------------------ |
| SC-001 | Guard blocks daemon operations when sync==current    | `TestDaemon*SkipsSameBranch` tests pass          |
| SC-002 | Daemon startup warns about misconfigured sync-branch | `TestDaemonStartupWarnsSameBranch` passes        |
| SC-003 | All test matrix scenarios pass                       | `go test ./cmd/bd -run "SameBranch"`             |
| SC-004 | Existing daemon tests continue to pass               | `go test ./cmd/bd -run "TestDaemon\|TestSync"`   |

## Scope

### Files to Modify

| File | Purpose |
|------|---------|
| `cmd/bd/daemon_sync.go` | Add guard to `performExport`, `performAutoImport`, `performSync` |
| `cmd/bd/daemon_sync_branch.go` | Add guard to `syncBranchCommitAndPushWithOptions`, `syncBranchPull` |
| `cmd/bd/daemon.go` | Add startup validation (warn if misconfigured) |

### Files to Reference

| File | Purpose |
|------|---------|
| `cmd/bd/sync.go:348` | Existing guard implementation |
| `internal/syncbranch/worktree.go:1175` | `IsSyncBranchSameAsCurrent()` function |

## Entry Points Requiring Guard

| Entry Point | File | Line | Called By |
|-------------|------|------|-----------|
| `performExport` | daemon_sync.go | 420 | `createExportFunc`, `createLocalExportFunc` |
| `performAutoImport` | daemon_sync.go | 569 | `createAutoImportFunc`, `createLocalAutoImportFunc` |
| `performSync` | daemon_sync.go | 708 | `createSyncFunc`, `createLocalSyncFunc` |
| `syncBranchCommitAndPushWithOptions` | daemon_sync_branch.go | 29 | `syncBranchCommitAndPush` |
| `syncBranchPull` | daemon_sync_branch.go | 250 | `performAutoImport`, `performSync` |

## Risks

| ID    | Risk                            | Likelihood | Impact | Mitigation                           |
| ----- | ------------------------------- | ---------- | ------ | ------------------------------------ |
| R-001 | Guard breaks valid daemon usage | Low        | High   | Fail-open pattern (match sync.go)    |
| R-002 | Dynamic branch switch missed    | Medium     | Medium | Re-check each operation, not startup |

## Test Matrix

| # | Scenario | sync-branch | current-branch | Expected | Test |
|---|----------|-------------|----------------|----------|------|
| 1 | Normal config | `beads-sync` | `main` | Allow | TestDaemonExportAllowsDifferentBranch |
| 2 | Same branch (config) | `main` | `main` | Block | TestDaemonExportSkipsSameBranch |
| 3 | Same branch (env) | `main` (env) | `main` | Block | TestDaemonExportSkipsEnvSameBranch |
| 4 | No sync-branch | (not set) | `main` | Allow | TestDaemonExportAllowsNoSyncBranch |
| 5 | Detached HEAD | `beads-sync` | (detached) | Allow | TestDaemonExportAllowsDetachedHead |
| 6 | Non-git directory | N/A | N/A | Allow | TestDaemonExportAllowsNonGit |
| 7 | Local-only mode | `beads-sync` | `main` | Allow | TestDaemonLocalExportAllows |

## Out of Scope / Future Work

**Worktree-free fallback mode** — Instead of blocking when sync-branch == current-branch, a future enhancement could implement direct pathspec operations:

| Operation | Worktree-free Approach |
|-----------|------------------------|
| Export | `git add .beads/ && git commit` (pathspec-limited) |
| Import | `git fetch && git checkout origin/{branch} -- .beads/` |

This would allow daemon sync to work even with "misconfigured" sync-branch, but requires:
- Different merge semantics (no 3-way merge)
- Conflict handling for .beads/ files
- Testing for edge cases (detached HEAD, bare repos)

**Not included in this spec** — Ship the guard first (safe, simple), consider fallback later.

## Spec Files

- [Requirements](requirements.md) — EARS format
- [Design](design.md) — Architecture decisions
- [Tasks](tasks.md) — Phase breakdown

## Execution

**Always use `/spec:run` to execute phases:**

```bash
/spec:run daemon-sync-branch-guard           # Execute next pending phase
/spec:run daemon-sync-branch-guard --phase 2 # Execute specific phase
```
