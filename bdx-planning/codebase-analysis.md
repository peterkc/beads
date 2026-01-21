# Codebase Analysis

Analysis of beads v0 codebase structure for bdx planning.

## Test Coverage

| Location | Files | Notes |
|----------|-------|-------|
| `cmd/bd/` | 145 | CLI-level tests, good for characterization |
| `internal/storage/sqlite/` | 46 | Storage layer |
| `internal/rpc/` | 20 | RPC client/server |
| `cmd/bd/doctor/` | 20 | Doctor command |
| **Total** | **335** | Substantial coverage |

### Testing Approaches

1. **In-process** (`runBDInProcess`) — Calls `rootCmd.Execute` directly, fast
2. **End-to-end** (`exec.Command`) — Runs actual binary, slow but accurate
3. **Integration tag** — `//go:build integration` separates fast/slow

### Characterization Test Status

- No explicit "characterization" test suite
- CLI tests in `cmd/bd/*_test.go` serve similar purpose
- `cli_fast_test.go` has good patterns for in-process testing

## Package Structure

```
internal/
├── storage/          # Storage interface + SQLite impl
│   ├── storage.go    # Interface definition (~50 methods)
│   ├── sqlite/       # SQLite implementation
│   ├── memory/       # Memory implementation (testing)
│   └── dolt/         # Dolt implementation (experimental)
├── rpc/              # Daemon RPC (no interface)
│   ├── client.go     # RPC client
│   ├── server*.go    # RPC server (split across files)
│   └── protocol.go   # Request/response types
├── git/              # Git operations (no interface)
│   ├── worktree.go   # WorktreeManager
│   └── gitdir.go     # Git directory detection
├── types/            # Domain types
├── config/           # Configuration
└── ... (25+ more packages)
```

## Interface Status

| Package | Interface | Size | Extraction Effort |
|---------|-----------|------|-------------------|
| **storage** | ✅ `Storage` | ~50 methods | Already done (may need ISP split) |
| **git** | ❌ None | ~5 functions | Small — good tracer bullet |
| **rpc** | ❌ None | Large | Medium — client/server/protocol |
| **daemon** | ❌ None | Medium | Depends on RPC |

### Storage Interface (Existing)

Located in `internal/storage/storage.go`:

```go
type Storage interface {
    // Issues (~10 methods)
    CreateIssue(ctx, issue, actor) error
    GetIssue(ctx, id) (*Issue, error)
    UpdateIssue(ctx, id, updates, actor) error
    // ... more

    // Dependencies (~10 methods)
    AddDependency(ctx, dep, actor) error
    GetDependencies(ctx, issueID) ([]*Issue, error)
    // ... more

    // Labels, Events, Comments, Config, etc.
}

type Transaction interface {
    // Subset of Storage for atomic operations
}
```

**Observation**: Interface is large. Consider splitting per ISP:
- `IssueRepository`
- `DependencyRepository`
- `LabelRepository`
- `ConfigStore`

### Git Package (No Interface)

```go
// internal/git/worktree.go
type WorktreeManager struct {
    repoPath string
}

func (wm *WorktreeManager) CreateBeadsWorktree(branch, path string) error
func (wm *WorktreeManager) RemoveBeadsWorktree(path string) error
func (wm *WorktreeManager) CheckWorktreeHealth(path string) error
```

Uses `exec.Command` for git operations — easy to mock with interface.

### RPC Package (No Interface)

Concrete structs:
- `Client` — connects to daemon via Unix socket
- `Server` — handles requests, manages storage
- `Request`/`Response` — protocol types

## Dependencies Between Packages

```
cmd/bd → internal/rpc (client)
       → internal/storage (direct, for no-daemon mode)
       → internal/types
       → internal/config

internal/rpc/server → internal/storage
                    → internal/types

internal/storage/sqlite → internal/types
                        → internal/config
                        → internal/idgen
```

## Recommendations for Stage 1

### Tracer Bullet: Git Interface

**Why Git first:**
1. Small surface (~5 functions)
2. Isolated — no deps on storage/rpc
3. Easy to test (mock exec.Command)
4. Low risk, high learning

### Characterization Tests

Before any refactoring:
1. Run full test suite, capture baseline
2. Identify tests that verify CLI behavior
3. Mark as "characterization" (tag or directory)
4. These must pass throughout migration

### Interface Extraction Order

1. **Git** — Tracer bullet, prove pattern
2. **Storage** — Already has interface, maybe split
3. **RPC Client** — Decouple CLI from daemon
4. **RPC Server** — Last, most complex
