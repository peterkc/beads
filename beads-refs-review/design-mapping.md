# Gist Design to Implementation Mapping

Maps each section of the [design gist](https://gist.github.com/bryanhirsch/5f003918e13a079975a27b5f7346fc37)
to the implementation in `bryanhirsch/beads#1`.

## Phase 1 Scope (this PR)

| Gist Concept                      | Implementation                                                 | Files                                       | Status      |
| --------------------------------- | -------------------------------------------------------------- | ------------------------------------------- | ----------- |
| `.beads/HEAD` pointer file        | `writeBeadsRefs()` writes `ref: refs/heads/<branch>\n`         | `dolt_head.go:59`                           | Implemented |
| `.beads/refs/heads/<branch>`      | `writeBeadsRefs()` writes Dolt commit hash                     | `dolt_head.go:71`                           | Implemented |
| Write after every Dolt commit     | Two triggers: `transact()` + `PersistentPostRun` with dedup    | `dolt_autocommit.go`, `main.go:639`         | Implemented |
| Mismatch detection (lazy)         | `checkBeadsRefSync()` in `PersistentPreRun`                    | `dolt_head.go`, `main.go:578-583`           | Implemented |
| 4-way behavior matrix             | prompt x reset_default combinations                            | `dolt_head.go:177-226`                      | Implemented |
| `bd reset` wrapper                | `gitResetCmd` wraps `git reset` + triggers sync                | `git_reset.go:26-67`                        | Implemented |
| `bd check-refs`                   | `checkRefsCmd` — standalone sync check                         | `git_reset.go:71-106`                       | Implemented |
| `branch_strategy.*` config        | YAML-only keys, survives `DOLT_RESET`                          | `yaml_config.go`, `config.go`               | Implemented |
| Opt-in (default off)              | `IsBranchStrategyEnabled()` checks section existence           | `config.go:748-752`                         | Implemented |
| `bd init` creates config template | `createConfigYaml()` with commented section                    | `init_templates.go`                         | Implemented |
| Rebase/stash safety               | Detection in `PersistentPreRun`, not git hooks                 | `main.go:578` (only fires on `bd` commands) | Implemented |
| Slashed branch names              | `os.MkdirAll` + `filepath.Join` handles nesting                | `dolt_head.go:66-71`                        | Implemented |
| Forward reset via `dolt_commits`  | `getDoltCommitMessage` queries all commits, not just reachable | `dolt_head.go`                              | Implemented |

## Phase 2+ (NOT in this PR)

| Gist Concept                     | Phase | Notes                                 |
| -------------------------------- | ----- | ------------------------------------- |
| Strategy A (stay on main)        | 2     | Current behavior — no Dolt branching  |
| Strategy B (merge with branch)   | 2     | `beads_branches` registry table       |
| Strategy C (merge on close)      | 3     | Dedicated merge connection pattern    |
| Cross-branch message cherry-pick | 4     | Cherry-pick to all active branches    |
| `beads_branches` registry table  | 2     | `WHERE status = 'active'` iteration   |
| Post-checkout Dolt branch switch | 2     | Hook stub exists (`hooks.go:837-855`) |

## Architecture Flow

```
bd create --title "Fix bug" --type task
  |
  v
PersistentPreRun
  -> checkBeadsRefSync()       # detect mismatch from prior git operations
  -> [prompt/auto-reset/silent based on config]
  |
  v
Command execution
  -> store.Create(...)         # Dolt write
  -> commandDidWrite = true
  |
  v
PersistentPostRun
  -> maybeAutoCommit()         # DOLT_COMMIT if not explicit
       -> writeBeadsRefs()     # write .beads/HEAD + refs/heads/<branch>
  -> writeBeadsRefs()          # unconditional fallback (dedup guard prevents double-write)
       -> os.WriteFile(.beads/HEAD, "ref: refs/heads/main\n")
       -> os.WriteFile(.beads/refs/heads/main, "<dolt-hash>\n")
       -> git add .beads/HEAD .beads/refs/heads/main  (best-effort)
```

```
bd reset --hard HEAD~1
  |
  v
git reset --hard HEAD~1       # git moves back, .beads/refs revert in working tree
  -> stdout: "HEAD is now at abc1234..."
  |
  v
checkBeadsRefSyncWithGitLine()
  -> readBeadsRefs()           # read reverted ref files (old hash)
  -> s.GetCurrentCommit()      # current Dolt HEAD (still at new hash)
  -> MISMATCH detected
  -> prompt: "Reset dolt HEAD to <old-hash>? [Y/n]"
  -> s.ResetToCommit(<old-hash>)  # DOLT_RESET('--hard', <old-hash>)
  -> writeBeadsRefs()          # update refs to match new Dolt state
```

## Design Decisions Worth Noting

**YAML, not Dolt, for config**: `branch_strategy` settings live in `.beads/config.yaml`
because storing them in Dolt would be self-defeating — a `DOLT_RESET --hard` would wipe
the `reset_dolt_with_git=true` setting that controls the reset.

**Lazy, not eager detection**: Mismatch detection runs in `PersistentPreRun` (on next `bd`
command), not in the `post-checkout` git hook. This means a `git reset` without a subsequent
`bd` command leaves Dolt out of sync, but avoids adding latency to every `git checkout`.

**`dolt_commits` not `dolt_log` for forward reset**: After backward reset, the "future" commit
is unreachable from HEAD via `dolt_log`. `dolt_commits` includes all commits in the store
regardless of reachability, enabling forward reset.
