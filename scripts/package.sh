#!/usr/bin/env bash
# Build skvoz.exe and assemble a ready-to-use release zip:
#   dist/skvoz-<version>-windows-amd64.zip
# containing skvoz.exe, WinDivert binaries, domain lists, presets and README.
#
# Usage: scripts/package.sh [version]
set -euo pipefail

VERSION="${1:-dev}"
WINDIVERT_VER="2.2.2"
WINDIVERT_URL="https://reqrypt.org/download/WinDivert-${WINDIVERT_VER}-A.zip"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

STAGE="dist/skvoz-${VERSION}-windows-amd64"
rm -rf "$STAGE"
mkdir -p "$STAGE/lists"

echo ">> building skvoz.exe"
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$STAGE/skvoz.exe" ./cmd/skvoz

echo ">> fetching WinDivert ${WINDIVERT_VER}"
TMP="$(mktemp -d)"
curl -fsSL "$WINDIVERT_URL" -o "$TMP/windivert.zip"
unzip -q "$TMP/windivert.zip" -d "$TMP"
cp "$TMP/WinDivert-${WINDIVERT_VER}-A/x64/WinDivert.dll"   "$STAGE/"
cp "$TMP/WinDivert-${WINDIVERT_VER}-A/x64/WinDivert64.sys" "$STAGE/"
rm -rf "$TMP"

echo ">> staging assets"
cp lists/*.txt "$STAGE/lists/"
cp presets/*.bat "$STAGE/"
cp README.md LICENSE NOTICE "$STAGE/"

echo ">> zipping"
(cd dist && zip -qr "skvoz-${VERSION}-windows-amd64.zip" "skvoz-${VERSION}-windows-amd64")
echo ">> done: dist/skvoz-${VERSION}-windows-amd64.zip"
