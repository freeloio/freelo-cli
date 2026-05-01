package credstore

import (
	"errors"
	"os"
	"testing"
)

// mockKeyring is an in-memory Keyring for unit tests.
type mockKeyring struct {
	store  map[string]string // key: service+"/"+key
	getErr error             // if set, Get returns this error
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{store: map[string]string{}}
}

func (m *mockKeyring) Get(service, key string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	v, ok := m.store[service+"/"+key]
	if !ok {
		return "", os.ErrNotExist
	}
	return v, nil
}

func (m *mockKeyring) Set(service, key, value string) error {
	m.store[service+"/"+key] = value
	return nil
}

func (m *mockKeyring) Delete(service, key string) error {
	delete(m.store, service+"/"+key)
	return nil
}

// newTestStore returns a Store wired to a fresh mock keyring under the
// "freelo-cli" namespace (matching prod-mode New(false)), plus the mock
// itself so tests can pre-seed state.
func newTestStore() (*Store, *mockKeyring) {
	m := newMockKeyring()
	return &Store{keyring: m, namespace: "freelo-cli"}, m
}

const prodNamespace = "freelo-cli"

func clearAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FREELO_EMAIL", "")
	t.Setenv("FREELO_API_KEY", "")
}

func TestGetCredentialsEnvVarsWin(t *testing.T) {
	t.Setenv("FREELO_EMAIL", "env@example.com")
	t.Setenv("FREELO_API_KEY", "env-key")

	s, kr := newTestStore()
	_ = kr.Set(prodNamespace, emailKey, "keyring@example.com")
	_ = kr.Set(prodNamespace, apiKeyKey, "keyring-key")

	email, key, err := s.GetCredentials()
	if err != nil {
		t.Fatalf("GetCredentials err=%v", err)
	}
	if email != "env@example.com" || key != "env-key" {
		t.Errorf("env did not win; got (%q, %q)", email, key)
	}
}

func TestGetCredentialsFallbackToKeyring(t *testing.T) {
	clearAuthEnv(t)
	s, kr := newTestStore()
	_ = kr.Set(prodNamespace, emailKey, "stored@example.com")
	_ = kr.Set(prodNamespace, apiKeyKey, "stored-key")

	email, key, err := s.GetCredentials()
	if err != nil {
		t.Fatalf("GetCredentials err=%v", err)
	}
	if email != "stored@example.com" || key != "stored-key" {
		t.Errorf("keyring values not returned; got (%q, %q)", email, key)
	}
}

func TestGetCredentialsPartialEnvFallsThrough(t *testing.T) {
	t.Setenv("FREELO_EMAIL", "env@example.com")
	t.Setenv("FREELO_API_KEY", "")

	s, kr := newTestStore()
	_ = kr.Set(prodNamespace, emailKey, "stored@example.com")
	_ = kr.Set(prodNamespace, apiKeyKey, "stored-key")

	email, key, _ := s.GetCredentials()
	if email != "stored@example.com" || key != "stored-key" {
		t.Errorf("partial env should fall through to keyring; got (%q, %q)", email, key)
	}
}

func TestGetCredentialsNotAuthenticated(t *testing.T) {
	clearAuthEnv(t)
	s, _ := newTestStore()

	_, _, err := s.GetCredentials()
	if err == nil {
		t.Fatal("expected error when unauthenticated")
	}
}

func TestStoreAndClear(t *testing.T) {
	clearAuthEnv(t)
	s, kr := newTestStore()

	if err := s.Store("new@example.com", "new-key"); err != nil {
		t.Fatalf("Store err=%v", err)
	}
	if kr.store[prodNamespace+"/"+emailKey] != "new@example.com" {
		t.Errorf("email not stored in keyring: %v", kr.store)
	}
	if kr.store[prodNamespace+"/"+apiKeyKey] != "new-key" {
		t.Errorf("api key not stored in keyring: %v", kr.store)
	}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear err=%v", err)
	}
	if _, ok := kr.store[prodNamespace+"/"+emailKey]; ok {
		t.Error("email still present after Clear")
	}
}

func TestIsAuthenticated(t *testing.T) {
	clearAuthEnv(t)
	s, kr := newTestStore()

	if s.IsAuthenticated() {
		t.Error("IsAuthenticated=true on empty keyring")
	}

	_ = kr.Set(prodNamespace, emailKey, "x@y.z")
	_ = kr.Set(prodNamespace, apiKeyKey, "k")
	if !s.IsAuthenticated() {
		t.Error("IsAuthenticated=false after Store")
	}
}

func TestKeyringErrorPropagates(t *testing.T) {
	clearAuthEnv(t)
	s, kr := newTestStore()
	kr.getErr = errors.New("keyring unavailable")

	_, _, err := s.GetCredentials()
	if err == nil {
		t.Fatal("expected error when keyring fails")
	}
}

func TestAsProviderUsesGetCredentials(t *testing.T) {
	t.Setenv("FREELO_EMAIL", "env@example.com")
	t.Setenv("FREELO_API_KEY", "env-key")

	s, _ := newTestStore()
	provider := s.AsProvider()
	if provider == nil {
		t.Fatal("AsProvider returned nil")
	}
}
