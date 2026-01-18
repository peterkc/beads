# Requirements

## Functional Requirements

### FR-001: Default Routing Mode

WHEN a user runs `bd init` without `--contributor` flag
THE SYSTEM SHALL NOT enable auto-routing by default.

**Rationale**: Auto-routing should be opt-in, not opt-out.

### FR-002: Explicit Routing Opt-In

WHEN a user runs `bd init --contributor`
THE SYSTEM SHALL set `routing.mode=auto` in the database config.

**Rationale**: Preserve existing contributor workflow.

### FR-003: Explicit Config Override

WHEN a user runs `bd config set routing.mode auto`
THE SYSTEM SHALL enable auto-routing for subsequent `bd create` commands.

**Rationale**: Allow users to opt-in manually.

## Non-Functional Requirements

### NFR-001: Backward Compatibility

THE SYSTEM SHALL maintain backward compatibility with existing `bd init --contributor` workflows.

### NFR-002: Test Coverage

THE SYSTEM SHALL include tests verifying:
- Default routing mode is empty/disabled
- Existing contributor workflow unchanged

## Deferred Requirements

The following are deferred to PR #1153 (var/ layout):

- **FR-004**: Prefix inheritance with var/ layout
- The var/ layout feature isn't released yet, so prefix inheritance bugs
  with that layout don't affect current users
