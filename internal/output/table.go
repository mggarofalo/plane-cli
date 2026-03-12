package output

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// WriteTable is a helper for writing tabular data.
func WriteTable(w io.Writer, headers []string, rows [][]string) {
	table := tablewriter.NewTable(w,
		tablewriter.WithHeader(headers),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithAlignment(tw.Alignment{tw.AlignLeft}),
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
