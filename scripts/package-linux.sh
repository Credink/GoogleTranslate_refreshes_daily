#!/usr/bin/env bash
# Build Linux amd64 Wails binary + AppImage into packages/.
# Bundles WebKitGTK / GTK3 / GLib / libsoup / cairo-pango / appindicator via
# linuxdeploy + GTK plugin (Wails-style helper relocation).
# DMG is macOS-only — Linux Release asset is AppImage (x86_64 desktop).
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
ARCH_LABEL="x86_64"

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
STAGE="$PKG_DIR/.linuxdeploy-stage"
APPDIR="$STAGE/AppDir"
rm -rf "$STAGE"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/lib" "$APPDIR/usr/share/applications" \
  "$APPDIR/usr/share/icons/hicolor/256x256/apps"

cp "$BIN" "$APPDIR/usr/bin/$APP"
chmod +x "$APPDIR/usr/bin/$APP"

ICON_SRC=""
for candidate in "$ROOT/build/appicon.png" "$ROOT/build/darwin/icon.png"; do
  if [[ -f "$candidate" ]]; then
    ICON_SRC="$candidate"
    break
  fi
done
ICON_FOR_DEPLOY=""
if [[ -n "$ICON_SRC" ]]; then
  # linuxdeploy only accepts fixed icon sizes (max 512); source appicon is often 1024.
  ICON_FOR_DEPLOY="$STAGE/${DESKTOP_ID}-256.png"
  mkdir -p "$STAGE"
  if command -v convert >/dev/null 2>&1; then
    convert "$ICON_SRC" -resize 256x256 "$ICON_FOR_DEPLOY"
  elif command -v magick >/dev/null 2>&1; then
    magick "$ICON_SRC" -resize 256x256 "$ICON_FOR_DEPLOY"
  else
    python3 - "$ICON_SRC" "$ICON_FOR_DEPLOY" <<'PY'
import struct, zlib, sys
src, dst = sys.argv[1], sys.argv[2]
# Prefer Pillow when available; else keep a copied PNG only if already 256 (rare).
try:
    from PIL import Image
    Image.open(src).convert("RGBA").resize((256, 256), Image.Resampling.LANCZOS).save(dst)
except Exception as exc:
    raise SystemExit(
        "need ImageMagick `convert`/`magick` or Pillow to resize app icon for linuxdeploy "
        f"(got {exc})"
    )
PY
  fi
  cp "$ICON_FOR_DEPLOY" "$APPDIR/.DirIcon"
  cp "$ICON_FOR_DEPLOY" "$APPDIR/usr/share/icons/hicolor/256x256/apps/${DESKTOP_ID}.png"
  ln -sfn .DirIcon "$APPDIR/${DESKTOP_ID}.png"
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

write_apprun() {
  # Wails-style AppRun: start from usr/lib so relocated WebKit helper paths resolve
  # both outside and inside WebKit's bubblewrap sandbox. Also source linuxdeploy GTK hooks.
  cat >"$APPDIR/AppRun" <<'EOF'
#!/bin/sh
set -e
APPDIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
export APPDIR
OWD="${OWD:-$PWD}"
export OWD
if [ -d "$APPDIR/apprun-hooks" ]; then
  for _hook in "$APPDIR"/apprun-hooks/*; do
    [ -f "$_hook" ] || continue
    # shellcheck disable=SC1090
    . "$_hook"
  done
fi
usr="$APPDIR/usr"
export PATH="$usr/bin:${PATH:-/usr/bin}"
export LD_LIBRARY_PATH="$usr/lib:$usr/lib/x86_64-linux-gnu:$usr/lib/aarch64-linux-gnu:$usr/lib64${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
export XDG_DATA_DIRS="$usr/share:${XDG_DATA_DIRS:-/usr/local/share:/usr/share}"
export GSETTINGS_SCHEMA_DIR="$usr/share/glib-2.0/schemas"
for bundle in "$usr"/lib/webkit*gtk-*/injected-bundle/*.so \
              "$usr"/lib/*/webkit*gtk-*/injected-bundle/*.so \
              "$usr"/lib*/*/webkit*gtk-*/injected-bundle/*.so; do
  if [ -e "$bundle" ]; then
    export WEBKIT_INJECTED_BUNDLE_PATH="$(dirname "$bundle")"
    break
  fi
done
cd "$usr/lib"
exec "$usr/bin/GoogleTranslateHostsUpdater" "$@"
EOF
  chmod +x "$APPDIR/AppRun"
}
write_apprun

# --- locate WebKit helper processes + injected bundle (webkit2gtk-4.0 on Ubuntu 22.04) ---
find_runtime_file() {
  local name="$1"
  local found=""
  found="$(find /usr/lib /usr/lib64 /usr/libexec -type f -name "$name" 2>/dev/null | head -n1 || true)"
  if [[ -z "$found" ]]; then
    echo "required WebKit runtime file not found: $name" >&2
    exit 1
  fi
  printf '%s\n' "$found"
}

echo "==> staging WebKit helpers into AppDir"
WEB_PROCESS="$(find_runtime_file WebKitWebProcess)"
NET_PROCESS="$(find_runtime_file WebKitNetworkProcess)"
INJECTED="$(find_runtime_file libwebkit2gtkinjectedbundle.so)"
HELPER_DIR="$(dirname "$WEB_PROCESS")"
HELPER_DIRS=("$HELPER_DIR")

copy_preserving_abs() {
  local src="$1"
  local dest="$APPDIR${src}"
  mkdir -p "$(dirname "$dest")"
  cp -a "$src" "$dest"
}

copy_preserving_abs "$WEB_PROCESS"
copy_preserving_abs "$NET_PROCESS"
copy_preserving_abs "$INJECTED"
if [[ -f "$HELPER_DIR/WebKitGPUProcess" ]]; then
  copy_preserving_abs "$HELPER_DIR/WebKitGPUProcess"
fi

# --- linuxdeploy + GTK plugin ---
LINUXDEPLOY="$TOOLS_DIR/linuxdeploy-${ARCH_LABEL}.AppImage"
LINUXDEPLOY_URL="https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-${ARCH_LABEL}.AppImage"
GTK_PLUGIN="$TOOLS_DIR/linuxdeploy-plugin-gtk.sh"
GTK_PLUGIN_URL="https://raw.githubusercontent.com/linuxdeploy/linuxdeploy-plugin-gtk/master/linuxdeploy-plugin-gtk.sh"

if [[ ! -x "$LINUXDEPLOY" ]]; then
  echo "==> downloading linuxdeploy"
  curl -fsSL -o "$LINUXDEPLOY" "$LINUXDEPLOY_URL"
  chmod +x "$LINUXDEPLOY"
fi
if [[ ! -f "$GTK_PLUGIN" ]]; then
  echo "==> downloading linuxdeploy-plugin-gtk"
  curl -fsSL -o "$GTK_PLUGIN" "$GTK_PLUGIN_URL"
  chmod +x "$GTK_PLUGIN"
fi
# linuxdeploy discovers plugins beside its binary or on PATH
cp -f "$GTK_PLUGIN" "$STAGE/linuxdeploy-plugin-gtk.sh"
chmod +x "$STAGE/linuxdeploy-plugin-gtk.sh"

# Optional tray libs (energye/systray → ayatana-appindicator / dbusmenu)
EXTRA_LIBS=()
add_lib_if_present() {
  local pattern="$1"
  local hit=""
  hit="$(ldconfig -p 2>/dev/null | awk -v p="$pattern" '$1 ~ p {print $NF; exit}')"
  if [[ -n "$hit" && -e "$hit" ]]; then
    EXTRA_LIBS+=("--library=$hit")
    echo "    + tray/dep library: $hit"
  fi
}
echo "==> resolving tray / indicator libraries"
add_lib_if_present 'libayatana-appindicator3\.so'
add_lib_if_present 'libappindicator3\.so'
add_lib_if_present 'libayatana-indicator3\.so'
add_lib_if_present 'libindicator3\.so'
add_lib_if_present 'libdbusmenu-glib\.so'
add_lib_if_present 'libdbusmenu-gtk3\.so'

# Helper process deps (webkit subprocesses are not the main ELF)
HELPER_DEP_ARGS=()
for helper in WebKitWebProcess WebKitNetworkProcess WebKitGPUProcess; do
  staged="$(find "$APPDIR" -type f -name "$helper" 2>/dev/null | head -n1 || true)"
  if [[ -n "$staged" ]]; then
    HELPER_DEP_ARGS+=("--deploy-deps-only=$staged")
  fi
done

export APPIMAGE_EXTRACT_AND_RUN=1
export ARCH=x86_64
export VERSION="$VER"
export DEPLOY_GTK_VERSION=3
export OUTPUT="${APP}-v${VER}-linux-amd64-${STAMP}.AppImage"
# Ubuntu 22.04+ / modern toolchains: linuxdeploy's bundled strip cannot handle .relr.dyn
export NO_STRIP=1
export PATH="$STAGE:$TOOLS_DIR:$PATH"

echo "==> linuxdeploy (GTK plugin + shared libs)"
LD_ARGS=(
  --appimage-extract-and-run
  --appdir "$APPDIR"
  --executable "$APPDIR/usr/bin/$APP"
  --desktop-file "$APPDIR/${DESKTOP_ID}.desktop"
  --plugin gtk
)
if [[ -n "$ICON_FOR_DEPLOY" && -f "$ICON_FOR_DEPLOY" ]]; then
  LD_ARGS+=(--icon-file "$ICON_FOR_DEPLOY")
fi
if ((${#EXTRA_LIBS[@]})); then
  LD_ARGS+=("${EXTRA_LIBS[@]}")
fi
if ((${#HELPER_DEP_ARGS[@]})); then
  LD_ARGS+=("${HELPER_DEP_ARGS[@]}")
fi
(
  cd "$STAGE"
  "$LINUXDEPLOY" "${LD_ARGS[@]}"
)
# linuxdeploy may replace AppRun; restore Wails-compatible launcher (+ hooks).
write_apprun

# --- relocate WebKit helper path strings inside bundled libwebkit (Wails approach) ---
relocate_webkit_helpers() {
  local appdir="$1"
  shift
  local libdir="$appdir/usr/lib"
  python3 - "$appdir" "$libdir" "$@" <<'PY'
import os, sys, glob, shutil

appdir, libdir = sys.argv[1], sys.argv[2]
helper_dirs = sys.argv[3:]
libs = sorted(glob.glob(os.path.join(libdir, "libwebkit*gtk-*.so*")))
if not libs:
    # linuxdeploy may place libs under usr/lib/<triplet>
    libs = sorted(glob.glob(os.path.join(libdir, "**", "libwebkit*gtk-*.so*"), recursive=True))
if not libs:
    raise SystemExit(f"no bundled libwebkit* under {libdir}")

contents = []
for lib in libs:
    with open(lib, "rb") as f:
        contents.append(bytearray(f.read()))

relocated = False
for directory in helper_dirs:
    abs_path = (directory + "\0").encode()
    base = os.path.basename(directory.rstrip("/"))
    rel = ("././" + base + "/").encode()
    if len(rel) >= len(abs_path):
        raise SystemExit(f"helper dir {directory!r} too short to relocate in place")
    rel = rel + (b"\0" * (len(abs_path) - len(rel)))
    referenced = False
    for buf in contents:
        if abs_path in buf:
            referenced = True
            idx = 0
            while True:
                i = buf.find(abs_path, idx)
                if i < 0:
                    break
                buf[i:i+len(abs_path)] = rel
                idx = i + len(abs_path)
    if not referenced:
        continue
    src = os.path.join(appdir, directory.lstrip("/"))
    dst = os.path.join(libdir, base)
    if os.path.abspath(src) != os.path.abspath(dst):
        if os.path.exists(dst):
            shutil.rmtree(dst) if os.path.isdir(dst) else os.remove(dst)
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.move(src, dst)
    relocated = True

if not relocated:
    raise SystemExit(f"no WebKit library references helper dirs {helper_dirs}")

for lib, buf in zip(libs, contents):
    with open(lib, "rb") as f:
        original = f.read()
    if bytes(buf) == original:
        continue
    with open(lib, "wb") as f:
        f.write(buf)
    print(f"patched {lib}")
PY
}

echo "==> relocating WebKit helper paths in bundled libraries"
relocate_webkit_helpers "$APPDIR" "${HELPER_DIRS[@]}"

OUT_NAME="$OUTPUT"
OUT="$PKG_DIR/$OUT_NAME"
rm -f "$OUT"

echo "==> linuxdeploy → AppImage $OUT_NAME"
(
  cd "$STAGE"
  export OUTPUT="$OUT_NAME"
  "$LINUXDEPLOY" --appimage-extract-and-run \
    --appdir "$APPDIR" \
    --output appimage
  if [[ -f "$OUT_NAME" ]]; then
    mv -f "$OUT_NAME" "$OUT"
  elif [[ -f "$STAGE/$OUT_NAME" ]]; then
    mv -f "$STAGE/$OUT_NAME" "$OUT"
  else
    # Fallback: whatever AppImage linuxdeploy emitted
    produced="$(find "$STAGE" -maxdepth 1 -type f -name '*.AppImage' ! -name 'linuxdeploy*' | head -n1 || true)"
    if [[ -z "$produced" ]]; then
      echo "AppImage not produced under $STAGE" >&2
      exit 1
    fi
    mv -f "$produced" "$OUT"
  fi
)

# Manifest of bundled shared objects (for docs / debugging)
MANIFEST="$PKG_DIR/.last-linux-appimage-libs.txt"
{
  echo "# Bundled libs snapshot for $OUT_NAME"
  echo "# Generated by scripts/package-linux.sh"
  find "$APPDIR/usr/lib" -type f \( -name '*.so' -o -name '*.so.*' \) 2>/dev/null | sed "s|^$APPDIR/||" | sort
} >"$MANIFEST" || true

rm -rf "$STAGE"

if [[ ! -f "$OUT" ]]; then
  echo "AppImage not produced: $OUT" >&2
  exit 1
fi
chmod +x "$OUT"
SIZE="$(wc -c <"$OUT" | tr -d ' ')"
echo "Linux AppImage: $OUT ($SIZE bytes)"
echo "Bundled: WebKitGTK(4.0)+helpers/JSC, GTK3/GDK, GLib/GObject/GIO, libsoup,"
echo "         cairo/pango/harfbuzz stack, gdk-pixbuf/GTK modules, appindicator when present."
echo "Still system-required: glibc/ld-linux, display server, GPU drivers, fonts, tray host, FUSE(optional)."
echo "Lib list snapshot: $MANIFEST"
