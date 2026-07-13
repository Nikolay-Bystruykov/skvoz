#!/usr/bin/env bash
# Stage the resources that get embedded into skvoz.exe:
#   - domain lists  -> internal/assets/lists/
#   - WinDivert 2.2.x driver (fetched) -> internal/assets/bin/
# The checked-in bin/ files are tiny placeholders so `go build` works on a fresh
# clone; this script overwrites them with the real signed driver before a
# shipping build. Run by scripts/package.sh and the CI/release workflows.
set -euo pipefail

WINDIVERT_VER="2.2.2"
WINDIVERT_URL="https://reqrypt.org/download/WinDivert-${WINDIVERT_VER}-A.zip"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p internal/assets/bin internal/assets/lists
cp lists/*.txt internal/assets/lists/

echo ">> fetching WinDivert ${WINDIVERT_VER}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$WINDIVERT_URL" -o "$TMP/windivert.zip"
unzip -q "$TMP/windivert.zip" -d "$TMP"
cp "$TMP/WinDivert-${WINDIVERT_VER}-A/x64/WinDivert.dll"   internal/assets/bin/WinDivert.dll
cp "$TMP/WinDivert-${WINDIVERT_VER}-A/x64/WinDivert64.sys" internal/assets/bin/WinDivert64.sys

# Sanity check: the real DLL is ~hundreds of KB, the placeholder is a few bytes.
# Fail loudly rather than ship an exe with a placeholder driver embedded.
sz="$(wc -c < internal/assets/bin/WinDivert.dll | tr -d ' ')"
if [ "$sz" -lt 10000 ]; then
  echo "ERROR: WinDivert.dll is only ${sz} bytes — fetch failed, refusing to build" >&2
  exit 1
fi
echo ">> assets staged (WinDivert.dll ${sz} bytes)"
