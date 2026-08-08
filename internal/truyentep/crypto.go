package truyentep

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const encryptedChunkSize = 64 * 1024

func keyFingerprint(encoded string) string {
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(raw) != 32 {
		return ""
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%X-%X-%X-%X", sum[0:2], sum[2:4], sum[4:6], sum[6:8])
}

func newCipherForPeer(peerPublic string) (cipher.AEAD, string, []byte, error) {
	remoteRaw, err := base64.RawStdEncoding.DecodeString(peerPublic)
	if err != nil {
		return nil, "", nil, errors.New("khóa máy nhận không hợp lệ")
	}
	remote, err := ecdh.X25519().NewPublicKey(remoteRaw)
	if err != nil {
		return nil, "", nil, errors.New("khóa máy nhận không hợp lệ")
	}
	ephemeral, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", nil, err
	}
	secret, err := ephemeral.ECDH(remote)
	if err != nil {
		return nil, "", nil, err
	}
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, "", nil, err
	}
	aead, err := makeAEAD(hkdfSHA256(secret, salt, []byte("TruyenTep/v2/AES-256-GCM"), 32))
	return aead, base64.RawStdEncoding.EncodeToString(ephemeral.PublicKey().Bytes()), salt, err
}

func cipherFromSender(privateRaw []byte, ephemeralEncoded string, salt []byte) (cipher.AEAD, error) {
	private, err := ecdh.X25519().NewPrivateKey(privateRaw)
	if err != nil {
		return nil, err
	}
	raw, err := base64.RawStdEncoding.DecodeString(ephemeralEncoded)
	if err != nil {
		return nil, err
	}
	public, err := ecdh.X25519().NewPublicKey(raw)
	if err != nil {
		return nil, err
	}
	secret, err := private.ECDH(public)
	if err != nil {
		return nil, err
	}
	return makeAEAD(hkdfSHA256(secret, salt, []byte("TruyenTep/v2/AES-256-GCM"), 32))
}

func makeAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func hkdfSHA256(secret, salt, info []byte, length int) []byte {
	extract := hmac.New(sha256.New, salt)
	_, _ = extract.Write(secret)
	prk := extract.Sum(nil)
	result, previous := make([]byte, 0, length), []byte(nil)
	for counter := byte(1); len(result) < length; counter++ {
		expand := hmac.New(sha256.New, prk)
		_, _ = expand.Write(previous)
		_, _ = expand.Write(info)
		_, _ = expand.Write([]byte{counter})
		previous = expand.Sum(nil)
		result = append(result, previous...)
	}
	return result[:length]
}

func encryptStream(dst io.Writer, src io.Reader, aead cipher.AEAD, noncePrefix []byte) (int64, error) {
	buf := make([]byte, encryptedChunkSize)
	var total int64
	var counter uint32
	for {
		n, readErr := io.ReadFull(src, buf)
		if errors.Is(readErr, io.ErrUnexpectedEOF) || errors.Is(readErr, io.EOF) {
			if n == 0 && total > 0 {
				return total, nil
			}
		} else if readErr != nil {
			return total, readErr
		}
		nonce := makeNonce(noncePrefix, counter)
		aad := makeAAD(counter, uint32(n))
		sealed := aead.Seal(nil, nonce, buf[:n], aad)
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], uint32(n))
		if _, err := dst.Write(header[:]); err != nil {
			return total, err
		}
		if _, err := dst.Write(sealed); err != nil {
			return total, err
		}
		total += int64(n)
		counter++
		if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
			return total, nil
		}
	}
}

func decryptStream(dst io.Writer, src io.Reader, aead cipher.AEAD, noncePrefix []byte, expected int64) (int64, error) {
	var total int64
	var counter uint32
	for total < expected {
		var header [4]byte
		if _, err := io.ReadFull(src, header[:]); err != nil {
			return total, errors.New("dữ liệu mã hóa bị thiếu")
		}
		n := binary.BigEndian.Uint32(header[:])
		if n > encryptedChunkSize || int64(n) > expected-total {
			return total, errors.New("khối mã hóa không hợp lệ")
		}
		sealed := make([]byte, int(n)+aead.Overhead())
		if _, err := io.ReadFull(src, sealed); err != nil {
			return total, errors.New("dữ liệu mã hóa bị thiếu")
		}
		plain, err := aead.Open(nil, makeNonce(noncePrefix, counter), sealed, makeAAD(counter, n))
		if err != nil {
			return total, errors.New("không xác thực được dữ liệu mã hóa")
		}
		if _, err := dst.Write(plain); err != nil {
			return total, err
		}
		total += int64(n)
		counter++
	}
	return total, nil
}

func makeNonce(prefix []byte, counter uint32) []byte {
	nonce := make([]byte, 12)
	copy(nonce, prefix)
	binary.BigEndian.PutUint32(nonce[8:], counter)
	return nonce
}
func makeAAD(counter, size uint32) []byte {
	var aad [8]byte
	binary.BigEndian.PutUint32(aad[:4], counter)
	binary.BigEndian.PutUint32(aad[4:], size)
	return aad[:]
}
