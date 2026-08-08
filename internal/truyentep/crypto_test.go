package truyentep

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestEncryptedStreamRoundTripAndTamperDetection(t *testing.T) {
	receiver, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	peerKey := encodePublicKey(receiver.PublicKey().Bytes())
	aead, ephemeral, salt, err := newCipherForPeer(peerKey)
	if err != nil {
		t.Fatal(err)
	}
	nonce := []byte("12345678")
	plain := bytes.Repeat([]byte("dữ liệu kín · "), 9000)
	var encrypted bytes.Buffer
	if _, err := encryptStream(&encrypted, bytes.NewReader(plain), aead, nonce); err != nil {
		t.Fatal(err)
	}
	receiverAEAD, err := cipherFromSender(receiver.Bytes(), ephemeral, salt)
	if err != nil {
		t.Fatal(err)
	}
	var opened bytes.Buffer
	if _, err := decryptStream(&opened, bytes.NewReader(encrypted.Bytes()), receiverAEAD, nonce, int64(len(plain))); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened.Bytes(), plain) {
		t.Fatal("nội dung sau giải mã không khớp")
	}

	tampered := append([]byte(nil), encrypted.Bytes()...)
	tampered[len(tampered)-1] ^= 1
	if _, err := decryptStream(&bytes.Buffer{}, bytes.NewReader(tampered), receiverAEAD, nonce, int64(len(plain))); err == nil {
		t.Fatal("đã chấp nhận dữ liệu mã hóa bị sửa")
	}
}

func encodePublicKey(raw []byte) string { return base64.RawStdEncoding.EncodeToString(raw) }
