# Requirements: bd orphans --db flag fix (Option D: Storage Interface)

## Functional Requirements

### FR-001: IssueProvider Interface

THE SYSTEM SHALL define an `IssueProvider` interface with the following methods:
- `GetOpenIssues(ctx context.Context) ([]*Issue, error)` - Returns issues with status open or in_progress
- `GetIssuePrefix() string` - Returns the configured issue prefix

### FR-002: Interface-Based Orphan Detection

WHEN `FindOrphanedIssues()` is called with an `IssueProvider`
THE SYSTEM SHALL use the provider to obtain issues and prefix instead of opening its own database connection.

### FR-003: Provider Resolution

WHEN the user runs `bd --db /path orphans`
THE SYSTEM SHALL create a provider backed by the specified database path.

WHEN the user runs `bd orphans` without `--db` flag
THE SYSTEM SHALL use the global store as the provider.

### FR-004: Backward Compatibility

WHEN no `--db` flag is specified
THE SYSTEM SHALL maintain existing behavior by using the default local `.beads/` storage.

### FR-005: Git Log Scanning

WHEN scanning git log for orphaned issues
THE SYSTEM SHALL use the prefix from `provider.GetIssuePrefix()` to build the regex pattern.

## Non-Functional Requirements

### NFR-001: Interface Minimalism

THE SYSTEM SHALL define a minimal interface with only 2 methods required for orphan detection.

**Validation**: Interface has exactly GetOpenIssues() and GetIssuePrefix().

### NFR-002: Testability

THE SYSTEM SHALL allow mock providers to be injected for testing.

**Validation**: Cross-repo tests use mock provider without real database.

### NFR-003: No Breaking Public API

THE SYSTEM SHALL update internal signatures without breaking external callers.

**Validation**: `bd orphans` command behavior unchanged for end users.

### NFR-004: Reasonable Code Change

THE SYSTEM SHALL implement the fix with minimal invasive changes.

**Validation**: Change set is under 150 lines (excluding tests).

## Test Matrix

### Unit Tests (Mock Provider)

| ID | Scenario | Provider | Git State | Expected Result |
|----|----------|----------|-----------|-----------------|
| UT-01 | Basic orphan detection | Mock: 1 open issue | Commit refs issue | 1 orphan found |
| UT-02 | No orphans | Mock: 1 open issue | No matching commits | 0 orphans |
| UT-03 | Custom prefix | Mock: prefix="TEST" | Commit refs TEST-001 | Finds TEST-001 |
| UT-04 | Multiple orphans | Mock: 3 open issues | Commits ref 2 of them | 2 orphans found |
| UT-05 | Closed issues ignored | Mock: 1 closed issue | Commit refs it | 0 orphans (closed) |
| UT-06 | In-progress included | Mock: 1 in_progress | Commit refs it | 1 orphan found |
| UT-07 | Provider error | Mock: returns error | Any | Empty slice, no error |
| UT-08 | Empty provider | Mock: no issues | Valid git | Empty slice |
| UT-09 | Hierarchical IDs | Mock: issue "bd-abc.1" | Commit refs bd-abc.1 | Finds hierarchical |

### Integration Tests (Real Storage)

| ID | Scenario | Setup | Action | Expected |
|----|----------|-------|--------|----------|
| IT-01 | Local .beads/ | Init bd in dir A | `bd orphans` in A | Uses local DB |
| IT-02 | Cross-repo --db | DB in dir A, git in B | `bd --db A/.beads/var/beads.db orphans` in B | Uses A's DB |
| IT-03 | Global store fallback | No --db flag | `bd orphans` | Uses PersistentPreRun store |
| IT-04 | Non-git directory | Valid DB | Run in non-git dir | Empty slice, no crash |

### Regression Tests (Existing Behavior)

| ID | Scenario | Before Fix | After Fix |
|----|----------|------------|-----------|
| RT-01 | `bd orphans` no flags | Works | Still works |
| RT-02 | `bd doctor` orphan check | Works | Still works |
| RT-03 | Existing test mocks | Pass | Pass (updated signature) |

### Edge Cases

| ID | Scenario | Input | Expected |
|----|----------|-------|----------|
| EC-01 | Very long prefix | 50-char prefix | Handles correctly |
| EC-02 | Unicode in commit | Commit with emoji | Parses correctly |
| EC-03 | Large git history | 10k+ commits | Completes in <5s |
| EC-04 | Malformed issue ID | "bd-!!!" in commit | Ignored (no match) |

## Interface Contract

```go
// IssueProvider abstracts issue storage for orphan detection.
// Implementations may be backed by SQLite, RPC, JSONL, or mocks.
type IssueProvider interface {
    // GetOpenIssues returns issues that are open or in_progress.
    // Should return empty slice (not error) if no issues exist.
    GetOpenIssues(ctx context.Context) ([]*Issue, error)

    // GetIssuePrefix returns the configured prefix (e.g., "bd", "TEST").
    // Should return "bd" as default if not configured.
    GetIssuePrefix() string
}
```
