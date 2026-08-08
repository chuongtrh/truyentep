# macOS Native UI Polish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Polish the embedded Truyền Tệp Web UI into a cohesive macOS-native interface without changing transfer APIs or adding external dependencies.

**Architecture:** Keep the existing HTML/CSS/vanilla JavaScript application and Go embedding boundary. Restructure only presentation markup, introduce a token-driven visual system, and make small JavaScript changes for semantic UI states and SVG event icons while preserving every endpoint and data flow.

**Tech Stack:** HTML5, CSS, vanilla JavaScript, Go `embed`, Go tests, Node syntax check.

---

### Task 1: Lock the semantic UI contract

**Files:**
- Create: `internal/truyentep/web_ui_test.go`
- Modify: `internal/truyentep/web/index.html`

**Step 1: Write the failing test**

Add a Go test that reads `web/index.html` from `webAssets` and requires the new semantic hooks: `class="app-status"`, `class="step-number"`, `id="action-hint"`, `class="action-grid"`, and an accessible file action description. Assert that the page contains no `http://` or `https://` asset URLs.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/truyentep -run TestWebUISemanticContract -count=1`

Expected: FAIL because the new macOS-native semantic hooks do not exist yet.

**Step 3: Implement the semantic shell**

Update `index.html` while preserving every existing element ID used by `app.js`:

- Convert the running state and privacy label into compact header status capsules.
- Replace eyebrow text with numbered step badges.
- Add `#action-hint` and connect disabled actions with `aria-describedby`.
- Wrap clipboard and file actions in `.action-grid` so they are equal visual choices.
- Add meaningful SVG decoration with `aria-hidden="true"`; keep labels and button names as real text.
- Preserve the peer list, manual connection form, progress box, history list, dialog, toast and script/style paths.

**Step 4: Run the focused test**

Run: `go test ./internal/truyentep -run TestWebUISemanticContract -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/truyentep/web_ui_test.go internal/truyentep/web/index.html
git commit -m "Refine UI semantic structure"
```

### Task 2: Build the macOS-native visual system

**Files:**
- Modify: `internal/truyentep/web/style.css`

**Step 1: Define visual tokens**

Replace the current ad-hoc palette with named tokens for canvas, material, elevated material, labels, separators, accent, destructive state, focus ring, radii and shadows. Provide equivalent graphite dark-mode values.

**Step 2: Polish layout and hierarchy**

- Use a restrained toolbar-style header and 720px content width.
- Give panels hairline borders, subtle layered shadows and consistent 18–22px radii.
- Style step numbers, status capsules and selected-peer destination label.
- Make peer cards feel like AirDrop targets with round avatars, online indicator, hover/selected/focus-visible states.
- Present clipboard and file actions as equal tiles on wider screens and stack them on narrow screens.
- Keep progress, event history, manual connection, dialog, toast and footer within the same component language.

**Step 3: Complete interaction and accessibility states**

Add visible `:focus-visible`, disabled, active, dragging, error, dark-mode, mobile and reduced-motion states. Ensure small screens at 560px and below have single-column actions and peers.

**Step 4: Run static validation**

Run: `git diff --check && go test ./internal/truyentep -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/truyentep/web/style.css
git commit -m "Polish macOS native visual system"
```

### Task 3: Refine live interaction states

**Files:**
- Modify: `internal/truyentep/web/app.js`
- Modify: `internal/truyentep/web_ui_test.go`

**Step 1: Extend the failing contract test**

Require JavaScript hooks for `actionHint`, refresh loading state, peer presence decoration and SVG event icons.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/truyentep -run TestWebUIJavaScriptContract -count=1`

Expected: FAIL because the interaction hooks are not implemented.

**Step 3: Implement minimal interaction polish**

- Update `#action-hint` to explain whether a peer must be selected or show the active destination.
- Toggle a loading class and `aria-busy` on the refresh button while state is fetched.
- Add a presence dot to peer cards without changing peer selection logic.
- Replace textual history symbols with inline SVG icons chosen from kind, direction and status.
- Preserve all API paths, headers, local storage key, upload behavior and polling interval.

**Step 4: Verify syntax and tests**

Run: `node --check internal/truyentep/web/app.js && go test ./internal/truyentep -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/truyentep/web/app.js internal/truyentep/web_ui_test.go
git commit -m "Refine UI interaction feedback"
```

### Task 4: Verify the complete application

**Files:**
- Verify only; modify earlier files only if a check reveals a defect.

**Step 1: Validate repository formatting and JavaScript**

Run: `git diff --check && node --check internal/truyentep/web/app.js`

Expected: PASS with no output.

**Step 2: Run all Go checks**

Run: `go test -count=1 ./... && go vet ./...`

Expected: PASS.

**Step 3: Build and inspect the macOS package**

Run: `./scripts/test-build-macos-no-source.sh`

Expected: the script reports that the release contains only the application and no source or project documents.

**Step 4: Review scope**

Run: `git status -sb && git log --oneline --decorate -5`

Expected: only intentional UI commits on `feature/macos-native-ui-polish`, with a clean worktree.
