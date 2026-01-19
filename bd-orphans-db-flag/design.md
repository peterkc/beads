# Design: bd orphans --db flag fix (Option D: Storage Interface)

## Current Architecture

```
orphans.go                    doctor/git.go
───────────                   ─────────────
orphansCmd.Run()
  │
  └─► findOrphanedIssues(".")
        │
        └─► doctor.FindOrphanedIssues(path)
              │
              ├─► beadsDir = filepath.Join(path, ".beads")  // HARDCODED!
              ├─► dbPath = filepath.Join(beadsDir, "beads.db")
              ├─► Open own DB connection
              └─► Query issues, scan git log
```

**Problems:**
1. `FindOrphanedIssues()` hardcodes database path
2. Opens its own DB connection (doesn't use shared storage layer)
3. Can't work with remote/RPC sources

## Proposed Fix: Option D (Storage Interface)

### New Interface

```go
// In internal/types/orphans.go or internal/storage/provider.go

// IssueProvider is a minimal interface for orphan detection.
// Allows any storage backend (SQLite, RPC, JSONL) to be used.
type IssueProvider interface {
    // GetOpenIssues returns all issues with status open or in_progress
    GetOpenIssues(ctx context.Context) ([]*Issue, error)

    // GetIssuePrefix returns the configured issue prefix (e.g., "bd", "TEST")
    GetIssuePrefix() string
}
```

### Updated Architecture

```
orphans.go                    doctor/git.go
───────────                   ─────────────
orphansCmd.Run()
  │
  ├─► provider := getIssueProvider()  // NEW: Uses global store or --db
  │
  └─► findOrphanedIssues(".", provider)
        │
        └─► doctor.FindOrphanedIssues(gitPath, provider)
              │
              ├─► prefix := provider.GetIssuePrefix()
              ├─► issues := provider.GetOpenIssues(ctx)
              └─► Scan git log for orphans
```

### Implementation Locations

| File | Change | Description |
|------|--------|-------------|
| `internal/types/orphans.go` | **New file** | Define `IssueProvider` interface |
| `internal/storage/sqlite/provider.go` | **New method** | Implement `IssueProvider` on SQLite storage |
| `cmd/bd/doctor/git.go` | **Modify** | Accept `IssueProvider` instead of `path` |
| `cmd/bd/orphans.go` | **Modify** | Build provider from `--db` flag or global store |

### Provider Resolution Logic

```go
// In orphans.go
func getIssueProvider() (types.IssueProvider, error) {
    // If --db flag is set, open a read-only connection to that DB
    if dbPath != "" {
        return sqlite.NewReadOnlyProvider(dbPath)
    }

    // Otherwise, use the global store (already opened in PersistentPreRun)
    // Store already implements IssueProvider
    return store, nil
}
```

### Why Storage Interface?

| Benefit | Explanation |
|---------|-------------|
| **Flexibility** | Works with SQLite, RPC daemon, JSONL, or mock providers |
| **Testability** | Easy to inject mock provider for unit tests |
| **Separation** | Doctor package doesn't know about CLI flags |
| **Future-proof** | Enables daemon-backed cross-repo detection |
| **Consistency** | Aligns with how other commands use storage layer |

## Call Site Analysis

### Primary Symbol: `FindOrphanedIssues`

| Location | Line | Role | Change Required |
|----------|------|------|-----------------|
| `cmd/bd/doctor/git.go` | 778 | **Definition** | Add `provider IssueProvider` param |
| `cmd/bd/doctor/git.go` | 940 | Caller (CheckOrphanedIssues) | Pass provider from path |
| `cmd/bd/orphans.go` | 15 | Variable assignment | Update signature |
| `cmd/bd/orphans.go` | 108 | Actual call | Pass provider |

### Secondary Symbol: `doctorFindOrphanedIssues`

| Location | Line | Role | Change Required |
|----------|------|------|-----------------|
| `cmd/bd/orphans.go` | 15 | Variable definition | `func(string, IssueProvider)` |
| `cmd/bd/orphans_test.go` | 12-13 | Mock setup | Update mock signature |
| `cmd/bd/orphans_test.go` | 44-45 | Error test mock | Update mock signature |

### Related Symbols

| Symbol | File | Impact |
|--------|------|--------|
| `OrphanIssue` struct | `cmd/bd/doctor/git.go` | No change (output type) |
| `CheckOrphanedIssues` | `cmd/bd/doctor/git.go` | Must build provider internally |
| `orphanIssueOutput` | `cmd/bd/orphans.go` | No change (CLI output) |

## Semantic References

```yaml
semantic:
  - pattern: "type IssueProvider interface"
    action: create new interface
    file: internal/types/orphans.go (NEW)
  - function: FindOrphanedIssues
    file: cmd/bd/doctor/git.go:778
    action: change signature (path, provider)
  - function: CheckOrphanedIssues
    file: cmd/bd/doctor/git.go:902
    action: build provider from path, pass to FindOrphanedIssues
  - variable: doctorFindOrphanedIssues
    file: cmd/bd/orphans.go:15
    action: update type signature
  - function: getIssueProvider
    file: cmd/bd/orphans.go (NEW)
    action: create - resolve --db flag to provider
```

## Risk Analysis

| Risk | Likelihood | Impact | Severity | Mitigation | Rollback |
|------|------------|--------|----------|------------|----------|
| Breaking signature | High | Low | **Medium** | Update all callers atomically in Phase 1 | Revert Phase 1 commit |
| Upstream rejection | Medium | Medium | **Medium** | Show testability benefits; keep changes minimal | Keep fork, propose alternative |
| Scope creep | Low | Low | **Low** | Interface is minimal (2 methods); no new features | N/A |
| Test complexity | Low | Low | **Low** | Interface makes testing easier, not harder | N/A |
| golangci-lint flags | Low | Low | **Low** | Follow existing interface patterns in codebase | Fix lint issues |
| Provider error handling | Medium | Low | **Low** | Return empty slice on error (matches existing behavior) | N/A |

## Alternatives Considered

### Option A: Check viper inside doctor

**Rejected**: Violates separation of concerns.

### Option C: Pass dbPath parameter

**Rejected**: Less flexible; still requires opening DB inside doctor.

### Option E: Multi-source scanning

**Deferred**: Useful but scope creep for this fix.

## Test Strategy

```go
// Mock provider for testing
type mockProvider struct {
    issues []*types.Issue
    prefix string
}

func (m *mockProvider) GetOpenIssues(ctx context.Context) ([]*types.Issue, error) {
    return m.issues, nil
}

func (m *mockProvider) GetIssuePrefix() string {
    return m.prefix
}

func TestFindOrphanedIssues_CrossRepo(t *testing.T) {
    provider := &mockProvider{
        issues: []*types.Issue{{ID: "TEST-001", Status: "open"}},
        prefix: "TEST",
    }

    // Create git repo with commit referencing TEST-001
    gitPath := setupGitWithCommit(t, "TEST-001")

    orphans, err := doctor.FindOrphanedIssues(gitPath, provider)
    require.NoError(t, err)
    assert.Len(t, orphans, 1)
    assert.Equal(t, "TEST-001", orphans[0].IssueID)
}
```

## Breaking Changes

### Internal API Changes

| Change | Impact | Migration |
|--------|--------|-----------|
| `FindOrphanedIssues(path)` → `FindOrphanedIssues(gitPath, provider)` | All callers must update | Add provider parameter |
| `doctorFindOrphanedIssues` type | Test mocks must update | Update mock signatures |

### External API Changes

| Change | Impact | Migration |
|--------|--------|-----------|
| **None** | End user CLI unchanged | N/A |

**Note**: This is an internal refactor. The `bd orphans` command interface is unchanged for end users.

### Backward Compatibility Wrapper (Optional)

If needed, provide a convenience function for simple cases:

```go
// FindOrphanedIssuesFromPath is for callers that don't need custom providers.
// It creates a local provider from the given path's .beads/ directory.
func FindOrphanedIssuesFromPath(gitPath string) ([]OrphanIssue, error) {
    provider, err := NewLocalProvider(filepath.Join(gitPath, ".beads"))
    if err != nil {
        return nil, err
    }
    return FindOrphanedIssues(gitPath, provider)
}
```

## Phase Atomicity Analysis

| Phase | Deliverable | Atomic? | INVEST Check | Build Status |
|-------|-------------|---------|--------------|--------------|
| **Phase 1: Tracer + Callers** | Working cross-repo detection with all callers | ✅ | End-to-end, valuable, build stays green | ✅ Green |
| **Phase 2: Tests** | Comprehensive test coverage | ✅ | Independent, additive | ✅ Green |
| **Phase 3: Closing** | PR merged upstream | ✅ | Final delivery | ✅ Green |

**Critical Design Decision:** Phase 1 includes ALL caller updates to ensure the build never breaks between phases. This addresses the Gemini reviewer's concern about broken builds.

**Rollback Points:**
- After Phase 1: Revert interface and all caller changes together (single commit)
- After Phase 2: Tests are additive, easy to remove
- Phase 3: PR not merged = no impact on upstream

## Migration Path

1. **Phase 1**: Add `IssueProvider` interface, update `FindOrphanedIssues` signature
2. **Phase 2**: Update `CheckOrphanedIssues` and other callers to pass provider
3. **Phase 3**: Add comprehensive tests with mock provider
4. **Phase 4**: Create PR, address feedback, merge
