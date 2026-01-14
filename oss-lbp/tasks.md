# Tasks: Fix Path Resolution Bugs

## Phase 1: Extract Helper (Tracer Bullet)

**Goal**: Extract `canonicalizeIfRelative` to utils and verify build

| ID   | Task                                                | Parallel | Status  |
| ---- | --------------------------------------------------- | -------- | ------- |
| T010 | Add `CanonicalizeIfRelative()` to `utils/path.go`   | -        | pending |
| T011 | Update `autoflush.go` to use `utils.Canonicalize…`  | -        | pending |
| T012 | Run `go build ./...` to verify no import cycles     | -        | pending |
| T013 | Run existing tests to verify no regression          | -        | pending |

**Validation**:
- `go build ./...` succeeds
- `go test ./...` passes
- `autoflush.go` behavior unchanged

---

## Phase 2: Fix All Bugs (MVS)

**Goal**: Apply fixes using consistent repo-root resolution

| ID   | Task                                                | Parallel | Depends | Status  |
| ---- | --------------------------------------------------- | -------- | ------- | ------- |
| T020 | Fix `multirepo_export.go` path resolution           | -        | T013    | pending |
| T021 | Fix `worktree_cmd.go` redirect computation          | [P]      | T013    | pending |
| T024 | Fix `config.go:ResolveExternalProjectPath` (same bug)| [P]      | T013    | pending |
| T022 | Manual test: `bd sync` from different CWDs          | -        | T020    | pending |
| T023 | Manual test: `bd worktree create .trees/a/b`        | -        | T021    | pending |
| T025 | Manual test: external_projects from different CWDs  | -        | T024    | pending |

**Validation**:
- `bd sync` from repo root and `.beads/` produce same result
- Worktree redirect at depth 3+ has correct `../` count
- `external_projects` paths resolve correctly regardless of CWD

---

## Phase 3: Test Coverage (MVS)

**Goal**: Comprehensive test coverage across all execution contexts

### Unit Tests (mock CWD)

| ID   | Task                                                | Parallel | Depends | Status  |
| ---- | --------------------------------------------------- | -------- | ------- | ------- |
| T030 | Add unit test for `CanonicalizeIfRelative()`        | -        | T025    | pending |
| T031 | Add test: multirepo export with relative path       | [P]      | T030    | pending |
| T032 | Add test: multirepo export with absolute path       | [P]      | T030    | pending |
| T033 | Add test: multirepo export with empty config        | [P]      | T030    | pending |
| T034 | Add test: worktree redirect at depths 1, 2, 3       | [P]      | T030    | pending |
| T035 | Add test: external_projects path resolution         | [P]      | T030    | pending |

### CWD Variation Tests (key regression tests)

| ID   | Task                                                | Parallel | Depends | Status  |
| ---- | --------------------------------------------------- | -------- | ------- | ------- |
| T040 | Test: repos.additional from repo root               | -        | T035    | pending |
| T041 | Test: repos.additional from `.beads/` directory     | [P]      | T040    | pending |
| T042 | Test: repos.additional from subdirectory            | [P]      | T040    | pending |

### Sync Mode Tests (E2E)

| ID   | Task                                                | Parallel | Depends | Status  |
| ---- | --------------------------------------------------- | -------- | ------- | ------- |
| T050 | Test: Normal sync mode path resolution              | -        | T042    | pending |
| T051 | Test: Sync-branch mode with daemon                  | [P]      | T050    | pending |
| T052 | Test: External BEADS_DIR mode                       | [P]      | T050    | pending |
| T053 | Run full test suite                                 | -        | T051-52 | pending |

**Validation**:
- `go test -v ./... -run "Canonicalize|Export|Worktree"` passes
- `go test -v ./cmd/bd/... -run "SyncMode"` passes
- E2E tests pass with daemon running

---

## Dependency Graph

```
T010 ─> T011 ─> T012 ─> T013
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
            T020                    T021
              │                       │
              ▼                       ▼
            T022                    T023
              │                       │
              └───────────┬───────────┘
                          ▼
                        T030
                          │
          ┌───────┬───────┼───────┬───────┐
          ▼       ▼       ▼       ▼       ▼
        T031    T032    T033    T034     ...
          │       │       │       │
          └───────┴───────┴───────┴─> T035
```

## Files to Modify

| Phase | File                                              | Change Type |
| ----- | ------------------------------------------------- | ----------- |
| 1     | `internal/utils/path.go`                          | Add func    |
| 1     | `cmd/bd/autoflush.go`                             | Refactor    |
| 2     | `internal/storage/sqlite/multirepo_export.go`     | Fix         |
| 2     | `cmd/bd/worktree_cmd.go`                          | Fix         |
| 3     | `internal/utils/path_test.go`                     | Add tests   |
| 3     | `internal/storage/sqlite/multirepo_export_test.go`| Add tests   |
| 3     | `cmd/bd/worktree_cmd_test.go`                     | Add tests   |
