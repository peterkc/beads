# Requirements: .beads/var/ Layout Migration

<!-- EARS Atomization Rule: One behavior per requirement.
     Split compound AND statements into separate FR-### entries.
-->

## Functional Requirements

### Core Behavior

FR-001: WHEN a volatile file path is requested
THE SYSTEM SHALL check if `.beads/var/` directory exists.

FR-002: IF `.beads/var/` directory exists
THEN THE SYSTEM SHALL return paths within the `var/` subdirectory.

FR-003: IF `.beads/var/` directory does NOT exist
THEN THE SYSTEM SHALL return paths at the `.beads/` root (legacy layout).

FR-004: WHEN `bd migrate var` is executed
THE SYSTEM SHALL create the `.beads/var/` directory.

FR-005: WHEN `bd migrate var` is executed
THE SYSTEM SHALL copy all volatile files to the `var/` subdirectory.

FR-006: WHEN `bd migrate var` is executed
THE SYSTEM SHALL preserve original files until `--cleanup` flag is used.

FR-007: WHEN `bd migrate var --dry-run` is executed
THE SYSTEM SHALL display planned changes without modifying files.

FR-008: WHEN `bd doctor` detects legacy layout
THE SYSTEM SHALL report var-layout migration as optional (priority 3).

### Daemon Files

FR-010: WHEN daemon starts
THE SYSTEM SHALL create pid file in VarPath("daemon.pid").

FR-011: WHEN daemon starts
THE SYSTEM SHALL create log file in VarPath("daemon.log").

FR-012: WHEN daemon starts
THE SYSTEM SHALL create lock file in VarPath("daemon.lock").

FR-013: WHEN daemon creates socket
THE SYSTEM SHALL use VarPath("bd.sock") if path length permits.

### Database Files

FR-020: WHEN database path is requested
THE SYSTEM SHALL return VarPath(config.Database).

FR-021: WHEN database connection opens
THE SYSTEM SHALL create journal/WAL files adjacent to database.

### Sync Files

FR-030: WHEN sync base state is saved
THE SYSTEM SHALL write to VarPath("sync_base.jsonl").

FR-031: WHEN sync lock is acquired
THE SYSTEM SHALL create VarPath(".sync.lock").

FR-032: WHEN sync state is saved
THE SYSTEM SHALL write to VarPath("sync-state.json").

### Gitignore

FR-040: WHEN gitignore template is generated
THE SYSTEM SHALL include `var/` as first pattern.

FR-041: WHEN gitignore template is generated
THE SYSTEM SHALL include legacy patterns for backward compatibility.

FR-042: WHEN gitignore is checked by doctor
THE SYSTEM SHALL accept both var/ and legacy patterns as valid.

### Edge Cases

FR-050: IF `redirect` file is needed
THEN THE SYSTEM SHALL create it at `.beads/redirect` (NOT in var/).

FR-051: IF migration is run on already-migrated directory
THEN THE SYSTEM SHALL exit with success message (idempotent).

FR-052: IF migration fails mid-operation
THEN THE SYSTEM SHALL preserve original files for manual recovery.

FR-053: IF BD_LEGACY_LAYOUT=1 environment variable is set
THEN THE SYSTEM SHALL use legacy layout regardless of var/ existence.

### Coexistence (Read-Both Pattern)

FR-060: WHEN reading a volatile file
THE SYSTEM SHALL check var/ location first.

FR-061: IF file exists in var/
THEN THE SYSTEM SHALL return the var/ path.

FR-062: IF file does NOT exist in var/ AND file exists at root
THEN THE SYSTEM SHALL return the root path (fallback).

FR-063: WHEN writing a new volatile file
THE SYSTEM SHALL use VarPathForWrite() which respects layout preference without fallback.

FR-064: THE SYSTEM SHALL NOT fail if volatile files exist in both locations simultaneously.

FR-065: WHEN bd doctor runs AND var/ layout is active AND volatile files exist at root
THE SYSTEM SHALL report "files in wrong location" as warning (Priority 2).

FR-066: WHEN bd doctor --fix runs AND files are in wrong location
THE SYSTEM SHALL move them to var/ automatically (consistent with other --fix behavior).

FR-067: WHEN bd doctor runs AND var/ layout is NOT active
THE SYSTEM SHALL report var/ migration as optional info (Priority 4).

FR-068: THE SYSTEM SHALL use `bd migrate var` for initial migration (creates var/).

FR-069: THE SYSTEM SHALL use `bd doctor --fix` for stray file cleanup (var/ already exists).

## Non-Functional Requirements

### Performance

NFR-001: THE SYSTEM SHALL complete layout detection within 1ms (single stat call).

NFR-002: THE SYSTEM SHALL NOT add latency to normal operations.

### Backward Compatibility

NFR-010: THE SYSTEM SHALL support legacy layout for minimum 6 months.

NFR-011: THE SYSTEM SHALL NOT require migration for existing users.

NFR-012: THE SYSTEM SHALL work with mixed-layout clones during sync.

### Security

NFR-020: THE SYSTEM SHALL create var/ directory with 0700 permissions.

NFR-021: THE SYSTEM SHALL NOT expose file paths in error messages.

## Acceptance Criteria

| ID     | Criterion                                        | Verification                                |
| ------ | ------------------------------------------------ | ------------------------------------------- |
| AC-001 | All existing tests pass                          | `go test ./...`                             |
| AC-002 | New layout works with all commands               | Integration test: bd list/create/show/sync  |
| AC-003 | Legacy layout continues working                  | Integration test: no var/ directory present |
| AC-004 | Migration command works                          | `bd migrate var` on test database           |
| AC-005 | Doctor shows optional migration                  | `bd doctor` output includes var-layout      |
| AC-006 | Mixed-layout sync works                          | Clone A (legacy) syncs with Clone B (var/)  |
| AC-007 | Daemon starts in both layouts                    | `bd daemon start && bd daemon stop`         |
| AC-008 | Environment variable override works              | `BD_LEGACY_LAYOUT=1 bd list`                |

## Sync-Branch Compatibility

**No changes needed** for sync-branch mode (protected branches workflow):

| Component | Sync Branch | var/ Impact |
|-----------|-------------|-------------|
| `issues.jsonl` | ✅ Committed to beads-sync | Stays at root |
| `metadata.json` | ✅ Committed to beads-sync | Stays at root |
| `sync_base.jsonl` | ❌ Gitignored (per-machine) | Moves to var/ |
| Worktree checkout | `.git/beads-worktrees/` | Unaffected |

The sync worktree only checks out tracked files — all files moving to `var/` are already gitignored.

## Out of Scope

- Changing `redirect` file location (breaks worktree discovery)
- Auto-migration during `bd init` (too aggressive for users)
- Removing legacy layout support (requires deprecation cycle)
- Migrating Python MCP integration paths (separate issue)
