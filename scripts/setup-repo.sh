#!/usr/bin/env bash
# Apply the hellnet repository standard to the current GitHub repository (idempotent):
#   - delete branch on merge, auto-merge allowed, wiki off
#   - ruleset "main": no deletion, no force-push, changes through pull requests
#   - Actions variable HELLNET_ACTIONS_CLIENT_ID (the GitHub App used by the labeler and release jobs)
#   - optional secret HELLNET_ACTIONS_PRIVATE_KEY from a .pem file
#
#   scripts/setup-repo.sh [--repo owner/name] [--private-key path/to/key.pem]
#
# Requires: gh (authenticated, with repo admin rights).
# The required status check "pr-gate" is NOT enabled here: enable it only after the secret exists,
# otherwise the labeler job fails and no pull request can be merged.
set -euo pipefail

CLIENT_ID_DEFAULT="Iv23li8wpWDJ0c6a00Dv" # public client id of the hellnet-actions GitHub App
REPO=""
KEY_FILE=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --repo) [ "$#" -ge 2 ] || { echo "missing value for --repo" >&2; exit 2; }; REPO="$2"; shift ;;
    --private-key) [ "$#" -ge 2 ] || { echo "missing value for --private-key" >&2; exit 2; }; KEY_FILE="$2"; shift ;;
    -h|--help) sed -n '2,13p' "$0"; exit 0 ;;
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

gh variable set HELLNET_ACTIONS_CLIENT_ID -R "$REPO" --body "${HELLNET_ACTIONS_CLIENT_ID:-$CLIENT_ID_DEFAULT}"
echo "variable  : HELLNET_ACTIONS_CLIENT_ID set"

if [ -n "$KEY_FILE" ]; then
  gh secret set HELLNET_ACTIONS_PRIVATE_KEY -R "$REPO" < "$KEY_FILE"
  echo "secret    : HELLNET_ACTIONS_PRIVATE_KEY set from $KEY_FILE"
elif gh secret list -R "$REPO" | grep -q '^HELLNET_ACTIONS_PRIVATE_KEY'; then
  echo "secret    : HELLNET_ACTIONS_PRIVATE_KEY already present"
else
  echo "secret    : MISSING. The labeler and release jobs need it:"
  echo "            gh secret set HELLNET_ACTIONS_PRIVATE_KEY -R $REPO < path/to/key.pem"
  echo "            (the hellnet-actions GitHub App must also be installed on this repository)"
fi
