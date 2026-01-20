#!/usr/bin/env bash
# setup-orphan-next.sh - Create orphan 'next' branch with clean history
#
# WARNING: This creates a branch with no common ancestor to main.
# Merging back to upstream will require special handling.
#
# Usage:
#   ./setup-orphan-next.sh

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m'

log() { echo -e "${BLUE}[setup]${NC} $1"; }
warn() { echo -e "${YELLOW}[setup]${NC} $1"; }
success() { echo -e "${GREEN}[setup]${NC} $1"; }
error() { echo -e "${RED}[setup]${NC} $1"; }

# Confirm
warn "This will create an orphan 'next' branch with NO common ancestor to main."
warn "Merging back to upstream will require 'git merge --allow-unrelated-histories'."
echo ""
read -p "Continue? [y/N] " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    log "Aborted"
    exit 0
fi

# Store current main HEAD for reference
MAIN_HEAD=$(git rev-parse main)
log "Current main HEAD: $MAIN_HEAD"

# Create orphan branch
log "Creating orphan branch 'next'..."
git checkout --orphan next

# Remove all files (clean slate)
git rm -rf . 2>/dev/null || true
git clean -fd

# Create initial structure with v1 architecture
log "Creating v1 directory structure..."

mkdir -p cmd/bdx
mkdir -p internal/{core,ports,adapters/sqlite,usecases,events,plugins}

# Create README
cat > README.md << 'EOF'
# Beads v1 (next branch)

This is the v1 rewrite of beads using clean architecture.

## Architecture

```
cmd/bdx/           # CLI entry point
internal/
├── core/          # Domain (pure Go)
├── ports/         # Interfaces
├── adapters/      # Implementations
├── usecases/      # Business operations
├── events/        # Event bus
└── plugins/       # Plugin system
```

## Relationship to v0

This branch is an orphan (no shared history with main/v0).
Use git notes to trace code lineage:

```bash
# Find what v0 code a v1 file is derived from
git notes show HEAD

# Find all v1 commits derived from a v0 commit
git log --notes --grep="derived-from: <v0-sha>"
```

## Building

```bash
go build -o bdx ./cmd/bdx
```
EOF

# Create go.mod
cat > go.mod << 'EOF'
module github.com/steveyegge/beads

go 1.22
EOF

# Create placeholder main
mkdir -p cmd/bdx
cat > cmd/bdx/main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("bdx v1.0.0-alpha (beads experimental)")
}
EOF

# Create ports/errors.go
cat > internal/ports/errors.go << 'EOF'
package ports

import "errors"

// ErrNotImplemented is returned by stub implementations
var ErrNotImplemented = errors.New("not implemented")
EOF

# Initial commit
git add -A
git commit -m "feat: initialize v1 architecture (orphan branch)

Clean-slate v1 rewrite with:
- Hexagonal architecture (ports/adapters)
- Plugin-based commands
- Event bus for loose coupling

This is an orphan branch - no shared history with main.
Use git notes for v0 → v1 traceability.

Derived-from: $MAIN_HEAD (main)"

# Add git note linking to main
git notes add -m "branch-origin: orphan
derived-from: $MAIN_HEAD (main)
reason: Clean v1 architecture without v0 history"

success "Created orphan 'next' branch!"
echo ""
log "Current branch: $(git branch --show-current)"
log "Initial commit: $(git rev-parse HEAD)"
echo ""
log "Next steps:"
log "  1. Push: git push -u origin next"
log "  2. Configure git notes: See ADR 0004"
log "  3. Start building v1 plugins"
echo ""
warn "Remember: Merging to upstream requires --allow-unrelated-histories"
