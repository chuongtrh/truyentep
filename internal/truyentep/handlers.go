package truyentep

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleStatic)
	mux.HandleFunc("/api/ping", a.handlePing)
	mux.HandleFunc("/api/state", a.localOnly(a.handleState))
	mux.HandleFunc("/api/send/clipboard", a.localOnly(a.handleSendClipboard))
	mux.HandleFunc("/api/send/file", a.localOnly(a.handleSendFile))
	mux.HandleFunc("/api/receive/clipboard", a.lanOnly(a.handleReceiveClipboard))
	mux.HandleFunc("/api/receive/file", a.lanOnly(a.handleReceiveFile))
	mux.HandleFunc("/api/peers/manual", a.localOnly(a.handleManualPeer))
	mux.HandleFunc("/api/open-downloads", a.localOnly(a.handleOpenDownloads))
	mux.HandleFunc("/api/quit", a.localOnly(a.handleQuit))
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func (a *App) localOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := requestIP(r.RemoteAddr)
		if ip == nil || !ip.IsLoopback() {
			writeError(w, http.StatusForbidden, "Chỉ có thể điều khiển Truyền Tệp trên máy này")
			return
		}
		if r.Method != http.MethodGet && r.Header.Get("X-Truyen-Tep-Local") != "1" {
			writeError(w, http.StatusForbidden, "Yêu cầu điều khiển không hợp lệ")
			return
		}
		next(w, r)
	}
}

func (a *App) lanOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := requestIP(r.RemoteAddr)
		if !isTrustedLANIP(ip) {
			writeError(w, http.StatusForbidden, "Chỉ nhận dữ liệu từ mạng nội bộ")
			return
		}
		if r.Header.Get("X-Truyen-Tep-Protocol") != strconv.Itoa(ProtocolVersion) {
			writeError(w, http.StatusBadRequest, "Phiên bản truyền dữ liệu không hợp lệ")
			return
		}
		next(w, r)
	}
}

func (a *App) handleStatic(w http.ResponseWriter, r *http.Request) {
	if !requestIP(r.RemoteAddr).IsLoopback() {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	path := ""
	contentType := ""
	switch r.URL.Path {
	case "/", "/index.html":
		path, contentType = "web/index.html", "text/html; charset=utf-8"
	case "/app.js":
		path, contentType = "web/app.js", "text/javascript; charset=utf-8"
	case "/style.css":
		path, contentType = "web/style.css", "text/css; charset=utf-8"
	default:
		http.NotFound(w, r)
		return
	}
	data, err := webAssets.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không đọc được giao diện")
		return
	}
	w.Header().Set("Content-Type", contentType)
	if path == "web/index.html" {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
}

func (a *App) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	ip := requestIP(r.RemoteAddr)
	if !isTrustedLANIP(ip) {
		writeError(w, http.StatusForbidden, "Chỉ phản hồi trong mạng nội bộ")
		return
	}
	w.Header().Set("X-Truyen-Tep-Protocol", strconv.Itoa(ProtocolVersion))
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        a.cfg.DeviceID,
		"name":      a.cfg.Name,
		"port":      a.cfg.Port,
		"version":   ProtocolVersion,
		"publicKey": a.cfg.PublicKey,
	})
}

func (a *App) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"self": map[string]any{
			"id":          a.cfg.DeviceID,
			"name":        a.cfg.Name,
			"port":        a.cfg.Port,
			"addresses":   localIPv4Addresses(),
			"downloadDir": a.cfg.DownloadDir,
		},
		"peers":              a.peerList(),
		"events":             a.events.List(),
		"discoveryError":     a.getDiscoveryError(),
		"clipboardAvailable": runtime.GOOS == "darwin",
		"version":            Version,
		"uptimeSeconds":      int64(time.Since(a.startedAt).Seconds()),
	})
}

func (a *App) handleSendClipboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	var input struct {
		PeerID string `json:"peerId"`
	}
	if !readJSON(w, r, &input, 8*1024) {
		return
	}
	peer, ok := a.peers.Get(input.PeerID)
	if !ok {
		writeError(w, http.StatusNotFound, "Máy nhận không còn trực tuyến")
		return
	}
	readCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	text, err := a.clipboard.Read(readCtx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len([]byte(text)) > MaxClipboardBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Clipboard lớn quá 1 MB")
		return
	}
	aead, ephemeral, salt, err := newCipherForPeer(peer.PublicKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	noncePrefix := make([]byte, 8)
	_, _ = rand.Read(noncePrefix)
	var encrypted strings.Builder
	if _, err = encryptStream(&encrypted, strings.NewReader(text), aead, noncePrefix); err != nil {
		writeError(w, http.StatusInternalServerError, "Không mã hóa được clipboard")
		return
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, peerURL(peer, "/api/receive/clipboard", nil), strings.NewReader(encrypted.String()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không tạo được yêu cầu gửi")
		return
	}
	setPeerHeaders(request, a.cfg)
	setEncryptionHeaders(request, ephemeral, salt, noncePrefix, int64(len([]byte(text))))
	request.Header.Set("Content-Type", "application/octet-stream")
	response, err := a.client.Do(request)
	if err != nil {
		a.addTransferEvent("clipboard", "sent", "Gửi clipboard thất bại", fmt.Sprintf("Không kết nối được %s", peer.Name), "failed")
		writeError(w, http.StatusBadGateway, "Không kết nối được máy nhận")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message := readRemoteError(response)
		a.addTransferEvent("clipboard", "sent", "Gửi clipboard thất bại", message, "failed")
		writeError(w, http.StatusBadGateway, message)
		return
	}
	a.addTransferEvent("clipboard", "sent", "Đã gửi clipboard", fmt.Sprintf("%d ký tự đến %s", len([]rune(text)), peer.Name), "success")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "characters": len([]rune(text))})
}

func (a *App) handleReceiveClipboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	expected, aead, noncePrefix, err := a.receiveCipher(r, MaxClipboardBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var plain strings.Builder
	if _, err := decryptStream(&plain, r.Body, aead, noncePrefix, expected); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	text := plain.String()
	if len([]byte(text)) > MaxClipboardBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Clipboard lớn quá 1 MB")
		return
	}
	writeCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := a.clipboard.Write(writeCtx, text); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sender := senderName(r)
	a.addTransferEvent("clipboard", "received", "Đã nhận clipboard", fmt.Sprintf("%d ký tự từ %s · đã mã hóa", len([]rune(text)), sender), "success")
	a.notifier.Notify("Truyền Tệp", "Đã nhận clipboard từ "+sender)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleSendFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	peer, ok := a.peers.Get(r.URL.Query().Get("peer"))
	if !ok {
		writeError(w, http.StatusNotFound, "Máy nhận không còn trực tuyến")
		return
	}
	name := sanitizeFilename(r.URL.Query().Get("name"))
	if r.ContentLength > MaxFileBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Tệp lớn quá 4 GB")
		return
	}
	if r.ContentLength < 0 {
		writeError(w, http.StatusLengthRequired, "Không xác định được kích thước tệp")
		return
	}
	body := http.MaxBytesReader(w, r.Body, MaxFileBytes)
	aead, ephemeral, salt, err := newCipherForPeer(peer.PublicKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	noncePrefix := make([]byte, 8)
	_, _ = rand.Read(noncePrefix)
	reader, writer := io.Pipe()
	go func() {
		_, cryptErr := encryptStream(writer, body, aead, noncePrefix)
		_ = writer.CloseWithError(cryptErr)
	}()
	query := url.Values{"name": []string{name}}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, peerURL(peer, "/api/receive/file", query), reader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Không tạo được yêu cầu gửi tệp")
		return
	}
	request.Header.Set("Content-Type", "application/octet-stream")
	setPeerHeaders(request, a.cfg)
	setEncryptionHeaders(request, ephemeral, salt, noncePrefix, r.ContentLength)
	response, err := a.client.Do(request)
	if err != nil {
		a.addTransferEvent("file", "sent", "Gửi tệp thất bại", fmt.Sprintf("%s · không kết nối được %s", name, peer.Name), "failed")
		writeError(w, http.StatusBadGateway, "Không gửi được tệp đến máy nhận")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message := readRemoteError(response)
		a.addTransferEvent("file", "sent", "Gửi tệp thất bại", name+" · "+message, "failed")
		writeError(w, http.StatusBadGateway, message)
		return
	}
	a.addTransferEvent("file", "sent", "Đã gửi tệp", fmt.Sprintf("%s đến %s", name, peer.Name), "success")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name})
}

func (a *App) handleReceiveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	if announced, err := strconv.ParseInt(r.Header.Get("X-Truyen-Tep-Size"), 10, 64); err == nil && announced > MaxFileBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Tệp lớn quá 4 GB")
		return
	}
	expected, aead, noncePrefix, err := a.receiveCipher(r, MaxFileBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := sanitizeFilename(r.URL.Query().Get("name"))
	reader, writer := io.Pipe()
	go func() {
		_, cryptErr := decryptStream(writer, r.Body, aead, noncePrefix, expected)
		_ = writer.CloseWithError(cryptErr)
	}()
	a.fileMu.Lock()
	path, size, err := saveIncomingFile(a.cfg.DownloadDir, name, reader)
	a.fileMu.Unlock()
	if err != nil {
		var maxError *http.MaxBytesError
		if errors.As(err, &maxError) {
			writeError(w, http.StatusRequestEntityTooLarge, "Tệp lớn quá 4 GB")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	savedName := filepath.Base(path)
	sender := senderName(r)
	a.addTransferEvent("file", "received", "Đã nhận tệp", fmt.Sprintf("%s · %s · từ %s", savedName, humanBytes(size), sender), "success")
	a.notifier.Notify("Truyền Tệp", "Đã nhận "+savedName+" từ "+sender)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": savedName, "bytes": size})
}

func (a *App) handleManualPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	var input struct {
		Address string `json:"address"`
	}
	if !readJSON(w, r, &input, 8*1024) {
		return
	}
	host, port, err := parseManualAddress(input.Address, a.cfg.Port)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	pingURL := (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, strconv.Itoa(port)), Path: "/api/ping"}).String()
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
	response, err := a.client.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Không kết nối được địa chỉ này")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("X-Truyen-Tep-Protocol") != strconv.Itoa(ProtocolVersion) {
		writeError(w, http.StatusBadGateway, "Địa chỉ này không chạy Truyền Tệp")
		return
	}
	var info struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Port      int    `json:"port"`
		PublicKey string `json:"publicKey"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 32*1024)).Decode(&info); err != nil || !validDeviceID(info.ID) || strings.TrimSpace(info.Name) == "" {
		writeError(w, http.StatusBadGateway, "Thông tin máy nhận không hợp lệ")
		return
	}
	if info.ID == a.cfg.DeviceID {
		writeError(w, http.StatusBadRequest, "Đây là địa chỉ của chính máy này")
		return
	}
	peer := Peer{ID: info.ID, Name: strings.TrimSpace(info.Name), IP: host, Port: port, LastSeen: time.Now(), Manual: true, PublicKey: info.PublicKey, Fingerprint: keyFingerprint(info.PublicKey)}
	a.peers.Upsert(peer)
	writeJSON(w, http.StatusOK, peer)
}

func (a *App) handleOpenDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	if err := os.MkdirAll(a.cfg.DownloadDir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, "Không tạo được thư mục nhận tệp")
		return
	}
	if err := OpenFolder(a.cfg.DownloadDir); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleQuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Phương thức không được hỗ trợ")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	a.quitOnce.Do(func() {
		time.AfterFunc(150*time.Millisecond, func() { close(a.quitCh) })
	})
}

func (a *App) addTransferEvent(kind, direction, title, detail, status string) {
	a.events.Add(Event{Kind: kind, Direction: direction, Title: title, Detail: detail, Status: status})
}

func setPeerHeaders(request *http.Request, cfg Config) {
	request.Header.Set("X-Truyen-Tep-Protocol", strconv.Itoa(ProtocolVersion))
	request.Header.Set("X-Truyen-Tep-Sender-Id", cfg.DeviceID)
	request.Header.Set("X-Truyen-Tep-Sender-Name", safeHeader(cfg.Name))
}

func setEncryptionHeaders(request *http.Request, ephemeral string, salt, noncePrefix []byte, plainSize int64) {
	request.Header.Set("X-Truyen-Tep-Key", ephemeral)
	request.Header.Set("X-Truyen-Tep-Salt", base64.RawStdEncoding.EncodeToString(salt))
	request.Header.Set("X-Truyen-Tep-Nonce", base64.RawStdEncoding.EncodeToString(noncePrefix))
	request.Header.Set("X-Truyen-Tep-Size", strconv.FormatInt(plainSize, 10))
}

func (a *App) receiveCipher(r *http.Request, maximum int64) (int64, cipher.AEAD, []byte, error) {
	expected, err := strconv.ParseInt(r.Header.Get("X-Truyen-Tep-Size"), 10, 64)
	if err != nil || expected < 0 || expected > maximum {
		return 0, nil, nil, errors.New("kích thước dữ liệu không hợp lệ")
	}
	salt, err := base64.RawStdEncoding.DecodeString(r.Header.Get("X-Truyen-Tep-Salt"))
	if err != nil || len(salt) != 32 {
		return 0, nil, nil, errors.New("tham số mã hóa không hợp lệ")
	}
	nonce, err := base64.RawStdEncoding.DecodeString(r.Header.Get("X-Truyen-Tep-Nonce"))
	if err != nil || len(nonce) != 8 {
		return 0, nil, nil, errors.New("tham số mã hóa không hợp lệ")
	}
	aead, err := cipherFromSender(a.cfg.PrivateKey, r.Header.Get("X-Truyen-Tep-Key"), salt)
	if err != nil {
		return 0, nil, nil, errors.New("không mở được khóa phiên")
	}
	return expected, aead, nonce, nil
}

func safeHeader(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}

func senderName(r *http.Request) string {
	name := strings.TrimSpace(safeHeader(r.Header.Get("X-Truyen-Tep-Sender-Name")))
	if name == "" {
		return "một máy khác"
	}
	runes := []rune(name)
	if len(runes) > 80 {
		name = string(runes[:80])
	}
	return name
}

func peerURL(peer Peer, path string, query url.Values) string {
	return (&url.URL{
		Scheme:   "http",
		Host:     net.JoinHostPort(peer.IP, strconv.Itoa(peer.Port)),
		Path:     path,
		RawQuery: query.Encode(),
	}).String()
}

func parseManualAddress(raw string, defaultPort int) (string, int, error) {
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(raw, "http://"), "https://"))
	raw = strings.TrimSuffix(raw, "/")
	if raw == "" {
		return "", 0, fmt.Errorf("Hãy nhập địa chỉ IP của máy nhận")
	}
	host := raw
	port := defaultPort
	if strings.Contains(raw, ":") {
		parsedHost, parsedPort, err := net.SplitHostPort(raw)
		if err != nil {
			return "", 0, fmt.Errorf("Địa chỉ phải có dạng 192.168.1.20 hoặc 192.168.1.20:8777")
		}
		host = parsedHost
		value, err := strconv.Atoi(parsedPort)
		if err != nil || value <= 0 || value > 65535 {
			return "", 0, fmt.Errorf("Cổng kết nối không hợp lệ")
		}
		port = value
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil || ip.To4() == nil || !isTrustedLANIP(ip) {
		return "", 0, fmt.Errorf("Chỉ chấp nhận địa chỉ IPv4 trong mạng nội bộ")
	}
	return ip.String(), port, nil
}

func readJSON(w http.ResponseWriter, r *http.Request, output any, maxBytes int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		var maxError *http.MaxBytesError
		if errors.As(err, &maxError) {
			writeError(w, http.StatusRequestEntityTooLarge, "Dữ liệu gửi lên quá lớn")
		} else {
			writeError(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		}
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return false
	}
	return true
}

func readRemoteError(response *http.Response) string {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&payload); err == nil && payload.Error != "" {
		return payload.Error
	}
	return "Máy nhận từ chối dữ liệu"
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func humanBytes(size int64) string {
	const unit = int64(1024)
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	divisor, exponent := unit, 0
	for value := size / unit; value >= unit && exponent < 3; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(divisor), "KMGT"[exponent])
}
