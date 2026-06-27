#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

usage() {
  cat <<'EOF'
用法:
  scripts/release/release.sh <tag|patch|minor|major>

示例:
  scripts/release/release.sh v0.15.3
  scripts/release/release.sh patch
  scripts/release/release.sh minor

说明:
  使用 patch/minor/major 时，脚本会先推导版本并提示确认。
  直接回车使用推导版本，也可以输入 v<major>.<minor>.<patch> 手动覆盖。

流程:
  1. 检查 tag 格式和版本递增关系
  2. 同步 main，并把 dev 快进合入 main
  3. 生成 CHANGELOG.md 和 .changes/<tag>.md
  4. 人工确认 changelog diff
  5. 提交 release commit 到 main
  6. 创建 annotated tag: git tag -a <tag> -m "Release <tag>"
  7. 推送 main 和 tag，触发 release workflow
  8. 将 release commit 快进同步回 dev 并推送 dev
EOF
}

derive_release_tag() {
  local bump="$1"
  local current_tag
  local current_major
  local current_minor
  local current_patch

  current_tag="$(latest_semver_tag)"
  if [ -z "$current_tag" ]; then
    die "无法根据最新版本推导 ${bump} 版本：仓库没有 v<major>.<minor>.<patch> 格式的 tag"
  fi

  read -r current_major current_minor current_patch < <(parse_version_parts "$current_tag")
  case "$bump" in
    patch)
      TARGET_MAJOR="$current_major"
      TARGET_MINOR="$current_minor"
      TARGET_PATCH="$((current_patch + 1))"
      ;;
    minor)
      TARGET_MAJOR="$current_major"
      TARGET_MINOR="$((current_minor + 1))"
      TARGET_PATCH="0"
      ;;
    major)
      TARGET_MAJOR="$((current_major + 1))"
      TARGET_MINOR="0"
      TARGET_PATCH="0"
      ;;
    *)
      die "未知版本推导参数 ${bump}"
      ;;
  esac

  TAG="v${TARGET_MAJOR}.${TARGET_MINOR}.${TARGET_PATCH}"
  echo "根据当前最新版本 ${current_tag} 推导发布版本: ${TAG}"
}

confirm_or_override_derived_tag() {
  local answer

  printf '发布版本 [%s]，直接回车确认，或输入版本覆盖: ' "$TAG"
  read -r answer || answer=""
  if [ -z "$answer" ]; then
    validate_release_tag "$TAG"
    return
  fi

  validate_release_tag "$answer"
  TAG="$answer"
}

confirm_major_release() {
  local current_tag="$1"
  local target_tag="$2"

  echo
  echo "检测到大版本发布: ${current_tag} -> ${target_tag}"
  echo "大版本发布通常表示兼容性或发布节奏变化，需要额外确认。"
  printf '输入 major yes 继续大版本发布: '
  read -r answer
  if [ "$answer" != "major yes" ]; then
    die "已取消大版本发布"
  fi
}

confirm_minor_release() {
  local current_tag="$1"
  local target_tag="$2"

  echo
  echo "检测到 minor 版本发布: ${current_tag} -> ${target_tag}"
  echo "minor 版本发布通常表示功能级更新，需要额外确认。"
  printf '输入 minor yes 继续 minor 版本发布: '
  read -r answer
  if [ "$answer" != "minor yes" ]; then
    die "已取消 minor 版本发布"
  fi
}

resolve_release_tag() {
  local value="$1"

  case "$value" in
    patch|minor|major)
      derive_release_tag "$value"
      confirm_or_override_derived_tag
      ;;
    *)
      validate_release_tag "$value"
      TAG="$value"
      ;;
  esac
}

ensure_tag_newer_than_current() {
  local tag="$1"
  local current_tag
  local current_major
  local current_minor
  local current_patch
  local comparison

  current_tag="$(latest_semver_tag)"
  if [ -z "$current_tag" ]; then
    return
  fi

  read -r current_major current_minor current_patch < <(parse_version_parts "$current_tag")
  comparison="$(compare_versions "$TARGET_MAJOR" "$TARGET_MINOR" "$TARGET_PATCH" "$current_major" "$current_minor" "$current_patch")"
  if [ "$comparison" -le 0 ]; then
    die "tag ${tag} 必须大于当前最新版本 ${current_tag}"
  fi

  if (( TARGET_MAJOR > current_major )); then
    confirm_major_release "$current_tag" "$tag"
  elif (( TARGET_MINOR > current_minor )); then
    confirm_minor_release "$current_tag" "$tag"
  fi
}

ensure_clean_worktree() {
  if ! git diff --quiet || ! git diff --cached --quiet; then
    die "工作区不干净，请先提交或暂存现有改动"
  fi
}

ensure_branch_exists() {
  local branch="$1"
  if ! git show-ref --verify --quiet "refs/heads/${branch}"; then
    die "本地分支 ${branch} 不存在"
  fi
}

ensure_remote_branch_exists() {
  local remote="$1"
  local branch="$2"
  if ! git show-ref --verify --quiet "refs/remotes/${remote}/${branch}"; then
    die "远端分支 ${remote}/${branch} 不存在"
  fi
}

ensure_branch_contains_remote() {
  local remote="$1"
  local branch="$2"
  if ! git merge-base --is-ancestor "${remote}/${branch}" "$branch"; then
    die "本地 ${branch} 不包含 ${remote}/${branch}，请先同步或处理分叉"
  fi
}

ensure_no_remote_tag() {
  local remote="$1"
  local tag="$2"
  if git ls-remote --exit-code --tags "$remote" "refs/tags/${tag}" >/dev/null 2>&1; then
    die "远端 tag ${tag} 已存在"
  fi
}

confirm_release() {
  local tag="$1"
  echo
  echo "将执行以下外部动作:"
  echo "  - 提交 chore: release ${tag} 到 main"
  echo "  - 创建 annotated tag ${tag}"
  echo "  - 推送 main"
  echo "  - 推送 tag ${tag} 触发 release workflow"
  echo "  - 将 release commit 同步回 dev 并推送 dev"
  echo
  printf '输入 yes 继续发布: '
  read -r answer
  if [ "$answer" != "yes" ]; then
    echo "已停止。生成的 changelog 改动保留在工作区，可检查后手动处理。"
    exit 1
  fi
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  if [ "$#" -ne 1 ]; then
    usage >&2
    exit 1
  fi

  RELEASE_INPUT="$1"
  TAG="$RELEASE_INPUT"
  REMOTE="origin"
  MAIN_BRANCH="main"
  DEV_BRANCH="dev"
  TARGET_MAJOR=""
  TARGET_MINOR=""
  TARGET_PATCH=""

  validate_release_input "$RELEASE_INPUT"

  require_command git
  require_command git-cliff

  cd "$ROOT"

  git fetch "$REMOTE" --tags
  resolve_release_tag "$RELEASE_INPUT"
  ensure_tag_newer_than_current "$TAG"

  ensure_clean_worktree
  ensure_branch_exists "$MAIN_BRANCH"
  ensure_branch_exists "$DEV_BRANCH"

  ensure_remote_branch_exists "$REMOTE" "$MAIN_BRANCH"
  ensure_remote_branch_exists "$REMOTE" "$DEV_BRANCH"
  ensure_branch_contains_remote "$REMOTE" "$DEV_BRANCH"
  ensure_no_remote_tag "$REMOTE" "$TAG"

  git checkout "$MAIN_BRANCH"
  git pull --ff-only "$REMOTE" "$MAIN_BRANCH"
  git merge --ff-only "$DEV_BRANCH"

  ensure_no_remote_tag "$REMOTE" "$TAG"

  scripts/release/gen-changelog.sh "$TAG"

  echo
  echo "生成的发布说明变更:"
  git diff -- "CHANGELOG.md" ".changes/${TAG}.md"

  confirm_release "$TAG"

  git add "CHANGELOG.md" ".changes/${TAG}.md"
  git commit -m "chore: release ${TAG}"
  git tag -a "$TAG" -m "Release $TAG"

  git push "$REMOTE" "$MAIN_BRANCH"
  git push "$REMOTE" "$TAG"

  git checkout "$DEV_BRANCH"
  git pull --ff-only "$REMOTE" "$DEV_BRANCH"
  git merge --ff-only "$MAIN_BRANCH"
  git push "$REMOTE" "$DEV_BRANCH"

  cat <<EOF

发布流程已提交并触发:
  main: 已推送到 ${REMOTE}/${MAIN_BRANCH}
  tag:  已推送 ${TAG}
  dev:  已同步 release commit 并推送到 ${REMOTE}/${DEV_BRANCH}

release workflow 会由 tag 自动触发，请到 GitHub Actions 页面查看执行结果。
EOF
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
