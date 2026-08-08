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

assert_structurally_valid_output() {
  name=$1
  shift

  "$@" >"$TMP_DIR/stdout" 2>"$TMP_DIR/stderr"
  status=$?
  output=$(<"$TMP_DIR/stdout")
  canonical_pattern='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)	([1-9][0-9]{0,3})$'
  printf '%s\n' "$output" >"$TMP_DIR/expected"
  if [ "$status" -eq 0 ] &&
    [ ! -s "$TMP_DIR/stderr" ] &&
    cmp -s "$TMP_DIR/expected" "$TMP_DIR/stdout" &&
    [[ "$output" =~ $canonical_pattern ]]; then
    pass "$name"
  else
    fail "$name (status=$status, output=$(tr '\n' ' ' < "$TMP_DIR/stdout"), stderr=$(tr '\n' ' ' < "$TMP_DIR/stderr"))"
  fi
}

assert_rejected() {
  name=$1
  path=$2

  "$READER" "$path" >"$TMP_DIR/rejected-stdout" 2>"$TMP_DIR/stderr"
  status=$?
  if [ "$status" -ne 0 ] &&
    [ -s "$TMP_DIR/stderr" ] &&
    [ ! -s "$TMP_DIR/rejected-stdout" ]; then
    pass "$name"
  else
    fail "$name (status=$status, stdout-bytes=$(wc -c < "$TMP_DIR/rejected-stdout" | tr -d ' '), stderr=$(tr '\n' ' ' < "$TMP_DIR/stderr"))"
  fi
}

write_fixture() {
  path=$1
  contents=$2
  printf '%s\n' "$contents" > "$path"
}

assert_structurally_valid_output "real config defaults to a structurally valid version.json" "$READER"

alternate_root="$TMP_DIR/alternate-version-root"
mkdir -p "$alternate_root/scripts"
cp "$READER" "$SCRIPT_DIR/read-version.go" "$alternate_root/scripts/"
write_fixture "$alternate_root/version.json" '{"version":"9.8.7","build":321}'
assert_output "default reader follows version.json as the single source" "$(printf '9.8.7\t321')" "$alternate_root/scripts/read-version.sh"

valid_path="$TMP_DIR/path with spaces/valid.json"
mkdir -p "$(dirname -- "$valid_path")"
write_fixture "$valid_path" '{"version":"12.34.56","build":42}'
assert_output "valid canonical macOS version fixture" "$(printf '12.34.56\t42')" "$READER" "$valid_path"

assert_rejected "missing config file" "$TMP_DIR/missing.json"

malformed="$TMP_DIR/malformed.json"
write_fixture "$malformed" '{"version":"1.2.3","build":'
assert_rejected "malformed JSON" "$malformed"

comment_json="$TMP_DIR/comment.json"
write_fixture "$comment_json" '{/*comment*/"version":"1.2.3","build":3}'
assert_rejected "JSON comments are forbidden" "$comment_json"

trailing_comma_json="$TMP_DIR/trailing-comma.json"
write_fixture "$trailing_comma_json" '{"version":"1.2.3","build":3,}'
assert_rejected "JSON trailing comma is forbidden" "$trailing_comma_json"

unary_plus_json="$TMP_DIR/unary-plus.json"
write_fixture "$unary_plus_json" '{"version":"1.2.3","build":+3}'
assert_rejected "JSON unary plus is forbidden" "$unary_plus_json"

unknown_field_json="$TMP_DIR/unknown-field.json"
write_fixture "$unknown_field_json" '{"version":"1.2.3","build":3,"extra":true}'
assert_rejected "unknown config fields are forbidden" "$unknown_field_json"

wrong_case_key_json="$TMP_DIR/wrong-case-key.json"
write_fixture "$wrong_case_key_json" '{"Version":"9.8.7","build":3}'
assert_rejected "config keys are case-sensitive" "$wrong_case_key_json"

duplicate_version_json="$TMP_DIR/duplicate-version.json"
write_fixture "$duplicate_version_json" '{"version":"1.2.3","version":"9.8.7","build":3}'
assert_rejected "duplicate version key is forbidden" "$duplicate_version_json"

duplicate_build_json="$TMP_DIR/duplicate-build.json"
write_fixture "$duplicate_build_json" '{"version":"1.2.3","build":3,"build":4}'
assert_rejected "duplicate build key is forbidden" "$duplicate_build_json"

escaped_duplicate_key_json="$TMP_DIR/escaped-duplicate-key.json"
write_fixture "$escaped_duplicate_key_json" '{"version":"1.2.3","\u0076ersion":"9.8.7","build":3}'
assert_rejected "escaped duplicate key is forbidden after decoding" "$escaped_duplicate_key_json"

multiple_values_json="$TMP_DIR/multiple-values.json"
write_fixture "$multiple_values_json" '{"version":"1.2.3","build":3} {"version":"4.5.6","build":7}'
assert_rejected "multiple top-level JSON values are forbidden" "$multiple_values_json"

xml_plist="$TMP_DIR/version.plist"
write_fixture "$xml_plist" '<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>version</key><string>1.2.3</string><key>build</key><integer>3</integer></dict></plist>'
assert_rejected "XML plist is not JSON config" "$xml_plist"

json_array="$TMP_DIR/version-array.json"
write_fixture "$json_array" '[{"version":"1.2.3","build":3}]'
assert_rejected "JSON array is not a config object" "$json_array"

missing_version="$TMP_DIR/missing-version.json"
write_fixture "$missing_version" '{"build":3}'
assert_rejected "missing version field" "$missing_version"

missing_build="$TMP_DIR/missing-build.json"
write_fixture "$missing_build" '{"version":"1.2.3"}'
assert_rejected "missing build field" "$missing_build"

newline_version="$TMP_DIR/newline-version.json"
write_fixture "$newline_version" '{"version":"1.2.3\nunsafe","build":3}'
assert_rejected "version containing escaped newline" "$newline_version"

carriage_return_version="$TMP_DIR/carriage-return-version.json"
write_fixture "$carriage_return_version" '{"version":"1.2.3\runsafe","build":3}'
assert_rejected "version containing escaped carriage return" "$carriage_return_version"

trailing_newline_version="$TMP_DIR/trailing-newline-version.json"
write_fixture "$trailing_newline_version" '{"version":"1.2.3\n","build":3}'
assert_rejected "version ending in escaped newline with byte-exact stdout check" "$trailing_newline_version"

for invalid_version in \
  '1.2' \
  'v1.2.3' \
  '01.2.3' \
  '1.02.3' \
  '1.2.3-beta' \
  '1.2.3+meta' \
  '1.2.3 unsafe' \
  '1.2.3/../../bad' \
  '1.2.3-'; do
  fixture="$TMP_DIR/invalid-version-$failed-$passed.json"
  write_fixture "$fixture" "{\"version\":\"$invalid_version\",\"build\":3}"
  assert_rejected "invalid version: $invalid_version" "$fixture"
done

for invalid_build in '0' '-1' '"abc"' '1.5' '"3"' '10000' '9223372036854775807'; do
  fixture="$TMP_DIR/invalid-build-$failed-$passed.json"
  write_fixture "$fixture" "{\"version\":\"1.2.3\",\"build\":$invalid_build}"
  assert_rejected "invalid build: $invalid_build" "$fixture"
done

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[ "$failed" -eq 0 ]
