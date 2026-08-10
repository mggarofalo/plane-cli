package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
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
		val, _ := traversePath(items[0], "id")
		s := formatRawValue(val)
		_, err := fmt.Fprint(w, s)
		return err
	}

	for i, item := range items {
		val, _ := traversePath(item, "id")
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

// ExtractField extracts a single field from JSON data and writes the raw value
// to w. Strings are printed without JSON quotes.
//
// The path is resolved against the response document first, so keys on a
// paginated envelope ("total_count") and on grouped objects that are not
// envelopes at all (work-item relations, keyed by relation type) both work.
// Only if that misses does it fall back to resolving against each row, one
// value per line — which is what makes "--field name" on a list still emit a
// name per item.
//
// Dotted paths may index into arrays: "results.0.name".
//
// A path that resolves nowhere is an error rather than silent blank output. A
// field that is present but null still prints as empty.
func ExtractField(w io.Writer, data []byte, fieldPath string) error {
	doc, err := parseDocument(data)
	if err != nil {
		return fmt.Errorf("parsing response for field extraction: %w", err)
	}

	// Whole-document match: a single object, an envelope key, or an indexed
	// path into a top-level array.
	if val, ok := traverseValue(doc, fieldPath); ok {
		_, err := fmt.Fprintln(w, formatRawValue(val))
		return err
	}

	items, single, err := parseResponseItems(data)
	if err != nil || single {
		// A single object was already covered by the document match above, so
		// reaching here means the path is simply absent.
		return fmt.Errorf("field %q not found in response", fieldPath)
	}

	// Buffer so a miss does not leave partial output behind before the error.
	var buf bytes.Buffer
	found := false
	for _, item := range items {
		val, ok := traverseValue(item, fieldPath)
		if ok {
			found = true
		}
		if _, err := fmt.Fprintln(&buf, formatRawValue(val)); err != nil {
			return err
		}
	}
	if !found {
		return fmt.Errorf("field %q not found in response", fieldPath)
	}

	_, err = w.Write(buf.Bytes())
	return err
}

// ExtractFields extracts multiple fields from JSON data and writes TSV output
// to w. The first line is a header row with field names. For arrays/paginated
// envelopes, each item produces one row. Strings are printed without JSON quotes.
//
// Unlike ExtractField, a path that resolves nowhere yields a blank cell rather
// than an error: the output is a table addressed by column position, so a
// column that is absent on some responses still has to occupy its slot.
// Resolution is per row only — envelope keys such as "total_count" are not
// visible here, since a scalar has no sensible place in a per-row table.
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
			val, _ := traversePath(item, fp)
			vals[i] = formatRawValue(val)
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

// parseDocument parses the response into a generic value: an object, an array,
// or a scalar. Unlike parseResponseItems it does not unwrap a paginated
// envelope, so callers can address the envelope's own keys.
func parseDocument(data []byte) (any, error) {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// traversePath walks a dotted path through a JSON object. It is a thin wrapper
// over traverseValue for callers that already hold a map.
func traversePath(item map[string]any, path string) (any, bool) {
	return traverseValue(item, path)
}

// traverseValue walks a dotted path (e.g. "state_detail.name" or
// "results.0.name") through nested maps and arrays.
//
// The boolean reports whether the path resolved. That is deliberately distinct
// from the value being nil: a field that is present and null is a successful
// lookup, while a missing field is not, and conflating the two is what made a
// mistyped path indistinguishable from an empty one.
func traverseValue(root any, path string) (any, bool) {
	current := root

	for _, part := range strings.Split(path, ".") {
		switch node := current.(type) {
		case map[string]any:
			val, ok := node[part]
			if !ok {
				return nil, false
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			current = node[idx]
		default:
			// A scalar with path segments still to consume.
			return nil, false
		}
	}
	return current, true
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
