package command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

const (
	cloudflaredImage     = "cloudflare/cloudflared:latest"
	cloudflaredNamespace = "default"
	cloudflaredName      = "tango-cloudflared"
	secretName           = "tango-cloudflared-token"
	configMapName        = "tango-cloudflared-config"
)

// ── ExposeService ─────────────────────────────────────────────────────────────

type ExposeServiceCommand struct {
	UserID       string
	ClusterID    string
	ConnectionID string
	Hostname     string // e.g. nginx.yourdomain.com
	ServiceURL   string // e.g. http://nginx-svc.default.svc.cluster.local:80
	Namespace    string // k8s namespace for cloudflared (defaults to "default")
}

type ImportClusterTunnelCommand struct {
	UserID       string
	ClusterID    string
	ConnectionID string
	TunnelID     string
	TunnelToken  string
	Namespace    string
	Overwrite    bool
}

type ExposeServiceHandler struct {
	clusterRepo    domain.ClusterRepository
	tunnelRepo     domain.ClusterTunnelRepository
	connectionRepo domain.CloudflareConnectionRepository
	kubeFactory    domain.KubeClientFactory
	cfFactory      domain.CloudflareClientFactory
	cipher         appservices.SecretCipher
}

func NewExposeServiceHandler(
	clusterRepo domain.ClusterRepository,
	tunnelRepo domain.ClusterTunnelRepository,
	connectionRepo domain.CloudflareConnectionRepository,
	kubeFactory domain.KubeClientFactory,
	cfFactory domain.CloudflareClientFactory,
	cipher appservices.SecretCipher,
) *ExposeServiceHandler {
	return &ExposeServiceHandler{
		clusterRepo:    clusterRepo,
		tunnelRepo:     tunnelRepo,
		connectionRepo: connectionRepo,
		kubeFactory:    kubeFactory,
		cfFactory:      cfFactory,
		cipher:         cipher,
	}
}

func (h *ExposeServiceHandler) Import(ctx context.Context, cmd ImportClusterTunnelCommand) (*domain.ClusterTunnel, error) {
	if cmd.Namespace == "" {
		cmd.Namespace = cloudflaredNamespace
	}
	if strings.TrimSpace(cmd.TunnelID) == "" || strings.TrimSpace(cmd.TunnelToken) == "" {
		return nil, domain.ErrInvalidInput
	}

	cluster, err := h.clusterRepo.GetByID(ctx, cmd.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("get cluster: %w", err)
	}
	if cluster.Status != domain.ClusterStatusReady {
		return nil, fmt.Errorf("cluster not ready (status: %s)", cluster.Status)
	}
	existing, err := h.tunnelRepo.GetByClusterID(ctx, cluster.ID)
	if err == nil && !cmd.Overwrite {
		return nil, domain.ErrClusterTunnelAlreadyExists
	} else if err != nil && !errors.Is(err, domain.ErrClusterTunnelNotFound) {
		return nil, fmt.Errorf("get cluster tunnel: %w", err)
	}

	kube, err := h.kubeFactory.GetClient(ctx, cmd.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("get kube client: %w", err)
	}
	tokenEnc, err := h.cipher.Encrypt(ctx, strings.TrimSpace(cmd.TunnelToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt tunnel token: %w", err)
	}
	if err := provisionTunnelRuntime(ctx, kube, cmd.Namespace, strings.TrimSpace(cmd.TunnelID), strings.TrimSpace(cmd.TunnelToken)); err != nil {
		return nil, err
	}
	connectionID := ""
	if strings.TrimSpace(cmd.ConnectionID) != "" {
		cfConnection, _, err := h.resolveConnectionClient(ctx, cmd.UserID, cmd.ConnectionID)
		if err != nil {
			return nil, err
		}
		connectionID = cfConnection.ID
	}

	tunnel := &domain.ClusterTunnel{
		ClusterID:              cluster.ID,
		CloudflareConnectionID: connectionID,
		TunnelID:               strings.TrimSpace(cmd.TunnelID),
		TokenEnc:               tokenEnc,
		Namespace:              cmd.Namespace,
	}
	if existing != nil {
		existing.CloudflareConnectionID = tunnel.CloudflareConnectionID
		existing.TunnelID = tunnel.TunnelID
		existing.TokenEnc = tunnel.TokenEnc
		existing.Namespace = tunnel.Namespace
		existing.Exposures = nil
		saved, err := h.tunnelRepo.Update(ctx, existing)
		if err != nil {
			return nil, fmt.Errorf("overwrite tunnel record: %w", err)
		}
		return saved, nil
	}
	saved, err := h.tunnelRepo.Save(ctx, tunnel)
	if err != nil {
		return nil, fmt.Errorf("save tunnel record: %w", err)
	}
	return saved, nil
}

func (h *ExposeServiceHandler) Handle(ctx context.Context, cmd ExposeServiceCommand) (*domain.ClusterTunnel, error) {
	if cmd.Namespace == "" {
		cmd.Namespace = cloudflaredNamespace
	}

	cluster, err := h.clusterRepo.GetByID(ctx, cmd.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("get cluster: %w", err)
	}
	if cluster.Status != domain.ClusterStatusReady {
		return nil, fmt.Errorf("cluster not ready (status: %s)", cluster.Status)
	}

	kube, err := h.kubeFactory.GetClient(ctx, cmd.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("get kube client: %w", err)
	}

	// ── 1. Get or create the cluster tunnel ───────────────────────────────────
	tunnel, cfClient, err := h.clusterTunnel(ctx, cluster, kube, cmd.UserID, cmd.ConnectionID, cmd.Namespace)
	if err != nil {
		return nil, err
	}

	// ── 2. Idempotency: skip if hostname already exposed ─────────────────────
	for _, e := range tunnel.Exposures {
		if e.Hostname == cmd.Hostname {
			return tunnel, nil
		}
	}

	// ── 3. Append new exposure and rebuild ConfigMap ──────────────────────────
	tunnel.Exposures = append(tunnel.Exposures, domain.TunnelExposure{
		Hostname:   cmd.Hostname,
		ServiceURL: cmd.ServiceURL,
		CreatedAt:  time.Now().UTC(),
	})

	if err := kube.CreateOrUpdateConfigMap(ctx, tunnel.Namespace, configMapName,
		buildConfigMapData(tunnel.TunnelID, tunnel.Exposures)); err != nil {
		return nil, fmt.Errorf("update configmap: %w", err)
	}

	// Trigger rolling restart so cloudflared picks up the new config.
	if err := kube.RolloutRestartDeployment(ctx, tunnel.Namespace, cloudflaredName); err != nil {
		return nil, fmt.Errorf("restart cloudflared: %w", err)
	}

	// ── 4. Create DNS CNAME ───────────────────────────────────────────────────
	if cfClient != nil {
		if err := cfClient.CreateCNAMERecord(ctx, cmd.Hostname, tunnel.TunnelID); err != nil {
			return nil, fmt.Errorf("create cname: %w", err)
		}
	}

	// ── 5. Persist ────────────────────────────────────────────────────────────
	updated, err := h.tunnelRepo.Update(ctx, tunnel)
	if err != nil {
		return nil, fmt.Errorf("persist tunnel: %w", err)
	}
	return updated, nil
}

// clusterTunnel returns the existing ClusterTunnel or creates a brand-new one
// (Cloudflare tunnel + k8s Secret + ConfigMap + Deployment).
func (h *ExposeServiceHandler) clusterTunnel(
	ctx context.Context,
	cluster *domain.Cluster,
	kube domain.KubeClient,
	userID string,
	connectionID string,
	namespace string,
) (*domain.ClusterTunnel, domain.CloudflareClient, error) {
	existing, err := h.tunnelRepo.GetByClusterID(ctx, cluster.ID)
	if err == nil {
		if strings.TrimSpace(existing.CloudflareConnectionID) == "" {
			return existing, nil, nil
		}
		cfConnection, cfClient, err := h.resolveConnectionClient(ctx, userID, existing.CloudflareConnectionID)
		if err != nil {
			return nil, nil, err
		}
		_ = cfConnection
		return existing, cfClient, nil
	}
	if !errors.Is(err, domain.ErrClusterTunnelNotFound) {
		return nil, nil, fmt.Errorf("get cluster tunnel: %w", err)
	}

	cfConnection, cfClient, err := h.resolveConnectionClient(ctx, userID, connectionID)
	if err != nil {
		return nil, nil, err
	}

	// First time: provision everything from scratch.
	cfTunnel, err := cfClient.CreateTunnel(ctx, "tango-"+cluster.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("create cloudflare tunnel: %w", err)
	}

	tokenEnc, err := h.cipher.Encrypt(ctx, cfTunnel.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt tunnel token: %w", err)
	}

	// Secret
	if err := provisionTunnelRuntime(ctx, kube, namespace, cfTunnel.ID, cfTunnel.Token); err != nil {
		return nil, nil, err
	}

	tunnel := &domain.ClusterTunnel{
		ClusterID:              cluster.ID,
		CloudflareConnectionID: cfConnection.ID,
		TunnelID:               cfTunnel.ID,
		TokenEnc:               tokenEnc,
		Namespace:              namespace,
	}
	saved, err := h.tunnelRepo.Save(ctx, tunnel)
	if err != nil {
		return nil, nil, fmt.Errorf("save tunnel record: %w", err)
	}
	return saved, cfClient, nil
}

// ── UnexposeService ───────────────────────────────────────────────────────────

type UnexposeServiceCommand struct {
	UserID    string
	ClusterID string
	Hostname  string
}

type UnexposeServiceHandler struct {
	tunnelRepo     domain.ClusterTunnelRepository
	connectionRepo domain.CloudflareConnectionRepository
	kubeFactory    domain.KubeClientFactory
	cfFactory      domain.CloudflareClientFactory
	cipher         appservices.SecretCipher
}

func NewUnexposeServiceHandler(
	tunnelRepo domain.ClusterTunnelRepository,
	connectionRepo domain.CloudflareConnectionRepository,
	kubeFactory domain.KubeClientFactory,
	cfFactory domain.CloudflareClientFactory,
	cipher appservices.SecretCipher,
) *UnexposeServiceHandler {
	return &UnexposeServiceHandler{
		tunnelRepo:     tunnelRepo,
		connectionRepo: connectionRepo,
		kubeFactory:    kubeFactory,
		cfFactory:      cfFactory,
		cipher:         cipher,
	}
}

func (h *UnexposeServiceHandler) Handle(ctx context.Context, cmd UnexposeServiceCommand) error {
	tunnel, err := h.tunnelRepo.GetByClusterID(ctx, cmd.ClusterID)
	if err != nil {
		return fmt.Errorf("get cluster tunnel: %w", err)
	}
	var cfClient domain.CloudflareClient
	if strings.TrimSpace(tunnel.CloudflareConnectionID) != "" {
		_, cfClient, err = h.resolveConnectionClient(ctx, cmd.UserID, tunnel.CloudflareConnectionID)
		if err != nil {
			return err
		}
	}

	filtered := tunnel.Exposures[:0]
	for _, e := range tunnel.Exposures {
		if e.Hostname != cmd.Hostname {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == len(tunnel.Exposures) {
		return nil // hostname not found, nothing to do
	}
	tunnel.Exposures = filtered

	kube, err := h.kubeFactory.GetClient(ctx, cmd.ClusterID)
	if err != nil {
		return fmt.Errorf("get kube client: %w", err)
	}

	if err := kube.CreateOrUpdateConfigMap(ctx, tunnel.Namespace, configMapName,
		buildConfigMapData(tunnel.TunnelID, tunnel.Exposures)); err != nil {
		return fmt.Errorf("update configmap: %w", err)
	}
	if err := kube.RolloutRestartDeployment(ctx, tunnel.Namespace, cloudflaredName); err != nil {
		return fmt.Errorf("restart cloudflared: %w", err)
	}
	if cfClient != nil {
		if err := cfClient.DeleteDNSRecord(ctx, cmd.Hostname); err != nil {
			return fmt.Errorf("delete dns record: %w", err)
		}
	}
	if _, err := h.tunnelRepo.Update(ctx, tunnel); err != nil {
		return fmt.Errorf("persist tunnel: %w", err)
	}
	return nil
}

func (h *ExposeServiceHandler) resolveConnectionClient(ctx context.Context, userID, connectionID string) (*domain.CloudflareConnection, domain.CloudflareClient, error) {
	return resolveConnectionClient(ctx, h.connectionRepo, h.cfFactory, h.cipher, userID, connectionID)
}

func (h *UnexposeServiceHandler) resolveConnectionClient(ctx context.Context, userID, connectionID string) (*domain.CloudflareConnection, domain.CloudflareClient, error) {
	return resolveConnectionClient(ctx, h.connectionRepo, h.cfFactory, h.cipher, userID, connectionID)
}

func resolveConnectionClient(
	ctx context.Context,
	connectionRepo domain.CloudflareConnectionRepository,
	cfFactory domain.CloudflareClientFactory,
	cipher appservices.SecretCipher,
	userID string,
	connectionID string,
) (*domain.CloudflareConnection, domain.CloudflareClient, error) {
	if strings.TrimSpace(connectionID) == "" {
		return nil, nil, domain.ErrClusterTunnelConnectionRequired
	}
	connection, err := connectionRepo.GetByID(ctx, strings.TrimSpace(connectionID))
	if err != nil {
		return nil, nil, err
	}
	if connection.UserID != strings.TrimSpace(userID) {
		return nil, nil, domain.ErrCloudflareConnectionNotFound
	}
	apiToken, err := cipher.Decrypt(ctx, connection.APITokenEncrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt cloudflare token: %w", err)
	}
	client := cfFactory.New(apiToken, connection.AccountID, connection.ZoneID)
	return connection, client, nil
}

func provisionTunnelRuntime(ctx context.Context, kube domain.KubeClient, namespace, tunnelID, tunnelToken string) error {
	if err := kube.CreateOrUpdateSecret(ctx, namespace, secretName, map[string]string{"token": tunnelToken}); err != nil {
		return fmt.Errorf("create secret: %w", err)
	}
	if err := kube.CreateOrUpdateConfigMap(ctx, namespace, configMapName, buildConfigMapData(tunnelID, nil)); err != nil {
		return fmt.Errorf("create configmap: %w", err)
	}
	if err := kube.CreateOrUpdateDeployment(ctx, namespace, domain.KubeDeploymentSpec{
		Name:            cloudflaredName,
		Namespace:       namespace,
		Replicas:        2,
		Labels:          map[string]string{"app": cloudflaredName},
		Image:           cloudflaredImage,
		Args:            []string{"tunnel", "--config", "/etc/cloudflared/config/config.yaml", "run", "--token", "$(TUNNEL_TOKEN)"},
		SecretEnv:       map[string]domain.KubeSecretKeyRef{"TUNNEL_TOKEN": {SecretName: secretName, Key: "token"}},
		ConfigMapMounts: map[string]string{configMapName: "/etc/cloudflared/config"},
	}); err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}
	return nil
}

// ── helper ────────────────────────────────────────────────────────────────────

// buildConfigMapData renders the cloudflared config.yaml content.
func buildConfigMapData(tunnelID string, exposures []domain.TunnelExposure) map[string]string {
	var sb strings.Builder
	sb.WriteString("tunnel: " + tunnelID + "\n")
	sb.WriteString("ingress:\n")
	for _, e := range exposures {
		sb.WriteString("  - hostname: " + e.Hostname + "\n")
		sb.WriteString("    service: " + e.ServiceURL + "\n")
	}
	sb.WriteString("  - service: http_status:404\n")
	return map[string]string{"config.yaml": sb.String()}
}
