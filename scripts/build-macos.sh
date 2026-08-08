#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version_reader="$project_root/scripts/read-version.sh"
artifact_suffix="${TRUYEN_TEP_ARTIFACT_SUFFIX:-}"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Việc đóng gói thanh trạng thái cần chạy trên macOS (cần swiftc, sips và iconutil)." >&2
  exit 1
fi
for command_name in go swiftc sips iconutil lipo zip plutil; do
  command -v "$command_name" >/dev/null || { echo "Thiếu công cụ: $command_name" >&2; exit 1; }
done
if [[ ! -x "$version_reader" ]]; then
  echo "Thiếu công cụ đọc phiên bản: $version_reader" >&2
  exit 1
fi

case "$artifact_suffix" in
  "")
    ;;
  -*)
    case "$artifact_suffix" in
      *[!0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz.-]*)
        echo "TRUYEN_TEP_ARTIFACT_SUFFIX chỉ được chứa chữ ASCII, số, dấu chấm và dấu gạch nối." >&2
        exit 1
        ;;
    esac
    ;;
  *)
    echo "TRUYEN_TEP_ARTIFACT_SUFFIX phải để trống hoặc bắt đầu bằng dấu gạch nối." >&2
    exit 1
    ;;
esac

temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT
version_values="$temporary/version-values"
"$version_reader" > "$version_values"
IFS="$(printf '\t')" read -r version build < "$version_values"

release_root="$project_root/dist/TruyenTep-macOS-v$version$artifact_suffix"
app_path="$release_root/Truyền Tệp.app"
cd "$project_root"

rm -rf "$release_root"
mkdir -p "$app_path/Contents/MacOS" "$app_path/Contents/Resources"

for arch in arm64 amd64; do
  CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" go build -buildvcs=false -trimpath \
    -ldflags "-s -w -X truyentep/internal/truyentep.Version=$version" \
    -o "$temporary/core-$arch" "$project_root/cmd/truyentep"
done
lipo -create "$temporary/core-arm64" "$temporary/core-amd64" -output "$app_path/Contents/Resources/truyentep-core"
chmod 0755 "$app_path/Contents/Resources/truyentep-core"
swiftc "$project_root/macos/TrayApp.swift" -O -framework AppKit -o "$app_path/Contents/MacOS/truyentep"

iconset="$temporary/AppIcon.iconset"; mkdir -p "$iconset"
for size in 16 32 128 256 512; do
  sips -z "$size" "$size" assets/AppIcon-1024.png --out "$iconset/icon_${size}x${size}.png" >/dev/null
  doubled=$((size * 2)); sips -z "$doubled" "$doubled" assets/AppIcon-1024.png --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$iconset" -o "$app_path/Contents/Resources/AppIcon.icns"
cp packaging/Info.plist "$app_path/Contents/Info.plist"
plutil -replace CFBundleShortVersionString -string "$version" "$app_path/Contents/Info.plist"
plutil -replace CFBundleVersion -string "$build" "$app_path/Contents/Info.plist"
codesign --force --deep --sign - "$app_path"

mkdir -p dist
archive="$project_root/dist/TruyenTep-macOS-v$version$artifact_suffix.zip"
rm -f "$archive"
(cd dist && zip -qry "$(basename "$archive")" "$(basename "$release_root")")
echo "$archive"
