package truyentep

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type SystemClipboard struct{}

func (SystemClipboard) Read(ctx context.Context) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", ErrClipboardUnsupported
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/pbpaste")
	cmd.Env = clipboardEnvironment(os.Environ())
	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("không đọc được clipboard: %w", err)
	}
	return output.String(), nil
}

func (SystemClipboard) Write(ctx context.Context, text string) error {
	if runtime.GOOS != "darwin" {
		return ErrClipboardUnsupported
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/pbcopy")
	cmd.Env = clipboardEnvironment(os.Environ())
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("không ghi được clipboard: %w", err)
	}
	return nil
}

func clipboardEnvironment(environ []string) []string {
	result := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if name != "LC_ALL" && name != "LC_CTYPE" {
			result = append(result, entry)
		}
	}
	return append(result, "LC_CTYPE=en_US.UTF-8")
}

type SystemNotifier struct{}

func (SystemNotifier) Notify(title, message string) {
	if runtime.GOOS != "darwin" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		script := fmt.Sprintf(
			"display notification \"%s\" with title \"%s\"",
			appleScriptText(message), appleScriptText(title),
		)
		_ = exec.CommandContext(ctx, "/usr/bin/osascript", "-e", script).Run()
	}()
}

func appleScriptText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	runes := []rune(value)
	if len(runes) > 180 {
		value = string(runes[:180]) + "…"
	}
	return value
}

func OpenBrowser(url string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command, args = "/usr/bin/open", []string{url}
	case "linux":
		command, args = "xdg-open", []string{url}
	default:
		return fmt.Errorf("hãy mở %s trong trình duyệt", url)
	}
	return exec.Command(command, args...).Start()
}

func OpenFolder(path string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("chỉ hỗ trợ mở thư mục tự động trên macOS")
	}
	return exec.Command("/usr/bin/open", path).Start()
}
