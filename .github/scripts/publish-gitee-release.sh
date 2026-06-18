#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <tag> <release_notes_file> <asset>..." >&2
}

if [ "$#" -lt 2 ]; then
  usage
  exit 1
fi

TAG="$1"
RELEASE_NOTES_FILE="$2"
shift 2

GITEE_ACCESS_TOKEN="${GITEE_ACCESS_TOKEN:-}"
GITEE_REPO="${GITEE_REPO:-qicfan/qmediasync}"
GITEE_API_BASE="${GITEE_API_BASE:-https://gitee.com/api/v5}"
GITEE_TARGET_COMMITISH="${GITEE_TARGET_COMMITISH:-main}"

if [ -z "$GITEE_ACCESS_TOKEN" ]; then
  echo "GITEE_ACCESS_TOKEN is not set, skipping Gitee release"
  exit 0
fi

if [ ! -f "$RELEASE_NOTES_FILE" ]; then
  echo "Release notes file not found: $RELEASE_NOTES_FILE" >&2
  exit 1
fi

PAYLOAD_FILE="$(mktemp)"
RESPONSE_FILE="$(mktemp)"
trap 'rm -f "$PAYLOAD_FILE" "$RESPONSE_FILE"' EXIT

python3 - "$TAG" "$RELEASE_NOTES_FILE" "$GITEE_ACCESS_TOKEN" "$GITEE_TARGET_COMMITISH" > "$PAYLOAD_FILE" <<'PY'
import json
import sys

tag, notes_file, token, target = sys.argv[1:5]
with open(notes_file, "r", encoding="utf-8") as fp:
    body = fp.read()

print(json.dumps({
    "access_token": token,
    "tag_name": tag,
    "name": f"Release {tag}",
    "body": body,
    "target_commitish": target,
    "prerelease": False,
}, ensure_ascii=False))
PY

curl -sS -X POST \
  "${GITEE_API_BASE}/repos/${GITEE_REPO}/releases" \
  -H "Content-Type: application/json" \
  --data-binary "@${PAYLOAD_FILE}" \
  -o "$RESPONSE_FILE"

RELEASE_ID="$(python3 - "$RESPONSE_FILE" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fp:
    data = json.load(fp)
print(data.get("id", ""))
PY
)"

if [ -z "$RELEASE_ID" ]; then
  echo "Failed to create Gitee release. Response:" >&2
  cat "$RESPONSE_FILE" >&2
  exit 1
fi

echo "Created Gitee release ${TAG} with id ${RELEASE_ID}"

for asset in "$@"; do
  if [ ! -f "$asset" ]; then
    continue
  fi

  echo "Uploading $asset to Gitee release ${RELEASE_ID}"
  curl -sS -X POST \
    "${GITEE_API_BASE}/repos/${GITEE_REPO}/releases/${RELEASE_ID}/attach_files" \
    -F "access_token=${GITEE_ACCESS_TOKEN}" \
    -F "file=@${asset}" \
    >/dev/null
done
