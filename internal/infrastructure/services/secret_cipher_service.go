package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	appservices "tango/internal/application/services"
)

type aesSecretCipher struct {
	key []byte
}

// NewAESSecretCipher creates an AES-GCM secret cipher using a 32-byte key.
func NewAESSecretCipher(key string) (appservices.SecretCipher, error) {
	key = strings.TrimSpace(key)
	if len(key) != 32 {
		return nil, fmt.Errorf("LLM_CONFIG_ENCRYPTION_KEY must be exactly 32 characters")
	}

	return &aesSecretCipher{key: []byte(key)}, nil
}

func (c *aesSecretCipher) Encrypt(_ context.Context, plaintext string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *aesSecretCipher) Decrypt(_ context.Context, encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
