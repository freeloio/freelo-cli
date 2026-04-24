//go:build integration

// Package integration runs end-to-end tests against a real Freelo API using a
// dedicated test project. Opt-in via: `make test-integration` or
// `go test -tags=integration ./test/integration/...`.
//
// The suite builds the freelo binary into a temp dir and drives it as a
// subprocess with --agent for deterministic JSON. This contract (CLI args
// in, JSON envelope out) intentionally survives the oapi-codegen migration
// in Phase 3 — do not switch these to call the internal client directly.
package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	binPath       string // path to the built freelo binary
	testEmail     string
	testAPIKey    string
	testProject   string // ID as string; resolved in TestMain
	freeloBaseURL = "https://api.freelo.io/v1"
)

func TestMain(m *testing.M) {
	// Locate repo root (two levels up from test/integration/).
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	// Try to source .env.freelo-test if env vars aren't already set.
	if os.Getenv("FREELO_EMAIL") == "" || os.Getenv("FREELO_API_KEY") == "" {
		loadDotEnv(filepath.Join(repoRoot, ".env.freelo-test"))
	}

	testEmail = os.Getenv("FREELO_EMAIL")
	testAPIKey = os.Getenv("FREELO_API_KEY")
	testProject = os.Getenv("FREELO_TEST_PROJECT_ID")
	if v := os.Getenv("FREELO_BASE_URL"); v != "" {
		freeloBaseURL = v
	}

	// If credentials aren't present, leave binPath empty; each test will skip.
	if testEmail == "" || testAPIKey == "" {
		fmt.Fprintln(os.Stderr, "[integration] skipping: FREELO_EMAIL / FREELO_API_KEY not set "+
			"(create .env.freelo-test or export the vars)")
		os.Exit(m.Run())
	}

	// Build the binary once.
	tmpDir, err := os.MkdirTemp("", "freelo-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[integration] mktmp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binPath = filepath.Join(tmpDir, "freelo")
	build := exec.Command("go", "build",
		"-ldflags", "-X github.com/freeloio/freelo-cli/internal/cli.Version=integration-test",
		"-o", binPath,
		"./cmd/freelo/",
	)
	build.Dir = repoRoot
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[integration] build failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// loadDotEnv parses `KEY=value` and `export KEY=value` lines into the process
// environment. Quotes around values are stripped. Comments (#) are ignored.
// Missing file is a no-op.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		v = strings.Trim(v, `"'`)
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}

// runFreelo invokes the built binary with the given args. It always passes
// --agent so stdout is raw JSON. Auth comes from the process env vars.
func runFreelo(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	if binPath == "" {
		t.Skip("integration binary not built (credentials missing)")
	}
	args = append(args, "--agent")
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(),
		"FREELO_EMAIL="+testEmail,
		"FREELO_API_KEY="+testAPIKey,
		"FREELO_BASE_URL="+freeloBaseURL,
	)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("exec: %v", err)
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// runFreeloJSON runs the binary and decodes stdout into a generic map.
// Because the harness always passes --agent, the envelope is stripped and
// the returned map IS the command's raw data payload (not {"ok":..,"data":..}).
func runFreeloJSON(t *testing.T, args ...string) map[string]any {
	t.Helper()
	stdout, stderr, code := runFreelo(t, args...)
	if code != 0 {
		t.Fatalf("freelo %v exited %d\nstdout: %s\nstderr: %s", args, code, stdout, stderr)
	}
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		t.Fatalf("empty stdout from freelo %v", args)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(stdout), &v); err != nil {
		t.Fatalf("non-JSON stdout from freelo %v: %v\n%s", args, err, stdout)
	}
	return v
}

// TestVersionSmoke does not hit the API. It validates that the binary built
// and the version ldflag injection works — cheap canary for the whole suite.
func TestVersionSmoke(t *testing.T) {
	got := runFreeloJSON(t, "version")
	if got["version"] != "integration-test" {
		t.Errorf("version=%v, want %q (ldflags injection broken?)", got["version"], "integration-test")
	}
}

// TestAuthStatusAuthenticated verifies that the test credentials round-trip
// through the auth-status path: env → Provider → API → envelope.
func TestAuthStatusAuthenticated(t *testing.T) {
	got := runFreeloJSON(t, "auth", "status")
	if got["authenticated"] != true {
		t.Errorf("authenticated=%v, want true (creds broken?)\nfull: %v", got["authenticated"], got)
	}
}

// TestUsersMe is the minimal real-API round-trip: `/users/me` has no
// pagination or filters and returns a stable shape. If this regresses during
// Phase 3 the wrapper is fundamentally broken.
func TestUsersMe(t *testing.T) {
	got := runFreeloJSON(t, "users", "me")
	if _, ok := got["id"]; !ok {
		t.Errorf("users/me response missing id: %v", got)
	}
	if got["email"] != testEmail {
		t.Errorf("users/me email=%v, want %q", got["email"], testEmail)
	}
}

// TestProjectShowByID documents a Freelo server quirk: the onboarding
// project (e.g. 580898) does NOT appear in /projects or /invited-projects
// listings, so consumers must fetch it by ID. The CLI happens to do this
// correctly today; the test exists so that Phase 3 refactoring can't
// silently regress it.
func TestProjectShowByID(t *testing.T) {
	if testProject == "" {
		t.Skip("FREELO_TEST_PROJECT_ID not set")
	}
	got := runFreeloJSON(t, "projects", "show", testProject)
	// json.Unmarshal decodes the numeric id as float64 — compare as string
	// to sidestep scientific notation ("5.80898e+05") for large numbers.
	if fmt.Sprintf("%.0f", got["id"].(float64)) != testProject {
		t.Errorf("projects show id=%v, want %s", got["id"], testProject)
	}
}

// TestTasksListInProject exercises the pagination/shape parsing for tasks.
// The test project is allowed to have zero tasks — we only assert the
// envelope round-trips to a recognizable shape.
func TestTasksListInProject(t *testing.T) {
	if testProject == "" {
		t.Skip("FREELO_TEST_PROJECT_ID not set")
	}
	stdout, stderr, code := runFreelo(t, "tasks", "list", "--project", testProject)
	if code != 0 {
		t.Fatalf("tasks list exited %d\nstderr: %s", code, stderr)
	}

	// In --agent mode the payload is either a bare array or an object with
	// "tasks"/"items"/etc. Both are valid envelopes in the Freelo API —
	// we accept any JSON that decodes.
	stdout = strings.TrimSpace(stdout)
	var asArr []any
	var asObj map[string]any
	if json.Unmarshal([]byte(stdout), &asArr) != nil &&
		json.Unmarshal([]byte(stdout), &asObj) != nil {
		t.Fatalf("tasks list: stdout is neither array nor object: %s", stdout)
	}
}
