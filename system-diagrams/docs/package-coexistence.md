# Package Coexistence: v0 and v1

## The Problem

The orphan branch approach breaks Strangler Fig:

```
next (orphan)                    main (v0)
─────────────                    ─────────
internal/                        internal/
├── core/        (v1)            ├── storage/     (v0)
├── plugins/     (v1)            ├── types/       (v0)
└── ...                          └── ...

❌ plugins/ can't import storage/ — they're in different branches!
```

**Strangler Fig requires v0 and v1 to coexist** so plugins can wrap v0 code.

---

## Revised Branch Strategy

**Branch from main, NOT orphan:**

```
peterkc/beads (fork)
│
├── main              # Tracks upstream (v0 only)
│
└── next              # Branched from main (has v0 + adds v1)
                      # NOT orphan — includes v0 code
```

**Result:** `next` branch contains both v0 and v1 packages:

```
next branch (branched from main)
─────────────────────────────────
internal/
│
├── storage/              # v0 (inherited from main)
│   ├── storage.go        # 75-method interface
│   └── sqlite/           # v0 implementation
│
├── types/                # v0 (inherited from main)
│   └── issue.go
│
├── core/                 # v1 (NEW)
│   ├── issue/
│   └── dependency/
│
├── ports/                # v1 (NEW)
│   └── repositories/
│
├── adapters/             # v1 (NEW)
│   └── sqlite/
│
├── usecases/             # v1 (NEW)
│
└── plugins/              # v1 (NEW) — wraps v0!
    └── core/
        └── create.go     # imports internal/storage
```

---

## Package Namespace Strategy

### Coexistence via Namespaces

| Namespace | Version | Purpose |
|-----------|---------|---------|
| `internal/storage/` | v0 | Existing 75-method god interface |
| `internal/types/` | v0 | Existing domain types |
| `internal/core/` | **v1** | New pure domain layer |
| `internal/ports/` | **v1** | New interface definitions |
| `internal/adapters/` | **v1** | New implementations |
| `internal/usecases/` | **v1** | New application layer |
| `internal/plugins/` | **v1** | New plugin system (wraps v0) |

### Import Rules

```go
// Stage 1: Plugins import v0 directly
package core

import (
    "github.com/steveyegge/beads/internal/storage"  // v0
    "github.com/steveyegge/beads/internal/types"    // v0
)

func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    // Wrap v0 code
    return ctx.Storage.CreateIssue(...)  // v0 method
}
```

```go
// Stage 2: Plugins import v1 ports
package core

import (
    "github.com/steveyegge/beads/internal/core/issue"     // v1
    "github.com/steveyegge/beads/internal/ports/repos"    // v1
)

func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    // Use v1 architecture
    return ctx.Issues.Create(...)  // v1 method
}
```

---

## Stage 1: Plugin Wraps v0

```go
// internal/plugins/context.go

package plugins

import (
    "github.com/steveyegge/beads/internal/storage"  // v0 import
)

// Context provides dependencies to plugins
type Context struct {
    // Stage 1: v0 storage directly
    Storage storage.Storage  // 75-method v0 interface

    // Stage 2: Will become v1 ports
    // Issues       ports.IssueRepository
    // Dependencies ports.DependencyRepository
}

func NewContext(db *sql.DB) *Context {
    return &Context{
        Storage: storage.New(db),  // v0 constructor
    }
}
```

```go
// internal/plugins/core/create.go

package core

import (
    "github.com/steveyegge/beads/internal/plugins"
    "github.com/steveyegge/beads/internal/types"  // v0 types
)

func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    title, desc, priority := parseArgs(args)

    issue := &types.Issue{  // v0 type
        Title:       title,
        Description: desc,
        Priority:    priority,
    }

    // Delegate to v0 storage
    err := ctx.Storage.CreateIssue(context.Background(), issue, "actor")
    if err != nil {
        return err
    }

    fmt.Printf("Created: %s\n", issue.ID)
    return nil
}
```

---

## Stage 2: Plugin Uses v1 Ports

```go
// internal/plugins/context.go (updated)

package plugins

import (
    "github.com/steveyegge/beads/internal/ports/repositories"
    "github.com/steveyegge/beads/internal/ports/events"
)

type Context struct {
    // Stage 2: v1 ports (interfaces)
    Issues       repositories.IssueRepository
    Dependencies repositories.DependencyRepository
    Labels       repositories.LabelRepository
    Events       events.EventBus
}
```

```go
// internal/plugins/core/create.go (updated)

package core

import (
    "github.com/steveyegge/beads/internal/core/issue"    // v1 domain
    "github.com/steveyegge/beads/internal/plugins"
)

func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    title, desc, priority := parseArgs(args)

    i := issue.New(title)  // v1 domain constructor
    i.Description = desc
    i.Priority = issue.Priority(priority)

    // Validate using v1 domain logic
    if err := i.Validate(); err != nil {
        return err
    }

    // Persist via v1 port
    if err := ctx.Issues.Create(context.Background(), i); err != nil {
        return err
    }

    // Publish event via v1 event bus
    ctx.Events.Publish(context.Background(), "issue.created", i)

    fmt.Printf("Created: %s\n", i.ID)
    return nil
}
```

---

## Directory Structure Evolution

### Stage 0-1: v0 + v1 Coexist

```
internal/
├── storage/          # v0 (existing)
├── types/            # v0 (existing)
├── linear/           # v0 (existing)
├── compact/          # v0 (existing)
│
├── core/             # v1 (NEW)
├── ports/            # v1 (NEW)
├── adapters/         # v1 (NEW, stubs)
├── usecases/         # v1 (NEW, stubs)
└── plugins/          # v1 (NEW, wraps v0)

cmd/
├── bd/               # v0 CLI
└── bdx/              # v1 CLI (uses plugins)
```

### Stage 2: v1 Replaces v0

```
internal/
├── storage/          # v0 — DELETED after Stage 2.5
├── types/            # v0 — DELETED after Stage 2.5
├── linear/           # v0 — DELETED after Stage 2.5
│
├── core/             # v1 (active)
├── ports/            # v1 (active)
├── adapters/         # v1 (active, implemented)
│   └── sqlite/       # Replaces internal/storage/sqlite/
├── usecases/         # v1 (active, implemented)
└── plugins/          # v1 (active, uses v1 ports)

cmd/
├── bd/               # DELETED or aliased to bdx
└── bdx/              # v1 CLI (renamed to bd)
```

---

## Import Graph

### Stage 1 (Wrapping v0)

```
cmd/bdx
    │
    └── internal/plugins
            │
            ├── internal/storage (v0)  ◄── plugins wrap v0
            └── internal/types (v0)
```

### Stage 2 (Using v1)

```
cmd/bdx
    │
    └── internal/plugins
            │
            ├── internal/core (v1)
            └── internal/ports (v1)
                    │
                    └── internal/adapters (v1)
```

---

## Build Tags for Conditional Compilation

If needed, use build tags to switch between v0 and v1:

```go
// internal/plugins/context_v0.go
//go:build !v1

package plugins

import "github.com/steveyegge/beads/internal/storage"

type Context struct {
    Storage storage.Storage  // v0
}
```

```go
// internal/plugins/context_v1.go
//go:build v1

package plugins

import "github.com/steveyegge/beads/internal/ports/repositories"

type Context struct {
    Issues repositories.IssueRepository  // v1
}
```

```bash
# Build with v0 backend (Stage 1)
go build -o bdx ./cmd/bdx

# Build with v1 backend (Stage 2)
go build -tags=v1 -o bdx ./cmd/bdx
```

---

## Orphan Branch: When?

**Orphan is optional** and only makes sense AFTER migration is complete:

| Phase | Branch Strategy |
|-------|-----------------|
| Stage 0 | Branch from main (need v0 code) |
| Stage 1 | Branch from main (plugins wrap v0) |
| Stage 2 | Branch from main (replacing v0) |
| **Post-migration** | *Optional:* Create orphan for clean history |

**If clean history is still desired:**
1. Complete Stage 2 (v0 code deleted)
2. Create orphan branch with only v1 code
3. Use git notes for traceability
4. Replace main with orphan (force push)

---

## Summary

| Question | Answer |
|----------|--------|
| How to separate v0/v1? | **Package namespaces** (storage/ vs core/) |
| Can plugins wrap v0? | **Yes** — same branch, direct imports |
| Branch strategy? | **Branch from main**, not orphan |
| When orphan? | **Post-migration** (optional, for clean history) |
| How to switch v0→v1? | **Update PluginContext** from Storage to Ports |
