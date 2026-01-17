---
title: '.beads/var/ Layout Migration'
status: draft
spec_type: implementation
created: '2026-01-17'
source_plan: ~/.claude/plans/encapsulated-seeking-backus.md
upstream_issue: https://github.com/steveyegge/beads/issues/919

skills:
  feature: [golang]
  foundational: [spec, beads]

location:
  remote: github.com/steveyegge/beads
  path: specs/beads-var-layout

beads:
  epic: null  # Set after bd create
  worktree_path: .worktrees/beads-var-layout
  worktree_branch: feature/beads-var-layout

success_criteria:
  - "SC-001: All existing tests pass without modification"
  - "SC-002: Both layouts (legacy/var) work transparently"
  - "SC-003: Migration command safely moves files"
  - "SC-004: Doctor detects and reports migration option"
  - "SC-005: Documentation updated"

phases:
  - name: 'Phase 1: End-to-End Tracer'
    type: tracer
    status: pending
    description: 'paths.go + DatabasePath + bd init var/ default + layout:v2'

  - name: 'Phase 2: Remaining Consumers'
    type: mvs
    status: pending
    description: 'Update 5 remaining consumer files to use VarPath()'

  - name: 'Phase 3: Doctor & Migration'
    type: mvs
    status: pending
    description: 'bd migrate var command + doctor --fix for strays'

  - name: 'Phase 4: Documentation & Tests'
    type: mvs
    status: pending
    description: 'ARCHITECTURE.md, integration tests, test matrices'

  - name: 'Phase 5: Closing'
    type: closing
    status: pending
    merge_strategy: pr
---

# .beads/var/ Layout Migration

> Introduce `.beads/var/` subdirectory for volatile files, simplifying `.gitignore` to a single `var/` rule with zero regressions.

## Success Criteria

| ID     | Criterion                                    | Validation                           |
| ------ | -------------------------------------------- | ------------------------------------ |
| SC-001 | All existing tests pass without modification | `go test ./...`                      |
| SC-002 | Both layouts (legacy/var) work transparently | Integration tests for both           |
| SC-003 | Migration command safely moves files         | `bd migrate var --dry-run` + execute |
| SC-004 | Doctor detects and reports migration option  | `bd doctor` shows optional migration |
| SC-005 | Documentation updated                        | ARCHITECTURE.md reflects new layout  |

## Scope

**New Files:**

- `internal/beads/paths.go` — Centralized volatile file path resolution (read-both pattern)
- `internal/beads/paths_test.go` — Unit tests
- `internal/configfile/layout.go` — Layout versioning (v1/v2)
- `cmd/bd/migrate_var.go` — Migration command

**Modified Files (with symbol paths):**

| File | Symbols to Modify | Line |
|------|-------------------|------|
| `internal/configfile/configfile.go` | `Config.DatabasePath` | :96 |
| `cmd/bd/init.go` | `initCmd.Run` | :44 |
| `cmd/bd/daemon_config.go` | `getPIDFilePath`, `getLogFilePath` | :77, :86 |
| `internal/rpc/socket_path.go` | `ShortSocketPath`, `EnsureSocketDir` | :31, :70 |
| `cmd/bd/sync_merge.go` | `loadBaseState`, `saveBaseState` | :522, :569 |
| `cmd/bd/daemon_sync_state.go` | `LoadSyncState`, `SaveSyncState` | :45, :70 |
| `cmd/bd/daemon_lock.go` | `acquireDaemonLock`, `tryDaemonLock`, `checkPIDFile` | :45, :90, :188 |
| `internal/lockfile/lock.go` | `TryDaemonLock`, `checkPIDFile` | :28, :76 |
| `cmd/bd/doctor/migration.go` | (new functions) | — |
| `cmd/bd/doctor/gitignore.go` | `GitignoreTemplate` | :14 |
| `docs/ARCHITECTURE.md` | Directory structure diagram | :314 |
- `docs/ARCHITECTURE.md` — Update directory structure diagram

## Directory Layout (After Migration)

```
.beads/
├── var/                      # VOLATILE (gitignored directory)
│   ├── beads.db              # SQLite database
│   ├── beads.db-journal      # SQLite journaling
│   ├── beads.db-wal          # SQLite WAL
│   ├── beads.db-shm          # SQLite shared memory
│   ├── daemon.lock           # Daemon flock
│   ├── daemon.log            # Daemon logs
│   ├── daemon.pid            # Daemon PID
│   ├── bd.sock               # Unix socket
│   ├── sync_base.jsonl       # Sync merge base state
│   ├── .sync.lock            # Sync concurrency guard
│   ├── sync-state.json       # Sync backoff state
│   ├── beads.base.jsonl      # Merge artifact
│   ├── beads.base.meta.json  # Merge artifact metadata
│   ├── beads.left.jsonl      # Merge artifact
│   ├── beads.left.meta.json  # Merge artifact metadata
│   ├── beads.right.jsonl     # Merge artifact
│   ├── beads.right.meta.json # Merge artifact metadata
│   ├── last-touched          # Last modified tracking
│   ├── .local_version        # Version tracking
│   └── export_hashes.db      # Export tracking
│
├── redirect                  # STAYS AT ROOT (worktree discovery)
├── issues.jsonl              # PERSISTENT (git-tracked)
├── interactions.jsonl        # PERSISTENT (git-tracked)
├── metadata.json             # CONFIG (git-tracked)
├── routes.jsonl              # CONFIG (git-tracked, if exists)
├── molecules.jsonl           # CONFIG (git-tracked, if exists)
└── .gitignore                # SIMPLIFIED: just "var/" + legacy patterns
```

## Risks

| ID    | Risk                          | L   | I   | Mitigation                              |
| ----- | ----------------------------- | --- | --- | --------------------------------------- |
| R-001 | External tools hardcode paths | M   | M   | 6-month backward compatibility window   |
| R-002 | sync_base.jsonl loss on move  | L   | L   | Read-both pattern finds it              |
| R-003 | Worktree redirect breaks      | L   | H   | Keep redirect at root, not in var/      |
| R-004 | Daemon socket path too long   | L   | M   | Existing fallback to /tmp/beads-{hash}/ |
| R-005 | **Old bd with new repo**      | M   | H   | `layout: v2` field, doctor warning      |
| R-006 | Python MCP hardcodes paths    | M   | M   | Audit MCP, separate PR                  |
| R-007 | CI/CD scripts hardcode paths  | L   | M   | Document in CHANGELOG                   |
| R-009 | Daemon running during migrate | M   | M   | Require daemon stop before migrate      |

**R-005 mitigation**: `layout: "v2"` in metadata.json. Future unknown layouts trigger "please upgrade".

## Unknowns

- Will maintainer accept var/ naming convention?
- Python MCP integration paths need audit (→ R-006)
- CI/CD scripts in ecosystem? (→ R-007)

## Atomicity

Each phase is independently mergeable and rollback-safe:

- **Phase 1**: VarPath() fallback ensures existing code works
- **Phase 2**: Consumers use VarPath() which handles both layouts
- **Phase 3**: Migration optional, doctor shows as "optional" priority
- **Rollback**: `mv .beads/var/* .beads/ && rmdir .beads/var`

## Spec Files

- [Requirements](requirements.md) — EARS format
- [Design](design.md) — Architecture decisions
- [Tasks](tasks.md) — Phase breakdown
- [ADR: var/ Directory Pattern](adr/0001-var-directory-for-volatile-files.md)

## Execution

**Always use `/spec:run` to execute phases:**

```bash
/spec:run beads-var-layout           # Execute next pending phase
/spec:run beads-var-layout --phase 2 # Execute specific phase
```

**Why:**

- Gate L3 approval enforced per phase
- Context isolation via implementor agent
- Beads tracking automated
- Commit workflow via /commit

**Never** implement phases directly without `/spec:run`.