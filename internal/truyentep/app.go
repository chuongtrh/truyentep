package truyentep

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

var Version = "dev"

type App struct {
	cfg       Config
	clipboard Clipboard
	notifier  Notifier
	peers     *PeerStore
	events    *EventStore
	client    *http.Client
	startedAt time.Time
	quitCh    chan struct{}
	quitOnce  sync.Once
	fileMu    sync.Mutex

	discoveryMu    sync.RWMutex
	discoveryError string
}

func New(cfg Config) (*App, error) {
	return newApp(cfg, SystemClipboard{}, SystemNotifier{})
}

func newApp(cfg Config, clipboard Clipboard, notifier Notifier) (*App, error) {
	prepared, err := prepareConfig(cfg)
	if err != nil {
		return nil, err
	}
	if clipboard == nil {
		clipboard = SystemClipboard{}
	}
	if notifier == nil {
		notifier = SystemNotifier{}
	}

	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: 4 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          16,
		IdleConnTimeout:       60 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableCompression:    true,
	}
	return &App{
		cfg:       prepared,
		clipboard: clipboard,
		notifier:  notifier,
		peers:     NewPeerStore(),
		events:    NewEventStore(),
		client:    &http.Client{Transport: transport},
		startedAt: time.Now(),
		quitCh:    make(chan struct{}),
	}, nil
}

func (a *App) Config() Config {
	return a.cfg
}

func (a *App) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", a.cfg.Port)
}

func (a *App) Handler() http.Handler {
	return a.routes()
}

func (a *App) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", a.cfg.Port))
	if err != nil {
		if a.cfg.Port != 0 && isChuyenRunning(a.cfg.Port) {
			return ErrAlreadyRunning
		}
		return fmt.Errorf("không mở được cổng %d: %w", a.cfg.Port, err)
	}
	defer listener.Close()
	a.cfg.Port = listener.Addr().(*net.TCPAddr).Port

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !a.cfg.NoDiscovery {
		discovery := NewDiscovery(a.cfg.DeviceID, a.cfg.Name, a.cfg.PublicKey, a.cfg.Port, a.cfg.DiscoveryPort, a.peers)
		go func() {
			if err := discovery.Run(runCtx); err != nil && runCtx.Err() == nil {
				a.setDiscoveryError(err.Error())
			}
		}()
	} else {
		a.setDiscoveryError("Đã tắt tìm máy tự động")
	}

	server := &http.Server{
		Handler:           a.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    32 * 1024,
	}
	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	if !a.cfg.NoOpen {
		go func() {
			timer := time.NewTimer(250 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-runCtx.Done():
			case <-timer.C:
				_ = OpenBrowser(a.URL())
			}
		}()
	}

	var runErr error
	select {
	case <-ctx.Done():
	case <-a.quitCh:
	case runErr = <-serveErr:
	}
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	if runErr != nil {
		return fmt.Errorf("máy chủ nội bộ bị ngắt: %w", runErr)
	}
	return nil
}

func (a *App) setDiscoveryError(message string) {
	a.discoveryMu.Lock()
	a.discoveryError = message
	a.discoveryMu.Unlock()
}

func (a *App) getDiscoveryError() string {
	a.discoveryMu.RLock()
	defer a.discoveryMu.RUnlock()
	return a.discoveryError
}

func (a *App) peerList() []Peer {
	peers := a.peers.List()
	sort.Slice(peers, func(i, j int) bool {
		if peers[i].Name == peers[j].Name {
			return peers[i].IP < peers[j].IP
		}
		return peers[i].Name < peers[j].Name
	})
	return peers
}

func isChuyenRunning(port int) bool {
	client := &http.Client{Timeout: 700 * time.Millisecond}
	response, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/api/ping")
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK && response.Header.Get("X-Truyen-Tep-Protocol") == strconv.Itoa(ProtocolVersion)
}
