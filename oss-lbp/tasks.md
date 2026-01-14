# Tasks: Fix Multi-repo Export Path Resolution

## Phase 1: Tracer Bullet

**Goal**: Implement fix and verify basic functionality

| ID   | Task                                           | Parallel | Status  |
| ---- | ---------------------------------------------- | -------- | ------- |
| T010 | Modify `exportToRepo()` path resolution logic  | -        | pending |
| T011 | Add config directory resolution with fallback  | -        | pending |
| T012 | Run existing tests to verify no regression     | -        | pending |
| T013 | Manual test: `bd sync` from different CWDs     | -        | pending |

**Validation**:
- `go test ./internal/storage/sqlite/...` passes
- `bd sync` from repo root produces same result as from `.beads/`

---

## Phase 2: Test Coverage

**Goal**: Add targeted test coverage for the fix

| ID   | Task                                          | Parallel | Depends | Status  |
| ---- | --------------------------------------------- | -------- | ------- | ------- |
| T020 | Add unit test: relative path + config dir     | -        | T013    | pending |
| T021 | Add unit test: absolute path bypass           | [P]      | T020    | pending |
| T022 | Add unit test: empty config fallback to dbPath| [P]      | T020    | pending |
| T023 | Add test: tilde expansion + relative          | [P]      | T020    | pending |

**Validation**: `go test -v ./internal/storage/sqlite/... -run Export` passes

---

## Dependency Graph

```
T010 ─> T011 ─> T012 ─> T013
                          │
                          ▼
            T020 ─┬─> T021 ─┐
                  ├─> T022 ─┼─> (complete)
                  └─> T023 ─┘
```

## Files to Modify

| Phase | File                                              | Change Type |
| ----- | ------------------------------------------------- | ----------- |
| 1     | `internal/storage/sqlite/multirepo_export.go`     | Modify      |
| 2     | `internal/storage/sqlite/multirepo_export_test.go`| Add/Extend  |
