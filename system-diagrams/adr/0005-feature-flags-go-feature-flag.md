# ADR 0005: Feature Flags for v0/v1 Migration

## Status

Accepted (Updated)

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
| Beads config integration | ❌ Global only | ❌ No | Native | Zero |
| go-feature-flag | ✅ User/project | ✅ Yes | OpenFeature | Active (1.9k★) |
| LaunchDarkly | ✅ Full | ✅ Yes | OpenFeature | Commercial |
| Separate YAML | ❌ Global only | ❌ No | None | DIY |

## Decision

**Use beads config for flags (simple), with optional go-feature-flag upgrade path (advanced).**

### Primary: Config-Based Flags

Integrate flags into existing `bd config` system:

```yaml
# .beads/config.yaml
sync:
  auto: true
  branch: main

# Feature flags section
flags:
  use_v1_create: false
  use_v1_list: false
  use_v1_ready: false
  use_v1_sync: false
  use_v1: false           # Master switch for all
```

**User experience:**

```bash
# Check current flags
bd config get flags
# flags.use_v1_create: false
# flags.use_v1_list: false
# ...

# Enable v1 for specific feature
bd config set flags.use_v1_create true

# Enable all v1 features
bd config set flags.use_v1 true
```

### CLI Flag Overrides

Integrates with existing **Cobra** CLI framework (`github.com/spf13/cobra`):

| Cobra Feature | Usage |
|---------------|-------|
| `PersistentFlags()` | `--flag`, `--v1`, `--v0` inherited by all subcommands |
| `StringArrayVar` | Multiple `--flag` values in single command |
| `PreRunE` | Build effective flag config before command executes |

Per-command overrides without changing config:

```bash
# Shorthand: all v1 or all v0
bd --v1 create "Test issue"
bd --v0 list

# Granular: specific flag
bd --flag use_v1_create create "Test issue"
bd --no-flag use_v1_create create "Test issue"

# Explicit value
bd --flag use_v1_create=true create "Test issue"
bd --flag use_v1_create=false create "Test issue"

# Multiple flags
bd --flag use_v1_create --flag use_v1_list list

# Debug: see effective flags
bd --dry-run --flag use_v1_create create "Test issue"
# Would use: use_v1_create=true, use_v1_list=false, ...
```

**Priority order (lowest to highest):**

```
1. Config file (.beads/config.yaml)
2. --v1 / --v0 shorthand
3. --flag name=value
4. --no-flag name
```

**Implementation:**

```go
// cmd/bd/main.go
var (
    flagOverrides   []string // --flag values
    noFlagOverrides []string // --no-flag values
    useV1           bool     // --v1 shorthand
    useV0           bool     // --v0 shorthand
)

func init() {
    rootCmd.PersistentFlags().StringArrayVar(&flagOverrides, "flag", nil,
        "Enable feature flag (e.g., --flag use_v1_create)")
    rootCmd.PersistentFlags().StringArrayVar(&noFlagOverrides, "no-flag", nil,
        "Disable feature flag (e.g., --no-flag use_v1_create)")
    rootCmd.PersistentFlags().BoolVar(&useV1, "v1", false, "Use all v1 features")
    rootCmd.PersistentFlags().BoolVar(&useV0, "v0", false, "Use all v0 features")
}

func buildFlagConfig(base *FlagConfig) *FlagConfig {
    result := *base // Copy from config file

    // Apply --v1/--v0 first (lowest priority)
    if useV1 {
        result.UseV1 = true
    }
    if useV0 {
        result.UseV1 = false
    }

    // Apply --flag overrides (higher priority)
    for _, f := range flagOverrides {
        name, value := parseFlag(f) // "use_v1_create" or "use_v1_create=true"
        result.Set(name, value)
    }

    // Apply --no-flag overrides (highest priority)
    for _, f := range noFlagOverrides {
        result.Set(f, false)
    }

    return &result
}

func parseFlag(s string) (string, bool) {
    if parts := strings.SplitN(s, "=", 2); len(parts) == 2 {
        return parts[0], parts[1] == "true"
    }
    return s, true // --flag name implies true
}
```

**Cobra PreRunE integration:**

```go
// cmd/bd/main.go
var pluginCtx *plugins.Context

func init() {
    // Apply flags before any command runs
    rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
        // Load config from file
        cfg, err := config.Load()
        if err != nil {
            return err
        }

        // Apply CLI overrides
        cfg.Flags = *buildFlagConfig(&cfg.Flags)

        // Build plugin context with effective flags
        pluginCtx, err = plugins.NewContext(cfg)
        return err
    }
}
```

### Config Structure

```go
// internal/config/config.go
type Config struct {
    Sync  SyncConfig  `yaml:"sync"`
    Flags FlagConfig  `yaml:"flags"`
}

type FlagConfig struct {
    UseV1       bool `yaml:"use_v1"`        // Master switch
    UseV1Create bool `yaml:"use_v1_create"`
    UseV1List   bool `yaml:"use_v1_list"`
    UseV1Ready  bool `yaml:"use_v1_ready"`
    UseV1Sync   bool `yaml:"use_v1_sync"`
}

func (f *FlagConfig) IsV1Enabled(feature string) bool {
    if f.UseV1 {
        return true  // Master switch overrides
    }
    switch feature {
    case "create":
        return f.UseV1Create
    case "list":
        return f.UseV1List
    case "ready":
        return f.UseV1Ready
    case "sync":
        return f.UseV1Sync
    default:
        return false
    }
}
```

**Plugin usage:**

```go
// internal/plugins/core/create.go
func (p *Plugin) Create(ctx *plugins.Context, args []string) error {
    if ctx.Config.Flags.IsV1Enabled("create") {
        return ctx.Issues.Create(...)  // v1 path
    }
    return ctx.Storage.CreateIssue(...)  // v0 path
}
```

### Optional: go-feature-flag Upgrade

For advanced use cases (percentage rollout, user targeting), upgrade to go-feature-flag:

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

---

## Backward Compatibility Requirements

**Critical:** If users can switch between v0/v1, data must be identical.

### Compatibility Matrix

| Layer | Must Match? | Why |
|-------|-------------|-----|
| SQLite schema | ✅ Yes | Same database file |
| JSONL format | ✅ Yes | Same export/import |
| CLI output | ⚠️ Similar | Scripts may parse output |
| Config format | ✅ Yes | Same config.yaml |

### Data Flow

```
v0 writes data  →  v1 must read correctly
v1 writes data  →  v0 must read correctly
        ↓
    SAME FORMAT
```

### Schema Evolution Rules

```
ADDITIVE ONLY (backward compatible):
─────────────────────────────────────
✅ Add optional field with default
✅ Add new table
✅ Add index
✅ Add new enum value (if code handles unknown)

BREAKING (avoid during v0/v1 coexistence):
────────────────────────────────────────────
❌ Remove field
❌ Change field type
❌ Rename field
❌ Change primary key
❌ Remove table
```

### Implementation Pattern

```go
// internal/core/issue/issue.go

type Issue struct {
    // Required fields (both v0 and v1)
    ID          string
    Title       string
    Status      Status
    Priority    int

    // Optional fields (v1 additions)
    // MUST have json omitempty and sensible defaults
    Tags     []string  `json:"tags,omitempty"`
    Metadata Metadata  `json:"metadata,omitempty"`
}

// v0 code reading v1 data:
// - Ignores unknown fields (Tags, Metadata)
// - Works correctly

// v1 code reading v0 data:
// - Missing optional fields default to zero value
// - Works correctly
```

### Compatibility Test

```bash
#!/usr/bin/env bash
# scripts/test-compatibility.sh

set -e

echo "=== v0/v1 Compatibility Test ==="

# 1. Create with v0
bd config set flags.use_v1 false
bd create "Test from v0" --priority 2
V0_ID=$(bd list --format=json | jq -r '.[-1].id')

# 2. Read with v1
bd config set flags.use_v1 true
bd show "$V0_ID" > /dev/null
echo "✅ v1 can read v0 data"

# 3. Create with v1
bd create "Test from v1" --priority 3
V1_ID=$(bd list --format=json | jq -r '.[-1].id')

# 4. Read with v0
bd config set flags.use_v1 false
bd show "$V1_ID" > /dev/null
echo "✅ v0 can read v1 data"

# 5. Verify JSONL compatibility
bd sync
bd config set flags.use_v1 true
bd sync
echo "✅ JSONL export/import compatible"

echo ""
echo "=== All compatibility tests passed ==="
```

### CI Integration

```yaml
# .github/workflows/compatibility.yml
name: v0/v1 Compatibility

on: [push, pull_request]

jobs:
  test-compatibility:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build both versions
        run: |
          go build -o bd ./cmd/bd
          go build -tags=v1 -o bdx ./cmd/bdx

      - name: Run compatibility tests
        run: ./scripts/test-compatibility.sh
```

## References

- [go-feature-flag](https://github.com/thomaspoignant/go-feature-flag)
- [OpenFeature Specification](https://openfeature.dev/)
- [ADR 0003: Migration Strategy](0003-migration-strategy-strangler-fig.md)
- [Feature Flags Best Practices - Martin Fowler](https://martinfowler.com/articles/feature-toggles.html)
