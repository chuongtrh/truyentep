package truyentep

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"testing"
)

func TestDiscoveryAcceptsValidLANPeer(t *testing.T) {
	store := NewPeerStore()
	publicKey := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	discovery := NewDiscovery(stringsRepeat("1", 32), "Máy A", publicKey, 8777, 47777, store)
	payload, err := json.Marshal(discoveryMessage{
		Magic: discoveryMagic, Version: ProtocolVersion,
		ID: stringsRepeat("2", 32), Name: "Máy B", Port: 9000, PublicKey: publicKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	discovery.accept(payload, &net.UDPAddr{IP: net.ParseIP("192.168.1.20"), Port: 50123})
	peer, ok := store.Get(stringsRepeat("2", 32))
	if !ok {
		t.Fatal("không thêm máy hợp lệ")
	}
	if peer.Name != "Máy B" || peer.IP != "192.168.1.20" || peer.Port != 9000 {
		t.Fatalf("máy nhận sai: %#v", peer)
	}
}

func TestDiscoveryRejectsInvalidMessages(t *testing.T) {
	store := NewPeerStore()
	publicKey := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	discovery := NewDiscovery(stringsRepeat("1", 32), "Máy A", publicKey, 8777, 47777, store)
	cases := []discoveryMessage{
		{Magic: "khac", Version: ProtocolVersion, ID: stringsRepeat("2", 32), Name: "B", Port: 8777, PublicKey: publicKey},
		{Magic: discoveryMagic, Version: 999, ID: stringsRepeat("2", 32), Name: "B", Port: 8777},
		{Magic: discoveryMagic, Version: ProtocolVersion, ID: "invalid", Name: "B", Port: 8777},
		{Magic: discoveryMagic, Version: ProtocolVersion, ID: stringsRepeat("2", 32), Name: "", Port: 8777},
		{Magic: discoveryMagic, Version: ProtocolVersion, ID: stringsRepeat("2", 32), Name: "B", Port: 0},
	}
	for _, message := range cases {
		payload, _ := json.Marshal(message)
		discovery.accept(payload, &net.UDPAddr{IP: net.ParseIP("192.168.1.20"), Port: 50123})
	}
	if len(store.List()) != 0 {
		t.Fatalf("đã nhận thông điệp không hợp lệ: %#v", store.List())
	}
}

func stringsRepeat(value string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += value
	}
	return result
}
