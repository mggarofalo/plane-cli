package output

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// WriteTable is a helper for writing tabular data with uniform left alignment.
func WriteTable(w io.Writer, headers []string, rows [][]string) {
	WriteTableAligned(w, headers, rows, nil)
}

// WriteTableAligned writes a table with per-column alignment.
// If alignments is nil or shorter than columns, unspecified columns default to left.
func WriteTableAligned(w io.Writer, headers []string, rows [][]string, alignments []tw.Align) {
	colAlign := tw.Alignment{tw.AlignLeft}
	for i, a := range alignments {
		colAlign = colAlign.Set(i, a)
	}

	table := tablewriter.NewTable(w,
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithHeader(headers),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithAlignment(colAlign),
		tablewriter.WithBorders(tw.Border{
			Left:   tw.Off,
			Right:  tw.Off,
			Top:    tw.Off,
			Bottom: tw.Off,
		}),
	)
	for _, row := range rows {
		anyRow := make([]any, len(row))
		for i, v := range row {
			anyRow[i] = v
		}
		table.Append(anyRow...)
	}
	table.Render()
}
