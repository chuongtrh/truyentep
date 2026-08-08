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
		`class="step-number"`,
		`id="action-hint"`,
		`class="action-grid"`,
		`aria-describedby="action-hint file-action-description"`,
		`id="file-action-description"`,
	} {
		if !strings.Contains(html, hook) {
			t.Errorf("web UI is missing semantic hook %q", hook)
		}
	}

	remoteAsset := regexp.MustCompile(`(?i)(?:href|src)=["']https?://`)
	if remoteAsset.MatchString(html) {
		t.Error("web UI must not load assets over http:// or https://")
	}
}
