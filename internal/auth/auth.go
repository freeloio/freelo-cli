package auth

import (
	"errors"
	"fmt"
	"os"
)

// Provider is the interface for authentication strategies.
// Currently implements Basic Auth; designed for future OAuth extension.
type Provider interface {
	// GetCredentials returns (email, secret) for API authentication.
	GetCredentials() (string, string, error)
	// Store persists credentials.
	Store(email, secret string) error
	// Clear removes stored credentials.
	Clear() error
	// IsAuthenticated checks if valid credentials exist.
	IsAuthenticated() bool
}

// BasicAuth implements Provider using email + API key. The OS keyring is
// addressed under a service namespace that differs between prod (`freelo-cli`)
// and dev (`freelo-cli-dev`) so the two never collide in Keychain /
// Credential Manager / Secret Service.
type BasicAuth struct {
	keyring   Keyring
	namespace string // service name used in the keyring; "freelo-cli" or "freelo-cli-dev"
}

// NewBasicAuth creates a BasicAuth provider for either prod or dev mode.
// In dev mode the keyring service name AND the file-fallback filename are
// suffixed with "-dev" so dev credentials don't shadow prod.
func NewBasicAuth(devMode bool) *BasicAuth {
	namespace := "freelo-cli"
	filename := "credentials.json"
	if devMode {
		namespace = "freelo-cli-dev"
		filename = "credentials-dev.json"
	}
	return &BasicAuth{
		keyring:   NewKeyring(filename),
		namespace: namespace,
	}
}

const (
	emailKey  = "email"
	apiKeyKey = "api_key"
)

func (b *BasicAuth) GetCredentials() (string, string, error) {
	// Env vars take priority (for CI/agents)
	envEmail := os.Getenv("FREELO_EMAIL")
	envKey := os.Getenv("FREELO_API_KEY")
	if envEmail != "" && envKey != "" {
		return envEmail, envKey, nil
	}

	// Fall back to keyring
	email, err := b.keyring.Get(b.namespace, emailKey)
	if err != nil {
		return "", "", errors.New("not authenticated — run 'freelo auth login' or set FREELO_EMAIL and FREELO_API_KEY env vars")
	}
	apiKey, err := b.keyring.Get(b.namespace, apiKeyKey)
	if err != nil {
		return "", "", errors.New("API key not found in keyring — run 'freelo auth login'")
	}
	return email, apiKey, nil
}

func (b *BasicAuth) Store(email, apiKey string) error {
	if err := b.keyring.Set(b.namespace, emailKey, email); err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}
	if err := b.keyring.Set(b.namespace, apiKeyKey, apiKey); err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}
	return nil
}

func (b *BasicAuth) Clear() error {
	_ = b.keyring.Delete(b.namespace, emailKey)
	_ = b.keyring.Delete(b.namespace, apiKeyKey)
	return nil
}

func (b *BasicAuth) IsAuthenticated() bool {
	_, _, err := b.GetCredentials()
	return err == nil
}
