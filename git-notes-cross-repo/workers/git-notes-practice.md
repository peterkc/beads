# Worker 1: Git Notes in Practice

*Research worker output - preserved for reference*

## Executive Summary

Git notes remain a powerful but underutilized feature for attaching metadata to commits without modifying commit objects. Three major usage patterns emerged: **mailing list linking** (Git project), **distributed code review** (git-appraise), and **preserving PR discussions** (Symfony).

## Key Examples

### git-appraise (Google) - Distributed Code Review

**Namespace Architecture:**
```
refs/notes/devtools/reviews    # Review requests
refs/notes/devtools/discuss    # Human review comments
refs/notes/devtools/ci         # CI build/test results
refs/notes/devtools/analyses   # Static analysis/robot comments
```

**Data Format:**
```json
{"timestamp":"1234567890","author":"user@example.com","v":0,"review":"lgtm"}
```

**Critical Pattern:** Single-line JSON + cat_sort_uniq = conflict-free merges

### Rebase Handling

```bash
git config notes.rewrite.amend true
git config notes.rewrite.rebase true
git config notes.rewriteRef refs/notes/commits
git config notes.rewriteMode cat_sort_uniq
```

### Sync Patterns

```bash
# Manual sync
git push origin refs/notes/devtools/reviews
git fetch origin '+refs/notes/*:refs/notes/*'

# Automated via .gitconfig
[remote "origin"]
    fetch = +refs/notes/*:refs/notes/*
```

## Why GitHub Disabled Notes (2014)

1. Performance: Linear scan doesn't scale for large repos
2. Low adoption: Limited usage didn't justify maintenance
3. Business: Competes with proprietary PR/issue system

## Sources

- [git-appraise](https://github.com/google/git-appraise)
- [Tyler Cipriani: Git Notes](https://tylercipriani.com/blog/2022/11/19/git-notes-gits-coolest-most-unloved-feature/)
- [Wouter J: Store Discussions in Git](https://wouterj.nl/2024/08/git-notes)
