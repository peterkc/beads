# Requirements

## Functional Requirements

### FR-001: Dead Code Removal

THE SYSTEM SHALL NOT contain the `restoreBeadsDirFromBranch` function after this change.

**Rationale**: The function has no call sites and serves no purpose.

### FR-002: Config Preservation

WHEN the user runs `bd sync` with sync-branch configured
AND config.yaml has uncommitted changes
THE SYSTEM SHALL preserve those uncommitted changes after sync completes.

**Rationale**: User config changes should not be silently overwritten by sync operations.

### FR-003: Backward Compatibility

WHEN the user runs `bd sync` in any mode (sync-branch, normal, from-main)
THE SYSTEM SHALL produce identical behavior to v0.47.0+ after this change.

**Rationale**: This is a cleanup, not a behavior change.

## Non-Functional Requirements

### NFR-001: Test Coverage

THE SYSTEM SHALL include a regression test that fails if config.yaml restoration
is reintroduced.

**Validation**: `go test -v -run TestConfigPreserved`

### NFR-002: Code Hygiene

THE SYSTEM SHALL compile without warnings after dead code removal.

**Validation**: `go build ./cmd/bd/...`
