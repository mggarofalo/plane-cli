package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter/tw"
)

// preferredColumns is the priority order for auto-selecting table columns.
var preferredColumns = []string{
	"id", "name", "identifier", "sequence_id", "priority", "state",
	"status", "display_name", "email", "role", "created_at", "start_date",
	"end_date", "target_date", "is_archived",
}

// numericColumns are columns that should be right-aligned.
var numericColumns = map[string]bool{
	"sequence_id": true, "sort_order": true, "point": true,
	"total_pages": true, "total_count": true,
}

const maxTableColumns = 7

// columnMaxLen returns the max display width for a column based on its type.
func columnMaxLen(col string) int {
	switch {
	case col == "id":
		return 8 // show short UUID
	case strings.HasSuffix(col, "_at"):
		return 10 // date only (YYYY-MM-DD)
	case strings.HasSuffix(col, "_date"):
		return 10
	default:
		return 50
	}
}

// FormatDynamicTable renders a JSON response as a table by auto-detecting columns.
func FormatDynamicTable(w io.Writer, data []byte) error {
	// Try to extract results from paginated envelope
	items, err := extractItems(data)
	if err != nil || len(items) == 0 {
		// Not an array or paginated — fall back to JSON
		fmt.Fprintf(w, "Table format not available for this response; showing JSON:\n")
		return (&JSONFormatter{}).Format(w, json.RawMessage(data))
	}

	// Determine columns from first item
	columns := selectColumns(items[0])
	if len(columns) == 0 {
		fmt.Fprintf(w, "Table format not available for this response; showing JSON:\n")
		return (&JSONFormatter{}).Format(w, json.RawMessage(data))
	}

	// Build headers with title case
	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = toTitleCase(c)
	}

	// Determine per-column alignment
	alignments := make([]tw.Align, len(columns))
	for i, col := range columns {
		if numericColumns[col] {
			alignments[i] = tw.AlignRight
		} else {
			alignments[i] = tw.AlignLeft
		}
	}

	var rows [][]string
	for _, item := range items {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = extractCellValue(item, col)
		}
		rows = append(rows, row)
	}

	WriteTableAligned(w, headers, rows, alignments)
	return nil
}

// toTitleCase converts "created_at" to "Created At".
func toTitleCase(s string) string {
	words := strings.Split(s, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func extractItems(data []byte) ([]map[string]any, error) {
	// Try paginated envelope
	var envelope struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Results != nil {
		var items []map[string]any
		if err := json.Unmarshal(envelope.Results, &items); err == nil {
			return items, nil
		}
	}

	// Try as plain array
	var items []map[string]any
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}

	// Try as single object
	var single map[string]any
	if err := json.Unmarshal(data, &single); err == nil {
		return []map[string]any{single}, nil
	}

	return nil, fmt.Errorf("cannot extract items from response")
}

func selectColumns(item map[string]any) []string {
	var columns []string

	// First, add preferred columns that exist
	for _, col := range preferredColumns {
		if _, ok := item[col]; ok {
			if isSimpleValue(item[col]) {
				columns = append(columns, col)
			}
		}
		if len(columns) >= maxTableColumns {
			break
		}
	}

	// If we have fewer than 3 columns, add more from the object
	if len(columns) < 3 {
		existing := make(map[string]bool)
		for _, c := range columns {
			existing[c] = true
		}
		for key, val := range item {
			if existing[key] || !isSimpleValue(val) {
				continue
			}
			columns = append(columns, key)
			if len(columns) >= maxTableColumns {
				break
			}
		}
	}

	return columns
}

func isSimpleValue(v any) bool {
	switch v.(type) {
	case string, float64, bool, nil:
		return true
	default:
		return false
	}
}

func extractCellValue(item map[string]any, key string) string {
	val, ok := item[key]
	if !ok {
		return ""
	}

	// Handle nested state_detail.name pattern
	if key == "state" || key == "status" {
		if val == nil {
			// Try state_detail
			if detail, ok := item["state_detail"].(map[string]any); ok {
				if name, ok := detail["name"].(string); ok {
					return truncate(name, columnMaxLen(key))
				}
			}
		}
	}

	var str string
	switch v := val.(type) {
	case string:
		str = v
	case float64:
		if v == float64(int(v)) {
			str = fmt.Sprintf("%d", int(v))
		} else {
			str = fmt.Sprintf("%v", v)
		}
	case bool:
		str = fmt.Sprintf("%v", v)
	case nil:
		str = ""
	default:
		str = fmt.Sprintf("%v", v)
	}

	return formatCell(str, key)
}

// formatCell applies column-aware formatting: truncates UUIDs, strips timestamps to dates.
func formatCell(s, col string) string {
	if s == "" {
		return s
	}
	// Shorten UUIDs to first 8 chars
	if col == "id" && looksLikeUUID(s) {
		return s[:8]
	}
	// Strip timestamps to date only for date columns
	if strings.HasSuffix(col, "_at") || strings.HasSuffix(col, "_date") {
		if idx := strings.IndexByte(s, 'T'); idx >= 0 {
			return s[:idx]
		}
	}
	return truncate(s, columnMaxLen(col))
}

// looksLikeUUID checks if s has UUID format (36 chars, dashes at 8,13,18,23).
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

func truncate(s string, maxLen int) string {
	if maxLen > 0 && len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
