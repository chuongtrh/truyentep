#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="0.2.0-no-source-test"
release_root="$project_root/dist/TruyenTep-macOS-v$version"
archive="$release_root.zip"

cleanup() {
  rm -rf "$release_root"
  rm -f "$archive"
}
trap cleanup EXIT

TRUYEN_TEP_VERSION="$version" "$project_root/scripts/build-macos.sh" >/dev/null

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
