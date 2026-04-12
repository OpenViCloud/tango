package command

import (
	"context"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"

	"github.com/google/uuid"
)

type CreateCloudflareConnectionCommand struct {
	UserID      string
	DisplayName string
	AccountID   string
	ZoneID      string
	APIToken    string
}

type CreateCloudflareConnectionHandler struct {
	repo      domain.CloudflareConnectionRepository
	cipher    appservices.SecretCipher
	cfFactory domain.CloudflareClientFactory
}

func NewCreateCloudflareConnectionHandler(
	repo domain.CloudflareConnectionRepository,
	cipher appservices.SecretCipher,
	cfFactory domain.CloudflareClientFactory,
) *CreateCloudflareConnectionHandler {
	return &CreateCloudflareConnectionHandler{
		repo:      repo,
		cipher:    cipher,
		cfFactory: cfFactory,
	}
}

func (h *CreateCloudflareConnectionHandler) Handle(ctx context.Context, cmd CreateCloudflareConnectionCommand) (*domain.CloudflareConnection, error) {
	client := h.cfFactory.New(strings.TrimSpace(cmd.APIToken), strings.TrimSpace(cmd.AccountID), strings.TrimSpace(cmd.ZoneID))
	if err := client.VerifyAccess(ctx); err != nil {
		return nil, err
	}
	encrypted, err := h.cipher.Encrypt(ctx, strings.TrimSpace(cmd.APIToken))
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	item, err := domain.NewCloudflareConnection(
		uuid.NewString(),
		cmd.UserID,
		cmd.DisplayName,
		cmd.AccountID,
		cmd.ZoneID,
		encrypted,
	)
	if err != nil {
		return nil, err
	}
	return h.repo.Save(ctx, item)
}

type UpdateCloudflareConnectionCommand struct {
	UserID      string
	ID          string
	DisplayName string
	AccountID   string
	ZoneID      string
	APIToken    string
}

type UpdateCloudflareConnectionHandler struct {
	repo      domain.CloudflareConnectionRepository
	cipher    appservices.SecretCipher
	cfFactory domain.CloudflareClientFactory
}

func NewUpdateCloudflareConnectionHandler(
	repo domain.CloudflareConnectionRepository,
	cipher appservices.SecretCipher,
	cfFactory domain.CloudflareClientFactory,
) *UpdateCloudflareConnectionHandler {
	return &UpdateCloudflareConnectionHandler{
		repo:      repo,
		cipher:    cipher,
		cfFactory: cfFactory,
	}
}

func (h *UpdateCloudflareConnectionHandler) Handle(ctx context.Context, cmd UpdateCloudflareConnectionCommand) (*domain.CloudflareConnection, error) {
	item, err := h.repo.GetByID(ctx, strings.TrimSpace(cmd.ID))
	if err != nil {
		return nil, err
	}
	if item.UserID != strings.TrimSpace(cmd.UserID) {
		return nil, domain.ErrCloudflareConnectionNotFound
	}
	apiTokenEncrypted := item.APITokenEncrypted
	if strings.TrimSpace(cmd.APIToken) != "" {
		encrypted, err := h.cipher.Encrypt(ctx, strings.TrimSpace(cmd.APIToken))
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		apiTokenEncrypted = encrypted
	}
	apiTokenPlain, err := h.cipher.Decrypt(ctx, apiTokenEncrypted)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	client := h.cfFactory.New(apiTokenPlain, strings.TrimSpace(cmd.AccountID), strings.TrimSpace(cmd.ZoneID))
	if err := client.VerifyAccess(ctx); err != nil {
		return nil, err
	}
	if err := item.Update(cmd.DisplayName, cmd.AccountID, cmd.ZoneID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.APIToken) != "" {
		if err := item.ReplaceEncryptedToken(apiTokenEncrypted); err != nil {
			return nil, err
		}
	}
	return h.repo.Update(ctx, item)
}
