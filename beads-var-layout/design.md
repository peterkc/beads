# Design: .beads/var/ Layout Migration

## Architecture Overview

Centralized path resolution module that abstracts volatile file locations, enabling transparent support for both legacy (flat) and new (var/) layouts.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        BEADS VAR/ MIGRATION IMPACT                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    NEW: internal/beads/paths.go                      │   │
│  │  Centralized volatile file path resolution (extends RepoContext)    │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │ VolatileFiles[]     VarPath()      VarDir()    IsVarLayout() │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         CONSUMERS (6 files)                          │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                      │   │
│  │  cmd/bd/daemon_config.go          internal/rpc/socket_path.go       │   │
│  │  ├─ getPIDFilePath()              ├─ ShortSocketPath()              │   │
│  │  ├─ getLogFilePath()              └─ bd.sock location               │   │
│  │  └─ daemon.lock location                                            │   │
│  │                                                                      │   │
│  │  cmd/bd/sync_merge.go             cmd/bd/daemon_sync_state.go       │   │
│  │  ├─ loadBaseState()               └─ sync-state.json location       │   │
│  │  └─ sync_base.jsonl location                                        │   │
│  │                                                                      │   │
│  │  internal/configfile/configfile.go                                  │   │
│  │  └─ DatabasePath() → uses VarPath() for beads.db                    │   │
│  │                                                                      │   │
│  │  internal/lockfile/lock.go                                          │   │
│  │  └─ daemon.lock, daemon.pid paths                                   │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    DOCTOR SUBSYSTEM (3 files)                        │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                      │   │
│  │  cmd/bd/doctor/gitignore.go       cmd/bd/doctor/migration.go        │   │
│  │  ├─ GitignoreTemplate (UPDATE)    ├─ needsVarMigration() (NEW)      │   │
│  │  ├─ requiredPatterns (UPDATE)     └─ DetectPendingMigrations()      │   │
│  │  └─ FixGitignore()                                                  │   │
│  │                                                                      │   │
│  │  cmd/bd/migrate_var.go (NEW)                                        │   │
│  │  ├─ runVarMigration()                                               │   │
│  │  ├─ runVarCleanup()                                                 │   │
│  │  └─ rollbackMigration()                                             │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      UNCHANGED (root-level)                          │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  redirect         - Must stay at root for worktree discovery        │   │
│  │  issues.jsonl     - Git-tracked data                                │   │
│  │  interactions.jsonl - Git-tracked audit log                         │   │
│  │  metadata.json    - Configuration                                   │   │
│  │  config.yaml      - User configuration                              │   │
│  │  .gitignore       - Stays at root (content changes)                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Decisions

### Decision 1: Centralized paths.go Module

**Context**: Volatile file paths are currently scattered across 6+ files with hardcoded joins.

**Options Considered**:

1. **Centralized module** — Single `paths.go` with `VarPath()` helper
2. **Per-file detection** — Each file checks layout independently
3. **Config-based** — Store layout preference in metadata.json

**Decision**: Centralized module (Option 1)

**Rationale**:
- Follows existing `RepoContext` pattern in `docs/REPO_CONTEXT.md`
- Single source of truth for path resolution
- Easy to maintain and test
- No config migration needed

### Decision 2: redirect Stays at Root

**Context**: The `redirect` file enables worktree discovery by pointing to main repo's `.beads/`.

**Options Considered**:

1. **Move to var/** — Consistent with other volatile files
2. **Keep at root** — Preserve worktree discovery semantics

**Decision**: Keep at root (Option 2)

**Rationale**:
- `FollowRedirect()` in `internal/beads/beads.go` reads redirect before beads context is established
- Moving would require redirect-within-redirect logic
- Redirect is already gitignored, no benefit to moving

### Decision 3: Legacy Patterns in Gitignore

**Context**: After migration, should gitignore only contain `var/`?

**Options Considered**:

1. **var/ only** — Clean but breaks unmigrated users
2. **var/ + legacy** — Redundant but backward compatible
3. **Conditional generation** — Complex logic per layout

**Decision**: var/ + legacy patterns (Option 2)

**Rationale**:
- Mixed-layout clones (different team members) must work during transition
- Legacy patterns are harmless in new layout (files don't exist at root)
- Simplifies migration — no gitignore changes required

### Decision 4: Migration is Optional

**Context**: Should existing users be forced to migrate?

**Options Considered**:

1. **Auto-migrate on upgrade** — Clean but risky
2. **Doctor warning** — Visible but optional
3. **Doctor info** — Minimally intrusive, truly optional

**Decision**: Doctor info, Priority 3 (Option 3)

**Rationale**:
- Zero regression requirement means no forced changes
- Users who want var/ benefits can opt in
- Gradual adoption allows ecosystem to catch up

### Decision 5: Read-Both Coexistence Pattern

**Context**: Can var/ and legacy layouts coexist for edge case safety?

**Options Considered**:

1. **Binary check** — If var/ exists use it, else use root (simple but fragile)
2. **Read-both, write-one** — Check both locations on read, write to layout preference
3. **Full bidirectional** — Read/write to both (overcomplicated)

**Decision**: Read-both, write-one (Option 2)

**Rationale**:
- Handles interrupted migrations gracefully
- External tools that haven't updated work during transition
- Doctor can find files in unexpected locations
- Two stat() calls worst-case is negligible (NFR-001 allows 1ms)
- Prevents "file not found" errors during migration edge cases

**Behavior**:
| Operation | var/ Layout | Legacy Layout | During Migration |
|-----------|------------|---------------|------------------|
| Read      | var/ first, then root | root only | Both checked |
| Write     | var/ only | root only | Layout preference |

This makes the system self-healing — files in wrong locations are still found.

## Component Design

### internal/beads/paths.go

**Purpose**: Centralized volatile file path resolution

**Interface**:

```go
package beads

import (
    "os"
    "path/filepath"
)

// VolatileFiles lists all files that should live in var/
var VolatileFiles = []string{
    "beads.db", "beads.db-journal", "beads.db-wal", "beads.db-shm",
    "daemon.lock", "daemon.log", "daemon.pid", "bd.sock",
    "sync_base.jsonl", ".sync.lock", "sync-state.json",
    "beads.base.jsonl", "beads.base.meta.json",
    "beads.left.jsonl", "beads.left.meta.json",
    "beads.right.jsonl", "beads.right.meta.json",
    "last-touched", ".local_version", "export_hashes.db",
}

// VarPath returns the path for a volatile file, using read-both pattern.
// For READS: checks var/ first, falls back to root (handles edge cases).
// For WRITES: uses layout preference (var/ if exists, else root).
func VarPath(beadsDir, filename string) string {
    // Environment override for emergency fallback
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return filepath.Join(beadsDir, filename)
    }

    varPath := filepath.Join(beadsDir, "var", filename)
    rootPath := filepath.Join(beadsDir, filename)

    // Read-both: check var/ first, then root (handles migration edge cases)
    if _, err := os.Stat(varPath); err == nil {
        return varPath
    }
    if _, err := os.Stat(rootPath); err == nil {
        return rootPath
    }

    // New file: use layout preference
    if IsVarLayout(beadsDir) {
        return varPath
    }
    return rootPath
}

// VarPathForWrite returns the path for writing a volatile file.
// Always respects layout preference (no fallback checking).
func VarPathForWrite(beadsDir, filename string) string {
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return filepath.Join(beadsDir, filename)
    }
    if IsVarLayout(beadsDir) {
        return filepath.Join(beadsDir, "var", filename)
    }
    return filepath.Join(beadsDir, filename)
}

// VarDir returns the directory for volatile files.
// Returns var/ if it exists, otherwise beadsDir root.
func VarDir(beadsDir string) string {
    if IsVarLayout(beadsDir) {
        return filepath.Join(beadsDir, "var")
    }
    return beadsDir
}

// IsVarLayout checks if .beads uses the var/ layout.
func IsVarLayout(beadsDir string) bool {
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return false
    }
    varDir := filepath.Join(beadsDir, "var")
    info, err := os.Stat(varDir)
    return err == nil && info.IsDir()
}

// EnsureVarDir creates the var/ directory if it doesn't exist.
func EnsureVarDir(beadsDir string) error {
    varDir := filepath.Join(beadsDir, "var")
    return os.MkdirAll(varDir, 0700)
}

// IsVolatileFile checks if a filename is a volatile file.
func IsVolatileFile(filename string) bool {
    for _, vf := range VolatileFiles {
        if filename == vf {
            return true
        }
    }
    // Also check glob patterns
    if matched, _ := filepath.Match("*.db-*", filename); matched {
        return true
    }
    return false
}
```

**Dependencies**: Standard library only (os, path/filepath)

### cmd/bd/migrate_var.go

**Purpose**: Migration command implementation

**Interface**:

```go
var migrateVarCmd = &cobra.Command{
    Use:   "var",
    Short: "Migrate to var/ layout for volatile files",
    Long: `Migrate .beads/ to use var/ subdirectory for volatile files.

This organizes machine-local files (database, daemon, sync state) into
.beads/var/, separating them from git-tracked files.

The migration copies files to var/ and removes originals. Use --dry-run
to preview changes first. If stray files appear later at root, use
'bd doctor --fix' to move them.`,
    Run: runMigrateVar,
}

func init() {
    migrateCmd.AddCommand(migrateVarCmd)
    migrateVarCmd.Flags().Bool("dry-run", false, "Preview changes without modifying files")
}
```

**Dependencies**: beads package, cobra

### cmd/bd/doctor/migration.go (Addition)

**Purpose**: Detect files in wrong location when var/ layout is active

**Interface**:

```go
// FilesInWrongLocation returns volatile files that exist at root
// when they should be in var/ (var/ layout is active)
func FilesInWrongLocation(beadsDir string) []string {
    if !beads.IsVarLayout(beadsDir) {
        return nil  // Not using var/ layout, no "wrong" location
    }

    wrongLocation := []string{}
    for _, f := range beads.VolatileFiles {
        rootPath := filepath.Join(beadsDir, f)
        varPath := filepath.Join(beadsDir, "var", f)

        // File at root but var/ layout is active
        // (Note: may also exist in var/, which is fine - we read var/ first)
        if fileExists(rootPath) {
            wrongLocation = append(wrongLocation, f)
        }
    }
    return wrongLocation
}

// In DetectPendingMigrations(), add:
if files := FilesInWrongLocation(beadsDir); len(files) > 0 {
    issues = append(issues, MigrationIssue{
        Name:        "files-wrong-location",
        Priority:    2,  // Warning (fixable)
        Description: fmt.Sprintf("%d volatile files at root should be in var/", len(files)),
        Files:       files,
        Fix:         "bd doctor --fix",  // Doctor handles stray cleanup
    })
}
```

**Behavior**:
- Only triggers when var/ layout is active (var/ directory exists)
- Reports files at root that should be in var/
- Doesn't auto-move (explicit migration preserves predictability)
- `bd doctor --fix` offers to run migration with confirmation

## Data Model

No data model changes. File system layout only.

## Error Handling

| Error Condition             | Handling Strategy                             |
| --------------------------- | --------------------------------------------- |
| var/ creation fails         | Return error with permission details          |
| File copy fails             | Preserve original, return error               |
| Already migrated            | Exit 0 with "Already using var/ layout"       |
| Daemon running              | Prompt to stop or use --force                 |
| Disk full                   | Fail fast, preserve originals                 |

## Risks and Mitigations

| ID    | Risk                          | Likelihood | Impact | Mitigation                       | Rollback                          |
| ----- | ----------------------------- | ---------- | ------ | -------------------------------- | --------------------------------- |
| R-001 | External tools hardcode paths | Medium     | Medium | 6-month compatibility window     | Keep legacy patterns in gitignore |
| R-002 | sync_base.jsonl loss          | Low        | Low    | Copy, don't move until cleanup   | Manual file recovery              |
| R-003 | Worktree redirect breaks      | Low        | High   | Keep redirect at root            | N/A (not moved)                   |
| R-004 | Socket path too long          | Low        | Medium | Existing /tmp fallback preserved | N/A (fallback exists)             |

## Gotchas

- `redirect` file MUST stay at root — read before beads context exists
- Socket path length limit (103 chars) — existing fallback handles this
- SQLite creates sibling files (`.db-journal`, `.db-wal`) — they follow db location automatically
- `export_hashes.db` is separate from main database — must also move to var/

## Workflow Paths

### Path A: New User (Fresh Install)

```
bd init
    │
    ▼
Creates .beads/var/ (new default)
    │
    ▼
.gitignore contains just "var/"
    │
    ▼
All volatile files in var/ from start
    │
    ▼
bd list, bd create, bd sync ── all work ✓


bd init --legacy (explicit opt-out)
    │
    ▼
Creates .beads/ (flat layout)
    │
    ▼
.gitignore contains all patterns
    │
    ▼
All commands work ✓
```

### Path B: Existing User Migration

```
Existing .beads/ (legacy layout)
    │
    ▼
Upgrade bd to version with var/ support
    │
    ▼
bd doctor
    │
    ▼
"ℹ Optional: var/ layout available"
    │
    ├── User ignores ──► No change, legacy continues working
    │
    └── User opts in
            │
            ▼
        bd daemon stop (if running)
            │
            ▼
        bd migrate var
            │
            ▼
        .beads/var/ created
        15 files moved
            │
            ▼
        bd daemon start
            │
            ▼
        Verify: bd list, bd show, bd sync
```

### Path C: Stray File Cleanup

```
.beads/var/ exists (migrated)
    │
    ▼
External tool writes to .beads/beads.db (root)
    │
    ▼
bd doctor
    │
    ▼
"⚠ 1 file in wrong location"
    │
    ├── bd doctor --fix ──► File moved to var/ ✓
    │
    └── User ignores ──► VarPath() reads from root (fallback) ✓
```

### Path D: Read-Both Fallback

```
VarPath("beads.db") called
    │
    ▼
BD_LEGACY_LAYOUT=1 set? ──yes──► Return .beads/beads.db
    │
    no
    │
    ▼
.beads/var/beads.db exists? ──yes──► Return .beads/var/beads.db
    │
    no
    │
    ▼
.beads/beads.db exists? ──yes──► Return .beads/beads.db (fallback)
    │
    no
    │
    ▼
IsVarLayout()? ──yes──► Return .beads/var/beads.db (new file)
    │
    no
    │
    ▼
Return .beads/beads.db (new file, legacy)
```

## Test Matrix

### Layout Compatibility Matrix

| Test | Legacy Layout | var/ Layout | Expected |
|------|---------------|-------------|----------|
| `bd init` | N/A | Creates .beads/var/ | var/ default |
| `bd init --legacy` | Creates .beads/ | N/A | Legacy on request |
| `bd list` | Works | Works | Same output |
| `bd create` | Works | Works | Issue created |
| `bd show` | Works | Works | Issue displayed |
| `bd sync` | Works | Works | Sync completes |
| `bd daemon start` | Works | Works | Daemon starts |
| `bd daemon stop` | Works | Works | Daemon stops |
| `bd doctor` | Shows var/ option | No var/ warning | Layout-specific |
| `bd config` | Works | Works | Same behavior |

### Migration Test Matrix

| Scenario | Input State | Command | Expected Output |
|----------|-------------|---------|-----------------|
| New user default | No .beads/ | `bd init` | var/ layout created |
| New user legacy | No .beads/ | `bd init --legacy` | Legacy layout created |
| Fresh migrate | Legacy, no var/ | `bd migrate var` | var/ created, files moved |
| Dry run | Legacy, no var/ | `bd migrate var --dry-run` | Preview only, no changes |
| Already migrated | var/ exists | `bd migrate var` | Exit 0, "Already migrated" |
| Daemon running | Legacy + daemon | `bd migrate var` | Error: stop daemon first |
| Stray files | var/ + root files | `bd doctor --fix` | Files moved to var/ |
| Mixed state | var/ + some root | `bd list` | Works (read-both) |

### VarPath() Unit Test Matrix

| beadsDir State | BD_LEGACY_LAYOUT | File Location | VarPath() Returns |
|----------------|------------------|---------------|-------------------|
| No var/ | unset | root | .beads/file |
| No var/ | "1" | root | .beads/file |
| Has var/ | unset | var/ only | .beads/var/file |
| Has var/ | unset | root only | .beads/file (fallback) |
| Has var/ | unset | both | .beads/var/file (priority) |
| Has var/ | unset | neither | .beads/var/file (new) |
| Has var/ | "1" | var/ only | .beads/file (override) |

### Doctor Detection Matrix

| Layout | Files at Root | Files in var/ | Doctor Output | --fix Action |
|--------|---------------|---------------|---------------|--------------|
| Legacy | Yes | N/A | "ℹ var/ available" | N/A |
| var/ | No | Yes | ✓ Clean | None |
| var/ | Yes | Yes | "⚠ wrong location" | Move to var/ |
| var/ | Yes | No | "⚠ wrong location" | Move to var/ |

### Sync-Branch Compatibility Matrix

| Component | Tracked in Git | var/ Migration Impact |
|-----------|----------------|----------------------|
| issues.jsonl | ✓ beads-sync branch | Stays at root |
| interactions.jsonl | ✓ beads-sync branch | Stays at root |
| metadata.json | ✓ beads-sync branch | Stays at root |
| sync_base.jsonl | ✗ gitignored | Moves to var/ |
| beads.db | ✗ gitignored | Moves to var/ |
| Worktree checkout | .git/beads-worktrees/ | Unaffected |

### Regression Test Commands

```bash
# Must pass before AND after migration
go test ./...
bd list --status=open
bd create --title="Test" --type=task && bd close $(bd list --format=ids | tail -1)
bd sync --status
bd daemon start && sleep 1 && bd daemon stop
bd doctor
```

## Testing Strategy

- **Unit**: VarPath(), VarPathForWrite(), IsVarLayout(), IsVolatileFile(), FilesInWrongLocation()
- **Integration**: All commands work in both layouts (see matrix above)
- **E2E**: Full migration workflow from legacy to var/
- **Regression**: Existing test suite passes without modification
- **Cross-layout**: Sync between legacy clone and var/ clone
