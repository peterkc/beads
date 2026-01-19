# Worker 2: Cross-Repo Tracking Alternatives

*Research worker output - preserved for reference*

## Comparison Table

| Approach | Mechanism | Pros | Cons |
|----------|-----------|------|------|
| **Commit Message Parsing + DB** | Scan for `PROJ-123` | Works across unlimited repos, query-friendly | Requires external service, delayed indexing |
| **Client-Side Hooks** | commit-msg validation | Works offline, enforces conventions | Per-repo setup, can be bypassed |
| **Webhook/Event-Driven** | GitHub webhooks | Real-time, centralized logic | Network dependency, complexity |
| **GitHub Actions** | Workflows on push | Native GitHub, version-controlled | Requires PAT for cross-repo |
| **Issue Aggregation** | GitHub Projects | Visual tracking, free | Limited to GitHub ecosystem |
| **Git Notes** | refs/notes namespace | No external deps, decentralized | Manual push, cross-repo queries complex |

## Key Insight

Most popular pattern: **Commit Message Parsing + Centralized Database**
- Used by Jira, Linear, GitHub (issue closing keywords)
- Requires convention: `fixes #123`, `PROJ-456`
- Database enables cross-repo queries

## Hybrid Approaches

Real tools combine multiple patterns:
- **Jira**: Parsing + webhooks + optional hooks
- **Linear**: Parsing + GitHub App for bidirectional sync
- **GitHub Projects**: Aggregation UI + parsing

## Recommendation for Beads

1. Keep current approach for local/offline tracking
2. Add optional `--git-path` for cross-repo scanning
3. Consider webhook export for external system integration

## Sources

- [Jira Git Integration](https://help.gitkraken.com/git-integration-for-jira-data-center/linking-git-commits-to-jira-issues-gij-self-managed/)
- [Linear GitHub Integration](https://linear.app/docs/github-integration)
- [GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/examples/multi-repo/issue-tracking/)
