package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	// Try as paginated envelope first
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
			// data is a map like {"tasks": [...]} or {"projects": [...]}
			if dataMap, ok := data.(map[string]any); ok {
				// Find the first array value in the map
				for _, v := range dataMap {
					if arr, ok := v.([]any); ok {
						return anySliceToMaps(arr), pr
					}
				}
			}
			// data is directly an array
			if arr, ok := data.([]any); ok {
				return anySliceToMaps(arr), pr
			}
		}
	}

	// Try as direct array
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
// a generic map, or returns an error if the HTTP status is 4xx/5xx. Used by
// Phase-3-migrated commands to bridge the typed client surface to the output
// layer (which expects untyped JSON trees).
//
// Keeping the output layer untyped is deliberate — the CLI's job is to
// round-trip API JSON to stdout; enforcing structural types per command
// would be a lot of code for very little user-visible benefit.
func decodeAPIObject(body []byte, resp *http.Response) (map[string]any, error) {
	if err := checkAPIStatus(body, resp); err != nil {
		return nil, err
	}
	var m map[string]any
	if len(body) == 0 {
		return m, nil
	}
	_ = json.Unmarshal(body, &m)
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
func consumeAPIObject(resp *http.Response, rerr error) (map[string]any, error) {
	if rerr != nil {
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
		return nil, rerr
	}
	body, err := readRawBody(resp)
	if err != nil {
		return nil, err
	}
	return body, checkAPIStatus(body, resp)
}
