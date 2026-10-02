#!/usr/bin/env bash
# setup.sh — bootstrap a derived project from hellnet-cli-template.
#
# Interactive by default; scriptable with flags:
#   ./scripts/setup.sh -n mycli -m github.com/me/mycli [-y]
#
# Renames the Go module, imports, cmd/app, the binary and the release metadata
# (delegates to scripts/init-from-template.sh), then runs `go mod tidy`, a build and the tests.
set -euo pipefail

cd "$(dirname "$0")/.."

APP_NAME=""
MODULE=""
ASSUME_YES=0

usage() {
  sed -n '2,7p' "$0" | sed 's/^# \{0,1\}//'
  exit 0
}

while getopts "n:m:yh" opt; do
  case $opt in
    n) APP_NAME="$OPTARG" ;;
    m) MODULE="$OPTARG" ;;
    y) ASSUME_YES=1 ;;
    h) usage ;;
    *) usage ;;
  esac
done

say() { printf '\033[36m==>\033[0m %s\n' "$1"; }

if [[ -z "$APP_NAME" ]]; then
  read -r -p "Application/binary name (kebab-case, e.g. mycli): " APP_NAME
fi
if [[ -z "$MODULE" ]]; then
  read -r -p "Go module path (e.g. github.com/you/${APP_NAME}): " MODULE
fi

if ! [[ "$APP_NAME" =~ ^[a-z][a-z0-9-]*$ ]]; then
  echo "ERROR: app name must be lowercase kebab-case starting with a letter" >&2
  exit 1
fi
if ! [[ "$MODULE" =~ ^github\.com/[^/]+/[^/]+$ ]]; then
  echo "ERROR: module path must look like github.com/<owner>/<repo>" >&2
  exit 1
fi

OWNER="${MODULE#github.com/}"
OWNER="${OWNER%%/*}"
REPO="${MODULE##*/}"

echo
say "Application : app -> ${APP_NAME}"
say "Module      : $(sed -n '1s/^module //p' go.mod) -> ${MODULE}"
if [[ "$ASSUME_YES" != 1 ]]; then
  read -r -p "Proceed? [y/N] " confirm
  [[ "${confirm:-n}" =~ ^[Yy]$ ]] || { echo "aborted"; exit 1; }
fi

say "Renaming module, imports and cmd/app..."
OWNER="$OWNER" scripts/init-from-template.sh "$REPO" "$APP_NAME"

# Drop template build leftovers so the repo starts clean.
rm -f app coverage.out coverage.html
go mod tidy

say "Smoke test..."
go build ./...
go test ./...

cat <<EOF

✅ Setup complete.

Next steps:
  1. Review README.md and replace the description of your tool
  2. go build -o bin/${APP_NAME} ./cmd/${APP_NAME}
  3. ./bin/${APP_NAME} --help
  4. Start writing business commands in cmd/${APP_NAME}/
     (copy newHealthCommand as the reference shape)
  5. scripts/setup-repo.sh        # repo settings, ruleset and CI variable

Optional:
  git tag v0.1.0 && git push origin main v0.1.0   # triggers the release pipeline
EOF
