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
	contracts := []struct {
		name    string
		pattern string
	}{
		{"action hint DOM reference", `actionHint\s*:\s*document\.querySelector\("#action-hint"\)`},
		{"action hint rendering", `ui\.actionHint\.textContent\s*=`},
		{"concurrent state load tracking", `stateLoadsInFlight\s*\+=\s*1`},
		{"refresh loading class", `ui\.refreshButton\.classList\.toggle\("loading",\s*loading\)`},
		{"refresh busy state", `ui\.refreshButton\.setAttribute\("aria-busy",\s*loading\s*\?\s*"true"\s*:\s*"false"\)`},
		{"peer presence decoration", `presence\.className\s*=\s*"peer-presence"`},
		{"safe SVG event icon", `document\.createElementNS\(SVG_NAMESPACE,\s*"svg"\)`},
		{"event icon selection", `createEventIcon\(event\)`},
	}
	for _, contract := range contracts {
		if !regexp.MustCompile(contract.pattern).MatchString(js) {
			t.Errorf("web UI script is missing %s", contract.name)
		}
	}

	if strings.Contains(js, `icon.textContent = event.kind`) {
		t.Error("history events must use SVG icons instead of textual symbols")
	}
}
