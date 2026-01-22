# Requirements: Fix bd init BEADS_DIR

## Functional Requirements

### FR-001: BEADS_DIR Priority in Existence Check

WHEN the user runs `bd init` AND `BEADS_DIR` environment variable is set THE SYSTEM SHALL check the `BEADS_DIR` path for existing beads data instead of the current directory.

Rationale: Matches the resolution order used by `FindBeadsDir()` and other runtime functions.

### FR-002: BEADS_DIR Priority in Database Creation

WHEN the user runs `bd init` AND `BEADS_DIR` environment variable is set AND no existing database at BEADS_DIR THE SYSTEM SHALL create the database at `BEADS_DIR` instead of `CWD/.beads`.

Rationale: Init should create the database where runtime operations will look for it.

### FR-003: BEADS_DIR as Contributor Wizard Default

WHEN the user runs `bd init --contributor` AND `BEADS_DIR` environment variable is set AND user continues despite the warning THE SYSTEM SHALL offer `BEADS_DIR` as the default planning repo path instead of `~/.beads-planning`.

Rationale: If user explicitly continues with BEADS_DIR set, they likely want that path used.

### FR-004: Skip CWD Check When BEADS_DIR Set

WHEN the user runs `bd init` AND `BEADS_DIR` environment variable is set THE SYSTEM SHALL NOT check CWD for existing beads data, only check BEADS_DIR.

Rationale: CWD may have unrelated `.beads/` that shouldn't block init at BEADS_DIR.

### FR-005: Error Message References Correct Path

WHEN the user runs `bd init` AND `BEADS_DIR` is set AND existing database found THE SYSTEM SHALL display an error mentioning the `BEADS_DIR` path, not the CWD path.

Rationale: User needs to know which path is blocking init.

## Non-Functional Requirements

### NFR-001: Backward Compatibility

THE SYSTEM SHALL maintain existing behavior when `BEADS_DIR` is not set.

Validation: `go test ./cmd/bd/... -run Init -v` (existing tests pass)

### NFR-002: Worktree Behavior Fallback

WHEN `BEADS_DIR` is not set THE SYSTEM SHALL maintain worktree detection and main-repo-root resolution.

Validation: `go test ./cmd/bd/... -run Worktree -v`

### NFR-003: Routing Code Unaffected

THE SYSTEM SHALL NOT modify the routing logic in `internal/routing/`.

Validation: `go test ./internal/routing/... -v`

## Test Cases

| ID | Scenario | BEADS_DIR | --contributor | Expected |
|----|----------|-----------|---------------|----------|
| TC-001 | Basic init, no BEADS_DIR | Not set | No | DB at `./.beads/` |
| TC-002 | Init with BEADS_DIR | Set to `/tmp/test/.beads` | No | DB at `/tmp/test/.beads/` |
| TC-003 | Init with BEADS_DIR, CWD has .beads | Set | No | Ignores CWD, uses BEADS_DIR |
| TC-004 | Init with BEADS_DIR, target exists | Set (has DB) | No | Error references BEADS_DIR |
| TC-005 | Contributor wizard with BEADS_DIR | Set | Yes | Wizard default = BEADS_DIR |
| TC-006 | Contributor wizard, no BEADS_DIR | Not set | Yes | Wizard default = ~/.beads-planning |
| TC-007 | Worktree, no BEADS_DIR | Not set | No | DB at main repo root |
| TC-008 | Worktree with BEADS_DIR | Set | No | DB at BEADS_DIR (overrides worktree) |
