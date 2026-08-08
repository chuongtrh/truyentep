package truyentep

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	discoveryMagic = "truyen-tep-lan"
	multicastIP    = "239.255.77.77"
)

type discoveryMessage struct {
	Magic     string `json:"magic"`
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Port      int    `json:"port"`
	PublicKey string `json:"publicKey"`
}

type Discovery struct {
	self  discoveryMessage
	port  int
	peers *PeerStore
}

func NewDiscovery(id, name, publicKey string, httpPort, discoveryPort int, peers *PeerStore) *Discovery {
	return &Discovery{
		self: discoveryMessage{
			Magic:     discoveryMagic,
			Version:   ProtocolVersion,
			ID:        id,
			Name:      name,
			Port:      httpPort,
			PublicKey: publicKey,
		},
		port:  discoveryPort,
		peers: peers,
	}
}

func (d *Discovery) Run(ctx context.Context) error {
	group, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", multicastIP, d.port))
	if err != nil {
		return fmt.Errorf("địa chỉ tìm máy không hợp lệ: %w", err)
	}
	receiver, err := net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		return fmt.Errorf("không mở được kênh tìm máy: %w", err)
	}
	defer receiver.Close()
	_ = receiver.SetReadBuffer(64 * 1024)

	sender, err := net.DialUDP("udp4", nil, group)
	if err != nil {
		return fmt.Errorf("không phát được tín hiệu tìm máy: %w", err)
	}
	defer sender.Close()

	payload, err := json.Marshal(d.self)
	if err != nil {
		return fmt.Errorf("không tạo được tín hiệu tìm máy: %w", err)
	}

	go d.announce(ctx, sender, payload)
	go func() {
		<-ctx.Done()
		_ = receiver.Close()
		_ = sender.Close()
	}()

	buffer := make([]byte, 2048)
	for {
		n, source, err := receiver.ReadFromUDP(buffer)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("kênh tìm máy bị ngắt: %w", err)
		}
		d.accept(buffer[:n], source)
	}
}

func (d *Discovery) announce(ctx context.Context, conn *net.UDPConn, payload []byte) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		_, _ = conn.Write(payload)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *Discovery) accept(payload []byte, source *net.UDPAddr) {
	if source == nil || source.IP.To4() == nil || !isTrustedLANIP(source.IP) {
		return
	}
	var message discoveryMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return
	}
	message.Name = strings.TrimSpace(message.Name)
	if message.Magic != discoveryMagic || message.Version != ProtocolVersion ||
		message.ID == "" || message.ID == d.self.ID || !validDeviceID(message.ID) ||
		message.Name == "" || len([]rune(message.Name)) > 80 ||
		message.Port <= 0 || message.Port > 65535 || keyFingerprint(message.PublicKey) == "" {
		return
	}
	d.peers.Upsert(Peer{
		ID:          message.ID,
		Name:        message.Name,
		IP:          source.IP.String(),
		Port:        message.Port,
		LastSeen:    time.Now(),
		PublicKey:   message.PublicKey,
		Fingerprint: keyFingerprint(message.PublicKey),
	})
}
