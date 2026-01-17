# Tasks: .beads/var/ Layout Migration

## Phase 1: Centralized Paths Module (Tracer)

**Goal**: Create `internal/beads/paths.go` with VarPath() and layout detection

| ID   | Task                            | Parallel | Status  |
| ---- | ------------------------------- | -------- | ------- |
| T001 | Create internal/beads/paths.go  | -        | pending |
| T002 | Implement VarPath() function    | -        | pending |
| T003 | Implement IsVarLayout() check   | -        | pending |
| T004 | Implement VarDir() helper       | -        | pending |
| T005 | Add BD_LEGACY_LAYOUT env check  | -        | pending |
| T006 | Create paths_test.go unit tests | -        | pending |

**Validation**:

```bash
go test ./internal/beads/... -v -run TestVarPath
go test ./internal/beads/... -v -run TestIsVarLayout
```

---

## Phase 2: Consumer Migration (MVS)

**Goal**: Update 6 consumer files to use VarPath()

| ID   | Task                                  | Parallel | Status  |
| ---- | ------------------------------------- | -------- | ------- |
| T010 | Update configfile.DatabasePath()      | -        | pending |
| T011 | Update daemon_config.go paths         | [P]      | pending |
| T012 | Update rpc/socket_path.go             | [P]      | pending |
| T013 | Update sync_merge.go paths            | [P]      | pending |
| T014 | Update daemon_sync_state.go paths     | [P]      | pending |
| T015 | Update lockfile/lock.go paths         | [P]      | pending |
| T016 | Run existing test suite               | -        | pending |

**Validation**:

```bash
go test ./...
bd list  # Verify basic operations work
bd daemon start && bd daemon stop  # Verify daemon works
```

---

## Phase 3: Doctor & Migration Command (MVS)

**Goal**: Add migration detection and `bd migrate var` command

| ID   | Task                               | Parallel | Status  |
| ---- | ---------------------------------- | -------- | ------- |
| T020 | Add needsVarMigration() detection  | -        | pending |
| T021 | Add to DetectPendingMigrations()   | -        | pending |
| T022 | Create cmd/bd/migrate_var.go       | -        | pending |
| T023 | Implement runVarMigration()        | -        | pending |
| T024 | Implement --dry-run flag           | [P]      | pending |
| T025 | Implement --cleanup flag           | [P]      | pending |
| T026 | Update GitignoreTemplate           | -        | pending |
| T027 | Add var/ to requiredPatterns       | -        | pending |
| T028 | Create migration tests             | -        | pending |

**Validation**:

```bash
bd doctor  # Should show optional var-layout migration
bd migrate var --dry-run  # Preview changes
# In test directory:
bd migrate var  # Execute migration
bd doctor  # Should show no migration needed
bd list  # Commands work in new layout
```

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

**Validation**: PR created at github.com/steveyegge/beads with all tests passing

---

## Dependency Graph

```
Phase 1: Tracer
T001 ─> T002 ─> T003 ─> T004 ─> T005 ─> T006
                                          │
Phase 2: Consumer Migration               ▼
         ┌────────────────────────────────┴─────────────────┐
         │                                                  │
T010 ────┼─> T011 ─┬─> T016 (run tests)                     │
         │   T012 ─┤                                        │
         │   T013 ─┤                                        │
         │   T014 ─┤                                        │
         │   T015 ─┘                                        │
         │                                                  │
Phase 3: Doctor & Migration                                 ▼
T020 ─> T021 ─> T022 ─> T023 ─┬─> T024 ─┬─> T026 ─> T027 ─> T028
                              └─> T025 ─┘
                                                            │
Phase 4: Docs & Tests                                       ▼
T030 ─┬─> T031 ─┬─> T034 ─> T035
      ├─> T032 ─┤
      └─> T033 ─┘
                                                            │
Phase 5: Closing                                            ▼
TC01 ─> TC02 ─> TC03 ─> TC04
```

## Code Examples

### T001-T005: paths.go Implementation

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

func VarPath(beadsDir, filename string) string {
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return filepath.Join(beadsDir, filename)
    }
    if IsVarLayout(beadsDir) {
        return filepath.Join(beadsDir, "var", filename)
    }
    return filepath.Join(beadsDir, filename)
}

func IsVarLayout(beadsDir string) bool {
    if os.Getenv("BD_LEGACY_LAYOUT") == "1" {
        return false
    }
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
