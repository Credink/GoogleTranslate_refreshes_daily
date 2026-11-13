#!/usr/bin/env bash
# Build macOS arm64 (Apple Silicon) Wails .app + UDZO .dmg into packages/.
# Run on macOS / CI macos-latest (arm64). DMG is the Release primary asset.
# Usage (from repo root):
#   bash scripts/package-darwin.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN_DIR="$ROOT/build/bin"
PKG_DIR="$ROOT/packages"
mkdir -p "$BIN_DIR" "$PKG_DIR"

VER="${VERSION:-0.1.3}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILT="$(date -Iseconds)"
LD="-X main.BuildCommit=${COMMIT} -X main.BuildTime=${BUILT}"
APP_NAME="GoogleTranslateHostsUpdater"
VOL_NAME="谷歌翻译 Hosts 更新器"

ARCH="$(uname -m)"
PLATFORM="darwin/arm64"
if [[ "$ARCH" != "arm64" ]]; then
  echo "warning: runner arch=$ARCH; Apple Silicon (arm64) is the supported Release target" >&2
  PLATFORM="darwin/${ARCH}"
fi

echo "==> wails build ($PLATFORM commit=$COMMIT)"
wails build -platform "$PLATFORM" -ldflags "$LD"

APP_BUNDLE="$BIN_DIR/${APP_NAME}.app"
if [[ ! -d "$APP_BUNDLE" ]]; then
  APP_BUNDLE="$(find "$BIN_DIR" -maxdepth 2 -type d -name "${APP_NAME}.app" | head -n1 || true)"
fi
if [[ -z "${APP_BUNDLE}" || ! -d "$APP_BUNDLE" ]]; then
  echo "missing .app under $BIN_DIR" >&2
  exit 1
fi

STAMP="$(date +%Y%m%d-%H%M)"
ARCH_TAG="arm64"
if [[ "$PLATFORM" != "darwin/arm64" ]]; then
  ARCH_TAG="$ARCH"
fi
DMG="$PKG_DIR/${APP_NAME}-v${VER}-darwin-${ARCH_TAG}-${STAMP}.dmg"

STAGE="$(mktemp -d "${TMPDIR:-/tmp}/gthu-dmg.XXXXXX")"
cleanup() { rm -rf "$STAGE"; }
trap cleanup EXIT

cp -R "$APP_BUNDLE" "$STAGE/${APP_NAME}.app"
ln -s /Applications "$STAGE/Applications"

rm -f "$DMG"
# UDZO = zlib-compressed read-only DMG (standard “disk image” distribute format).
hdiutil create \
  -volname "$VOL_NAME" \
  -srcfolder "$STAGE" \
  -ov \
  -format UDZO \
  "$DMG"

echo "macOS DMG: $DMG"
echo "Note: notarization not included; Gatekeeper may require System Settings → Privacy & Security → Open Anyway."
