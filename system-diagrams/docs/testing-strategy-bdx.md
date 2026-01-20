# Testing Strategy: v0 vs bdx

## v0 Current Testing Approach

| Aspect | v0 Approach | Issue |
|--------|-------------|-------|
| Framework | Standard `testing` only | Verbose assertions |
| Assertions | Manual `if x != y { t.Errorf(...) }` | Boilerplate |
| Mocking | None (uses real SQLite) | Slow, coupled |
| Test Helpers | `internal/testutil/` | Custom, limited |
| Integration | Build tags (`//go:build integration`) | Good |
| BDD | None | No behavior specs |
| Property | None | Missing edge cases |

### v0 Test Example (Verbose)

```go
func TestCreateIssue(t *testing.T) {
    tmpDir := t.TempDir()
    store, err := sqlite.New(context.Background(), filepath.Join(tmpDir, "test.db"))
    if err != nil {
        t.Fatalf("failed to create store: %v", err)
    }
    defer store.Close()

    issue := &types.Issue{Title: "Test"}
    err = store.CreateIssue(context.Background(), issue, "actor")
    if err != nil {
        t.Fatalf("CreateIssue failed: %v", err)
    }

    got, err := store.GetIssue(context.Background(), issue.ID)
    if err != nil {
        t.Fatalf("GetIssue failed: %v", err)
    }
    if got.Title != "Test" {
        t.Errorf("expected title %q, got %q", "Test", got.Title)
    }
}
```

**Problems:**
- 20 lines for a simple test
- No clear arrange/act/assert
- Manual error checking boilerplate
- Real database (slow)

---

## bdx Recommended Testing Stack

### Framework Selection

| Layer | Tool | Why |
|-------|------|-----|
| **Assertions** | [testify/assert](https://github.com/stretchr/testify) | Clean assertions, 27% adoption |
| **Mocking** | [gomock](https://github.com/uber-go/mock) | Interface-based, auto-generated |
| **Table Tests** | Standard + testify | Native Go pattern |
| **Property** | [rapid](https://github.com/flyingmutant/rapid) | Modern, auto-minimization |
| **Concurrency** | `testing/synctest` (Go 1.25) | Virtualized time |
| **Integration** | Build tags | Standard pattern |
| **E2E** | Custom + real binary | Full system tests |

### go.mod Testing Dependencies

```go
require (
    github.com/stretchr/testify v1.9.0
    go.uber.org/mock v0.5.0
    pgregory.net/rapid v1.2.0
)
```

### Generate Mocks

```bash
# Install mockgen
go install go.uber.org/mock/mockgen@latest

# Generate mocks for all ports
go generate ./internal/ports/...
```

```go
// internal/ports/repositories/issue.go
//go:generate mockgen -source=issue.go -destination=../../mocks/issue_repo_mock.go -package=mocks

type IssueRepository interface {
    Create(ctx context.Context, issue *core.Issue) error
    Get(ctx context.Context, id string) (*core.Issue, error)
    Update(ctx context.Context, id string, updates core.IssueUpdates) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, filter core.IssueFilter) ([]*core.Issue, error)
}
```

---

## Testing Patterns by Layer

### 1. Core (Domain) — Pure Unit Tests

**No mocks needed** — core has no dependencies.

```go
// internal/core/issue/issue_test.go
package issue_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/steveyegge/beads/internal/core/issue"
)

func TestIssue_CanClose(t *testing.T) {
    tests := []struct {
        name     string
        status   issue.Status
        expected bool
    }{
        {"open can close", issue.StatusOpen, true},
        {"in_progress can close", issue.StatusInProgress, true},
        {"closed cannot close", issue.StatusClosed, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            i := &issue.Issue{Status: tt.status}
            assert.Equal(t, tt.expected, i.CanClose())
        })
    }
}
```

**Property-based test for domain invariants:**

```go
func TestIssue_IDFormat(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        title := rapid.String().Draw(t, "title")
        i := issue.New(title)

        // Property: ID always has correct prefix
        assert.True(t, strings.HasPrefix(i.ID, "beads-"))

        // Property: ID is always lowercase
        assert.Equal(t, strings.ToLower(i.ID), i.ID)
    })
}
```

---

### 2. Use Cases — Mock Ports

```go
// internal/usecases/issue/create_test.go
package issue_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"

    "github.com/steveyegge/beads/internal/core/issue"
    "github.com/steveyegge/beads/internal/mocks"
    usecase "github.com/steveyegge/beads/internal/usecases/issue"
)

func TestCreateIssue(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockIssueRepository(ctrl)
    mockEvents := mocks.NewMockEventBus(ctrl)

    uc := usecase.NewCreateIssue(mockRepo, mockEvents)

    t.Run("success", func(t *testing.T) {
        // Arrange
        mockRepo.EXPECT().
            Create(gomock.Any(), gomock.Any()).
            Return(nil)

        mockEvents.EXPECT().
            Publish(gomock.Any(), "issue.created", gomock.Any()).
            Return(nil)

        // Act
        result, err := uc.Execute(context.Background(), usecase.CreateInput{
            Title: "Test Issue",
        })

        // Assert
        require.NoError(t, err)
        assert.NotEmpty(t, result.ID)
        assert.Equal(t, "Test Issue", result.Title)
    })

    t.Run("validation error", func(t *testing.T) {
        // Act
        _, err := uc.Execute(context.Background(), usecase.CreateInput{
            Title: "", // Empty title
        })

        // Assert
        assert.ErrorIs(t, err, usecase.ErrTitleRequired)
    })
}
```

---

### 3. Adapters — Integration Tests

```go
// internal/adapters/sqlite/issue_repo_test.go
//go:build integration

package sqlite_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/steveyegge/beads/internal/adapters/sqlite"
    "github.com/steveyegge/beads/internal/core/issue"
)

func TestIssueRepository_CRUD(t *testing.T) {
    // Use in-memory SQLite for speed
    db := sqlite.NewTestDB(t)
    repo := sqlite.NewIssueRepository(db)
    ctx := context.Background()

    t.Run("create and get", func(t *testing.T) {
        // Arrange
        i := &issue.Issue{Title: "Test"}

        // Act
        err := repo.Create(ctx, i)
        require.NoError(t, err)

        got, err := repo.Get(ctx, i.ID)
        require.NoError(t, err)

        // Assert
        assert.Equal(t, "Test", got.Title)
    })

    t.Run("update", func(t *testing.T) {
        // ... similar pattern
    })
}
```

---

### 4. Plugins — Behavioral Tests (Pseudo-BDD)

Using testify's suite for setup/teardown:

```go
// internal/plugins/core/create_test.go
package core_test

import (
    "bytes"
    "testing"

    "github.com/stretchr/testify/suite"

    "github.com/steveyegge/beads/internal/plugins"
    "github.com/steveyegge/beads/internal/plugins/core"
)

type CreateSuite struct {
    suite.Suite
    plugin  *core.Plugin
    ctx     *plugins.Context
    stdout  *bytes.Buffer
}

func (s *CreateSuite) SetupTest() {
    s.stdout = &bytes.Buffer{}
    s.ctx = plugins.NewTestContext(s.T())
    s.ctx.Stdout = s.stdout
    s.plugin = &core.Plugin{}
}

func (s *CreateSuite) TestCreate_WithTitle() {
    // Given a valid title
    args := []string{"--title", "My Issue"}

    // When I run create
    err := s.plugin.Create(s.ctx, args)

    // Then it succeeds
    s.NoError(err)

    // And outputs the issue ID
    s.Contains(s.stdout.String(), "Created: beads-")
}

func (s *CreateSuite) TestCreate_WithoutTitle_Fails() {
    // Given no title
    args := []string{}

    // When I run create
    err := s.plugin.Create(s.ctx, args)

    // Then it returns an error
    s.Error(err)
    s.Contains(err.Error(), "title required")
}

func TestCreateSuite(t *testing.T) {
    suite.Run(t, new(CreateSuite))
}
```

---

### 5. Event Bus — Concurrent Tests (Go 1.25)

```go
// internal/adapters/events/memory_test.go
package events_test

import (
    "context"
    "testing"
    "testing/synctest"

    "github.com/stretchr/testify/assert"

    "github.com/steveyegge/beads/internal/adapters/events"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
    synctest.Test(t, func(ctx context.Context) {
        bus := events.NewMemoryBus()
        received := make(chan string, 1)

        // Subscribe
        bus.Subscribe("issue.created", func(e events.Event) {
            received <- e.Payload.(string)
        })

        // Publish
        bus.Publish(ctx, "issue.created", "beads-123")

        // Wait for all goroutines to settle
        synctest.Wait()

        // Assert
        select {
        case id := <-received:
            assert.Equal(t, "beads-123", id)
        default:
            t.Error("event not received")
        }
    })
}
```

---

### 6. E2E Tests — Full Binary

```go
// e2e/create_test.go
//go:build e2e

package e2e_test

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestE2E_CreateListClose(t *testing.T) {
    // Setup: temp directory with .beads
    tmpDir := t.TempDir()
    beadsDir := filepath.Join(tmpDir, ".beads")
    require.NoError(t, os.MkdirAll(beadsDir, 0755))

    bdx := buildBinary(t)

    // Create
    out, err := exec.Command(bdx, "create", "--title", "E2E Test").
        Dir(tmpDir).
        CombinedOutput()
    require.NoError(t, err, string(out))
    assert.Contains(t, string(out), "Created: beads-")

    // List
    out, err = exec.Command(bdx, "list").
        Dir(tmpDir).
        CombinedOutput()
    require.NoError(t, err, string(out))
    assert.Contains(t, string(out), "E2E Test")

    // Extract ID and close
    id := extractID(string(out))
    out, err = exec.Command(bdx, "close", id).
        Dir(tmpDir).
        CombinedOutput()
    require.NoError(t, err, string(out))
}

func buildBinary(t *testing.T) string {
    t.Helper()
    binary := filepath.Join(t.TempDir(), "bdx")
    cmd := exec.Command("go", "build", "-o", binary, "./cmd/bdx")
    require.NoError(t, cmd.Run())
    return binary
}
```

---

## Test Organization

```
internal/
├── core/
│   └── issue/
│       ├── issue.go
│       └── issue_test.go           # Pure unit tests
│
├── ports/
│   └── repositories/
│       ├── issue.go
│       └── issue.go                # Interface (no tests needed)
│
├── mocks/                          # Generated by mockgen
│   ├── issue_repo_mock.go
│   └── event_bus_mock.go
│
├── adapters/
│   └── sqlite/
│       ├── issue_repo.go
│       └── issue_repo_test.go      # Integration tests (//go:build integration)
│
├── usecases/
│   └── issue/
│       ├── create.go
│       └── create_test.go          # Unit tests with mocks
│
└── plugins/
    └── core/
        ├── create.go
        └── create_test.go          # Behavioral tests

e2e/
├── create_test.go                  # E2E tests (//go:build e2e)
└── sync_test.go
```

---

## Running Tests

```bash
# Unit tests only (fast, no external deps)
go test ./internal/core/... ./internal/usecases/...

# Integration tests (requires SQLite)
go test -tags=integration ./internal/adapters/...

# All tests except E2E
go test ./...

# E2E tests (builds binary, slower)
go test -tags=e2e ./e2e/...

# With race detector
go test -race ./...

# Verbose with coverage
go test -v -cover ./...

# Property-based tests with more iterations
go test -rapid.checks=10000 ./internal/core/...
```

---

## Makefile Targets

```makefile
.PHONY: test test-unit test-integration test-e2e test-all

# Fast unit tests (default)
test: test-unit

test-unit:
	go test ./internal/core/... ./internal/usecases/... ./internal/plugins/...

test-integration:
	go test -tags=integration ./internal/adapters/...

test-e2e:
	go test -tags=e2e ./e2e/...

test-all: test-unit test-integration test-e2e

# Generate mocks
generate:
	go generate ./internal/ports/...

# Coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
```

---

## Framework Decision Matrix

| Criteria | stdlib | testify | gomock | ginkgo | rapid |
|----------|--------|---------|--------|--------|-------|
| Learning curve | ✅ None | ✅ Low | ⚠️ Medium | ❌ High | ⚠️ Medium |
| Adoption | 100% | 27% | 21% | 5% | Growing |
| Assertions | ❌ Verbose | ✅ Clean | N/A | ✅ Clean | ✅ Clean |
| Mocking | ❌ Manual | ⚠️ Basic | ✅ Excellent | ⚠️ Basic | N/A |
| BDD style | ❌ No | ⚠️ Suites | ❌ No | ✅ Full | ❌ No |
| Property tests | ❌ No | ❌ No | ❌ No | ❌ No | ✅ Yes |
| Dependencies | 0 | 1 | 1 | Many | 1 |

---

## Recommendation Summary

| Test Type | Tool | Example |
|-----------|------|---------|
| **Unit (domain)** | testify/assert + table tests | `assert.Equal(t, expected, actual)` |
| **Unit (use cases)** | gomock + testify | Mock repos, assert behavior |
| **Integration** | testify + real SQLite | `//go:build integration` |
| **Concurrent** | testing/synctest (Go 1.25) | Event bus, async operations |
| **Property** | rapid | Edge cases, invariants |
| **E2E** | testify + exec.Command | Full binary tests |

**NOT recommended for bdx:**
- **Ginkgo/Gomega** — Too complex for CLI tool, 5% adoption
- **goconvey** — Declining usage, web UI unnecessary
- **testing/quick** — Feature-frozen, use rapid instead

---

## References

- [Go Wiki: TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
- [Testify](https://github.com/stretchr/testify) — 27% adoption
- [GoMock](https://github.com/uber-go/mock) — 21% adoption
- [Rapid](https://github.com/flyingmutant/rapid) — Modern property testing
- [Go Testing Best Practices](https://bmuschko.com/blog/go-testing-frameworks/)
- [Go Ecosystem 2025](https://blog.jetbrains.com/go/2025/11/10/go-language-trends-ecosystem-2025/)
