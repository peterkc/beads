# System Diagrams Research

> Visual architecture documentation for beads using C4 model and D2 diagramming.

## Status

**Status**: Active — Migration strategy documented, ready for implementation

## Decision Summary

This research hub documents the beads v1.0 (bdx) architecture redesign:

1. **Migration Strategy**: Strangler Fig with versioned directories (`internal/v0/` + `internal/next/`)
1. **Architecture**: Hexagonal (ports/adapters) with internal plugin system
1. **Testing**: Testing-first approach with characterization tests as safety net
1. **End-Game**: Replace (not merge) with git notes for traceability

## Goal

Create maintainable architecture diagrams that:

1. Help new contributors understand the codebase
1. Stay synchronized with code changes
1. Support refactoring efforts by visualizing dependencies

## Structure

```
system-diagrams/
├── README.md           # This file
├── adr/                # Architecture Decision Records
│   └── 0001-*.md       # Diagramming approach decisions
├── diagrams/           # D2 source files
│   ├── c4-context.d2   # Level 1: System Context
│   ├── c4-container.d2 # Level 2: Container
│   ├── c4-*.d2         # Level 3: Component diagrams
│   └── generated/      # Auto-generated outputs (SVG, PNG)
└── scripts/            # Automation
    └── gen-deps.sh     # go list → D2 generator
```

## Diagram Levels (C4 Model)

| Level | Diagram        | Audience         | Update Trigger        |
| ----- | -------------- | ---------------- | --------------------- |
| L1    | System Context | Everyone         | New integrations      |
| L2    | Container      | Developers       | Architecture changes  |
| L3    | Components     | Core maintainers | Package restructuring |
| Auto  | Package Deps   | CI/Review        | Every PR (generated)  |

## Quick Start

```bash
# Generate diagrams
./scripts/gen-deps.sh

# View with live reload
d2 --watch diagrams/c4-context.d2 diagrams/generated/c4-context.svg
```

## Status

- [x] C4 Level 1: System Context (`c4-context.d2`)
- [x] C4 Level 2: Container (`c4-container.d2`)
- [x] C4 Level 3: Storage layer (`c4-component-storage.d2`)
- [x] C4 Level 3: Sync layer (`c4-component-sync.d2`)
- [ ] C4 Level 3: Core/Types (future)
- [x] Automation script (`scripts/gen-deps.sh`)
- [ ] CI integration (future)

## v1.0 Architecture Proposal

Based on first principles analysis and code exploration, we've drafted a comprehensive
v1.0 architecture proposal. See [beads-v1-architecture.md](beads-v1-architecture.md).

**Key findings from analysis:**

| Issue                                | Severity | Proposed Solution               |
| ------------------------------------ | -------- | ------------------------------- |
| Storage interface bloat (62 methods) | High     | Split into 5 focused interfaces |
| RPC coupling hotspot (33 imports)    | High     | Use case based commands         |
| SQLite god package (188 methods)     | Medium   | Adapter per interface           |
| DRY violations (55 row scans)        | Medium   | Generic row mapper              |

**Next steps:**

- [ ] Create ADR for interface segregation approach
- [ ] Prototype row mapper with labels.go
- [ ] Design event bus API
- [ ] Define plugin API v1

## Related

- [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - Text-based architecture docs
- [INTERNALS.md](../../docs/INTERNALS.md) - Implementation details
- [beads-v1-architecture.md](beads-v1-architecture.md) - v1.0 redesign proposal
