# ADR 0005: Feature Flags with go-feature-flag

## Status

Accepted

## Context

During the v0→v1 migration (see [ADR 0003](0003-migration-strategy-strangler-fig.md)), we need:

1. **Safe rollout** — Test v1 code paths without affecting all users
2. **Instant rollback** — Revert to v0 if issues discovered
3. **Gradual migration** — Percentage-based rollout per feature
4. **Per-user testing** — Enable v1 for specific users/projects first

Additionally, feature flags enable future capabilities:
- A/B testing new algorithms
- Beta features for opt-in users
- Kill switches for problematic features

**Options considered:**

| Option | Targeting | Rollout % | Standard | Maintenance |
|--------|-----------|-----------|----------|-------------|
| go-feature-flag | ✅ User/project | ✅ Yes | OpenFeature | Active (1.9k★) |
| LaunchDarkly | ✅ Full | ✅ Yes | OpenFeature | Commercial |
| Built-in YAML | ❌ Global only | ❌ No | None | DIY |
| Environment vars | ❌ Global only | ❌ No | None | DIY |

## Decision

**Use [go-feature-flag](https://github.com/thomaspoignant/go-feature-flag) for feature flag management.**

### Rationale

1. **OpenFeature standard** — Portable across providers if needs change
2. **File-based config** — YAML/JSON in `.beads/` or user config
3. **Targeting rules** — Per-user, per-project rollout
4. **Percentage rollout** — Gradual migration support
5. **Self-hosted** — No external service dependency
6. **Active maintenance** — 1.9k stars, regular updates

### Integration with Plugin Architecture

```go
// internal/plugins/context.go
import (
    ffclient "github.com/thomaspoignant/go-feature-flag"
    "github.com/thomaspoignant/go-feature-flag/retriever/fileretriever"
)

type Context struct {
    // v0 Storage (Stage 1)
    Storage *storage.Storage

    // v1 ports (Stage 2, added incrementally)
    Issues       ports.IssueRepository
    Dependencies ports.DependencyRepository
    Work         ports.WorkRepository
    Config       ports.ConfigStore
    Events       ports.EventBus

    // Feature flags
    Flags *ffclient.GoFeatureFlag
}

func NewContext(db *sql.DB) (*Context, error) {
    // Initialize feature flags
    ff, err := ffclient.New(ffclient.Config{
        PollingInterval: 60 * time.Second,
        Retriever: &fileretriever.Retriever{
            Path: findFlagsConfig(), // .beads/flags.yaml or ~/.config/beads/flags.yaml
        },
    })
    if err != nil {
        return nil, fmt.Errorf("init feature flags: %w", err)
    }

    return &Context{
        Storage: storage.New(db),
        Flags:   ff,
    }, nil
}

func findFlagsConfig() string {
    // Priority: project → user → default
    if exists(".beads/flags.yaml") {
        return ".beads/flags.yaml"
    }
    if exists("~/.config/beads/flags.yaml") {
        return expandHome("~/.config/beads/flags.yaml")
    }
    return "" // No flags, all defaults
}
```

### Flag Usage in Plugins

```go
// internal/plugins/core/create.go
func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    // Build evaluation context
    evalCtx := ffclient.NewEvaluationContext(getUser())
    evalCtx.AddCustom("project", getProjectName())

    // Check feature flag
    useV1, _ := ctx.Flags.BoolVariation("beads-v1-create", evalCtx, false)

    if useV1 && ctx.Issues != nil {
        // v1 code path
        issue := core.NewIssue(title, description, priority)
        return ctx.Issues.Create(context.Background(), issue)
    }

    // v0 code path (default)
    _, err := ctx.Storage.CreateIssue(title, description, priority)
    return err
}
```

### Flag Configuration

```yaml
# .beads/flags.yaml
# Project-level feature flags

beads-v1-create:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: disabled
  targeting:
    # Enable for specific users
    - query: user eq "peterkc"
      variation: enabled
    # Enable for experimental projects
    - query: project contains "experimental"
      variation: enabled

beads-v1-list:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    # Gradual rollout: 20% get v1
    percentage:
      enabled: 20
      disabled: 80

beads-v1-ready:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: disabled  # Not ready yet
```

### User-Level Configuration

```yaml
# ~/.config/beads/flags.yaml
# User-level overrides (for testing)

beads-v1-create:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: enabled  # Always use v1 for this user

beads-v1-list:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: enabled
```

### Migration Rollout Strategy

```
PHASE 1: Internal Testing
─────────────────────────
beads-v1-create: targeting → user eq "maintainer"
beads-v1-list:   targeting → user eq "maintainer"
beads-v1-ready:  disabled

PHASE 2: Early Adopters (opt-in)
────────────────────────────────
beads-v1-create: targeting → user in ["maintainer", "early-adopter"]
beads-v1-list:   percentage → 10% enabled
beads-v1-ready:  targeting → user eq "maintainer"

PHASE 3: Gradual Rollout
────────────────────────
beads-v1-create: percentage → 50% enabled
beads-v1-list:   percentage → 50% enabled
beads-v1-ready:  percentage → 20% enabled

PHASE 4: Full Rollout
─────────────────────
beads-v1-create: 100% enabled → REMOVE FLAG, DELETE v0 CODE
beads-v1-list:   100% enabled → REMOVE FLAG, DELETE v0 CODE
beads-v1-ready:  100% enabled → REMOVE FLAG, DELETE v0 CODE
```

### Flag Naming Convention

```
beads-{version}-{feature}

Examples:
- beads-v1-create      # v1 create command
- beads-v1-list        # v1 list command
- beads-v1-ready       # v1 ready algorithm
- beads-exp-molecules  # Experimental molecules feature
- beads-beta-linear    # Beta Linear integration
```

### Graceful Degradation

If flags file missing or invalid, default to v0:

```go
func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    useV1 := false

    if ctx.Flags != nil {
        evalCtx := ffclient.NewEvaluationContext(getUser())
        useV1, _ = ctx.Flags.BoolVariation("beads-v1-create", evalCtx, false)
    }

    // ... rest of implementation
}
```

### CLI Flag Override

For debugging, allow CLI override:

```bash
# Force v1 for this command
bd --use-v1 create "Test issue"

# Force v0 for this command
bd --use-v0 list

# Check current flag state
bd flags list
bd flags get beads-v1-create
```

```go
// cmd/bd/main.go
var forceV1 = flag.Bool("use-v1", false, "Force v1 code path")
var forceV0 = flag.Bool("use-v0", false, "Force v0 code path")
```

## Consequences

### Positive

- **Safe migration** — Test v1 without risk to all users
- **Instant rollback** — Change YAML, no redeploy
- **Gradual rollout** — Percentage-based migration
- **Per-user control** — Maintainers can test first
- **Future extensibility** — Flags useful beyond migration
- **No external dependency** — Self-hosted, file-based

### Negative

- **Added dependency** — go-feature-flag library
- **Configuration overhead** — Another YAML file to manage
- **Code complexity** — Flag checks in each feature

### Mitigations

- Flags are optional (missing file = all defaults)
- Remove flags after migration complete (cleanup phase)
- Centralize flag checks in plugin layer

## Flag Lifecycle

```
1. CREATE    → Add flag to flags.yaml (disabled by default)
2. TEST      → Enable for maintainers via targeting
3. ROLLOUT   → Gradual percentage increase
4. COMPLETE  → 100% enabled, monitor for issues
5. CLEANUP   → Remove flag, delete v0 code path
```

**Important:** Flags are temporary for migration. After v1 is stable, remove all `beads-v1-*` flags and their associated v0 code.

## References

- [go-feature-flag](https://github.com/thomaspoignant/go-feature-flag)
- [OpenFeature Specification](https://openfeature.dev/)
- [ADR 0003: Migration Strategy](0003-migration-strategy-strangler-fig.md)
- [Feature Flags Best Practices - Martin Fowler](https://martinfowler.com/articles/feature-toggles.html)
