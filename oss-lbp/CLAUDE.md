---
title: 'Fix Path Resolution Bugs (oss-lbp + GH#1098)'
status: draft
spec_type: implementation
created: '2026-01-14'

skills: [golang]

phases:
  - name: 'Phase 1: Extract Helper'
    type: tracer
    status: pending

  - name: 'Phase 2: Fix Both Bugs'
    type: mvs
    status: pending

  - name: 'Phase 3: Test Coverage'
    type: mvs
    status: pending

beads:
  issue: oss-lbp
  related: [oss-3ui]
---

# Fix Path Resolution Bugs

> Extract `canonicalizeIfRelative` to utils and fix two related path resolution bugs.

**Upstream**: [GH#1098](https://github.com/steveyegge/beads/issues/1098) (worktree redirect depth)

## Success Criteria

| ID     | Criterion                                          | Validation                              |
| ------ | -------------------------------------------------- | --------------------------------------- |
| SC-001 | Helper extracted to utils/                         | `go build ./...`                        |
| SC-002 | Multi-repo export uses config dir base             | Unit test with mock CWD                 |
| SC-003 | Worktree redirect has correct depth                | `bd worktree create .trees/deep/nested` |
| SC-004 | Existing tests pass                                | `go test ./...`                         |
| SC-005 | No spurious directories in daemon context          | Manual test with `bd sync --daemon`     |

## Scope

**Bug 1 (oss-lbp)**: Multi-repo export resolves from CWD
- `internal/storage/sqlite/multirepo_export.go:121`

**Bug 2 (GH#1098)**: Worktree redirect depth incorrect
- `cmd/bd/worktree_cmd.go:205`

**Bug 3 (discovered)**: Same CWD bug in external_projects resolution
- `internal/config/config.go:456-461` (ResolveExternalProjectPath)
- Comment says "from config file location or cwd" but only uses CWD

**Shared helper**:
- `internal/utils/path.go` (extract from autoflush.go)
- `cmd/bd/autoflush.go` (update to use utils)

## Risks

| ID    | Risk                            | Likelihood | Impact | Mitigation                    |
| ----- | ------------------------------- | ---------- | ------ | ----------------------------- |
| R-001 | Break existing absolute paths   | Low        | High   | Preserve absolute path bypass |
| R-002 | Config file path unavailable    | Low        | Medium | Fall back to dbPath directory |
| R-003 | Sync-branch mode different CWD  | Medium     | High   | Test internal worktree context |
| R-004 | External BEADS_DIR mode         | Medium     | Medium | Config discovery may differ |
| R-005 | Daemon startup race condition   | Low        | Medium | Config loaded before CWD change |

## Unknowns

- Whether `config.ConfigFileUsed()` is always set in daemon context
- If dbPath is a better base than config file location

## Test Matrix

*Multi-dimensional test coverage for path resolution across all execution contexts.*

### Dimension 1: Sync Modes (from docs/SYNC.md)

| Mode | Trigger | Config Used | Test Focus |
|------|---------|-------------|------------|
| Normal | Default `bd sync` | Project `.beads/config.yaml` | repos.additional paths |
| Sync-branch | `sync.branch` config | Project config | Daemon CWD in internal worktree |
| External | `BEADS_DIR` env | External repo config | Config discovery across repos |
| From-main | `sync.from_main` config | Main branch config | Branch-switching context |
| Local-only | No git remote | Project config | Export path resolution |
| Export-only | `--no-pull` flag | Project config | Skip-pull path handling |

### Dimension 2: Execution Context

| Context | CWD | Config Discovery | Risk |
|---------|-----|------------------|------|
| CLI from repo root | `/repo/` | Walks up, finds `.beads/` | Low |
| CLI from .beads/ | `/repo/.beads/` | Immediate find | **HIGH** - Bug trigger |
| CLI from subdirectory | `/repo/src/` | Walks up | Medium |
| Daemon (normal) | `/repo/.beads/` | Set at startup | **HIGH** - Bug trigger |
| Daemon (sync-branch) | Internal worktree | Different .beads location | High |
| Worktree | `/repo/.worktrees/feat/` | Redirected | Medium |

### Dimension 3: Path Types

| Input Path | Expected Resolution | Notes |
|------------|---------------------|-------|
| `oss/` (relative) | `{repo}/oss/` | NOT `.beads/oss/` |
| `./oss/` (explicit relative) | `{repo}/oss/` | Same as above |
| `../sibling/` (parent-relative) | `{repo}/../sibling/` | For external_projects |
| `/abs/path/` (absolute) | `/abs/path/` | Unchanged |
| `~/path/` (tilde) | Expanded, then resolved | Two-step |

### Dimension 4: Bug × Mode × Context Matrix

| Bug | Mode | CWD | Expected | Test |
|-----|------|-----|----------|------|
| 1 (multi-repo) | Normal | repo root | ✓ Works | Baseline |
| 1 (multi-repo) | Normal | `.beads/` | ❌ Fails | **Key test** |
| 1 (multi-repo) | Daemon | `.beads/` | ❌ Fails | **Key test** |
| 1 (multi-repo) | External | ext repo | ? Untested | New test |
| 2 (worktree) | N/A | repo root | ❌ Fails | GH#1098 |
| 2 (worktree) | N/A | subdirectory | ❌ Fails | Depth varies |
| 3 (ext_projects) | Normal | repo root | ✓ Works | Baseline |
| 3 (ext_projects) | Normal | `.beads/` | ❌ Fails | Same pattern |
| 3 (ext_projects) | Daemon | `.beads/` | ❌ Fails | Same pattern |

### Existing Test Files to Extend

| File | Current Coverage | Add |
|------|------------------|-----|
| `sync_modes_test.go` | Mode transitions | CWD variation |
| `sync_external_test.go` | BEADS_DIR mode | Multi-repo paths |
| `worktree_cmd_test.go` | Basic creation | Depth variations |
| `config_test.go` | Config loading | CWD mocking |

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
