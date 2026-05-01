package credstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestOSKeyringRoundtrip exercises Set / Get / Delete against go-keyring's
// in-memory mock backend. Confirms our wrapper translates ErrNotFound →
// os.ErrNotExist correctly.
func TestOSKeyringRoundtrip(t *testing.T) {
	keyring.MockInit()

	k := osKeyring{}

	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get on empty keyring: err=%v, want os.ErrNotExist", err)
	}

	if err := k.Set("freelo-cli", "email", "marek@example.com"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := k.Get("freelo-cli", "email")
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}
	if got != "marek@example.com" {
		t.Errorf("Get returned %q, want %q", got, "marek@example.com")
	}

	if err := k.Delete("freelo-cli", "email"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get after Delete: err=%v, want os.ErrNotExist", err)
	}

	if err := k.Delete("freelo-cli", "email"); err != nil {
		t.Fatalf("Delete on missing key returned error: %v", err)
	}
}

func TestOSKeyringNamespaceIsolation(t *testing.T) {
	keyring.MockInit()
	k := osKeyring{}

	_ = k.Set("freelo-cli", "email", "prod@example.com")
	_ = k.Set("freelo-cli-dev", "email", "dev@example.com")

	prod, _ := k.Get("freelo-cli", "email")
	dev, _ := k.Get("freelo-cli-dev", "email")

	if prod != "prod@example.com" || dev != "dev@example.com" {
		t.Errorf("namespaces leaked: prod=%q, dev=%q", prod, dev)
	}
}

func TestOSKeyringErrorWrapping(t *testing.T) {
	keyring.MockInitWithError(errors.New("dbus: no session bus"))
	defer keyring.MockInit()

	k := osKeyring{}
	_, err := k.Get("freelo-cli", "email")
	if err == nil {
		t.Fatal("expected error from broken backend")
	}
	if !contains(err.Error(), "FREELO_KEYRING=file") {
		t.Errorf("error missing fallback hint: %v", err)
	}

	if err := k.Set("freelo-cli", "email", "x"); err == nil {
		t.Fatal("expected error from broken backend on Set")
	} else if !contains(err.Error(), "FREELO_KEYRING=file") {
		t.Errorf("Set error missing fallback hint: %v", err)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestFileKeyringRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	k := &fileKeyring{filename: "credentials-test.json"}

	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get on empty file: err=%v, want os.ErrNotExist", err)
	}

	if err := k.Set("freelo-cli", "email", "x@y.z"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if v, _ := k.Get("freelo-cli", "email"); v != "x@y.z" {
		t.Errorf("Get returned %q, want x@y.z", v)
	}

	path := filepath.Join(tmp, ".config", "freelo", "credentials-test.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600", info.Mode().Perm())
	}

	_ = k.Set("freelo-cli", "api_key", "k")
	_ = k.Delete("freelo-cli", "email")
	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("email still present after Delete")
	}
	if v, _ := k.Get("freelo-cli", "api_key"); v != "k" {
		t.Errorf("api_key was wiped by email Delete: %q", v)
	}
}

func TestNewKeyringRespectsEnvVar(t *testing.T) {
	t.Setenv("FREELO_KEYRING", "file")
	k := NewKeyring("credentials.json")
	if _, ok := k.(*fileKeyring); !ok {
		t.Errorf("FREELO_KEYRING=file gave type %T, want *fileKeyring", k)
	}

	t.Setenv("FREELO_KEYRING", "")
	k = NewKeyring("credentials.json")
	if _, ok := k.(osKeyring); !ok {
		t.Errorf("FREELO_KEYRING unset gave type %T, want osKeyring", k)
	}
}

// TestNewDevModeIsolation confirms that prod and dev Stores target
// different namespaces in the keyring.
func TestNewDevModeIsolation(t *testing.T) {
	t.Setenv("FREELO_KEYRING", "file")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	clearAuthEnv(t)

	prod := New(false)
	dev := New(true)

	if err := prod.Store("prod@x.cz", "prod-key"); err != nil {
		t.Fatal(err)
	}
	if err := dev.Store("dev@x.cz", "dev-key"); err != nil {
		t.Fatal(err)
	}

	prodEmail, prodKey, err := prod.GetCredentials()
	if err != nil {
		t.Fatalf("prod GetCredentials: %v", err)
	}
	devEmail, devKey, err := dev.GetCredentials()
	if err != nil {
		t.Fatalf("dev GetCredentials: %v", err)
	}

	if prodEmail != "prod@x.cz" || prodKey != "prod-key" {
		t.Errorf("prod creds bled into dev: (%q, %q)", prodEmail, prodKey)
	}
	if devEmail != "dev@x.cz" || devKey != "dev-key" {
		t.Errorf("dev creds bled into prod: (%q, %q)", devEmail, devKey)
	}

	for _, name := range []string{"credentials.json", "credentials-dev.json"} {
		path := filepath.Join(tmp, ".config", "freelo", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist: %v", path, err)
		}
	}
}
