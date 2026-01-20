# Justfile for beads-next (bdx development)
# Run `just --list` for available recipes

# Default recipe
default: build

# === Build ===

# Build bd (v0 CLI)
build:
    @echo "Building bd..."
    go build -ldflags="-X main.Build=$(git rev-parse --short HEAD)" -o bd ./cmd/bd

# Build bdx (v1 CLI) - available after Stage 2
build-bdx:
    @echo "Building bdx..."
    go build -ldflags="-X main.Build=$(git rev-parse --short HEAD)" -o bdx ./cmd/bdx

# Build both CLIs
build-all: build build-bdx

# === Test ===

# Run all tests (skips known broken tests)
test:
    @echo "Running tests..."
    TEST_COVER=1 ./scripts/test.sh

# Run v1 tests only (internal/next/)
test-v1:
    @echo "Running v1 tests..."
    go test -v ./internal/next/...

# Run characterization tests (behavior validation)
test-char:
    @echo "Running characterization tests..."
    go test -v -tags=characterization ./characterization/...

# Run all tests with race detection
test-race:
    @echo "Running tests with race detection..."
    go test -race ./...

# === Benchmarks ===

# Run performance benchmarks (generates CPU profiles)
bench:
    @echo "Running performance benchmarks..."
    go test -bench=. -benchtime=1s -tags=bench -run=^$$ ./internal/storage/sqlite/ -timeout=30m
    @echo "Profile files saved in internal/storage/sqlite/"
    @echo "View: go tool pprof -http=:8080 internal/storage/sqlite/bench-cpu-*.prof"

# Run quick benchmarks (faster feedback)
bench-quick:
    @echo "Running quick benchmarks..."
    go test -bench=. -benchtime=100ms -tags=bench -run=^$$ ./internal/storage/sqlite/ -timeout=15m

# === Install ===

# Install bd to GOPATH/bin
install:
    @echo "Installing bd to $(go env GOPATH)/bin..."
    go install -ldflags="-X main.Commit=$(git rev-parse HEAD) -X main.Branch=$(git rev-parse --abbrev-ref HEAD)" ./cmd/bd

# Install bdx to GOPATH/bin
install-bdx:
    @echo "Installing bdx to $(go env GOPATH)/bin..."
    go install -ldflags="-X main.Commit=$(git rev-parse HEAD) -X main.Branch=$(git rev-parse --abbrev-ref HEAD)" ./cmd/bdx

# === Code Quality ===

# Run linter
lint:
    @echo "Running golangci-lint..."
    golangci-lint run

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...
    gofumpt -w .

# Generate mocks (for v1 ports)
generate:
    @echo "Generating mocks..."
    go generate ./internal/next/ports/...

# === Validation Gates ===

# Stage 1 checkpoint: Foundation
check-stage1: test-char test-v1 generate
    @echo "✅ Stage 1 checkpoint passed"

# Stage 2 checkpoint: Pluginize (bd and bdx produce same output)
check-stage2: test-char
    @echo "Comparing bd and bdx output..."
    ./scripts/compare-bd-bdx.sh
    @echo "✅ Stage 2 checkpoint passed"

# Stage 3 checkpoint: Modernize
check-stage3: test
    @echo "✅ Stage 3 checkpoint passed - all tests pass"

# === Clean ===

# Remove build artifacts
clean:
    @echo "Cleaning..."
    rm -f bd bdx
    rm -f internal/storage/sqlite/bench-cpu-*.prof
    rm -f beads-perf-*.prof

# === Development ===

# Watch and rebuild on changes (requires entr)
watch:
    @echo "Watching for changes..."
    find . -name '*.go' | entr -c just build

# Show stub count (unimplemented v1 code)
stubs:
    @echo "Unimplemented stubs in v1:"
    @grep -r "ErrNotImplemented" internal/next --include="*.go" -l 2>/dev/null | while read f; do \
        count=$(grep -c "ErrNotImplemented" "$f"); \
        echo "  ❌ $f ($count stubs)"; \
    done || echo "  ✅ No stubs found"

# === Semantic Search ===

# Build ck index
ck-index:
    @echo "Building semantic search index..."
    ck --index .

# Search codebase
ck-search query:
    ck --sem "{{query}}" .
