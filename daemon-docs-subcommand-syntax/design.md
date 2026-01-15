# Design

## Replacement Strategy

### Approach: Regex-Based Find/Replace

Use targeted regex patterns to handle all variations safely:

```bash
# Pattern 1: --start with potential flags
s/daemon --start(\s|$)/daemon start\1/g

# Pattern 2: --stop (not --stop-all)
s/daemon --stop(\s|$)/daemon stop\1/g

# Pattern 3: --status
s/daemon --status/daemon status/g

# Pattern 4: --stop-all
s/daemon --stop-all/daemon killall/g

# Pattern 5: --health
s/daemon --health/daemon status --all/g
```

### Key Decisions

#### KD-001: Order of Operations

**Decision**: Apply replacements in this order:
1. `--stop-all` → `killall` (before `--stop` to avoid partial match)
2. `--health` → `status --all`
3. `--start` → `start`
4. `--stop` → `stop`
5. `--status` → `status`

**Rationale**: `--stop-all` must be replaced before `--stop` to prevent `--stop-all` becoming `stop-all` (with `--stop` partially matched).

#### KD-002: Tool Selection

**Decision**: Use `sd` (structured diff) over `sed` for replacements.

**Rationale**:
- `sd` uses familiar regex syntax (no escaping nightmare)
- Better Unicode handling
- Consistent cross-platform behavior
- Already available in ACF environment

**Alternative considered**: Manual `Edit` tool — rejected due to 40 occurrences across 12 files.

#### KD-003: Verification Method

**Decision**: Use grep count before/after to verify completeness.

```bash
# Before: count deprecated patterns
rg "daemon --(start|stop|status|stop-all|health)" --type md -c

# After: should return no matches (except CHANGELOG)
rg "daemon --(start|stop|status|stop-all|health)" --type md | grep -v CHANGELOG
```

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Partial match creates invalid syntax | Low | Medium | Test regex on sample before bulk apply |
| CHANGELOG modified accidentally | Low | Low | Exclude explicitly in commands |
| Formatting broken in code blocks | Low | Medium | Verify markdown renders correctly |

## Out of Scope

- Updating any Go source code (deprecation warnings remain)
- Modifying CHANGELOG.md entries
- Adding new documentation about subcommand syntax
