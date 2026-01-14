---
title: 'Fix Multi-repo Export Path Resolution'
status: draft
spec_type: implementation
created: '2026-01-14'

skills: [golang]

phases:
  - name: 'Phase 1: Tracer Bullet'
    type: tracer
    status: pending

  - name: 'Phase 2: Test Coverage'
    type: mvs
    status: pending

beads:
  issue: oss-lbp
---

# Fix Multi-repo Export Path Resolution

> Resolve relative paths in multi-repo export from config file directory instead of CWD.

## Success Criteria

| ID     | Criterion                                          | Validation                          |
| ------ | -------------------------------------------------- | ----------------------------------- |
| SC-001 | Relative paths resolve from config directory       | Unit test with mock CWD             |
| SC-002 | Existing tests pass                                | `go test ./...`                     |
| SC-003 | No spurious directories created in daemon context | Manual test with `bd sync --daemon` |

## Scope

- `internal/storage/sqlite/multirepo_export.go:121`
- `internal/storage/sqlite/multirepo_export_test.go` (new/extend)

## Risks

| ID    | Risk                            | Likelihood | Impact | Mitigation                    |
| ----- | ------------------------------- | ---------- | ------ | ----------------------------- |
| R-001 | Break existing absolute paths   | Low        | High   | Preserve absolute path bypass |
| R-002 | Config file path unavailable    | Low        | Medium | Fall back to dbPath directory |

## Unknowns

- Whether `config.ConfigFileUsed()` is always set in daemon context
- If dbPath is a better base than config file location

## Atomicity

Each phase is independently mergeable and rollback-safe:

- **Phase 1**: Fix + basic verification
- **Phase 2**: Comprehensive test coverage
- **Rollback**: `git revert` safe for any phase

## Existing Patterns

The codebase has established patterns for this exact problem:

```go
// From cmd/bd/autoflush.go:95-100
// canonicalizeIfRelative ensures path is absolute for filepath.Rel() compatibility.
// Guards against any code path that might set dbPath to relative.
// See GH#959 for root cause analysis.
func canonicalizeIfRelative(path string) string {
    if path != "" && !filepath.IsAbs(path) {
        return utils.CanonicalizePath(path)
    }
    return path
}
```

## Spec Files

- [Requirements](requirements.md) — EARS format
- [Design](design.md) — Architecture decisions
- [Tasks](tasks.md) — Phase breakdown

## Execution

**Always use `/spec:run` to execute phases:**

```bash
/spec:run oss-lbp           # Execute next pending phase
/spec:run oss-lbp --phase 2 # Execute specific phase
```
