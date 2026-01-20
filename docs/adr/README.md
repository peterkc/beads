# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for bdx (beads v1).

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-daemon-version-supersession.md) | Daemon Version Supersession | Accepted | 2026-01-20 |
| [0002](0002-interface-segregation.md) | Interface Segregation (Ports & Adapters) | Proposed | 2026-01-20 |
| [0003](0003-documentation-evolution.md) | Documentation Evolution Strategy | Proposed | 2026-01-20 |
| [0004](0004-ci-workflow-strategy.md) | CI/CD Workflow Strategy | Accepted | 2026-01-20 |

## What is an ADR?

An ADR documents a significant architectural decision along with its context and consequences. ADRs are immutable once accepted — superseded decisions get a new ADR that references the old one.

## When to Create an ADR

Create an ADR when:

- Introducing a new interface or port
- Changing plugin contracts
- Adding external dependencies
- Making breaking changes to internal APIs
- Choosing between multiple valid approaches

## ADR Template

```markdown
# ADR NNNN: Title

## Status

Proposed | Accepted | Deprecated | Superseded by [ADR-XXXX](XXXX-title.md)

## Context

What is the issue that we're seeing that is motivating this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or more difficult because of this change?

### Positive

- ...

### Negative

- ...

### Neutral

- ...
```

## File Naming

`NNNN-kebab-case-title.md`

Examples:
- `0001-interface-segregation.md`
- `0002-plugin-architecture.md`

## Related

- [Migration ADRs](../../research/system-diagrams/adr/) — v0→v1 migration decisions (temporary)
- [ARCHITECTURE.md](../ARCHITECTURE.md) — System overview
