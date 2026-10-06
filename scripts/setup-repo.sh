#!/usr/bin/env bash
# Apply the hellnet repository standard to the current GitHub repository (idempotent):
#   - delete branch on merge, auto-merge allowed, wiki off
#   - ruleset "main": no deletion, no force-push, changes through pull requests
#   - (no secret needed: the Octo STS GitHub App must be installed on the repository)
#
#   scripts/setup-repo.sh [--repo owner/name]
#
# Requires: gh (authenticated, with repo admin rights).
# The required status check "pr-gate" is NOT enabled here: enable it after the Octo STS App is installed
# on the repository, otherwise the labeler job fails and no pull request can be merged.
set -euo pipefail

REPO=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --repo) [ "$#" -ge 2 ] || { echo "missing value for --repo" >&2; exit 2; }; REPO="$2"; shift ;;
    -h|--help) sed -n '2,/^set -euo pipefail/{/^set -euo/!p;}' "$0"; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done

command -v gh >/dev/null || { echo "gh is required" >&2; exit 1; }
[ -n "$REPO" ] || REPO="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
echo "repository: $REPO"

gh api -X PATCH "repos/$REPO" -F delete_branch_on_merge=true -F allow_auto_merge=true -F has_wiki=false >/dev/null
echo "settings  : delete_branch_on_merge, allow_auto_merge, wiki off"

if gh api "repos/$REPO/rulesets" --jq '.[].name' | grep -qx main; then
  echo "ruleset   : 'main' already exists (skipped)"
else
  gh api -X POST "repos/$REPO/rulesets" --input - >/dev/null <<'JSON'
{
  "name": "main",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [],
  "conditions": { "ref_name": { "include": ["refs/heads/main"], "exclude": [] } },
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 0,
        "dismiss_stale_reviews_on_push": false,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false,
        "allowed_merge_methods": ["merge", "squash", "rebase"]
      }
    }
  ]
}
JSON
  echo "ruleset   : 'main' created"
fi

echo "octo sts  : install https://github.com/apps/octo-sts on $REPO (policies live in .github/chainguard/)"
