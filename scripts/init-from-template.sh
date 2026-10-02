#!/usr/bin/env bash
# Initialise a repository created from a hellnet-*-template: renames the Go module, imports,
# README/CI references and (for services) the cmd/<service> directory.
#
#   scripts/init-from-template.sh <repo-name> [service-name]
#
#   repo-name     new repository name, e.g. fast-listeners
#   service-name  new cmd/ directory and binary name, e.g. listeners (optional; ignored for libraries)
#
# The owner is taken from the origin remote (override with OWNER=...). Run it once, right after
# "Use this template", review `git diff`, then commit. Afterwards run scripts/setup-repo.sh.
set -euo pipefail

cd "$(dirname "$0")/.."

NAME="${1:-}"
SERVICE="${2:-}"
if [ -z "$NAME" ] || [ "$#" -gt 2 ]; then
  echo "usage: $0 <repo-name> [service-name]" >&2
  exit 2
fi

OWNER="${OWNER:-$(git remote get-url origin 2>/dev/null | sed -E 's#.*[:/]([^/]+)/[^/]+$#\1#')}"
[ -n "$OWNER" ] || { echo "cannot detect the owner: set OWNER=<github-user>" >&2; exit 1; }

OLD_MODULE="$(sed -n '1s/^module //p' go.mod)"
OLD_NAME="${OLD_MODULE##*/}"
NEW_MODULE="github.com/${OWNER}/${NAME}"

if [ "$OLD_MODULE" = "$NEW_MODULE" ]; then
  echo "already initialised: module is $NEW_MODULE"
  exit 0
fi
case "$OLD_NAME" in
  *-template) ;;
  *) echo "unexpected module '$OLD_MODULE': this script only runs on an untouched template" >&2; exit 1 ;;
esac

# Text files tracked by git, except lockfiles and this script (no mapfile: macOS ships bash 3.2).
load_files() {
  files=()
  while IFS= read -r f; do files+=("$f"); done < <(git ls-files | grep -v -E '^(go\.sum|scripts/init-from-template\.sh)$')
}
load_files

replace() { # replace <from> <to>
  local from="$1" to="$2" f
  for f in "${files[@]}"; do
    [ -f "$f" ] && grep -Iq . "$f" 2>/dev/null && grep -qF -- "$from" "$f" && sed -i.bak "s|$from|$to|g" "$f" && rm -f "$f.bak"
  done
  return 0
}

replace "$OLD_MODULE" "$NEW_MODULE"
replace "$OLD_NAME" "$NAME"

# Service entry point: cmd/<old> -> cmd/<service> (single-binary templates only).
if [ -n "$SERVICE" ] && [ -d cmd ]; then
  cmds=()
  while IFS= read -r d; do cmds+=("$d"); done < <(find cmd -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)
  if [ "${#cmds[@]}" -eq 1 ] && [ "${cmds[0]}" != "$SERVICE" ]; then
    OLD_CMD="${cmds[0]}"
    git mv "cmd/$OLD_CMD" "cmd/$SERVICE"
    load_files
    replace "cmd/$OLD_CMD" "cmd/$SERVICE"
    # Binary name in the Containerfile (/bin/<svc>, "/<svc>", "COPY ... /<svc>") and in GoReleaser.
    for f in Containerfile .goreleaser.yaml; do
      [ -f "$f" ] || continue
      sed -i.bak -E \
        -e "s#/bin/${OLD_CMD}([^A-Za-z0-9_-]|\$)#/bin/${SERVICE}\1#g" \
        -e "s#([[:space:]\"])/${OLD_CMD}([[:space:]\"]|\$)#\1/${SERVICE}\2#g" \
        -e "s#([[:space:]\"])/${OLD_CMD}([[:space:]\"]|\$)#\1/${SERVICE}\2#g" \
        -e "s#^([[:space:]-]*(id|binary):[[:space:]]*)${OLD_CMD}\$#\1${SERVICE}#" "$f" && rm -f "$f.bak"
    done
    echo "service: cmd/$OLD_CMD -> cmd/$SERVICE"
  fi
fi

echo "module : $OLD_MODULE -> $NEW_MODULE"
echo
echo "Next steps:"
echo "  1. go build ./... && go vet ./... && go test ./...   # sanity check"
echo "  2. git add -A && git commit -m 'chore(module): initialise from template'"
echo "  3. scripts/setup-repo.sh                             # repo settings, ruleset and CI variable"
