package auth

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

// newTestAuth returns a BasicAuth wired to a fresh mock keyring, plus the
// mock itself so tests can pre-seed state.
func newTestAuth() (*BasicAuth, *mockKeyring) {
	m := newMockKeyring()
	return &BasicAuth{keyring: m}, m
}

// clearAuthEnv removes FREELO_EMAIL / FREELO_API_KEY for the duration of a test.
func clearAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FREELO_EMAIL", "")
	t.Setenv("FREELO_API_KEY", "")
}

func TestGetCredentialsEnvVarsWin(t *testing.T) {
	t.Setenv("FREELO_EMAIL", "env@example.com")
	t.Setenv("FREELO_API_KEY", "env-key")

	ba, kr := newTestAuth()
	// Pre-seed the keyring with different values so we can prove env wins.
	_ = kr.Set(serviceName, emailKey, "keyring@example.com")
	_ = kr.Set(serviceName, apiKeyKey, "keyring-key")

	email, key, err := ba.GetCredentials()
	if err != nil {
		t.Fatalf("GetCredentials err=%v", err)
	}
	if email != "env@example.com" || key != "env-key" {
		t.Errorf("env did not win; got (%q, %q)", email, key)
	}
}

func TestGetCredentialsFallbackToKeyring(t *testing.T) {
	clearAuthEnv(t)
	ba, kr := newTestAuth()
	_ = kr.Set(serviceName, emailKey, "stored@example.com")
	_ = kr.Set(serviceName, apiKeyKey, "stored-key")

	email, key, err := ba.GetCredentials()
	if err != nil {
		t.Fatalf("GetCredentials err=%v", err)
	}
	if email != "stored@example.com" || key != "stored-key" {
		t.Errorf("keyring values not returned; got (%q, %q)", email, key)
	}
}

func TestGetCredentialsPartialEnvFallsThrough(t *testing.T) {
	// Only email is set in env → should fall back to keyring entirely
	// (code requires BOTH env vars before using env).
	t.Setenv("FREELO_EMAIL", "env@example.com")
	t.Setenv("FREELO_API_KEY", "")

	ba, kr := newTestAuth()
	_ = kr.Set(serviceName, emailKey, "stored@example.com")
	_ = kr.Set(serviceName, apiKeyKey, "stored-key")

	email, key, _ := ba.GetCredentials()
	if email != "stored@example.com" || key != "stored-key" {
		t.Errorf("partial env should fall through to keyring; got (%q, %q)", email, key)
	}
}

func TestGetCredentialsNotAuthenticated(t *testing.T) {
	clearAuthEnv(t)
	ba, _ := newTestAuth() // empty keyring

	_, _, err := ba.GetCredentials()
	if err == nil {
		t.Fatal("expected error when unauthenticated")
	}
}

func TestStoreAndClear(t *testing.T) {
	clearAuthEnv(t)
	ba, kr := newTestAuth()

	if err := ba.Store("new@example.com", "new-key"); err != nil {
		t.Fatalf("Store err=%v", err)
	}
	if kr.store[serviceName+"/"+emailKey] != "new@example.com" {
		t.Errorf("email not stored in keyring: %v", kr.store)
	}
	if kr.store[serviceName+"/"+apiKeyKey] != "new-key" {
		t.Errorf("api key not stored in keyring: %v", kr.store)
	}

	if err := ba.Clear(); err != nil {
		t.Fatalf("Clear err=%v", err)
	}
	if _, ok := kr.store[serviceName+"/"+emailKey]; ok {
		t.Error("email still present after Clear")
	}
}

func TestIsAuthenticated(t *testing.T) {
	clearAuthEnv(t)
	ba, kr := newTestAuth()

	if ba.IsAuthenticated() {
		t.Error("IsAuthenticated=true on empty keyring")
	}

	_ = kr.Set(serviceName, emailKey, "x@y.z")
	_ = kr.Set(serviceName, apiKeyKey, "k")
	if !ba.IsAuthenticated() {
		t.Error("IsAuthenticated=false after Store")
	}
}

func TestKeyringErrorPropagates(t *testing.T) {
	clearAuthEnv(t)
	ba, kr := newTestAuth()
	kr.getErr = errors.New("keyring unavailable")

	_, _, err := ba.GetCredentials()
	if err == nil {
		t.Fatal("expected error when keyring fails")
	}
}
