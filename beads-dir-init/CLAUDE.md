---
title: 'Fix bd init to respect BEADS_DIR'
status: draft
spec_type: implementation
created: '2026-01-22'

location:
  remote: github.com/steveyegge/beads
  path: specs/beads-dir-init

beads:
  epic: bd-8gc
  worktree_path: .worktrees/beads-dir-init
  worktree_branch: feature/beads-dir-init

success_criteria:
  - "SC-001: bd init respects BEADS_DIR when checking for existing data"
  - "SC-002: Tests pass for BEADS_DIR scenarios"
  - "SC-003: Existing worktree detection behavior unchanged"

phases:
  - name: 'Phase 1: Tracer Bullet'
    type: tracer
    status: pending

  - name: 'Phase 2: Closing'
    type: closing
    status: pending
    merge_strategy: pr
---

# Fix bd init to respect BEADS_DIR

> Align `checkExistingBeadsData()` with BEADS_DIR-first resolution order used throughout beads.

## Success Criteria

| ID     | Criterion                                              | Validation                              |
| ------ | ------------------------------------------------------ | --------------------------------------- |
| SC-001 | bd init respects BEADS_DIR when checking existing data | `go test ./cmd/bd/... -run BEADS_DIR`   |
| SC-002 | Tests pass for BEADS_DIR scenarios                     | `go test ./cmd/bd/... -run Init -v`     |
| SC-003 | Existing worktree behavior unchanged                   | `go test ./cmd/bd/... -run Worktree -v` |

## Scope

- `cmd/bd/init.go` — Add BEADS_DIR check to `checkExistingBeadsData()`
- `cmd/bd/init_test.go` — Add test for BEADS_DIR scenario

## Risks

| ID    | Risk                           | Likelihood | Impact | Mitigation                         |
| ----- | ------------------------------ | ---------- | ------ | ---------------------------------- |
| R-001 | Break existing worktree logic  | Low        | High   | Test worktree scenarios explicitly |
| R-002 | Change default behavior        | None       | High   | BEADS_DIR check only when var set  |

## Unknowns

- None — pattern established in `FindBeadsDir()`

## Atomicity

Each phase is independently mergeable and rollback-safe:

- **Phase 1**: Adds BEADS_DIR check with tests
- **Rollback**: `git revert` safe for any phase

## Problem Statement

`bd init` ignores `BEADS_DIR` when checking for existing beads data:

| Function                   | File                         | Checks BEADS_DIR? |
| -------------------------- | ---------------------------- | ----------------- |
| `FindBeadsDir()`           | `internal/beads/beads.go:482` | Yes, first        |
| `FindDatabasePath()`       | `internal/beads/beads.go:407` | Yes, first        |
| `checkExistingBeadsData()` | `cmd/bd/init.go:811`          | **No**            |

This causes false "already initialized" errors when BEADS_DIR routes to a different directory.

## Spec Files

- [Requirements](requirements.md) — EARS format
- [Design](design.md) — Architecture decisions
- [Tasks](tasks.md) — Phase breakdown

## Execution

**Always use `/spec:run` to execute phases:**

```bash
/spec:run beads-dir-init           # Execute next pending phase
/spec:run beads-dir-init --phase 2 # Execute specific phase
```
