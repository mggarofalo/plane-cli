package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Format represents an output format.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// Formatter writes structured data to an output writer.
type Formatter interface {
	Format(w io.Writer, data any) error
}

// New returns a Formatter for the given format string.
func New(format string) Formatter {
	switch Format(format) {
	case FormatTable:
		return &TableFormatter{}
	default:
		return &JSONFormatter{}
	}
}

// JSONFormatter outputs data as indented JSON.
type JSONFormatter struct{}

func (f *JSONFormatter) Format(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

// TableFormatter outputs data as a human-readable table.
type TableFormatter struct{}

func (f *TableFormatter) Format(w io.Writer, data any) error {
	// For table output, we support specific types via type switches.
	// For unknown types, fall back to JSON.
	switch v := data.(type) {
	case TableRenderable:
		return v.RenderTable(w)
	default:
		// Fallback: just print as JSON for types without table support
		fmt.Fprintf(w, "Table format not supported for this data type; showing JSON:\n")
		return (&JSONFormatter{}).Format(w, data)
	}
}

// TableRenderable can be implemented by types that support table rendering.
type TableRenderable interface {
	RenderTable(w io.Writer) error
}
