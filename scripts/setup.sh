#!/usr/bin/env bash
# setup.sh — bootstrap a derived project from hellnet-cli-template.
#
# Interactive by default; scriptable with flags:
#   ./scripts/setup.sh -n mycli -m github.com/me/mycli [-y]
set -euo pipefail

OLD_MODULE="github.com/guilhermelinosp/hellnet-cli-template"
OLD_APP="app"
OLD_BINARY="hellnet-cli-template"

APP_NAME=""
MODULE=""
ASSUME_YES=0

usage() {
  sed -n '2,6p' "$0" | sed 's/^# \{0,1\}//'
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
[[ -z "$MODULE" ]] && read -r -p "Go module path (e.g. github.com/you/${APP_NAME}): " MODULE

if ! [[ "$APP_NAME" =~ ^[a-z][a-z0-9-]*$ ]]; then
  echo "ERROR: app name must be lowercase kebab-case starting with a letter" >&2
  exit 1
fi
if [[ -z "$MODULE" ]]; then
  echo "ERROR: module path is required" >&2
  exit 1
fi

echo
say "Renaming application : ${OLD_APP} -> ${APP_NAME}"
say "Module path          : ${OLD_MODULE} -> ${MODULE}"
if [[ "$ASSUME_YES" != 1 ]]; then
  read -r -p "Proceed? [y/N] " confirm
  [[ "${confirm:-n}" =~ ^[Yy]$ ]] || { echo "aborted"; exit 1; }
fi

# 1) module path everywhere: go.mod cascades into internal imports; the
#    Makefile carries ldflags -X targets that must follow along.
say "Rewriting module references..."
grep -rl --include='*.go' --exclude-dir=.git "$OLD_MODULE" . | xargs sed -i '' \
  -e "s|${OLD_MODULE}|${MODULE}|g" 2>/dev/null \
|| grep -rl --include='*.go' --exclude-dir=.git "$OLD_MODULE" . | xargs sed -i \
     -e "s|${OLD_MODULE}|${MODULE}|g"
sed -i '' "s|module ${OLD_MODULE}|module ${MODULE}|" go.mod 2>/dev/null \
  || sed -i "s|module ${OLD_MODULE}|module ${MODULE}|" go.mod
sed -i '' "s|${OLD_MODULE}|${MODULE}|g" Makefile 2>/dev/null \
  || sed -i "s|${OLD_MODULE}|${MODULE}|g" Makefile

# 2) rename the command directory + Makefile entrypoint/binary defaults.
if [[ -d "cmd/${OLD_APP}" ]]; then
  say "Moving cmd/${OLD_APP} -> cmd/${APP_NAME}"
  mv "cmd/${OLD_APP}" "cmd/${APP_NAME}"
fi
sed -i '' -e "s|^APP ?= .*|APP ?= ${APP_NAME}|" \
          -e "s|^CMD_DIR := .*|CMD_DIR := ./cmd/${APP_NAME}|" Makefile 2>/dev/null \
  || sed -i     -e "s|^APP ?= .*|APP ?= ${APP_NAME}|" \
                -e "s|^CMD_DIR := .*|CMD_DIR := ./cmd/${APP_NAME}|" Makefile

# 3) drop template history noise so the repo starts clean.
rm -f app coverage.out coverage.html
go mod tidy

say "Smoke test..."
go build ./...
go test ./...

cat <<EOF

✅ Setup complete.

Next steps:
  1. Review README.md and replace the description of your tool
  2. make build        # produces bin/${APP_NAME}
  3. ./bin/${APP_NAME} --help
  4. Start writing business commands in cmd/${APP_NAME}/
     (copy newHealthCommand as the reference shape)

Optional:
  git tag v0.1.0 && git push origin main v0.1.0   # triggers the release pipeline
EOF
