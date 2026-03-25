package suiauth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/blake2b"
)

const (
	// flagEd25519 is the Sui signature scheme flag for Ed25519.
	flagEd25519 = 0x00

	// serializedSigLen is the expected length of a Sui serialized Ed25519 signature:
	// 1 (flag) + 64 (signature) + 32 (public key) = 97 bytes.
	serializedSigLen = 97
)

// Verifier verifies a Sui wallet signature and returns the derived address.
type Verifier interface {
	Verify(message, serializedSig string) (address string, err error)
}

type verifier struct{}

// New returns a production Verifier.
func New() Verifier {
	return &verifier{}
}

// MessageForAction returns the canonical message the client must sign.
// resourceID is empty for upload (no pre-existing resource).
func MessageForAction(action, resourceID string, timestamp int64) string {
	return fmt.Sprintf("paylock:%s:%s:%d", action, resourceID, timestamp)
}

// Verify parses a base64-encoded Sui serialized signature, verifies the Ed25519
// signature over the personal-message intent digest of message, and returns the
// derived Sui address.
func (v *verifier) Verify(message, serializedSig string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(serializedSig)
	if err != nil {
		return "", errors.New("invalid signature encoding")
	}

	if len(raw) != serializedSigLen {
		return "", fmt.Errorf("invalid signature length: expected %d, got %d", serializedSigLen, len(raw))
	}

	flag := raw[0]
	if flag != flagEd25519 {
		return "", fmt.Errorf("unsupported signature scheme: 0x%02x", flag)
	}

	sig := raw[1:65]
	pubkey := raw[65:97]

	digest := personalMessageDigest([]byte(message))

	if !ed25519.Verify(ed25519.PublicKey(pubkey), digest, sig) {
		return "", errors.New("signature verification failed")
	}

	addr := deriveAddress(flag, pubkey)
	return addr, nil
}

// personalMessageDigest computes the Blake2b-256 hash of the personal message
// with Sui intent bytes prepended: [0x03, 0x00, 0x00] || BCS(message).
func personalMessageDigest(msg []byte) []byte {
	// Intent bytes: scope=PersonalMessage(3), version=V0(0), appId=Sui(0)
	intent := []byte{0x03, 0x00, 0x00}

	// BCS encoding of a byte vector: ULEB128 length prefix + raw bytes
	bcsMsg := append(uleb128(uint64(len(msg))), msg...)

	data := append(intent, bcsMsg...)
	h, _ := blake2b.New256(nil)
	h.Write(data)
	return h.Sum(nil)
}

// deriveAddress computes the Sui address from a flag byte and public key:
// Blake2b-256([flag] || pubkey), hex-encoded with 0x prefix.
func deriveAddress(flag byte, pubkey []byte) string {
	h, _ := blake2b.New256(nil)
	h.Write([]byte{flag})
	h.Write(pubkey)
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

// uleb128 encodes a uint64 as a ULEB128 byte sequence.
func uleb128(v uint64) []byte {
	if v == 0 {
		return []byte{0}
	}
	var buf []byte
	for v > 0 {
		b := byte(v & 0x7f)
		v >>= 7
		if v > 0 {
			b |= 0x80
		}
		buf = append(buf, b)
	}
	return buf
}

// NormalizeAddress lowercases and ensures 0x prefix for consistent comparison.
func NormalizeAddress(addr string) string {
	return strings.ToLower(addr)
}
