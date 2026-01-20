# Revised Migration Phases (Testing-First)

## Problem with Original Phases

The original ADR 0003 phases jumped straight into PLUGINIZE without:
1. Testing infrastructure to validate changes
2. Characterization tests to capture v0 behavior
3. Core domain layer (pure Go foundation)

**Risk:** We could break v0 compatibility without knowing it.

---

## Revised Three-Stage Migration

```
STAGE 0: FOUNDATION          STAGE 1: PLUGINIZE          STAGE 2: MODERNIZE
──────────────────           ──────────────────          ──────────────────

┌─────────────────┐          ┌─────────────────┐         ┌─────────────────┐
│ 0.1 Testing     │          │ 1.1 Plugin Infra│         │ 2.1 Adapters    │
│ 0.2 Char Tests  │    ───►  │ 1.2 Core Plugin │   ───►  │ 2.2 Use Cases   │
│ 0.3 Core Domain │          │ 1.3 Work Plugin │         │ 2.3 Wire Up     │
│ 0.4 Ports       │          │ 1.4 Sync Plugin │         │ 2.4 Validate    │
└─────────────────┘          │ 1.5 Other Plugs │         │ 2.5 Cleanup     │
                             │ 1.6 Wire Main   │         └─────────────────┘
Safety net +                 └─────────────────┘
contracts first              Wrap v0 in plugins          Replace internals
```

---

## Stage 0: FOUNDATION (New)

### Why Foundation First?

| Without Stage 0 | With Stage 0 |
|-----------------|--------------|
| No way to verify v0 compatibility | Characterization tests catch regressions |
| Tests written after code (error-prone) | Tests guide design (TDD) |
| Core mixed with infrastructure | Pure domain, easy to test |
| Ports undefined | Contracts locked before implementation |

---

### Phase 0.1: Testing Infrastructure

**Goal:** Set up modern testing stack before writing any code.

**Duration:** 1-2 days

**Files:**

```
internal/
├── testutil/
│   ├── fixtures.go        # Test data generators
│   ├── assertions.go      # Custom assertions
│   └── db.go              # In-memory SQLite helper
│
├── mocks/
│   └── .gitkeep           # Generated mocks go here
│
go.mod (additions):
    github.com/stretchr/testify v1.9.0
    go.uber.org/mock v0.5.0
    pgregory.net/rapid v1.2.0
```

**Deliverables:**
- [ ] go.mod updated with testing deps
- [ ] `testutil` package with fixtures
- [ ] Makefile targets: `test-unit`, `test-integration`, `test-e2e`
- [ ] CI workflow for tests

**Validation:**
```bash
go test ./internal/testutil/...  # Passes
make test-unit                   # Works
```

---

### Phase 0.2: Characterization Tests

**Goal:** Capture v0 behavior as executable specifications.

**Duration:** 3-5 days

**What are characterization tests?**
Tests that document *current* behavior (even bugs). They become the safety net.

```go
// characterization/create_test.go
//go:build characterization

package characterization_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/steveyegge/beads/internal/storage/sqlite"
    "github.com/steveyegge/beads/internal/types"
)

// TestCreate_V0Behavior captures current v0 create behavior
func TestCreate_V0Behavior(t *testing.T) {
    store := setupV0Store(t)
    ctx := context.Background()

    tests := []struct {
        name     string
        input    types.Issue
        wantErr  bool
        validate func(t *testing.T, got *types.Issue)
    }{
        {
            name:  "basic create",
            input: types.Issue{Title: "Test"},
            validate: func(t *testing.T, got *types.Issue) {
                assert.NotEmpty(t, got.ID)
                assert.Equal(t, "Test", got.Title)
                assert.Equal(t, "open", got.Status)
            },
        },
        {
            name:    "empty title allowed", // Document current behavior
            input:   types.Issue{Title: ""},
            wantErr: false, // v0 allows this (maybe a bug?)
        },
        {
            name:  "priority defaults to 2",
            input: types.Issue{Title: "Test"},
            validate: func(t *testing.T, got *types.Issue) {
                assert.Equal(t, 2, got.Priority)
            },
        },
        // ... capture ALL behaviors
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := store.CreateIssue(ctx, &tt.input, "test")
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)

            got, _ := store.GetIssue(ctx, tt.input.ID)
            if tt.validate != nil {
                tt.validate(t, got)
            }
        })
    }
}
```

**Files:**

```
characterization/
├── create_test.go         # Create behavior
├── list_test.go           # List/search behavior
├── update_test.go         # Update behavior
├── close_test.go          # Close behavior
├── dependency_test.go     # Dependency behavior
├── sync_test.go           # Sync behavior
├── helpers_test.go        # Shared setup
└── README.md              # Documents captured behaviors
```

**Deliverables:**
- [ ] Characterization tests for all core operations
- [ ] Edge cases documented (empty strings, nulls, etc.)
- [ ] Known quirks/bugs documented as tests
- [ ] All tests pass against v0

**Validation:**
```bash
go test -tags=characterization ./characterization/...  # 100% pass
```

---

### Phase 0.3: Core Domain Layer

**Goal:** Create pure Go domain with no dependencies.

**Duration:** 2-3 days

**Why pure domain first?**
- No mocks needed (pure functions)
- Fast tests (no I/O)
- Forces clean design
- Becomes foundation for everything else

**Files:**

```
internal/core/
├── issue/
│   ├── issue.go           # Issue entity
│   ├── issue_test.go      # Unit tests
│   ├── status.go          # Status enum
│   ├── status_test.go
│   ├── priority.go        # Priority enum
│   └── priority_test.go
│
├── dependency/
│   ├── dependency.go      # Dependency entity
│   ├── dependency_test.go
│   ├── graph.go           # Graph algorithms (cycle detection)
│   └── graph_test.go
│
├── label/
│   ├── label.go           # Label value object
│   └── label_test.go
│
├── comment/
│   ├── comment.go         # Comment entity
│   └── comment_test.go
│
└── events/
    ├── event.go           # Domain events
    ├── event_test.go
    └── types.go           # Event type enum
```

**Example:**

```go
// internal/core/issue/issue.go
package issue

import (
    "errors"
    "time"
)

var (
    ErrTitleRequired = errors.New("title is required")
    ErrAlreadyClosed = errors.New("issue already closed")
)

type Issue struct {
    ID          string
    Title       string
    Description string
    Status      Status
    Priority    Priority
    Labels      []string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ClosedAt    *time.Time
}

// CanClose returns true if the issue can be closed
func (i *Issue) CanClose() bool {
    return i.Status != StatusClosed
}

// Close marks the issue as closed
func (i *Issue) Close(reason string) error {
    if !i.CanClose() {
        return ErrAlreadyClosed
    }
    now := time.Now()
    i.Status = StatusClosed
    i.ClosedAt = &now
    i.UpdatedAt = now
    return nil
}

// Validate checks business rules
func (i *Issue) Validate() error {
    if i.Title == "" {
        return ErrTitleRequired
    }
    return nil
}
```

**Tests with property-based testing:**

```go
// internal/core/issue/issue_test.go
package issue_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "pgregory.net/rapid"

    "github.com/steveyegge/beads/internal/core/issue"
)

func TestIssue_Validate(t *testing.T) {
    t.Run("empty title fails", func(t *testing.T) {
        i := &issue.Issue{Title: ""}
        assert.ErrorIs(t, i.Validate(), issue.ErrTitleRequired)
    })

    t.Run("non-empty title passes", func(t *testing.T) {
        rapid.Check(t, func(t *rapid.T) {
            title := rapid.StringN(1, 100, 100).Draw(t, "title")
            i := &issue.Issue{Title: title}
            assert.NoError(t, i.Validate())
        })
    })
}

func TestIssue_Close(t *testing.T) {
    t.Run("open issue can close", func(t *testing.T) {
        i := &issue.Issue{Status: issue.StatusOpen}
        assert.True(t, i.CanClose())
        assert.NoError(t, i.Close("done"))
        assert.Equal(t, issue.StatusClosed, i.Status)
        assert.NotNil(t, i.ClosedAt)
    })

    t.Run("closed issue cannot close again", func(t *testing.T) {
        i := &issue.Issue{Status: issue.StatusClosed}
        assert.False(t, i.CanClose())
        assert.ErrorIs(t, i.Close("done"), issue.ErrAlreadyClosed)
    })
}
```

**Deliverables:**
- [ ] Core entities with business logic
- [ ] Unit tests for all domain rules
- [ ] Property-based tests for invariants
- [ ] Zero external dependencies in `core/`

**Validation:**
```bash
go test ./internal/core/...           # 100% pass
go list -m all | grep -v "core"       # No deps in core
```

---

### Phase 0.4: Ports (Interfaces)

**Goal:** Define contracts before implementation.

**Duration:** 1-2 days

**Files:**

```
internal/ports/
├── repositories/
│   ├── issue.go           # IssueRepository interface
│   ├── dependency.go      # DependencyRepository interface
│   ├── label.go           # LabelRepository interface
│   ├── comment.go         # CommentRepository interface
│   ├── config.go          # ConfigRepository interface
│   └── sync.go            # SyncRepository interface
│
├── services/
│   ├── linear.go          # LinearService interface
│   ├── git.go             # GitService interface
│   └── notify.go          # NotificationService interface
│
└── events/
    └── bus.go             # EventBus interface
```

**Example:**

```go
// internal/ports/repositories/issue.go
package repositories

//go:generate mockgen -source=issue.go -destination=../../mocks/issue_repo_mock.go -package=mocks

import (
    "context"

    "github.com/steveyegge/beads/internal/core/issue"
)

// IssueRepository defines the contract for issue persistence.
// Implementations: sqlite.IssueRepository, memory.IssueRepository
type IssueRepository interface {
    // Create persists a new issue. Returns error if ID already exists.
    Create(ctx context.Context, issue *issue.Issue) error

    // Get retrieves an issue by ID. Returns nil, nil if not found.
    Get(ctx context.Context, id string) (*issue.Issue, error)

    // Update applies changes to an existing issue.
    Update(ctx context.Context, id string, updates issue.Updates) error

    // Delete removes an issue. Returns error if not found.
    Delete(ctx context.Context, id string) error

    // Search finds issues matching the filter.
    Search(ctx context.Context, filter issue.Filter) ([]*issue.Issue, error)
}
```

**Generate mocks:**

```bash
go generate ./internal/ports/...
```

**Deliverables:**
- [ ] All port interfaces defined
- [ ] Generated mocks in `internal/mocks/`
- [ ] Interface contracts documented
- [ ] No implementations yet (just contracts)

**Validation:**
```bash
go generate ./internal/ports/...      # Mocks generate
go build ./internal/ports/...         # Compiles
ls internal/mocks/*.go                # Mocks exist
```

---

## Stage 0 Checkpoint

At this point:

```
✅ Testing infrastructure (testify, gomock, rapid)
✅ Characterization tests (v0 behavior captured)
✅ Core domain (pure Go, fully tested)
✅ Port interfaces (contracts defined)
✅ Generated mocks (ready for use case tests)
```

**What we have:**
- Safety net to catch regressions
- Clean domain layer to build on
- Contracts that won't change
- Mocks for fast tests

**What we DON'T have yet:**
- Plugin system (Stage 1)
- Adapter implementations (Stage 2)
- CLI wiring (Stage 1)

---

## Stage 1: PLUGINIZE (Adjusted)

Now we can safely wrap v0 with confidence:

### Phase 1.1: Plugin Infrastructure + Tests

Same as original, but now **with tests from day one**:

```go
// internal/plugins/registry_test.go
func TestRegistry_Execute(t *testing.T) {
    reg := plugins.NewRegistry()

    mockPlugin := &MockPlugin{
        commands: []plugins.Command{
            {Name: "test", Run: func(ctx *plugins.Context, args []string) error {
                return nil
            }},
        },
    }
    reg.Register(mockPlugin)

    err := reg.Execute("test", nil, nil)
    assert.NoError(t, err)
}

func TestRegistry_UnknownCommand(t *testing.T) {
    reg := plugins.NewRegistry()
    err := reg.Execute("unknown", nil, nil)
    assert.Error(t, err)
}
```

### Phase 1.2-1.6: Same as Original

But each phase includes:
1. Behavioral tests for the plugin
2. Characterization tests still pass (regression check)

---

## Stage 2: MODERNIZE (Adjusted)

### Phase 2.1: Adapters with Integration Tests

```go
// internal/adapters/sqlite/issue_repo_test.go
//go:build integration

func TestIssueRepository_Create(t *testing.T) {
    db := sqlite.NewTestDB(t)
    repo := sqlite.NewIssueRepository(db)
    ctx := context.Background()

    i := &issue.Issue{Title: "Test"}
    err := repo.Create(ctx, i)
    require.NoError(t, err)

    got, err := repo.Get(ctx, i.ID)
    require.NoError(t, err)
    assert.Equal(t, "Test", got.Title)
}
```

### Phase 2.2: Use Cases with Mocked Ports

```go
// internal/usecases/issue/create_test.go
func TestCreateIssue_Success(t *testing.T) {
    ctrl := gomock.NewController(t)
    mockRepo := mocks.NewMockIssueRepository(ctrl)
    mockEvents := mocks.NewMockEventBus(ctrl)

    mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
    mockEvents.EXPECT().Publish(gomock.Any(), "issue.created", gomock.Any())

    uc := usecase.NewCreateIssue(mockRepo, mockEvents)
    result, err := uc.Execute(ctx, usecase.CreateInput{Title: "Test"})

    require.NoError(t, err)
    assert.NotEmpty(t, result.ID)
}
```

### Phase 2.3-2.5: Same as Original

With tests validating each step.

---

## Revised Phase Summary

| Stage | Phase | Duration | Key Output |
|-------|-------|----------|------------|
| **0** | 0.1 Testing Infra | 1-2 days | testify, gomock, rapid |
| **0** | 0.2 Char Tests | 3-5 days | v0 behavior captured |
| **0** | 0.3 Core Domain | 2-3 days | Pure Go entities |
| **0** | 0.4 Ports | 1-2 days | Interfaces + mocks |
| **1** | 1.1-1.6 Plugins | 5-7 days | v0 wrapped in plugins |
| **2** | 2.1-2.5 Modernize | 7-10 days | v1 adapters + cleanup |

**Total:** ~20-30 days (vs original ~15-20 days)

**Trade-off:** +5-10 days upfront → much safer migration

---

## Test Pyramid for bdx

```
                    ╱╲
                   ╱  ╲
                  ╱ E2E╲           Few, slow, full system
                 ╱──────╲
                ╱        ╲
               ╱Integration╲       Adapters + real DB
              ╱────────────╲
             ╱              ╲
            ╱   Use Case     ╲     Mocked ports
           ╱──────────────────╲
          ╱                    ╲
         ╱    Core (Unit)       ╲  Pure, fast, many
        ╱────────────────────────╲
       ╱                          ╲
      ╱   Characterization (v0)    ╲  Safety net
     ╱──────────────────────────────╲
```

---

## Validation Gates

Each phase must pass before proceeding:

| Gate | Criteria |
|------|----------|
| **0.1** | `make test-unit` passes |
| **0.2** | Characterization tests 100% pass |
| **0.3** | Core tests 100% pass, 0 external deps |
| **0.4** | Mocks generate, interfaces compile |
| **1.x** | Plugin tests pass + char tests still pass |
| **2.x** | Integration tests pass + char tests still pass |

---

## Benefits of Testing-First

1. **Regressions caught immediately** — char tests fail if v0 behavior changes
2. **Confidence to refactor** — tests prove correctness
3. **Documentation** — tests document expected behavior
4. **Faster debugging** — small tests isolate issues
5. **Better design** — TDD forces clean interfaces
