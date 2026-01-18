# Tasks

## Phase 1: Tracer Bullet - Disable Auto-Routing Default

**Goal**: Change the default so fresh `bd init` + `bd create` works locally.

### Tasks

- [ ] Change `routing.mode` default from `"auto"` to `""` in `internal/config/config.go:103`
- [ ] Add test verifying default routing mode is empty
- [ ] Run existing routing tests to verify no regressions

### Validation

```bash
# Run routing tests
go test -v ./internal/routing/...
go test -v ./cmd/bd/... -run "Routing"

# Manual verification
cd /tmp && rm -rf test-tracer && mkdir test-tracer && cd test-tracer
git init && git remote add origin /tmp/bare-repo
bd init --prefix tracer
bd create "Test issue" -p 2
bd list  # Should show tracer-xxx locally
```

### Exit Criteria

- [ ] `go test -v ./internal/config/... -run "RoutingMode"` passes
- [ ] `go test -v ./internal/routing/...` passes (0 failures)
- [ ] `go test -v ./cmd/bd/... -run "Routing"` passes (0 failures)
- [ ] Manual verification shows local issue creation

---

## Phase 2: Fix Prefix Inheritance

**Goal**: When routing is explicitly enabled, prefix is correctly inherited.

### Tasks

- [ ] Refactor `ensureBeadsDirForPath()` to use `factory.NewFromConfig()` instead of direct path
- [ ] Ensure prefix is set in target store before returning
- [ ] Add test for prefix inheritance with var/ layout
- [ ] Add debug logging for routing decisions

### Validation

```bash
# Run all create tests
go test -v ./cmd/bd/... -run "Create"

# Manual verification with explicit routing
cd /tmp && rm -rf test-prefix ~/.beads-planning
mkdir test-prefix && cd test-prefix
git init && git remote add origin /tmp/bare-repo
bd init --prefix prefix
bd config set routing.mode auto
bd create "Test issue" -p 2

# Check planning repo has correct prefix
sqlite3 ~/.beads-planning/.beads/var/beads.db "SELECT value FROM config WHERE key='issue_prefix';"
# Should show: prefix
```

### Exit Criteria

- [ ] `go test -v ./cmd/bd/... -run "Create"` passes (0 failures)
- [ ] `go test -v ./cmd/bd/... -run "PrefixInheritance"` passes
- [ ] Manual verification shows prefix inherited correctly

---

## Phase 3: Closing

**Goal**: Merge changes and clean up.

### Tasks

- [ ] Run full test suite
- [ ] Create PR with description referencing GH#1165
- [ ] Clean up worktree after merge

### Validation

```bash
# Full test suite
go test ./...

# Verify no linting issues
golangci-lint run
```

### PR Description Template

```markdown
## Summary

Fixes #1165 - Fresh `bd init` unexpectedly routes to `~/.beads-planning`

Two bugs fixed:
1. **Default auto-routing**: Changed `routing.mode` default from `"auto"` to `""` (disabled)
2. **Prefix inheritance**: Fixed `ensureBeadsDirForPath()` to use factory and inherit prefix correctly

## Test Plan

- [ ] `go test ./...` passes
- [ ] Manual verification: fresh init + create works locally
- [ ] Manual verification: explicit routing inherits prefix
```
