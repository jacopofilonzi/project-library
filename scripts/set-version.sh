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

# check: the old version must not appear anywhere any more (except the sample comment in project.nsi)
if left=$(git grep -nIF "$old" -- build internal frontend/package.json frontend/package-lock.json | grep -v 'project.nsi'); then
  echo "version $old still present:" >&2
  echo "$left" >&2
  exit 1
fi
echo "$old → $new"
