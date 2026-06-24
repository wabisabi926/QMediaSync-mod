#!/usr/bin/env bash
set -euo pipefail

# 从 git 提交日志自动生成指定版本的 changelog（基于 git-cliff + 仓库根目录的 cliff.toml）。
#
# 作用：
#   1) 生成本版本的发布说明，写入 .changes/<tag>.md（release 工作流会读取它作为 GitHub Release 正文）
#   2) 把本版本段落插入 CHANGELOG.md 顶部（# Changelog 标题之后），保留历史内容
#
# 用法：
#   scripts/release/gen-changelog.sh v0.14.24
#
# 依赖：git-cliff（安装：https://git-cliff.org/docs/installation/，例如 `cargo install git-cliff`
#       / `brew install git-cliff` / `npm i -g git-cliff`）

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

TAG="${1:-}"
if [ -z "$TAG" ]; then
  echo "用法: $0 <tag>，例如 $0 v0.14.24" >&2
  exit 1
fi

if ! command -v git-cliff >/dev/null 2>&1; then
  echo "未找到 git-cliff，请先安装：https://git-cliff.org/docs/installation/" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

CHANGELOG="CHANGELOG.md"
NOTES=".changes/${TAG}.md"

ensure_new_changelog_version "$TAG" "$CHANGELOG" ".changes"

mkdir -p .changes

# 1) 生成本版本发布说明（仅上一个 tag 至今的未发布区间，按 cliff.toml 分组），不含 # Changelog 头
git-cliff --unreleased --tag "$TAG" --strip header -o "$NOTES"

if [ ! -s "$NOTES" ]; then
  echo "警告：自上一个 tag 以来没有符合 conventional commits 规范的提交，$NOTES 为空。" >&2
fi

# 2) 将本版本段落插入 CHANGELOG.md 顶部（# Changelog 之后），保留历史内容
tmp="$(mktemp)"
inserted=0
while IFS= read -r line || [ -n "$line" ]; do
  printf '%s\n' "$line" >> "$tmp"
  if [ "$inserted" -eq 0 ] && printf '%s' "$line" | grep -q '^# Changelog'; then
    printf '\n' >> "$tmp"
    cat "$NOTES" >> "$tmp"
    inserted=1
  fi
done < "$CHANGELOG"

if [ "$inserted" -eq 0 ]; then
  # CHANGELOG.md 里没有 # Changelog 标题，则重建一个并把新内容放最前
  { printf '# Changelog\n\n'; cat "$NOTES"; printf '\n'; cat "$CHANGELOG"; } > "$tmp"
fi

mv "$tmp" "$CHANGELOG"

echo "已生成 $NOTES 并更新 $CHANGELOG"
echo "请检查内容后提交：git add $CHANGELOG $NOTES && git commit -m \"chore: release $TAG\""
