# ADR-0001: Cross-Repo Orphan Detection Approach

**Status**: Proposed
**Date**: 2026-01-19
**Context**: GH#1196 (git notes proposal), PR#1200 (--db fix)

## Context

When beads database lives in a separate repository from code, orphan detection cannot correlate commit messages (`(bd-xxx)`) with open issues because it scans the wrong git history.

GH#1196 proposed native git notes support. PR#1200 fixed the `--db` flag, which inadvertently enables a simpler solution.

## Decision Drivers

- **Simplicity**: Minimize new code, leverage existing infrastructure
- **UX**: Intuitive workflow for users
- **Flexibility**: Support multiple use cases
- **Maintenance**: Low ongoing burden

## Considered Options

### Option 1: Document `--db` Workflow (Recommended)

**How it works**: Run `bd orphans` from code repo with `--db` pointing to beads repo.

```bash
cd ~/my-code-repo
bd orphans --db ~/my-beads-repo/.beads/beads.db
```

| Pros | Cons |
|------|------|
| Already works (PR#1200) | Requires running from code repo |
| Zero implementation cost | Mental model inversion |
| Uses existing flags | Less discoverable |

### Option 2: Add `--git-path` Flag

**How it works**: Specify where to scan commits, independent of cwd.

```bash
cd ~/my-beads-repo
bd orphans --git-path ~/my-code-repo
```

| Pros | Cons |
|------|------|
| Run from either repo | New flag to implement |
| More intuitive for beads users | Minor code change |
| Explicit control | Yet another flag |

### Option 3: Native Git Notes Support

**How it works**: Full bidirectional annotation system per GH#1196.

```bash
bd notes add PLAN-123
bd notes push
bd orphans --include-notes
```

| Pros | Cons |
|------|------|
| Bidirectional linking | Significant implementation |
| Works with remote repos | Rebase fragility |
| Standard git mechanism | Extra sync burden |
| Retroactive annotation | GitHub doesn't display |

## Decision

**Recommend Option 1** (document `--db` workflow) as the immediate solution.

**Consider Option 2** (`--git-path`) if user feedback indicates the inverted workflow is confusing.

**Defer Option 3** (git notes) until there's clear demand for bidirectional linking or remote-only scenarios.

## Consequences

### Positive
- No implementation needed for primary use case
- PR#1200 provides value immediately upon merge
- Clear path for future enhancement if needed

### Negative
- Users must learn "flip perspective" mental model
- Documentation must clearly explain the pattern
- May revisit if workflow proves unintuitive

### Risks
- Users may not discover the `--db` solution
- Mitigation: Add examples to `bd orphans --help` and docs

## Validation

Before closing GH#1196:
1. Verify PR#1200 merged and working
2. Test cross-repo workflow end-to-end
3. Update documentation with examples
4. Gather user feedback on workflow clarity
