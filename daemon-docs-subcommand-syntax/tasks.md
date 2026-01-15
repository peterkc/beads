# Tasks

## Phase 1: Tracer Bullet

Validate replacement pattern on a single file before bulk application.

### Tasks

- [ ] Select `docs/DAEMON.md` as test file (5 occurrences, core daemon docs)
- [ ] Create backup: `cp docs/DAEMON.md docs/DAEMON.md.bak`
- [ ] Apply replacement pattern using `sd`:
  ```bash
  sd 'daemon --stop-all' 'daemon killall' docs/DAEMON.md
  sd 'daemon --health' 'daemon status --all' docs/DAEMON.md
  sd 'daemon --start' 'daemon start' docs/DAEMON.md
  sd 'daemon --stop(\s|$)' 'daemon stop$1' docs/DAEMON.md
  sd 'daemon --status' 'daemon status' docs/DAEMON.md
  ```
- [ ] Verify no deprecated patterns remain: `rg "daemon --" docs/DAEMON.md`
- [ ] Verify commands work: manually test one example from docs
- [ ] Remove backup if successful

### Validation

```bash
# Zero deprecated patterns in test file
rg "daemon --(start|stop|status)" docs/DAEMON.md | wc -l
# Expected: 0

# Commands still valid syntax
bd daemon start --help >/dev/null && echo "start: OK"
bd daemon stop --help >/dev/null && echo "stop: OK"
bd daemon status --help >/dev/null && echo "status: OK"
```

---

## Phase 2: Bulk Updates

Apply validated pattern to remaining files.

### Tasks

- [ ] Update high-priority docs: [P]
  - [ ] `docs/PROTECTED_BRANCHES.md` (12 occurrences)
- [ ] Update integration docs: [P]
  - [ ] `integrations/beads-mcp/SETUP_DAEMON.md` (4)
  - [ ] `integrations/beads-mcp/README.md` (1)
- [ ] Update plugin docs: [P]
  - [ ] `claude-plugin/commands/daemon.md` (4)
  - [ ] `claude-plugin/skills/beads/resources/TROUBLESHOOTING.md` (1)
- [ ] Update examples: [P]
  - [ ] `examples/team-workflow/README.md` (4)
  - [ ] `examples/protected-branch/README.md` (3)
  - [ ] `examples/multiple-personas/README.md` (1)
  - [ ] `examples/multi-phase-development/README.md` (1)
- [ ] Update remaining docs: [P]
  - [ ] `docs/WORKTREES.md` (1)
  - [ ] `docs/TROUBLESHOOTING.md` (1)

### Validation

```bash
# Count remaining deprecated patterns (excluding CHANGELOG)
rg "daemon --(start|stop|status|stop-all|health)" --type md | grep -v CHANGELOG.md | wc -l
# Expected: 0

# Total replacements made
git diff --stat | tail -1
```

---

## Phase 3: Verification

Final validation before PR.

### Tasks

- [ ] Run full grep scan to confirm zero deprecated patterns
- [ ] Spot-check 3 files for correct formatting:
  - [ ] `docs/DAEMON.md` — core docs
  - [ ] `docs/PROTECTED_BRANCHES.md` — highest count
  - [ ] `claude-plugin/commands/daemon.md` — plugin docs
- [ ] Verify CHANGELOG.md unchanged: `git diff CHANGELOG.md`
- [ ] Test replacement commands actually work:
  ```bash
  bd daemon start --help
  bd daemon stop --help
  bd daemon status --help
  bd daemon killall --help
  bd daemon status --all --help
  ```
- [ ] Create PR with conventional commit

### Validation

```bash
# Final check - no deprecated patterns outside CHANGELOG
rg "daemon --(start|stop|status|stop-all|health)" --type md -l | grep -v CHANGELOG.md
# Expected: empty output

# CHANGELOG preserved
git diff --name-only | grep -v CHANGELOG.md | wc -l
# Expected: 12 (all modified files, none being CHANGELOG)
```

### PR Template

```markdown
## Summary

Update documentation to use new `bd daemon` subcommand syntax.

Fixes #1050

## Changes

- Replace `bd daemon --start` → `bd daemon start`
- Replace `bd daemon --stop` → `bd daemon stop`
- Replace `bd daemon --status` → `bd daemon status`
- Replace `bd daemon --stop-all` → `bd daemon killall`
- Replace `bd daemon --health` → `bd daemon status --all`

## Test Plan

- [x] Verified replacement commands work (`bd daemon start --help`, etc.)
- [x] Confirmed CHANGELOG.md unchanged (historical accuracy)
- [x] Spot-checked formatting in 3 files
