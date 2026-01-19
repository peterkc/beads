# System Diagrams Research

Visual architecture documentation for beads using C4 model and D2 diagramming.

## Goal

Create maintainable architecture diagrams that:
1. Help new contributors understand the codebase
2. Stay synchronized with code changes
3. Support refactoring efforts by visualizing dependencies

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

| Level | Diagram | Audience | Update Trigger |
|-------|---------|----------|----------------|
| L1 | System Context | Everyone | New integrations |
| L2 | Container | Developers | Architecture changes |
| L3 | Components | Core maintainers | Package restructuring |
| Auto | Package Deps | CI/Review | Every PR (generated) |

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

## Related

- [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - Text-based architecture docs
- [INTERNALS.md](../../docs/INTERNALS.md) - Implementation details
