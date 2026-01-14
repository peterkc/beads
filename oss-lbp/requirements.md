# Requirements: Fix Path Resolution Bugs

## Functional Requirements

### Helper Extraction

FR-001: WHEN `canonicalizeIfRelative()` is called with a relative path
THE SYSTEM SHALL convert it to an absolute path using `utils.CanonicalizePath()`.

FR-002: WHEN `canonicalizeIfRelative()` is called with an absolute path
THE SYSTEM SHALL return the path unchanged.

FR-003: WHEN `canonicalizeIfRelative()` is called with an empty string
THE SYSTEM SHALL return an empty string.

### Bug 1: Multi-repo Export (oss-lbp)

FR-010: WHEN `exportToRepo()` receives a relative path in `repos.additional`
THE SYSTEM SHALL resolve it relative to the config file directory.

FR-011: WHEN the config file location is unavailable
THE SYSTEM SHALL fall back to resolving relative to the database path directory.

FR-012: WHEN `exportToRepo()` receives an absolute path
THE SYSTEM SHALL use the path unchanged.

### Bug 2: Worktree Redirect (GH#1098)

FR-020: WHEN creating a worktree redirect file
THE SYSTEM SHALL ensure `mainBeadsDir` is absolute before computing relative path.

FR-021: WHEN `filepath.Rel()` is called for redirect computation
THE SYSTEM SHALL receive two absolute paths as arguments.

FR-022: WHEN worktree is nested multiple levels (e.g., `.trees/deep/nested/`)
THE SYSTEM SHALL generate correct `../` depth in redirect file.

### Edge Cases

FR-030: IF the expanded path contains `~` (tilde)
THEN THE SYSTEM SHALL expand tilde before determining relative/absolute status.

FR-031: IF the resulting directory does not exist
THEN THE SYSTEM SHALL create it with appropriate permissions (0755).

## Non-Functional Requirements

### Consistency

NFR-001: THE SYSTEM SHALL use the same path canonicalization pattern across all path resolution code.

NFR-002: THE SYSTEM SHALL produce consistent results regardless of process working directory.

### Maintainability

NFR-010: THE SYSTEM SHALL have a single source of truth for path canonicalization (`utils.CanonicalizePath`).

## Acceptance Criteria

| ID     | Criterion                                               | Verification                                  |
| ------ | ------------------------------------------------------- | --------------------------------------------- |
| AC-001 | `repos.additional: ["oss/"]` exports to correct path    | `bd sync` from repo root vs `.beads/`         |
| AC-002 | `bd worktree create .trees/a/b` generates correct depth | Verify redirect contains `../../../../.beads` |
| AC-003 | Existing absolute path configs continue to work         | Unit test with absolute path                  |
| AC-004 | `autoflush.go` uses extracted helper                    | Code review                                   |

## Out of Scope

- Changing the config file format
- Adding validation for repo paths
- Path resolution in other modules (future work)
