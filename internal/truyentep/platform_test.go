package truyentep

import (
	"strings"
	"testing"
)

func TestClipboardEnvironmentForcesUTF8Locale(t *testing.T) {
	environment := clipboardEnvironment([]string{
		"PATH=/usr/bin:/bin",
		"LANG=C",
		"LC_ALL=C",
		"LC_CTYPE=US-ASCII",
		"TRUYEN_TEP_TEST=kept",
	})

	values := make(map[string]string)
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if found {
			values[name] = value
		}
	}
	if values["LC_CTYPE"] != "en_US.UTF-8" {
		t.Fatalf("LC_CTYPE = %q", values["LC_CTYPE"])
	}
	if _, exists := values["LC_ALL"]; exists {
		t.Fatalf("LC_ALL vẫn ghi đè locale UTF-8: %q", values["LC_ALL"])
	}
	if values["TRUYEN_TEP_TEST"] != "kept" {
		t.Fatal("biến môi trường không liên quan đã bị mất")
	}
}
