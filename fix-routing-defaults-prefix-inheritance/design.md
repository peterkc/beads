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
        Works!                  +---------------------------+
                                | ensureBeadsDirForPath()   |
                                | Creates ~/.beads-planning |
                                | BUT prefix not inherited! |
                                +---------------------------+
                                            |
                                            v
                                "issue_prefix config is missing"
```

## Root Causes

### Bug 1: Default Auto-Routing

Location: `internal/config/config.go:103`

```go
// CURRENT (problematic)
v.SetDefault("routing.mode", "auto")

// PROPOSED
v.SetDefault("routing.mode", "")  // Empty = disabled
```

### Bug 2: Prefix Inheritance Failure

Location: `cmd/bd/create.go:954-1002`

```
ensureBeadsDirForPath() creates:

~/.beads-planning/
    .beads/
        var/
            beads.db  <-- Created with sqlite.New()
        issues.jsonl

Problem: sqlite.New() creates DB with default config.
         Prefix is set AFTER via tempStore.SetConfig().
         But the path calculation uses old layout assumption.
```

The issue is that `ensureBeadsDirForPath()` calculates:
```go
dbPath := filepath.Join(beadsDir, "beads.db")  // OLD layout
```

But with var/ layout, the actual path is:
```go
dbPath := filepath.Join(beadsDir, "var", "beads.db")  // NEW layout
```

## Proposed Fix

### Fix 1: Change Default

```go
// internal/config/config.go

// Routing configuration defaults
v.SetDefault("routing.mode", "")  // Empty = no auto-routing
v.SetDefault("routing.default", ".")
v.SetDefault("routing.maintainer", ".")
v.SetDefault("routing.contributor", "~/.beads-planning")
```

### Fix 2: Fix Prefix Inheritance

```go
// cmd/bd/create.go - ensureBeadsDirForPath()

func ensureBeadsDirForPath(ctx context.Context, targetPath string, sourceStore storage.Storage) error {
    beadsDir := filepath.Join(targetPath, ".beads")

    // Use factory to create store (respects var/ layout)
    targetStore, err := factory.NewFromConfig(ctx, beadsDir)
    if err != nil {
        return fmt.Errorf("failed to initialize target database: %w", err)
    }
    defer targetStore.Close()

    // Inherit prefix from source
    if sourceStore != nil {
        sourcePrefix, err := sourceStore.GetConfig(ctx, "issue_prefix")
        if err == nil && sourcePrefix != "" {
            if err := targetStore.SetConfig(ctx, "issue_prefix", sourcePrefix); err != nil {
                return fmt.Errorf("failed to set prefix in target store: %w", err)
            }
        }
    }

    return nil
}
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

## Key Decisions

### KD-001: Empty String vs "disabled" for Default

**Decision**: Use empty string `""` for routing.mode default.

**Rationale**:
- Empty string is falsy in Go (`routingMode == ""`)
- Matches pattern used elsewhere in config
- "disabled" would require string comparison

### KD-002: Use Factory for Store Creation

**Decision**: Use `factory.NewFromConfig()` instead of `sqlite.New()` directly.

**Rationale**:
- Factory respects backend config (sqlite vs dolt)
- Factory handles var/ layout automatically
- Consistent with how main store is created

## Test Strategy

### Test 1: Default Routing Mode

```go
func TestRoutingModeDefaultIsEmpty(t *testing.T) {
    // Fresh config initialization
    config.Initialize()

    mode := config.GetString("routing.mode")
    assert.Equal(t, "", mode)
}
```

### Test 2: Prefix Inheritance

```go
func TestPrefixInheritanceOnRouting(t *testing.T) {
    // Create source repo with prefix
    // Enable routing to temp target
    // Run bd create
    // Verify target has same prefix
}
```
