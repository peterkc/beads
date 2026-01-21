# Stage Definitions

Detailed scope and exit criteria for each bdx implementation stage.

## Prerequisites (Before Stage 1)

| Item | Status | Notes |
|------|--------|-------|
| Open PRs merged | ⏸️ Waiting | #1148, #1153, #1186, #1200 |
| beads-next synced | ⏸️ Blocked | After PRs merge |
| Characterization baseline | ⏸️ Blocked | After sync |
| ADRs 0001-0004 | ✅ Done | Daemon, ISP, Docs, CI |

---

## Stage 1: Foundation

**Goal**: Establish patterns and prove approach with minimal risk.

### Scope

1. **Characterization test baseline**
   - Run full test suite on synced codebase
   - Document coverage metrics
   - Tag/mark tests that verify CLI behavior

2. **Git interface extraction** (tracer bullet)
   - Define `ports.GitRepository` interface
   - Wrap `WorktreeManager` as adapter
   - Write tests against interface
   - Validate pattern works

3. **cmd/bdx stub**
   - Minimal binary that compiles
   - Shares code with bd where possible
   - Proves build chain works

4. **Directory structure**
   - Create `internal/ports/`
   - Create `internal/adapters/`
   - Do NOT move existing code yet

### Exit Criteria

- [ ] All v0 tests pass (baseline established)
- [ ] `ports.GitRepository` interface defined and tested
- [ ] `adapters/git` wraps existing WorktreeManager
- [ ] `cmd/bdx` builds and runs `bdx version`
- [ ] CI validates both `bd` and `bdx` build

### Not In Scope

- Moving existing code to `internal/v0/`
- Storage interface changes
- RPC interface extraction
- Any CLI command changes

---

## Stage 2: Pluginize

**Goal**: Migrate CLI commands to use interfaces without changing behavior.

### Scope

1. **Storage interface refinement**
   - Evaluate ISP split (IssueRepo, DependencyRepo, etc.)
   - Keep existing interface if split not needed
   - Add missing interfaces if discovered

2. **RPC client interface**
   - Define `ports.DaemonClient` interface
   - Wrap existing client as adapter

3. **CLI command migration**
   - Commands use interfaces, not concrete types
   - One command at a time
   - `bd` and `bdx` produce identical output

4. **Comparison tooling**
   - `scripts/compare-bd-bdx.sh`
   - Validates output parity

### Exit Criteria

- [ ] All characterization tests pass
- [ ] `bd` and `bdx` produce identical output for all commands
- [ ] CLI code uses interfaces exclusively
- [ ] Comparison script validates parity

### Not In Scope

- New features
- Performance optimization
- UI changes

---

## Stage 3: Cleanup

**Goal**: Remove scaffolding, polish for release.

### Scope

1. **Code organization**
   - Move remaining v0 code or delete
   - Clean package structure
   - Remove temporary adapters

2. **Platform support**
   - Windows validation
   - Nix flake updates

3. **Documentation**
   - Update ARCHITECTURE.md
   - Migration guide for users
   - CHANGELOG for v1.0

4. **Release preparation**
   - Version bump to v1.0.0
   - Release workflow (`release-bdx.yml`)
   - Homebrew/package updates

### Exit Criteria

- [ ] All tests pass on Linux, macOS, Windows
- [ ] Documentation updated
- [ ] v1.0.0 release published
- [ ] Old v0 workflows removed

---

## Stage Summary

| Stage | Focus | Duration Est. | Risk |
|-------|-------|---------------|------|
| **1. Foundation** | Patterns, tracer bullet | 1-2 weeks | Low |
| **2. Pluginize** | Migration, parity | 3-4 weeks | Medium |
| **3. Cleanup** | Polish, release | 1-2 weeks | Low |

**Total estimated**: 5-8 weeks (after prerequisites)

---

## Open Questions

1. **Storage ISP split** — Is it needed, or is current interface fine?
2. **Daemon architecture** — Any changes needed for v1?
3. **Feature freeze** — Any v0 features to add before migration?
