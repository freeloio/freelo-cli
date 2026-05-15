package commands

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParsePaginatedItems exercises the four pagination shapes the Freelo
// API uses across its list endpoints. If a server-side change starts
// returning a fifth shape, this is the test that should fail first.
func TestParsePaginatedItems(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantItems int
		wantTotal int
		wantPage  int
	}{
		{
			name:      "data.tasks envelope",
			body:      `{"data": {"tasks": [{"id": 1}, {"id": 2}, {"id": 3}]}, "total": 7, "count": 3, "page": 1}`,
			wantItems: 3,
			wantTotal: 7,
			wantPage:  1,
		},
		{
			name:      "data.projects envelope",
			body:      `{"data": {"projects": [{"id": 10}, {"id": 20}]}, "total": 2}`,
			wantItems: 2,
			wantTotal: 2,
		},
		{
			name:      "data is a bare array envelope",
			body:      `{"data": [{"id": 100}], "total": 1, "page": 2}`,
			wantItems: 1,
			wantTotal: 1,
			wantPage:  2,
		},
		{
			name:      "top-level array",
			body:      `[{"id": 1}, {"id": 2}]`,
			wantItems: 2,
		},
		{
			name:      "empty envelope",
			body:      `{"data": [], "total": 0}`,
			wantItems: 0,
		},
		{
			name:      "empty top-level array",
			body:      `[]`,
			wantItems: 0,
		},
		{
			name:      "envelope with non-numeric total degrades gracefully",
			body:      `{"data": [{"id": 1}], "total": "weird"}`,
			wantItems: 1,
			wantTotal: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items, pr := parsePaginatedItems(json.RawMessage(tc.body))
			if len(items) != tc.wantItems {
				t.Errorf("items count = %d, want %d", len(items), tc.wantItems)
			}
			if tc.wantTotal != 0 || tc.wantPage != 0 {
				if pr == nil {
					t.Fatalf("expected pagination metadata, got nil")
				}
				if pr.Total != tc.wantTotal {
					t.Errorf("total = %d, want %d", pr.Total, tc.wantTotal)
				}
				if pr.Page != tc.wantPage {
					t.Errorf("page = %d, want %d", pr.Page, tc.wantPage)
				}
			}
		})
	}
}

func TestParseIntArg(t *testing.T) {
	if v, err := parseIntArg("42", "task-id"); err != nil || v != 42 {
		t.Errorf("parseIntArg(\"42\")=%d,%v; want 42,nil", v, err)
	}
	if _, err := parseIntArg("abc", "task-id"); err == nil {
		t.Error("parseIntArg(\"abc\") should error")
	} else if !strings.Contains(err.Error(), "task-id") {
		t.Errorf("error message missing arg name: %v", err)
	}
	if _, err := parseIntArg("0", "task-id"); err == nil {
		t.Error("parseIntArg(\"0\") should error (must be positive)")
	}
	if _, err := parseIntArg("-5", "task-id"); err == nil {
		t.Error("parseIntArg(\"-5\") should error (must be positive)")
	}
}

func TestValidateAndWrapFileUUIDs(t *testing.T) {
	out, err := validateAndWrapFileUUIDs([]string{"550e8400-e29b-41d4-a716-446655440000"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0]["uuid"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected wrapped output: %v", out)
	}

	if _, err := validateAndWrapFileUUIDs([]string{"not-a-uuid"}); err == nil {
		t.Error("expected error on invalid UUID")
	}
}
