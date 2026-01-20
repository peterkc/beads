# ADR 0003: Documentation Evolution Strategy

## Status

Proposed

## Date

2026-01-20

## Context

The `docs/` directory contains extensive v0 documentation (~30 files). As we implement bdx (v1), we need a strategy for evolving documentation.

**Current docs structure:**
```
docs/
├── ARCHITECTURE.md      # Three-layer data model
├── DAEMON.md            # Daemon operation
├── INTERNALS.md         # FlushManager, caches
├── ROUTING.md           # Multi-repo auto-routing
├── ADAPTIVE_IDS.md      # ID generation algorithm
├── WORKTREES.md         # Worktree support
├── ... (25+ more)
└── adr/                 # Architecture Decision Records (NEW)
```

### Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A. Evolve in place** | Update docs as code changes | Single source, no duplication | Breaks v0 docs if user on old binary |
| **B. Version directories** | `docs/v0/`, `docs/v1/` | Clear separation | Duplication, maintenance burden |
| **C. Branch-based** | v0 docs on `main`, v1 docs on `v1` branch | Git handles versioning | Confusing for contributors |
| **D. Generate from code** | GoDoc + examples, minimal prose | Always accurate | Loses conceptual docs |
| **E. Hybrid evolution** | Keep stable docs, evolve implementation docs | Best of both worlds | Requires judgment calls |

## Decision

**Use Hybrid Evolution (Option E)** with these rules:

### 1. Stable Docs (Keep As-Is)

Conceptual documentation that remains valid across versions:

- `ARCHITECTURE.md` — Three-layer model (SQLite → JSONL → Git) unchanged
- `QUICKSTART.md` — User-facing, update when CLI changes
- `INSTALLING.md` — Build instructions
- `FAQ.md` — User questions
- `WORKTREES.md` — Git worktree behavior

### 2. Implementation Docs (Evolve)

Technical docs tied to internal structure — update as v1 replaces v0:

- `INTERNALS.md` → Rewrite for v1 architecture (ports/adapters)
- `DAEMON.md` → Update for v1 daemon changes
- `ROUTING.md` → Update if routing implementation changes
- `EXCLUSIVE_LOCK.md` → Update for v1 locking

### 3. New v1 Docs (Create)

Documentation for concepts new in v1:

- `docs/PORTS_AND_ADAPTERS.md` — New architecture overview
- `docs/MIGRATION.md` — v0 → v1 migration guide (for users)
- `docs/EXTENDING.md` — How to write adapters

### 4. ADRs (Decision Record)

All architectural decisions get an ADR in `docs/adr/`. ADRs are immutable — superseded decisions get new ADRs that reference old ones.

### Update Timing

| Phase | Docs Action |
|-------|-------------|
| Stage 1 (Foundation) | Create ADRs, draft PORTS_AND_ADAPTERS.md |
| Stage 2 (Pluginize) | Update INTERNALS.md, DAEMON.md |
| Stage 3 (Cleanup) | Full docs review, MIGRATION.md |
| Release | Finalize CHANGELOG.md |

### Version Banner

For docs that change behavior between v0 and v1, add a version banner:

```markdown
> **Note:** This document describes v1 behavior. For v0, see [git tag v0.x.x](link).
```

## Consequences

### Positive

- No documentation duplication
- Gradual evolution matches code migration
- ADRs capture "why" decisions
- Users on v0 can reference tagged releases

### Negative

- Judgment required on "stable vs evolve"
- Brief window where docs may be partially outdated

### Neutral

- Existing links remain valid (no URL changes)
- Contributors update docs alongside code changes

## Checklist for PR Reviews

When reviewing PRs that change `internal/`:

- [ ] If changing adapter implementation → update relevant `docs/*.md`
- [ ] If adding new port/interface → consider new doc or ADR
- [ ] If breaking v0 behavior → add version banner to affected docs
