# Research Findings: Go TUI Libraries for Diagnostic CLI Output

## Executive Summary

TUI frameworks (tview, bubbletea) are **not recommended** for diagnostic CLI output like `bd doctor`. The current lipgloss-based approach is architecturally correct and follows industry best practices.

---

## Libraries Evaluated

### tview (rivo/tview) — 13.4k ⭐

**Architecture**: Retained-mode GUI with widget-based components

**Pros**:
- Mature, production-tested (K9s, lazysql)
- Low complexity for basic UIs
- Rich ANSI color support

**Critical Flaw**: **No headless mode** — requires TTY, breaks in CI/CD

```go
// tview REQUIRES a TTY - this would fail in CI:
app := tview.NewApplication()
app.SetRoot(list, true).Run()  // ❌ Crashes without terminal
```

### bubbletea (charmbracelet) — 38.4k ⭐

**Architecture**: Elm-style functional reactive (Init → Update → View cycle)

**Pros**:
- Beautiful ecosystem (lipgloss, bubbles)
- Inline + full-window modes
- Major adoption (Microsoft Azure, AWS)

**Cons for diagnostics**:
- Steep learning curve (Elm architecture)
- Non-interactive output is problematic
- Overkill for "print status and exit"

### pterm — 5.3k ⭐

**Architecture**: Simple output-only library

```go
pterm.DefaultTable.WithData(data).Render()
pterm.Success.Println("All checks passed")
```

**Pros**: Purpose-built for diagnostic output, no TTY requirement
**Consider**: Already have lipgloss; pterm would be alternative, not upgrade

---

## Best Practices: CLI Diagnostic Output

### When to Use TUI vs Plain Text

| Use Case | Recommendation |
|----------|----------------|
| Health checks | Plain text |
| CI/CD pipelines | Plain text |
| Screen reader accessibility | Plain text |
| Interactive navigation | TUI (opt-in) |
| Real-time monitoring | TUI (opt-in) |

### Terminal Detection Pattern

```go
isInteractive := isatty.IsTerminal(os.Stdout.Fd()) &&
                 os.Getenv("TERM") != "dumb" &&
                 os.Getenv("CI") == ""
```

### Industry Standard (kubectl, gh, docker)

- Default: Plain text with optional color
- `--json` flag for machine-readable output
- Auto-detect TTY for formatting decisions
- **No TUI for diagnostic commands**

### Accessibility Concerns

> "Modern TUIs are hostile to accessibility... a dumb, linear CLI stream is infinitely superior to a 'smart' TUI."

TUI frameworks redraw the screen, breaking screen readers that expect linear output flow.

---

## Current `bd doctor` Architecture

### Already Correct

| Component | Implementation |
|-----------|---------------|
| Styling | lipgloss with Ayu theme (adaptive light/dark) |
| Structure | Category grouping, status icons |
| Machine output | `--json` flag, `--output` file export |
| TTY detection | `ShouldUseColor()` in ui package |
| Color profile | TrueColor when TTY, Ascii otherwise |

### Code Structure

```
cmd/bd/doctor.go          # Main command, printDiagnostics()
cmd/bd/doctor/*.go        # Individual checks
internal/ui/styles.go     # lipgloss styles, colors, icons
```

### Output Format

```
CATEGORY NAME
  ✓  Check Name message text
     └─ detail line

WARNINGS
  ⚠  1. Check Name: message
        └─ fix suggestion
```

---

## Trade-Off Analysis

| Dimension | Current (lipgloss) | TUI (tview/bubbletea) |
|-----------|-------------------|----------------------|
| CI/CD compatibility | ✅ Works | ❌ Fails |
| Accessibility | ✅ Linear, screen reader friendly | ❌ Breaks screen readers |
| Complexity | ✅ Simple fmt.Printf | ❌ Event loop, state |
| Piped output | ✅ Works naturally | ❌ Needs special handling |
| Interactive fixes | ⚠️ Would need addition | ✅ Natural fit |

---

## Recommendation

### Don't Migrate to TUI

The research consensus: **TUI is the wrong abstraction for diagnostic output**.

### Potential Enhancements (Without TUI)

If richer output is desired:

1. **lipgloss tables**: Built-in table support in lipgloss v1.0+
2. **Progress spinners**: For long-running checks (with `--no-animation` flag)
3. **Interactive fixes only**: Add bubbletea for `--interactive` mode (explicit opt-in)

---

## Sources

- tview: https://github.com/rivo/tview
- bubbletea: https://github.com/charmbracelet/bubbletea
- pterm: https://github.com/pterm/pterm
- Accessibility: https://xogium.me/the-text-mode-lie-why-modern-tuis-are-a-nightmare-for-accessibility
- kubectl formatting: https://www.baeldung.com/ops/kubectl-output-format
- gh CLI formatting: https://cli.github.com/manual/gh_help_formatting
