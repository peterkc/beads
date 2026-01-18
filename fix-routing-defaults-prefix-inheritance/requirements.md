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

### FR-004: Prefix Inheritance on Routing

WHEN auto-routing creates a new beads directory at the target location
THE SYSTEM SHALL copy `issue_prefix` from the source database to the target database.

**Rationale**: Issues created in routed repos should use the source's prefix.

### FR-005: Prefix Inheritance with var/ Layout

WHEN the target beads directory uses the var/ layout (`var/beads.db`)
THE SYSTEM SHALL correctly set `issue_prefix` in the target's config table.

**Rationale**: Fix the current bug where prefix isn't set with new layout.

## Non-Functional Requirements

### NFR-001: Backward Compatibility

THE SYSTEM SHALL maintain backward compatibility with existing `bd init --contributor` workflows.

### NFR-002: Test Coverage

THE SYSTEM SHALL include tests verifying:
- Default routing mode is empty/disabled
- Prefix inheritance works with var/ layout
- Existing contributor workflow unchanged
