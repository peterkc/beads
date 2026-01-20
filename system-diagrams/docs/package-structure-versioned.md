# Package Structure: Versioned Directories

## Recommended Structure

```
internal/
├── v0/                          # All v0 code (moved from current locations)
│   ├── storage/
│   │   ├── storage.go           # 75-method interface
│   │   ├── sqlite/
│   │   └── memory/
│   ├── types/
│   │   └── issue.go
│   ├── linear/
│   ├── compact/
│   ├── molecules/
│   ├── export/
│   ├── importer/
│   ├── config/
│   ├── configfile/
│   ├── hooks/
│   ├── git/
│   ├── rpc/
│   ├── daemon/
│   └── ...                      # All other v0 packages
│
└── next/                        # All v1 code (new)
    ├── core/                    # Domain layer
    │   ├── issue/
    │   ├── dependency/
    │   ├── label/
    │   └── events/
    │
    ├── ports/                   # Interface definitions
    │   ├── repositories/
    │   ├── services/
    │   └── events/
    │
    ├── adapters/                # Implementations
    │   ├── sqlite/
    │   ├── memory/
    │   ├── linear/
    │   └── git/
    │
    ├── usecases/                # Application logic
    │   ├── issue/
    │   ├── dependency/
    │   └── sync/
    │
    ├── plugins/                 # Plugin system
    │   ├── registry.go
    │   ├── context.go
    │   ├── core/
    │   ├── work/
    │   ├── sync/
    │   ├── linear/
    │   └── daemon/
    │
    └── infra/                   # Infrastructure
        ├── config/
        ├── logging/
        └── errors/

cmd/
├── bd/                          # v0 CLI (imports internal/v0/)
└── bdx/                         # v1 CLI (imports internal/next/)
```

______________________________________________________________________

## Benefits

| Benefit                  | Why                                              |
| ------------------------ | ------------------------------------------------ |
| **Explicit versioning**  | Import path shows version: `internal/v0/storage` |
| **Trivial cleanup**      | Stage 2.5 = `rm -rf internal/v0/`                |
| **No ambiguity**         | Every package clearly belongs to v0 or next      |
| **Parallel development** | v0 and next don't interfere                      |
| **Clear migration path** | Move imports from `v0/` to `next/`               |

______________________________________________________________________

## Import Examples

### Stage 1: Plugin Wraps v0

```go
// internal/next/plugins/core/create.go
package core

import (
    "github.com/steveyegge/beads/internal/v0/storage"  // v0 import
    "github.com/steveyegge/beads/internal/v0/types"    // v0 import
    "github.com/steveyegge/beads/internal/next/plugins"
)

func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    issue := &types.Issue{Title: title}
    return ctx.Storage.CreateIssue(ctx, issue)  // v0 method
}
```

### Stage 2: Plugin Uses v1

```go
// internal/next/plugins/core/create.go
package core

import (
    "github.com/steveyegge/beads/internal/next/core/issue"    // v1 import
    "github.com/steveyegge/beads/internal/next/ports/repos"   // v1 import
    "github.com/steveyegge/beads/internal/next/plugins"
)

func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    i := issue.New(title)              // v1 domain
    return ctx.Issues.Create(ctx, i)   // v1 port
}
```

______________________________________________________________________

## PluginContext Evolution

### Stage 1: Wraps v0

```go
// internal/next/plugins/context.go
package plugins

import (
    "github.com/steveyegge/beads/internal/v0/storage"
)

type Context struct {
    // Stage 1: v0 storage
    Storage storage.Storage
}

func NewContext(db *sql.DB) *Context {
    return &Context{
        Storage: storage.New(db),  // v0 constructor
    }
}
```

### Stage 2: Uses v1 Ports

```go
// internal/next/plugins/context.go
package plugins

import (
    "github.com/steveyegge/beads/internal/next/ports/repositories"
    "github.com/steveyegge/beads/internal/next/ports/events"
)

type Context struct {
    // Stage 2: v1 ports
    Issues       repositories.IssueRepository
    Dependencies repositories.DependencyRepository
    Events       events.EventBus
}

func NewContext(db *sql.DB) *Context {
    return &Context{
        Issues:       sqlite.NewIssueRepository(db),
        Dependencies: sqlite.NewDependencyRepository(db),
        Events:       membus.New(),
    }
}
```

______________________________________________________________________

## Migration Steps

### Phase 0: Move v0 Code

Before starting v1 development, reorganize v0:

```bash
# Create v0 directory
mkdir -p internal/v0

# Move existing packages
git mv internal/storage internal/v0/
git mv internal/types internal/v0/
git mv internal/linear internal/v0/
git mv internal/compact internal/v0/
# ... move all v0 packages

# Update all imports
sed -i 's|internal/storage|internal/v0/storage|g' **/*.go
sed -i 's|internal/types|internal/v0/types|g' **/*.go
# ... update all imports

# Commit the reorganization
git commit -m "refactor: move v0 code to internal/v0/"
```

**Automated script:**

```bash
#!/bin/bash
# scripts/reorganize-v0.sh

set -euo pipefail

V0_PACKAGES=(
    storage types linear compact molecules
    export importer config configfile hooks
    git rpc daemon beads debug formula
    idgen lockfile merge recipes routing
    syncbranch templates testutil timeparsing
    ui util utils validation audit autoimport
)

mkdir -p internal/v0

for pkg in "${V0_PACKAGES[@]}"; do
    if [ -d "internal/$pkg" ]; then
        git mv "internal/$pkg" "internal/v0/"
    fi
done

# Update imports using goimports or sed
find . -name "*.go" -exec sed -i '' \
    -e 's|"github.com/steveyegge/beads/internal/storage|"github.com/steveyegge/beads/internal/v0/storage|g' \
    -e 's|"github.com/steveyegge/beads/internal/types|"github.com/steveyegge/beads/internal/v0/types|g' \
    {} \;
# ... repeat for all packages

echo "Done! Review changes and commit."
```

### Phase 0.3-0.4: Create next Directory

```bash
mkdir -p internal/next/{core,ports,adapters,usecases,plugins,infra}
```

### Stage 2.5: Delete v0

```bash
# After all plugins use v1 ports:
rm -rf internal/v0/
git commit -m "chore: remove v0 code after migration complete"
```

______________________________________________________________________

## Directory Comparison

| Current (Flat)      | Versioned              | Clarity         |
| ------------------- | ---------------------- | --------------- |
| `internal/storage/` | `internal/v0/storage/` | ✅ Explicit v0  |
| `internal/core/`    | `internal/next/core/`  | ✅ Explicit v1  |
| Mixed at same level | Separated by version   | ✅ No confusion |
| Manual cleanup      | `rm -rf internal/v0/`  | ✅ One command  |

______________________________________________________________________

## Naming: `v1` vs `next`

| Name   | Pros                    | Cons                     |
| ------ | ----------------------- | ------------------------ |
| `v1`   | Explicit version number | Tied to specific version |
| `next` | Reusable (v2, v3...)    | Less explicit            |

**Recommendation:** Use `next` — it's the Go community convention (Node.js, React) and reusable.

______________________________________________________________________

## Import Path Length

| Structure    | Import Path                |
| ------------ | -------------------------- |
| Flat         | `internal/storage`         |
| Versioned    | `internal/v0/storage`      |
| Versioned v1 | `internal/next/core/issue` |

**Trade-off:** Slightly longer paths, but much clearer intent.

______________________________________________________________________

## Summary

| Question               | Answer                                       |
| ---------------------- | -------------------------------------------- |
| Where is v0 code?      | `internal/v0/`                               |
| Where is v1 code?      | `internal/next/`                             |
| How to migrate?        | Move imports from `v0/` to `next/`           |
| How to cleanup?        | `rm -rf internal/v0/`                        |
| What's the first step? | Reorganize existing code into `internal/v0/` |
