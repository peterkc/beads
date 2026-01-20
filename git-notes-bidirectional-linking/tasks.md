# Tasks: Git Notes Bidirectional Linking

## Phase 1: Tracer Bullet

End-to-end skeleton: `bd notes add` creates a note, `bd notes list` shows it.

### Tasks

- [ ] Create `cmd/bd/notes.go` with Cobra subcommand structure
- [ ] Implement `Note` struct in `internal/types/notes.go`
  ```go
  type Note struct {
      Issue     string `json:"issue"`
      Timestamp int64  `json:"ts"`
      Actor     string `json:"actor"`
  }
  ```
- [ ] Implement `AddNote(issueID, commitSHA string)` using git exec
- [ ] Implement `ListNotes()` parsing git notes output
- [ ] Add `bd notes add <issue-id> [commit]` command
- [ ] Add `bd notes list [--json]` command
- [ ] Write unit tests for Note parsing
- [ ] Write integration test: add note → list shows it

### Phase Validation

```bash
# Create test repo
cd /tmp && mkdir test-notes && cd test-notes && git init
echo "test" > file.txt && git add . && git commit -m "initial"

# Test add
bd notes add TEST-001
git notes --ref=beads show HEAD  # Should show JSON

# Test list
bd notes list  # Should show commit + TEST-001
bd notes list --json | jq .  # Should be valid JSON
```

### Exit Criteria

- `bd notes add` creates note on HEAD
- `bd notes list` shows note with issue ID
- `go test ./cmd/bd/... ./internal/types/... -v` passes

---

## Phase 2: Orphan Integration

Add `--include-notes` flag to `bd orphans` command.

### Tasks

- [ ] Add `scanBeadsNotes(gitPath string) []string` function in `cmd/bd/doctor/git.go`
- [ ] Parse note JSON to extract issue IDs
- [ ] Add `--include-notes` flag to orphans command
- [ ] Update `FindOrphanedIssues` signature to accept `includeNotes bool`
- [ ] Merge note-based orphans with message-based orphans
- [ ] Write test: commit with note-only reference detected
- [ ] Write test: flag disabled = notes not scanned

### Validation

```bash
# Setup: commit with note, no message reference
cd /tmp/test-notes
echo "change" >> file.txt && git add . && git commit -m "no issue ref"
bd notes add TEST-002

# Initialize beads with test issue
bd init --prefix=TEST
bd create -t task "Test issue"  # Creates TEST-xxx

# Test orphan detection
bd orphans                      # Should NOT show TEST-002
bd orphans --include-notes      # SHOULD show TEST-002
```

### Exit Criteria

- `--include-notes` flag works
- Commits with only notes-based references are found
- Default behavior (no flag) unchanged

---

## Phase 3: Sync & Init

Push/fetch commands and rebase preservation configuration.

### Tasks

- [ ] Implement `bd notes push [remote]` — pushes refs/notes/beads
- [ ] Implement `bd notes fetch [remote]` — fetches and merges refs/notes/beads
- [ ] Implement `bd notes init` — configures rewriteRef for rebase safety
- [ ] Add first-use warning if init not run
- [ ] Write test: push to remote, clone, fetch shows notes
- [ ] Write test: rebase preserves notes after init
- [ ] Update docs/CLI_REFERENCE.md with all notes commands

### Validation

```bash
# Test init
bd notes init
git config notes.rewriteRef  # Should show refs/notes/beads

# Test push/fetch (requires remote)
# In real repo with remote:
bd notes push
# In fresh clone:
bd notes fetch
bd notes list  # Should show pushed notes

# Test rebase preservation
git checkout -b feature
echo "feature" >> file.txt && git add . && git commit -m "feature"
bd notes add TEST-003
git rebase main
bd notes list  # TEST-003 should still be attached
```

### Exit Criteria

- Push/fetch work with default remote
- Init configures all required git settings
- Rebase preserves notes after init
- Documentation complete

---

## Phase 4: Closing

Create PR and clean up worktree.

### Tasks

- [ ] Run full test suite
- [ ] Verify documentation is complete
- [ ] Create draft PR with:
  - Summary of changes
  - Test matrix results
  - Link to spec and research
- [ ] Address review feedback
- [ ] Merge PR
- [ ] Clean up worktree

### PR Description Template

```markdown
## Summary

Add native git notes support for bidirectional commit↔issue linking.

**Spec**: specs/git-notes-bidirectional-linking/
**Research**: research/git-notes-cross-repo/
**Related**: #1196

## Changes

| File | Change |
|------|--------|
| `cmd/bd/notes.go` | NEW: Notes subcommand |
| `internal/types/notes.go` | NEW: Note struct |
| `cmd/bd/doctor/git.go` | UPDATE: --include-notes flag |
| `docs/CLI_REFERENCE.md` | UPDATE: Notes documentation |

## Test Results

| Test | Status |
|------|--------|
| T-001: Add note to HEAD | ✅ |
| T-002: Add note to specific commit | ✅ |
| ... | ... |
```

### Exit Criteria

- PR merged to main
- Worktree removed
- Beads issues closed
