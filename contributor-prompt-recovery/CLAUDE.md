# Spec: Contributor Prompt Recovery

## Metadata

```yaml
name: contributor-prompt-recovery
status: draft
created: 2026-01-18
issue: GH#1174
epic: null
```

## Summary

Simplify contributor detection using prompt at init + push-fail recovery instead of 5-tier detection.

## Success Criteria

- SC-001: Plain `bd init` prompts "Contributing to someone else's repo?"
- SC-002: `bd init --contributor` skips prompt, runs wizard directly
- SC-003: Push failure with 403/permission-denied shows recovery guidance
- SC-004: Recovery guidance points to existing commands (no new commands)
- SC-005: Reinit respects existing `beads.role` config

## Skills

- golang
- commit

## Phases

1. Phase 1: Init Prompt (Tracer Bullet)
2. Phase 2: Push Error Detection (MVS)
3. Phase 3: Closing
