package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// maxResponseBody caps how much we read from an HTTP response — keeps us
// safe against a pathological server returning gigabytes of JSON.
const maxResponseBody = 10 * 1024 * 1024

// PaginatedResult holds parsed paginated API response metadata.
type PaginatedResult struct {
	Total int
	Count int
	Page  int
}

// parsePaginatedItems extracts an array of items from Freelo's paginated response.
// Freelo API returns different shapes:
//   - {"data": {"tasks": [...]}, "total": N}    (all-tasks)
//   - {"data": {"projects": [...]}, "total": N}  (all-projects)
//   - {"data": [...], "total": N}                 (all-comments, work-reports, etc.)
//   - [...]                                       (direct array, e.g. /projects)
//
// This function handles all cases and returns the items + pagination metadata.
func parsePaginatedItems(result json.RawMessage) ([]map[string]any, *PaginatedResult) {
	var envelope map[string]any
	if json.Unmarshal(result, &envelope) == nil {
		pr := &PaginatedResult{}
		if v, ok := envelope["total"].(float64); ok {
			pr.Total = int(v)
		}
		if v, ok := envelope["count"].(float64); ok {
			pr.Count = int(v)
		}
		if v, ok := envelope["page"].(float64); ok {
			pr.Page = int(v)
		}

		if data, ok := envelope["data"]; ok {
			if dataMap, ok := data.(map[string]any); ok {
				for _, v := range dataMap {
					if arr, ok := v.([]any); ok {
						return anySliceToMaps(arr), pr
					}
				}
			}
			if arr, ok := data.([]any); ok {
				return anySliceToMaps(arr), pr
			}
		}
	}

	var items []map[string]any
	if json.Unmarshal(result, &items) == nil {
		return items, nil
	}

	return nil, nil
}

func anySliceToMaps(arr []any) []map[string]any {
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

// decodeAPIObject turns the raw body of an oapi-codegen typed response into
// a generic map, or returns an error if the HTTP status is 4xx/5xx, or if
// the body is non-empty but not valid JSON. Used by Phase-3-migrated commands
// to bridge the typed client surface to the output layer (which expects
// untyped JSON trees).
//
// Returning a parse error here is important: a 200 OK with malformed JSON
// (e.g. an HTML error page from an upstream proxy) would otherwise surface
// as out.OK(nil, ...) — a silent success.
func decodeAPIObject(body []byte, resp *http.Response) (map[string]any, error) {
	if err := checkAPIStatus(body, resp); err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("decode response: %w (body starts with: %s)", err, truncateForError(body))
	}
	return m, nil
}

// checkAPIStatus returns a truncated error describing the server response
// when the status is 4xx/5xx. Keeps error messages short enough that a
// rogue server echoing back auth headers can't leak them in full.
func checkAPIStatus(body []byte, resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("no HTTP response")
	}
	if resp.StatusCode >= 400 {
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return fmt.Errorf("API error %d: %s", resp.StatusCode, msg)
	}
	return nil
}

// truncateForError returns at most 120 bytes of the body for embedding in
// parse-error messages. Avoids dumping a full HTML page into stderr.
func truncateForError(body []byte) string {
	const cap = 120
	s := string(body)
	if len(s) > cap {
		return s[:cap] + "..."
	}
	return s
}

// readRawBody consumes the response body (with a size cap) and closes it.
// Meant for callers who want the raw bytes regardless of HTTP status, so
// they can inspect error envelopes from the server.
//
// We use this instead of the oapi-codegen-generated *WithResponse methods
// because those try to auto-unmarshal 2xx JSON into typed structs that
// expect RFC3339 timestamps — Freelo returns timestamps without a timezone
// suffix ("2026-04-24T11:10:19"), which makes time.Time parsing fail and
// the whole call return an error even on HTTP 200. Going through the raw
// Client methods (returning *http.Response) keeps us safe from that while
// still benefiting from typed params + the wrapper's auth/UA/retry.
func readRawBody(resp *http.Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("no HTTP response")
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
}

// consumeAPIObject combines a raw http.Response + transport error into a
// decoded map, propagating any network or status error.
//
// Even on transport error, the SDK can return a non-nil *http.Response
// alongside the error (partial body from a retried request, etc.) — we
// must still close it to avoid leaking the underlying TCP connection.
func consumeAPIObject(resp *http.Response, rerr error) (map[string]any, error) {
	if rerr != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, rerr
	}
	body, err := readRawBody(resp)
	if err != nil {
		return nil, err
	}
	return decodeAPIObject(body, resp)
}

// consumeAPIBody variant for callers that need the raw bytes (typically
// list endpoints that feed into parsePaginatedItems).
func consumeAPIBody(resp *http.Response, rerr error) ([]byte, error) {
	if rerr != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, rerr
	}
	body, err := readRawBody(resp)
	if err != nil {
		return nil, err
	}
	return body, checkAPIStatus(body, resp)
}

// validateAndWrapFileUUIDs takes a slice of UUID strings (typically from a
// repeatable --file flag) and returns the [{uuid: <uuid>}] payload shape
// the Freelo server expects in `files` arrays on comments / task descriptions.
//
// The OpenAPI spec models attachments as {download_url, filename}, but the
// live server treats `download_url` as "fetch this URL and store its
// contents" — passing https://app.freelo.io/file/<uuid> there fetches the
// HTML page, not the file. The shape the server actually accepts for an
// already-uploaded file is {"uuid": "..."}, which is undocumented in the
// spec but works. This helper centralizes that body shape so
// comments/tasks-description don't each re-encode it.
func validateAndWrapFileUUIDs(uuids []string) ([]map[string]string, error) {
	wrapped := make([]map[string]string, 0, len(uuids))
	for _, u := range uuids {
		if _, err := uuid.Parse(u); err != nil {
			return nil, fmt.Errorf("--file %q is not a valid UUID: %w", u, err)
		}
		wrapped = append(wrapped, map[string]string{"uuid": u})
	}
	return wrapped, nil
}

// parseIntArg parses a positional CLI argument that's expected to be a
// positive integer ID. Returns a clear CLI-level error rather than letting
// strconv.Atoi-style silence + a downstream "GET /resource/0" 404 surface
// as the user's only feedback.
func parseIntArg(s, name string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", name, s)
	}
	if v <= 0 {
		return 0, fmt.Errorf("%s must be a positive number, got %d", name, v)
	}
	return v, nil
}

// setProjectsFilter sets the typed ProjectsIds filter when projectID > 0.
// The double-pointer signature lets the helper own the slice allocation so
// callers stay one-liner.
func setProjectsFilter(target **[]int, projectID int) {
	if projectID > 0 {
		ids := []int{projectID}
		*target = &ids
	}
}

// setUsersFilter is the worker/user equivalent of setProjectsFilter.
func setUsersFilter(target **[]int, userID int) {
	if userID > 0 {
		ids := []int{userID}
		*target = &ids
	}
}

// setPageFilter wires the --page flag into a typed *PageParam filter, with
// 1-indexed semantics and a clear error for --page 0 or negative input.
// Omitting --page leaves the server default (first page) in effect.
func setPageFilter(cmd *cobra.Command, target **freelo.PageParam) error {
	page, _ := cmd.Flags().GetInt("page")
	if cmd.Flags().Changed("page") && page < 1 {
		return fmt.Errorf("--page must be 1 or greater (got %d)", page)
	}
	if page > 0 {
		p := freelo.PageParam(page)
		*target = &p
	}
	return nil
}
