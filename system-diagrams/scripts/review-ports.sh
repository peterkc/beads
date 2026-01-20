#!/usr/bin/env bash
# review-ports.sh - Review and merge Claude-ported PRs locally
#
# Usage:
#   ./review-ports.sh              # Interactive review of all port PRs
#   ./review-ports.sh --list       # List pending port PRs
#   ./review-ports.sh --merge-all  # Merge all approved port PRs
#
# Prerequisites:
#   - Graphite CLI: npm install -g @withgraphite/graphite-cli
#   - Authenticated: gt auth

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

log() { echo -e "${BLUE}[review]${NC} $1"; }
success() { echo -e "${GREEN}[review]${NC} $1"; }
warn() { echo -e "${YELLOW}[review]${NC} $1"; }
error() { echo -e "${RED}[review]${NC} $1"; }

# Check prerequisites
check_prereqs() {
    if ! command -v gt &> /dev/null; then
        error "Graphite CLI not found. Install with: npm install -g @withgraphite/graphite-cli"
        exit 1
    fi

    if ! command -v gh &> /dev/null; then
        error "GitHub CLI not found. Install with: brew install gh"
        exit 1
    fi
}

# List port PRs
list_port_prs() {
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}              Pending Port PRs (Stacked)               ${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════${NC}"
    echo ""

    # Get port PRs from GitHub
    gh pr list --search "port( in:title" --json number,title,headRefName,reviews,mergeable \
        | jq -r '.[] | "  #\(.number) \(.headRefName)\n      \(.title)\n      Mergeable: \(.mergeable)\n"'

    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════${NC}"
}

# Show Graphite stack
show_stack() {
    log "Current stack:"
    echo ""
    gt log --short 2>/dev/null || echo "  (no stack)"
    echo ""
}

# Review a single PR
review_pr() {
    local pr_number="$1"

    echo -e "\n${BOLD}Reviewing PR #${pr_number}${NC}\n"

    # Show PR details
    gh pr view "$pr_number"

    echo ""
    echo -e "${BOLD}Options:${NC}"
    echo "  [a] Approve and continue"
    echo "  [e] Edit (open in editor)"
    echo "  [d] View diff"
    echo "  [c] Checkout locally"
    echo "  [s] Skip (review later)"
    echo "  [r] Request changes"
    echo "  [q] Quit review"
    echo ""

    read -p "Choice: " choice

    case "$choice" in
        a|A)
            gh pr review "$pr_number" --approve -b "LGTM - ported correctly"
            success "Approved PR #$pr_number"
            return 0
            ;;
        e|E)
            local branch=$(gh pr view "$pr_number" --json headRefName -q '.headRefName')
            git checkout "$branch"
            log "Make your changes, then: git push && ./review-ports.sh"
            return 1
            ;;
        d|D)
            gh pr diff "$pr_number"
            review_pr "$pr_number"  # Re-prompt after showing diff
            ;;
        c|C)
            gh pr checkout "$pr_number"
            log "Checked out PR #$pr_number. Return with: git checkout next"
            return 1
            ;;
        s|S)
            warn "Skipped PR #$pr_number"
            return 0
            ;;
        r|R)
            read -p "Comment: " comment
            gh pr review "$pr_number" --request-changes -b "$comment"
            warn "Requested changes on PR #$pr_number"
            return 0
            ;;
        q|Q)
            log "Quitting review"
            exit 0
            ;;
        *)
            error "Invalid choice"
            review_pr "$pr_number"
            ;;
    esac
}

# Interactive review of all port PRs
interactive_review() {
    log "Fetching port PRs..."

    local prs=$(gh pr list --search "port( in:title" --json number --jq '.[].number')

    if [[ -z "$prs" ]]; then
        log "No port PRs to review"
        return
    fi

    local count=$(echo "$prs" | wc -l | tr -d ' ')
    log "Found $count port PR(s) to review"
    echo ""

    show_stack

    for pr in $prs; do
        review_pr "$pr" || break
    done

    echo ""
    success "Review complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Check stack: gt log"
    echo "  2. Merge approved: gt merge --all"
    echo "  3. Or merge individually: gt merge"
}

# Merge all approved port PRs
merge_all() {
    log "Merging all approved port PRs..."

    # Use Graphite to merge the stack
    gt stack submit
    gt merge --all

    success "Merged all approved port PRs"
}

# Main
main() {
    check_prereqs

    case "${1:-}" in
        --list|-l)
            list_port_prs
            ;;
        --merge-all|-m)
            merge_all
            ;;
        --stack|-s)
            show_stack
            ;;
        --help|-h)
            echo "Usage: $0 [--list|--merge-all|--stack]"
            echo ""
            echo "Options:"
            echo "  --list, -l       List pending port PRs"
            echo "  --merge-all, -m  Merge all approved port PRs"
            echo "  --stack, -s      Show current Graphite stack"
            echo "  (no args)        Interactive review"
            ;;
        *)
            interactive_review
            ;;
    esac
}

main "$@"
