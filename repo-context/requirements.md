# Requirements: Centralize CWD/BEADS_DIR Resolution

## Functional Requirements

### FR-001: Repository Context Detection
WHEN beads operations require git commands
THE SYSTEM SHALL determine the correct repository root based on:
1. BEADS_DIR environment variable (if set)
2. Redirect file in local .beads directory
3. Git worktree detection
4. Current working directory fallback

### FR-002: Redirect-Aware Resolution
WHEN .beads/redirect file exists and points to valid directory
THE SYSTEM SHALL use the parent of the redirect target as RepoRoot

### FR-003: Worktree-Aware Resolution
WHEN current directory is inside a git worktree
THE SYSTEM SHALL use the main repository root (not worktree root) for beads operations

### FR-004: Git Command Execution
WHEN executing git commands that affect beads files
THE SYSTEM SHALL run commands in RepoRoot context using cmd.Dir pattern

### FR-005: CWD Git Commands
WHEN executing git commands that query user's working state (e.g., current branch)
THE SYSTEM SHALL run commands in CWDRepoRoot context

### FR-006: Path Relativization
WHEN git commands require file paths
THE SYSTEM SHALL convert absolute paths to paths relative to RepoRoot

### FR-007: Context Caching
WHEN GetRepoContext is called multiple times
THE SYSTEM SHALL return cached result (CWD/BEADS_DIR don't change during execution)

### FR-008: Test Isolation
WHEN running tests
THE SYSTEM SHALL provide ResetCaches() to clear cached context

## Security Requirements

### SEC-001: Git Hook Disabling
WHEN GitCmd() executes git commands
THE SYSTEM SHALL disable git hooks by setting `GIT_HOOKS_PATH=` environment variable
SO THAT malicious repositories cannot execute arbitrary code

### SEC-002: Git Template Disabling
WHEN GitCmd() executes git commands
THE SYSTEM SHALL disable git templates by setting `GIT_TEMPLATE_DIR=` environment variable
SO THAT malicious template directories cannot inject content

### SEC-003: Path Boundary Validation
WHEN BEADS_DIR is resolved to an absolute path
THE SYSTEM SHALL validate the path is not in sensitive system directories
INCLUDING /etc, /usr, /var, /root, /System, /Library
SO THAT path traversal attacks are prevented

### SEC-004: Redirect Boundary Validation
WHEN .beads/redirect specifies a relative path
THE SYSTEM SHALL validate the resolved target does not escape the repository root
SO THAT redirect injection attacks are prevented

### SEC-005: Safe Directory Check
WHEN executing git commands in a repository
THE SYSTEM SHOULD verify git's safe.directory configuration allows the target
SO THAT CVE-2022-24765 protections are respected

## Daemon Requirements

### DMN-001: Workspace-Specific Context
WHEN daemon handles requests for different workspaces
THE SYSTEM SHALL provide GetRepoContextForWorkspace(path) function
THAT resolves context fresh for each workspace (no sync.Once caching)
SO THAT long-running daemons don't use stale context

### DMN-002: Context Validation
WHEN using cached RepoContext
THE SYSTEM SHALL provide Validate() method
THAT verifies BeadsDir and RepoRoot still exist
SO THAT stale context can be detected

### DMN-003: Daemon Unification
BEFORE Phase 1 migration begins
THE SYSTEM SHALL unify internal/daemon/discovery.go:findBeadsDirForWorkspace()
WITH the RepoContext API
SO THAT duplicate worktree detection logic is eliminated

## Non-Functional Requirements

### NFR-001: Backward Compatibility
THE SYSTEM SHALL maintain existing behavior for repos without BEADS_DIR or redirects

### NFR-002: Single Source of Truth
THE SYSTEM SHALL centralize all repo root resolution in RepoContext
AND remove duplicate helpers (getRepoRootForWorktree, syncbranch.GetRepoRoot)

### NFR-003: Error Handling
WHEN repository root cannot be determined
THE SYSTEM SHALL return descriptive error (not panic or silent fallback)

## Backward Compatibility Requirements

### BC-001: Deprecated API Wrapper
WHEN removing syncbranch.GetRepoRoot()
THE SYSTEM SHALL provide deprecated wrapper function for 1 release cycle
THAT delegates to beads.GetRepoContext().RepoRoot
SO THAT third-party extensions have migration time

### BC-002: Error Message Stability
WHEN error messages change format
THE SYSTEM SHALL document changed messages in release notes
SO THAT scripts parsing error output can be updated

## Test Scenarios

### Core Scenarios (P0 - Must pass before Phase 1)

| ID | Scenario | CWD | BEADS_DIR | Expected RepoRoot |
|----|----------|-----|-----------|-------------------|
| TS-001 | Normal | /repo | (unset) | /repo |
| TS-002 | Worktree | /repo/.worktrees/feat | (unset) | /repo |
| TS-003 | Redirect | /repoA | /repoB/.beads | /repoB |
| TS-004 | Combined | /repoA/.worktrees/x | /repoB/.beads | /repoB |
| TS-005 | Subdirectory | /repo/src/deep/path | (unset) | /repo |
| TS-006 | Non-git with BEADS_DIR | /tmp/notgit | /repo/.beads | /repo |

### Boundary Conditions (P0)

| ID | Scenario | Input | Expected |
|----|----------|-------|----------|
| TS-BC-001 | Empty BEADS_DIR | BEADS_DIR="" | Falls back to CWD resolution |
| TS-BC-002 | Non-existent BEADS_DIR | BEADS_DIR="/nonexistent" | Descriptive error |
| TS-BC-003 | BEADS_DIR is file | BEADS_DIR="/path/to/file" | Descriptive error |
| TS-BC-004 | Trailing slashes | BEADS_DIR="/repo/.beads///" | Normalized, works |

### Git Repository States (P0)

| ID | Scenario | Expected |
|----|----------|----------|
| TS-GIT-001 | Bare repository | RepoRoot returns bare repo path |
| TS-GIT-002 | Detached HEAD | Normal operation |
| TS-GIT-003 | Mid-rebase state | Normal operation |
| TS-GIT-004 | Shallow clone | Normal operation |

### Symlink Edge Cases (P1)

| ID | Scenario | Expected |
|----|----------|----------|
| TS-SYM-001 | BEADS_DIR is symlink | Resolves symlink, works |
| TS-SYM-002 | CWD inside symlinked dir | Canonical path returned |
| TS-SYM-003 | .beads/ is symlink | Follows symlink |
| TS-SYM-004 | Circular symlink | Error (not infinite loop) |

### Redirect Edge Cases (P1)

| ID | Scenario | Expected |
|----|----------|----------|
| TS-RED-001 | Redirect with Windows line endings | Works correctly |
| TS-RED-002 | Redirect with UTF-8 BOM | Works correctly |
| TS-RED-003 | Redirect path with spaces | Works correctly |
| TS-RED-004 | Redirect escapes repo boundary | Error (security) |

### Security Scenarios (P0)

| ID | Scenario | Expected |
|----|----------|----------|
| TS-SEC-001 | BEADS_DIR=/etc/passwd/.beads | Error: unsafe boundary |
| TS-SEC-002 | Malicious .git/hooks/ in target | Hooks NOT executed |
| TS-SEC-003 | Redirect to ../../../etc | Error: escapes boundary |

### Concurrent Access (P1)

| ID | Scenario | Expected |
|----|----------|----------|
| TS-CON-001 | Multiple goroutines GetRepoContext() | Same cached result |
| TS-CON-002 | Daemon + CLI simultaneously | Independent contexts |

### Test Isolation (P0)

| ID | Scenario | Expected |
|----|----------|----------|
| TS-ISO-001 | ResetCaches() alone | Insufficient (git cache stale) |
| TS-ISO-002 | ResetCaches() + git.ResetCaches() | Full reset works |
