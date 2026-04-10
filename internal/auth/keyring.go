package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Keyring is the interface for credential storage.
// Uses OS keyring when available, falls back to file-based storage.
type Keyring interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

// OSKeyring uses a file-based credential store.
// TODO: Replace with github.com/zalando/go-keyring for real OS keyring integration.
// File-based approach is simpler for MVP and works consistently across platforms.
type OSKeyring struct {
	Filename string // "credentials.json" or "credentials-dev.json"
}

func (k *OSKeyring) credentialsPath() string {
	home, _ := os.UserHomeDir()
	filename := k.Filename
	if filename == "" {
		filename = "credentials.json"
	}
	return filepath.Join(home, ".config", "freelo", filename)
}

func (k *OSKeyring) loadCredentials() (map[string]map[string]string, error) {
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

func (k *OSKeyring) saveCredentials(creds map[string]map[string]string) error {
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

func (k *OSKeyring) Get(service, key string) (string, error) {
	creds, _ := k.loadCredentials()
	if svc, ok := creds[service]; ok {
		if val, ok := svc[key]; ok {
			return val, nil
		}
	}
	return "", os.ErrNotExist
}

func (k *OSKeyring) Set(service, key, value string) error {
	creds, _ := k.loadCredentials()
	if _, ok := creds[service]; !ok {
		creds[service] = make(map[string]string)
	}
	creds[service][key] = value
	return k.saveCredentials(creds)
}

func (k *OSKeyring) Delete(service, key string) error {
	creds, _ := k.loadCredentials()
	if svc, ok := creds[service]; ok {
		delete(svc, key)
	}
	return k.saveCredentials(creds)
}
