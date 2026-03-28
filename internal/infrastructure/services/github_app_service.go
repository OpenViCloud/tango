package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	appservices "tango/internal/application/services"

	"github.com/golang-jwt/jwt/v5"
)

type gitHubAppService struct {
	client *http.Client
}

func NewGitHubAppService() appservices.GitHubAppService {
	return &gitHubAppService{
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *gitHubAppService) BuildManifest(appName, redirectURL, setupURL, webhookURL string) appservices.GitHubAppManifest {
	callbackURL := strings.TrimRight(strings.TrimSpace(redirectURL), "/") + "/api/source-connections/github/callback"

	return appservices.GitHubAppManifest{
		Name:         strings.TrimSpace(appName),
		URL:          strings.TrimRight(strings.TrimSpace(redirectURL), "/"),
		RedirectURL:  callbackURL,
		CallbackURLs: []string{callbackURL},
		SetupURL:     strings.TrimSpace(setupURL),
		HookAttributes: appservices.GitHubHookAttributes{
			URL:    strings.TrimSpace(webhookURL),
			Active: true,
		},
		Public:        false,
		DefaultEvents: []string{"push", "pull_request"},
		DefaultPermissions: map[string]string{
			"contents":      "read",
			"metadata":      "read",
			"pull_requests": "read",
		},
	}
}

func (s *gitHubAppService) BuildCreateAppURL(state string) string {
	return "https://github.com/settings/apps/new?state=" + url.QueryEscape(strings.TrimSpace(state))
}

func (s *gitHubAppService) BuildInstallURL(appSlug, state string) string {
	values := url.Values{}
	values.Set("state", strings.TrimSpace(state))
	return "https://github.com/apps/" + strings.TrimSpace(appSlug) + "/installations/new?" + values.Encode()
}

func (s *gitHubAppService) ExchangeManifest(ctx context.Context, code string) (*appservices.GitHubAppRegistration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/app-manifests/"+url.PathEscape(strings.TrimSpace(code))+"/conversions", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	var payload struct {
		ID            int64  `json:"id"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		WebhookSecret string `json:"webhook_secret"`
		PEM           string `json:"pem"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		HTMLURL       string `json:"html_url"`
	}
	if err := s.doJSON(req, &payload); err != nil {
		return nil, err
	}
	return &appservices.GitHubAppRegistration{
		AppID:         payload.ID,
		ClientID:      payload.ClientID,
		ClientSecret:  payload.ClientSecret,
		WebhookSecret: payload.WebhookSecret,
		PrivateKeyPEM: payload.PEM,
		Name:          payload.Name,
		Slug:          payload.Slug,
		HTMLURL:       payload.HTMLURL,
	}, nil
}

func (s *gitHubAppService) GetInstallation(ctx context.Context, credentials appservices.GitHubAppCredentials, installationID int64) (*appservices.GitHubInstallation, error) {
	jwtToken, err := s.signAppJWT(credentials)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/app/installations/"+strconv.FormatInt(installationID, 10), nil)
	if err != nil {
		return nil, err
	}
	s.applyAppHeaders(req, jwtToken)

	var payload struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"account"`
	}
	if err := s.doJSON(req, &payload); err != nil {
		return nil, err
	}
	return &appservices.GitHubInstallation{
		ID:           payload.ID,
		AccountLogin: payload.Account.Login,
		AccountType:  payload.Account.Type,
	}, nil
}

func (s *gitHubAppService) CreateInstallationToken(ctx context.Context, credentials appservices.GitHubAppCredentials, installationID int64) (string, error) {
	jwtToken, err := s.signAppJWT(credentials)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/app/installations/"+strconv.FormatInt(installationID, 10)+"/access_tokens", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	s.applyAppHeaders(req, jwtToken)

	var payload struct {
		Token string `json:"token"`
	}
	if err := s.doJSON(req, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Token), nil
}

func (s *gitHubAppService) ListRepositories(ctx context.Context, installationToken string) ([]appservices.GitRepository, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/installation/repositories?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	s.applyInstallationHeaders(req, installationToken)

	var payload struct {
		Repositories []struct {
			Name          string `json:"name"`
			FullName      string `json:"full_name"`
			Private       bool   `json:"private"`
			DefaultBranch string `json:"default_branch"`
			CloneURL      string `json:"clone_url"`
			Owner         struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repositories"`
	}
	if err := s.doJSON(req, &payload); err != nil {
		return nil, err
	}

	items := make([]appservices.GitRepository, 0, len(payload.Repositories))
	for _, repo := range payload.Repositories {
		items = append(items, appservices.GitRepository{
			Owner:      repo.Owner.Login,
			Name:       repo.Name,
			FullName:   repo.FullName,
			CloneURL:   repo.CloneURL,
			Private:    repo.Private,
			DefaultRef: repo.DefaultBranch,
		})
	}
	return items, nil
}

func (s *gitHubAppService) ListUserRepositories(ctx context.Context, pat string) ([]appservices.GitRepository, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/repos?per_page=100&sort=updated", nil)
	if err != nil {
		return nil, err
	}
	s.applyInstallationHeaders(req, pat)

	var payload []struct {
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Private       bool   `json:"private"`
		DefaultBranch string `json:"default_branch"`
		CloneURL      string `json:"clone_url"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := s.doJSON(req, &payload); err != nil {
		return nil, err
	}

	items := make([]appservices.GitRepository, 0, len(payload))
	for _, repo := range payload {
		items = append(items, appservices.GitRepository{
			Owner:      repo.Owner.Login,
			Name:       repo.Name,
			FullName:   repo.FullName,
			CloneURL:   repo.CloneURL,
			Private:    repo.Private,
			DefaultRef: repo.DefaultBranch,
		})
	}
	return items, nil
}

func (s *gitHubAppService) ListBranches(ctx context.Context, installationToken, owner, repo string) ([]appservices.GitBranch, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(repo)+"/branches?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	s.applyInstallationHeaders(req, installationToken)

	var payload []struct {
		Name string `json:"name"`
	}
	if err := s.doJSON(req, &payload); err != nil {
		return nil, err
	}

	defaultBranch := ""
	repos, err := s.ListRepositories(ctx, installationToken)
	if err == nil {
		fullName := owner + "/" + repo
		for _, item := range repos {
			if item.FullName == fullName {
				defaultBranch = item.DefaultRef
				break
			}
		}
	}

	items := make([]appservices.GitBranch, 0, len(payload))
	for _, branch := range payload {
		items = append(items, appservices.GitBranch{Name: branch.Name, IsDefault: branch.Name == defaultBranch})
	}
	return items, nil
}

func (s *gitHubAppService) signAppJWT(credentials appservices.GitHubAppCredentials) (string, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(credentials.PrivateKeyPEM)))
	if block == nil {
		return "", fmt.Errorf("github app private key is invalid")
	}
	privateKeyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		privateKeyAny, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", err
		}
	}
	privateKey, ok := privateKeyAny.(*rsa.PrivateKey)
	if !ok {
		pkcs1Key, ok := privateKeyAny.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("github app private key is not RSA")
		}
		privateKey = pkcs1Key
	}
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": credentials.AppID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func (s *gitHubAppService) applyAppHeaders(req *http.Request, jwtToken string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(jwtToken))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (s *gitHubAppService) VerifyPAT(ctx context.Context, token string) (*appservices.GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	s.applyInstallationHeaders(req, token)

	var payload struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	}
	if err := s.doJSON(req, &payload); err != nil {
		return nil, fmt.Errorf("verify PAT: %w", err)
	}
	if payload.ID == 0 || payload.Login == "" {
		return nil, fmt.Errorf("verify PAT: unexpected empty response")
	}
	return &appservices.GitHubUser{ID: payload.ID, Login: payload.Login}, nil
}

func (s *gitHubAppService) applyInstallationHeaders(req *http.Request, installationToken string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(installationToken))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (s *gitHubAppService) doJSON(req *http.Request, dest any) error {
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github request failed: %s", strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return err
	}
	return nil
}

func newRandomState(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	}
	return prefix + fmt.Sprintf("%x", b[:])
}
