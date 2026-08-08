package truyentep

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := map[string]string{
		"../../secret.zip":    "secret.zip",
		`..\\..\\ghi chú.txt`: "ghi chú.txt",
		"  .hello.txt.  ":     "hello.txt",
		"\x00\n":              "tep-nhan",
		"bao-cao-đầu-tư.zip":  "bao-cao-đầu-tư.zip",
	}
	for input, expected := range tests {
		if actual := sanitizeFilename(input); actual != expected {
			t.Errorf("sanitizeFilename(%q) = %q, muốn %q", input, actual, expected)
		}
	}
}

func TestSaveIncomingFileDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ghi-chu.txt"), []byte("cũ"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, size, err := saveIncomingFile(dir, "ghi-chu.txt", strings.NewReader("mới"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "ghi-chu (1).txt" {
		t.Fatalf("tên tệp = %q", filepath.Base(path))
	}
	if size != int64(len("mới")) {
		t.Fatalf("kích thước = %d", size)
	}
	oldData, _ := os.ReadFile(filepath.Join(dir, "ghi-chu.txt"))
	newData, _ := os.ReadFile(path)
	if string(oldData) != "cũ" || string(newData) != "mới" {
		t.Fatalf("nội dung không đúng: cũ=%q mới=%q", oldData, newData)
	}
}
