# Beads v1.0 (bdx)

Architecture rewrite using hexagonal (ports/adapters) pattern with plugin-based commands.

## Quick Links

- [Architecture ADRs](docs/adr/) — bdx design decisions (permanent)
- [Migration ADRs](research/system-diagrams/adr/) — v0→v1 transition (temporary)
- [Implementation Plan](research/system-diagrams/docs/implementation-plan.md)
- [Package Structure](research/system-diagrams/docs/package-structure-versioned.md)

## Current Stage

**Stage 1: Foundation** — Tests, core domain, port interfaces

## Directory Structure

```
internal/
├── v0/                 # Existing code (DO NOT MODIFY except Stage 2)
└── next/               # v1 code (new work goes here)
    ├── core/           # Domain layer (pure Go, no external imports)
    │   ├── issue/      # Issue entity
    │   ├── dependency/ # Dependency entity + graph
    │   └── events/     # Domain events
    ├── ports/          # Interfaces (repositories, services)
    ├── adapters/       # Implementations (sqlite, git, linear)
    ├── usecases/       # Application logic
    └── plugins/        # Command plugins (wrap v0 initially)

cmd/
├── bd/                 # v0 CLI (existing)
└── bdx/                # v1 CLI (new)

docs/
└── adr/                # Architecture decisions (permanent)

research/               # Migration docs (orphan branch, temporary)
specs/                  # Feature specs (orphan branch)
.beads-planning/        # bdx issue tracking (orphan branch)
```

## Rules

1. **Never modify `internal/v0/`** except during Stage 2 reorganization
2. **Characterization tests must pass** before structural changes
3. **`internal/next/core/` has ZERO external imports** — pure domain logic
4. **One plugin at a time** — complete and test before moving on

## ADR Discipline

When making architectural changes:

1. **Check existing ADRs** — Is there a relevant decision in `docs/adr/`?
2. **Create new ADR** if decision is significant:
   - New interface or port
   - Changed plugin contract
   - New external dependency
   - Breaking change to internal API
3. **Update ADR status** if superseding a previous decision

**Two ADR locations:**

| Location | Purpose | Lifespan |
|----------|---------|----------|
| `docs/adr/` | bdx architecture decisions | Permanent |
| `research/system-diagrams/adr/` | Migration decisions | Until v1 ships |

## Workflows

### Feature Development (Graphite)

```bash
# Create stacked PRs
gt create -m "feat(core): add issue entity"
# ... work ...
gt modify                               # Amend current PR
gt create -m "feat(ports): add IssueRepository"  # Stack next PR
gt sync                                 # Rebase on main
gt merge --all                          # Merge when approved
```

### Issue Tracking

```bash
bd create --title="Stage 1a: Testing infrastructure"
bd list --status=open
bd update bdx-xxx --status=in_progress
bd close bdx-xxx
bd sync                                 # After completing work
```

### Testing

```bash
go test ./internal/next/...                       # v1 tests
go test -tags=characterization ./characterization/...  # Behavior validation
go test ./...                                     # All tests
```

## Validation Gates

Each stage has a checkpoint. Do not proceed until:

```bash
# Stage 1: Foundation
go test -tags=characterization ./characterization/...  # 100% pass
go test ./internal/next/core/...                       # 100% pass
go generate ./internal/next/ports/...                  # Mocks generate

# Stage 2: Pluginize
./scripts/compare-bd-bdx.sh                            # Identical output

# Stage 3: Modernize
go test ./...                                          # All pass
```

## Key Patterns

### Interface Segregation

v0's 75-method Storage interface splits into focused interfaces:

| v1 Interface         | Methods | Purpose            |
| -------------------- | ------- | ------------------ |
| `IssueRepository`    | 5       | Issue CRUD         |
| `DependencyRepository`| 4      | Dependency graph   |
| `LabelRepository`    | 3       | Labels             |
| `CommentRepository`  | 3       | Comments           |
| `SyncTracker`        | 4       | Dirty tracking     |

### Plugin Wrapping (Stage 2)

Plugins initially wrap v0:

```go
func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    // Parse args...

    // Delegate to v0
    issue, err := ctx.Storage.CreateIssue(title, description, priority)
    if err != nil {
        return err
    }

    fmt.Printf("Created: %s\n", issue.ID)
    return nil
}
```

### Adapter Pattern (Stage 3)

Replace v0 delegation with v1 implementations:

```go
// Before: ctx.Storage.CreateIssue(...)
// After:  ctx.Issues.Create(...)

type Context struct {
    Issues       ports.IssueRepository
    Dependencies ports.DependencyRepository
    // ...
}
```

## Abort Procedures

If something goes wrong, see [ADR 0003 Abort Procedures](research/system-diagrams/adr/0003-migration-strategy-strangler-fig.md#abort-procedures) for rollback steps.
