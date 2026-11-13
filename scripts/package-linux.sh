#!/usr/bin/env bash
# Build Linux amd64 Wails app + tarball. Run on a Linux desktop host (or CI).
# Usage (from repo root):
#   bash scripts/package-linux.sh
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

echo "==> wails build (linux/amd64 commit=$COMMIT)"
wails build -platform linux/amd64 -ldflags "$LD"

APP="GoogleTranslateHostsUpdater"
BIN="$BIN_DIR/$APP"
if [[ ! -f "$BIN" ]]; then
  # Some Wails versions emit lowercase or nested paths.
  BIN="$(find "$BIN_DIR" -type f -name "$APP" | head -n1 || true)"
fi
if [[ -z "${BIN}" || ! -f "$BIN" ]]; then
  echo "missing binary under $BIN_DIR" >&2
  exit 1
fi

STAMP="$(date +%Y%m%d-%H%M)"
STAGE="$DIST_DIR/stage-linux-amd64"
rm -rf "$STAGE"
mkdir -p "$STAGE"
cp "$BIN" "$STAGE/$APP"
chmod +x "$STAGE/$APP"
cat >"$STAGE/README.txt" <<EOF
谷歌翻译 Hosts 更新器 (linux/amd64)
运行: ./$APP
需要: 现代桌面环境 + WebKitGTK（发行版通常已有浏览器依赖）
写 /etc/hosts 时会通过 pkexec/sudo 提权。
详见 docs/user-flow.md
EOF

TAR="$DIST_DIR/${APP}-v${VER}-linux-amd64-${STAMP}.tar.gz"
tar -C "$STAGE" -czf "$TAR" .
rm -rf "$STAGE"
echo "Linux package: $TAR"
