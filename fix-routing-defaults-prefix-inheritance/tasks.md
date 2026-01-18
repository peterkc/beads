# Tasks

## Phase 1: Fix Routing Default

**Goal**: Change the default so fresh `bd init` + `bd create` works locally.

### Tasks

- [ ] Change `routing.mode` default from `"auto"` to `""` in `internal/config/config.go:103`
- [ ] Add test verifying default routing mode is empty
- [ ] Update `docs/ROUTING.md` to clarify auto-routing requires opt-in
- [ ] Update `docs/CONTRIBUTOR_NAMESPACE_ISOLATION.md` code example
- [ ] Run existing routing tests to verify no regressions

### Validation

```bash
# Run config tests
go test -v ./internal/config/... -run "RoutingMode"

# Run routing tests
go test -v ./internal/routing/...

# Manual verification: fresh init (should work locally)
cd /tmp && rm -rf test-tracer && mkdir test-tracer && cd test-tracer
git init && git remote add origin /tmp/gh1165-bare
bd init --prefix tracer
bd create "Test issue" -p 2
bd list  # Should show tracer-xxx locally

# Regression check: --contributor flag still works
cd /tmp && rm -rf test-contrib ~/.beads-planning && mkdir test-contrib && cd test-contrib
git init && git remote add origin /tmp/gh1165-bare
bd init --prefix contrib --contributor
bd create "Test issue" -p 2
ls ~/.beads-planning/.beads/  # Should exist
```

### Exit Criteria

- [ ] `go test -v ./internal/config/... -run "RoutingMode"` passes
- [ ] `go test -v ./internal/routing/...` passes (0 failures)
- [ ] Manual verification: fresh init creates issue locally
- [ ] Regression check: `--contributor` still routes to ~/.beads-planning

---

## Phase 2: Closing

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

Changed `routing.mode` default from `"auto"` to `""` (disabled).

Auto-routing now requires explicit opt-in via:
- `bd init --contributor` flag, OR
- `bd config set routing.mode auto`

## Test Plan

- [ ] `go test ./...` passes
- [ ] Manual verification: fresh init + create works locally
```
