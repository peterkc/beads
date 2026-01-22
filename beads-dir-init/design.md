# Design: Fix bd init BEADS_DIR

## Overview

Make `bd init` respect BEADS_DIR environment variable throughout the initialization process:
1. Safety check (`checkExistingBeadsData`)
2. Database path determination (`initDBPath`)
3. Contributor wizard default (`runContributorWizard`)

## Current Architecture

```
bd init execution flow:

1. checkExistingBeadsData(prefix)
   |
   +-- if worktree: beadsDir = mainRepoRoot/.beads
   +-- else: beadsDir = cwd/.beads           <-- BUG: ignores BEADS_DIR
   |
   +-- Check beadsDir for existing data

2. Determine initDBPath
   |
   +-- if --db flag: use flag value
   +-- if BEADS_DB env: use env value
   +-- else: ".beads/beads.db" or ".beads/dolt"  <-- BUG: ignores BEADS_DIR

3. [if --contributor] runContributorWizard()
   |
   +-- Warn if BEADS_DIR set
   +-- Ask for planning repo path
   +-- Default: ~/.beads-planning              <-- BUG: should be BEADS_DIR
```

## Proposed Architecture

```
bd init execution flow (after fix):

1. checkExistingBeadsData(prefix)
   |
   +-- if BEADS_DIR set: beadsDir = BEADS_DIR    <-- NEW: check first
   +-- elif worktree: beadsDir = mainRepoRoot/.beads
   +-- else: beadsDir = cwd/.beads
   |
   +-- Check beadsDir for existing data

2. Determine initDBPath
   |
   +-- if --db flag: use flag value
   +-- if BEADS_DB env: use env value
   +-- if BEADS_DIR env: use BEADS_DIR           <-- NEW: check before default
   +-- else: ".beads/beads.db" or ".beads/dolt"

3. [if --contributor] runContributorWizard()
   |
   +-- Warn if BEADS_DIR set
   +-- Ask for planning repo path
   +-- Default: BEADS_DIR (if set)               <-- NEW: use BEADS_DIR
   +-- Fallback: ~/.beads-planning
```

## Implementation Details

### Change 1: checkExistingBeadsData() in cmd/bd/init.go

Add BEADS_DIR check at start of function:

```go
func checkExistingBeadsData(prefix string) error {
    // NEW: Check BEADS_DIR environment variable first (matches FindBeadsDir pattern)
    if envBeadsDir := os.Getenv("BEADS_DIR"); envBeadsDir != "" {
        absBeadsDir := utils.CanonicalizePath(envBeadsDir)
        // Check this path instead of CWD
        return checkExistingBeadsDataAt(absBeadsDir, prefix)
    }

    // Existing logic for worktree and CWD checks...
    cwd, err := os.Getwd()
    // ...
}
```

### Change 2: initDBPath determination in cmd/bd/init.go

Add BEADS_DIR check before default path:

```go
// Around line 143-151
initDBPath := dbPath
if backend == configfile.BackendDolt {
    if envBeadsDir := os.Getenv("BEADS_DIR"); envBeadsDir != "" {
        initDBPath = filepath.Join(envBeadsDir, "dolt")  // NEW
    } else {
        initDBPath = filepath.Join(".beads", "dolt")
    }
} else if initDBPath == "" {
    if envBeadsDir := os.Getenv("BEADS_DIR"); envBeadsDir != "" {
        initDBPath = filepath.Join(envBeadsDir, beads.CanonicalDatabaseName)  // NEW
    } else {
        localBeadsDir := filepath.Join(".", ".beads")
        targetBeadsDir := beads.FollowRedirect(localBeadsDir)
        initDBPath = filepath.Join(targetBeadsDir, beads.CanonicalDatabaseName)
    }
}
```

### Change 3: runContributorWizard() in cmd/bd/init_contributor.go

Use BEADS_DIR as default when set:

```go
// Around where default planning repo is set
defaultPlanningRepo := "~/.beads-planning"
if envBeadsDir := os.Getenv("BEADS_DIR"); envBeadsDir != "" {
    // Use BEADS_DIR as default since user explicitly set it
    defaultPlanningRepo = envBeadsDir
}

fmt.Printf("Where should contributor planning issues be stored?\n")
fmt.Printf("Default: %s\n", defaultPlanningRepo)
```

## Key Decisions

### KD-001: BEADS_DIR Takes Absolute Priority Over Worktree

**Decision**: When BEADS_DIR is set, skip worktree detection entirely.

**Rationale**: BEADS_DIR is an explicit override. User knows what they want.

**Alternative considered**: Check both, warn about mismatch. Rejected: adds complexity.

### KD-002: BEADS_DIR Overrides --db Flag Precedence

**Decision**: Keep existing precedence: `--db` > `BEADS_DB` > `BEADS_DIR` > default.

**Rationale**: Explicit flags should still win for flexibility.

### KD-003: Wizard Default vs Skip Wizard

**Decision**: If BEADS_DIR set and user continues, use BEADS_DIR as default (not skip).

**Rationale**: User explicitly chose to continue; honor their BEADS_DIR preference.

## Applied Patterns

- **BEADS_DIR-first resolution**: Consistent with `FindBeadsDir()`, `FindDatabasePath()`
- **Minimal change principle**: Early returns, existing code mostly unchanged
- **Precedence preservation**: Explicit flags still override env vars

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Break worktree detection | BEADS_DIR check is additive; worktree logic unchanged when not set |
| Break contributor routing | Routing code untouched; only wizard default affected |
| Change default behavior | All changes guarded by `if BEADS_DIR set` |

## Test Strategy

1. **Unit tests**: Add to `cmd/bd/init_test.go`
   - `TestCheckExistingBeadsData_WithBEADS_DIR`
   - `TestInitDBPath_WithBEADS_DIR`
   - `TestContributorWizard_BEADS_DIR_Default`

2. **Integration**: Manual test in beads-next with BEADS_DIR routing

3. **Regression**: Run full test suite to verify no breakage
