# Requirements

## Functional Requirements

### FR-001: Replace --start Flag

WHEN documentation contains `bd daemon --start`
THE SYSTEM SHALL replace it with `bd daemon start`
AND preserve any additional flags (e.g., `--auto-commit`, `--auto-push`)

### FR-002: Replace --stop Flag

WHEN documentation contains `bd daemon --stop`
THE SYSTEM SHALL replace it with `bd daemon stop`

### FR-003: Replace --status Flag

WHEN documentation contains `bd daemon --status`
THE SYSTEM SHALL replace it with `bd daemon status`

### FR-004: Replace --stop-all Flag

WHEN documentation contains `bd daemon --stop-all`
THE SYSTEM SHALL replace it with `bd daemon killall`

### FR-005: Replace --health Flag

WHEN documentation contains `bd daemon --health`
THE SYSTEM SHALL replace it with `bd daemon status --all`

### FR-006: Preserve CHANGELOG History

WHEN the file is `CHANGELOG.md`
THE SYSTEM SHALL NOT modify any daemon flag references
SO THAT historical accuracy is maintained

### FR-007: Preserve Additional Flags

WHEN a deprecated flag is followed by additional flags
THE SYSTEM SHALL preserve those flags after the replacement subcommand

Example:
- `bd daemon --start --auto-commit` → `bd daemon start --auto-commit`
- `bd daemon --stop && bd daemon --start` → `bd daemon stop && bd daemon start`

## Non-Functional Requirements

### NFR-001: Formatting Preservation

THE SYSTEM SHALL preserve markdown formatting including:
- Code block syntax (triple backticks)
- Indentation levels
- Inline code formatting (single backticks)

### NFR-002: Idempotency

WHEN running the replacement multiple times
THE SYSTEM SHALL produce the same result
AND NOT create malformed commands (e.g., `bd daemon start start`)

## Test Matrix

| Test Case | Input | Expected Output |
|-----------|-------|-----------------|
| TC-001 | `bd daemon --start` | `bd daemon start` |
| TC-002 | `bd daemon --start --auto-commit` | `bd daemon start --auto-commit` |
| TC-003 | `bd daemon --stop && bd daemon --start` | `bd daemon stop && bd daemon start` |
| TC-004 | `bd daemon --status` | `bd daemon status` |
| TC-005 | `bd daemon --stop-all` | `bd daemon killall` |
| TC-006 | `bd daemon --health` | `bd daemon status --all` |
| TC-007 | `./bd daemon --start` (prefixed) | `./bd daemon start` |
