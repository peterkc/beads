# Design: Git Notes Bidirectional Linking

## Architecture Overview

```
bd notes add <issue>
       |
       v
  notes.go ──> git notes --ref=beads add -m '{json}'
       |
       v
  refs/notes/beads (local)
       |
  bd notes push
       |
       v
  refs/notes/beads (remote)

bd orphans --include-notes
       |
       v
  git.go ──> scanCommitMessages() + scanBeadsNotes()
       |
       v
  Combined orphan results
```

## Key Decisions

### KD-001: Namespace Selection

**Decision**: Use `refs/notes/beads` as the note namespace.

**Rationale**:
- Avoids collision with default `refs/notes/commits`
- Clear ownership by beads tool
- Matches git-appraise pattern of tool-specific namespaces

**Alternatives considered**:
- `refs/notes/commits` (default) — Risk of collision with other tools
- `refs/notes/issues` — Too generic

### KD-002: Note Format

**Decision**: Single-line JSON with issue ID, timestamp, and actor.

```json
{"issue":"bd-123","ts":"1705680000","actor":"alice"}
```

**Rationale**:
- Single-line enables `cat_sort_uniq` merge mode (conflict-free)
- JSON provides structured data for future extensions
- Matches git-appraise proven pattern

**Alternatives considered**:
- Plain text issue ID — No metadata, harder to extend
- Multi-line JSON — Merge conflicts on concurrent edits

### KD-003: Rebase Handling

**Decision**: Use `notes.rewriteRef` git configuration.

```bash
git config notes.rewrite.amend true
git config notes.rewrite.rebase true
git config notes.rewriteRef refs/notes/beads
git config notes.rewriteMode cat_sort_uniq
```

**Rationale**:
- Native git support, no custom logic needed
- `cat_sort_uniq` handles duplicate notes from concurrent work
- Well-documented in git-notes man page

### KD-004: Orphan Integration Approach

**Decision**: Add `--include-notes` flag to `bd orphans`, off by default.

**Rationale**:
- Backward compatible (existing behavior unchanged)
- Explicit opt-in for notes scanning
- Clear separation of concerns

**Alternatives considered**:
- Always scan notes — Breaking change for users without notes
- Separate `bd notes orphans` command — Fragments UX

## Applied Patterns

### From git-appraise (Google)

| Pattern | Application |
|---------|-------------|
| Tool-specific namespace | `refs/notes/beads` |
| Single-line JSON | Note format |
| cat_sort_uniq merge | Rebase config |

### From beads IssueProvider (PR#1200)

The `IssueProvider` interface enables notes integration:

```go
type IssueProvider interface {
    GetOpenIssues(ctx context.Context) ([]*Issue, error)
    GetIssuePrefix() string
}
```

Notes scanning will extract issue IDs and validate against provider.

## Implementation Notes

### Note Operations (notes.go)

```go
// AddNote creates a beads note on a commit
func AddNote(issueID, commitSHA string) error {
    note := Note{Issue: issueID, Timestamp: time.Now().Unix(), Actor: getActor()}
    json, _ := json.Marshal(note)
    return exec.Command("git", "notes", "--ref=beads", "add", "-m", string(json), commitSHA).Run()
}

// ListNotes returns all beads notes in the repo
func ListNotes() ([]NoteEntry, error) {
    // git notes --ref=beads list
    // Parse output: <note-sha> <commit-sha>
    // For each, git notes --ref=beads show <commit-sha>
}
```

### Orphan Scanning Extension (git.go)

```go
func FindOrphanedIssues(gitPath string, provider IssueProvider, includeNotes bool) ([]OrphanedIssue, error) {
    orphans := scanCommitMessages(gitPath, provider)

    if includeNotes {
        noteOrphans := scanBeadsNotes(gitPath, provider)
        orphans = mergeOrphans(orphans, noteOrphans)
    }

    return orphans, nil
}
```

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Notes lost on rebase | High | `bd notes init` configures preservation; warning on first use |
| GitHub doesn't show notes | Medium | Document limitation; CLI-first workflow |
| Large repos slow | Low | Benchmark; add `--limit` flag if needed |
| Users forget to push notes | Medium | `bd sync` could include notes push (future) |

## Future Enhancements

Not in scope for this spec, but documented for context:

1. **Auto-capture via hook** — Commit-msg hook to auto-create notes
2. **bd sync notes integration** — Include notes in standard sync workflow
3. **Multi-repo aggregation** — Query notes across multiple repos
4. **GitHub Action** — Publish notes to issue comments (workaround for UI limitation)
