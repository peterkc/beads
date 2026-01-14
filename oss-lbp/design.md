# Design: Fix Multi-repo Export Path Resolution

## Architecture Overview

The fix modifies path resolution in `exportToRepo()` to use config file directory as the base for relative paths, instead of the process's current working directory.

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  repos.additional│────▶│  resolveRepoPath │────▶│  .beads/issues  │
│  ["oss/"]       │     │  (config-relative)│     │  .jsonl         │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                         config.ConfigFileUsed()
                         ────────────────────
                         /path/to/repo/.beads/config.yaml
```

## Key Decisions

### Decision 1: Base Directory for Relative Paths

**Context**: Relative paths in `repos.additional` need a deterministic base directory.

**Options Considered**:

1. **Config file directory** — Where `config.yaml` lives
   - Pro: Intuitive (paths relative to where they're defined)
   - Pro: Consistent with Viper/config file conventions
   - Con: Requires config path to be available

2. **Database path directory** — Where `beads.db` lives
   - Pro: Always available via `dbPath`
   - Con: Less intuitive for users editing config

3. **Git repository root** — Where `.git` lives
   - Pro: Consistent with git conventions
   - Con: Requires git discovery, more complex

**Decision**: Config file directory with dbPath fallback.

**Rationale**: Users edit config.yaml, so paths should be relative to that file. The dbPath fallback handles edge cases where config location is unknown.

### Decision 2: Implementation Pattern

**Context**: Need to resolve paths without breaking existing behavior.

**Options Considered**:

1. **Inline fix** — Modify `exportToRepo()` directly
2. **Helper function** — Create `resolveRepoPath()` helper
3. **Reuse existing** — Use `utils.CanonicalizePath()`

**Decision**: Inline fix using existing `config.ConfigFileUsed()` API.

**Rationale**:
- The fix is localized to one function
- `CanonicalizePath()` does more than needed (symlink resolution)
- Helper function adds abstraction for a single use case

## Component Design

### Modified Function: `exportToRepo()`

**Current** (problematic):
```go
// Get absolute path
absRepoPath, err := filepath.Abs(expandedPath)  // Resolves from CWD ❌
```

**Fixed**:
```go
// Resolve relative paths from config directory, not CWD
var absRepoPath string
if filepath.IsAbs(expandedPath) {
    absRepoPath = expandedPath
} else {
    // Get config file directory as base for relative paths
    configFile := config.ConfigFileUsed()
    if configFile != "" {
        configDir := filepath.Dir(configFile)
        absRepoPath = filepath.Join(configDir, expandedPath)
    } else {
        // Fallback to dbPath directory
        absRepoPath = filepath.Join(filepath.Dir(s.dbPath), expandedPath)
    }
}
```

**Dependencies**:
- `github.com/steveyegge/beads/internal/config` (existing)

## Error Handling

| Error Condition              | Handling Strategy                     |
| ---------------------------- | ------------------------------------- |
| Config file path empty       | Fall back to dbPath directory         |
| Directory creation fails     | Return error with path context        |
| Invalid path characters      | Let OS report error on MkdirAll       |

## Risks and Mitigations

| ID    | Risk                          | Likelihood | Impact | Mitigation                      | Rollback        |
| ----- | ----------------------------- | ---------- | ------ | ------------------------------- | --------------- |
| R-001 | Config path empty in daemon   | Low        | Medium | dbPath fallback + test coverage | git revert      |
| R-002 | Path separator issues (Win)   | Low        | Low    | Use filepath.Join throughout    | N/A (Go stdlib) |

## Gotchas

- `config.ConfigFileUsed()` returns empty string if no config file was loaded
- The dbPath in `SQLiteStorage` struct already stores the absolute path
- Daemon runs from `.beads/` directory, making CWD unreliable

## Testing Strategy

- **Unit**: Mock config file path, verify resolution logic
- **Integration**: Create temp config with relative paths, verify export location
- **Manual**: Run `bd sync --daemon` and verify no spurious directories
