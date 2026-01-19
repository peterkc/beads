# br Error Patterns Worth Porting

The structured error handling in br (`src/error/structured.rs`) is designed for AI coding agents. These patterns could improve bd's agent ergonomics.

## ErrorCode Enum

br uses explicit error codes for programmatic handling:

```rust
pub enum ErrorCode {
    // Database Errors (exit code 2)
    DatabaseNotFound,
    DatabaseLocked,
    SchemaMismatch,
    NotInitialized,
    AlreadyInitialized,

    // Issue Errors (exit code 3)
    IssueNotFound,
    AmbiguousId,
    IdCollision,
    InvalidId,

    // Validation Errors (exit code 4)
    ValidationFailed,
    InvalidStatus,
    InvalidPriority,
    InvalidType,

    // Dependency Errors (exit code 5)
    DependencyCycle,
    SelfDependency,
    DependencyNotFound,

    // Sync Errors (exit code 6)
    SyncConflict,
    JsonlParseError,
    // ...
}
```

**Benefit**: Agents can switch on error codes instead of parsing messages.

## Levenshtein Suggestions

When an issue ID isn't found, br suggests similar IDs:

```rust
fn suggest_similar_ids(input: &str, all_ids: &[String]) -> Vec<String> {
    all_ids
        .iter()
        .filter(|id| levenshtein(input, id) <= 2)
        .take(3)
        .cloned()
        .collect()
}
```

**Example Output**:
```json
{
  "code": "IssueNotFound",
  "message": "Issue 'bd-abc1' not found",
  "hint": "Did you mean: bd-abc2, bd-abc3?",
  "similar_ids": ["bd-abc2", "bd-abc3", "bd-abcd"]
}
```

**Benefit**: Agents can auto-retry with suggestions.

## Retryability Flags

Each error indicates whether retry might succeed:

```rust
pub struct StructuredError {
    pub retryable: bool,
    // ...
}

impl ErrorCode {
    pub fn is_retryable(&self) -> bool {
        matches!(self,
            Self::DatabaseLocked |
            Self::SyncConflict |
            Self::NetworkError
        )
    }
}
```

**Benefit**: Agents know when to retry vs fail fast.

## Intent Detection

br recognizes common agent mistakes:

```rust
// Detect "status=open" vs "--status=open" confusion
if input.contains("status=") && !input.starts_with("--") {
    hint = Some("Use '--status=open' (with dashes) for filtering");
}

// Detect priority word vs number
if input.parse::<Priority>().is_err() {
    if let Some(num) = priority_word_to_number(input) {
        hint = Some(format!("Use '{}' instead of '{}'", num, input));
    }
}
```

**Benefit**: Self-correcting error messages.

## Porting Strategy

### Option A: Minimal Port

Add structured JSON errors to bd without changing existing behavior:

```go
// bd could add --structured-errors flag
type StructuredError struct {
    Code      string            `json:"code"`
    Message   string            `json:"message"`
    Hint      string            `json:"hint,omitempty"`
    Retryable bool              `json:"retryable"`
    Context   map[string]any    `json:"context,omitempty"`
}
```

### Option B: Full Port

Adopt br's error enum pattern with Go's error wrapping:

```go
type ErrorCode int

const (
    ErrDatabaseNotFound ErrorCode = iota + 200
    ErrDatabaseLocked
    ErrNotInitialized
    // ...
)

func (e ErrorCode) IsRetryable() bool {
    return e == ErrDatabaseLocked || e == ErrSyncConflict
}
```

### Option C: MCP Integration

Expose structured errors via MCP tool responses:

```json
{
  "error": {
    "code": "ISSUE_NOT_FOUND",
    "message": "Issue 'bd-xyz' not found",
    "hint": "Did you mean: bd-xyz1?",
    "retryable": false
  }
}
```

## Recommendation

Start with **Option A** (minimal port) as a `--agent-errors` flag. This:
- Doesn't break existing behavior
- Lets agents opt-in
- Can be extended incrementally

Track as upstream issue if maintainer is receptive.
