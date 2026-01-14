# Design: Fix Path Resolution Bugs

## Architecture Overview

Extract the existing `canonicalizeIfRelative()` helper to `utils/path.go` and use it consistently in both bug locations.

```
┌─────────────────────────────────────────────────────────────────┐
│                    utils/path.go                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  CanonicalizeIfRelative(path) → absolute path           │   │
│  │  (extracted from cmd/bd/autoflush.go)                   │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                    │                           │
         ┌──────────┴──────────┐     ┌──────────┴──────────┐
         │  multirepo_export   │     │    worktree_cmd     │
         │  (Bug 1: oss-lbp)   │     │  (Bug 2: GH#1098)   │
         └─────────────────────┘     └─────────────────────┘
```

## Key Decisions

### Decision 1: Helper Location

**Context**: `canonicalizeIfRelative()` exists in `cmd/bd/autoflush.go` but is needed elsewhere.

**Options Considered**:

1. **Extract to utils/path.go** — Alongside existing `CanonicalizePath()`
   - Pro: Logical grouping, single import
   - Pro: `CanonicalizePath()` already exists there

2. **Duplicate inline** — Copy pattern to each location
   - Pro: No refactoring of existing code
   - Con: Violates DRY, drift risk

3. **Leave in autoflush, import** — Keep in cmd/bd
   - Con: Circular import potential
   - Con: Wrong abstraction layer

**Decision**: Extract to `utils/path.go` as `CanonicalizeIfRelative()` (exported).

**Rationale**: Groups all path utilities together, follows existing pattern.

### Decision 2: Bug 1 Fix - Base Directory

**Context**: Relative paths in `repos.additional` need a deterministic base.

**Options Considered**:

1. **Config file directory** (`.beads/`) — Where config.yaml lives
   - Con: `oss/` would become `.beads/oss/` (wrong!)

2. **Repo root** — Parent of `.beads/`
   - Pro: Matches user mental model (`oss/` means `{repo}/oss/`)
   - Pro: Consistent with how `external_projects` paths are documented

3. **Database directory** — Where `beads.db` lives
   - Same as option 1 (`.beads/`)

**Decision**: Use **repo root** (parent of config file directory).

**Rationale**: Users configure `repos.additional: ["oss/"]` meaning `{repo}/oss/`, not `.beads/oss/`.
This is consistent with how `external_projects` paths like `../beads` are documented (sibling of repo).

### Decision 3: Bug 2 Fix - Ensure Absolute

**Context**: `filepath.Rel()` produces incorrect results with mixed paths.

**Decision**: Canonicalize `mainBeadsDir` before calling `filepath.Rel()`.

**Rationale**: Minimal change, matches existing pattern in codebase.

## Component Design

### New Export: `utils.CanonicalizeIfRelative()`

```go
// CanonicalizeIfRelative ensures path is absolute for filepath.Rel() compatibility.
// If path is already absolute, returns it unchanged.
// If path is relative, converts to absolute using CanonicalizePath().
// If path is empty, returns empty string.
//
// This function guards against code paths that might have relative paths
// where absolute paths are expected. See GH#959 for root cause analysis.
func CanonicalizeIfRelative(path string) string {
    if path != "" && !filepath.IsAbs(path) {
        return CanonicalizePath(path)
    }
    return path
}
```

### Bug 1 Fix: `multirepo_export.go`

**Current** (line 120):
```go
absRepoPath, err := filepath.Abs(expandedPath)  // Resolves from CWD ❌
```

**Critical insight**: Config file is at `.beads/config.yaml`, but paths like `oss/` are
**repo-relative** (meaning `{repo}/oss/`), NOT `.beads/`-relative.

**Fixed**:
```go
var absRepoPath string
if filepath.IsAbs(expandedPath) {
    absRepoPath = expandedPath
} else {
    // Resolve relative to repo root (parent of .beads/)
    // Config is at .beads/config.yaml, so go up twice
    configFile := config.ConfigFileUsed()
    if configFile != "" {
        repoRoot := filepath.Dir(filepath.Dir(configFile))  // .beads/config.yaml -> repo/
        absRepoPath = filepath.Join(repoRoot, expandedPath)
    } else {
        // Fallback: dbPath is .beads/beads.db, go up one level to repo root
        repoRoot := filepath.Dir(filepath.Dir(s.dbPath))
        absRepoPath = filepath.Join(repoRoot, expandedPath)
    }
}
```

### Bug 2 Fix: `worktree_cmd.go`

**Current** (line 205):
```go
relPath, err := filepath.Rel(worktreeBeadsDir, mainBeadsDir)  // mainBeadsDir may be relative ❌
```

**Fixed**:
```go
// Ensure both paths are absolute for correct filepath.Rel() computation
absMainBeadsDir := utils.CanonicalizeIfRelative(mainBeadsDir)
relPath, err := filepath.Rel(worktreeBeadsDir, absMainBeadsDir)
```

### Update: `autoflush.go`

Replace local function with utils import:
```go
// Before (local)
func canonicalizeIfRelative(path string) string { ... }

// After (use utils)
import "github.com/steveyegge/beads/internal/utils"
// Replace calls: canonicalizeIfRelative(x) → utils.CanonicalizeIfRelative(x)
```

## Error Handling

| Error Condition              | Handling Strategy                     |
| ---------------------------- | ------------------------------------- |
| Config file path empty       | Fall back to dbPath directory         |
| Both config and dbPath empty | Return original path (let caller fail)|
| Directory creation fails     | Return error with path context        |

## Risks and Mitigations

| ID    | Risk                          | L   | I   | Mitigation                      | Rollback        |
| ----- | ----------------------------- | --- | --- | ------------------------------- | --------------- |
| R-001 | Config path empty in daemon   | Low | Med | dbPath fallback + test coverage | git revert      |
| R-002 | Import cycle                  | Low | High| utils has no internal deps      | Revert extract  |
| R-003 | Behavior change in autoflush  | Low | Med | Same logic, just moved          | git revert      |

## Gotchas

- `config.ConfigFileUsed()` returns empty string if no config loaded
- `beads.FindBeadsDir()` may return relative path in some contexts
- `filepath.Rel()` requires both paths to be the same type (both abs or both rel)

## Testing Strategy

- **Unit (helper)**: Test `CanonicalizeIfRelative()` with abs, rel, empty inputs
- **Unit (bug 1)**: Mock config path, verify export location
- **Unit (bug 2)**: Create worktree at various depths, verify redirect content
- **Integration**: Run `bd sync` and `bd worktree create` from different CWDs
