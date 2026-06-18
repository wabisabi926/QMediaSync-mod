#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <tag> <arch> <binary> <web_statics> <out_dir>" >&2
}

if [ "$#" -ne 5 ]; then
  usage
  exit 1
fi

TAG="$1"
TARGET_ARCH="$2"
BINARY_PATH="$3"
WEB_STATICS_DIR="$4"
OUT_DIR="$5"

case "$TARGET_ARCH" in
  amd64|arm64) ;;
  *)
    echo "Unsupported FNOS arch: $TARGET_ARCH" >&2
    exit 1
    ;;
esac

if ! command -v fnpack >/dev/null 2>&1; then
  if [ "${REQUIRE_FNPACK:-0}" = "1" ]; then
    echo "fnpack is required but was not found" >&2
    exit 1
  fi
  echo "fnpack not found, skipping FNOS package for ${TARGET_ARCH}"
  exit 0
fi

if [ ! -f "$BINARY_PATH" ]; then
  echo "Binary not found: $BINARY_PATH" >&2
  exit 1
fi

if [ ! -d "$WEB_STATICS_DIR" ]; then
  echo "web_statics directory not found: $WEB_STATICS_DIR" >&2
  exit 1
fi

FNOS_DIR="backend/FNOS/qmediasync-${TARGET_ARCH}"
APP_DIR="$FNOS_DIR/app"
MANIFEST="$FNOS_DIR/manifest"
FNOS_VERSION="${TAG#v}"

if [ ! -d "$FNOS_DIR" ]; then
  echo "FNOS directory not found: $FNOS_DIR" >&2
  exit 1
fi

mkdir -p "$APP_DIR" "$OUT_DIR"
rm -f "$APP_DIR/QMediaSync"
rm -rf "$APP_DIR/web_statics"

if [ -f "$MANIFEST" ]; then
  sed -i "s/^version[[:space:]]*=.*/version = ${FNOS_VERSION}/g" "$MANIFEST"
fi

cp "$BINARY_PATH" "$APP_DIR/QMediaSync"
chmod +x "$APP_DIR/QMediaSync"
cp -R "$WEB_STATICS_DIR" "$APP_DIR/web_statics"

if [ -f "backend/assets/db_config.html" ]; then
  cp "backend/assets/db_config.html" "$APP_DIR/web_statics/"
fi

(cd "$FNOS_DIR" && fnpack build)

if [ ! -f "$FNOS_DIR/qmediasync.fpk" ]; then
  echo "FNOS package was not generated: $FNOS_DIR/qmediasync.fpk" >&2
  exit 1
fi

cp "$FNOS_DIR/qmediasync.fpk" "$OUT_DIR/QMediaSync_${TARGET_ARCH}.fpk"
echo "Created $OUT_DIR/QMediaSync_${TARGET_ARCH}.fpk"
