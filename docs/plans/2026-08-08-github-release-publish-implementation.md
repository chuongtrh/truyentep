# GitHub Release Publishing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a safe command that publishes the version in `version.json` as a tagged GitHub Release with the macOS ZIP attached and refuses duplicate remote tags.

**Architecture:** A Bash entry point performs read-only preflight checks before invoking tests or build, then creates and pushes one annotated tag before calling GitHub CLI. A shell integration test uses real temporary Git repositories and small fake project commands so Git state transitions are exercised without contacting GitHub or building the macOS app.

**Tech Stack:** Bash, Git, GitHub CLI, existing Go/macOS build scripts

---

### Task 1: Duplicate-version and repository preflight

**Files:**
- Create: `scripts/test-publish-release.sh`
- Create: `scripts/publish-release.sh`

**Step 1: Write the failing tests**

Create a temporary working repository and bare `origin`, then add cases asserting that a remote `v0.3.0` tag is rejected before test/build commands run and that a dirty working tree is rejected.

**Step 2: Run tests to verify they fail**

Run: `./scripts/test-publish-release.sh`

Expected: FAIL because `scripts/publish-release.sh` does not exist.

**Step 3: Write minimal implementation**

Implement helpers for errors and required commands; resolve the project root; read the canonical version; verify a clean working tree, configured `origin`, an upstream at the current commit, GitHub authentication, and absence of `refs/tags/v<version>` from `origin`.

**Step 4: Run tests to verify they pass**

Run: `./scripts/test-publish-release.sh`

Expected: duplicate-tag and dirty-tree cases PASS.

**Step 5: Commit**

Run: `git add scripts/publish-release.sh scripts/test-publish-release.sh && git commit -m "feat: validate release publishing preconditions"`

### Task 2: Build, tag, push, and attach ZIP

**Files:**
- Modify: `scripts/test-publish-release.sh`
- Modify: `scripts/publish-release.sh`

**Step 1: Write the failing tests**

Add a case where the build command fails and assert no tag is created. Add a successful case asserting an annotated tag reaches `origin`, the GitHub CLI receives `release create`, `--verify-tag`, `--generate-notes`, and the exact ZIP path.

**Step 2: Run tests to verify they fail**

Run: `./scripts/test-publish-release.sh`

Expected: FAIL because the script does not yet run tests/build or publish a release.

**Step 3: Write minimal implementation**

Run `go test ./...`, `scripts/test-version-config.sh`, and `scripts/build-macos.sh`; validate the returned artifact path; create and push an annotated tag; then run `gh release create` with generated notes. On GitHub release failure, retain the remote tag and print a retry command.

**Step 4: Run tests to verify they pass**

Run: `./scripts/test-publish-release.sh`

Expected: all publish integration cases PASS.

**Step 5: Commit**

Run: `git add scripts/publish-release.sh scripts/test-publish-release.sh && git commit -m "feat: publish macOS archive to GitHub Releases"`

### Task 3: Documentation and full verification

**Files:**
- Modify: `README.md`

**Step 1: Document usage and safety behavior**

Add a release section explaining `./scripts/publish-release.sh`, required tools/authentication, source of version, generated tag, and duplicate remote-tag refusal.

**Step 2: Run shell syntax checks**

Run: `bash -n scripts/publish-release.sh scripts/test-publish-release.sh`

Expected: exit 0 with no output.

**Step 3: Run all tests**

Run: `./scripts/test-publish-release.sh && ./scripts/test-version-config.sh && go test ./...`

Expected: every test passes.

**Step 4: Verify the patch**

Run: `git diff --check && git status --short`

Expected: no whitespace errors; only intended files are modified.

**Step 5: Commit**

Run: `git add README.md docs/plans/2026-08-08-github-release-publish-implementation.md && git commit -m "docs: explain GitHub release publishing"`
