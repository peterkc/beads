# Tasks: Contributor Prompt Recovery

## Test Matrix

| Scenario | Command | Before | After | Status |
|----------|---------|--------|-------|--------|
| Fresh init, answer N | `bd init` → N | No prompt | Prompt, set maintainer | 🔲 |
| Fresh init, answer Y | `bd init` → Y | No prompt | Prompt, run contributor wizard | 🔲 |
| Init with --contributor | `bd init --contributor` | Wizard | Skip prompt, run wizard | 🔲 |
| Init with --team | `bd init --team` | Wizard | Skip prompt, run wizard | 🔲 |
| Reinit, keep role | `bd init` (role exists) → N | Re-runs wizard | Show current, keep | 🔲 |
| Reinit, change role | `bd init` (role exists) → Y | Re-runs wizard | Clear config, re-prompt | 🔲 |
| SSH fork user | `bd create` (SSH remote) | Detected as maintainer | Uses beads.role config | 🔲 |
| HTTPS user | `bd create` (HTTPS remote) | Detected as contributor | Uses beads.role config | 🔲 |
| No config set | `bd create` (no beads.role) | URL heuristic silently | URL heuristic + warning | 🔲 |
| Push denied (GitHub) | `bd sync` → 403 | Generic error | Show recovery guidance | 🔲 |
| Push denied (GitLab) | `bd sync` → permission denied | Generic error | Show recovery guidance | 🔲 |
| Push denied (generic) | `bd sync` → not allowed | Generic error | Show recovery guidance | 🔲 |
| Push succeeds | `bd sync` → OK | Normal | No change | 🔲 |
| Non-permission error | `bd sync` → network error | Generic error | No guidance (pass through) | 🔲 |
| RepoContext.Role() | Config exists | — | Returns (role, true) | 🔲 |
| RepoContext.Role() | No config | — | Returns ("", false) | 🔲 |
| RepoContext.RequireRole() | Config exists | — | Returns nil | 🔲 |
| RepoContext.RequireRole() | No config | — | Returns ErrRoleNotConfigured | 🔲 |
| bd doctor | No beads.role | — | Warning + "Fix: bd init" | 🔲 |
| bd doctor | Has beads.role | — | OK + shows role | 🔲 |
| IsContributor() | role=contributor | — | Returns true | 🔲 |
| IsMaintainer() | role=maintainer | — | Returns true | 🔲 |
| IsContributor() | No config | — | Returns false | 🔲 |
| Existing .beads/, no role | `bd init` | Full wizard | Prompt role only, skip wizard | 🔲 |
| Stale config | .beads/ missing, config exists | — | Warn about stale config | 🔲 |
| Invalid config | beads.role=invalid | — | Treat as not configured | 🔲 |
| No remote | `bd create` (no origin) | Contributor | Heuristic (contributor) + warning | 🔲 |
| Auth error | `bd sync` → authentication failed | — | No guidance (not permission error) | 🔲 |
| **REGRESSION**: Existing maintainer | `bd create` (SSH, no changes) | Works | Still works (heuristic + warning) | 🔲 |
| **REGRESSION**: Existing contributor | `bd create` (HTTPS, no changes) | Works | Still works (heuristic + warning) | 🔲 |
| **REGRESSION**: Non-interactive | `bd create --title "X"` in script | Works | Still works (no prompt) | 🔲 |

---

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

4. Add role helpers to `internal/beads/context.go`
   - `Role() (UserRole, bool)` — reads git config fresh each call
   - `IsContributor()`, `IsMaintainer()` — convenience checks
   - `RequireRole()` — returns error if not configured

5. Update `internal/routing/routing.go`
   - Config check first, URL heuristic fallback with warning
   - Show deprecation warning when using heuristic
   - Keep existing users working (graceful degradation)

6. Add `checkBeadsRole()` to `cmd/bd/doctor.go`
   - Status: warning if not configured
   - Fix: `bd init` (not a new command)

7. Update `docs/QUICKSTART.md` with prompt behavior

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
