# Design: Fix bd init BEADS_DIR

## Overview

Align `checkExistingBeadsData()` with the BEADS_DIR-first resolution order used throughout beads.

## Current Architecture

```
checkExistingBeadsData(prefix):
  1. Get CWD
  2. If worktree → beadsDir = mainRepoRoot/.beads
  3. Else → beadsDir = cwd/.beads
  4. Check beadsDir for existing data
```

**Problem**: Step 2-3 ignores BEADS_DIR environment variable.

## Proposed Architecture

```
checkExistingBeadsData(prefix):
  1. If BEADS_DIR set → beadsDir = BEADS_DIR (ADDED)
  2. Else if worktree → beadsDir = mainRepoRoot/.beads
  3. Else → beadsDir = cwd/.beads
  4. Check beadsDir for existing data
```

This mirrors the resolution order in `FindBeadsDir()` (lines 482-496).

## Implementation Approach

### Change to `cmd/bd/init.go`

Add BEADS_DIR check at the start of `checkExistingBeadsData()`:

```go
func checkExistingBeadsData(prefix string) error {
    // 1. Check BEADS_DIR environment variable first (preferred, matches FindBeadsDir)
    if envBeadsDir := os.Getenv("BEADS_DIR"); envBeadsDir != "" {
        absBeadsDir := utils.CanonicalizePath(envBeadsDir)
        return checkExistingBeadsDataAt(absBeadsDir, prefix)
    }

    // 2. Existing logic for worktree and local checks
    cwd, err := os.Getwd()
    // ... rest unchanged
}
```

### Helper Extraction (Optional Refactor)

Consider extracting the existence check logic into a helper:

```go
func checkExistingBeadsDataAt(beadsDir string, prefix string) error {
    // Check if .beads directory exists
    if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
        return nil // No .beads directory, safe to init
    }

    // Check for existing database (SQLite or Dolt)
    // ... existing logic from lines 838-858
}
```

**Decision**: Keep inline for Phase 1 (tracer). Refactor if tests reveal duplication issues.

## Key Decisions

### KD-001: BEADS_DIR Takes Absolute Priority

**Decision**: When BEADS_DIR is set, skip ALL local/worktree checks.

**Rationale**: BEADS_DIR is an explicit override. Mixed resolution could cause confusing errors.

**Alternative considered**: Check both BEADS_DIR and local, warn about both. Rejected: adds complexity without benefit.

### KD-002: Error Message References Actual Target

**Decision**: Error message shows BEADS_DIR path when that env var is set.

**Rationale**: User needs to know which path is blocking init.

## Applied Patterns

- **BEADS_DIR-first resolution**: Consistent with `FindBeadsDir()`, `FindDatabasePath()`, `findLocalBeadsDir()`
- **Minimal change principle**: Single early-return, existing code unchanged

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Break existing worktree behavior | Low | High | Test worktree scenarios explicitly |
| Change default behavior | None | High | BEADS_DIR check only triggers when set |

## Test Strategy

1. **Unit test**: `TestCheckExistingBeadsData_WithBEADS_DIR`
2. **Integration**: Manual test in beads-next with BEADS_DIR routing
