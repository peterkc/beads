# Requirements

## Functional Requirements

### FR-001: Pre-Store Staleness Detection

WHEN a read-only command is invoked in --no-daemon mode
THE SYSTEM SHALL check if JSONL file mtime is newer than database file mtime
BEFORE opening the SQLite store.

### FR-002: Dynamic Read-Only Mode Selection

WHEN JSONL is detected as newer than database
THE SYSTEM SHALL open the store in read-write mode
SO THAT auto-import can proceed without write failures.

### FR-003: Preserve Read-Only Optimization

WHEN JSONL is NOT newer than database
THE SYSTEM SHALL open the store in read-only mode
SO THAT file watcher noise is avoided (per GH#804).

### FR-004: Missing File Handling

WHEN database file does not exist
THE SYSTEM SHALL default to read-write mode
SO THAT initial bootstrap can proceed.

### FR-005: Missing JSONL Handling

WHEN JSONL file does not exist
THE SYSTEM SHALL proceed with original read-only decision
SO THAT fresh repositories without JSONL work correctly.

## Non-Functional Requirements

### NFR-001: No Store Dependency

The staleness check MUST NOT require an open store connection.
Rationale: The check must run before the store is opened.

### NFR-002: Minimal Overhead

The staleness check SHOULD complete in <1ms for typical file sizes.
Rationale: This runs on every read-only command invocation.

### NFR-003: Symlink Awareness

The staleness check MUST use `os.Lstat` for JSONL mtime.
Rationale: Matches existing behavior in `autoimport.CheckStaleness` for NixOS compatibility.

## Test Requirements

### TR-001: Stale DB Scenario

WHEN database is stale (JSONL newer)
AND user runs `bd --no-daemon ready`
THEN no sqlite write warning appears
AND issues are correctly displayed.

### TR-002: Fresh DB Scenario

WHEN database is fresh (DB newer or equal)
AND user runs `bd --no-daemon list`
THEN store opens in read-only mode
AND no auto-import occurs.

### TR-003: Missing DB Scenario

WHEN database does not exist
AND JSONL exists
AND user runs `bd --no-daemon show bd-xxx`
THEN store opens in read-write mode
AND auto-bootstrap proceeds.
