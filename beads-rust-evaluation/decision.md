# Decision: beads_rust (br) vs beads (bd)

## Context

Evaluated br (beads_rust) as potential replacement for bd (beads) in the ACF ecosystem.

## Decision

**Stay with beads (bd)**, continue upstream contributions.

## Rationale

### Deal-Breaker: Multi-Repo Support

ACF architecture fundamentally depends on multi-repo aggregation:

```
bd list                # Shows acf-*, spec-*, research-*, oss-*, ctx-*
bd repo sync           # Hydrates from nested repos
```

br explicitly does not support multi-repo and won't add it ("frozen architecture").

### Deal-Breaker: Molecules/Templates

ACF uses molecules for spec-driven development:

```bash
bd mol catalog         # Work templates
bd pour spec --var feature="Auth"
bd wisp create spec    # Ephemeral exploration
```

br has no template system.

### Deal-Breaker: bd prime

ACF session hooks depend on `bd prime` for agent context injection:

```bash
bd prime               # AI-optimized CLI reference (~1-2k tokens)
```

br has no equivalent.

### What br Does Better

| Feature | Impact |
|---------|--------|
| Binary size (5.2MB vs 30MB) | Minor (disk is cheap) |
| Structured error codes | Nice-to-have |
| Levenshtein suggestions | Nice-to-have |
| Simpler mental model (no daemon) | Minor |

None of these outweigh the missing critical features.

## Alternatives Considered

### 1. Switch to br entirely

**Rejected**: Would require rebuilding ACF's multi-repo architecture from scratch.

### 2. Use both (br for simple projects, bd for ACF)

**Possible but unnecessary**: bd handles simple projects fine.

### 3. Contribute multi-repo to br

**Rejected**: br's philosophy explicitly excludes this ("frozen architecture").

### 4. Port br's error patterns to bd

**Accepted as follow-up**: Structured errors could improve bd's agent ergonomics.

## Consequences

### Positive

- No migration effort
- Existing ACF skills/hooks continue working
- Benefit from bd's active development (2400 commits/month)

### Negative

- Miss br's cleaner error handling (mitigated by potential upstream port)
- Larger binary (acceptable trade-off)

## Action Items

1. **Continue bd contributions** — features, fixes, documentation
2. **Consider error pattern PR** — port br's structured errors to bd
3. **Monitor br** — interesting patterns may emerge for future porting

## References

- [beads_rust repo](https://github.com/Dicklesworthstone/beads_rust)
- [beads repo](https://github.com/steveyegge/beads)
- Local clone: `/Volumes/atlas/beads_rust`
