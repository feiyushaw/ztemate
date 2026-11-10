#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ "$(uname -s)" != Darwin ]]; then
  echo 'DMG packaging requires macOS.' >&2
  exit 1
fi
target="${1:-darwin/arm64}"
shift || true
case "$target" in darwin/arm64|darwin/amd64|darwin/universal) ;; *) echo "Unsupported platform: $target" >&2; exit 1 ;; esac
# Sign on the native temporary filesystem: external volumes can add Finder
# metadata that codesign rejects. Copy the finished disk image afterwards.
stage="$(mktemp -d "${TMPDIR:-/tmp}/ztemate-package.XXXXXX")"
trap 'rm -rf "$stage"' EXIT
go run github.com/egoist/mygo/cmd/mygo build -platform "$target" -o "$stage" "$@"
destination="build/${target//\//-}"
mkdir -p "$destination"
found=false
for artifact in "$stage"/*/*.dmg; do
  [[ -f "$artifact" ]] || continue
  cp "$artifact" "$destination/"
  echo "DMG: $PWD/$destination/$(basename "$artifact")"
  found=true
done
if [[ "$found" != true ]]; then echo 'No DMG produced. Check the MyGo output above.' >&2; exit 1; fi
