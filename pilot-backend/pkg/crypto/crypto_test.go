package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

const testKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" // 32 bytes, base64

func TestNewEncryptor(t *testing.T) {
	if _, err := NewEncryptor(testKey); err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}
}

func TestNewEncryptor_InvalidBase64(t *testing.T) {
	if _, err := NewEncryptor("not-valid-base64!!!"); err == nil {
		t.Fatal("expected an error for invalid base64, got nil")
	}
}

func TestNewEncryptor_WrongKeyLength(t *testing.T) {
	shortKey := base64.StdEncoding.EncodeToString([]byte("too-short"))
	_, err := NewEncryptor(shortKey)
	if err == nil {
		t.Fatal("expected an error for a non-32-byte key, got nil")
	}
	if !strings.Contains(err.Error(), "32 bytes") {
		t.Errorf("error = %v, want it to mention the 32-byte requirement", err)
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	enc, err := NewEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	const plaintext = "super-secret-oauth-token"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext equals plaintext — not actually encrypted")
	}
	if ciphertext == "" {
		t.Fatal("ciphertext is empty")
	}

	got, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got != plaintext {
		t.Errorf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	// AES-GCM must use a fresh random nonce per call, so encrypting the
	// same plaintext twice must not produce identical ciphertext.
	enc, err := NewEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	a, err := enc.Encrypt("same-plaintext")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	b, err := enc.Encrypt("same-plaintext")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if a == b {
		t.Error("two encryptions of the same plaintext produced identical ciphertext (nonce reuse?)")
	}
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	enc, err := NewEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	ciphertext, err := enc.Encrypt("authentic data")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("decode ciphertext error = %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // flip the last byte
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := enc.Decrypt(tampered); err == nil {
		t.Fatal("expected GCM authentication to reject tampered ciphertext, got nil error")
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	encA, err := NewEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}
	otherKey := base64.StdEncoding.EncodeToString([]byte("zyxwvutsrqponmlkjihgfedcba098765"))
	encB, err := NewEncryptor(otherKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	ciphertext, err := encA.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if _, err := encB.Decrypt(ciphertext); err == nil {
		t.Fatal("expected decryption with the wrong key to fail, got nil error")
	}
}

func TestDecrypt_MalformedInput(t *testing.T) {
	enc, err := NewEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	tests := map[string]string{
		"not base64":        "not-valid-base64!!!",
		"too short for GCM": base64.StdEncoding.EncodeToString([]byte("x")),
		"empty":             "",
	}
	for name, ciphertext := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := enc.Decrypt(ciphertext); err == nil {
				t.Errorf("Decrypt(%q) error = nil, want an error", ciphertext)
			}
		})
	}
}
