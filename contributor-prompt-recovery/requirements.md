# Requirements: Contributor Prompt Recovery

## Functional Requirements

### Init Prompt

- REQ-001: WHEN user runs `bd init` without flags THE SYSTEM SHALL prompt "Contributing to someone else's repo? [y/N]"
- REQ-002: WHEN user answers [Y] THE SYSTEM SHALL run the contributor wizard
- REQ-003: WHEN user answers [N] or presses Enter THE SYSTEM SHALL proceed as maintainer
- REQ-004: WHEN user runs `bd init --contributor` THE SYSTEM SHALL skip the prompt and run contributor wizard directly
- REQ-005: WHEN user runs `bd init --team` THE SYSTEM SHALL skip the prompt and run team wizard directly

### Reinit Behavior

- REQ-006: WHEN `beads.role` git config exists THE SYSTEM SHALL skip prompt and show current setting
- REQ-007: WHEN `beads.role` exists THE SYSTEM SHALL offer "Already configured as {role}. Change? [y/N]"
- REQ-008: WHEN user runs `bd init --force` THE SYSTEM SHALL clear `beads.role` and prompt fresh

### Config Lifecycle

- REQ-009: WHEN user completes init prompt THE SYSTEM SHALL set `git config beads.role {role}`
- REQ-010: WHEN `.beads/` is missing but `beads.role` config exists THE SYSTEM SHALL warn about stale config

### Push Error Detection

- REQ-011: WHEN `bd sync` push fails with permission error THE SYSTEM SHALL detect the failure
- REQ-012: THE SYSTEM SHALL recognize patterns: "Permission denied", "403", "not allowed to push"
- REQ-013: WHEN permission error detected THE SYSTEM SHALL display recovery guidance
- REQ-014: THE SYSTEM SHALL reference existing commands (`git config beads.role`, `bd init --contributor`)
- REQ-015: THE SYSTEM SHALL NOT run wizards or prompts from sync (separation of concerns)

## Non-Functional Requirements

- NFR-001: Error parsing SHALL be provider-agnostic (GitHub, GitLab, Bitbucket, self-hosted)
- NFR-002: No external API calls SHALL be required for detection
- NFR-003: Detection SHALL work offline (git-only)
