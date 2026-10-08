#!/usr/bin/env bash
# Imposta la versione dell'app in tutti i file che la contengono.
# Uso: scripts/set-version.sh 0.1.4
set -euo pipefail
new=${1:?uso: scripts/set-version.sh X.Y.Z}
[[ $new =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "versione non valida: $new" >&2; exit 1; }
cd "$(dirname "$0")/.."
old=$(grep -E '^\s+version:\s*"' build/config.yml | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
[ "$old" != "$new" ] || { echo "già alla versione $new"; exit 0; }
q=$(printf '%s' "$old" | sed 's/[.]/[.]/g') # punti letterali nelle regex

sed -i -E "s/(version: \")$q\"/\1$new\"/" build/config.yml build/linux/nfpm/nfpm.yaml
sed -i -E "s/\"$q\"/\"$new\"/g" build/windows/info.json
sed -i -E "s#<string>$q</string>#<string>$new</string>#g" build/darwin/Info.plist build/darwin/Info.dev.plist
sed -i -E "s/(INFO_PRODUCTVERSION \")$q\"/\1$new\"/" build/windows/nsis/wails_tools.nsh
# manifest e MSIX vogliono quattro numeri
sed -i -E "s/version=\"$q\.0\"/version=\"$new.0\"/" build/windows/wails.exe.manifest
sed -i -E "s/Version=\"$q\.0\"/Version=\"$new.0\"/" build/windows/msix/app_manifest.xml build/windows/msix/template.xml
sed -i -E "s/(var Version = \")$q\"/\1$new\"/" internal/core/library.go
sed -i -E "0,/\"version\": \"$q\"/s//\"version\": \"$new\"/" frontend/package.json
(cd frontend && npm install --package-lock-only --silent)

# controllo: la vecchia versione non deve comparire più (a parte il commento di esempio in project.nsi)
if left=$(git grep -nIF "$old" -- build internal frontend/package.json frontend/package-lock.json | grep -v 'project.nsi'); then
  echo "versione $old ancora presente:" >&2
  echo "$left" >&2
  exit 1
fi
echo "$old → $new"
