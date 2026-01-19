---
name: contributor-prompt-recovery
status: draft
created: 2026-01-18
issue: GH#1174
spec_type: implementation
phases:
  - name: 'Phase 1: Init Prompt'
    type: tracer
    status: pending
    description: Add contributor prompt to `bd init` with TTY detection
    validation: '`go test ./cmd/bd/... -run TestInitPrompt -v`'
  - name: 'Phase 2: Role Helpers'
    type: mvs
    status: pending
    description: Add RepoContext role methods to context.go
    validation: '`go test ./internal/beads/... -run TestRole -v`'
  - name: 'Phase 3: Migration Infrastructure'
    type: mvs
    status: pending
    description: Update routing.go with deprecation warning, add doctor check, update docs
    validation: '`go test ./cmd/bd/... -run TestDoctor -v && go test ./internal/routing/... -v`'
  - name: 'Phase 4: Push Error Detection'
    type: mvs
    status: pending
    description: Detect permission errors and show recovery guidance
    validation: '`go test ./cmd/bd/... -run TestPushErrorDetection -v`'
  - name: 'Phase 5: Closing'
    type: closing
    status: pending
    merge_strategy: pr
success_criteria:
  - 'SC-001: Plain `bd init` prompts "Contributing to someone else''s repo?"'
  - 'SC-002: `bd init --contributor` skips prompt, runs wizard directly'
  - 'SC-003: Push failure with 403/permission-denied shows recovery guidance'
  - 'SC-004: Recovery guidance points to existing commands (no new commands)'
  - 'SC-005: Reinit respects existing `beads.role` config'
---

# Spec: Contributor Prompt Recovery

## Summary

Simplify contributor detection using prompt at init + push-fail recovery instead of 5-tier detection.

## Problem Statement

The current 5-tier detection system (config→cache→upstream→API→heuristic) is over-engineered and unreliable for SSH fork users. This spec replaces it with explicit user declaration + push-fail recovery.

## Skills

- golang
- commit

## Scope

**In Scope**:

- `cmd/bd/init.go` - Add contributor prompt before wizard selection
- `cmd/bd/sync_git.go` - Add `isPushPermissionDenied()` function
- `cmd/bd/sync.go` - Show recovery guidance on push failure
- `cmd/bd/doctor.go` - Add `checkBeadsRole()` check
- `internal/beads/context.go` - Add `Role()`, `IsContributor()`, `IsMaintainer()`
- `internal/routing/routing.go` - Keep URL heuristic with deprecation warning
- `docs/QUICKSTART.md` - Update with prompt behavior

**Out of Scope**:

- New CLI commands (recovery uses existing `bd init --contributor`)
- GitHub/GitLab API integration
- Automatic fork detection

## Gotchas

| Gotcha | Risk | Mitigation |
|--------|------|------------|
| TTY detection | Prompt in non-interactive contexts (CI, scripts) blocks automation | Check `isatty(stdin)` before prompting; skip if false |
| Worktree config | `beads.role` is repo-scoped, not worktree-scoped | Config reads from main worktree's `.git/config`; document this behavior |
| Signal handling | Ctrl+C during prompt leaves terminal in bad state | Use `term.ReadPassword` or equivalent that handles SIGINT cleanly |
| Existing users | No `beads.role` configured; would break if we require it | Graceful degradation: URL heuristic continues + deprecation warning |

### TTY Detection Pattern

```go
import "golang.org/x/term"

func shouldPrompt() bool {
    // Skip prompt in non-interactive contexts
    return term.IsTerminal(int(os.Stdin.Fd()))
}
```

## Unknowns

Items to discover during implementation:

| Unknown | Discovery Task | Phase |
|---------|---------------|-------|
| Wizard state complexity | Review `runContributorWizard` for side effects that need rollback | Phase 1 |
| Git hook interactions | Test if hooks fire during config writes; may need `--no-verify` | Phase 2 |
| Existing test patterns | Check `cmd/bd/init_test.go` for mock patterns to follow | Phase 1 |
| Prompt library choice | Evaluate `survey` vs raw `bufio.Reader` vs `term` package | Phase 1 |
| Error message corpus | Collect real 403/permission errors from GitHub, GitLab, Bitbucket | Phase 3 |

### Rollback Strategy

If the prompt causes user friction (negative feedback):
1. Add `--no-prompt` flag to skip without removing feature
2. Consider inverting default: prompt only if `--interactive` flag
