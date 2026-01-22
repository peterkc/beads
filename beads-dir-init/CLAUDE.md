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
  - "SC-001: bd init creates database at BEADS_DIR when env var is set"
  - "SC-002: bd init --contributor wizard uses BEADS_DIR as default planning path"
  - "SC-003: Existing behavior unchanged when BEADS_DIR not set"
  - "SC-004: Tests pass for all BEADS_DIR scenarios in test matrix"

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

> Make `bd init` respect BEADS_DIR environment variable for both database creation and contributor wizard defaults.

## Problem Visual

```
CURRENT BEHAVIOR (Bug)
======================

User sets: BEADS_DIR=/repo/.beads-planning/.beads
User runs: bd init --backend dolt --prefix bdx

    bd init
       |
       v
    checkExistingBeadsData()
       |
       +---> Checks: /repo/.beads/ (CWD)        <-- WRONG! Ignores BEADS_DIR
       |
       v
    Create database
       |
       +---> Creates: /repo/.beads/dolt         <-- WRONG! Ignores BEADS_DIR
       |
       v
    [if --contributor] runContributorWizard()
       |
       +---> Default: ~/.beads-planning         <-- WRONG! Should offer BEADS_DIR
       |
       v
    Result: DB at wrong location, routing to third location


EXPECTED BEHAVIOR (After Fix)
=============================

User sets: BEADS_DIR=/repo/.beads-planning/.beads
User runs: bd init --backend dolt --prefix bdx

    bd init
       |
       v
    checkExistingBeadsData()
       |
       +---> Checks: BEADS_DIR first            <-- CORRECT
       |     (falls back to CWD if not set)
       v
    Create database
       |
       +---> Creates: BEADS_DIR/dolt            <-- CORRECT
       |     (falls back to CWD/.beads if not set)
       v
    [if --contributor] runContributorWizard()
       |
       +---> Default: BEADS_DIR (if set)        <-- CORRECT
       |     (falls back to ~/.beads-planning)
       v
    Result: DB at BEADS_DIR, consistent behavior
```

## Two Related Bugs

| Bug | Location | Issue |
|-----|----------|-------|
| **Bug 1** | `checkExistingBeadsData()` | Ignores BEADS_DIR when checking for existing data |
| **Bug 1b** | Init path determination | Ignores BEADS_DIR when creating database |
| **Bug 2** | `runContributorWizard()` | Uses `~/.beads-planning` as default instead of BEADS_DIR |

These are related: both involve init not respecting BEADS_DIR. The upstream docs state "BEADS_DIR takes precedence over routing" but init doesn't honor this.

## Success Criteria

| ID     | Criterion                                              | Validation                              |
| ------ | ------------------------------------------------------ | --------------------------------------- |
| SC-001 | bd init creates database at BEADS_DIR when set         | `go test ./cmd/bd/... -run BEADS_DIR`   |
| SC-002 | Contributor wizard uses BEADS_DIR as default           | Manual + unit test                      |
| SC-003 | Existing behavior unchanged when BEADS_DIR not set     | `go test ./cmd/bd/... -run Init -v`     |
| SC-004 | All test matrix scenarios pass                         | See Test Matrix below                   |

## Test Matrix

Based on empirical testing (2026-01-22):

| ID | BEADS_DIR | --contributor | upstream remote | Expected DB Location | Expected Wizard Default | Current Behavior |
|----|-----------|---------------|-----------------|---------------------|------------------------|------------------|
| T1 | Not set | No | N/A | `./.beads/` | N/A | ✅ Works |
| T2 | Not set | Yes | Yes | `./.beads/` | `~/.beads-planning` | ✅ Works |
| T3 | Not set | Yes | No | `./.beads/` | `~/.beads-planning` | ✅ Works |
| T4 | Set | No | N/A | `$BEADS_DIR` | N/A | ❌ Uses CWD |
| T5 | Set | Yes | Yes | `$BEADS_DIR` | `$BEADS_DIR` | ❌ Uses CWD, wrong default |
| T6 | Set | Yes | No | `$BEADS_DIR` | `$BEADS_DIR` | ❌ Uses CWD, wrong default |
| T7 | Set (exists) | No | N/A | Error (already init) | N/A | ❌ Checks CWD instead |
| T8 | Set | No (worktree) | N/A | `$BEADS_DIR` | N/A | ❌ Uses main root |

**Legend:**
- T1-T3: BEADS_DIR not set (current behavior correct)
- T4-T6: BEADS_DIR set (bugs manifest)
- T7: BEADS_DIR points to existing DB, CWD has different DB
- T8: In worktree with BEADS_DIR set

## Scope

### Files to Modify

| File | Change |
|------|--------|
| `cmd/bd/init.go` | Add BEADS_DIR check to `checkExistingBeadsData()` |
| `cmd/bd/init.go` | Use BEADS_DIR for `initDBPath` determination |
| `cmd/bd/init_contributor.go` | Use BEADS_DIR as default planning path when set |
| `cmd/bd/init_test.go` | Add tests for BEADS_DIR scenarios |

### Out of Scope

- Changes to `FindBeadsDir()` (already correct)
- Changes to routing logic in `internal/routing/`
- Daemon behavior changes

## Risks

| ID    | Risk                           | Likelihood | Impact | Mitigation                         |
| ----- | ------------------------------ | ---------- | ------ | ---------------------------------- |
| R-001 | Break existing worktree logic  | Low        | High   | Test worktree scenarios explicitly |
| R-002 | Change default behavior        | None       | High   | BEADS_DIR check only when var set  |
| R-003 | Break contributor routing      | Low        | Medium | Routing code untouched             |

## Atomicity

Each phase is independently mergeable and rollback-safe:

- **Phase 1**: Adds BEADS_DIR check with tests
- **Rollback**: `git revert` safe for any phase

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
