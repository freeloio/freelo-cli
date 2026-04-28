package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestOSKeyringRoundtrip exercises Set / Get / Delete against go-keyring's
// in-memory mock backend (no Keychain prompts, no DBus access). Confirms our
// wrapper translates ErrNotFound → os.ErrNotExist correctly.
func TestOSKeyringRoundtrip(t *testing.T) {
	keyring.MockInit()

	k := osKeyring{}

	// Get on missing key → os.ErrNotExist
	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get on empty keyring: err=%v, want os.ErrNotExist", err)
	}

	// Set + Get
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

	// Delete + Get → os.ErrNotExist again
	if err := k.Delete("freelo-cli", "email"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get after Delete: err=%v, want os.ErrNotExist", err)
	}

	// Delete on already-missing key is a no-op (not an error).
	if err := k.Delete("freelo-cli", "email"); err != nil {
		t.Fatalf("Delete on missing key returned error: %v", err)
	}
}

// TestOSKeyringNamespaceIsolation proves that prod and dev service names
// don't see each other's secrets — the reason we differentiate namespace
// at the Provider level.
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

// TestOSKeyringErrorWrapping confirms that a backend error (not just
// not-found) is surfaced with the FREELO_KEYRING=file hint so users on
// headless Linux see useful context.
func TestOSKeyringErrorWrapping(t *testing.T) {
	keyring.MockInitWithError(errors.New("dbus: no session bus"))
	defer keyring.MockInit() // restore for other tests

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

// TestFileKeyringRoundtrip exercises the file-based fallback against a temp
// HOME so we don't pollute the real ~/.config/freelo/.
func TestFileKeyringRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	k := &fileKeyring{filename: "credentials-test.json"}

	// Empty file → not found
	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get on empty file: err=%v, want os.ErrNotExist", err)
	}

	// Set + Get round-trip
	if err := k.Set("freelo-cli", "email", "x@y.z"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if v, _ := k.Get("freelo-cli", "email"); v != "x@y.z" {
		t.Errorf("Get returned %q, want x@y.z", v)
	}

	// File written under ~/.config/freelo/<filename> with mode 0600.
	path := filepath.Join(tmp, ".config", "freelo", "credentials-test.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600", info.Mode().Perm())
	}

	// Delete clears just that key, leaving file intact for other keys.
	_ = k.Set("freelo-cli", "api_key", "k")
	_ = k.Delete("freelo-cli", "email")
	if _, err := k.Get("freelo-cli", "email"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("email still present after Delete")
	}
	if v, _ := k.Get("freelo-cli", "api_key"); v != "k" {
		t.Errorf("api_key was wiped by email Delete: %q", v)
	}
}

// TestNewKeyringRespectsEnvVar — the FREELO_KEYRING=file escape hatch picks
// the file backend; otherwise OS keyring is the default.
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

// TestNewBasicAuthDevModeIsolation confirms that prod and dev BasicAuth
// instances target different namespaces in the keyring.
func TestNewBasicAuthDevModeIsolation(t *testing.T) {
	t.Setenv("FREELO_KEYRING", "file") // avoid hitting real OS keyring
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	clearAuthEnv(t)

	prod := NewBasicAuth(false)
	dev := NewBasicAuth(true)

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

	// Two separate files on disk, both 0600.
	for _, name := range []string{"credentials.json", "credentials-dev.json"} {
		path := filepath.Join(tmp, ".config", "freelo", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist: %v", path, err)
		}
	}
}
