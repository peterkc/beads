# Test Coverage Matrix

## New Test Files

| File                                     | Tests       | Pass | Fail | Skip      |
| ---------------------------------------- | ----------- | ---- | ---- | --------- |
| `cmd/bd/git_reset_test.go`               | 24          | 12   | 0    | 10 (Dolt) |
| `cmd/bd/dolt_head_test.go`               | 3           | 3    | 0    | 0         |
| `cmd/bd/hooks_checkout_test.go`          | 6           | 1    | 1    | 4 (Dolt)  |
| `cmd/bd/init_hooks_test.go`              | new tests   | pass | 0    | 0         |
| `cmd/bd/init_templates.go`               | (impl only) | —    | —    | —         |
| `cmd/bd/template_test.go`                | new tests   | pass | 0    | 0         |
| `internal/config/yaml_config_test.go`    | 60+         | all  | 0    | 0         |
| `internal/config/config_test.go`         | 4           | 4    | 0    | 0         |
| `internal/doltserver/doltserver_test.go` | new         | pass | 0    | 0         |
| `cmd/bd/doctor_health_test.go`           | new         | pass | 0    | 0         |

## Coverage by Feature

| Feature                        | Unit Tests                                      | Integration Tests         | E2E Tests                     | Gaps                            |
| ------------------------------ | ----------------------------------------------- | ------------------------- | ----------------------------- | ------------------------------- |
| Ref writing (`writeBeadsRefs`) | nil guard, dedup                                | via autocommit            | Round-trip (Docker)           | Concurrent writes               |
| Ref reading (`readBeadsRefs`)  | 6 subtests (happy, malformed, missing, slashed) | —                         | —                             | Crafted HEAD (path traversal)   |
| Mismatch detection             | nil store, disabled                             | auto-reset, silent        | git reset round-trip (Docker) | Server mode warning             |
| `bd reset`                     | command registered                              | —                         | Real git repo (Docker)        | No-config feedback              |
| `bd check-refs`                | command registered, disabled                    | behavior (Docker)         | —                             | —                               |
| `isRebaseInProgress`           | **BROKEN** (B1)                                 | —                         | —                             | Worktree scenario (T1)          |
| `bd init` config.yaml          | —                                               | —                         | —                             | No assertion (T2)               |
| YAML config writer             | 11 update tests, 13 format tests                | —                         | —                             | Comment preservation edge cases |
| `IsBranchStrategyEnabled`      | 4 subtests                                      | —                         | —                             | Typo detection                  |
| Stale ref cleanup              | 3 tests                                         | git staging               | —                             | —                               |
| Slashed branch names           | read round-trip                                 | write round-trip (Docker) | —                             | —                               |
| Post-checkout hook             | arg filtering (3 subtests)                      | —                         | —                             | No sync call (stub)             |

## Dolt-Dependent Tests (require Docker)

These 12 tests cover the core behavioral guarantees. Require Docker with
`dolthub/dolt-sql-server:1.83.0`. **All verified passing** (2026-03-11).

| Test                                    | Result   | What It Validates                             |
| --------------------------------------- | -------- | --------------------------------------------- |
| `TestCheckBeadsRefSyncAutoReset`        | PASS     | prompt=false, reset=true auto-resets          |
| `TestCheckBeadsRefSyncSilentDivergence` | PASS     | prompt=false, reset=false is silent           |
| `TestSlashedBranchRefRoundTrip`         | PASS     | `feature/foo` branch creates nested dirs      |
| `TestCheckRefsBehavior` (3 subtests)    | PASS     | in-sync no-op, mismatch reset, disabled no-op |
| `TestCheckBeadsRefSyncNilStore`         | PASS     | nil store guard                               |
| `TestCheckBeadsRefSyncDisabledNoAction` | PASS     | disabled feature skips sync                   |
| `TestAutoResetRoundTrip`                | PASS     | backward + forward reset via `dolt_commits`   |
| `TestResetRoundTripWithGitRepo`         | PASS     | Full E2E with real git repo + Dolt            |
| `TestCheckBeadsRefSyncResetCorrectness` | PASS     | labels, comments, deps survive reset          |
| `TestCheckBeadsRefSyncInSync`           | PASS     | no action when hashes match                   |
| `TestCheckBeadsRefSyncNoRefs`           | PASS     | no action when no ref files                   |
| `TestCheckBeadsRefSyncSilentMode`       | PASS     | silent mode variant                           |
| `TestIsRebaseInProgress` (subtests 2-3) | **FAIL** | B1 confirmed — `sync.Once` cache bug          |

## Pre-existing Failures (not introduced by this PR)

| Package                       | Tests                                             | Root Cause                                 |
| ----------------------------- | ------------------------------------------------- | ------------------------------------------ |
| `github.com/steveyegge/beads` | `TestOpenFromConfig_ServerModeFailsWithoutServer` | Local Dolt server running                  |
| `internal/storage/dolt`       | 5x `TestApplyConfigDefaults_*`                    | `BEADS_DOLT_SERVER_PORT=3309` env override |
| `internal/tracker`            | 14x `TestEngine*`                                 | Docker unavailable                         |
| `cmd/bd/doctor`               | `TestDoltServerConfig_PopulatesFromConfig`        | Env var contamination                      |
| `cmd/bd/doctor/fix`           | `TestFixMissingMetadata_DoltRepair`               | Version mismatch                           |
| `internal/configfile`         | 5x `TestDoltServerMode*`                          | `BEADS_DOLT_SERVER_*` env contamination    |

## Race Check

No data races detected. Ran new beads-refs tests with `-race` flag — clean.

## Recommendations

1. **Fix B1** — `TestIsRebaseInProgress` must use `git init` or `git.ResetCaches()`
1. **Fix B2** — `TestWriteBeadsRefsDisabled` must call `writeBeadsRefs` to be meaningful
1. **Add worktree test** (T1) — Exercise the actual code path the fix was written for
1. **Add init assertion** (T2) — `TestInitCommand` should verify config.yaml exists
1. **Consider CI with Dolt** — 10 core tests are skip-only without Docker
