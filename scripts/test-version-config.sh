#!/usr/bin/env bash

set -u

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
READER="$SCRIPT_DIR/read-version.sh"
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/truyentep-version-test.XXXXXX") || exit 1

cleanup() {
  rm -rf -- "$TMP_DIR"
}
trap cleanup EXIT HUP INT TERM

passed=0
failed=0

pass() {
  printf 'PASS: %s\n' "$1"
  passed=$((passed + 1))
}

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  failed=$((failed + 1))
}

assert_output() {
  name=$1
  expected=$2
  shift 2

  "$@" >"$TMP_DIR/stdout" 2>"$TMP_DIR/stderr"
  status=$?
  printf '%s\n' "$expected" >"$TMP_DIR/expected"
  if [ "$status" -eq 0 ] &&
    cmp -s "$TMP_DIR/expected" "$TMP_DIR/stdout" &&
    [ ! -s "$TMP_DIR/stderr" ]; then
    pass "$name"
  else
    fail "$name (status=$status, output=$(tr '\n' ' ' < "$TMP_DIR/stdout"), stderr=$(tr '\n' ' ' < "$TMP_DIR/stderr"))"
  fi
}

assert_rejected() {
  name=$1
  path=$2

  output=$("$READER" "$path" 2>"$TMP_DIR/stderr")
  status=$?
  if [ "$status" -ne 0 ] && [ -s "$TMP_DIR/stderr" ] && [ -z "$output" ]; then
    pass "$name"
  else
    fail "$name (status=$status, output=$(printf '%s' "$output"), stderr=$(tr '\n' ' ' < "$TMP_DIR/stderr"))"
  fi
}

write_fixture() {
  path=$1
  contents=$2
  printf '%s\n' "$contents" > "$path"
}

assert_output "real config defaults to repository version.json" "$(printf '0.3.0\t3')" "$READER"

valid_path="$TMP_DIR/path with spaces/valid.json"
mkdir -p "$(dirname -- "$valid_path")"
write_fixture "$valid_path" '{"version":"1.2.3-beta.1+build.7","build":42}'
assert_output "valid SemVer prerelease and metadata fixture" "$(printf '1.2.3-beta.1+build.7\t42')" "$READER" "$valid_path"

assert_rejected "missing config file" "$TMP_DIR/missing.json"

malformed="$TMP_DIR/malformed.json"
write_fixture "$malformed" '{"version":"1.2.3","build":'
assert_rejected "malformed JSON" "$malformed"

missing_version="$TMP_DIR/missing-version.json"
write_fixture "$missing_version" '{"build":3}'
assert_rejected "missing version field" "$missing_version"

missing_build="$TMP_DIR/missing-build.json"
write_fixture "$missing_build" '{"version":"1.2.3"}'
assert_rejected "missing build field" "$missing_build"

for invalid_version in \
  '1.2' \
  'v1.2.3' \
  '1.2.3 unsafe' \
  '1.2.3/../../bad' \
  '1.2.3-'; do
  fixture="$TMP_DIR/invalid-version-$failed-$passed.json"
  write_fixture "$fixture" "{\"version\":\"$invalid_version\",\"build\":3}"
  assert_rejected "invalid version: $invalid_version" "$fixture"
done

for invalid_build in '0' '-1' '"abc"' '1.5' '"3"'; do
  fixture="$TMP_DIR/invalid-build-$failed-$passed.json"
  write_fixture "$fixture" "{\"version\":\"1.2.3\",\"build\":$invalid_build}"
  assert_rejected "invalid build: $invalid_build" "$fixture"
done

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[ "$failed" -eq 0 ]
