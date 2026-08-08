package truyentep

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func DefaultConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("không tìm thấy thư mục người dùng: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "Máy Mac"
	}
	hostname = strings.TrimSuffix(hostname, ".local")

	return Config{
		Name:          hostname,
		Port:          DefaultPort,
		DiscoveryPort: DefaultDiscoveryPort,
		DownloadDir:   filepath.Join(home, "Downloads", "Truyền Tệp"),
	}, nil
}

func prepareConfig(cfg Config) (Config, error) {
	if cfg.Port < 0 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("cổng %d không hợp lệ", cfg.Port)
	}
	if cfg.DiscoveryPort <= 0 || cfg.DiscoveryPort > 65535 {
		return Config{}, fmt.Errorf("cổng tìm máy %d không hợp lệ", cfg.DiscoveryPort)
	}
	cfg.Name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, cfg.Name))
	if cfg.Name == "" {
		cfg.Name = "Máy Mac"
	}
	if len([]rune(cfg.Name)) > 80 {
		return Config{}, errors.New("tên máy dài quá 80 ký tự")
	}
	if strings.TrimSpace(cfg.DownloadDir) == "" {
		return Config{}, errors.New("thư mục nhận tệp không được để trống")
	}
	abs, err := filepath.Abs(cfg.DownloadDir)
	if err != nil {
		return Config{}, fmt.Errorf("thư mục nhận tệp không hợp lệ: %w", err)
	}
	cfg.DownloadDir = abs
	if len(cfg.PrivateKey) == 0 {
		privateKey, err := loadOrCreateIdentity()
		if err != nil {
			return Config{}, err
		}
		cfg.PrivateKey = privateKey
	}
	key, err := ecdh.X25519().NewPrivateKey(cfg.PrivateKey)
	if err != nil {
		return Config{}, errors.New("khóa thiết bị không hợp lệ")
	}
	public := key.PublicKey().Bytes()
	cfg.PublicKey = base64.RawStdEncoding.EncodeToString(public)
	sum := sha256.Sum256(public)
	cfg.DeviceID = hex.EncodeToString(sum[:16])
	return cfg, nil
}

func loadOrCreateIdentity() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy thư mục người dùng: %w", err)
	}
	dir := filepath.Join(home, "Library", "Application Support", "TruyenTep")
	path := filepath.Join(dir, "identity-key")
	if data, err := os.ReadFile(path); err == nil {
		decoded, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if decodeErr == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("không tạo được dữ liệu ứng dụng: %w", err)
	}
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("không tạo được mã máy: %w", err)
	}
	raw := key.Bytes()
	if err := os.WriteFile(path, []byte(base64.RawStdEncoding.EncodeToString(raw)+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("không lưu được mã máy: %w", err)
	}
	return raw, nil
}

func validDeviceID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func localIPv4Addresses() []string {
	addrs, err := netInterfaceAddrs()
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, addr := range addrs {
		ip := addressIP(addr)
		if ip == "" || seen[ip] {
			continue
		}
		seen[ip] = true
		result = append(result, ip)
	}
	sort.Strings(result)
	return result
}

var netInterfaceAddrs = interfaceAddrs
