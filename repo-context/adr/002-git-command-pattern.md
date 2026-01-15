# ADR-002: Use cmd.Dir Pattern for Git Commands

**Status**: Proposed

**Date**: 2025-01-14

## Context

When executing git commands that need to run in a specific directory, there are
two common patterns in Go:

```go
// Pattern A: -C flag
cmd := exec.Command("git", "-C", "/path/to/repo", "status")

// Pattern B: cmd.Dir
cmd := exec.Command("git", "status")
cmd.Dir = "/path/to/repo"
```

The beads codebase currently uses both patterns inconsistently:
- `fork_protection.go` uses `-C` flag
- `routing.go` uses `cmd.Dir`
- Most files use neither (rely on CWD)

We need to standardize on one approach for the new `RepoContext.GitCmd()` method.

## Decision Drivers

- **Universality**: Must work with all git commands
- **Go idiom**: Follow standard Go patterns
- **Composability**: Easy to add other exec.Cmd properties
- **Debuggability**: Clear in logs/errors which directory was used

## Considered Options

### Option 1: cmd.Dir Pattern

```go
func (rc *RepoContext) GitCmd(ctx context.Context, args ...string) *exec.Cmd {
    cmd := exec.CommandContext(ctx, "git", args...)
    cmd.Dir = rc.RepoRoot
    return cmd
}
```

- **Good, because** Go-idiomatic (standard library pattern)
- **Good, because** works with ALL git commands (even those without `-C` support)
- **Good, because** composable with other cmd properties (Env, Stdin, Stdout)
- **Good, because** callers can inspect `cmd.Dir` for debugging
- **Bad, because** directory not visible in process list (ps aux)

### Option 2: -C Flag Pattern

```go
func (rc *RepoContext) GitCmd(ctx context.Context, args ...string) *exec.Cmd {
    fullArgs := append([]string{"-C", rc.RepoRoot}, args...)
    return exec.CommandContext(ctx, "git", fullArgs...)
}
```

- **Good, because** visible in process list
- **Good, because** explicit in command string
- **Bad, because** some git commands may not support `-C`
- **Bad, because** harder to modify args after creation
- **Bad, because** less Go-idiomatic

### Option 3: Wrapper Script

Create a shell script that cd's and runs git:

```bash
#!/bin/bash
cd "$1" && shift && git "$@"
```

- **Good, because** visible directory in process
- **Bad, because** external dependency
- **Bad, because** performance overhead
- **Bad, because** cross-platform issues

## Decision

**Chosen option**: Option 1 (cmd.Dir Pattern), because it's the Go-idiomatic
approach, works universally with all git commands, and allows composition with
other exec.Cmd properties.

```
PATTERN:
┌─────────────────────────────────────────────────────────┐
│  rc.GitCmd(ctx, "pull", "origin", "main")              │
│                    │                                    │
│                    ▼                                    │
│  ┌───────────────────────────────────────────────────┐ │
│  │  cmd := exec.CommandContext(ctx, "git", args...)  │ │
│  │  cmd.Dir = rc.RepoRoot  ← sets working directory  │ │
│  │  return cmd                                        │ │
│  └───────────────────────────────────────────────────┘ │
│                    │                                    │
│                    ▼                                    │
│  Equivalent to: cd /path/to/repo && git pull origin    │
└─────────────────────────────────────────────────────────┘
```

## Consequences

### Positive

- Consistent pattern across entire codebase
- No edge cases with git commands that don't support `-C`
- Easy to add environment variables, stdin, etc.
- Caller can inspect/modify cmd before running

### Negative

- Directory not visible in `ps aux` output
- Existing `-C` usages should be migrated for consistency

### Neutral

- Either pattern produces same git behavior
- Logging should include `cmd.Dir` for debuggability

## Implementation Notes

- **v1 (2025-01-14)**: Proposed as part of RepoContext design

## Related

- ADR-001: Centralize Repository Context Resolution
- Spec: `specs/beads-repo-context/design.md`
