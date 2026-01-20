# Go Features for bdx (v1 Rewrite)

## Version Context

| Branch | Go Version | Status |
|--------|------------|--------|
| v0 (main) | 1.24.0 | Current |
| v1 (next) | **1.25.0** | Target |

**Strategy:** bdx should use Go 1.25 and leverage ALL new features from 1.24+. Many 1.24 features may be underutilized in v0.

---

## Go 1.24 Features (Available Now, May Be Underutilized)

### Language: Generic Type Aliases

```go
// Before (workaround)
type IssueMap map[string]Issue

// After (1.24+) - parameterized aliases
type Map[K comparable, V any] = map[K]V
type IssueMap = Map[string, Issue]
```

**Use in bdx:** Clean type aliases for plugin interfaces.

---

### New Package: `weak` (Weak Pointers)

```go
import "weak"

// Memory-efficient caching without preventing GC
type IssueCache struct {
    cache map[string]weak.Pointer[Issue]
}

func (c *IssueCache) Get(id string) (*Issue, bool) {
    if wp, ok := c.cache[id]; ok {
        if issue := wp.Value(); issue != nil {
            return issue, true  // Still in memory
        }
        delete(c.cache, id)  // GC'd, clean up
    }
    return nil, false
}
```

**Use in bdx:** Issue cache that doesn't prevent GC of unused issues.

---

### New Package: `os.Root` (Sandboxed Filesystem)

```go
import "os"

// Restrict file access to .beads/ directory
root, err := os.OpenRoot(".beads")
if err != nil {
    return err
}
defer root.Close()

// Can only access files under .beads/
f, err := root.Open("issues.jsonl")      // OK
f, err := root.Open("../secrets.env")    // Error: escapes root
```

**Use in bdx:** Secure file access - plugins can't escape their sandbox.

---

### Testing: `B.Loop()` for Benchmarks

```go
// Before (error-prone)
func BenchmarkCreate(b *testing.B) {
    for i := 0; i < b.N; i++ {
        createIssue()  // Easy to mess up loop
    }
}

// After (1.24+)
func BenchmarkCreate(b *testing.B) {
    for b.Loop() {
        createIssue()  // Cleaner, handles edge cases
    }
}
```

**Use in bdx:** All benchmarks should use `b.Loop()`.

---

### Testing: `T.Context()` and `T.Chdir()`

```go
func TestSync(t *testing.T) {
    ctx := t.Context()  // Auto-canceled when test ends

    t.Chdir("/tmp/test-beads")  // Auto-restored after test

    // Test with clean working directory
    err := sync.Run(ctx)
}
```

**Use in bdx:** Clean test isolation for file-based operations.

---

### Encoding: `omitzero` Tag

```go
type Issue struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description,omitzero"`  // Omit if ""
    Priority    int       `json:"priority,omitzero"`     // Omit if 0
    Labels      []string  `json:"labels,omitzero"`       // Omit if nil/empty
    CreatedAt   time.Time `json:"created_at,omitzero"`   // Omit if zero time
}
```

**Use in bdx:** Cleaner JSON output without empty fields.

---

### Strings/Bytes: Iterator Functions

```go
import "strings"

// Parse JSONL efficiently with iterators
for line := range strings.Lines(content) {
    var issue Issue
    json.Unmarshal([]byte(line), &issue)
}

// Split with iterators (lazy evaluation)
for field := range strings.FieldsSeq(line) {
    // Process each field
}
```

**Use in bdx:** Efficient JSONL parsing, output formatting.

---

### Crypto: `cipher.NewGCMWithRandomNonce()`

```go
import "crypto/cipher"

// Auto-generate and prepend nonce
aead, _ := cipher.NewGCMWithRandomNonce(block)
ciphertext := aead.Seal(nil, nil, plaintext, nil)  // Nonce auto-prepended
```

**Use in bdx:** If implementing encrypted storage or sync.

---

### Performance: Swiss Tables Map

Go 1.24 uses Swiss Tables for `map` by default:
- **2-3% CPU reduction** across the board
- Better cache locality
- Automatic (no code changes needed)

**Use in bdx:** Free performance win for issue maps, caches.

---

### Tooling: `tool` Directive in go.mod

```go
// go.mod
module github.com/steveyegge/beads

go 1.25

tool (
    golang.org/x/tools/cmd/stringer
    github.com/golangci/golangci-lint/cmd/golangci-lint
)
```

**Use in bdx:** Replace `tools.go` pattern with native directive.

---

### Tooling: `go build -json`

```bash
go build -json ./cmd/bdx 2>&1 | jq '.Action'
```

**Use in bdx:** CI/CD integration, build automation scripts.

---

### Runtime: `AddCleanup()` (Better Finalizers)

```go
import "runtime"

type DBConnection struct {
    conn *sql.DB
}

func NewDB(path string) *DBConnection {
    db := &DBConnection{conn: openDB(path)}

    runtime.AddCleanup(db, func(db *DBConnection) {
        db.conn.Close()  // Guaranteed cleanup
    })

    return db
}
```

**Use in bdx:** Resource cleanup for database connections, file handles.

---

## Go 1.25 Features (New for bdx)

### Concurrency: `sync.WaitGroup.Go()`

```go
var wg sync.WaitGroup

// Before (verbose)
wg.Add(1)
go func() {
    defer wg.Done()
    loadIssues()
}()

// After (1.25)
wg.Go(func() {
    loadIssues()
})
```

**Use in bdx:** Plugin initialization, parallel operations.

---

### Testing: `testing/synctest` (Virtualized Time)

```go
import "testing/synctest"

func TestEventBus(t *testing.T) {
    synctest.Test(t, func(ctx context.Context) {
        bus := events.NewBus()
        received := make(chan bool, 1)

        bus.Subscribe("issue.created", func(e Event) {
            received <- true
        })

        bus.Publish("issue.created", Issue{})

        synctest.Wait()  // All goroutines blocked

        select {
        case <-received:
            // OK
        default:
            t.Error("event not received")
        }
    })
}
```

**Use in bdx:** Deterministic tests for event bus, async operations.

---

### Testing: `T.Attr()` for Structured Output

```go
func TestCreate(t *testing.T) {
    t.Attr("issue_type", "bug")
    t.Attr("priority", "P1")

    // Output:
    // === ATTR  TestCreate issue_type bug
    // === ATTR  TestCreate priority P1
}
```

**Use in bdx:** CI/CD test analytics, structured test metadata.

---

### JSON v2 (Experimental)

```bash
GOEXPERIMENT=jsonv2 go build -o bdx ./cmd/bdx
```

```go
import "encoding/json/v2"

// Streaming without intermediate buffers
jsonv2.MarshalWrite(w, issues)
jsonv2.UnmarshalRead(r, &issues)

// Better error messages
// "json: cannot unmarshal string into Go struct field Issue.Priority of type int at line 42, column 15"
```

**Use in bdx:** Faster JSONL parsing, better error messages.

---

### Runtime: Container-Aware GOMAXPROCS

```go
// Automatic on Linux with cgroups
// K8s pod with cpu.limit=2 → GOMAXPROCS=2 (not node CPU count)

// Re-enable if manually overridden
runtime.SetDefaultGOMAXPROCS()
```

**Use in bdx:** Automatic tuning in containerized CI/CD.

---

### Runtime: Flight Recorder

```go
import "runtime/trace"

var recorder *trace.FlightRecorder

func init() {
    recorder = trace.NewFlightRecorder(&trace.FlightRecorderConfig{
        Size: 10 << 20,  // 10MB ring buffer
    })
    recorder.Start()
}

// On hang or error:
func dumpTrace(filename string) error {
    f, _ := os.Create(filename)
    defer f.Close()
    return recorder.WriteTo(f)  // Snapshot last N seconds
}
```

**Use in bdx:** Debug production hangs in `bd sync` or daemon.

---

### Garbage Collector: Green Tea GC (Experimental)

```bash
GOEXPERIMENT=greenteagc go build -o bdx ./cmd/bdx
```

- **10-40% GC overhead reduction**
- Better for small objects (issues, events)

**Use in bdx:** Benchmark with large `.beads/` databases.

---

### go.mod: `ignore` Directive

```go
// go.mod
module github.com/steveyegge/beads

go 1.25

ignore (
    research/
    specs/
)
```

**Use in bdx:** Exclude nested repos from package resolution.

---

## Feature Adoption Matrix

| Feature | Category | Effort | Impact | Priority |
|---------|----------|--------|--------|----------|
| `WaitGroup.Go()` | Concurrency | Low | Medium | **P0** |
| `testing/synctest` | Testing | Medium | High | **P0** |
| `omitzero` tag | Encoding | Low | Medium | **P0** |
| `T.Context()`/`T.Chdir()` | Testing | Low | Medium | **P1** |
| `B.Loop()` | Testing | Low | Low | **P1** |
| `strings.Lines()` | Performance | Low | Medium | **P1** |
| `os.Root` | Security | Medium | High | **P1** |
| `weak` pointers | Memory | Medium | Medium | **P2** |
| `tool` directive | Tooling | Low | Low | **P2** |
| JSON v2 | Performance | Medium | High | **P2** |
| Flight Recorder | Debugging | Medium | Medium | **P2** |
| Green Tea GC | Performance | Low | High | **P3** |

---

## Recommended go.mod for bdx

```go
module github.com/steveyegge/beads

go 1.25

toolchain go1.25.0

ignore (
    research/
    specs/
)

tool (
    golang.org/x/tools/cmd/stringer
    github.com/golangci/golangci-lint/cmd/golangci-lint
)

require (
    // dependencies...
)
```

---

## Breaking Changes to Watch

### 1. Nil Pointer Check Fix (1.25)

```go
// ❌ PANICS in 1.25 (was silent bug)
f, err := os.Open("file")
name := f.Name()  // Deref before check!
if err != nil {
    return err
}

// ✅ CORRECT
f, err := os.Open("file")
if err != nil {
    return err
}
name := f.Name()
```

**Action:** Audit all error handling in v0 before porting.

### 2. TLS SHA-1 Disabled (1.25)

SHA-1 signatures disallowed in TLS 1.2 by default.

```go
// Re-enable if needed (not recommended)
os.Setenv("GODEBUG", "tlssha1=1")
```

**Action:** Check Linear integration TLS connections.

### 3. `math/rand.Seed()` No-Op (1.24)

```go
rand.Seed(time.Now().UnixNano())  // Does nothing in 1.24+
```

**Action:** Remove any `rand.Seed()` calls.

---

## References

- [Go 1.24 Release Notes](https://go.dev/doc/go1.24)
- [Go 1.25 Release Notes](https://go.dev/doc/go1.25)
- [Go 1.25 Interactive Tour](https://antonz.org/go-1-25/)
