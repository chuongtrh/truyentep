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
