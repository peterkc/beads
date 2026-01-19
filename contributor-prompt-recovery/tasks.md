# Tasks: Contributor Prompt Recovery

## Phase 1: Init Prompt

**Type**: Tracer Bullet
**Goal**: Add contributor prompt to plain `bd init`

### Tasks

1. Add `promptContributorMode()` function to `cmd/bd/init.go`
   - Check for existing `beads.role` git config
   - If exists: show current, offer to change
   - If not: prompt "Contributing to someone else's repo? [y/N]"

2. Integrate prompt into init flow
   - Before wizard selection logic
   - Skip if `--contributor` or `--team` flag present
   - Set `git config beads.role` based on answer

3. Add unit tests for prompt logic
   - Test existing config detection
   - Test flag bypass (`--contributor`, `--team`)
   - Test config setting after prompt

4. Remove URL heuristic from `internal/routing/routing.go`
   - `DetectUserRole()` should only check `beads.role` config
   - Return `ErrRoleNotConfigured` if not set
   - Update callers to handle unconfigured state

5. Update `docs/QUICKSTART.md` with prompt behavior

### Validation

```bash
go build ./cmd/bd/...
go test ./cmd/bd/... -run TestInitPrompt -v
```

---

## Phase 2: Push Error Detection

**Type**: MVS Slice
**Goal**: Detect permission errors and show helpful guidance

### Tasks

1. Add `isPushPermissionDenied()` to `cmd/bd/sync_git.go`
   - Pattern match common error messages
   - Provider-agnostic (GitHub, GitLab, Bitbucket, self-hosted)

2. Integrate detection into `gitPush()` error handling
   - On permission error: show recovery guidance
   - Point to `git config beads.role contributor` and `bd init --contributor`
   - Reference `docs/ROUTING.md` for full setup

3. Add unit tests for error detection
   - Test GitHub 403 error
   - Test GitLab permission denied
   - Test generic permission errors
   - Test non-permission errors (don't trigger)

### Validation

```bash
go test ./cmd/bd/... -run TestPushErrorDetection -v
```

---

## Phase 3: Closing

**Type**: Closing
**Merge Strategy**: PR

### Tasks

1. Run full test suite
   ```bash
   go test ./... -v
   golangci-lint run ./...
   ```

2. Final documentation review
   ```bash
   lychee --offline docs/*.md
   ```

3. Create PR against upstream
   - Title: `feat(init): add contributor prompt and push-fail recovery`
   - Reference GH#1174

4. Clean up worktree after merge
