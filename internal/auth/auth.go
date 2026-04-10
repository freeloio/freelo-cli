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

// BasicAuth implements Provider using email + API key.
type BasicAuth struct {
	keyring Keyring
}

// NewBasicAuth creates a BasicAuth provider.
// When dev is true, credentials are stored separately in credentials-dev.json.
func NewBasicAuth(credentialsFile string) *BasicAuth {
	return &BasicAuth{
		keyring: &OSKeyring{Filename: credentialsFile},
	}
}

const (
	serviceName = "freelo-cli"
	emailKey    = "email"
	apiKeyKey   = "api_key"
)

func (b *BasicAuth) GetCredentials() (string, string, error) {
	// Env vars take priority (for CI/agents)
	envEmail := os.Getenv("FREELO_EMAIL")
	envKey := os.Getenv("FREELO_API_KEY")
	if envEmail != "" && envKey != "" {
		return envEmail, envKey, nil
	}

	// Fall back to keyring
	email, err := b.keyring.Get(serviceName, emailKey)
	if err != nil {
		return "", "", errors.New("not authenticated — run 'freelo auth login' or set FREELO_EMAIL and FREELO_API_KEY env vars")
	}
	apiKey, err := b.keyring.Get(serviceName, apiKeyKey)
	if err != nil {
		return "", "", errors.New("API key not found in keyring — run 'freelo auth login'")
	}
	return email, apiKey, nil
}

func (b *BasicAuth) Store(email, apiKey string) error {
	if err := b.keyring.Set(serviceName, emailKey, email); err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}
	if err := b.keyring.Set(serviceName, apiKeyKey, apiKey); err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}
	return nil
}

func (b *BasicAuth) Clear() error {
	_ = b.keyring.Delete(serviceName, emailKey)
	_ = b.keyring.Delete(serviceName, apiKeyKey)
	return nil
}

func (b *BasicAuth) IsAuthenticated() bool {
	_, _, err := b.GetCredentials()
	return err == nil
}
