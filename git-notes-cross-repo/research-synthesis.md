# Research Synthesis: Cross-Repo Orphan Detection

**Date**: 2026-01-19
**Workers**: 4 parallel research agents
**Scope**: Git notes, cross-repo alternatives, implementation patterns, CLI UX

---

## Executive Summary

Research reveals that **cross-repo commit-issue tracking is a solved problem** with multiple proven approaches. The key decision is **where to store the relationship data**:

| Storage Location | Examples | Trade-off |
|------------------|----------|-----------|
| Centralized DB | Jira, Linear | Fast queries, vendor dependency |
| Git notes | git-appraise | Decentralized, limited UI |
| Commit messages | GitHub keywords | Simple, no extra tooling |
| Beads `--db` | PR #1200 | Git-native, requires workflow flip |

**Recommendation**: Beads' current `--db` approach is valid. Enhance with `--git-path` for ergonomics, defer git notes to Phase 2.

---

## Key Findings by Topic

### 1. Git Notes in Practice

**Production Examples:**

| Project | Use Case | Namespace | Key Pattern |
|---------|----------|-----------|-------------|
| Git project | Mailing list links | `refs/notes/amlog` | Message-Id linking |
| git-appraise (Google) | Distributed code review | `refs/notes/devtools/*` | Single-line JSON |
| Symfony | PR discussion preservation | `refs/notes/github-comments` | API → notes |

**Critical Implementation Detail (git-appraise):**
```
Single-line JSON + cat_sort_uniq merge = conflict-free distributed notes
```

**Rebase Handling:**
```bash
git config notes.rewrite.rebase true
git config notes.rewriteRef refs/notes/commits
git config notes.rewriteMode cat_sort_uniq
```

**Why GitHub disabled notes display (2014):**
- Performance: Linear scan through notes tree doesn't scale
- Low adoption: Limited usage didn't justify maintenance
- Business: Competes with proprietary PR/issue system

### 2. Cross-Repo Tracking Approaches

**Industry Approaches Ranked by Popularity:**

1. **Commit message parsing + centralized DB** (Jira, Linear)
   - Most common, works across unlimited repos
   - Requires convention (`PROJ-123`, `fixes #123`)

2. **Webhook/event-driven** (GitHub Apps)
   - Real-time, integrates with CI/CD
   - Requires endpoint infrastructure

3. **GitHub Actions automation**
   - Native GitHub, version-controlled logic
   - Cross-repo requires PAT tokens

4. **Git notes** (git-appraise)
   - Fully decentralized, works offline
   - Limited UI, manual sync

**Key Insight**: Most tools **combine approaches** (e.g., Jira uses parsing + webhooks + hooks)

### 3. Implementation Patterns

**For Beads (Go codebase):**

| Pattern | When to Use | Complexity |
|---------|-------------|------------|
| Multi-remote git log | Existing local repos | Low |
| go-git in-memory | External repo scanning | Medium |
| Shallow clone + scan | Large repos, bandwidth limited | Low |
| GitHub API | GitHub-only, web tools | Medium |

**Recommended for `--git-path`:**
```go
// Current: hardcoded path
orphans, err := doctorFindOrphanedIssues(".", provider)

// Enhanced: configurable git path
gitPath := getGitPath(cmd)  // from --git-path flag or "."
orphans, err := doctorFindOrphanedIssues(gitPath, provider)
```

### 4. CLI UX Patterns

**Best Practices from Research:**

| Pattern | Example | When to Use |
|---------|---------|-------------|
| `--repo` flag | `gh issue list --repo org/repo` | One-off cross-repo ops |
| Context switching | `linear-cli ws switch team` | Frequent same-context ops |
| Directory-based | `cd .worktrees/feature` | Parallel workflows |
| Config file defaults | `.linear.toml` | Per-repo customization |

**Anti-Patterns to Avoid:**
- Implicit context without indication
- Ambiguous error messages (missing repo context)
- Mixed metaphors (some commands use flags, others require cd)

**Recommended for Beads:**
```bash
# Pattern 1: --repo flag (explicit)
bd orphans --repo ~/my-beads-repo

# Pattern 2: --git-path flag (scan external)
bd orphans --git-path ~/my-code-repo

# Pattern 3: Current --db (already works)
cd ~/my-code-repo && bd orphans --db ~/my-beads-repo/.beads
```

---

## Architectural Decision Points

### Decision 1: Where to store cross-repo links?

| Option | Beads Fit | Effort | UX |
|--------|-----------|--------|-----|
| A. Issue metadata (ExternalRef) | ✓ exists | Low | Manual linking |
| B. Git notes in code repo | New | High | Auto-discovery |
| C. Dependency edges | ✓ exists | Low | Graph queries |
| D. Scan on demand (`--git-path`) | New flag | Low | Explicit |

**Recommendation**: Option D (scan on demand) for MVP, Option B (git notes) for Phase 2

### Decision 2: CLI ergonomics

| Approach | Command | Pros | Cons |
|----------|---------|------|------|
| Flip perspective | `cd code && bd orphans --db beads` | Already works | Unintuitive |
| Explicit git path | `bd orphans --git-path code` | Clear intent | New flag |
| Config-based | `.beads/config.yaml: code_repo: path` | Set once | Config complexity |

**Recommendation**: Add `--git-path` flag (low effort, clear UX)

---

## Implementation Roadmap

### Phase 1: Document + Minor Enhancement (Low effort)
1. ✅ Document `--db` workflow in docs
2. Add `--git-path` flag to `bd orphans`
3. Add examples to `bd orphans --help`

### Phase 2: Git Notes Support (Medium effort)
1. `bd notes add <issue-id>` — Annotate HEAD
2. `bd notes list` — Show annotations
3. `bd notes push/fetch` — Sync refs/notes/beads
4. `bd orphans --include-notes` — Enhanced scanning

### Phase 3: Advanced Features (Future)
1. Auto-create tracking issues (GitHub Agentic Workflows pattern)
2. Webhook export for external tracking systems
3. Cross-repo dependency visualization

---

## Sources

### Git Notes
- [git-appraise (Google)](https://github.com/google/git-appraise)
- [Tyler Cipriani: Git Notes](https://tylercipriani.com/blog/2022/11/19/git-notes-gits-coolest-most-unloved-feature/)
- [Wouter J: Store Discussions in Git](https://wouterj.nl/2024/08/git-notes)

### Cross-Repo Tracking
- [GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/examples/multi-repo/issue-tracking/)
- [Jira Git Integration](https://help.gitkraken.com/git-integration-for-jira-data-center/linking-git-commits-to-jira-issues-gij-self-managed/)
- [Linear GitHub Integration](https://linear.app/docs/github-integration)

### Implementation
- [go-git documentation](https://pkg.go.dev/github.com/go-git/go-git/v5)
- [pygit2 repository docs](https://www.pygit2.org/repository.html)
- [GitHub REST API commits](https://docs.github.com/en/rest/commits)

### CLI UX
- [GitHub CLI Manual](https://cli.github.com/manual/)
- [UX Patterns for CLI Tools](https://lucasfcosta.com/2022/06/01/ux-patterns-cli-tools.html)
- [Meta CLI](https://github.com/mateodelnorte/meta)
