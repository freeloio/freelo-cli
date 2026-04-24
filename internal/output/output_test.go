package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs f and returns whatever it wrote to os.Stdout.
// Tests in this file must not run in parallel because they swap the
// global os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()
	_ = w.Close()
	os.Stderr = orig
	return <-done
}

func TestJSONEnvelopeOK(t *testing.T) {
	w := NewWriter(FormatJSON)
	data := map[string]any{"id": 42, "name": "thing"}
	crumbs := []Breadcrumb{{Action: "open", Cmd: "freelo tasks get 42", Description: "View task"}}

	out := captureStdout(t, func() {
		w.OK(data, "1 thing", crumbs)
	})

	var got Response
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, out)
	}
	if !got.OK {
		t.Errorf("ok=false, want true")
	}
	if got.Summary != "1 thing" {
		t.Errorf("summary=%q, want %q", got.Summary, "1 thing")
	}
	if len(got.Breadcrumbs) != 1 || got.Breadcrumbs[0].Cmd != "freelo tasks get 42" {
		t.Errorf("breadcrumbs not round-tripped: %+v", got.Breadcrumbs)
	}
	// Data is map[string]any after json round-trip
	if m, ok := got.Data.(map[string]any); !ok || m["name"] != "thing" {
		t.Errorf("data not round-tripped: %+v", got.Data)
	}
}

func TestAgentFormatSkipsEnvelope(t *testing.T) {
	w := NewWriter(FormatAgent)
	data := map[string]any{"key": "value"}

	out := captureStdout(t, func() {
		w.OK(data, "ignored summary", nil)
	})

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, out)
	}
	if _, hasOK := got["ok"]; hasOK {
		t.Errorf("agent format leaked envelope field: %v", got)
	}
	if got["key"] != "value" {
		t.Errorf("raw data not preserved: %v", got)
	}
}

func TestJSONErrEnvelope(t *testing.T) {
	w := NewWriter(FormatJSON)

	out := captureStdout(t, func() {
		w.Err(errors.New("boom"), "API_429", "wait 60s")
	})

	var got ErrorResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, out)
	}
	if got.OK {
		t.Errorf("ok=true on error envelope")
	}
	if got.Error != "boom" || got.Code != "API_429" || got.Hint != "wait 60s" {
		t.Errorf("unexpected error envelope: %+v", got)
	}
}

func TestHumanErrGoesToStderr(t *testing.T) {
	w := NewWriter(FormatQuiet) // not JSON/Agent → human path

	stderr := captureStderr(t, func() {
		w.Err(errors.New("not found"), "404", "check the ID")
	})

	if !strings.Contains(stderr, "Error: not found") {
		t.Errorf("stderr missing error line: %q", stderr)
	}
	if !strings.Contains(stderr, "Hint: check the ID") {
		t.Errorf("stderr missing hint line: %q", stderr)
	}
}

func TestCountFormat(t *testing.T) {
	cases := []struct {
		name string
		data any
		want string
	}{
		{"slice of 3", []any{1, 2, 3}, "3\n"},
		{"empty slice", []any{}, "0\n"},
		{"single object", map[string]any{"id": 1}, "1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWriter(FormatCount)
			got := captureStdout(t, func() { w.OK(tc.data, "", nil) })
			if got != tc.want {
				t.Errorf("count=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestIDsFormat(t *testing.T) {
	w := NewWriter(FormatIDs)
	data := []map[string]any{
		{"id": 10, "name": "a"},
		{"id": 20, "name": "b"},
		{"name": "no id here"},
	}

	got := captureStdout(t, func() { w.OK(data, "", nil) })

	// Only items with an id field should be printed, one per line
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 || lines[0] != "10" || lines[1] != "20" {
		t.Errorf("IDs output=%q, want [10, 20]", got)
	}
}

func TestQuietPrintsSummaryOnly(t *testing.T) {
	w := NewWriter(FormatQuiet)
	out := captureStdout(t, func() {
		w.OK(map[string]any{"should": "be ignored"}, "Created task 123", nil)
	})
	if strings.TrimSpace(out) != "Created task 123" {
		t.Errorf("quiet output=%q, want just the summary", out)
	}
}

func TestFormatValueEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "—"},
		{"true", true, "✓"},
		{"false", false, "✗"},
		{"int-as-float", float64(42), "42"},
		{"float", 3.14, "3.14"},
		{"short string", "hi", "hi"},
		{"long string truncates", strings.Repeat("x", 100), strings.Repeat("x", 57) + "..."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatValue(tc.in); got != tc.want {
				t.Errorf("formatValue(%v)=%q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsComplexField(t *testing.T) {
	if !isComplexField(map[string]any{}) {
		t.Error("map should be complex")
	}
	if !isComplexField([]any{}) {
		t.Error("slice should be complex")
	}
	if isComplexField("string") {
		t.Error("string should not be complex")
	}
	if isComplexField(42) {
		t.Error("int should not be complex")
	}
}
