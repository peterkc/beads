# ADR 0004: CI/CD Workflow Strategy

## Status

Accepted

## Date

2026-01-20

## Context

The beads-next clone inherits v0 GitHub Actions workflows from upstream. During bdx development:

1. **CI requirements diverge** — bdx needs characterization tests, stage gates, dual-binary builds
2. **v0 workflows are suboptimal** — room for improvement in caching, parallelism, and test organization
3. **Upstream sync matters** — we still want to pull upstream improvements occasionally

### Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A. Gitignore | Exclude `.github/` | Clean sync | No CI |
| B. Separate branch | Workflows on `ci` branch | Isolated | Complex |
| C. Replace inline | Edit workflows directly | Simple | Merge conflicts |
| D. Conditional | `if: startsWith(github.ref, 'v1')` | Single source | Complex files |
| **E. New workflow files** | Add `ci-bdx.yml`, etc. | Clean separation | Old files become dead code |

## Decision

**Use Option E: New workflow files with bdx suffix.**

### File Strategy

```
.github/workflows/
├── ci.yml              # v0 (unchanged, triggers on v0.x tags)
├── release.yml         # v0 (unchanged)
├── ci-bdx.yml          # NEW: bdx CI
├── release-bdx.yml     # NEW: bdx release (future)
└── ...
```

### Trigger Strategy

**v0 workflows** — restrict to maintenance:
```yaml
on:
  push:
    tags: ['v0.*']
    branches: ['v0-maintenance']  # if needed
```

**bdx workflows** — run on main:
```yaml
on:
  push:
    branches: [main]
    tags: ['v1.*']
  pull_request:
    branches: [main]
```

### bdx CI Improvements

| Area | v0 | bdx |
|------|-----|-----|
| Characterization tests | None | Required gate |
| Stage validation | None | `just check-stage1` |
| Dual binary | bd only | bd + bdx |
| Windows | Smoke tests | Deferred (Stage 3) |
| Nix | Basic test | Deferred (Stage 3) |
| Caching | Basic | Go module + build cache |

## Consequences

### Positive

- Clean separation — no merge conflicts with upstream
- Optimized CI for bdx development workflow
- Stage gates catch regressions early
- Old workflows can be deleted post-v1

### Negative

- Duplicate workflow files during transition
- Must remember to update correct file

### Migration Path

1. **Now**: Create `ci-bdx.yml`, keep v0 workflows unchanged
2. **Stage 2**: Add `release-bdx.yml` for v1 releases
3. **Post-v1**: Delete v0 workflow files, rename bdx → standard names

## Notes

- Windows support deferred to Stage 3 (daemon architecture changes needed)
- Nix flake will need updates for bdx binary — deferred to Stage 3
