# bdx Roadmap

Architecture rewrite of beads using hexagonal (ports/adapters) pattern.

**Status**: Planning complete, awaiting prerequisites

---

## Prerequisites

Before Stage 1 can begin:

- [ ] PR #1148 merged (migrate sync orphan branch)
- [ ] PR #1153 merged (.beads/var/ layout)
- [ ] PR #1186 merged (init contributor prompt)
- [ ] PR #1200 merged (orphans --db flag)
- [ ] beads-next synced with upstream
- [ ] Characterization test baseline established

---

## Stage 1: Foundation

**Goal**: Establish patterns and prove approach with minimal risk.

- [ ] Run full test suite, document baseline coverage
- [ ] Define `ports.GitRepository` interface (tracer bullet)
- [ ] Create `adapters/git` wrapping WorktreeManager
- [ ] Create `cmd/bdx` stub (builds, runs `bdx version`)
- [ ] Create `internal/ports/` and `internal/adapters/` directories
- [ ] CI validates both `bd` and `bdx` build

**Exit criteria**: All v0 tests pass, Git interface works, bdx builds.

---

## Stage 2: Pluginize

**Goal**: Migrate CLI to use interfaces without changing behavior.

- [ ] Evaluate Storage interface ISP split
- [ ] Define `ports.DaemonClient` interface
- [ ] Migrate CLI commands to use interfaces
- [ ] Create `scripts/compare-bd-bdx.sh`
- [ ] Validate `bd` and `bdx` produce identical output

**Exit criteria**: Output parity between bd and bdx for all commands.

---

## Stage 3: Cleanup

**Goal**: Remove scaffolding, polish for v1.0 release.

- [ ] Clean package structure
- [ ] Windows validation
- [ ] Nix flake updates
- [ ] Update documentation (ARCHITECTURE.md, migration guide)
- [ ] v1.0.0 release

**Exit criteria**: All platforms pass, docs updated, released.

---

## Timeline

| Stage | Estimate | Dependencies |
|-------|----------|--------------|
| Prerequisites | Waiting on upstream | PRs merged |
| Stage 1 | 1-2 weeks | Prerequisites |
| Stage 2 | 3-4 weeks | Stage 1 |
| Stage 3 | 1-2 weeks | Stage 2 |

**Total**: 5-8 weeks after prerequisites

---

## Resources

- [Architecture ADRs](docs/adr/) — Design decisions
- [Codebase Analysis](research/bdx-planning/codebase-analysis.md) — Current structure
- [Stage Definitions](research/bdx-planning/stage-definitions.md) — Detailed scope
- [Issue Tracking](https://github.com/peterkc/beads/issues) — bdx-* prefix in .beads-planning/
