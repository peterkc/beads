# Beads Specs

This directory is an **orphan branch** (`specs`) in peterkc/beads, not part of the main codebase.

## Purpose

All specs for beads contributions are stored here. Specs define feature implementations, bug fixes, and enhancements before coding begins.

## Git Context

```
Branch: specs (orphan)
Remote: peterkc/beads
```

**Important**: This is a separate git history from `main`. Commits here don't appear in the main branch and vice versa.

## Workflow

1. Create specs here using `/spec:create`
2. Execute specs from any worktree using `/spec:run`
3. Specs reference the target branch/worktree for implementation

## Session Close

Standard git workflow (no beads sync needed):

```bash
git add <files>
git commit -m "..."
git push origin specs
```
