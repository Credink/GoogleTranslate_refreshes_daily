#!/usr/bin/env bash
# Build macOS arm64 (Apple Silicon) Wails app + zip. Run on macOS 14+ / CI macos-14.
# Usage (from repo root):
#   bash scripts/package-darwin.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN_DIR="$ROOT/build/bin"
DIST_DIR="$ROOT/build/dist"
mkdir -p "$BIN_DIR" "$DIST_DIR"

VER="${VERSION:-0.1.2}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILT="$(date -Iseconds)"
LD="-X main.BuildCommit=${COMMIT} -X main.BuildTime=${BUILT}"

echo "==> wails build (darwin/arm64 commit=$COMMIT)"
# Prefer native arm64 on Apple Silicon runners.
wails build -platform darwin/arm64 -ldflags "$LD"

APP_BUNDLE="$BIN_DIR/GoogleTranslateHostsUpdater.app"
if [[ ! -d "$APP_BUNDLE" ]]; then
  APP_BUNDLE="$(find "$BIN_DIR" -maxdepth 2 -type d -name 'GoogleTranslateHostsUpdater.app' | head -n1 || true)"
fi
if [[ -z "${APP_BUNDLE}" || ! -d "$APP_BUNDLE" ]]; then
  echo "missing .app under $BIN_DIR" >&2
  exit 1
fi

STAMP="$(date +%Y%m%d-%H%M)"
ZIP="$DIST_DIR/GoogleTranslateHostsUpdater-v${VER}-darwin-arm64-${STAMP}.zip"
rm -f "$ZIP"
ditto -c -k --sequesterRsrc --keepParent "$APP_BUNDLE" "$ZIP"
echo "macOS package: $ZIP"
echo "Note: notarization/DMG not included; Gatekeeper may require manual allow on first open."
