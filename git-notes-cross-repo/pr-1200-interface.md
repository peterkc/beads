# PR #1200: IssueProvider Interface Analysis

**PR**: [#1200 - Fix --db flag in orphan detection](https://github.com/steveyegge/beads/pull/1200)
**Branch**: `fix/orphans-db-flag` (worktree at `.worktrees/orphans-db-flag`)

## Interface Design

PR #1200 introduced a clean abstraction:

```go
// internal/types/orphans.go
type IssueProvider interface {
    GetOpenIssues(ctx context.Context) ([]*Issue, error)
    GetIssuePrefix() string
}
```

## Function Signature Change

Before:
```go
func FindOrphanedIssues(path string) ([]OrphanIssue, error)
// path used for BOTH git scanning AND database location
```

After:
```go
func FindOrphanedIssues(gitPath string, provider IssueProvider) ([]OrphanIssue, error)
// gitPath: where to scan commits
// provider: where to get issues (decoupled!)
```

## CLI Wiring

```go
// cmd/bd/orphans.go

func getIssueProvider() (types.IssueProvider, func(), error) {
    if dbPath != "" {
        // --db flag: create provider from specified path
        provider, err := doctor.NewLocalProvider(dbPath)
        return provider, func() { provider.Close() }, nil
    }
    // Default: use global store
    provider := storage.NewStorageProvider(store)
    return provider, func() {}, nil
}

func findOrphanedIssues(path string) ([]orphanIssueOutput, error) {
    provider, cleanup, err := getIssueProvider()
    defer cleanup()

    orphans, err := doctorFindOrphanedIssues(path, provider)  // path = "."
    // ...
}
```

## Current Behavior

| Flag | Git Path | Issue Source |
|------|----------|--------------|
| None | `.` (cwd) | Auto-discovered `.beads/` |
| `--db /path` | `.` (cwd) | Specified database |

## Gap for Cross-Repo

The `path` parameter is hardcoded to `"."` in `orphans.go:39`. To fully support cross-repo from the beads side, would need:

```go
// Hypothetical --git-path flag
gitPath, _ := cmd.Flags().GetString("git-path")
if gitPath == "" {
    gitPath = "."
}
orphans, err := doctorFindOrphanedIssues(gitPath, provider)
```

## Workaround (Already Works)

Run from code repo with `--db` pointing to beads:

```bash
cd ~/my-code-repo
bd orphans --db ~/my-beads-repo/.beads/beads.db
```

This achieves cross-repo without any code changes.

## Implementations

| Provider | Location | Purpose |
|----------|----------|---------|
| `LocalProvider` | `cmd/bd/doctor/providers.go` | Direct SQLite access |
| `StorageProvider` | `internal/storage/provider.go` | Wraps existing Storage interface |
| Mock providers | Tests | Testing |
