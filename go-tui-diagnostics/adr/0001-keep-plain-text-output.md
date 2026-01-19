# ADR-0001: Keep Plain Text Output for bd doctor

## Status

Accepted

## Context

We evaluated whether `bd doctor` should migrate from the current `fmt.Printf` + lipgloss styling approach to a full TUI framework (tview, bubbletea) for richer output rendering.

The evaluation was triggered by issues #1170, #1171, #1172 which identified improvements needed in doctor output formatting.

## Decision

**Keep the current lipgloss-based plain text approach.** Do not adopt a TUI framework for diagnostic output.

## Rationale

### 1. CI/CD Compatibility (Non-negotiable)

Diagnostic tools frequently run in non-TTY environments:
- GitHub Actions, GitLab CI, Jenkins
- Piped output (`bd doctor | grep`)
- Redirected to files (`bd doctor > report.txt`)

TUI frameworks (tview, bubbletea) require a terminal and fail in these contexts.

### 2. Accessibility

TUI frameworks treat the terminal as a 2D canvas with screen redraws. This breaks screen readers which expect linear, append-only output. For blind users, plain CLI output is "infinitely superior" to TUI.

### 3. Industry Standard

Popular CLIs (kubectl, gh, docker) use enhanced plain text for diagnostic commands:
- TTY-aware color/formatting
- `--json` for machine-readable output
- No TUI for health checks

### 4. Already Correct Architecture

`bd doctor` already follows best practices:
- lipgloss styling with adaptive light/dark theme
- JSON export via `--json` and `--output` flags
- TTY-aware color profile selection
- Category grouping with status icons

### 5. Complexity vs Value

| Approach | Complexity | Value for Diagnostics |
|----------|------------|----------------------|
| fmt.Printf + lipgloss | Low | High (works everywhere) |
| TUI framework | High | Negative (breaks CI/accessibility) |

## Alternatives Considered

### 1. tview (Rejected)

- Pro: Mature, rich widgets
- Con: **Requires TTY** — dealbreaker for CI/CD

### 2. bubbletea (Rejected)

- Pro: Elegant Elm architecture, lipgloss integration
- Con: **Overkill** for diagnostic output, steep learning curve, non-interactive output problematic

### 3. pterm (Not Needed)

- Pro: Simple output-only library, no TTY requirement
- Con: Redundant with existing lipgloss setup

## Consequences

### Positive

- CI/CD pipelines continue to work
- Screen reader accessibility preserved
- Piped output works naturally
- No learning curve for new TUI framework
- No migration effort required

### Negative

- No interactive navigation of check results
- No real-time progress indicators (spinners) by default

### Mitigations

If interactive features are needed later:
1. Add bubbletea **only** for explicit `--interactive` mode
2. Keep plain text as the default
3. Add `--no-animation` flag for spinner-style output

## Related

- PR #1187: Fix multiline Fix message indentation (plain text improvement)
- Issues #1170, #1171, #1172: Doctor output corrections
- Research hub: `research/go-tui-diagnostics/`
