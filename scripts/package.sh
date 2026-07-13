#!/usr/bin/env bash
# Build the single, self-contained Skvoz release binary:
#   dist/skvoz.exe          - one file; driver + lists embedded
#   dist/skvoz.exe.sha256   - checksum for the release page
#
# Usage: scripts/package.sh [version]
set -euo pipefail

VERSION="${1:-dev}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Fetch the driver and copy lists into the embed dirs.
scripts/stage-assets.sh

echo ">> building skvoz.exe (${VERSION})"
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -H=windowsgui -X main.version=${VERSION}" \
  -o dist/skvoz.exe ./cmd/skvoz

echo ">> checksum"
if command -v sha256sum >/dev/null 2>&1; then
  ( cd dist && sha256sum skvoz.exe > skvoz.exe.sha256 )
else
  ( cd dist && shasum -a 256 skvoz.exe > skvoz.exe.sha256 )
fi

echo ">> done:"
ls -la dist/skvoz.exe dist/skvoz.exe.sha256
cat dist/skvoz.exe.sha256
