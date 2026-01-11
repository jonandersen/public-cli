package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Formatter handles output formatting (table or JSON).
type Formatter struct {
	Writer   io.Writer
	JSONMode bool
}

// New creates a new Formatter with the specified writer and JSON mode.
func New(w io.Writer, jsonMode bool) *Formatter {
	return &Formatter{
		Writer:   w,
		JSONMode: jsonMode,
	}
}

// Table outputs data as a formatted table or JSON array depending on mode.
// Headers define column names, rows contain the data.
func (f *Formatter) Table(headers []string, rows [][]string) error {
	if f.JSONMode {
		return f.tableAsJSON(headers, rows)
	}
	return f.tableAsText(headers, rows)
}

// tableAsText renders a table with aligned columns.
func (f *Formatter) tableAsText(headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(f.Writer, 0, 0, 2, ' ', 0)

	// Print headers
	if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
		return err
	}

	// Print separator
	separators := make([]string, len(headers))
	for i, h := range headers {
		separators[i] = strings.Repeat("-", len(h))
	}
	if _, err := fmt.Fprintln(tw, strings.Join(separators, "\t")); err != nil {
		return err
	}

	// Print rows
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}

	return tw.Flush()
}

// tableAsJSON renders a table as a JSON array of objects.
func (f *Formatter) tableAsJSON(headers []string, rows [][]string) error {
	result := make([]map[string]string, 0, len(rows))

	for _, row := range rows {
		obj := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				obj[header] = row[i]
			} else {
				obj[header] = ""
			}
		}
		result = append(result, obj)
	}

	return f.Print(result)
}

// Print outputs data as formatted JSON (pretty-printed) or as a simple string representation.
func (f *Formatter) Print(data any) error {
	if f.JSONMode {
		encoder := json.NewEncoder(f.Writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	}

	// In non-JSON mode, use a simple representation
	_, err := fmt.Fprintf(f.Writer, "%v\n", data)
	return err
}
