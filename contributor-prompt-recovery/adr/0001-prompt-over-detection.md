# ADR-0001: Prompt Over Detection

## Status

Accepted

## Context

SSH forks are incorrectly detected as maintainer because SSH URLs indicate local write access, but the user may not have push access to upstream (GH#1174).

Initial approach proposed 5-tier detection:
1. Explicit git config
2. Cached role
3. Upstream remote detection
4. GitHub API
5. SSH/HTTPS heuristic

## Decision

Use **Prompt + Push-Fail Recovery** instead of upfront detection.

### Two Gates

1. **Init prompt**: "Contributing to someone else's repo? [y/N]"
2. **Push failure**: Parse permission errors, offer recovery wizard

### Rationale

| Consideration | Decision |
|---------------|----------|
| Contributors must explicitly configure | Prompt respects user intent |
| Upstream remote is unreliable | Many contributors don't set it up |
| Humans make mistakes | Push failure catches wrong answers |
| Provider diversity | Error parsing is provider-agnostic |
| Token requirement | No API calls needed |

## Alternatives Considered

### 5-Tier Detection (Rejected)

```
config → cache → upstream → API → heuristic
```

**Why rejected:**
- Over-engineered for the actual problem
- API tier requires token and is GitHub-specific
- Upstream remote check is unreliable
- Solves a workflow problem with detection infrastructure

### Upstream Remote Only (Rejected)

Check for `upstream` remote as fork signal.

**Why rejected:**
- Many contributors clone their fork and never add upstream
- Would miss the common case

### Require Explicit Flag (Rejected)

Always require `bd init --contributor` for contributors.

**Why rejected:**
- Breaking change
- Worse UX for new users
- No recovery path for mistakes

## Consequences

### Positive

- Simpler implementation (~50 lines vs ~500 lines)
- Works with any git provider
- No external dependencies (tokens, APIs)
- Clear recovery path for mistakes

### Negative

- User may discover mistake late (on first sync)
- Existing issues stay in wrong location until manually migrated

### Neutral

- `beads.role` git config becomes source of truth
- Recovery is guidance-only (points to existing commands)
