#!/usr/bin/env bash
# Sets the app version in every file that holds it.
# Usage: scripts/set-version.sh 0.1.4
set -euo pipefail
new=${1:?usage: scripts/set-version.sh X.Y.Z}
[[ $new =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "invalid version: $new" >&2; exit 1; }
cd "$(dirname "$0")/.."
old=$(grep -E '^\s+version:\s*"' build/config.yml | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
[ "$old" != "$new" ] || { echo "already at version $new"; exit 0; }
q=$(printf '%s' "$old" | sed 's/[.]/[.]/g') # literal dots in the regexes

sed -i -E "s/(version: \")$q\"/\1$new\"/" build/config.yml build/linux/nfpm/nfpm.yaml
sed -i -E "s/\"$q\"/\"$new\"/g" build/windows/info.json
sed -i -E "s#<string>$q</string>#<string>$new</string>#g" build/darwin/Info.plist build/darwin/Info.dev.plist
sed -i -E "s/(INFO_PRODUCTVERSION \")$q\"/\1$new\"/" build/windows/nsis/wails_tools.nsh
# the manifest and MSIX want four numbers
sed -i -E "s/version=\"$q\.0\"/version=\"$new.0\"/" build/windows/wails.exe.manifest
sed -i -E "s/Version=\"$q\.0\"/Version=\"$new.0\"/" build/windows/msix/app_manifest.xml build/windows/msix/template.xml
sed -i -E "s/(var Version = \")$q\"/\1$new\"/" internal/core/library.go
sed -i -E "0,/\"version\": \"$q\"/s//\"version\": \"$new\"/" frontend/package.json
(cd frontend && npm install --package-lock-only --silent)

# check: the old version (as a whole: 0.1.9 must not match 0.1.91) is left in none of the version files.
# Only these files: elsewhere (tests, npm dependencies) the same number can appear legitimately.
files=(build/config.yml build/linux/nfpm/nfpm.yaml build/windows/info.json build/darwin/Info.plist build/darwin/Info.dev.plist
  build/windows/nsis/wails_tools.nsh build/windows/wails.exe.manifest build/windows/msix/app_manifest.xml build/windows/msix/template.xml
  internal/core/library.go frontend/package.json)
pattern="(^|[^0-9.])$q([^0-9]|$)"
left=$(grep -nHE "$pattern" "${files[@]}" || true)
# package-lock.json: only the app's own entries (the first lines), not the dependencies
left+=$(head -12 frontend/package-lock.json | grep -nE "$pattern" | sed 's#^#frontend/package-lock.json:#' || true)
if [ -n "$left" ]; then
  echo "version $old still present:" >&2
  echo "$left" >&2
  exit 1
fi
echo "$old → $new"
