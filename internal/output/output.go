package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"text/tabwriter"
)

// errorPrinted records whether any Writer.Err call has fired in this
// process. The CLI's main() inspects it so it can skip its own
// stderr-fallback when a command already rendered the error to stderr or
// to the JSON envelope. Without this, validation errors would print twice:
// once by Writer.Err and once by main's safety net for unhandled errors.
var errorPrinted atomic.Bool

// ErrorWasRendered reports whether any Writer.Err has been called in this
// process. Intended for the CLI entrypoint to suppress double-printing.
func ErrorWasRendered() bool { return errorPrinted.Load() }

// Format represents the output format.
type Format int

const (
	FormatAuto  Format = iota // Styled if TTY, JSON if piped
	FormatJSON                // JSON with envelope
	FormatAgent               // Raw JSON data only (--agent = --json + --quiet)
	FormatQuiet               // Minimal output
	FormatIDs                 // IDs only, one per line
	FormatCount               // Just the count
)

// Breadcrumb suggests a follow-up action to the user or agent.
type Breadcrumb struct {
	Action      string `json:"action"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

// Response is the standard JSON envelope for successful responses.
type Response struct {
	OK          bool         `json:"ok"`
	Data        any          `json:"data"`
	Summary     string       `json:"summary,omitempty"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs,omitempty"`
}

// ErrorResponse is the JSON envelope for errors.
type ErrorResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

// Writer handles output formatting based on the selected format.
type Writer struct {
	Format Format
}

// NewWriter creates an output writer. Defaults to auto-detection.
func NewWriter(format Format) *Writer {
	return &Writer{Format: format}
}

// OK outputs a successful result.
func (w *Writer) OK(data any, summary string, breadcrumbs []Breadcrumb) {
	switch w.Format {
	case FormatAgent:
		w.printJSON(data)
	case FormatJSON:
		w.printJSON(Response{
			OK:          true,
			Data:        data,
			Summary:     summary,
			Breadcrumbs: breadcrumbs,
		})
	case FormatQuiet:
		if summary != "" {
			fmt.Println(summary)
		}
	case FormatIDs:
		w.printIDs(data)
	case FormatCount:
		w.printCount(data)
	default: // FormatAuto
		if isTerminal() {
			w.printStyled(data, summary, breadcrumbs)
		} else {
			w.printJSON(Response{
				OK:          true,
				Data:        data,
				Summary:     summary,
				Breadcrumbs: breadcrumbs,
			})
		}
	}
}

// Err outputs an error and marks the process as having rendered one
// (see ErrorWasRendered).
func (w *Writer) Err(err error, code, hint string) {
	errorPrinted.Store(true)
	switch w.Format {
	case FormatAgent, FormatJSON:
		w.printJSON(ErrorResponse{
			OK:    false,
			Error: err.Error(),
			Code:  code,
			Hint:  hint,
		})
	default:
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		if hint != "" {
			fmt.Fprintf(os.Stderr, "Hint: %s\n", hint)
		}
	}
}

func (w *Writer) printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// MarshalIndent only fails for unrepresentable types (channels,
		// funcs, etc.) which shouldn't appear in our payloads. Fall back
		// to a Go-format dump on stderr so we never silently print "".
		fmt.Fprintf(os.Stderr, "output: failed to marshal JSON (%v); raw value: %#v\n", err, v)
		return
	}
	fmt.Println(string(data))
}

func (w *Writer) printIDs(data any) {
	// Fast paths first — avoid the round-trip when the caller already
	// hands us the canonical shapes.
	switch v := data.(type) {
	case []map[string]any:
		for _, item := range v {
			if id, exists := item["id"]; exists {
				fmt.Printf("%v\n", id)
			}
		}
		return
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if id, exists := m["id"]; exists {
					fmt.Println(id)
				}
			}
		}
		return
	}
	// Fallback: marshal/unmarshal to normalize odd shapes (typed slices,
	// pointer slices) into []map[string]any.
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	var items []map[string]any
	if json.Unmarshal(raw, &items) == nil {
		for _, item := range items {
			if id, exists := item["id"]; exists {
				fmt.Printf("%v\n", id)
			}
		}
	}
}

func (w *Writer) printCount(data any) {
	// Fast path — no round-trip when caller hands us a canonical slice.
	switch v := data.(type) {
	case []map[string]any:
		fmt.Println(len(v))
		return
	case []any:
		fmt.Println(len(v))
		return
	case nil:
		fmt.Println(0)
		return
	}
	// Fallback: marshal/unmarshal to detect array-shaped data we didn't
	// fast-path above. Non-array payloads return 0 — using "1" for a
	// single object was misleading (`--count` on `tasks show <id>`
	// previously claimed 1 even when the request 404'd into an envelope).
	raw, err := json.Marshal(data)
	if err != nil {
		fmt.Println(0)
		return
	}
	var items []any
	if json.Unmarshal(raw, &items) == nil {
		fmt.Println(len(items))
		return
	}
	fmt.Println(0)
}

func (w *Writer) printStyled(data any, summary string, breadcrumbs []Breadcrumb) {
	// Convert to generic form for table printing
	raw, _ := json.Marshal(data)

	// Try as array of objects
	var items []map[string]any
	if json.Unmarshal(raw, &items) == nil && len(items) > 0 {
		printTable(items)
		if summary != "" {
			fmt.Printf("\n%s\n", summary)
		}
		if len(breadcrumbs) > 0 {
			fmt.Println()
			for _, bc := range breadcrumbs {
				fmt.Printf("  → %s: %s\n", bc.Description, bc.Cmd)
			}
		}
		return
	}

	// Single object — pretty print key/value
	var obj map[string]any
	if json.Unmarshal(raw, &obj) == nil {
		printObject(obj)
		if len(breadcrumbs) > 0 {
			fmt.Println()
			for _, bc := range breadcrumbs {
				fmt.Printf("  → %s: %s\n", bc.Description, bc.Cmd)
			}
		}
		return
	}

	// Fallback: raw JSON
	w.printJSON(data)
}

func printTable(items []map[string]any) {
	if len(items) == 0 {
		return
	}

	// Determine columns from common project management fields in priority order
	priorityFields := []string{"id", "name", "state", "priority", "worker", "due_date", "date_add"}
	var columns []string
	for _, f := range priorityFields {
		if _, exists := items[0][f]; exists {
			columns = append(columns, f)
		}
	}
	// Add remaining fields
	for k := range items[0] {
		found := false
		for _, c := range columns {
			if c == k {
				found = true
				break
			}
		}
		if !found && !isComplexField(items[0][k]) {
			columns = append(columns, k)
		}
	}

	// Cap at 8 columns for readability
	if len(columns) > 8 {
		columns = columns[:8]
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	// Header
	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = strings.ToUpper(c)
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	// Separator
	seps := make([]string, len(columns))
	for i, h := range headers {
		seps[i] = strings.Repeat("─", len(h))
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Rows
	for _, item := range items {
		vals := make([]string, len(columns))
		for i, c := range columns {
			vals[i] = formatValue(item[c])
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	_ = tw.Flush()
}

func printObject(obj map[string]any) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for k, v := range obj {
		if !isComplexField(v) {
			fmt.Fprintf(tw, "%s:\t%s\n", k, formatValue(v))
		}
	}
	_ = tw.Flush()
}

func formatValue(v any) string {
	if v == nil {
		return "—"
	}
	switch val := v.(type) {
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "✓"
		}
		return "✗"
	case string:
		if len(val) > 60 {
			return val[:57] + "..."
		}
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

func isComplexField(v any) bool {
	switch v.(type) {
	case map[string]any, []any:
		return true
	}
	return false
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
