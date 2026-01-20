---
spec_type: implementation
status: deferred
created: 2026-01-19
deferred_date: 2026-01-19
research: /Volumes/atlas/beads/research/git-notes-cross-repo/

deferred_reason: |
  Cross-repo orphan detection is solved by PR #1200's --db flag.
  Git notes only add value for retroactive tagging (commits missing
  issue references), which is a rare edge case. The complexity of
  exposing bd notes commands isn't justified by current use cases.

  Revisit if:
  - Users frequently forget (bd-xxx) in commit messages
  - Bidirectional commit→issue queries become a common need
  - Hook-based auto-capture is requested

success_criteria:
  - "SC-001: bd notes add annotates HEAD with issue reference"
  - "SC-002: bd notes list shows annotations on commits"
  - "SC-003: bd orphans --include-notes detects notes-annotated commits"
  - "SC-004: Rebase preserves notes via configured rewriteRef"

phases:
  - name: "Phase 1: Tracer Bullet"
    type: tracer
    status: pending
    description: "bd notes add + bd notes list working end-to-end"

  - name: "Phase 2: Orphan Integration"
    type: mvs
    status: pending
    description: "--include-notes flag for bd orphans"

  - name: "Phase 3: Sync & Init"
    type: mvs
    status: pending
    description: "push/fetch commands and init configuration"

  - name: "Phase 4: Closing"
    type: closing
    status: pending
    merge_strategy: pr
---

# Spec: Git Notes Bidirectional Linking

Add native git notes support to beads for bidirectional commit↔issue linking.

## Scope

### In Scope

| File | Purpose |
|------|---------|
| `cmd/bd/notes.go` | NEW: Notes subcommand (add, list, push, fetch, init) |
| `cmd/bd/doctor/git.go` | UPDATE: Add --include-notes to FindOrphanedIssues |
| `internal/types/notes.go` | NEW: Note struct and parser |
| `docs/CLI_REFERENCE.md` | UPDATE: Document bd notes commands |

### Out of Scope

- GitHub notes rendering (disabled since 2014)
- Automatic note creation on commit (hook-based, separate concern)
- Multi-repo note aggregation (future enhancement)

## Context

### Research Source

See `research/git-notes-cross-repo/` for full analysis including:
- git-appraise patterns (namespace architecture, JSON format)
- Rebase handling configuration
- Sync patterns for remote collaboration

### Why Git Notes?

| Capability | `--db` (PR#1200) | Git Notes |
|------------|-----------------|-----------|
| Cross-repo orphan detection | ✅ | ✅ |
| Retroactive commit tagging | ❌ | ✅ |
| Commit → Issue linking | ❌ | ✅ |
| Portable with commits | ❌ | ✅ |

**Key value**: Annotate commits *after* they're created, without changing SHAs.

### Interview Context

**Q1: What problem does this solve?**
Bidirectional linking—commits know their issues, not just issues knowing commits.

**Q2: Who benefits from this?**
Users with cross-repo setups, retroactive taggers, teams wanting commit-level traceability.

**Q3: What's the smallest useful version?**
`bd notes add ISSUE-ID` + `bd orphans --include-notes`

**Q4: What would make this fail?**
Notes lost on rebase (mitigated by `notes.rewriteRef` config), GitHub invisibility (documented limitation).

**Q5: How will we know it worked?**
Orphan detection finds commits annotated via notes that have no conventional `(bd-xxx)` message.

## Links

- [GH#1196](https://github.com/steveyegge/beads/issues/1196) — Original feature request
- [PR#1200](https://github.com/steveyegge/beads/pull/1200) — IssueProvider refactor (enables this)
- [Research Hub](../../research/git-notes-cross-repo/) — Full analysis
