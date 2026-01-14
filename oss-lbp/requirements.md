# Requirements: Fix Multi-repo Export Path Resolution

## Functional Requirements

### Core Behavior

FR-001: WHEN `exportToRepo()` receives a relative path (e.g., `oss/`)
THE SYSTEM SHALL resolve it relative to the config file directory.

FR-002: WHEN `exportToRepo()` receives an absolute path
THE SYSTEM SHALL use the path unchanged.

FR-003: WHEN the config file location is unavailable
THE SYSTEM SHALL fall back to resolving relative to the database path directory.

### Edge Cases

FR-010: IF the expanded path contains `~` (tilde)
THEN THE SYSTEM SHALL expand tilde before determining relative/absolute status.

FR-011: IF the resulting directory does not exist
THEN THE SYSTEM SHALL create it with appropriate permissions (0755).

## Non-Functional Requirements

### Consistency

NFR-001: THE SYSTEM SHALL use the same path resolution pattern as `canonicalizeIfRelative()` in autoflush.go.

NFR-002: THE SYSTEM SHALL produce consistent results regardless of daemon's current working directory.

## Acceptance Criteria

| ID     | Criterion                                             | Verification                                    |
| ------ | ----------------------------------------------------- | ----------------------------------------------- |
| AC-001 | `repos.additional: ["oss/"]` exports to correct path  | `bd sync` from repo root vs `.beads/` directory |
| AC-002 | Existing absolute path configs continue to work       | Unit test with absolute path                    |
| AC-003 | No `.beads/oss/.beads/` directories created           | Manual verification after daemon sync           |

## Out of Scope

- Changing the config file format
- Adding validation for repo paths
- Fixing similar issues in other modules (tracked separately)
