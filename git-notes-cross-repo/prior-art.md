# Prior Art: Cross-Repo Issue Tracking

**Research Date**: 2026-01-19

## Industry Approaches

### 1. GitHub Native References

**Mechanism**: `OWNER/REPO#ISSUE-NUMBER` in commit messages

| Feature | Same Repo | Cross Repo |
|---------|-----------|------------|
| Auto-link | Yes | Yes |
| Auto-close (`Fixes #123`) | Yes | **No** |
| Visibility | High | Medium |

**Limitation**: Keywords like `Fixes`, `Resolves`, `Closes` only auto-close if commit is in same repo.

**Source**: [Link GitHub Commit to Issue](https://gitdailies.com/articles/link-github-commit-to-issue/)

### 2. GitHub Projects (Centralized Board)

Aggregates issues, PRs, and notes from multiple repositories onto a single board.

**Use case**: Organizations needing unified view across component repos.

**Limitation**: Issues still belong to one repo—Projects just provide a view layer.

**Source**: [GitHub Community Discussion #6433](https://github.com/orgs/community/discussions/6433)

### 3. GitHub Agentic Workflows

Automated approach: when issues are created in component repos, tracking issues are auto-created in a central repository.

**Use case**:
- Component-based architectures
- Tracking external dependencies
- Coordinating cross-project initiatives
- Aggregating metrics from distributed repos

**Source**: [GitHub Agentic Workflows - Cross-Repo Tracking](https://githubnext.github.io/gh-aw/examples/multi-repo/issue-tracking/)

### 4. GitLab Crosslinking

Creates relationships between issues through:
- Commit messages
- Branch names (starting with issue number)
- MR descriptions

**Key feature**: Branch naming convention `123-feature-name` auto-links issue and MR.

**Source**: [GitLab Crosslinking Issues](https://docs.gitlab.com/ee/user/project/issues/crosslinking_issues.html)

### 5. Gitea External References

Supports `owner/repository#1234` format for cross-repo references.

**Key feature**: Uses `!` marker to distinguish PRs from issues when using external trackers.

**Source**: [Gitea Automatic References](https://docs.gitea.com/usage/automatically-linked-references)

### 6. Dedicated Issue Trackers

Many projects use external tools with native cross-repo support:

| Tool | Used By |
|------|---------|
| Jira | Open edX, Swift, OpenMRS |
| YouTrack | Kotlin |
| Linear | Many startups |
| Trac | Django |

**Advantage**: Purpose-built for cross-project tracking, dependency graphs.

### 7. Separate Issue Repository Pattern

Some teams maintain a dedicated repo purely for issue tracking:

> "Sometimes issues need to be tracked across projects too - in such cases, you could use another repository that is only used for issue tracking."

**Trade-off**: Discoverability suffers—GitHub search across repos is inferior to single-repo search.

**Source**: [Kinsta - Monorepo vs Multi-Repo](https://kinsta.com/blog/monorepo-vs-multi-repo/)

## Patterns Relevant to Beads

| Pattern | Beads Equivalent | Gap |
|---------|------------------|-----|
| Central issue repo | `BEADS_DIR` pointing to separate repo | Orphan detection broken |
| `OWNER/REPO#123` format | `(bd-xxx)` in commit messages | Not cross-repo aware |
| Auto-created tracking issues | N/A | Could be a feature |
| Branch naming convention | N/A | Could auto-link via hooks |

## Key Insight

Most platforms solve cross-repo tracking at the **view layer** (Projects, dashboards) rather than the **data layer** (commit-issue binding).

Git notes would solve it at the **commit layer**—attaching metadata directly to commits without changing SHAs.

Beads' `--db` approach solves it at the **scan layer**—telling the scanner where to look for commits vs issues.

## Comparison Matrix

| Approach | Data Location | Discovery | Retroactive | Rebase-Safe |
|----------|---------------|-----------|-------------|-------------|
| GitHub Projects | View layer | Good | Yes | N/A |
| Git notes | Commit metadata | Poor (no UI) | Yes | Configurable |
| Beads `--db` | Separate scan | Manual | Yes | N/A |
| External trackers | Centralized DB | Good | Yes | N/A |

## Recommendations for Beads

1. **Short-term**: Document `--db` workflow (already works)
2. **Medium-term**: Consider `--git-path` for ergonomics
3. **Long-term**: Evaluate git notes for bidirectional linking
4. **Alternative**: Auto-create tracking issues in beads repo from code repo commits (GitHub Agentic Workflows pattern)
