# Worker 4: CLI UX Patterns for Cross-Repo

*Research worker output - preserved for reference*

## Pattern Catalog

### Pattern 1: Explicit `--repo` Flag (GitHub CLI)

```bash
gh issue list --repo OWNER/REPO
gh pr create --repo organization/other-repo
```

**When to use**: One-off operations, scripting

### Pattern 2: Context Switching (Linear CLI)

```bash
linear-cli ws switch personal
linear-cli ws current
```

**When to use**: Frequent operations in same workspace

### Pattern 3: Directory-Based (git worktree)

```bash
cd .worktrees/feature-branch
# All commands now operate on that context
```

**When to use**: Parallel workflows, AI agents

### Pattern 4: Meta-Repository (Meta CLI)

```bash
meta git status  # Runs across all child repos
meta exec <cmd>  # Execute in all repos
```

**When to use**: Coordinated multi-repo changes

## Anti-Patterns

| Anti-Pattern | Better Alternative |
|--------------|-------------------|
| Implicit context without indication | Show context in prompt/output |
| Require cd for every operation | Support `--repo` flag |
| No config file for defaults | Per-repo config (`.linear.toml`) |
| Ambiguous error messages | Include repo context in errors |

## Recommendation for Beads

```bash
# Option A: --git-path (scan external code repo)
bd orphans --git-path ~/my-code-repo

# Option B: Current --db (flip perspective)
cd ~/my-code-repo && bd orphans --db ~/my-beads-repo/.beads

# Both should work, --git-path is more intuitive
```

**Key principles:**
1. Explicit > Implicit
2. Show context in output
3. Support both flags and config defaults
4. Clear error messages with repo context

## Sources

- [GitHub CLI Manual](https://cli.github.com/manual/)
- [UX Patterns for CLI Tools](https://lucasfcosta.com/2022/06/01/ux-patterns-cli-tools.html)
- [Meta CLI](https://github.com/mateodelnorte/meta)
