# Design

## Current Behavior (Bug)

```
User runs: bd init --prefix myapp
           bd create "Test issue"

                    bd create
                        |
                        v
            +---------------------+
            | Check routing.mode  |
            | (viper default)     |
            +---------------------+
                        |
                        v
            routing.mode = "auto" (DEFAULT!)
                        |
                        v
            +---------------------+
            | DetectUserRole()    |
            | in routing.go       |
            +---------------------+
                        |
            +-----------+-----------+
            |                       |
            v                       v
        SSH URL?                HTTPS/file URL?
        (git@...)               (https://... or /path)
            |                       |
            v                       v
        Maintainer              Contributor  <-- Most users hit this!
            |                       |
            v                       v
        Local .beads/           ~/.beads-planning/
            |                       |
            v                       v
        Works!                  "issue_prefix config is missing"
```

## Root Cause

Location: `internal/config/config.go:103`

```go
// CURRENT (problematic)
v.SetDefault("routing.mode", "auto")

// PROPOSED
v.SetDefault("routing.mode", "")  // Empty = disabled
```

The issue is that Viper's `SetDefault` applies at runtime even if no config
file exists. Users who run `bd init` without `--contributor` still get
`routing.mode=auto` from the default, triggering contributor detection.

## Proposed Fix

```go
// internal/config/config.go

// Routing configuration defaults
v.SetDefault("routing.mode", "")  // Empty = no auto-routing
v.SetDefault("routing.default", ".")
v.SetDefault("routing.maintainer", ".")
v.SetDefault("routing.contributor", "~/.beads-planning")
```

## Fixed Behavior

```
User runs: bd init --prefix myapp
           bd create "Test issue"

                    bd create
                        |
                        v
            +---------------------+
            | Check routing.mode  |
            | (viper default)     |
            +---------------------+
                        |
                        v
            routing.mode = "" (EMPTY!)
                        |
                        v
            +---------------------+
            | No auto-routing     |
            | Use current repo    |
            +---------------------+
                        |
                        v
                Local .beads/
                        |
                        v
                Issue created: myapp-xxx
```

## Key Decision

### KD-001: Empty String vs "disabled" for Default

**Decision**: Use empty string `""` for routing.mode default.

**Rationale**:
- Empty string is falsy in Go (`routingMode == ""`)
- Matches pattern used elsewhere in config
- "disabled" would require string comparison

## Test Strategy

```go
func TestRoutingModeDefaultIsEmpty(t *testing.T) {
    // Fresh config initialization
    config.Initialize()

    mode := config.GetString("routing.mode")
    assert.Equal(t, "", mode)
}
```

## Deferred

Prefix inheritance with var/ layout is deferred to PR #1153.
That layout isn't released yet, so doesn't affect current users.
