package credstore

import (
	"encoding/json"
	"errors"
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
	if errors.Is(err, keyring.ErrNotFound) {
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
	if errors.Is(err, keyring.ErrNotFound) {
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

// saveCredentials writes the credentials map to disk atomically: write to
// a temp file in the same directory, fsync, chmod 0600, then rename over
// the target. This way a crash, full disk, or signal between truncate and
// final write can't leave the user with zero credentials. (POSIX rename is
// atomic; the temp file is removed if anything fails before the rename.)
func (k *fileKeyring) saveCredentials(creds map[string]map[string]string) error {
	target := k.credentialsPath()
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".credentials-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp credentials file: %w", err)
	}
	tmpName := tmp.Name()
	// On any failure below, remove the temp file. Successful rename
	// renders the Remove a no-op.
	defer func() { _ = os.Remove(tmpName) }()

	// Chmod first so the secret bytes never sit on disk world-readable,
	// even briefly. os.WriteFile's mode arg isn't honored on existing
	// files; we use an explicit Chmod against the freshly created file.
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp credentials file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp credentials file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp credentials file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp credentials file: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("rename credentials file into place: %w", err)
	}
	return nil
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
