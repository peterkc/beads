# Feature Matrix: br vs bd

## Core Issue Tracking

| Feature | br | bd | Notes |
|---------|----|----|-------|
| Create/Read/Update/Delete | ✅ | ✅ | Parity |
| Status tracking | ✅ | ✅ | open, in_progress, closed |
| Priority levels | ✅ | ✅ | P0-P4 |
| Type classification | ✅ | ✅ | task, bug, feature, epic |
| Labels/tags | ✅ | ✅ | Multi-label support |
| Comments | ✅ | ✅ | Thread support |
| Search | ✅ | ✅ | Full-text |

## Dependencies & Graph

| Feature | br | bd | Notes |
|---------|----|----|-------|
| Dependency tracking | ✅ | ✅ | blocks/blocked-by |
| Cycle detection | ✅ | ✅ | Prevents circular deps |
| Graph visualization | ✅ | ✅ | `br graph`, `bd graph` |
| Ready queue | ✅ | ✅ | Unblocked work |
| Blocked list | ✅ | ✅ | Blocked work |

## Sync & Storage

| Feature | br | bd | Notes |
|---------|----|----|-------|
| SQLite storage | ✅ | ✅ | Primary storage |
| JSONL export | ✅ | ✅ | Git-friendly |
| 3-way merge | ✅ | ✅ | br has explicit safety guards |
| Dolt backend | ❌ | ✅ | Future-proofing in bd |
| Background daemon | ❌ | ✅ | br is CLI-only |
| Auto-sync | ❌ | ✅ | br requires explicit sync |

## Advanced Features

| Feature | br | bd | ACF Impact |
|---------|----|----|------------|
| **Multi-repo aggregation** | ❌ | ✅ | **Critical** - ACF depends on this |
| **Molecules (templates)** | ❌ | ✅ | **Critical** - spec phases use this |
| **bd prime (agent context)** | ❌ | ✅ | **Critical** - hooks depend on this |
| **Wisps (ephemeral issues)** | ❌ | ✅ | Important for scratch work |
| Hierarchical IDs | ❌ | ✅ | `bd-a3f8.1.1` pattern |
| Saved queries | ✅ | ✅ | Parity |
| Defer/undefer | ✅ | ✅ | Parity |
| Audit trail | ✅ | ✅ | Parity |
| Changelog generation | ✅ | ✅ | Parity |
| Orphan detection | ✅ | ✅ | Parity |

## Agent Integration

| Feature | br | bd | Notes |
|---------|----|----|-------|
| JSON output | ✅ | ✅ | `--json` flag |
| Structured error codes | ✅ | Partial | br has full ErrorCode enum |
| Levenshtein suggestions | ✅ | ❌ | br suggests similar IDs |
| Retryability flags | ✅ | ❌ | br indicates if retry possible |
| AGENTS.md management | ✅ | ❓ | `br agents` command |

## Platform & Deployment

| Feature | br | bd | Notes |
|---------|----|----|-------|
| Binary size | 5.2 MB | ~30 MB | br 6x smaller |
| Self-update | ✅ | ✅ | `br upgrade`, `bd upgrade` |
| Shell completions | ✅ | ✅ | Bash, Zsh, Fish |
| Cross-platform | ✅ | ✅ | Linux, macOS, Windows |
