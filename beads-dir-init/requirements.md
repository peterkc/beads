# Requirements: Fix bd init BEADS_DIR

## Functional Requirements

### FR-001: BEADS_DIR Priority in Existence Check

WHEN the user runs `bd init` AND `BEADS_DIR` environment variable is set THE SYSTEM SHALL check the `BEADS_DIR` path for existing beads data before checking the current directory.

Rationale: Matches the resolution order used by `FindBeadsDir()` and other runtime functions.

### FR-002: Skip Local Check When BEADS_DIR Set

WHEN the user runs `bd init` AND `BEADS_DIR` environment variable is set AND `BEADS_DIR` target does not have existing beads data THE SYSTEM SHALL proceed with initialization at `BEADS_DIR` target, ignoring any `.beads/` in the current directory.

Rationale: The local `.beads/` is irrelevant when BEADS_DIR routes elsewhere.

### FR-003: Error on Existing Data at BEADS_DIR Target

WHEN the user runs `bd init` AND `BEADS_DIR` environment variable is set AND `BEADS_DIR` target already has beads data THE SYSTEM SHALL display an error mentioning the `BEADS_DIR` path (not local path).

Rationale: Error message should reference the actual target, not a potentially unrelated local directory.

## Non-Functional Requirements

### NFR-001: Backward Compatibility

THE SYSTEM SHALL maintain existing behavior when `BEADS_DIR` is not set.

Validation: `go test ./cmd/bd/... -run Init -v`

### NFR-002: Worktree Behavior Unchanged

THE SYSTEM SHALL maintain worktree detection and main-repo-root resolution when `BEADS_DIR` is not set.

Validation: `go test ./cmd/bd/... -run Worktree -v`

## Test Cases

| ID     | Scenario                                      | Expected                              |
| ------ | --------------------------------------------- | ------------------------------------- |
| TC-001 | BEADS_DIR set, target empty, local has .beads | Init succeeds at BEADS_DIR            |
| TC-002 | BEADS_DIR set, target has data                | Error references BEADS_DIR path       |
| TC-003 | BEADS_DIR not set, local has .beads           | Error references local path (current) |
| TC-004 | BEADS_DIR not set, worktree, main has .beads  | Error references main path (current)  |
