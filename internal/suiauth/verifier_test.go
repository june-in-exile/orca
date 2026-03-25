package suiauth

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/blake2b"
)

// buildTestSig creates a valid Sui serialized signature for a given message
// using the provided Ed25519 key pair.
func buildTestSig(privkey ed25519.PrivateKey, pubkey ed25519.PublicKey, message string) string {
	digest := personalMessageDigest([]byte(message))
	sig := ed25519.Sign(privkey, digest)

	// Sui serialized format: [flag(1)] [sig(64)] [pubkey(32)]
	serialized := make([]byte, 0, serializedSigLen)
	serialized = append(serialized, flagEd25519)
	serialized = append(serialized, sig...)
	serialized = append(serialized, pubkey...)
	return base64.StdEncoding.EncodeToString(serialized)
}

// testAddress derives the expected Sui address from a public key.
func testAddress(pubkey ed25519.PublicKey) string {
	h, _ := blake2b.New256(nil)
	h.Write([]byte{flagEd25519})
	h.Write(pubkey)
	return "0x" + encodeHex(h.Sum(nil))
}

func encodeHex(b []byte) string {
	const hextable = "0123456789abcdef"
	dst := make([]byte, len(b)*2)
	for i, v := range b {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}

func TestVerify_ValidSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	msg := MessageForAction("upload", "", 1711368000)
	sig := buildTestSig(priv, pub, msg)

	v := New()
	addr, err := v.Verify(msg, sig)
	if err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}

	expected := testAddress(pub)
	if addr != expected {
		t.Fatalf("address mismatch: got %s, want %s", addr, expected)
	}
}

func TestVerify_WrongMessage(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	sig := buildTestSig(priv, pub, "paylock:upload::1711368000")

	v := New()
	_, err = v.Verify("paylock:delete:vid-001:1711368000", sig)
	if err == nil {
		t.Fatal("expected error for wrong message, got nil")
	}
}

func TestVerify_MalformedBase64(t *testing.T) {
	v := New()
	_, err := v.Verify("test", "not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for malformed base64")
	}
}

func TestVerify_WrongLength(t *testing.T) {
	v := New()
	short := base64.StdEncoding.EncodeToString(make([]byte, 50))
	_, err := v.Verify("test", short)
	if err == nil {
		t.Fatal("expected error for wrong length")
	}
}

func TestVerify_UnsupportedScheme(t *testing.T) {
	// flag byte 0x01 = Secp256k1
	raw := make([]byte, serializedSigLen)
	raw[0] = 0x01
	sig := base64.StdEncoding.EncodeToString(raw)

	v := New()
	_, err := v.Verify("test", sig)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestMessageForAction(t *testing.T) {
	tests := []struct {
		action, resource string
		ts               int64
		want             string
	}{
		{"upload", "", 1711368000, "paylock:upload::1711368000"},
		{"delete", "vid-abc", 1711368000, "paylock:delete:vid-abc:1711368000"},
		{"update", "vid-xyz", 9999999999, "paylock:update:vid-xyz:9999999999"},
	}

	for _, tt := range tests {
		got := MessageForAction(tt.action, tt.resource, tt.ts)
		if got != tt.want {
			t.Errorf("MessageForAction(%q, %q, %d) = %q, want %q", tt.action, tt.resource, tt.ts, got, tt.want)
		}
	}
}

func TestVerify_DeleteAction(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	msg := MessageForAction("delete", "vid-001", 1711368000)
	sig := buildTestSig(priv, pub, msg)

	v := New()
	addr, err := v.Verify(msg, sig)
	if err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}

	expected := testAddress(pub)
	if addr != expected {
		t.Fatalf("address mismatch: got %s, want %s", addr, expected)
	}
}
