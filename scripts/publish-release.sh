#!/usr/bin/env bash
set -euo pipefail

error() {
  printf 'Lỗi publish: %s\n' "$1" >&2
  exit 1
}

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version_reader="$project_root/scripts/read-version.sh"

for command_name in git gh go; do
  command -v "$command_name" >/dev/null || error "thiếu công cụ: $command_name"
done
[[ -x "$version_reader" ]] || error "thiếu công cụ đọc phiên bản: $version_reader"

cd "$project_root"
[[ -z "$(git status --porcelain)" ]] ||
  error "working tree có thay đổi chưa commit; hãy commit hoặc cất các thay đổi trước."
git remote get-url origin >/dev/null 2>&1 || error "không tìm thấy Git remote 'origin'."

temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT
"$version_reader" > "$temporary/version-values"
IFS="$(printf '\t')" read -r version build < "$temporary/version-values"
tag="v$version"

if git ls-remote --exit-code --tags origin "refs/tags/$tag" > "$temporary/remote-tag" 2>/dev/null; then
  error "tag $tag đã tồn tại trên remote origin; version $version trong version.json đã được publish."
else
  remote_tag_status=$?
  [[ "$remote_tag_status" -eq 2 ]] || error "không thể kiểm tra tag $tag trên remote origin."
fi

upstream="$(git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null)" ||
  error "branch hiện tại chưa có upstream."
case "$upstream" in
  origin/*) remote_branch="${upstream#origin/}" ;;
  *) error "upstream phải thuộc remote origin, hiện tại là $upstream." ;;
esac

remote_head="$(git ls-remote --heads origin "refs/heads/$remote_branch" | awk 'NR == 1 { print $1 }')"
[[ -n "$remote_head" ]] || error "không tìm thấy branch $remote_branch trên remote origin."
local_head="$(git rev-parse HEAD)"
[[ "$local_head" == "$remote_head" ]] ||
  error "commit hiện tại chưa đồng bộ với origin/$remote_branch; hãy push hoặc cập nhật branch trước."

gh auth status >/dev/null 2>&1 || error "GitHub CLI chưa đăng nhập; hãy chạy 'gh auth login'."

printf 'Preflight hợp lệ cho %s (build %s).\n' "$tag" "$build"
