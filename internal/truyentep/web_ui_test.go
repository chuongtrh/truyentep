package truyentep

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestWebUIShowsRuntimeVersion(t *testing.T) {
	page, err := webAssets.ReadFile("web/index.html")
	if err != nil {
		t.Fatalf("read embedded web UI: %v", err)
	}
	script, err := webAssets.ReadFile("web/app.js")
	if err != nil {
		t.Fatalf("read embedded web UI script: %v", err)
	}
	stylesheet, err := webAssets.ReadFile("web/style.css")
	if err != nil {
		t.Fatalf("read embedded web UI stylesheet: %v", err)
	}

	footer := regexp.MustCompile(`(?s)<footer\b[^>]*>(.*?)</footer>`).FindStringSubmatch(string(page))
	if len(footer) != 2 {
		t.Fatal("web UI must have a footer")
	}
	if !regexp.MustCompile(`<p\s+id="app-version"[^>]*>\s*Phiên bản …\s*</p>`).MatchString(footer[1]) {
		t.Error("footer must include a semantic #app-version paragraph with its initial loading text")
	}

	js := string(script)
	if !strings.Contains(js, `appVersion: document.querySelector("#app-version")`) {
		t.Error("web UI script must query the app version element")
	}
	for _, contract := range []string{
		`function formatAppVersion(value)`,
		`typeof value !== "string"`,
		`const version = value.trim()`,
		`return version || "dev"`,
		`ui.appVersion.textContent = ` + "`" + `Phiên bản ${formatAppVersion(state.version)}` + "`",
	} {
		if !strings.Contains(js, contract) {
			t.Errorf("runtime version rendering is missing contract %q", contract)
		}
	}
	if strings.Contains(js, "version.json") {
		t.Error("web UI must use runtime state instead of reading version.json")
	}
	if !regexp.MustCompile(`(?s)\.app-version\s*\{[^}]*color:\s*var\(--label-secondary\);[^}]*\}`).Match(stylesheet) {
		t.Error("app version must use the higher-contrast shared light/dark text color token")
	}
}

func TestStateReportsRuntimeVersion(t *testing.T) {
	expectedVersion := Version

	app := testApp(t, "Mac", strings.Repeat("9", 32), t.TempDir(), &fakeClipboard{})
	request := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("state status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	if payload.Version != expectedVersion {
		t.Fatalf("state version = %q, want runtime Version %q", payload.Version, expectedVersion)
	}
}

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
		`let activeStateLoadController = null`,
		`const STATE_LOAD_TIMEOUT_MS = [1-9][0-9]{2,4}`,
	} {
		if !regexp.MustCompile(pattern).MatchString(js) {
			t.Errorf("state request lifecycle is missing %q", pattern)
		}
	}
	for _, pattern := range []string{
		`const loadId\s*=\s*\+\+stateLoadSequence;\s*activeStateLoadController\?\.abort\(\);\s*const controller = new AbortController\(\);\s*activeStateLoadController = controller;\s*stateLoadsInFlight \+= 1;\s*setRefreshLoading\(true\)`,
		`let timedOut = false;\s*const timeoutTimer = window\.setTimeout\(\(\) => \{\s*timedOut = true;\s*controller\.abort\(\);\s*\}, STATE_LOAD_TIMEOUT_MS\)`,
		`const nextState\s*=\s*await api\("/api/state", \{ signal: controller\.signal \}\);\s*if \(loadId !== stateLoadSequence\) return;\s*state\s*=\s*nextState;\s*render\(\)`,
		`catch \(error\) \{\s*if \(loadId !== stateLoadSequence\) return;\s*const message = timedOut \? "Không thể cập nhật trạng thái\. Vui lòng thử lại\." : error\.message;\s*if \(showFailure\) showToast\(message, true\);\s*ui\.selfStatus\.innerHTML`,
		`finally \{\s*window\.clearTimeout\(timeoutTimer\);\s*if \(activeStateLoadController === controller\) activeStateLoadController = null;\s*stateLoadsInFlight\s*=\s*Math\.max\(0, stateLoadsInFlight - 1\);\s*setRefreshLoading\(stateLoadsInFlight > 0\);\s*\}`,
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
		`if \(!peer\.manual\) \{\s*const presence = document\.createElement\("span"\);\s*presence\.className = "peer-presence";\s*presence\.setAttribute\("aria-hidden", "true"\);\s*avatar\.append\(presence\);\s*\}`,
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

	for _, pattern := range []string{
		`ui\.refreshButton\.addEventListener\("click", \(\) => loadState\(true\)\)`,
		`pollTimer = window\.setInterval\(\(\) => \{\s*if \(stateLoadsInFlight > 0\) return;\s*loadState\(\);\s*\}, 2000\)`,
	} {
		if !regexp.MustCompile(pattern).MatchString(js) {
			t.Errorf("state refresh scheduling is missing %q", pattern)
		}
	}
}

func TestWebUIFileActionUsesOneNativeFocusTarget(t *testing.T) {
	page, err := webAssets.ReadFile("web/index.html")
	if err != nil {
		t.Fatalf("read embedded web UI: %v", err)
	}
	script, err := webAssets.ReadFile("web/app.js")
	if err != nil {
		t.Fatalf("read embedded web UI script: %v", err)
	}
	stylesheet, err := webAssets.ReadFile("web/style.css")
	if err != nil {
		t.Fatalf("read embedded web UI stylesheet: %v", err)
	}

	html := string(page)
	js := string(script)
	css := string(stylesheet)
	dropZone := regexp.MustCompile(`<label\s+id="drop-zone"([^>]*)>`).FindStringSubmatch(html)
	if len(dropZone) != 2 {
		t.Fatal("file action must remain a label for the native file input")
	}
	if regexp.MustCompile(`(?i)\btabindex\s*=`).MatchString(dropZone[1]) {
		t.Error("file action label must not add a second keyboard focus target")
	}
	if !regexp.MustCompile(`<input\s+id="file-input"\s+type="file"\s+multiple\s+disabled>`).MatchString(html) {
		t.Error("file action must use the native disabled file input as its only focus target")
	}
	for _, forbidden := range []string{
		`ui.dropZone.tabIndex`,
		`ui.dropZone.addEventListener("keydown"`,
		`ui.fileInput.click()`,
	} {
		if strings.Contains(js, forbidden) {
			t.Errorf("file action must not use redundant label keyboard activation %q", forbidden)
		}
	}
	for _, pattern := range []string{
		`(?s)\.drop-zone input\s*\{[^}]*position:\s*absolute;[^}]*inset:\s*0;[^}]*width:\s*100%;[^}]*height:\s*100%;[^}]*opacity:\s*0;[^}]*cursor:\s*pointer;[^}]*\}`,
		`(?s)\.drop-zone:focus-within\s*\{[^}]*outline:\s*3px solid var\(--focus-ring\);[^}]*outline-offset:\s*2px;[^}]*\}`,
	} {
		if !regexp.MustCompile(pattern).MatchString(css) {
			t.Errorf("file action CSS is missing the native focus architecture %q", pattern)
		}
	}
}

func TestWebUIStateLoadFailureUsesOfflineStatus(t *testing.T) {
	script, err := webAssets.ReadFile("web/app.js")
	if err != nil {
		t.Fatalf("read embedded web UI script: %v", err)
	}
	stylesheet, err := webAssets.ReadFile("web/style.css")
	if err != nil {
		t.Fatalf("read embedded web UI stylesheet: %v", err)
	}

	if !strings.Contains(string(script), `<span class="status-dot offline"></span>Mất kết nối với ứng dụng`) {
		t.Error("state load failure must render a semantically offline status dot")
	}
	if !regexp.MustCompile(`(?s)\.status-dot\.offline\s*\{[^}]*background:[^;}]+;[^}]*box-shadow:[^;}]+;[^}]*\}`).Match(stylesheet) {
		t.Error("offline status dot must have a distinct non-green visual")
	}
}
