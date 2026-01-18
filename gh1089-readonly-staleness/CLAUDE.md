# Fix Read-Only Commands Producing SQLite Write Warnings (GH#1089)

```yaml
spec_type: implementation
status: draft
created: 2026-01-17
github_issue: https://github.com/steveyegge/beads/issues/1089

success_criteria:
  - "SC-001: bd --no-daemon ready runs without sqlite write warnings when DB is stale"
  - "SC-002: bd --no-daemon list runs without sqlite write warnings when DB is stale"
  - "SC-003: All existing staleness tests pass"
  - "SC-004: New tests cover pre-store staleness detection"

phases:
  - name: "Phase 1: Tracer Bullet"
    type: tracer
    status: pending
    description: "Add isJSONLNewerThanDB() helper and wire into read-only decision"

  - name: "Phase 2: Edge Cases & Tests"
    type: mvs
    status: pending
    description: "Handle edge cases (missing files, clock skew) and add comprehensive tests"

  - name: "Phase 3: Closing"
    type: closing
    status: pending
    merge_strategy: pr

beads:
  epic: oss-2pg
  worktree_path: .worktrees/gh1089-readonly-staleness
  worktree_branch: feature/gh1089-readonly-staleness

location:
  remote: github.com/peterkc/beads-specs  # Nested repo in beads fork
  path: gh1089-readonly-staleness
```

## Problem Statement

Read-only commands (`bd ready`, `bd list`, `bd show`) produce spurious SQLite write warnings when the JSONL file is newer than the database:

```
Warning: failed to clear export_hashes before import: failed to clear export hashes: sqlite3: attempt to write a readonly database
```

## Root Cause

The read-only mode decision is made **before** staleness is known:

1. `main.go:749`: `useReadOnly := isReadOnlyCommand(cmd.Name())` — based only on command name
2. `main.go:769`: Store opened with `opts.ReadOnly = useReadOnly`
3. Later: `ensureDatabaseFresh()` → `autoImportIfNewer()` → `ClearAllExportHashes()` → **fails**

## Solution

Check staleness **before** determining store mode (Option B from the original issue). If DB is stale, open in read-write mode to allow auto-import.

## Scope

### Files to Modify

| File | Changes |
|------|---------|
| `cmd/bd/main.go` | Add pre-store staleness check before line 749 |
| `cmd/bd/staleness.go` | Add `isJSONLNewerThanDB()` helper function |
| `cmd/bd/staleness_test.go` | Add tests for new helper |

### Files to Read (Context)

- `internal/autoimport/autoimport.go` — Existing mtime comparison patterns
- `cmd/bd/autoflush.go:203-340` — `autoImportIfNewer()` implementation

## Links

- [requirements.md](requirements.md) — EARS format requirements
- [design.md](design.md) — Architecture decisions
- [tasks.md](tasks.md) — Phase breakdown
