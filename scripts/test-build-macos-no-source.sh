#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version_values="$(mktemp "${TMPDIR:-/tmp}/truyentep-build-version.XXXXXX")"
"$project_root/scripts/read-version.sh" > "$version_values"
IFS="$(printf '\t')" read -r version build < "$version_values"
rm -f "$version_values"

artifact_suffix="-no-source-test"
release_root="$project_root/dist/TruyenTep-macOS-v$version$artifact_suffix"
archive="$release_root.zip"
app_path="$release_root/Truyền Tệp.app"

cleanup() {
  rm -rf "$release_root"
  rm -f "$archive"
}
trap cleanup EXIT

TRUYEN_TEP_ARTIFACT_SUFFIX="$artifact_suffix" "$project_root/scripts/build-macos.sh" >/dev/null

if [[ ! -d "$release_root" || ! -f "$archive" ]]; then
  echo "Không tìm thấy artifact mang phiên bản đã cấu hình: $release_root" >&2
  exit 1
fi

actual_core_version="$("$app_path/Contents/Resources/truyentep-core" --version)"
if [[ "$actual_core_version" != "Truyền Tệp $version" ]]; then
  echo "Core có phiên bản '$actual_core_version', mong đợi 'Truyền Tệp $version'." >&2
  exit 1
fi

actual_short_version="$(plutil -extract CFBundleShortVersionString raw -o - "$app_path/Contents/Info.plist")"
if [[ "$actual_short_version" != "$version" ]]; then
  echo "CFBundleShortVersionString là '$actual_short_version', mong đợi '$version'." >&2
  exit 1
fi

actual_build="$(plutil -extract CFBundleVersion raw -o - "$app_path/Contents/Info.plist")"
if [[ "$actual_build" != "$build" ]]; then
  echo "CFBundleVersion là '$actual_build', mong đợi '$build'." >&2
  exit 1
fi

if [[ -e "$release_root/Mã nguồn" ]]; then
  echo "Bản phát hành vẫn chứa thư mục mã nguồn." >&2
  exit 1
fi

if unzip -Z1 "$archive" | grep -Fq '/Mã nguồn/'; then
  echo "Tệp ZIP vẫn chứa mã nguồn." >&2
  exit 1
fi

for document_name in README.md HUONG-DAN-SU-DUNG.md; do
  if [[ -e "$release_root/$document_name" ]]; then
    echo "Bản phát hành vẫn chứa $document_name." >&2
    exit 1
  fi

  if unzip -Z1 "$archive" | grep -Fq "/$document_name"; then
    echo "Tệp ZIP vẫn chứa $document_name." >&2
    exit 1
  fi
done

echo "Bản phát hành macOS chỉ chứa ứng dụng, không chứa mã nguồn hoặc tài liệu dự án."
