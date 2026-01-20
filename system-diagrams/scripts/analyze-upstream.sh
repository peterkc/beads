#!/usr/bin/env bash
# analyze-upstream.sh - Analyze upstream commits for plugin impact
#
# Usage:
#   ./analyze-upstream.sh                    # Analyze new commits since last sync
#   ./analyze-upstream.sh abc123             # Analyze specific commit
#   ./analyze-upstream.sh abc123..def456     # Analyze commit range
#
# Output: JSON with affected plugins and porting recommendations

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m'

log() { echo -e "${BLUE}[analyze]${NC} $1" >&2; }
warn() { echo -e "${YELLOW}[analyze]${NC} $1" >&2; }

# File → Plugin mapping
get_plugin() {
    local file="$1"
    case "$file" in
        # Core plugin
        internal/storage/sqlite/queries.go|internal/storage/sqlite/issues.go)
            echo "core" ;;
        cmd/bd/create.go|cmd/bd/list.go|cmd/bd/show.go|cmd/bd/update.go|cmd/bd/close.go)
            echo "core" ;;

        # Work plugin
        internal/storage/sqlite/ready.go|internal/storage/sqlite/dependencies.go)
            echo "work" ;;
        cmd/bd/ready.go|cmd/bd/dep.go|cmd/bd/blocked.go)
            echo "work" ;;

        # Sync plugin
        internal/storage/sqlite/dirty.go|internal/storage/sqlite/sync*.go)
            echo "sync" ;;
        internal/export/*|internal/importer/*|internal/syncbranch/*)
            echo "sync" ;;
        cmd/bd/sync.go|cmd/bd/export.go|cmd/bd/import.go)
            echo "sync" ;;

        # Linear plugin
        internal/linear/*)
            echo "linear" ;;
        cmd/bd/linear.go)
            echo "linear" ;;

        # Molecules plugin
        internal/molecules/*)
            echo "molecules" ;;

        # Compact plugin
        internal/compact/*)
            echo "compact" ;;

        # Shared (affects multiple plugins)
        internal/storage/storage.go|internal/storage/sqlite/storage.go)
            echo "shared:core,work,sync" ;;
        internal/types/*)
            echo "shared:all" ;;
        internal/config/*)
            echo "config" ;;
        internal/rpc/*)
            echo "rpc" ;;

        # Infrastructure (usually safe to ignore for plugin porting)
        go.mod|go.sum|Makefile|README.md|*.md)
            echo "infra" ;;
        .github/*|scripts/*)
            echo "ci" ;;

        *)
            echo "unknown" ;;
    esac
}

# Analyze a single commit
analyze_commit() {
    local sha="$1"
    local subject=$(git log -1 --format="%s" "$sha")
    local author=$(git log -1 --format="%an" "$sha")
    local date=$(git log -1 --format="%ci" "$sha")

    local plugins=()
    local files=()
    local unknown_files=()

    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        files+=("$file")

        local plugin=$(get_plugin "$file")

        if [[ "$plugin" == "unknown" ]]; then
            unknown_files+=("$file")
        elif [[ "$plugin" == shared:* ]]; then
            # Expand shared to multiple plugins
            local affected="${plugin#shared:}"
            if [[ "$affected" == "all" ]]; then
                plugins+=("core" "work" "sync" "linear" "molecules" "compact")
            else
                IFS=',' read -ra parts <<< "$affected"
                plugins+=("${parts[@]}")
            fi
        elif [[ "$plugin" != "infra" && "$plugin" != "ci" ]]; then
            plugins+=("$plugin")
        fi
    done < <(git show --name-only --format="" "$sha")

    # Deduplicate plugins
    local unique_plugins=($(printf '%s\n' "${plugins[@]}" | sort -u))

    # Determine priority
    local priority="low"
    if [[ " ${unique_plugins[*]} " =~ " core " ]]; then
        priority="high"
    elif [[ " ${unique_plugins[*]} " =~ " work " ]] || [[ " ${unique_plugins[*]} " =~ " sync " ]]; then
        priority="medium"
    fi

    # Output JSON
    cat <<EOF
{
  "sha": "$sha",
  "subject": "$subject",
  "author": "$author",
  "date": "$date",
  "plugins": [$(printf '"%s",' "${unique_plugins[@]}" | sed 's/,$//')]
  "files_changed": ${#files[@]},
  "unknown_files": [$(printf '"%s",' "${unknown_files[@]}" | sed 's/,$//')]
  "priority": "$priority"
}
EOF
}

# Main
main() {
    local range="${1:-}"

    # If no range, find commits since last sync
    if [[ -z "$range" ]]; then
        # Check if we have upstream remote
        if ! git remote | grep -q upstream; then
            warn "No upstream remote. Adding..."
            git remote add upstream https://github.com/steveyegge/beads.git
        fi

        git fetch upstream --quiet

        # Find commits on upstream/main not in origin/main
        range="origin/main..upstream/main"
        log "Analyzing commits: $range"
    fi

    # Check if it's a single commit or range
    if [[ "$range" == *..* ]]; then
        # Range
        local commits=$(git rev-list "$range" 2>/dev/null || echo "")
    else
        # Single commit
        local commits="$range"
    fi

    if [[ -z "$commits" ]]; then
        log "No new commits to analyze"
        echo "[]"
        exit 0
    fi

    # Analyze each commit
    echo "["
    local first=true
    for sha in $commits; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        analyze_commit "$sha"
    done
    echo "]"
}

main "$@"
