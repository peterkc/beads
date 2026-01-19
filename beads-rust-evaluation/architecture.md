# br Architecture Analysis

## Codebase Structure

```
beads_rust/
├── src/
│   ├── cli/
│   │   ├── commands/     # 37 command modules
│   │   └── mod.rs        # CLI definition (clap derive)
│   ├── config/           # Configuration management
│   ├── error/
│   │   ├── mod.rs        # BeadsError enum
│   │   └── structured.rs # Agent-friendly errors (1050 lines)
│   ├── format/           # Output formatting
│   ├── model/            # Issue, Dependency types
│   ├── storage/
│   │   ├── sqlite.rs     # Primary storage (4734 lines)
│   │   └── events.rs     # Event sourcing
│   ├── sync/             # JSONL sync (4623 lines)
│   ├── util/             # ID generation, hashing
│   └── validation/       # Input validation
├── tests/                # Integration tests
└── benches/              # Criterion benchmarks
```

## Line Counts

| Module | Lines | Purpose |
|--------|-------|---------|
| storage/sqlite.rs | 4,734 | Core database operations |
| sync/mod.rs | 4,623 | JSONL import/export, 3-way merge |
| config/mod.rs | 2,043 | Configuration layering |
| model/mod.rs | 1,465 | Data types |
| cli/mod.rs | 1,445 | Command definitions |
| error/structured.rs | 1,050 | Agent-friendly errors |
| **Total src/** | ~40,000 | |

## Quality Indicators

### Rust Best Practices

- **Edition**: 2024 (latest)
- **MSRV**: 1.85 (very recent)
- **Clippy**: Pedantic + Nursery (0 warnings)
- **unsafe_code**: Forbidden via lint

### Dependencies (minimal)

```toml
clap = "4.5"          # CLI framework
rusqlite = "0.38"     # Database (bundled SQLite)
serde = "1.0"         # Serialization
chrono = "0.4"        # Time handling
anyhow = "1.0"        # Error handling
tracing = "0.1"       # Logging
```

### Testing Strategy

| Type | Framework | Coverage |
|------|-----------|----------|
| Unit | Built-in | Inline tests |
| Integration | assert_cmd | CLI behavior |
| Snapshot | insta | Output regression |
| Property | proptest | Fuzzing |
| Benchmark | criterion | Performance |

## Design Patterns

### Atomic Writes

All file operations use temp-file-then-rename:

```rust
// From sync/mod.rs
let temp_path = path.with_extension("jsonl.tmp");
write_to(&temp_path)?;
std::fs::rename(&temp_path, &path)?;
```

### Error Handling

Structured errors with agent-friendly metadata:

```rust
pub struct StructuredError {
    pub code: ErrorCode,
    pub message: String,
    pub hint: Option<String>,
    pub retryable: bool,
    pub context: HashMap<String, Value>,
}
```

### Configuration Layering

1. Built-in defaults
2. Global config (`~/.config/beads/config.yaml`)
3. Project config (`.beads/config.yaml`)
4. CLI overrides

## Comparison with bd (Go)

| Aspect | br | bd |
|--------|----|----|
| Storage | SQLite only | SQLite + Dolt |
| Sync | CLI-driven | Daemon-assisted |
| Multi-repo | Not supported | Hydration pattern |
| Templates | Not supported | Molecules/Wisps |
| Error style | Structured codes | Traditional |
