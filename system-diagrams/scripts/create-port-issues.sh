#!/usr/bin/env bash
# create-port-issues.sh - Create beads issues for porting upstream commits
#
# Usage:
#   ./analyze-upstream.sh | ./create-port-issues.sh
#   ./create-port-issues.sh < analysis.json
#
# Creates beads issues with:
#   - Title: Port {sha}: {subject}
#   - Labels: upstream, port, {plugin}
#   - Priority based on affected plugins

set -euo pipefail

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[port-issues]${NC} $1" >&2; }
success() { echo -e "${GREEN}[port-issues]${NC} $1" >&2; }

# Read JSON from stdin
json=$(cat)

# Check if empty
if [[ "$json" == "[]" ]]; then
    log "No commits to port"
    exit 0
fi

# Process each commit
echo "$json" | jq -c '.[]' | while read -r commit; do
    sha=$(echo "$commit" | jq -r '.sha' | cut -c1-7)
    subject=$(echo "$commit" | jq -r '.subject')
    plugins=$(echo "$commit" | jq -r '.plugins | join(", ")')
    priority=$(echo "$commit" | jq -r '.priority')
    files_changed=$(echo "$commit" | jq -r '.files_changed')

    # Skip if no plugins affected (infra-only changes)
    if [[ -z "$plugins" || "$plugins" == "null" ]]; then
        log "Skipping $sha (no plugins affected)"
        continue
    fi

    # Map priority to beads priority (0=critical, 4=backlog)
    case "$priority" in
        high)   bd_priority=1 ;;
        medium) bd_priority=2 ;;
        low)    bd_priority=3 ;;
        *)      bd_priority=2 ;;
    esac

    # Build description
    description="Port upstream commit to v1 plugins.

**Upstream commit:** $sha
**Subject:** $subject
**Files changed:** $files_changed
**Affected plugins:** $plugins

## Porting checklist

- [ ] Review upstream changes
- [ ] Update affected plugin(s)
- [ ] Run compatibility tests
- [ ] Add git note (backport tracking)

## Commands

\`\`\`bash
# View the commit
git show $sha

# After porting, add git note
git notes add -m \"ported-from: $sha (upstream)\"
\`\`\`
"

    # Check if issue already exists
    existing=$(bd list --status=open 2>/dev/null | grep -c "Port $sha" || echo "0")
    if [[ "$existing" -gt 0 ]]; then
        log "Issue for $sha already exists, skipping"
        continue
    fi

    # Create the issue
    log "Creating issue for $sha: $subject"
    issue_id=$(bd create \
        --title "Port $sha: $subject" \
        --priority "$bd_priority" \
        --label "upstream" \
        --label "port" \
        -d "$description" \
        2>/dev/null | grep -oE 'beads-[a-z0-9]+' | head -1)

    if [[ -n "$issue_id" ]]; then
        success "Created $issue_id for $sha"
    fi
done

log "Done creating port issues"
