# ADR-0001: Use IssueProvider Interface for Orphan Detection

## Status

Proposed

## Context

The `bd orphans` command ignores the `--db` flag because `FindOrphanedIssues()` in `cmd/bd/doctor/git.go` hardcodes the database path to `filepath.Join(path, ".beads")`.

This prevents cross-repo orphan detection where issues live in a separate planning repository.

### Call Sites Analysis

```
cmd/bd/doctor/git.go:778   FindOrphanedIssues(path string)     ← Definition
cmd/bd/doctor/git.go:940   FindOrphanedIssues(path)            ← Called by CheckOrphanedIssues
cmd/bd/orphans.go:15       doctorFindOrphanedIssues = doctor.FindOrphanedIssues  ← Variable
cmd/bd/orphans.go:108      doctorFindOrphanedIssues(path)      ← Actual call
cmd/bd/orphans_test.go:13  Mock via doctorFindOrphanedIssues   ← Test mock
```

## Decision

Introduce an `IssueProvider` interface that abstracts issue storage for orphan detection:

```go
type IssueProvider interface {
    GetOpenIssues(ctx context.Context) ([]*Issue, error)
    GetIssuePrefix() string
}
```

Change `FindOrphanedIssues` signature from:
```go
func FindOrphanedIssues(path string) ([]OrphanIssue, error)
```

To:
```go
func FindOrphanedIssues(gitPath string, provider IssueProvider) ([]OrphanIssue, error)
```

## Alternatives Considered

### Option A: Check viper inside doctor package

Add `viper.GetString("db")` inside `FindOrphanedIssues()`.

**Rejected**: Doctor package would depend on CLI configuration, violating separation of concerns.

### Option C: Pass dbPath as parameter

Change signature to `FindOrphanedIssues(path string, dbPath string)`.

**Rejected**: Still requires opening database inside doctor. Less flexible for testing and future RPC support.

### Option E: Multi-source scanning

Support multiple databases and git repos simultaneously.

**Deferred**: Useful but scope creep for this bug fix.

## Consequences

### Positive

- **Testability**: Mock providers enable clean unit tests
- **Flexibility**: Works with SQLite, RPC, JSONL, or any future backend
- **Separation**: Doctor package doesn't know about CLI flags
- **Future-proof**: Enables daemon-backed cross-repo detection

### Negative

- **Breaking change**: All callers must be updated
- **More code**: Interface definition + provider implementations
- **Migration burden**: Tests need updating to new signature

### Breaking Changes

| Location | Current | After Fix |
|----------|---------|-----------|
| `CheckOrphanedIssues()` | Calls `FindOrphanedIssues(path)` | Must pass provider |
| `doctorFindOrphanedIssues` | `func(string)` signature | `func(string, IssueProvider)` |
| Test mocks | Mock single-arg function | Mock two-arg function |

## Migration Strategy

1. Add `IssueProvider` interface to `internal/types/`
2. Update `FindOrphanedIssues` signature
3. Update `CheckOrphanedIssues` to build provider from path
4. Update `orphans.go` to build provider from `--db` flag or global store
5. Update test mocks to match new signature

## References

- GH#1196: Separate Beads Repo with Commit Correlation
- oss-zo5: Fix bd orphans ignores --db flag
