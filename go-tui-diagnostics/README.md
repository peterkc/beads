# Go TUI Libraries for Diagnostic CLI Output

**Status**: Complete
**Date**: 2026-01-18
**Context**: Evaluating TUI frameworks for `bd doctor` output rendering

## Decision Summary

**Recommendation**: Keep current lipgloss-based plain text approach. TUI frameworks are architecturally wrong for diagnostic CLI output.

## Research Question

Should `bd doctor` migrate from `fmt.Printf` + lipgloss styling to a full TUI framework (tview, bubbletea) for richer output rendering?

## Key Findings

| Library | Verdict | Primary Concern |
|---------|---------|-----------------|
| tview | ❌ Not recommended | Requires TTY — breaks CI/CD |
| bubbletea | ❌ Not recommended | Overkill, steep learning curve |
| pterm | ⚠️ Consider for enhancements | May be redundant with existing lipgloss |
| Current (lipgloss) | ✅ Keep | Already correct for use case |

### Critical Constraints

1. **CI/CD Compatibility**: Diagnostic tools run in non-TTY environments (pipelines, scripts)
2. **Accessibility**: TUI frameworks break screen readers (treat terminal as 2D canvas)
3. **Piped Output**: Users need `bd doctor | grep` to work
4. **JSON Output**: Already supported via `--json` flag

### Industry Standard Pattern

Popular CLIs (kubectl, gh, docker) use enhanced plain text for diagnostics:
- TTY detection for color/formatting
- `--json` flag for machine-readable output
- No TUI for health checks

## Current Architecture

`bd doctor` already follows best practices:
- lipgloss styling with adaptive light/dark (Ayu theme)
- Category grouping with status icons
- JSON export via `--json` and `--output`
- TTY-aware color profile selection

## Artifacts

- `findings.md` - Detailed research synthesis
- `adr/0001-keep-plain-text-output.md` - Architecture decision record

## Related

- PR #1187: Fix multiline Fix message indentation (plain text improvement)
- Issue #1170, #1171, #1172: Doctor output message corrections
