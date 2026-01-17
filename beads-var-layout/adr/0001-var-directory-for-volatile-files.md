# ADR-0001: var/ Directory for Volatile Files

## Status

Proposed

## Context

The `.beads/` directory contains both:

1. **Persistent files** (git-tracked): `issues.jsonl`, `interactions.jsonl`, `metadata.json`
2. **Volatile files** (gitignored): `beads.db`, `daemon.*`, `sync_base.jsonl`, merge artifacts

Currently, all files live at the `.beads/` root, requiring an extensive `.gitignore` file that changes frequently as new runtime artifacts are added. This causes:

- **Friction**: Every new volatile file requires gitignore updates
- **Doctor warnings**: Outdated gitignore patterns trigger repair prompts
- **Maintenance burden**: Protected-branch mode users face extra sync overhead

Issue #919 proposes organizing volatile files into a `var/` subdirectory.

## Decision

Introduce `.beads/var/` subdirectory for all volatile (machine-local) files.

### Directory Structure

```
.beads/
├── var/                      # VOLATILE (gitignored)
│   ├── beads.db
│   ├── daemon.{lock,log,pid}
│   ├── bd.sock
│   ├── sync_base.jsonl
│   ├── .sync.lock
│   └── ... (all runtime files)
│
├── issues.jsonl              # PERSISTENT (git-tracked)
├── interactions.jsonl
├── metadata.json
├── redirect                  # Special: stays at root
└── .gitignore                # Simplified: "var/" + legacy
```

### Simplified Gitignore

```gitignore
# Volatile files directory
var/

# Legacy patterns (backward compatibility)
*.db
...
```

### Key Decisions

1. **`redirect` stays at root**: Required for worktree discovery before beads context exists
2. **Legacy patterns preserved**: Enables mixed-layout clones during transition
3. **Migration is optional**: Existing users continue working without changes
4. **6-month support window**: Legacy layout supported minimum 6 months
5. **Read-both coexistence**: VarPath() checks both locations on read for edge case safety

## Alternatives Considered

### Alternative 1: Semantic Directories

```
.beads/
├── data/       # Database
├── runtime/    # Daemon, locks
├── sync/       # Sync state
└── config/     # Configuration
```

**Rejected**: Too many directories, higher complexity, marginal benefit over single `var/`.

### Alternative 2: XDG-Style Layout

Use `~/.local/share/beads/{project-hash}/` for runtime files.

**Rejected**:
- Breaks project isolation principle
- Complicates backup/cleanup
- Inconsistent with repo-local design

### Alternative 3: Environment-Based Paths

`BD_VAR_DIR` environment variable to override volatile path.

**Rejected**:
- Adds configuration burden
- Inconsistent across clones
- Doesn't solve gitignore problem

## Consequences

### Positive

- **Future-proof gitignore**: New volatile files automatically ignored via `var/`
- **Clear separation**: Persistent vs volatile files visually distinct
- **Follows conventions**: FHS `/var/` pattern widely understood
- **Reduced doctor noise**: Fewer gitignore validation failures

### Negative

- **Migration required**: Users must run `bd migrate var` for benefits
- **Transition period**: Mixed-layout clones during adoption
- **External tool updates**: Tools hardcoding paths need updates

### Risks

| Risk | Mitigation |
|------|------------|
| External tools break | 6-month backward compatibility window |
| sync_base.jsonl loss | Copy-then-delete, not move |
| Worktree discovery | redirect stays at root |

## Implementation

See `specs/beads-var-layout/` for full specification:

- `requirements.md` — EARS format requirements
- `design.md` — Architecture and component design
- `tasks.md` — Phased implementation plan

## References

- [GitHub Issue #919](https://github.com/steveyegge/beads/issues/919) — Original proposal
- [FHS /var/ specification](https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05.html) — Unix convention
- [docs/REPO_CONTEXT.md](../../../docs/REPO_CONTEXT.md) — Existing path resolution pattern
