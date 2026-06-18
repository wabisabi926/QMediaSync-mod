#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <tag> <release_notes_file>" >&2
}

if [ "$#" -ne 2 ]; then
  usage
  exit 1
fi

TAG="$1"
RELEASE_NOTES_FILE="$2"

if [ ! -f "$RELEASE_NOTES_FILE" ]; then
  echo "Release notes file not found: $RELEASE_NOTES_FILE" >&2
  exit 1
fi

MESSAGE="$(cat "$RELEASE_NOTES_FILE")"

if [ -n "${TELEGRAM_BOT_TOKEN:-}" ] && [ -n "${TELEGRAM_CHAT_ID:-}" ]; then
  TELEGRAM_PAYLOAD="$(mktemp)"
  python3 - "$TELEGRAM_CHAT_ID" "$MESSAGE" > "$TELEGRAM_PAYLOAD" <<'PY'
import json
import sys

chat_id, message = sys.argv[1:3]
print(json.dumps({
    "chat_id": chat_id,
    "text": message,
    "parse_mode": "Markdown",
}, ensure_ascii=False))
PY

  if ! curl -fsS -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
    -H "Content-Type: application/json" \
    --data-binary "@${TELEGRAM_PAYLOAD}" \
    >/dev/null; then
    echo "Telegram notification failed for ${TAG}" >&2
  fi
  rm -f "$TELEGRAM_PAYLOAD"
else
  echo "Telegram secrets are not set, skipping Telegram notification"
fi

if [ -n "${MEOW_API_URL:-}" ]; then
  if ! curl -fsS -X POST "$MEOW_API_URL" \
    -H "Content-Type: text/plain" \
    --data-binary "@${RELEASE_NOTES_FILE}" \
    >/dev/null; then
    echo "MeoW notification failed for ${TAG}" >&2
  fi
else
  echo "MEOW_API_URL is not set, skipping MeoW notification"
fi
