#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
publish_source="$project_root/scripts/publish-release.sh"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT

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

new_fixture() {
  fixture="$(mktemp -d "$temporary/fixture.XXXXXX")"
  repo="$fixture/repo"
  origin="$fixture/origin.git"
  fake_bin="$fixture/bin"
  command_log="$fixture/commands.log"

  git init --bare -q "$origin"
  git init -q -b main "$repo"
  git -C "$repo" config user.name "Release Test"
  git -C "$repo" config user.email "release-test@example.com"
  mkdir -p "$repo/scripts" "$fake_bin"
  cp "$publish_source" "$repo/scripts/publish-release.sh"

  printf '{"version":"0.3.0","build":3}\n' > "$repo/version.json"
  printf 'fixture\n' > "$repo/README.md"

  cat > "$repo/scripts/read-version.sh" <<'EOF'
#!/usr/bin/env bash
printf '0.3.0\t3\n'
EOF
  cat > "$repo/scripts/test-version-config.sh" <<'EOF'
#!/usr/bin/env bash
printf 'version-test\n' >> "$PUBLISH_TEST_LOG"
EOF
  cat > "$repo/scripts/build-macos.sh" <<'EOF'
#!/usr/bin/env bash
printf 'build\n' >> "$PUBLISH_TEST_LOG"
if [[ "${PUBLISH_TEST_BUILD_FAIL:-0}" == "1" ]]; then
  printf 'fixture build failed\n' >&2
  exit 42
fi
mkdir -p "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/dist"
archive="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/dist/TruyenTep-macOS-v0.3.0.zip"
: > "$archive"
printf '%s\n' "$archive"
EOF
  cat > "$fake_bin/go" <<'EOF'
#!/usr/bin/env bash
printf 'go %s\n' "$*" >> "$PUBLISH_TEST_LOG"
EOF
  cat > "$fake_bin/gh" <<'EOF'
#!/usr/bin/env bash
printf 'gh %s\n' "$*" >> "$PUBLISH_TEST_LOG"
if [[ "${1:-}" == "release" && "${2:-}" == "create" &&
      "${PUBLISH_TEST_GH_FAIL:-0}" == "1" ]]; then
  exit 41
fi
EOF
  chmod +x "$repo/scripts/"*.sh "$fake_bin/go" "$fake_bin/gh"

  git -C "$repo" add .
  git -C "$repo" commit -q -m "fixture"
  git -C "$repo" remote add origin "$origin"
  git -C "$repo" push -q -u origin main
}

run_publish() {
  output_file="$fixture/output"
  if (
    cd "$repo"
    PATH="$fake_bin:$PATH" PUBLISH_TEST_LOG="$command_log" \
      PUBLISH_TEST_BUILD_FAIL="${PUBLISH_TEST_BUILD_FAIL:-0}" \
      PUBLISH_TEST_GH_FAIL="${PUBLISH_TEST_GH_FAIL:-0}" \
      ./scripts/publish-release.sh
  ) >"$output_file" 2>&1; then
    publish_status=0
  else
    publish_status=$?
  fi
}

test_duplicate_remote_tag() {
  new_fixture
  git -C "$repo" tag -a v0.3.0 -m "existing"
  git -C "$repo" push -q origin v0.3.0
  git -C "$repo" tag -d v0.3.0 >/dev/null

  run_publish

  if [[ "$publish_status" -ne 0 ]] &&
    grep -q 'v0.3.0' "$output_file" &&
    grep -qi 'đã tồn tại\|da ton tai\|already exists' "$output_file" &&
    ! grep -Eq '^(go |version-test|build|gh release)' "$command_log" 2>/dev/null; then
    pass "remote tag matching version.json is rejected before build"
  else
    fail "remote tag matching version.json is rejected before build"
    sed -n '1,120p' "$output_file" >&2
  fi
}

test_dirty_worktree() {
  new_fixture
  printf 'dirty\n' >> "$repo/README.md"

  run_publish

  if [[ "$publish_status" -ne 0 ]] &&
    grep -qi 'working tree\|chưa commit\|chua commit' "$output_file" &&
    ! grep -Eq '^(go |version-test|build|gh release)' "$command_log" 2>/dev/null; then
    pass "dirty working tree is rejected before build"
  else
    fail "dirty working tree is rejected before build"
    sed -n '1,120p' "$output_file" >&2
  fi
}

test_build_failure_does_not_create_tag() {
  new_fixture
  PUBLISH_TEST_BUILD_FAIL=1 run_publish
  unset PUBLISH_TEST_BUILD_FAIL

  if [[ "$publish_status" -ne 0 ]] &&
    grep -q '^build$' "$command_log" &&
    ! git -C "$repo" rev-parse --verify --quiet refs/tags/v0.3.0 >/dev/null &&
    ! git --git-dir="$origin" rev-parse --verify --quiet refs/tags/v0.3.0 >/dev/null &&
    ! grep -q '^gh release' "$command_log"; then
    pass "build failure does not create or push a tag"
  else
    fail "build failure does not create or push a tag"
    sed -n '1,120p' "$output_file" >&2
    sed -n '1,120p' "$command_log" >&2
  fi
}

test_successful_publish() {
  new_fixture

  run_publish
  archive="$repo/dist/TruyenTep-macOS-v0.3.0.zip"
  remote_tag_type="$(git --git-dir="$origin" cat-file -t refs/tags/v0.3.0 2>/dev/null || true)"

  if [[ "$publish_status" -eq 0 ]] &&
    [[ "$remote_tag_type" == "tag" ]] &&
    grep -q '^go test ./\.\.\.$' "$command_log" &&
    grep -q '^version-test$' "$command_log" &&
    grep -q '^build$' "$command_log" &&
    grep -Fq "gh release create v0.3.0 $archive" "$command_log" &&
    grep -q -- '--verify-tag' "$command_log" &&
    grep -q -- '--generate-notes' "$command_log"; then
    pass "successful publish pushes an annotated tag and attaches the ZIP"
  else
    fail "successful publish pushes an annotated tag and attaches the ZIP"
    sed -n '1,120p' "$output_file" >&2
    sed -n '1,120p' "$command_log" >&2
  fi
}

test_release_failure_keeps_tag_and_prints_retry() {
  new_fixture
  PUBLISH_TEST_GH_FAIL=1 run_publish
  unset PUBLISH_TEST_GH_FAIL
  remote_tag_type="$(git --git-dir="$origin" cat-file -t refs/tags/v0.3.0 2>/dev/null || true)"

  if [[ "$publish_status" -ne 0 ]] &&
    [[ "$remote_tag_type" == "tag" ]] &&
    grep -qi 'giữ nguyên tag\|giu nguyen tag' "$output_file" &&
    grep -q 'gh release create' "$output_file"; then
    pass "GitHub release failure keeps the remote tag and prints a retry command"
  else
    fail "GitHub release failure keeps the remote tag and prints a retry command"
    sed -n '1,160p' "$output_file" >&2
    sed -n '1,120p' "$command_log" >&2
  fi
}

if [[ ! -x "$publish_source" ]]; then
  printf 'FAIL: missing executable %s\n' "$publish_source" >&2
  exit 1
fi

test_duplicate_remote_tag
test_dirty_worktree
test_build_failure_does_not_create_tag
test_successful_publish
test_release_failure_keeps_tag_and_prints_retry

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[[ "$failed" -eq 0 ]]
