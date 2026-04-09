package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"time"

	"tango/internal/domain"
	appservices "tango/internal/application/services"

	"golang.org/x/crypto/ssh"
)

const (
	platformKeyPrivate = "ssh.private_key_enc"
	platformKeyPublic  = "ssh.public_key"
)

// Manager handles Tango's global SSH keypair and SSH connectivity tests.
// The private key is stored encrypted (AES-GCM) in platform_configs table.
type Manager struct {
	configRepo domain.PlatformConfigRepository
	cipher     appservices.SecretCipher
}

func NewManager(configRepo domain.PlatformConfigRepository, cipher appservices.SecretCipher) *Manager {
	return &Manager{configRepo: configRepo, cipher: cipher}
}

// EnsureKeypair generates and stores an ed25519 keypair if one doesn't exist yet.
func (m *Manager) EnsureKeypair(ctx context.Context) error {
	existing, err := m.configRepo.Get(ctx, platformKeyPublic)
	if err != nil && err != domain.ErrPlatformConfigNotFound {
		return fmt.Errorf("check ssh keypair: %w", err)
	}
	if existing != nil && existing.Value != "" {
		return nil // already exists
	}
	return m.generateAndStore(ctx)
}

// PublicKey returns the Tango SSH public key in authorized_keys format.
func (m *Manager) PublicKey(ctx context.Context) (string, error) {
	cfg, err := m.configRepo.Get(ctx, platformKeyPublic)
	if err != nil {
		return "", fmt.Errorf("get ssh public key: %w", err)
	}
	return cfg.Value, nil
}

// PrivateKeyPEM returns the decrypted PEM-encoded private key.
func (m *Manager) PrivateKeyPEM(ctx context.Context) ([]byte, error) {
	cfg, err := m.configRepo.Get(ctx, platformKeyPrivate)
	if err != nil {
		return nil, fmt.Errorf("get ssh private key: %w", err)
	}
	plain, err := m.cipher.Decrypt(ctx, cfg.Value)
	if err != nil {
		return nil, fmt.Errorf("decrypt ssh private key: %w", err)
	}
	return []byte(plain), nil
}

// Ping dials the server over SSH using Tango's keypair and returns nil on success.
func (m *Manager) Ping(ctx context.Context, server *domain.Server) error {
	privPEM, err := m.PrivateKeyPEM(ctx)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(privPEM)
	if err != nil {
		return fmt.Errorf("parse ssh private key: %w", err)
	}

	sshUser := server.SSHUser
	if sshUser == "" {
		sshUser = "root"
	}
	sshPort := server.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	cfg := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec — user controls the server IPs
		Timeout:         10 * time.Second,
	}

	dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", server.PublicIP, sshPort)
	conn, err := dialWithContext(dialCtx, "tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	conn.Close()
	return nil
}

// generateAndStore creates a new ed25519 keypair and persists it.
func (m *Manager) generateAndStore(ctx context.Context) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate ed25519 key: %w", err)
	}

	// Encode private key as OpenSSH PEM
	privPEM, err := ssh.MarshalPrivateKey(priv, "tango-cloud")
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	privPEMBytes := pem.EncodeToMemory(privPEM)

	// Encode public key in authorized_keys format
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return fmt.Errorf("create ssh public key: %w", err)
	}
	pubKeyStr := string(ssh.MarshalAuthorizedKey(sshPub))

	// Encrypt private key before storing
	privEnc, err := m.cipher.Encrypt(ctx, string(privPEMBytes))
	if err != nil {
		return fmt.Errorf("encrypt private key: %w", err)
	}

	if err := m.configRepo.Set(ctx, platformKeyPrivate, privEnc); err != nil {
		return fmt.Errorf("store private key: %w", err)
	}
	if err := m.configRepo.Set(ctx, platformKeyPublic, pubKeyStr); err != nil {
		return fmt.Errorf("store public key: %w", err)
	}
	return nil
}

// dialWithContext wraps ssh.Dial with context cancellation support.
func dialWithContext(ctx context.Context, network, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	d := net.Dialer{Timeout: cfg.Timeout}
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}
