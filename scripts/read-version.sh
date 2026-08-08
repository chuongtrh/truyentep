#!/usr/bin/env bash

set -u

error() {
  printf 'Lỗi phiên bản: %s\n' "$1" >&2
  exit 1
}

if [ "$#" -gt 1 ]; then
  error "chỉ chấp nhận tối đa một đường dẫn config."
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd) ||
  error "không xác định được thư mục script."
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd) ||
  error "không xác định được thư mục dự án."
CONFIG_PATH=${1:-"$ROOT_DIR/version.json"}

if [ ! -f "$CONFIG_PATH" ]; then
  error "không tìm thấy file config: $CONFIG_PATH"
fi

if [ -x /usr/bin/plutil ]; then
  PLUTIL=/usr/bin/plutil
else
  PLUTIL=$(command -v plutil 2>/dev/null) ||
    error "không tìm thấy lệnh plutil."
fi

"$PLUTIL" -convert json -o /dev/null -- "$CONFIG_PATH" >/dev/null 2>&1 ||
  error "JSON không hợp lệ: $CONFIG_PATH"

version=$("$PLUTIL" -extract version raw -expect string -o - -- "$CONFIG_PATH" 2>/dev/null) ||
  error "thiếu trường version dạng chuỗi."
build=$("$PLUTIL" -extract build raw -expect integer -o - -- "$CONFIG_PATH" 2>/dev/null) ||
  error "thiếu trường build dạng số nguyên."

version_pattern='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?(\+[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$'
case "$version" in
  *[!0-9A-Za-z.+-]*)
    error "version chứa ký tự không an toàn."
    ;;
esac

[[ "$version" =~ $version_pattern ]] ||
  error "version không đúng định dạng X.Y.Z[-prerelease][+metadata]."

case "$build" in
  ''|*[!0-9]*|0)
    error "build phải là số nguyên dương."
    ;;
esac

printf '%s\t%s\n' "$version" "$build"
