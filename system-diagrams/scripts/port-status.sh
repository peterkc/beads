#!/usr/bin/env bash
# port-status.sh - Dashboard showing upstream port status
#
# Usage:
#   ./port-status.sh              # Show status
#   ./port-status.sh --json       # JSON output
#   ./port-status.sh --pending    # Only show pending
#   ./port-status.sh --by-plugin  # Group by plugin

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

# Parse args
JSON_OUTPUT=false
PENDING_ONLY=false
BY_PLUGIN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_OUTPUT=true ;;
        --pending) PENDING_ONLY=true ;;
        --by-plugin) BY_PLUGIN=true ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
    shift
done

# Get port issues from beads
get_port_issues() {
    bd list --label=port --format=json 2>/dev/null || echo "[]"
}

# Get upstream commits not yet tracked
get_untracked() {
    git fetch upstream --quiet 2>/dev/null || true

    # Commits in upstream not in our main
    local upstream_commits=$(git rev-list origin/main..upstream/main 2>/dev/null || echo "")

    if [[ -z "$upstream_commits" ]]; then
        echo "[]"
        return
    fi

    # Check which have port issues
    local issues=$(get_port_issues)
    local untracked="["
    local first=true

    for sha in $upstream_commits; do
        local short_sha=$(echo "$sha" | cut -c1-7)
        local has_issue=$(echo "$issues" | jq -r --arg sha "$short_sha" '[.[] | select(.title | contains($sha))] | length')

        if [[ "$has_issue" == "0" ]]; then
            local subject=$(git log -1 --format="%s" "$sha")
            if [[ "$first" == "true" ]]; then
                first=false
            else
                untracked+=","
            fi
            untracked+="{\"sha\":\"$short_sha\",\"subject\":\"$subject\"}"
        fi
    done

    untracked+="]"
    echo "$untracked"
}

# Main dashboard
main() {
    local issues=$(get_port_issues)
    local untracked=$(get_untracked)

    # Count by status
    local total=$(echo "$issues" | jq 'length')
    local open=$(echo "$issues" | jq '[.[] | select(.status == "open")] | length')
    local in_progress=$(echo "$issues" | jq '[.[] | select(.status == "in_progress")] | length')
    local closed=$(echo "$issues" | jq '[.[] | select(.status == "closed")] | length')
    local untracked_count=$(echo "$untracked" | jq 'length')

    if [[ "$JSON_OUTPUT" == "true" ]]; then
        cat <<EOF
{
  "summary": {
    "total_tracked": $total,
    "open": $open,
    "in_progress": $in_progress,
    "closed": $closed,
    "untracked": $untracked_count
  },
  "issues": $issues,
  "untracked_commits": $untracked
}
EOF
        return
    fi

    # Pretty output
    echo ""
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}           Upstream Port Status Dashboard              ${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════${NC}"
    echo ""

    # Summary
    echo -e "${BOLD}Summary${NC}"
    echo -e "─────────────────────────────────"
    printf "  %-20s %s\n" "Total tracked:" "$total"
    printf "  %-20s ${RED}%s${NC}\n" "Open:" "$open"
    printf "  %-20s ${YELLOW}%s${NC}\n" "In Progress:" "$in_progress"
    printf "  %-20s ${GREEN}%s${NC}\n" "Closed:" "$closed"
    printf "  %-20s ${RED}%s${NC}\n" "Untracked:" "$untracked_count"
    echo ""

    # Progress bar
    if [[ "$total" -gt 0 ]]; then
        local pct=$((closed * 100 / total))
        local filled=$((pct / 5))
        local empty=$((20 - filled))

        echo -e "${BOLD}Progress${NC}"
        echo -e "─────────────────────────────────"
        printf "  ["
        printf "${GREEN}%0.s█${NC}" $(seq 1 $filled) 2>/dev/null || true
        printf "%0.s░" $(seq 1 $empty) 2>/dev/null || true
        printf "] %d%%\n" "$pct"
        echo ""
    fi

    # Untracked commits (if any)
    if [[ "$untracked_count" -gt 0 ]]; then
        echo -e "${BOLD}${RED}Untracked Upstream Commits${NC}"
        echo -e "─────────────────────────────────"
        echo "$untracked" | jq -r '.[] | "  \(.sha)  \(.subject)"'
        echo ""
        echo -e "  ${YELLOW}Run: ./analyze-upstream.sh | ./create-port-issues.sh${NC}"
        echo ""
    fi

    # Open issues (if not pending only)
    if [[ "$PENDING_ONLY" == "false" && "$open" -gt 0 ]]; then
        echo -e "${BOLD}Open Port Issues${NC}"
        echo -e "─────────────────────────────────"
        echo "$issues" | jq -r '.[] | select(.status == "open") | "  \(.id)  \(.title)"'
        echo ""
    fi

    # In progress
    if [[ "$in_progress" -gt 0 ]]; then
        echo -e "${BOLD}${YELLOW}In Progress${NC}"
        echo -e "─────────────────────────────────"
        echo "$issues" | jq -r '.[] | select(.status == "in_progress") | "  \(.id)  \(.title)"'
        echo ""
    fi

    # By plugin (if requested)
    if [[ "$BY_PLUGIN" == "true" ]]; then
        echo -e "${BOLD}By Plugin${NC}"
        echo -e "─────────────────────────────────"
        for plugin in core work sync linear molecules compact; do
            local count=$(echo "$issues" | jq --arg p "$plugin" '[.[] | select(.title | contains($p)) | select(.status != "closed")] | length')
            if [[ "$count" -gt 0 ]]; then
                printf "  %-12s %s pending\n" "$plugin:" "$count"
            fi
        done
        echo ""
    fi

    echo -e "${CYAN}═══════════════════════════════════════════════════════${NC}"
}

main
