# ADR 0001: Diagramming Tool Selection

## Status

Accepted

## Date

2026-01-19

## Context

Beads has extensive text documentation (ARCHITECTURE.md, INTERNALS.md) but lacks visual diagrams. As the codebase evolves rapidly with multiple contributors, we need architecture diagrams that:

1. Help new contributors understand the system
2. Stay synchronized with code changes
3. Support refactoring by visualizing dependencies
4. Are reviewable in PRs (text-based source)

### Options Considered

| Tool | Pros | Cons |
|------|------|------|
| **D2** | Text-based, C4 native, fast iteration, git-diffable | Less polish than commercial tools |
| **Python diagrams** | 3000+ cloud icons, programmatic | Overkill for Go CLI, no C4 support |
| **Mermaid** | GitHub native rendering | Limited styling, no C4 support |
| **PlantUML** | Mature, C4 extension | Verbose syntax, Java dependency |
| **Draw.io/Excalidraw** | Visual editors | Binary files, hard to diff |

### Decision Drivers

1. **Git-first**: Source files must be reviewable in PRs
2. **C4 Model**: Industry-standard architecture levels (Context → Container → Component)
3. **Automation**: Must support generation from `go list` output
4. **Low friction**: Easy to update when code changes

## Decision

Use **D2** for all architecture diagrams.

### Rationale

1. **Text-based source** — `.d2` files are git-diffable, PR reviewers see exactly what changed
2. **C4 native support** — Built-in patterns for Context → Container → Component
3. **Live reload** — `d2 --watch` enables fast iteration
4. **Automation-friendly** — Simple syntax makes generation from `go list` straightforward
5. **Multiple output formats** — SVG for docs, PNG for README, PDF for presentations

## Consequences

### Positive

- Diagrams live alongside code, versioned together
- Contributors can update diagrams with PRs
- Semi-automated generation keeps diagrams accurate
- Low barrier to entry (just text files)

### Negative

- Requires D2 CLI installation for local development
- Less visual polish than commercial tools
- Learning curve for D2 syntax (mitigated by templates)

### Mitigations

- Provide D2 templates for common diagram types
- CI can generate SVG/PNG as build artifacts
- Document common D2 patterns in research hub

## Implementation

1. Create C4 diagrams in `diagrams/` directory
2. Build `scripts/gen-deps.sh` for auto-generation
3. Add generated SVGs to `docs/` for GitHub rendering
4. Consider CI integration for diagram validation

## References

- [D2 Documentation](https://d2lang.com)
- [C4 Model](https://c4model.com)
- [beads/docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md)
