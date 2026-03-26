package services

import "context"

// SecretCipher encrypts and decrypts secrets persisted by the application.
type SecretCipher interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	Decrypt(ctx context.Context, ciphertext string) (string, error)
}
