package truyentep

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClipboard struct {
	mu   sync.Mutex
	text string
}

func (c *fakeClipboard) Read(context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.text, nil
}

func (c *fakeClipboard) Write(_ context.Context, text string) error {
	c.mu.Lock()
	c.text = text
	c.mu.Unlock()
	return nil
}

type quietNotifier struct{}

func (quietNotifier) Notify(string, string) {}

func testApp(t *testing.T, name, id, dir string, clipboard Clipboard) *App {
	t.Helper()
	app, err := newApp(Config{
		Name:          name,
		DeviceID:      id,
		Port:          8777,
		DiscoveryPort: 47777,
		DownloadDir:   dir,
		NoOpen:        true,
		NoDiscovery:   true,
		PrivateKey:    bytes.Repeat([]byte{id[0]}, 32),
	}, clipboard, quietNotifier{})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestClipboardTransferEndToEnd(t *testing.T) {
	receiverClipboard := &fakeClipboard{}
	receiver := testApp(t, "Mac nhận", strings.Repeat("2", 32), t.TempDir(), receiverClipboard)
	receiverServer := httptest.NewServer(receiver.Handler())
	defer receiverServer.Close()

	senderClipboard := &fakeClipboard{text: "Xin chào từ clipboard"}
	sender := testApp(t, "Mac gửi", strings.Repeat("1", 32), t.TempDir(), senderClipboard)
	addServerPeer(t, sender, receiverServer.URL, receiver.cfg)

	body := strings.NewReader(`{"peerId":"` + receiver.cfg.DeviceID + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/send/clipboard", body)
	request.RemoteAddr = "127.0.0.1:55000"
	request.Header.Set("X-Truyen-Tep-Local", "1")
	response := httptest.NewRecorder()
	sender.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("trạng thái = %d, nội dung = %s", response.Code, response.Body.String())
	}
	receiverClipboard.mu.Lock()
	got := receiverClipboard.text
	receiverClipboard.mu.Unlock()
	if got != "Xin chào từ clipboard" {
		t.Fatalf("clipboard nhận = %q", got)
	}
	if len(sender.events.List()) != 1 || len(receiver.events.List()) != 1 {
		t.Fatalf("lịch sử gửi=%d nhận=%d", len(sender.events.List()), len(receiver.events.List()))
	}
}

func TestFileTransferEndToEnd(t *testing.T) {
	receiveDir := t.TempDir()
	receiver := testApp(t, "Mac nhận", strings.Repeat("4", 32), receiveDir, &fakeClipboard{})
	receiverServer := httptest.NewServer(receiver.Handler())
	defer receiverServer.Close()

	sender := testApp(t, "Mac gửi", strings.Repeat("3", 32), t.TempDir(), &fakeClipboard{})
	addServerPeer(t, sender, receiverServer.URL, receiver.cfg)

	content := []byte("nội dung tệp văn bản")
	query := url.Values{"peer": {receiver.cfg.DeviceID}, "name": {"../ghi chú.txt"}}
	request := httptest.NewRequest(http.MethodPost, "/api/send/file?"+query.Encode(), bytes.NewReader(content))
	request.RemoteAddr = "127.0.0.1:55000"
	request.Header.Set("X-Truyen-Tep-Local", "1")
	request.Header.Set("Content-Type", "application/octet-stream")
	response := httptest.NewRecorder()
	sender.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("trạng thái = %d, nội dung = %s", response.Code, response.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(receiveDir, "ghi chú.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("nội dung nhận = %q", data)
	}
}

func TestReceiveRejectsPublicNetwork(t *testing.T) {
	app := testApp(t, "Mac", strings.Repeat("5", 32), t.TempDir(), &fakeClipboard{})
	request := httptest.NewRequest(http.MethodPost, "/api/receive/clipboard", strings.NewReader(`{"text":"x"}`))
	request.RemoteAddr = "8.8.8.8:1234"
	request.Header.Set("X-Truyen-Tep-Protocol", strconv.Itoa(ProtocolVersion))
	request.Header.Set("X-Truyen-Tep-Size", strconv.FormatInt(MaxFileBytes+1, 10))
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("trạng thái = %d", response.Code)
	}
}

func TestLocalControlRequiresLoopbackAndHeader(t *testing.T) {
	app := testApp(t, "Mac", strings.Repeat("6", 32), t.TempDir(), &fakeClipboard{})

	request := httptest.NewRequest(http.MethodPost, "/api/quit", nil)
	request.RemoteAddr = "192.168.1.9:1234"
	request.Header.Set("X-Truyen-Tep-Local", "1")
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("điều khiển từ xa: trạng thái = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/quit", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	response = httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("thiếu đầu mục bảo vệ: trạng thái = %d", response.Code)
	}
}

func TestParseManualAddress(t *testing.T) {
	host, port, err := parseManualAddress("192.168.1.20", 8777)
	if err != nil || host != "192.168.1.20" || port != 8777 {
		t.Fatalf("host=%q port=%d err=%v", host, port, err)
	}
	host, port, err = parseManualAddress("http://10.0.0.2:9000/", 8777)
	if err != nil || host != "10.0.0.2" || port != 9000 {
		t.Fatalf("host=%q port=%d err=%v", host, port, err)
	}
	if _, _, err := parseManualAddress("8.8.8.8:8777", 8777); err == nil {
		t.Fatal("đã chấp nhận địa chỉ công cộng")
	}
}

func TestStateDoesNotExposeClipboardContents(t *testing.T) {
	clipboard := &fakeClipboard{text: "bí mật"}
	app := testApp(t, "Mac", strings.Repeat("7", 32), t.TempDir(), clipboard)
	request := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatal(response.Code)
	}
	if strings.Contains(response.Body.String(), "bí mật") {
		t.Fatal("trạng thái làm lộ nội dung clipboard")
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
}

func addServerPeer(t *testing.T, app *App, serverURL string, remote Config) {
	t.Helper()
	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	host, rawPort, found := strings.Cut(parsed.Host, ":")
	if !found {
		t.Fatal("máy chủ kiểm thử thiếu cổng")
	}
	port, conversionErr := strconv.Atoi(rawPort)
	if conversionErr != nil {
		t.Fatal(conversionErr)
	}
	app.peers.Upsert(Peer{ID: remote.DeviceID, Name: remote.Name, IP: host, Port: port, LastSeen: time.Now(), Manual: true, PublicKey: remote.PublicKey, Fingerprint: keyFingerprint(remote.PublicKey)})
}

func TestMaxFileBytesResponse(t *testing.T) {
	app := testApp(t, "Mac", strings.Repeat("8", 32), t.TempDir(), &fakeClipboard{})
	request := httptest.NewRequest(http.MethodPost, "/api/receive/file?name=big.zip", io.NopCloser(strings.NewReader("x")))
	request.RemoteAddr = "192.168.1.2:1234"
	request.ContentLength = MaxFileBytes + 1
	request.Header.Set("X-Truyen-Tep-Protocol", strconv.Itoa(ProtocolVersion))
	request.Header.Set("X-Truyen-Tep-Size", strconv.FormatInt(MaxFileBytes+1, 10))
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("trạng thái = %d", response.Code)
	}
}
