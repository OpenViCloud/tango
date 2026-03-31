package traefik

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"tango/internal/domain"
)

// FileProvider writes Traefik dynamic-configuration YAML files so that routing
// changes take effect immediately without restarting containers.
// Traefik must be configured with --providers.file.directory pointing to configDir
// and --providers.file.watch=true.
type FileProvider struct {
	configDir string
}

func NewFileProvider(configDir string) *FileProvider {
	return &FileProvider{configDir: configDir}
}

// ── YAML schema ───────────────────────────────────────────────────────────────

type traefikDynamic struct {
	HTTP traefikHTTP `yaml:"http"`
}

type traefikHTTP struct {
	Routers     map[string]traefikRouter     `yaml:"routers,omitempty"`
	Services    map[string]traefikService    `yaml:"services,omitempty"`
	Middlewares map[string]traefikMiddleware `yaml:"middlewares,omitempty"`
}

type traefikRouter struct {
	Rule        string      `yaml:"rule"`
	EntryPoints []string    `yaml:"entryPoints"`
	Service     string      `yaml:"service"`
	TLS         *traefikTLS `yaml:"tls,omitempty"`
	Middlewares []string    `yaml:"middlewares,omitempty"`
}

type traefikTLS struct {
	CertResolver string `yaml:"certResolver,omitempty"`
}

type traefikService struct {
	LoadBalancer traefikLB `yaml:"loadBalancer"`
}

type traefikLB struct {
	Servers []traefikServer `yaml:"servers"`
}

type traefikServer struct {
	URL string `yaml:"url"`
}

type traefikMiddleware struct {
	RedirectScheme *traefikRedirectScheme `yaml:"redirectScheme,omitempty"`
}

type traefikRedirectScheme struct {
	Scheme    string `yaml:"scheme"`
	Permanent bool   `yaml:"permanent"`
}

// ── Public API ────────────────────────────────────────────────────────────────

// Write generates a Traefik config file for the given resource.
// Each domain gets its own router. Auto domains are always HTTP only.
// Verified domains use per-domain TLSEnabled and TargetPort to determine
// whether to add TLS + certResolver (HTTPS with HTTP→HTTPS redirect) or plain HTTP.
// containerName is the Docker container name resolved via Docker DNS on tango_net.
func (p *FileProvider) Write(resourceID string, domains []*domain.ResourceDomain, containerName string, certResolver string) error {
	// Use first 12 chars of stripped resource ID as stable name base
	routerBase := "r-" + strings.ReplaceAll(resourceID, "-", "")[:12]

	cfg := traefikDynamic{
		HTTP: traefikHTTP{
			Routers: map[string]traefikRouter{},
			Services: map[string]traefikService{},
		},
	}

	hasRoutes := false
	for i, d := range domains {
		routerName := fmt.Sprintf("%s-%d", routerBase, i)
		svcName := fmt.Sprintf("%s-%d-svc", routerBase, i)
		hostRule := fmt.Sprintf("Host(`%s`)", d.Host)
		targetPort := d.TargetPort
		if targetPort <= 0 {
			targetPort = 80
		}
		cfg.HTTP.Services[svcName] = traefikService{
			LoadBalancer: traefikLB{
				Servers: []traefikServer{{URL: fmt.Sprintf("http://%s:%d", containerName, targetPort)}},
			},
		}

		if d.Type == domain.ResourceDomainTypeAuto {
			// Auto domains: HTTP only (localhost / internal names cannot get TLS certs)
			cfg.HTTP.Routers[routerName] = traefikRouter{
				Rule:        hostRule,
				EntryPoints: []string{"web"},
				Service:     svcName,
			}
			hasRoutes = true
		} else if d.Verified {
			// Custom verified domains: per-domain TLS decision
			if d.TLSEnabled && certResolver != "" {
				mw := routerName + "-redirect"
				if cfg.HTTP.Middlewares == nil {
					cfg.HTTP.Middlewares = map[string]traefikMiddleware{}
				}
				cfg.HTTP.Middlewares[mw] = traefikMiddleware{
					RedirectScheme: &traefikRedirectScheme{Scheme: "https", Permanent: true},
				}
				cfg.HTTP.Routers[routerName+"-http"] = traefikRouter{
					Rule:        hostRule,
					EntryPoints: []string{"web"},
					Service:     svcName,
					Middlewares: []string{mw},
				}
				cfg.HTTP.Routers[routerName] = traefikRouter{
					Rule:        hostRule,
					EntryPoints: []string{"websecure"},
					Service:     svcName,
					TLS:         &traefikTLS{CertResolver: certResolver},
				}
			} else {
				cfg.HTTP.Routers[routerName] = traefikRouter{
					Rule:        hostRule,
					EntryPoints: []string{"web"},
					Service:     svcName,
				}
			}
			hasRoutes = true
		}
	}

	if !hasRoutes {
		return p.Delete(resourceID)
	}

	return p.writeFile(filepath.Join(p.configDir, "resource-"+resourceID+".yaml"), cfg)
}

// Delete removes the Traefik config file for the resource.
func (p *FileProvider) Delete(resourceID string) error {
	path := filepath.Join(p.configDir, "resource-"+resourceID+".yaml")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove traefik config for resource %s: %w", resourceID, err)
	}
	return nil
}

// WriteAppConfig generates the Traefik config for the Tango app itself.
// backendURL is the app's internal address, e.g. "http://app:8080".
func (p *FileProvider) WriteAppConfig(appDomain string, tlsEnabled bool, certResolver string, backendURL string) error {
	if appDomain == "" || backendURL == "" {
		return p.DeleteAppConfig()
	}

	rule := fmt.Sprintf("Host(`%s`)", appDomain)
	cfg := traefikDynamic{
		HTTP: traefikHTTP{
			Routers: map[string]traefikRouter{},
			Services: map[string]traefikService{
				"tango-svc": {
					LoadBalancer: traefikLB{
						Servers: []traefikServer{{URL: backendURL}},
					},
				},
			},
		},
	}

	if tlsEnabled && certResolver != "" {
		mw := "tango-redirect"
		cfg.HTTP.Middlewares = map[string]traefikMiddleware{
			mw: {RedirectScheme: &traefikRedirectScheme{Scheme: "https", Permanent: true}},
		}
		cfg.HTTP.Routers["tango-http"] = traefikRouter{
			Rule:        rule,
			EntryPoints: []string{"web"},
			Service:     "tango-svc",
			Middlewares: []string{mw},
		}
		cfg.HTTP.Routers["tango"] = traefikRouter{
			Rule:        rule,
			EntryPoints: []string{"websecure"},
			Service:     "tango-svc",
			TLS:         &traefikTLS{CertResolver: certResolver},
		}
	} else {
		cfg.HTTP.Routers["tango"] = traefikRouter{
			Rule:        rule,
			EntryPoints: []string{"web"},
			Service:     "tango-svc",
		}
	}

	return p.writeFile(filepath.Join(p.configDir, "tango-app.yaml"), cfg)
}

// DeleteAppConfig removes the Tango app's Traefik config file.
func (p *FileProvider) DeleteAppConfig() error {
	path := filepath.Join(p.configDir, "tango-app.yaml")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove traefik app config: %w", err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (p *FileProvider) writeFile(path string, cfg traefikDynamic) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create traefik config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal traefik config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
