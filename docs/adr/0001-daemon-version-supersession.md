# ADR 0001: Daemon Version Supersession

## Status

Accepted

## Context

During the v0 → v1 migration period, users may have both `bd` (v0) and `bdx` (v1) binaries installed. Both use the same daemon architecture with Unix socket IPC (`bd.sock`). We need to define how daemons from different major versions interact.

Key questions:
1. What happens when `bdx` starts and a `bd` daemon is running?
2. Can both daemons coexist?
3. How do we ensure users don't accidentally downgrade their daemon?

### Existing Mechanism

The current codebase already has version-aware daemon startup (`daemon_start.go:62-84`):

```go
// Version mismatch - auto-stop old daemon
if healthErr == nil && !health.Compatible {
    fmt.Fprintf(os.Stderr, "Warning: daemon version mismatch...")
    stopDaemon(pidFile)
}
```

Version compatibility uses semver (`server_routing_validation_diagnostics.go`):
- Same major version, daemon newer → OK (backward compatible)
- Same major version, client newer → Daemon auto-restarts
- Different major version → Error with upgrade instructions

### Alternatives Considered

| Approach | Description | Pros | Cons |
|----------|-------------|------|------|
| **A. Version ordering** | Higher version always supersedes | Simple, natural flow | No coexistence |
| **B. Separate sockets** | `bd.sock` vs `bdx.sock` | True coexistence | Resource doubling, confusion |
| **C. Binary-aware** | Check binary name, refuse downgrade | Graceful coexistence | More code, edge cases |

## Decision

**Use version ordering (Option A) with major version check.**

1. `bdx` (v1.x) automatically stops any running `bd` (v0.x) daemon
2. `bd` (v0.x) refuses to start if `bdx` (v1.x) daemon is running
3. Both use the same socket path (`bd.sock`)
4. Major version mismatch triggers clear error messages

### Implementation

The existing mechanism handles this with one adjustment:

**For bdx (v1.0.0+):**
- Version check detects v0.x daemon → stops it → starts v1 daemon
- Users see: `Warning: daemon version mismatch (daemon: 0.x, client: 1.0). Stopping old daemon...`

**For bd (v0.x) if run after bdx:**
- Version check detects v1.x daemon → refuses with clear error
- Users see: `Error: incompatible major versions: client 0.x, daemon 1.x. Client is older; upgrade to bdx.`

No code changes needed — the existing semver logic handles this correctly.

## Consequences

### Positive

- Zero additional code complexity
- Clear migration path: once on bdx, stay on bdx
- No resource duplication from parallel daemons
- Existing version check infrastructure handles edge cases

### Negative

- No true coexistence (can't run both v0 and v1 daemons simultaneously)
- Users who accidentally run `bd` after `bdx` get an error (intentional, but may confuse)

### Neutral

- Migration is one-way (v0 → v1), which aligns with release strategy
- Documentation should clearly explain the supersession behavior

## Notes

- This ADR covers daemon interaction only, not CLI coexistence (both binaries can exist)
- The socket path convention (`bd.sock`) is intentionally shared to leverage existing discovery
- Future major versions (v2+) would follow the same pattern
