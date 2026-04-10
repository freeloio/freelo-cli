package commands

import (
	"encoding/json"
)

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
