package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// VaultSecretBackend reads and writes secrets via HashiCorp Vault KV v2.
type VaultSecretBackend struct {
	addr   string // Vault address, e.g. "https://vault.example.com"
	token  string // Vault token
	mount  string // KV mount path, e.g. "secret"
	prefix string // path prefix within the mount
	client *http.Client
}

// NewVaultSecretBackend returns a VaultSecretBackend using the given Vault address,
// token, KV mount, and key prefix.
func NewVaultSecretBackend(addr, token, mount, prefix string) *VaultSecretBackend {
	return &VaultSecretBackend{
		addr:   strings.TrimRight(addr, "/"),
		token:  token,
		mount:  mount,
		prefix: prefix,
		client: &http.Client{},
	}
}

func (v *VaultSecretBackend) kvURL(name string) string {
	path := v.prefix + "/" + name
	return fmt.Sprintf("%s/v1/%s/data/%s", v.addr, v.mount, path)
}

func (v *VaultSecretBackend) Get(ctx context.Context, name string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.kvURL(name), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", v.token)

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault get %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("secret %q not found in vault", name)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("vault get %s: status %d", name, resp.StatusCode)
	}

	var result struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("vault decode: %w", err)
	}
	val, ok := result.Data.Data["value"]
	if !ok {
		return "", fmt.Errorf("vault secret %s has no 'value' field", name)
	}
	return val, nil
}

func (v *VaultSecretBackend) Set(ctx context.Context, name, value string) error {
	payload, _ := json.Marshal(map[string]any{
		"data": map[string]string{"value": value},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.kvURL(name), strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", v.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("vault set %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault set %s: status %d: %s", name, resp.StatusCode, body)
	}
	return nil
}

func (v *VaultSecretBackend) Delete(ctx context.Context, name string) error {
	// KV v2 delete metadata permanently removes all versions.
	metaURL := fmt.Sprintf("%s/v1/%s/metadata/%s/%s", v.addr, v.mount, v.prefix, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, metaURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", v.token)
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("vault delete %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("vault delete %s: status %d", name, resp.StatusCode)
	}
	return nil
}
