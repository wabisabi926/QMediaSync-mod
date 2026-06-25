# shellcheck shell=bash

die() {
  echo "错误: $*" >&2
  exit 1
}

require_command() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    die "未找到命令 ${name}"
  fi
}

escape_regex() {
  printf '%s' "$1" | sed 's/[][(){}.^$*+?|\\]/\\&/g'
}

validate_release_tag() {
  local tag="$1"
  if [[ ! "$tag" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    die "tag 格式必须是 v<major>.<minor>.<patch>，例如 v0.15.3；大版本请使用 v16.0.0"
  fi
  TARGET_MAJOR="${BASH_REMATCH[1]}"
  TARGET_MINOR="${BASH_REMATCH[2]}"
  TARGET_PATCH="${BASH_REMATCH[3]}"
}

validate_release_input() {
  local value="$1"
  case "$value" in
    patch|minor|major)
      return
      ;;
  esac

  validate_release_tag "$value"
}

parse_version_parts() {
  local tag="$1"
  if [[ ! "$tag" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    return 1
  fi
  printf '%s %s %s\n' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}"
}

compare_versions() {
  local left_major="$1"
  local left_minor="$2"
  local left_patch="$3"
  local right_major="$4"
  local right_minor="$5"
  local right_patch="$6"

  if (( left_major > right_major )); then
    printf '1\n'
    return
  fi
  if (( left_major < right_major )); then
    printf -- '-1\n'
    return
  fi
  if (( left_minor > right_minor )); then
    printf '1\n'
    return
  fi
  if (( left_minor < right_minor )); then
    printf -- '-1\n'
    return
  fi
  if (( left_patch > right_patch )); then
    printf '1\n'
    return
  fi
  if (( left_patch < right_patch )); then
    printf -- '-1\n'
    return
  fi
  printf '0\n'
}

latest_semver_tag() {
  local tag
  while IFS= read -r tag; do
    if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      printf '%s\n' "$tag"
      return
    fi
  done < <(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname)
}

ensure_new_release_version() {
  local tag="$1"
  local changelog="$2"
  local changes_dir="$3"
  local notes="${changes_dir}/${tag}.md"
  local escaped_tag

  validate_release_tag "$tag"

  if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
    die "版本 ${tag} 的 git tag 已存在"
  fi

  if [ -e "$notes" ]; then
    die "版本 ${tag} 的发布说明已存在: ${notes}"
  fi

  escaped_tag="$(escape_regex "$tag")"
  if [ -f "$changelog" ] && grep -Eq "^## \\[?${escaped_tag}\\]?([[:space:]]|$)" "$changelog"; then
    die "版本 ${tag} 已存在于 ${changelog}"
  fi
}
