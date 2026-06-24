#!/usr/bin/env bash

escape_regex() {
  printf '%s' "$1" | sed 's/[][(){}.^$*+?|\\]/\\&/g'
}

ensure_new_changelog_version() {
  local tag="$1"
  local changelog="$2"
  local changes_dir="$3"
  local notes="${changes_dir}/${tag}.md"
  local escaped_tag

  if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
    echo "版本 ${tag} 的 git tag 已存在" >&2
    return 1
  fi

  if [ -e "$notes" ]; then
    echo "版本 ${tag} 的发布说明已存在: ${notes}" >&2
    return 1
  fi

  escaped_tag="$(escape_regex "$tag")"
  if [ -f "$changelog" ] && grep -Eq "^## \\[?${escaped_tag}\\]?([[:space:]]|$)" "$changelog"; then
    echo "版本 ${tag} 已存在于 ${changelog}" >&2
    return 1
  fi
}
