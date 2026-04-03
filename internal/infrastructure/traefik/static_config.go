package traefik

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// staticConfig mirrors the Traefik v3 static configuration YAML schema.
type staticConfig struct {
	API           staticAPI                         `yaml:"api"`
	Providers     staticProviders                   `yaml:"providers"`
	EntryPoints   map[string]staticEntryPoint       `yaml:"entryPoints"`
	Ping          *staticPing                       `yaml:"ping,omitempty"`
	CertResolvers map[string]staticCertResolver     `yaml:"certificatesResolvers,omitempty"`
}

type staticAPI struct {
	Dashboard bool `yaml:"dashboard"`
	Insecure  bool `yaml:"insecure"`
}

type staticProviders struct {
	Docker staticDockerProvider `yaml:"docker"`
	File   staticFileProvider   `yaml:"file"`
}

type staticDockerProvider struct {
	ExposedByDefault bool `yaml:"exposedByDefault"`
}

type staticFileProvider struct {
	Directory string `yaml:"directory"`
	Watch     bool   `yaml:"watch"`
}

type staticEntryPoint struct {
	Address string `yaml:"address"`
}

type staticPing struct{}

type staticCertResolver struct {
	ACME staticACME `yaml:"acme"`
}

type staticACME struct {
	Email         string              `yaml:"email"`
	Storage       string              `yaml:"storage"`
	HTTPChallenge staticHTTPChallenge `yaml:"httpChallenge"`
}

type staticHTTPChallenge struct {
	EntryPoint string `yaml:"entryPoint"`
}

// WriteStaticConfig writes /traefik/traefik.yml (parent of configDir).
// Passing acmeEmail="" generates a config without Let's Encrypt (ACME disabled).
// Passing a non-empty acmeEmail adds the letsencrypt cert resolver.
// Traefik must be restarted after this call for changes to take effect.
func (p *FileProvider) WriteStaticConfig(acmeEmail string) error {
	cfg := staticConfig{
		API: staticAPI{Dashboard: true, Insecure: false},
		Providers: staticProviders{
			Docker: staticDockerProvider{ExposedByDefault: false},
			File: staticFileProvider{
				Directory: p.configDir,
				Watch:     true,
			},
		},
		EntryPoints: map[string]staticEntryPoint{
			"web":       {Address: ":80"},
			"websecure": {Address: ":443"},
		},
		Ping: &staticPing{},
	}

	if acmeEmail != "" {
		cfg.CertResolvers = map[string]staticCertResolver{
			"letsencrypt": {
				ACME: staticACME{
					Email:   acmeEmail,
					Storage: "/letsencrypt/acme.json",
					HTTPChallenge: staticHTTPChallenge{
						EntryPoint: "web",
					},
				},
			},
		}
	}

	staticPath := filepath.Join(filepath.Dir(p.configDir), "traefik.yml")
	if err := os.MkdirAll(filepath.Dir(staticPath), 0o755); err != nil {
		return fmt.Errorf("create traefik static config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal traefik static config: %w", err)
	}
	return os.WriteFile(staticPath, data, 0o644)
}
