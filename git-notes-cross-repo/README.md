# Git Notes for Cross-Repo Orphan Detection

**Status**: Research Complete → Ready for Spec
**Related**: [GH#1196](https://github.com/steveyegge/beads/issues/1196), [PR#1200](https://github.com/steveyegge/beads/pull/1200)
**Started**: 2026-01-19
**Updated**: 2026-01-19

## Decision Summary

Cross-repo commit-issue tracking is a **solved problem** with multiple approaches. Research recommends:

| Phase | Action | Effort |
|-------|--------|--------|
| **1** | Document `--db` workflow + add `--git-path` flag | Low |
| **2** | Git notes support (`bd notes add/list/push`) | Medium |
| **3** | Auto-tracking issues, webhook export | Future |

**Key insight**: PR #1200's `IssueProvider` interface already enables cross-repo—just need UX improvements.

## Problem Statement

When beads database lives in a **separate repository** (via `BEADS_DIR` or `--db`), orphan detection fails because `bd orphans` scans its own git history for `(bd-xxx)` patterns, but code commits with those references are in a different repo.

### Scenario

```
~/my-beads-repo/.beads/     # Beads database (separate repo)
~/my-code-repo/             # Code with commits containing (bd-xxx)
```

Running `bd orphans` from beads repo scans beads repo's git history—no code commits found.

## Key Discovery: PR #1200 Already Enables This

PR #1200 introduced the `IssueProvider` interface that decouples:

| Concern | Source | Configurable Via |
|---------|--------|------------------|
| Git commit scanning | `gitPath` parameter | Current working directory |
| Issue retrieval | `IssueProvider` interface | `--db` flag |

### Working Solution (No New Flags Needed)

```bash
cd ~/my-code-repo                           # Stand in CODE repo
bd orphans --db ~/my-beads-repo/.beads      # Point to BEADS database
```

This scans commits in the code repo while fetching issues from the beads repo.

## Git Notes Proposal (from GH#1196)

The original proposal suggested native git notes support:

```bash
bd notes add PLAN-123        # Annotate HEAD in code repo
bd notes list                # Show annotations
bd notes push/fetch          # Sync refs/notes/beads
bd notes init                # Configure rebase preservation
bd orphans --include-notes   # Scan commits + notes
```

### Git Notes Evaluation

| Criterion | Assessment |
|-----------|------------|
| Solves cross-repo | Yes |
| Non-invasive to commits | Yes (no SHA changes) |
| Retroactive | Yes (can annotate existing commits) |
| GitHub visibility | No (disabled since 2014) |
| Rebase safety | Requires config (`notes.rewriteRef`) |
| Implementation effort | Medium (new subcommand, sync logic) |

## Alternatives Analysis

| Approach | Complexity | Already Works? | Notes |
|----------|------------|----------------|-------|
| `--db` from code repo | None | **Yes (PR#1200)** | Flip perspective |
| `--git-path` flag | Low | No | Explicit git path |
| Git notes | Medium | No | Full bidirectional |
| Hook-based linking | Medium | No | Auto-capture |

## Recommendation

### Phase 1: Document Existing Solution
The `--db` flag already solves the primary use case. Document this pattern:

```bash
# From code repo, scan commits against external beads DB
cd ~/my-code-repo
bd orphans --db ~/my-beads-repo/.beads/beads.db
```

### Phase 2: Consider `--git-path` (Optional)
If users need to run from the beads repo instead:

```bash
# From beads repo, scan external code repo
cd ~/my-beads-repo
bd orphans --git-path ~/my-code-repo
```

This would require exposing the existing `gitPath` parameter in `FindOrphanedIssues()`.

### Phase 3: Git Notes (Future)
Consider git notes only if:
- Users need bidirectional commit↔issue linking
- Remote-only code repos require annotation storage
- Multiple beads repos track the same code repo

## Open Questions

1. Should GH#1196 be closed as "solved by PR#1200"?
2. Is the `--db` workflow intuitive enough, or does UX need improvement?
3. Are there scenarios where git notes provide value beyond `--db`?

## References

- [Git Notes Documentation](https://git-scm.com/docs/git-notes)
- [Tyler Cipriani: Git Notes - Git's Coolest, Most Unloved Feature](https://tylercipriani.com/blog/2022/11/19/git-notes-gits-coolest-most-unloved-feature/)
- [DEV: Git Notes Unraveled](https://dev.to/shrsv/git-notes-unraveled-history-mechanics-and-practical-uses-25i9)
