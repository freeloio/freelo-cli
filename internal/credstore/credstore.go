// Package credstore is the CLI's credential store. It owns the OS keyring
// and the 0600 JSON file fallback (CLI-specific concerns), and exposes
// stored credentials to the freelo-go SDK via a CredentialsFunc adapter.
//
// Login / logout / status commands talk to *Store. The transport in the
// SDK only sees the Provider returned by AsProvider — it has no knowledge
// of where credentials live.
package credstore

import (
	"context"
	"errors"
	"fmt"
	"os"

	freeloauth "github.com/freeloio/freelo-go/auth"
)

const (
	emailKey  = "email"
	apiKeyKey = "api_key"
)

// Store wraps a Keyring with the namespacing and env-var override rules
// the CLI needs. The keyring is addressed under a service namespace that
// differs between prod (`freelo-cli`) and dev (`freelo-cli-dev`) so the
// two never collide in Keychain / Credential Manager / Secret Service.
type Store struct {
	keyring   Keyring
	namespace string
}

// New creates a Store for either prod or dev mode. In dev mode the
// keyring service name AND the file-fallback filename are suffixed with
// "-dev" so dev credentials don't shadow prod.
func New(devMode bool) *Store {
	namespace := "freelo-cli"
	filename := "credentials.json"
	if devMode {
		namespace = "freelo-cli-dev"
		filename = "credentials-dev.json"
	}
	return &Store{
		keyring:   NewKeyring(filename),
		namespace: namespace,
	}
}

// GetCredentials returns the email and API key for outbound requests.
// Env vars take priority for CI / agent / sandboxed runs; the keyring is
// the fallback used by interactive login flows.
func (s *Store) GetCredentials() (string, string, error) {
	if envEmail, envKey := os.Getenv("FREELO_EMAIL"), os.Getenv("FREELO_API_KEY"); envEmail != "" && envKey != "" {
		return envEmail, envKey, nil
	}

	email, err := s.keyring.Get(s.namespace, emailKey)
	if err != nil {
		return "", "", errors.New("not authenticated — run 'freelo auth login' or set FREELO_EMAIL and FREELO_API_KEY env vars")
	}
	apiKey, err := s.keyring.Get(s.namespace, apiKeyKey)
	if err != nil {
		return "", "", errors.New("API key not found in keyring — run 'freelo auth login'")
	}
	return email, apiKey, nil
}

// Store persists credentials to the keyring under this Store's namespace.
//
// If the second Set fails after the first succeeded, we roll back the
// first write so the store never sits in a half-saved state where
// GetCredentials would return "API key not found" instead of "not
// authenticated" and confuse the user.
func (s *Store) Store(email, apiKey string) error {
	if err := s.keyring.Set(s.namespace, emailKey, email); err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}
	if err := s.keyring.Set(s.namespace, apiKeyKey, apiKey); err != nil {
		// Best-effort rollback. If rollback also fails, we still want the
		// user to see the original error, not a confusing secondary one.
		_ = s.keyring.Delete(s.namespace, emailKey)
		return fmt.Errorf("failed to store API key: %w", err)
	}
	return nil
}

// Clear removes both fields from the keyring. Errors are ignored — the
// goal is "leave the keyring without our credentials", and a missing
// entry is the same outcome as a successful delete.
func (s *Store) Clear() error {
	_ = s.keyring.Delete(s.namespace, emailKey)
	_ = s.keyring.Delete(s.namespace, apiKeyKey)
	return nil
}

// IsAuthenticated reports whether GetCredentials currently returns a pair.
// Does not contact the API — credential validity is a server-side question.
func (s *Store) IsAuthenticated() bool {
	_, _, err := s.GetCredentials()
	return err == nil
}

// AsProvider returns a freelo-go auth provider that resolves credentials
// per outgoing request via GetCredentials. This is the seam between the
// CLI's keyring-aware storage and the SDK's transport. The SDK never
// touches the keyring directly.
func (s *Store) AsProvider() freeloauth.Provider {
	return freeloauth.CredentialsFunc(func(_ context.Context) (string, string, error) {
		return s.GetCredentials()
	})
}
