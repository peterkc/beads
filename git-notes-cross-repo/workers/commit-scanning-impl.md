# Worker 3: Commit Scanning Implementation

*Research worker output - preserved for reference*

## Implementation Patterns

### Pattern 1: go-git In-Memory (Recommended for Beads)

```go
import (
    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/storage/memory"
)

// Clone into memory - no filesystem I/O
r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
    URL:        "https://github.com/owner/repo",
    NoCheckout: true,
})

// Iterate commits
cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
```

### Pattern 2: Multi-Remote git log (Existing Setup)

```bash
git remote add code-repo https://github.com/owner/code
git fetch --all
git log --all --oneline | grep "(bd-"
```

### Pattern 3: Shallow Clone + Scan (Large Repos)

```bash
git clone --depth=1 --filter=blob:none <url>
git log --all --oneline --since="2026-01-01"
```

**Performance**: 70-90% reduction in data transfer

## Decision Matrix

| Requirement | Recommended Pattern |
|-------------|---------------------|
| No filesystem access | go-git in-memory |
| Large repos (>1GB) | Shallow + sparse |
| Existing local repos | Multi-remote git log |
| GitHub-only | REST API |

## For `--git-path` Implementation

Simplest approach: Pass path to existing `git log` command
```go
cmd := exec.Command("git", "log", "--oneline", "--all")
cmd.Dir = gitPath  // Use provided path instead of "."
```

## Sources

- [go-git documentation](https://pkg.go.dev/github.com/go-git/go-git/v5)
- [pygit2 repository docs](https://www.pygit2.org/repository.html)
- [GitHub REST API commits](https://docs.github.com/en/rest/commits)
