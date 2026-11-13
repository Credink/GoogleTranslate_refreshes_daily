#!/usr/bin/env bash
# Build Linux amd64 Wails binary + AppImage into packages/.
# DMG is a macOS-only format — Linux Release asset is AppImage (x86_64 desktop).
# Usage (from repo root):
#   bash scripts/package-linux.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN_DIR="$ROOT/build/bin"
PKG_DIR="$ROOT/packages"
TOOLS_DIR="$ROOT/tools"
mkdir -p "$BIN_DIR" "$PKG_DIR" "$TOOLS_DIR"

VER="${VERSION:-0.1.3}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILT="$(date -Iseconds)"
LD="-X main.BuildCommit=${COMMIT} -X main.BuildTime=${BUILT}"
APP="GoogleTranslateHostsUpdater"
DESKTOP_ID="googletranslatehostsupdater"

echo "==> wails build (linux/amd64 commit=$COMMIT)"
wails build -platform linux/amd64 -ldflags "$LD"

BIN="$BIN_DIR/$APP"
if [[ ! -f "$BIN" ]]; then
  BIN="$(find "$BIN_DIR" -type f -name "$APP" | head -n1 || true)"
fi
if [[ -z "${BIN}" || ! -f "$BIN" ]]; then
  echo "missing binary under $BIN_DIR" >&2
  exit 1
fi

STAMP="$(date +%Y%m%d-%H%M)"
APPDIR="$PKG_DIR/.AppDir-linux-amd64"
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/applications" "$APPDIR/usr/share/icons/hicolor/256x256/apps"

cp "$BIN" "$APPDIR/usr/bin/$APP"
chmod +x "$APPDIR/usr/bin/$APP"

# AppRun entry
cat >"$APPDIR/AppRun" <<EOF
#!/bin/sh
HERE="\$(dirname "\$(readlink -f "\$0")")"
exec "\$HERE/usr/bin/$APP" "\$@"
EOF
chmod +x "$APPDIR/AppRun"

ICON_SRC=""
for candidate in "$ROOT/build/appicon.png" "$ROOT/build/darwin/icon.png"; do
  if [[ -f "$candidate" ]]; then
    ICON_SRC="$candidate"
    break
  fi
done
if [[ -n "$ICON_SRC" ]]; then
  cp "$ICON_SRC" "$APPDIR/${DESKTOP_ID}.png"
  cp "$ICON_SRC" "$APPDIR/.DirIcon"
  cp "$ICON_SRC" "$APPDIR/usr/share/icons/hicolor/256x256/apps/${DESKTOP_ID}.png"
fi

cat >"$APPDIR/${DESKTOP_ID}.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=谷歌翻译 Hosts 更新器
Comment=Google Translate hosts availability checker
Exec=$APP
Icon=$DESKTOP_ID
Categories=Network;Utility;
Terminal=false
StartupNotify=true
EOF
cp "$APPDIR/${DESKTOP_ID}.desktop" "$APPDIR/usr/share/applications/${DESKTOP_ID}.desktop"

# Fetch appimagetool (cached under tools/)
AIT="$TOOLS_DIR/appimagetool-x86_64.AppImage"
AIT_URL="https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"
if [[ ! -x "$AIT" ]]; then
  echo "==> downloading appimagetool"
  curl -fsSL -o "$AIT" "$AIT_URL"
  chmod +x "$AIT"
fi

OUT_NAME="${APP}-v${VER}-linux-amd64-${STAMP}.AppImage"
OUT="$PKG_DIR/$OUT_NAME"
rm -f "$OUT"

# CI runners often lack FUSE; extract-and-run avoids mount.
export APPIMAGE_EXTRACT_AND_RUN=1
export ARCH=x86_64
export VERSION="$VER"

echo "==> appimagetool → $OUT"
(
  cd "$PKG_DIR"
  # Destination must be absolute or relative to cwd; keep name stable for Release.
  "$AIT" --no-appstream "$APPDIR" "$OUT_NAME"
)

rm -rf "$APPDIR"

if [[ ! -f "$OUT" ]]; then
  echo "AppImage not produced: $OUT" >&2
  exit 1
fi
chmod +x "$OUT"
echo "Linux AppImage: $OUT"
echo "Note: DMG is macOS-only; AppImage is the Linux equivalent single-file package."
echo "Runtime still expects desktop WebKitGTK/GTK on the host (not fully bundled)."
