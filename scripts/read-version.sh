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

GO_COMMAND=$(command -v go 2>/dev/null) ||
  error "không tìm thấy lệnh go; Go là dependency bắt buộc để đọc version config."

TRUYENTEP_VERSION_CONFIG=$CONFIG_PATH
export TRUYENTEP_VERSION_CONFIG
exec "$GO_COMMAND" run "$SCRIPT_DIR/read-version.go"
