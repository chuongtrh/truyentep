package truyentep

import (
	"regexp"
	"strings"
	"testing"
)

func TestWebUISemanticContract(t *testing.T) {
	page, err := webAssets.ReadFile("web/index.html")
	if err != nil {
		t.Fatalf("read embedded web UI: %v", err)
	}

	html := string(page)
	for _, hook := range []string{
		`class="app-status"`,
		`id="action-hint"`,
		`class="action-grid"`,
		`aria-describedby="action-hint file-action-description"`,
		`id="file-action-description"`,
	} {
		if !strings.Contains(html, hook) {
			t.Errorf("web UI is missing semantic hook %q", hook)
		}
	}

	stepBadge := regexp.MustCompile(`<span class="step-number"([^>]*)>([^<]*)</span>`)
	stepBadges := stepBadge.FindAllStringSubmatch(html, -1)
	if len(stepBadges) != 2 {
		t.Errorf("web UI must have exactly two step badges, got %d", len(stepBadges))
	}
	ariaHidden := regexp.MustCompile(`(?i)aria-hidden\s*=\s*["']true["']`)
	for _, badge := range stepBadges {
		if ariaHidden.MatchString(badge[1]) {
			t.Errorf("step badge %q must be available to screen readers", strings.TrimSpace(badge[2]))
		}
	}

	remoteAsset := regexp.MustCompile(`(?i)(?:href|src|srcset)\s*=\s*["'][^"']*https?://`)
	if remoteAsset.MatchString(html) {
		t.Error("web UI must not load assets over http:// or https://")
	}
}

func TestWebUIInteractionContract(t *testing.T) {
	script, err := webAssets.ReadFile("web/app.js")
	if err != nil {
		t.Fatalf("read embedded web UI script: %v", err)
	}

	js := string(script)
	functionSource := func(name, next string) string {
		t.Helper()
		start := strings.Index(js, name)
		if start < 0 {
			t.Fatalf("web UI script is missing %s", name)
		}
		end := strings.Index(js[start:], next)
		if end < 0 {
			t.Fatalf("cannot isolate %s", name)
		}
		return js[start : start+end]
	}

	loadState := functionSource("async function loadState", "function setRefreshLoading")
	latestStartedGuard := `if (loadId !== stateLoadSequence) return;`
	if count := strings.Count(loadState, latestStartedGuard); count != 2 {
		t.Errorf("loadState must guard both success and failure against the latest started request, got %d guards", count)
	}
	for _, pattern := range []string{
		`const loadId\s*=\s*\+\+stateLoadSequence;\s*stateLoadsInFlight \+= 1;\s*setRefreshLoading\(true\)`,
		`const nextState\s*=\s*await api\("/api/state"\);\s*if \(loadId !== stateLoadSequence\) return;\s*state\s*=\s*nextState;\s*render\(\)`,
		`catch \(error\) \{\s*if \(loadId !== stateLoadSequence\) return;\s*if \(showFailure\) showToast\(error\.message, true\);\s*ui\.selfStatus\.innerHTML`,
		`finally \{\s*stateLoadsInFlight\s*=\s*Math\.max\(0, stateLoadsInFlight - 1\);\s*setRefreshLoading\(stateLoadsInFlight > 0\);\s*\}`,
	} {
		if !regexp.MustCompile(pattern).MatchString(loadState) {
			t.Errorf("loadState does not satisfy interaction contract %q", pattern)
		}
	}

	refreshLoading := functionSource("function setRefreshLoading", "function render()")
	for _, hook := range []string{
		`ui.refreshButton.classList.toggle("loading", loading)`,
		`ui.refreshButton.setAttribute("aria-busy", loading ? "true" : "false")`,
	} {
		if !strings.Contains(refreshLoading, hook) {
			t.Errorf("refresh loading helper is missing %q", hook)
		}
	}

	actions := functionSource("function renderActions", "function renderEvents")
	if !strings.Contains(js, `actionHint: document.querySelector("#action-hint")`) {
		t.Error("web UI script must reference the action hint element")
	}
	for _, pattern := range []string{
		`if \(!selected\) \{\s*ui\.actionHint\.textContent = "Chọn một máy nhận`,
		`else if \(sending\) \{\s*ui\.actionHint\.textContent = ` + "`" + `Đang gửi đến \$\{selected\.name\}`,
		`else \{\s*ui\.actionHint\.textContent = ` + "`" + `Sẵn sàng gửi đến \$\{selected\.name\}`,
	} {
		if !regexp.MustCompile(pattern).MatchString(actions) {
			t.Errorf("renderActions is missing a distinct hint state matching %q", pattern)
		}
	}

	peers := functionSource("function renderPeers", "function renderActions")
	for _, pattern := range []string{
		`const presence = document\.createElement\("span"\);\s*presence\.className = "peer-presence";\s*presence\.setAttribute\("aria-hidden", "true"\);\s*avatar\.append\(presence\)`,
		`button\.className = ` + "`" + `peer-card\$\{peer\.id === selectedPeerId \? " selected" : ""\}`,
		`button\.setAttribute\("aria-pressed", peer\.id === selectedPeerId \? "true" : "false"\)`,
		`button\.addEventListener\("click", \(\) => \{\s*selectedPeerId = peer\.id;\s*saveSelectedPeer\(\);\s*render\(\)`,
	} {
		if !regexp.MustCompile(pattern).MatchString(peers) {
			t.Errorf("renderPeers does not preserve presence/selection behavior %q", pattern)
		}
	}

	events := functionSource("function renderEvents", "function createEventIcon")
	if !strings.Contains(events, "icon.append(createEventIcon(event))") {
		t.Error("renderEvents must use the safe SVG icon helper")
	}

	eventIcon := functionSource("function createEventIcon", "function renderOwnAddress")
	for _, pattern := range []string{
		`document\.createElementNS\(SVG_NAMESPACE, "svg"\)`,
		`svg\.setAttribute\("aria-hidden", "true"\)`,
		`event\.status === "failed"`,
		`event\.kind === "clipboard"`,
		`event\.direction === "received"`,
		`document\.createElementNS\(SVG_NAMESPACE, "path"\)`,
	} {
		if !regexp.MustCompile(pattern).MatchString(eventIcon) {
			t.Errorf("event SVG helper is missing %q", pattern)
		}
	}
	if strings.Contains(eventIcon, "innerHTML") {
		t.Error("event SVG helper must not construct icons with innerHTML")
	}
}
