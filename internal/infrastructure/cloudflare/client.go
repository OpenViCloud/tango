package cloudflare

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"tango/internal/domain"
)

const baseURL = "https://api.cloudflare.com/client/v4"

// Client implements domain.CloudflareClient using the Cloudflare REST API.
type Client struct {
	apiToken  string
	accountID string
	zoneID    string
	http      *http.Client
}

// New creates a new Cloudflare API client.
func New(apiToken, accountID, zoneID string) *Client {
	return &Client{
		apiToken:  apiToken,
		accountID: accountID,
		zoneID:    zoneID,
		http:      &http.Client{},
	}
}

// ── domain.CloudflareClient ────────────────────────────────────────────────────

// CreateTunnel provisions a new Named Tunnel and returns its credentials.
func (c *Client) CreateTunnel(ctx context.Context, name string) (*domain.CloudflareTunnel, error) {
	body, _ := json.Marshal(map[string]any{"name": name, "tunnel_secret": randomSecret()})
	var resp struct {
		Result struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			TunnelSecret string `json:"tunnel_secret"`
			Token        string `json:"token"`
		} `json:"result"`
		Errors []cfError `json:"errors"`
	}
	if err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/accounts/%s/cfd_tunnel", c.accountID),
		body, &resp); err != nil {
		return nil, fmt.Errorf("create tunnel: %w", err)
	}
	return &domain.CloudflareTunnel{
		ID:    resp.Result.ID,
		Name:  resp.Result.Name,
		Token: resp.Result.Token,
	}, nil
}

// DeleteTunnel deletes the tunnel and cleans up its connectors.
func (c *Client) DeleteTunnel(ctx context.Context, tunnelID string) error {
	// Force-delete any active connections first.
	_ = c.do(ctx, http.MethodDelete,
		fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/connections", c.accountID, tunnelID),
		nil, nil)

	if err := c.do(ctx, http.MethodDelete,
		fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", c.accountID, tunnelID),
		nil, nil); err != nil {
		return fmt.Errorf("delete tunnel: %w", err)
	}
	return nil
}

// CreateCNAMERecord creates a proxied CNAME: hostname → <tunnelID>.cfargotunnel.com.
func (c *Client) CreateCNAMERecord(ctx context.Context, hostname, tunnelID string) error {
	body, _ := json.Marshal(map[string]any{
		"type":    "CNAME",
		"name":    hostname,
		"content": tunnelID + ".cfargotunnel.com",
		"proxied": true,
		"ttl":     1, // automatic
	})
	if err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/zones/%s/dns_records", c.zoneID),
		body, nil); err != nil {
		return fmt.Errorf("create cname record: %w", err)
	}
	return nil
}

// DeleteDNSRecord removes the CNAME record for the given hostname.
func (c *Client) DeleteDNSRecord(ctx context.Context, hostname string) error {
	id, err := c.findDNSRecordID(ctx, hostname)
	if err != nil {
		return err
	}
	if id == "" {
		return nil // already gone
	}
	if err := c.do(ctx, http.MethodDelete,
		fmt.Sprintf("/zones/%s/dns_records/%s", c.zoneID, id),
		nil, nil); err != nil {
		return fmt.Errorf("delete dns record: %w", err)
	}
	return nil
}

// VerifyAccess validates that the configured API token can access the account and zone.
func (c *Client) VerifyAccess(ctx context.Context) error {
	var accountResp struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/accounts/%s", c.accountID), nil, &accountResp); err != nil {
		return fmt.Errorf("verify account access: %w", err)
	}
	if accountResp.Result.ID == "" {
		return fmt.Errorf("verify account access: account not found")
	}

	var zoneResp struct {
		Result struct {
			ID      string `json:"id"`
			Account struct {
				ID string `json:"id"`
			} `json:"account"`
		} `json:"result"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/zones/%s", c.zoneID), nil, &zoneResp); err != nil {
		return fmt.Errorf("verify zone access: %w", err)
	}
	if zoneResp.Result.ID == "" {
		return fmt.Errorf("verify zone access: zone not found")
	}
	if zoneResp.Result.Account.ID != "" && zoneResp.Result.Account.ID != c.accountID {
		return fmt.Errorf("verify zone access: zone does not belong to account")
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (c *Client) findDNSRecordID(ctx context.Context, hostname string) (string, error) {
	var resp struct {
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
		Errors []cfError `json:"errors"`
	}
	url := fmt.Sprintf("/zones/%s/dns_records?name=%s&type=CNAME", c.zoneID, hostname)
	if err := c.do(ctx, http.MethodGet, url, nil, &resp); err != nil {
		return "", fmt.Errorf("list dns records: %w", err)
	}
	if len(resp.Result) == 0 {
		return "", nil
	}
	return resp.Result[0].ID, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("cloudflare API %s %s → %d: %s", method, path, res.StatusCode, raw)
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// randomSecret generates a 32-byte random base64 string for the tunnel secret.
func randomSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
