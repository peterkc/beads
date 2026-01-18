# Design

## Architecture

### Current Flow (Problematic)

```
PersistentPreRun
    │
    ├─► isReadOnlyCommand(cmd.Name())     ← Decision based only on command name
    │         │
    │         ▼
    ├─► factory.NewWithOptions(ReadOnly)  ← Store opened with mode locked in
    │         │
    │         ▼
    └─► ensureDatabaseFresh()
              │
              ▼
        autoImportIfNewer()
              │
              ▼
        ClearAllExportHashes()            ← FAILS: write on read-only DB
```

### Proposed Flow (Fixed)

```
PersistentPreRun
    │
    ├─► isReadOnlyCommand(cmd.Name())
    │         │
    │         ▼
    ├─► isJSONLNewerThanDB()              ← NEW: Pre-store staleness check
    │         │
    │         ▼
    │   useReadOnly = readOnlyCmd && !stale
    │         │
    │         ▼
    ├─► factory.NewWithOptions(ReadOnly)  ← Mode now considers staleness
    │         │
    │         ▼
    └─► ensureDatabaseFresh()
              │
              ▼
        autoImportIfNewer()
              │
              ▼
        ClearAllExportHashes()            ← SUCCESS: store is read-write
```

## Key Decisions

### KD-001: File Mtime vs Content Hash

**Decision**: Use file mtime comparison, not content hash.

**Rationale**:
- Content hash requires reading entire JSONL file (can be >5MB)
- Mtime check is O(1) — two stat calls
- Existing `autoimport.CheckStaleness` also uses mtime as first check
- Edge cases (clock skew) are acceptable: worst case is unnecessary read-write mode

**Trade-off**: May occasionally open read-write when not strictly needed. This is acceptable because:
1. Read-write mode is the default for most commands anyway
2. The optimization (read-only) is about file watcher noise, not correctness

### KD-002: Helper Function Location

**Decision**: Add `isJSONLNewerThanDB()` to `staleness.go`.

**Rationale**:
- Groups staleness-related logic together
- `staleness.go` already has `ensureDatabaseFresh()`
- Avoids adding to already-large `main.go`

### KD-003: Symlink Handling

**Decision**: Use `os.Lstat` for JSONL, `os.Stat` for DB.

**Rationale**:
- JSONL may be symlinked (NixOS, home-manager)
- DB file is never symlinked
- Matches existing pattern in `autoimport/autoimport.go:292`

## Implementation Details

### New Function: `isJSONLNewerThanDB`

```go
// isJSONLNewerThanDB checks if JSONL file is newer than database file.
// Uses file mtime comparison without requiring an open store.
// Returns false if either file doesn't exist (safe default for fresh repos).
func isJSONLNewerThanDB(beadsDir, dbPath string) bool {
    jsonlPath := filepath.Join(beadsDir, "issues.jsonl")

    // Use Lstat for JSONL (symlink-aware per autoimport.go)
    jsonlStat, err := os.Lstat(jsonlPath)
    if err != nil {
        return false // JSONL doesn't exist, not stale
    }

    // Use Stat for DB (never symlinked)
    dbStat, err := os.Stat(dbPath)
    if err != nil {
        return true // DB doesn't exist, needs bootstrap (read-write)
    }

    return jsonlStat.ModTime().After(dbStat.ModTime())
}
```

### Integration Point

In `main.go`, around line 749:

```go
// Check if this is a read-only command (GH#804)
useReadOnly := isReadOnlyCommand(cmd.Name())

// GH#1089: If DB is stale, we need read-write mode for auto-import
if useReadOnly {
    beadsDir := filepath.Dir(dbPath)
    if isJSONLNewerThanDB(beadsDir, dbPath) {
        debug.Logf("DB stale (JSONL newer), using read-write mode for auto-import")
        useReadOnly = false
    }
}
```

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Clock skew causes false positive | Low | Low | Acceptable: opens read-write unnecessarily |
| Git operations change mtime unpredictably | Medium | Low | Matches existing behavior in autoimport |
| Performance regression from extra stat calls | Very Low | Low | Two stat calls are ~microseconds |

## Test Strategy

1. **Unit tests** for `isJSONLNewerThanDB()` covering:
   - JSONL newer than DB
   - DB newer than JSONL
   - JSONL missing
   - DB missing
   - Both missing

2. **Integration test** simulating the original bug:
   - Create DB, make JSONL newer
   - Run `bd --no-daemon ready`
   - Assert no warning in stderr
