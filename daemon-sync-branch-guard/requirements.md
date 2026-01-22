# Requirements: Daemon Sync Branch Guard

## Functional Requirements

### FR-001: Guard Daemon Export

WHEN daemon triggers export AND sync-branch equals current-branch
THE SYSTEM SHALL skip the export operation with a log message explaining why

### FR-002: Guard Daemon Import

WHEN daemon triggers auto-import AND sync-branch equals current-branch
THE SYSTEM SHALL skip the import operation with a log message explaining why

### FR-003: Guard Daemon Sync Cycle

WHEN daemon triggers sync cycle AND sync-branch equals current-branch
THE SYSTEM SHALL skip the sync cycle with a log message explaining why

### FR-004: Guard Sync Branch Commit

WHEN sync branch commit is attempted AND sync-branch equals current-branch
THE SYSTEM SHALL return early without committing, with a log message explaining why

### FR-005: Guard Sync Branch Pull

WHEN sync branch pull is attempted AND sync-branch equals current-branch
THE SYSTEM SHALL return early without pulling, with a log message explaining why

### FR-006: Daemon Startup Validation

WHEN daemon starts AND sync-branch is configured AND sync-branch equals current-branch
THE SYSTEM SHALL log a warning message explaining the misconfiguration

### FR-007: Dynamic Branch Detection

WHILE daemon is running
THE SYSTEM SHALL re-check sync-branch vs current-branch before each operation
(NOT just at startup, because user may switch branches)

### FR-008: BEADS_SYNC_BRANCH Override Detection

WHEN BEADS_SYNC_BRANCH env var is set AND its value equals current-branch
THE SYSTEM SHALL apply the same guard as config-based sync-branch

## Non-Functional Requirements

### NFR-001: Fail-Open on Detection Error

WHEN branch detection fails (e.g., not a git repo, detached HEAD)
THE SYSTEM SHALL allow the operation to proceed (fail-open)
(Matches existing guard behavior in sync.go)

### NFR-002: Log Format Consistency

WHEN guard blocks an operation
THE SYSTEM SHALL use consistent log message format:
"Skipping {operation}: sync-branch '{branch}' is your current branch"

### NFR-003: Local-Only Mode Compatibility

WHEN daemon runs with --local flag (no remote)
THE SYSTEM SHALL skip sync-branch guards (no sync-branch in local mode)

## Out of Scope

- Blocking daemon startup (only warn, don't prevent)
- Auto-correcting misconfigured sync-branch
- UI prompts for reconfiguration
- Worktree-free fallback mode (future enhancement — see CLAUDE.md)
