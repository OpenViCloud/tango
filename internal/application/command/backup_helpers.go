package command

import (
	"context"
	"encoding/json"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

func encryptConnectionSecrets(ctx context.Context, cipher appservices.SecretCipher, password string, uri string) (string, string, error) {
	var encryptedPassword string
	if password != "" {
		value, err := cipher.Encrypt(ctx, password)
		if err != nil {
			return "", "", domain.ErrInvalidInput
		}
		encryptedPassword = value
	}
	var encryptedURI string
	if uri != "" {
		value, err := cipher.Encrypt(ctx, uri)
		if err != nil {
			return "", "", domain.ErrInvalidInput
		}
		encryptedURI = value
	}
	return encryptedPassword, encryptedURI, nil
}

func encryptJSONMap(ctx context.Context, cipher appservices.SecretCipher, value map[string]any) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return cipher.Encrypt(ctx, string(raw))
}
