// Package crypto provides AES-256-GCM authenticated encryption for sensitive
// values (e.g. OAuth tokens) before they are persisted to the database.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encryptor encrypts and decrypts strings using AES-256-GCM with a 32-byte key.
type Encryptor struct {
	key []byte
}

// NewEncryptor builds an Encryptor from a base64-encoded 32-byte (AES-256) key.
func NewEncryptor(keyBase64 string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes (AES-256)")
	}
	return &Encryptor{key: key}, nil
}

// Encrypt returns the base64-encoded nonce+ciphertext for plaintext.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt. It returns an error if the ciphertext is
// malformed or fails authentication (tampered, wrong key, etc).
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aead.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext2 := data[:nonceSize], data[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext2, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
