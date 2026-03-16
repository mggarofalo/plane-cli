package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ExtractID extracts the "id" field from a JSON response and writes the raw UUID
// to w without a trailing newline. For arrays/paginated envelopes, it writes one
// ID per line (each without trailing newline, separated by newlines).
func ExtractID(w io.Writer, data []byte) error {
	items, single, err := parseResponseItems(data)
	if err != nil {
		return fmt.Errorf("parsing response for id extraction: %w", err)
	}

	if single {
		val := traversePath(items[0], "id")
		s := formatRawValue(val)
		_, err := fmt.Fprint(w, s)
		return err
	}

	for i, item := range items {
		val := traversePath(item, "id")
		s := formatRawValue(val)
		if i > 0 {
			if _, err := fmt.Fprint(w, "\n"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, s); err != nil {
			return err
		}
	}
	return nil
}

// ExtractField extracts a single field (supporting dotted paths) from JSON data
// and writes the raw value to w. For arrays/paginated envelopes, it writes one
// value per line. Strings are printed without JSON quotes.
func ExtractField(w io.Writer, data []byte, fieldPath string) error {
	items, single, err := parseResponseItems(data)
	if err != nil {
		return fmt.Errorf("parsing response for field extraction: %w", err)
	}

	if single {
		val := traversePath(items[0], fieldPath)
		_, err := fmt.Fprintln(w, formatRawValue(val))
		return err
	}

	for _, item := range items {
		val := traversePath(item, fieldPath)
		if _, err := fmt.Fprintln(w, formatRawValue(val)); err != nil {
			return err
		}
	}
	return nil
}

// ExtractFields extracts multiple fields from JSON data and writes TSV output
// to w. The first line is a header row with field names. For arrays/paginated
// envelopes, each item produces one row. Strings are printed without JSON quotes.
func ExtractFields(w io.Writer, data []byte, fieldPaths []string) error {
	items, _, err := parseResponseItems(data)
	if err != nil {
		return fmt.Errorf("parsing response for field extraction: %w", err)
	}

	// Header row
	if _, err := fmt.Fprintln(w, strings.Join(fieldPaths, "\t")); err != nil {
		return err
	}

	for _, item := range items {
		vals := make([]string, len(fieldPaths))
		for i, fp := range fieldPaths {
			vals[i] = formatRawValue(traversePath(item, fp))
		}
		if _, err := fmt.Fprintln(w, strings.Join(vals, "\t")); err != nil {
			return err
		}
	}
	return nil
}

// parseResponseItems parses JSON data into a slice of maps. It handles:
// - Paginated envelopes ({"results": [...]})
// - Plain arrays ([...])
// - Single objects ({...})
// The second return value is true when the input was a single object (not wrapped
// in an array or paginated envelope).
func parseResponseItems(data []byte) ([]map[string]any, bool, error) {
	// Try paginated envelope
	var envelope struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Results != nil {
		var items []map[string]any
		if err := json.Unmarshal(envelope.Results, &items); err == nil {
			return items, false, nil
		}
	}

	// Try as plain array
	var items []map[string]any
	if err := json.Unmarshal(data, &items); err == nil {
		return items, false, nil
	}

	// Try as single object
	var single map[string]any
	if err := json.Unmarshal(data, &single); err == nil {
		return []map[string]any{single}, true, nil
	}

	return nil, false, fmt.Errorf("response is not a JSON object or array")
}

// traversePath walks a dotted path (e.g. "state_detail.name") through nested maps.
// Returns nil if any segment is missing or the value at any intermediate segment
// is not a map.
func traversePath(item map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = item

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}

// formatRawValue converts a JSON value to a raw string representation.
// Strings are returned without quotes; numbers use minimal formatting;
// nil becomes empty string; booleans become "true"/"false".
func formatRawValue(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case json.Number:
		return v.String()
	default:
		// For nested objects/arrays that aren't traversed, marshal as compact JSON
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	}
}
