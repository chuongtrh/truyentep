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

json_envelope=$(LC_ALL=C tr -d '[:space:]' < "$CONFIG_PATH") ||
  error "không đọc được file config: $CONFIG_PATH"
case "$json_envelope" in
  \{*\})
    ;;
  *)
    error "config phải là một JSON object."
    ;;
esac

"$PLUTIL" -convert json -o /dev/null -- "$CONFIG_PATH" >/dev/null 2>&1 ||
  error "JSON không hợp lệ: $CONFIG_PATH"

version_with_sentinel=$(
  "$PLUTIL" -extract version raw -expect string -n -o - -- "$CONFIG_PATH" 2>/dev/null
  extract_status=$?
  printf '\034'
  exit "$extract_status"
) || error "thiếu trường version dạng chuỗi."
version=${version_with_sentinel%?}

build=$("$PLUTIL" -extract build raw -expect integer -n -o - -- "$CONFIG_PATH" 2>/dev/null) ||
  error "thiếu trường build dạng số nguyên."

version_pattern='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
case "$version" in
  *[!0-9.]*)
    error "version chứa ký tự không an toàn."
    ;;
esac

[[ "$version" =~ $version_pattern ]] ||
  error "version phải gồm đúng ba số nguyên không có số 0 thừa: X.Y.Z."

case "$build" in
  ''|*[!0-9]*|0)
    error "build phải là số nguyên dương từ 1 đến 9999."
    ;;
esac

if [ "$build" -gt 9999 ]; then
  error "build phải là số nguyên dương từ 1 đến 9999."
fi

printf '%s\t%s\n' "$version" "$build"
