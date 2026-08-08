package truyentep

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r == 0 || r == '/' || r == '\\' || unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")
	if name == "" {
		return "tep-nhan"
	}
	runes := []rune(name)
	if len(runes) > 220 {
		ext := filepath.Ext(name)
		maxStem := 220 - len([]rune(ext))
		if maxStem < 1 {
			maxStem = 220
			ext = ""
		}
		stemRunes := []rune(strings.TrimSuffix(name, filepath.Ext(name)))
		if len(stemRunes) > maxStem {
			stemRunes = stemRunes[:maxStem]
		}
		name = string(stemRunes) + ext
	}
	return name
}

func saveIncomingFile(dir, requestedName string, source io.Reader) (string, int64, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 0, fmt.Errorf("không tạo được thư mục nhận tệp: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".truyen-tep-*.part")
	if err != nil {
		return "", 0, fmt.Errorf("không tạo được tệp tạm: %w", err)
	}
	tempPath := temp.Name()
	keep := false
	defer func() {
		_ = temp.Close()
		if !keep {
			_ = os.Remove(tempPath)
		}
	}()

	written, err := io.Copy(temp, source)
	if err != nil {
		return "", written, fmt.Errorf("không ghi được tệp nhận: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return "", written, fmt.Errorf("không đồng bộ được tệp nhận: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", written, fmt.Errorf("không đóng được tệp nhận: %w", err)
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		return "", written, fmt.Errorf("không đặt được quyền cho tệp nhận: %w", err)
	}

	finalPath, err := reserveUniquePath(dir, sanitizeFilename(requestedName))
	if err != nil {
		return "", written, err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(finalPath)
		return "", written, fmt.Errorf("không hoàn tất được tệp nhận: %w", err)
	}
	keep = true
	return finalPath, written, nil
}

func reserveUniquePath(dir, name string) (string, error) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for index := 0; index < 10_000; index++ {
		candidateName := name
		if index > 0 {
			candidateName = fmt.Sprintf("%s (%d)%s", stem, index, ext)
		}
		candidate := filepath.Join(dir, candidateName)
		file, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(candidate)
				return "", fmt.Errorf("không giữ được tên tệp: %w", closeErr)
			}
			return candidate, nil
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("không chọn được tên tệp: %w", err)
		}
	}
	return "", fmt.Errorf("không tìm được tên trống cho %q", name)
}
