package truyentep

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ProtocolVersion      = 2
	DefaultPort          = 8777
	DefaultDiscoveryPort = 47777
	MaxClipboardBytes    = 1 << 20 // 1 MiB
	MaxFileBytes         = int64(4) << 30
	peerTimeout          = 9 * time.Second
	maxEvents            = 30
)

var (
	ErrAlreadyRunning       = errors.New("Truyền Tệp đang chạy")
	ErrClipboardUnsupported = errors.New("clipboard chỉ được hỗ trợ trên macOS")
)

type Config struct {
	Name          string
	DeviceID      string
	Port          int
	DiscoveryPort int
	DownloadDir   string
	NoOpen        bool
	NoDiscovery   bool
	PrivateKey    []byte `json:"-"`
	PublicKey     string `json:"-"`
}

type Clipboard interface {
	Read(context.Context) (string, error)
	Write(context.Context, string) error
}

type Notifier interface {
	Notify(title, message string)
}

type Peer struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IP          string    `json:"ip"`
	Port        int       `json:"port"`
	LastSeen    time.Time `json:"lastSeen"`
	Manual      bool      `json:"manual"`
	PublicKey   string    `json:"publicKey"`
	Fingerprint string    `json:"fingerprint"`
}

type Event struct {
	ID        uint64    `json:"id"`
	Kind      string    `json:"kind"`
	Direction string    `json:"direction"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail"`
	Status    string    `json:"status"`
	At        time.Time `json:"at"`
}

type PeerStore struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

func NewPeerStore() *PeerStore {
	return &PeerStore{peers: make(map[string]Peer)}
}

func (s *PeerStore) Upsert(peer Peer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.peers[peer.ID]; ok && current.Manual {
		peer.Manual = true
	}
	s.peers[peer.ID] = peer
}

func (s *PeerStore) Get(id string) (Peer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	peer, ok := s.peers[id]
	if !ok || (!peer.Manual && time.Since(peer.LastSeen) > peerTimeout) {
		return Peer{}, false
	}
	return peer, true
}

func (s *PeerStore) List() []Peer {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Peer, 0, len(s.peers))
	for id, peer := range s.peers {
		if !peer.Manual && now.Sub(peer.LastSeen) > peerTimeout {
			delete(s.peers, id)
			continue
		}
		items = append(items, peer)
	}
	return items
}

type EventStore struct {
	mu     sync.RWMutex
	events []Event
	nextID atomic.Uint64
}

func NewEventStore() *EventStore {
	return &EventStore{}
}

func (s *EventStore) Add(event Event) Event {
	event.ID = s.nextID.Add(1)
	if event.At.IsZero() {
		event.At = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append([]Event{event}, s.events...)
	if len(s.events) > maxEvents {
		s.events = s.events[:maxEvents]
	}
	return event
}

func (s *EventStore) List() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Event, len(s.events))
	copy(items, s.events)
	return items
}
