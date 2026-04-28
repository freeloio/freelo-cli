package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// Keyring is the interface for credential storage.
//
// Two implementations:
//   - osKeyring (default) — Keychain on macOS, Credential Manager on Windows,
//     Secret Service on Linux desktop, via zalando/go-keyring.
//   - fileKeyring (fallback) — 0600 JSON in ~/.config/freelo/, used when the
//     user opts in via FREELO_KEYRING=file (headless Linux, Docker without
//     DBus, sandboxed CI runners).
//
// Tests use a third in-memory mock that lives in auth_test.go.
type Keyring interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

// NewKeyring returns the keyring backend appropriate for the runtime.
//
// `fallbackFilename` is only consulted when the file backend is selected;
// pass "credentials.json" for prod, "credentials-dev.json" for --dev.
func NewKeyring(fallbackFilename string) Keyring {
	if os.Getenv("FREELO_KEYRING") == "file" {
		return &fileKeyring{filename: fallbackFilename}
	}
	return osKeyring{}
}

// osKeyring wraps zalando/go-keyring. Errors are wrapped with a hint about
// the FREELO_KEYRING=file escape so users on headless Linux see a useful
// message instead of a bare Secret Service / DBus error.
type osKeyring struct{}

func (osKeyring) Get(service, key string) (string, error) {
	v, err := keyring.Get(service, key)
	if err == keyring.ErrNotFound {
		return "", os.ErrNotExist
	}
	if err != nil {
		return "", fmt.Errorf("OS keyring read failed: %w (set FREELO_KEYRING=file to use a 0600 JSON file instead)", err)
	}
	return v, nil
}

func (osKeyring) Set(service, key, value string) error {
	if err := keyring.Set(service, key, value); err != nil {
		return fmt.Errorf("OS keyring write failed: %w (set FREELO_KEYRING=file to use a 0600 JSON file instead)", err)
	}
	return nil
}

func (osKeyring) Delete(service, key string) error {
	err := keyring.Delete(service, key)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

// fileKeyring is the legacy 0600 JSON store at ~/.config/freelo/<filename>.
// Kept as an explicit opt-in for environments without a usable OS keyring.
type fileKeyring struct {
	filename string
}

func (k *fileKeyring) credentialsPath() string {
	home, _ := os.UserHomeDir()
	filename := k.filename
	if filename == "" {
		filename = "credentials.json"
	}
	return filepath.Join(home, ".config", "freelo", filename)
}

func (k *fileKeyring) loadCredentials() (map[string]map[string]string, error) {
	data, err := os.ReadFile(k.credentialsPath())
	if err != nil {
		return make(map[string]map[string]string), nil
	}
	var creds map[string]map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		return make(map[string]map[string]string), nil
	}
	return creds, nil
}

func (k *fileKeyring) saveCredentials(creds map[string]map[string]string) error {
	dir := filepath.Dir(k.credentialsPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(k.credentialsPath(), data, 0600)
}

func (k *fileKeyring) Get(service, key string) (string, error) {
	creds, _ := k.loadCredentials()
	if svc, ok := creds[service]; ok {
		if val, ok := svc[key]; ok {
			return val, nil
		}
	}
	return "", os.ErrNotExist
}

func (k *fileKeyring) Set(service, key, value string) error {
	creds, _ := k.loadCredentials()
	if _, ok := creds[service]; !ok {
		creds[service] = make(map[string]string)
	}
	creds[service][key] = value
	return k.saveCredentials(creds)
}

func (k *fileKeyring) Delete(service, key string) error {
	creds, _ := k.loadCredentials()
	if svc, ok := creds[service]; ok {
		delete(svc, key)
	}
	return k.saveCredentials(creds)
}
