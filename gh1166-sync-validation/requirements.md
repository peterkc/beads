# Requirements: GH#1166 Sync-Branch Validation

## Functional Requirements

### FR-001: Config-Time Validation for YAML Path

WHEN a user runs `bd config set sync.branch <value>`
AND the value equals "main" or "master"
THE SYSTEM SHALL reject the configuration with an error message explaining that main/master cannot be used as sync branch.

### FR-002: Config-Time Validation for YAML Path (Alternate Keys)

WHEN a user runs `bd config set sync-branch <value>`
THE SYSTEM SHALL apply the same validation as FR-001.

### FR-003: Runtime Validation Before Worktree Operations

WHEN `bd sync` is invoked
AND sync.branch is configured
AND the current branch equals the sync branch
THE SYSTEM SHALL exit with an error before attempting any worktree operations.

### FR-004: Clear Error Message for Runtime Conflict

WHEN the runtime validation in FR-003 fails
THE SYSTEM SHALL display an error message that:
- States the sync branch name
- Explains that the user is currently on that branch
- Suggests checking out a different branch or using a dedicated sync branch like "beads-sync"

### FR-005: Reuse Existing Utilities

WHEN implementing validation
THE SYSTEM SHALL reuse:
- `syncbranch.ValidateSyncBranchName()` for static validation
- `syncbranch.IsSyncBranchSameAsCurrent()` for dynamic validation

## Non-Functional Requirements

### NFR-001: Defense in Depth

THE SYSTEM SHALL validate at both config-time AND runtime to catch:
- Direct `bd config set` usage
- Manual edits to config.yaml
- Branch switches after configuration
- Edge cases with detached HEAD

### NFR-002: No New Dependencies

THE SYSTEM SHALL implement fixes using only existing internal packages.

### NFR-003: Backward Compatibility

THE SYSTEM SHALL maintain existing behavior for valid sync-branch configurations.
