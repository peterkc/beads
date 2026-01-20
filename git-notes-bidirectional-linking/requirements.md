# Requirements: Git Notes Bidirectional Linking

## Functional Requirements

### FR-001: Add Note to Commit

WHEN the user runs `bd notes add <issue-id>` THE SYSTEM SHALL create a git note on HEAD in `refs/notes/beads` containing the issue reference.

**Acceptance**: `git notes --ref=beads show HEAD` returns the issue ID.

### FR-002: Add Note to Specific Commit

WHEN the user runs `bd notes add <issue-id> <commit-sha>` THE SYSTEM SHALL create a git note on the specified commit in `refs/notes/beads`.

**Acceptance**: `git notes --ref=beads show <sha>` returns the issue ID.

### FR-003: List Notes

WHEN the user runs `bd notes list` THE SYSTEM SHALL display all beads notes in the current repository with commit SHA and issue reference.

**Acceptance**: Output shows `<sha> <issue-id>` pairs, one per line.

### FR-004: List Notes JSON

WHEN the user runs `bd notes list --json` THE SYSTEM SHALL output notes as a JSON array with `commit`, `issue_id`, and `timestamp` fields.

**Acceptance**: Valid JSON array parseable by `jq`.

### FR-005: Push Notes to Remote

WHEN the user runs `bd notes push` THE SYSTEM SHALL push `refs/notes/beads` to the default remote.

**Acceptance**: Remote ref updated, verifiable via `git ls-remote`.

### FR-006: Fetch Notes from Remote

WHEN the user runs `bd notes fetch` THE SYSTEM SHALL fetch `refs/notes/beads` from the default remote and merge with local notes.

**Acceptance**: Remote notes visible in `bd notes list`.

### FR-007: Initialize Rebase Preservation

WHEN the user runs `bd notes init` THE SYSTEM SHALL configure git to preserve notes during rebase/amend via `notes.rewriteRef`.

**Acceptance**: `git config notes.rewriteRef` returns `refs/notes/beads`.

### FR-008: Orphan Detection with Notes

WHEN the user runs `bd orphans --include-notes` THE SYSTEM SHALL scan both commit messages AND git notes for issue references.

**Acceptance**: Commits with only notes-based references (no message pattern) appear in orphan list.

### FR-009: Note Format

THE SYSTEM SHALL store notes as single-line JSON: `{"issue":"<id>","ts":"<unix>","actor":"<name>"}`.

**Rationale**: Single-line JSON enables conflict-free merges via `cat_sort_uniq` mode.

## Non-Functional Requirements

### NFR-001: Rebase Safety

THE SYSTEM SHALL document that notes may be lost during rebase without `bd notes init` configuration.

**Mitigation**: `bd notes init` warning displayed on first use if config not set.

### NFR-002: GitHub Limitation

THE SYSTEM SHALL document that GitHub does not display git notes in its web UI.

**Mitigation**: Note in documentation; local/CLI workflow is primary use case.

### NFR-003: Performance

THE SYSTEM SHALL complete `bd notes list` in under 1 second for repositories with up to 10,000 notes.

**Validation**: Benchmark test with synthetic notes.

## Test Matrix

| ID | Scenario | Input | Expected | Status |
|----|----------|-------|----------|--------|
| T-001 | Add note to HEAD | `bd notes add bd-123` | Note created on HEAD | Pending |
| T-002 | Add note to specific commit | `bd notes add bd-123 abc123` | Note created on abc123 | Pending |
| T-003 | List notes | `bd notes list` | Shows all notes | Pending |
| T-004 | List notes JSON | `bd notes list --json` | Valid JSON output | Pending |
| T-005 | Orphan with notes | Commit with note, no message ref | Found by --include-notes | Pending |
| T-006 | Orphan without notes | Same commit | NOT found without flag | Pending |
| T-007 | Push notes | `bd notes push` | Remote ref updated | Pending |
| T-008 | Fetch notes | `bd notes fetch` | Local notes updated | Pending |
| T-009 | Init config | `bd notes init` | rewriteRef configured | Pending |
| T-010 | Rebase preserves notes | Rebase after init | Notes follow commits | Pending |
