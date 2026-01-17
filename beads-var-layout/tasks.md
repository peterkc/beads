# Tasks: .beads/var/ Layout Migration

## Phase 1: End-to-End Tracer Bullet

**Goal**: Prove var/ layout works end-to-end with ONE consumer before full migration

**Tracer Path**: `paths.go` → `DatabasePath()` → `bd init` → `bd list`

**Patterns**: See [design.md](design.md#decision-5-read-both-coexistence-pattern) for read-both pattern

| ID   | Task                                      | Parallel | Status  |
| ---- | ----------------------------------------- | -------- | ------- |
| T001 | Create internal/beads/paths.go (minimal)  | -        | pending |
| T002 | Implement VarPath() with read-both        | -        | pending |
| T003 | Implement IsVarLayout() check             | -        | pending |
| T004 | Update configfile.DatabasePath() ONLY     | -        | pending |
| T005 | Update bd init to create var/ by default  | -        | pending |
| T006 | Add --legacy flag to bd init              | -        | pending |
| T007 | Create paths_test.go unit tests           | -        | pending |

**Tracer Validation** (must all pass before Phase 2):

```bash
# Unit tests
go test ./internal/beads/... -v -run TestVarPath
# Expected: PASS

# End-to-end: New user path
cd $(mktemp -d) && git init && bd init
ls .beads/var/          # Expected: beads.db exists
bd list                 # Expected: "No issues found" or empty list
cat .beads/metadata.json | grep layout  # Expected: "layout": "v2"

# End-to-end: Legacy path
cd $(mktemp -d) && git init && bd init --legacy
ls .beads/              # Expected: beads.db at root, no var/
bd list                 # Expected: "No issues found" or empty list

# End-to-end: Existing legacy user
# (use existing .beads/ without var/)
bd list                 # Expected: Works, no regression
```

**If validation fails**: `bd doctor --verbose` to diagnose

**Why This Tracer**:
- Touches all layers: new module → existing consumer → CLI → file system
- Validates read-both pattern works in practice
- Proves zero regression for legacy users
- If this fails, we know before migrating 5 more consumers

---

## Phase 2: Remaining Consumer Migration (MVS)

**Goal**: Update remaining 5 consumer files to use VarPath()

**Note**: configfile.DatabasePath() already migrated in Phase 1 tracer

| ID   | Task                                  | Parallel | Status  |
| ---- | ------------------------------------- | -------- | ------- |
| T010 | Update daemon_config.go paths         | [P]      | pending |
| T011 | Update rpc/socket_path.go             | [P]      | pending |
| T012 | Update sync_merge.go paths            | [P]      | pending |
| T013 | Update daemon_sync_state.go paths     | [P]      | pending |
| T014 | Update lockfile/lock.go paths         | [P]      | pending |
| T015 | Run existing test suite               | -        | pending |

**Validation**:

```bash
go test ./...
bd list  # Verify basic operations work
bd daemon start && bd daemon stop  # Verify daemon works
bd sync  # Verify sync works
```

---

## Phase 3a: Doctor Detection (MVS)

**Goal**: Add migration detection and doctor --fix for stray files

**Patterns**: See [design.md](design.md#cmdbd-doctor-migration-go-addition) for FilesInWrongLocation() design

| ID   | Task                                  | Parallel | Status  |
| ---- | ------------------------------------- | -------- | ------- |
| T020 | Add needsVarMigration() detection     | -        | pending |
| T021 | Add FilesInWrongLocation() detection  | -        | pending |
| T022 | Add to DetectPendingMigrations()      | -        | pending |
| T023 | Add stray file fix to doctor --fix    | -        | pending |
| T024 | Update GitignoreTemplate              | -        | pending |
| T025 | Add var/ to requiredPatterns          | -        | pending |

**Validation**:

```bash
# Legacy user sees option
bd doctor  # Expected: "ℹ Optional: var/ layout available"

# Stray file cleanup (after manual var/ creation)
mkdir -p .beads/var && touch .beads/stray.db
bd doctor                 # Expected: "⚠ 1 file in wrong location"
bd doctor --fix           # Expected: Moves stray.db to var/
```

**If validation fails**: `bd doctor --verbose`

---

## Phase 3b: Migration Command (MVS)

**Goal**: Implement `bd migrate var` command

| ID   | Task                                  | Parallel | Status  |
| ---- | ------------------------------------- | -------- | ------- |
| T030 | Create cmd/bd/migrate_var.go          | -        | pending |
| T031 | Implement runVarMigration()           | -        | pending |
| T032 | Implement --dry-run flag              | -        | pending |
| T033 | Create migration tests                | -        | pending |

**Validation**:

```bash
# Dry run
bd migrate var --dry-run  # Expected: "Would create var/", "Would move 12 files"

# Execute migration
bd migrate var            # Expected: "Created var/", "Moved 12 files"
bd doctor                 # Expected: No warnings
ls .beads/var/            # Expected: beads.db, daemon.*, etc.
```

**If validation fails**: `bd doctor --verbose`

---

## Phase 4: Documentation & Tests (MVS)

**Goal**: Update documentation and add comprehensive tests

| ID   | Task                                    | Parallel | Status  |
| ---- | --------------------------------------- | -------- | ------- |
| T030 | Update docs/ARCHITECTURE.md             | -        | pending |
| T031 | Add migration section to TROUBLESHOOTING| [P]      | pending |
| T032 | Add integration tests for both layouts  | [P]      | pending |
| T033 | Add integration tests for migration     | [P]      | pending |
| T034 | Test mixed-layout sync scenario         | -        | pending |
| T035 | Update CHANGELOG.md                     | -        | pending |

**Validation**:

```bash
go test ./... -v
# Manual verification:
# 1. Clone A: legacy layout, Clone B: var/ layout
# 2. Create issue in A, sync to B
# 3. Update in B, sync back to A
```

---

## Phase 5: Closing

**Goal**: Create PR and finalize contribution

**merge_strategy: pr**

| ID   | Task                       | Parallel | Status  |
| ---- | -------------------------- | -------- | ------- |
| TC01 | Push branch to remote      | -        | pending |
| TC02 | Create draft PR            | -        | pending |
| TC03 | Link to GH#919 in PR body  | -        | pending |
| TC04 | Close beads tracking issue | -        | pending |

**Validation**:

```bash
# Verify PR exists and CI passes
gh pr view --json state,statusCheckRollup
gh pr checks
```

---

## Dependency Graph

```
Phase 1: End-to-End Tracer
─────────────────────────────────────────────────────────────
T001 ─> T002 ─> T003 ─> T004 ─> T005 ─> T006 ─> T007
  │       │       │       │       │       │
paths.go  │    IsVar   DB only  init   --legacy  tests
       VarPath  Layout           var/    flag
                                                    │
                      TRACER VALIDATION GATE        ▼
                                                    │
Phase 2: Remaining Consumers                        ▼
─────────────────────────────────────────────────────────────
         T010 ─┬─> T015 (run tests)
         T011 ─┤
         T012 ─┤
         T013 ─┤
         T014 ─┘
           │
      (all parallel)
                                                    │
Phase 3a: Doctor Detection                          ▼
─────────────────────────────────────────────────────────────
T020 ─> T021 ─> T022 ─> T023 ─> T024 ─> T025
  │       │       │       │       │       │
needs   files   detect  --fix   git    patterns
VarMig  wrong   pending         ignore
                                                    │
Phase 3b: Migration Command                         ▼
─────────────────────────────────────────────────────────────
T030 ─> T031 ─> T032 ─> T033
  │       │       │       │
migrate  run    dry-run  tests
.go      Mig
                                                    │
Phase 4: Docs & Tests                               ▼
─────────────────────────────────────────────────────────────
T040 ─┬─> T041 ─┬─> T044 ─> T045
  │   ├─> T042 ─┤     │       │
ARCH  │   └─> T043 ─┘  mixed  CHANGELOG
      │         │      sync
   TROUBLE   integ
   SHOOT     tests
                                                    │
Phase 5: Closing                                    ▼
─────────────────────────────────────────────────────────────
TC01 ─> TC02 ─> TC03 ─> TC04
  │       │       │       │
push    draft   link    close
        PR     GH#919  issue
```

## Code Examples

### T001-T003: paths.go Implementation (Read-Both Pattern)

```go
// internal/beads/paths.go
package beads

import (
    "os"
    "path/filepath"
)

var VolatileFiles = []string{
    "beads.db", "daemon.lock", "daemon.log", "daemon.pid",
    "bd.sock", "sync_base.jsonl", ".sync.lock", "sync-state.json",
    "beads.base.jsonl", "beads.left.jsonl", "beads.right.jsonl",
    "beads.base.meta.json", "beads.left.meta.json", "beads.right.meta.json",
    "last-touched", ".local_version", "export_hashes.db",
}

// VarPath returns path for volatile file using read-both pattern.
// For READS: checks var/ first, falls back to root (handles edge cases).
// For NEW files: uses layout preference (var/ if exists, else root).
func VarPath(beadsDir, filename string) string {
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return filepath.Join(beadsDir, filename)
    }

    varPath := filepath.Join(beadsDir, "var", filename)
    rootPath := filepath.Join(beadsDir, filename)

    // Read-both: check var/ first, then root
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

// VarPathForWrite returns path for writing (no fallback checking).
func VarPathForWrite(beadsDir, filename string) string {
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return filepath.Join(beadsDir, filename)
    }
    if IsVarLayout(beadsDir) {
        return filepath.Join(beadsDir, "var", filename)
    }
    return filepath.Join(beadsDir, filename)
}

// IsVarLayout checks layout field in metadata.json
// Falls back to var/ directory check for bootstrap scenarios
func IsVarLayout(beadsDir string, meta *Metadata) bool {
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return false
    }
    if meta != nil {
        return meta.Layout == "v2"
    }
    // Fallback for bootstrap
    varDir := filepath.Join(beadsDir, "var")
    info, err := os.Stat(varDir)
    return err == nil && info.IsDir()
}

func VarDir(beadsDir string) string {
    if IsVarLayout(beadsDir) {
        return filepath.Join(beadsDir, "var")
    }
    return beadsDir
}
```

### T010: configfile.DatabasePath() Update

```go
// Before
func (c *Config) DatabasePath(beadsDir string) string {
    return filepath.Join(beadsDir, c.Database)
}

// After
func (c *Config) DatabasePath(beadsDir string) string {
    return beads.VarPath(beadsDir, c.Database)
}
```

### T026: GitignoreTemplate Update

```go
const GitignoreTemplate = `# Volatile files directory
var/

# Legacy patterns (backward compatibility)
*.db
*.db?*
*.db-journal
*.db-wal
*.db-shm
daemon.lock
daemon.log
daemon.pid
bd.sock
sync-state.json
last-touched
.local_version
db.sqlite
bd.db
redirect
beads.base.jsonl
beads.base.meta.json
beads.left.jsonl
beads.left.meta.json
beads.right.jsonl
beads.right.meta.json
.sync.lock
sync_base.jsonl
export_hashes.db
`
```
