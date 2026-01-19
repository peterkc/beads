# beads_rust (br) Evaluation

Research hub for evaluating [beads_rust](https://github.com/Dicklesworthstone/beads_rust) as potential alternative to beads (bd).

## Summary

**Verdict**: Quality Rust implementation, excellent agent ergonomics, but architecturally frozen without multi-repo support needed for ACF.

**Recommendation**: Stay with bd, continue upstream contributions. Consider porting br's structured error patterns.

## Key Findings

| Aspect | br (Rust) | bd (Go) | Winner |
|--------|-----------|---------|--------|
| Binary size | 5.2 MB | ~30 MB | br |
| Source lines | ~40k | ~276k | br |
| Agent error handling | Structured codes + hints | Basic | br |
| Multi-repo support | None | Full | bd |
| Molecules/templates | None | Full | bd |
| Community | 1 contributor | 162 contributors | bd |
| Commit velocity | ~100/month | ~2400/month | bd |

## Files

| File | Purpose |
|------|---------|
| `feature-matrix.md` | Detailed feature comparison |
| `architecture.md` | Code architecture analysis |
| `error-patterns.md` | Structured error patterns worth porting |
| `decision.md` | Final decision rationale |

## Context

- **Evaluated**: 2026-01-19
- **br version**: 0.1.7
- **bd version**: 0.48.0
- **Local clone**: `/Volumes/atlas/beads_rust`
