#!/usr/bin/env bash
set -euo pipefail

error() {
  printf 'Lỗi publish: %s\n' "$1" >&2
  exit 1
}

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version_reader="$project_root/scripts/read-version.sh"
version_test="$project_root/scripts/test-version-config.sh"
build_script="$project_root/scripts/build-macos.sh"

for command_name in git gh go; do
  command -v "$command_name" >/dev/null || error "thiếu công cụ: $command_name"
done
[[ -x "$version_reader" ]] || error "thiếu công cụ đọc phiên bản: $version_reader"
[[ -x "$version_test" ]] || error "thiếu test phiên bản: $version_test"
[[ -x "$build_script" ]] || error "thiếu script build: $build_script"

cd "$project_root"
[[ -z "$(git status --porcelain)" ]] ||
  error "working tree có thay đổi chưa commit; hãy commit hoặc cất các thay đổi trước."
git remote get-url origin >/dev/null 2>&1 || error "không tìm thấy Git remote 'origin'."

temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT
"$version_reader" > "$temporary/version-values"
IFS="$(printf '\t')" read -r version build < "$temporary/version-values"
tag="v$version"

if git show-ref --verify --quiet "refs/tags/$tag"; then
  error "tag $tag đã tồn tại trong repository local; version $version trong version.json đã được dùng."
fi

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

printf 'Đang kiểm thử %s (build %s)...\n' "$tag" "$build"
go test ./...
"$version_test"

printf 'Đang build ứng dụng macOS...\n'
TRUYEN_TEP_ARTIFACT_SUFFIX= "$build_script" | tee "$temporary/build-output"
archive="$(tail -n 1 "$temporary/build-output")"
expected_archive="$project_root/dist/TruyenTep-macOS-v$version.zip"
[[ "$archive" == "$expected_archive" ]] ||
  error "script build trả về artifact không mong đợi: $archive"
[[ -f "$archive" ]] || error "không tìm thấy artifact sau khi build: $archive"

printf 'Đang tạo và push tag %s...\n' "$tag"
git tag -a "$tag" -m "Release $tag"
git push origin "$tag"

printf 'Đang tạo GitHub Release và upload %s...\n' "$(basename "$archive")"
if gh release create "$tag" "$archive" --verify-tag --generate-notes --title "$tag"; then
  printf 'Đã publish thành công %s với artifact:\n%s\n' "$tag" "$archive"
else
  printf 'Lỗi publish: tag %s đã được push nhưng chưa tạo được GitHub Release.\n' "$tag" >&2
  printf 'Giữ nguyên tag để tránh xóa dữ liệu remote. Thử lại bằng lệnh:\n  ' >&2
  printf 'gh release create %q %q --verify-tag --generate-notes --title %q\n' \
    "$tag" "$archive" "$tag" >&2
  exit 1
fi
