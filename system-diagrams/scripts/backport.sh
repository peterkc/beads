#!/usr/bin/env bash
# backport.sh - Cherry-pick with git notes tracking
# Usage: ./backport.sh <source-branch> <commit-sha> [reason]
#
# Example:
#   ./backport.sh next abc1234 "Row mapper benefits bd users"

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[backport]${NC} $1"; }
success() { echo -e "${GREEN}[backport]${NC} $1"; }
error() { echo -e "${RED}[backport]${NC} $1" >&2; }

# Args
SOURCE_BRANCH="${1:-}"
COMMIT_SHA="${2:-}"
REASON="${3:-Backported from $SOURCE_BRANCH}"

# Validate
if [ -z "$SOURCE_BRANCH" ] || [ -z "$COMMIT_SHA" ]; then
    error "Usage: backport.sh <source-branch> <commit-sha> [reason]"
    error ""
    error "Examples:"
    error "  backport.sh next abc1234"
    error "  backport.sh next abc1234 'DRY improvement for row scanning'"
    exit 1
fi

# Verify commit exists
if ! git cat-file -e "$COMMIT_SHA" 2>/dev/null; then
    error "Commit $COMMIT_SHA not found"
    error "Try: git fetch origin $SOURCE_BRANCH"
    exit 1
fi

# Check if already backported
if git notes show "$COMMIT_SHA" 2>/dev/null | grep -q "backported-to:"; then
    error "Commit $COMMIT_SHA was already backported:"
    git notes show "$COMMIT_SHA"
    exit 1
fi

# Store current branch
CURRENT_BRANCH=$(git branch --show-current)

log "Backporting $COMMIT_SHA from $SOURCE_BRANCH to main"
log "Reason: $REASON"

# Cherry-pick
git checkout main
if ! git cherry-pick "$COMMIT_SHA" --no-edit; then
    error "Cherry-pick failed. Resolve conflicts, then run:"
    error "  git cherry-pick --continue"
    error "  # Then manually add notes:"
    error "  git notes add -m 'backport-from: $COMMIT_SHA ($SOURCE_BRANCH)'"
    exit 1
fi

NEW_SHA=$(git rev-parse HEAD)

# Add notes
log "Adding git notes..."

git notes add -m "$(cat <<EOF
backport-from: $COMMIT_SHA ($SOURCE_BRANCH)
backport-reason: $REASON
backport-date: $(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF
)"

git notes add "$COMMIT_SHA" -m "backported-to: $NEW_SHA (main)"

# Push
log "Pushing to origin..."
git push origin main
git push origin refs/notes/*

success "✅ Backported successfully!"
success "   Source: $COMMIT_SHA ($SOURCE_BRANCH)"
success "   Target: $NEW_SHA (main)"
success ""
success "View notes:"
success "   git notes show $NEW_SHA"
success "   git notes show $COMMIT_SHA"

# Return to original branch
if [ "$CURRENT_BRANCH" != "main" ]; then
    git checkout "$CURRENT_BRANCH"
fi
