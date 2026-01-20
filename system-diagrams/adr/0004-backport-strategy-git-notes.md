# ADR 0004: Backport Strategy with Git Notes

## Status

Accepted

## Context

With parallel development on `main` (bd) and `next` (bdx), we need a strategy to:

1. Backport improvements from `next` → `main` (bdx discoveries benefit bd users)
2. Track which commits have been backported (avoid duplicates, maintain audit trail)
3. Preserve commit context across branches (why was this backported?)
4. Ease eventual merge by reducing branch divergence

## Decision

**Use git notes to track backport relationships between branches.**

### Backport Workflow

```bash
# 1. Cherry-pick from next to main
git checkout main
git cherry-pick <next-commit-sha>

# 2. Add note to new commit
git notes add -m "backport-from: <next-commit-sha> (next)"

# 3. Optionally note the original
git notes add <next-commit-sha> -m "backported-to: <new-sha> (main)"

# 4. Push notes to remote
git push origin refs/notes/*
```

### Note Format

```
backport-from: <sha> (<branch>)
backport-to: <sha> (<branch>)
backport-reason: <description>
backport-date: <ISO-8601>
```

**Example note:**
```
backport-from: abc1234 (next)
backport-reason: Row mapper DRY improvement benefits bd users
backport-date: 2026-01-19T10:30:00Z
```

### Automation Script

```bash
#!/usr/bin/env bash
# scripts/backport.sh - Cherry-pick with note tracking
set -euo pipefail

SOURCE_BRANCH="${1:-next}"
COMMIT_SHA="$2"
REASON="${3:-Backported from $SOURCE_BRANCH}"

# Validate
if [ -z "$COMMIT_SHA" ]; then
    echo "Usage: backport.sh <source-branch> <commit-sha> [reason]"
    exit 1
fi

# Cherry-pick
git checkout main
git cherry-pick "$COMMIT_SHA" --no-edit
NEW_SHA=$(git rev-parse HEAD)

# Add notes
git notes add -m "$(cat <<EOF
backport-from: $COMMIT_SHA ($SOURCE_BRANCH)
backport-reason: $REASON
backport-date: $(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF
)"

git notes add "$COMMIT_SHA" -m "backported-to: $NEW_SHA (main)"

# Push
git push origin main
git push origin refs/notes/*

echo "✅ Backported $COMMIT_SHA → $NEW_SHA"
```

### Query Commands

```bash
# Find all backported commits on main
git log --notes --grep="backport-from:" main

# Check if commit was already backported
git notes show <sha> 2>/dev/null | grep -q "backported-to:" && echo "Already backported"

# List all notes
git notes list

# Fetch notes from remote
git fetch origin refs/notes/*:refs/notes/*
```

## Backport Decision Matrix

| Change Type | Backport? | Reason |
|-------------|-----------|--------|
| Bug fixes | ✅ Yes | Users on bd need fixes |
| DRY refactors | ✅ Yes | Reduces future merge conflicts |
| Shared interfaces (ports/) | ✅ Yes | Foundation for both |
| v1-only adapters | ❌ No | bdx specific |
| Breaking changes | ❌ No | Would break bd |

## Convergence Effect

```
Backport Frequency    Final Merge Difficulty
─────────────────    ─────────────────────────
Never                🔴 Painful (massive diff)
Monthly              🟡 Moderate (batch conflicts)
Weekly               🟢 Easy (incremental)
Continuous           ✅ Trivial (minimal diff)
```

**Goal:** Backport frequently to minimize divergence.

## Shared Package Strategy

For code that should exist in both bd and bdx:

```
internal/
├── core/           # Shared (develop on main, merge to next)
│   ├── types/
│   ├── ports/      # v1 interfaces (additive)
│   └── util/
│
├── adapters/       # bdx only (v1 implementations)
└── storage/        # bd only (v0 implementations)
```

**Workflow for shared code:**
```bash
# Develop on main (benefits bd immediately)
git checkout main
git checkout -b feature/shared-improvement

# Implement in internal/core/
# ...

# Merge to main
git checkout main
git merge feature/shared-improvement

# Merge UP to next (not cherry-pick)
git checkout next
git merge main

# No backport note needed - it's a merge, not cherry-pick
```

## Consequences

### Positive

- Full audit trail of backport relationships
- Queryable history (`git log --notes --grep`)
- Prevents duplicate backports
- Context preserved without polluting commit messages
- Eases final merge by reducing divergence

### Negative

- Notes require explicit push/fetch (`refs/notes/*`)
- Notes don't auto-follow rebases (must re-attach)
- Extra step in workflow (mitigated by script)

### Mitigations

- Automation script handles notes automatically
- GitHub Action can verify notes are pushed
- Document rebase note recovery procedure

## Note Recovery After Rebase

If `next` is rebased, notes referencing old SHAs become orphaned:

```bash
# After rebase, find orphaned notes
git notes list | while read note_sha commit_sha; do
    if ! git cat-file -e "$commit_sha" 2>/dev/null; then
        echo "Orphaned note: $note_sha (was on $commit_sha)"
    fi
done

# Manual recovery: find new SHA, re-attach note
git notes copy <old-sha> <new-sha>
```

## References

- [Git Notes Documentation](https://git-scm.com/docs/git-notes)
- [ADR 0003: Migration Strategy](0003-migration-strategy-strangler-fig.md)
- [research/git-notes-cross-repo/](../git-notes-cross-repo/) — Related exploration
