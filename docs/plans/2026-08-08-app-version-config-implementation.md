# App Version Configuration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `version.json` the single source of application version/build metadata and display the embedded version in the Web UI.

**Architecture:** A small validated shell reader parses the root JSON config. The macOS build consumes that reader for Go ldflags, bundle metadata and artifact naming; the runtime continues exposing its embedded version through `/api/state`, which the existing UI renders.

**Tech Stack:** JSON, Bash 3.2-compatible shell, macOS `plutil`, Go ldflags/tests, HTML/CSS/vanilla JavaScript.

---

### Task 1: Add and validate the version source

**Files:**
- Create: `version.json`
- Create: `scripts/read-version.sh`
- Create: `scripts/test-version-config.sh`

**Step 1: Write the failing shell test**

Create `scripts/test-version-config.sh`. It must run the reader against the real config and temporary valid/invalid fixtures. Require output `0.3.0<TAB>3`; reject missing files, malformed JSON, versions outside the accepted SemVer-like pattern, zero/non-numeric builds and missing fields.

**Step 2: Run it and verify RED**

Run: `./scripts/test-version-config.sh`

Expected: FAIL because `version.json` and the reader do not exist.

**Step 3: Implement minimal config reader**

Add root `version.json` with version `0.3.0` and numeric build `3`. Add an executable Bash reader that accepts an optional config path, verifies `plutil`, extracts both fields, validates version/build, prints one tab-separated line, and writes Vietnamese errors to stderr. It must not source/eval config content.

**Step 4: Verify GREEN**

Run: `./scripts/test-version-config.sh`

Expected: PASS with a clear success message.

**Step 5: Commit**

```bash
git add version.json scripts/read-version.sh scripts/test-version-config.sh
git commit -m "Add validated app version config"
```

### Task 2: Drive macOS builds from the config

**Files:**
- Modify: `scripts/build-macos.sh`
- Modify: `scripts/test-build-macos-no-source.sh`
- Modify: `packaging/Info.plist`

**Step 1: Extend packaging verification first**

Change the no-source build test to read expected values through `read-version.sh`, use only `TRUYEN_TEP_ARTIFACT_SUFFIX=-no-source-test` for isolated artifact names, and assert:

- ZIP/release path contains configured app version plus test suffix.
- Core `--version` reports configured app version, without suffix.
- `CFBundleShortVersionString` equals configured version.
- `CFBundleVersion` equals configured build number.
- Existing no-source assertions remain.

**Step 2: Run and verify RED**

Run: `./scripts/test-build-macos-no-source.sh`

Expected: FAIL because the build script still uses hard-coded/env version and does not set bundle build number.

**Step 3: Implement build integration**

Read version/build before destructive output setup. Remove `TRUYEN_TEP_VERSION`. Use configured version for ldflags and both Info.plist values. Add optional `TRUYEN_TEP_ARTIFACT_SUFFIX` only to release directory/ZIP names, validate that suffix as safe, and add `plutil` to required build tools. Keep the app's embedded version free of the artifact suffix.

Set template plist values to neutral placeholders that are always replaced during build.

**Step 4: Verify GREEN**

Run: `./scripts/test-version-config.sh && ./scripts/test-build-macos-no-source.sh`

Expected: both scripts PASS.

**Step 5: Commit**

```bash
git add scripts/build-macos.sh scripts/test-build-macos-no-source.sh packaging/Info.plist
git commit -m "Build app version from config"
```

### Task 3: Display the runtime version in the UI

**Files:**
- Modify: `internal/truyentep/web/index.html`
- Modify: `internal/truyentep/web/app.js`
- Modify: `internal/truyentep/web/style.css`
- Modify: `internal/truyentep/web_ui_test.go`
- Modify: `README.md`

**Step 1: Add a failing UI contract test**

Require an `#app-version` footer element, JavaScript query hook and render assignment from `state.version`. Also lock that the state endpoint continues exposing the runtime `Version` value.

**Step 2: Run and verify RED**

Run: `go test ./internal/truyentep -run 'TestWebUI.*Version|TestState.*Version' -count=1`

Expected: FAIL because the UI hook/rendering is absent.

**Step 3: Implement runtime display**

Add a subtle footer label with initial text `Phiên bản …`. Query it in `app.js` and set `Phiên bản ${state.version || "dev"}` inside `render()`. Style it consistently for light/dark UI without changing interaction. Update README build instructions to tell maintainers to edit only `version.json` and remove the hard-coded artifact version from prose.

**Step 4: Verify GREEN**

Run: `node --check internal/truyentep/web/app.js && go test -count=1 ./... && go vet ./...`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/truyentep/web/index.html internal/truyentep/web/app.js internal/truyentep/web/style.css internal/truyentep/web_ui_test.go README.md
git commit -m "Show app version in the UI"
```

### Task 4: Verify the complete release path

**Files:** Verify only.

**Step 1: Static checks**

Run: `git diff --check && node --check internal/truyentep/web/app.js`

Expected: PASS.

**Step 2: Config and application tests**

Run: `./scripts/test-version-config.sh && go test -count=1 ./... && go vet ./...`

Expected: PASS.

**Step 3: Full macOS package test**

Run: `./scripts/test-build-macos-no-source.sh`

Expected: PASS, including embedded core version and both bundle version fields.

**Step 4: Scope review**

Run: `git status -sb && git diff --stat main...HEAD && git log --oneline --decorate -8`

Expected: clean feature branch containing only config/build/UI/docs/test changes.
